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

const historyLimit = 1024

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

	state.HashHistory = orderedmap.New[uint64, map[common.Hash]supplyInfo](historyLimit)

	return state
}

// setHead sets the current block as the state head
func (s *State) setHead(supply supplyInfo) {
	s.Lock()
	defer s.Unlock()

	// Set current block as state head
	s.BlockNumber = supply.Number
	s.Hash = supply.Hash
	s.ParentHash = supply.ParentHash
}

// add adds the supply data to the state
func (s *State) add(supply supplyInfo) {
	s.Lock()
	defer s.Unlock()

	// Add supply to state
	s.TotalDelta.Add(s.TotalDelta, supply.Delta)
	s.TotalReward.Add(s.TotalReward, supply.Reward)
	s.TotalWithdrawals.Add(s.TotalWithdrawals, supply.Withdrawals)
	s.TotalBurn.Add(s.TotalBurn, supply.Burn)
}

// sub subtracts the supply data from the state
func (s *State) sub(supply supplyInfo) {
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
		if supply.Number-1 > s.BlockNumber {
			// Forward to block parent
			s.forwardTo(supply.Number-1, supply.ParentHash, errCh)
		} else if supply.Number <= s.BlockNumber || supply.ParentHash != s.Hash {
			// Rewind back to block parent
			s.rewindTo(supply.Number-1, supply.ParentHash, errCh)
		}

		if supply.Number != s.BlockNumber+1 || supply.ParentHash != s.Hash {
			errCh <- fmt.Errorf("skipping block %d entry. ParentHash check failed.\n\tTrace ParentHash:\t%s\n\tCurrent state Hash:\t%s", supply.Number, supply.ParentHash, s.Hash)
			return
		}
	}

	s.setHead(supply)
	s.add(supply)

	// Prepend current block to history for potential future rewinds.
	s.addToHistory(supply)

	// Clean history to maintain only recent blocks
	s.cleanHistory()
}

// rewindTo rewinds the state to the specified block number and hash
func (s *State) rewindTo(number uint64, hash common.Hash, errCh chan error) {
	// log.Println("Rewinding \n\tto number", number, "hash", hash, "\n\tfrom number", s.BlockNumber, "hash", s.Hash)

	fromBlock := s.BlockNumber
	newestTrace := s.HashHistory.Newest()
	oldestTrace := s.HashHistory.Oldest()

	// Check if the block to rewind to is in history
	if newestTrace.Key < number || oldestTrace.Key > number {
		errCh <- fmt.Errorf("cannot rewind to block %d, it is not in history. History oldest number: %d, newest number: %d", number, oldestTrace.Key, newestTrace.Key)
		return
	}

	var lookupHash common.Hash

	depth := 0

	for pair := s.HashHistory.Newest(); pair != nil; pair = pair.Prev() {
		hNumber, hashes := pair.Key, pair.Value

		// History can have newer blocks that the one we need, ignore them
		if hNumber > s.BlockNumber {
			continue
		}

		// We reached the block we're looking for, lookup by hash instead of parent hash
		if s.BlockNumber == number {
			lookupHash = hash
		} else if hNumber == s.BlockNumber {
			lookupHash = s.Hash
		} else {
			lookupHash = s.ParentHash
		}

		supplyHistory, exists := hashes[lookupHash]
		if !exists {
			errCh <- fmt.Errorf("cannot rewind to hash %s for block %d, it is not in history", hash, number)
			return
		}

		// Set current state to the block we are aiming to rewind to
		s.setHead(supplyHistory)

		// Rewinded successfully
		if number >= supplyHistory.Number {
			break
		}

		// Reverse totals, skip the block we are rewinding to
		s.sub(supplyHistory)

		depth++
	}
	if depth > 3 {
		log.Println("Rewinded successfully to block", number, hash, "from block", fromBlock, "depth", depth)
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

	// TODO: set length
	canonicalChain := []supplyInfo{}

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

		// We reached the block we're looking for.
		// Take care that state might be behind the target, needing to rewind further
		if hNumber < number && lookupHash == supply.Hash && s.BlockNumber >= hNumber {
			breakLoop = true
		}

		// Next block lookupHash
		lookupHash = supply.ParentHash

		canonicalChain = append([]supplyInfo{supply}, canonicalChain...)

		if breakLoop {
			break
		}
	}

	for _, supply := range canonicalChain {
		if s.BlockNumber > supply.Number-1 {
			s.rewindTo(supply.Number-1, supply.ParentHash, errCh)
		}

		// Set current state to the block
		s.setHead(supply)

		// Add totals for this block
		s.add(supply)
	}

	if s.BlockNumber != number {
		errCh <- fmt.Errorf("cannot forward to block. want: %d, have: %d", s.BlockNumber, number)
		return
	}

	if len(canonicalChain) > 3 {
		log.Println("Forwarded successfully to block", number, hash, "from block", canonicalChain[0].Number, canonicalChain[0].Hash, "depth", len(canonicalChain))
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
