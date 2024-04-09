package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// historyLimit is the maximum number of blocks to keep in history
const historyLimit = 1024

// TotalSupply represents the total supply data
type TotalSupply struct {
	BlockNumber uint64      `json:"blockNumber"` // Block number of the current state
	Hash        common.Hash `json:"hash"`        // Hash of the current state
	ParentHash  common.Hash `json:"parentHash"`  // Parent hash of the current state

	TotalDelta       *big.Int `json:"totalDelta"`
	TotalReward      *big.Int `json:"totalReward"`
	TotalWithdrawals *big.Int `json:"totalWithdrawals"`
	TotalBurn        *big.Int `json:"totalBurn"`
}

// State represents the latest state of the parsed supply data
type State struct {
	TotalSupply

	sync.RWMutex

	canonicalChain         map[uint64]common.Hash
	HashHistory *orderedmap.OrderedMap[uint64, map[common.Hash]supplyInfo] `json:"-"`
}

type PersistedState struct {
	TotalSupply
	File string `json:"file"`
}

type SaveLastParsedFile string

func NewState() *State {
	state := &State{}
	state.TotalDelta = big.NewInt(0)
	state.TotalReward = big.NewInt(0)
	state.TotalWithdrawals = big.NewInt(0)
	state.TotalBurn = big.NewInt(0)

	state.canonicalChain = make(map[uint64]common.Hash)
	state.HashHistory = orderedmap.New[uint64, map[common.Hash]supplyInfo](historyLimit)

	return state
}

// setHead sets the current block as the state head
func (s *State) setHead(supply *supplyInfo) {
	s.Lock()
	defer s.Unlock()

	// Set current block as state head
	s.BlockNumber = supply.Number
	s.Hash = supply.Hash
	s.ParentHash = supply.ParentHash

	s.canonicalChain[supply.Number] = supply.Hash
}

// add adds the supply data to the state
func (s *State) add(supply *supplyInfo) {
	s.Lock()
	defer s.Unlock()

	// Add supply to state
	s.TotalDelta.Add(s.TotalDelta, supply.Delta)
	s.TotalReward.Add(s.TotalReward, supply.Reward)
	s.TotalWithdrawals.Add(s.TotalWithdrawals, supply.Withdrawals)
	s.TotalBurn.Add(s.TotalBurn, supply.Burn)
}

// sub subtracts the supply data from the state
func (s *State) sub(supply *supplyInfo) {
	s.Lock()
	defer s.Unlock()

	// Subtract supply from state
	s.TotalDelta.Sub(s.TotalDelta, supply.Delta)
	s.TotalReward.Sub(s.TotalReward, supply.Reward)
	s.TotalWithdrawals.Sub(s.TotalWithdrawals, supply.Withdrawals)
	s.TotalBurn.Sub(s.TotalBurn, supply.Burn)
}

// addToHistory adds the supply data to the history
func (s *State) addToHistory(entry supplyInfo) {
	s.Lock()
	defer s.Unlock()

	hashes, exists := s.HashHistory.Get(entry.Number)
	if !exists {
		hashes = make(map[common.Hash]supplyInfo)
		s.HashHistory.Set(entry.Number, hashes)
	}

	hashes[entry.Hash] = entry

	s.HashHistory.Set(entry.Number, hashes)
}

// getSupply returns the supply data for the specified block number and hash
func (s *State) getSupply(hash common.Hash, number uint64) (*supplyInfo, bool) {
	s.RLock()
	defer s.RUnlock()

	hashes, exists := s.HashHistory.Get(number)
	if !exists {
		return nil, false
	}

	for hHash, supply := range hashes {
		if hHash == hash {
			return &supply, true
		}
	}

	return nil, false
}

// getSupplyByHash returns the supply data for the specified block hash
func (s *State) getSupplyByHash(hash common.Hash) (*supplyInfo, bool) {
	s.RLock()
	defer s.RUnlock()

	// It's slow to loop through all history, but performance is not an issue for this app,
	// and it's not worth the complexity of maintaining a reverse lookup map.

	for pair := s.HashHistory.Newest(); pair != nil; pair = pair.Prev() {
		hashes := pair.Value
		for hHash, supply := range hashes {
			if hHash == hash {
				return &supply, true
			}
		}
	}

	return nil, false
}

// cleanHistory cleans the history to maintain only recent blocks
func (s *State) cleanHistory() {
	s.Lock()
	defer s.Unlock()

	var pairToDelete *orderedmap.Pair[uint64, map[common.Hash]supplyInfo]

	for pair := s.HashHistory.Oldest(); pair != nil; pair = pair.Next() {
		if s.HashHistory.Len() <= historyLimit {
			break
		}

		// Delete previous loop pair
		if pairToDelete != nil {
			s.HashHistory.Delete(pairToDelete.Key)
		}

		pairToDelete = pair
	}
}

// handleEntry updates the state with the new supply data.
func (s *State) handleEntry(supply supplyInfo, errCh chan error) {
	isInitialBlockHandling := s.BlockNumber == 0 && s.Hash == common.Hash{}

	if !isInitialBlockHandling {
		// When state is behind, forward to block parent
		if supply.Number-1 > s.BlockNumber {
			s.forwardTo(supply.Number-1, supply.ParentHash, errCh)

			// When state is ahead or parent is not correct, rewind back
		} else if supply.Number <= s.BlockNumber || supply.ParentHash != s.Hash {

			// Rewind to parent
			blockNumberHint := supply.Number - 1

			// If the parent is not correct, then rewind by hash only
			if supply.ParentHash != s.Hash {
				blockNumberHint = 0
			}

			s.rewindTo(supply.ParentHash, blockNumberHint, errCh)
		}

		// TODO: the validation happens after the chain reorgs to prepare the state for the new block.
		// Do we want to revert the reorg in case the validation fails?
		if supply.Number-1 != s.BlockNumber || supply.ParentHash != s.Hash {
			errCh <- fmt.Errorf("skipping block %d entry. ParentHash check failed.\n\tCurrent %d ParentHash:\t%s\n\tParent %d Hash:\t%s", supply.Number, supply.Number, supply.ParentHash, s.BlockNumber, s.Hash)
			return
		}
	}

	// Update state
	s.setHead(&supply)
	s.add(&supply)

	// Prepend current block to history for potential future rewinds.
	s.addToHistory(supply)

	// Clean history to maintain only recent blocks
	s.cleanHistory()
}

// rewindTo rewinds the state to the specified block number and hash
func (s *State) rewindTo(hash common.Hash, numberHint uint64, errCh chan error) {
	// log.Println("Rewinding \n\tto number", numberHint, "hash", hash, "\n\tfrom number", s.BlockNumber, "hash", s.Hash)

	fromBlock := s.BlockNumber
	newestTrace := s.HashHistory.Newest()
	oldestTrace := s.HashHistory.Oldest()

	number := uint64(0)

	// Set number and hash of block to rewind to
	if numberHint == 0 {
		lookupSupply, found := s.getSupplyByHash(hash)
		if !found {
			errCh <- fmt.Errorf("cannot rewind to block hash %s, it is not in history", hash)
			return
		}
		number = lookupSupply.Number

		// Check if the block to rewind to for a known blockNumber is in history
	} else {
		number = numberHint

		// Check if the block to rewind to is in history
		if newestTrace.Key < number || oldestTrace.Key > number {
			errCh <- fmt.Errorf("cannot rewind to block %d, it is not in history. History oldest number: %d, newest number: %d", number, oldestTrace.Key, newestTrace.Key)
			return
		}
	}

	// After rewinding to set block, forward back to expected head if needed
	var forwardToNumber uint64
	var forwardToHash common.Hash
	defer func() {
		if forwardToNumber > 0 && forwardToHash != (common.Hash{}) {
			s.forwardTo(forwardToNumber, forwardToHash, errCh)
		}
	}()

	// Check if we need to replace the current head (same number for entry and state) with a different hash
	if number == s.BlockNumber {
		// Set block to forward to after rewinding to set block
		forwardToNumber = number
		forwardToHash = hash

		// Rewind to parent block
		number -= 1
	}

	// Revert the canonical chain
	var hNumber uint64
	depth := 0

	for hNumber = s.BlockNumber; hNumber >= number; hNumber-- {
		hHash, found := s.canonicalChain[hNumber]
		if !found {
			errCh <- fmt.Errorf("cannot find canonChain hash for block number %d", hNumber)
			return
		}

		supply, found := s.getSupply(hHash, hNumber)
		if !found {
			errCh <- fmt.Errorf("cannot find supply info for block number %d (%s)", hNumber, hHash)
			return
		}

		// Set current state to the block we are aiming to rewind to
		s.setHead(supply)

		// Rewinded successfully, don't reverse last block totals
		if hNumber == number {
			break
		}

		// Reverse totals, skip the block we are rewinding to
		s.sub(supply)

		depth++
	}

	if depth > 3 {
		log.Println("Rewinded successfully to block", hNumber, "from block", fromBlock, "depth", depth)
	}
}

// forwardTo forwards the state to the specified block number and hash
func (s *State) forwardTo(number uint64, hash common.Hash, errCh chan error) {
	// log.Println("Forwarding \n\tto number", number, "hash", hash, "\n\tfrom number", s.BlockNumber, "hash", s.Hash)

	newestTrace := s.HashHistory.Newest()
	oldestTrace := s.HashHistory.Oldest()

	// Check if the block to forward to is in history
	if newestTrace.Key < number || oldestTrace.Key >= number {
		errCh <- fmt.Errorf("cannot forward to block %d, it is not in history. History oldest number: %d, newest number: %d", number, oldestTrace.Key, newestTrace.Key)
		return
	}

	// We first need to find the block we're looking for
	lookupHash := hash
	breakLoop := false

	var pair *orderedmap.Pair[uint64, map[common.Hash]supplyInfo]

	forwardedChain := []supplyInfo{}

	// Locate the block in history
	for pair = s.HashHistory.Newest(); pair != nil; pair = pair.Prev() {
		hNumber, hashes := pair.Key, pair.Value

		// History can have newer blocks, ignore them
		if hNumber > number {
			continue
		}

		supply, found := hashes[lookupHash]
		if !found {
			errCh <- fmt.Errorf("cannot find hash %s in history for block %d", lookupHash, hNumber)
			return
		}

		// We reached the block we're looking for
		// 1. history item is smaller than target block number
		// 2. history item BlockNumber is smaller than the current state block number
		if hNumber < number && s.BlockNumber > hNumber {
			breakLoop = true
		}

		// Next block lookupHash
		lookupHash = supply.ParentHash

		if breakLoop {
			break
		}

		forwardedChain = append([]supplyInfo{supply}, forwardedChain...)
	}

	// Forward the state up to block
	for _, supply := range forwardedChain {
		if s.BlockNumber >= supply.Number {
			s.rewindTo(supply.ParentHash, supply.Number-1, errCh)
		}

		// Set current state
		s.setHead(&supply)
		s.add(&supply)
	}

	if s.BlockNumber != number {
		errCh <- fmt.Errorf("cannot forward to block. want: %d, have: %d", s.BlockNumber, number)
		return
	}

	if len(forwardedChain) > 3 {
		log.Println("Forwarded successfully to block", number, "from block", forwardedChain[0].Number, "depth", len(forwardedChain))
	}
}

// SaveState saves the current state to a file
func (s *State) SaveState(path, lastParsedFilename string) {
	s.RLock()

	data := PersistedState{}

	data.BlockNumber = s.BlockNumber
	data.Hash = s.Hash
	data.ParentHash = s.ParentHash

	data.TotalDelta = s.TotalDelta
	data.TotalReward = s.TotalReward
	data.TotalWithdrawals = s.TotalWithdrawals
	data.TotalBurn = s.TotalBurn

	data.File = lastParsedFilename

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("failed to marshal state: %v", err)
	}

	s.RUnlock()

	err = os.WriteFile(path, jsonData, 0644)
	if err != nil {
		log.Fatalf("failed to write state to file: %v", err)
	}
}

// LoadState loads the state from a file
func (s *State) LoadState(file string) (lastFile string, err error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("state file reading: %v", err)
	}

	var persistedState PersistedState
	err = json.Unmarshal(bytes, &persistedState)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal state file: %v", err)
	}

	s.BlockNumber = persistedState.BlockNumber
	s.Hash = persistedState.Hash
	s.ParentHash = persistedState.ParentHash

	s.TotalDelta = persistedState.TotalDelta
	s.TotalReward = persistedState.TotalReward
	s.TotalWithdrawals = persistedState.TotalWithdrawals
	s.TotalBurn = persistedState.TotalBurn

	log.Println("Loaded state from file", file)

	return persistedState.File, nil
}
