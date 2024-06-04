package main

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var (
	big0 = big.NewInt(0)
	big1 = big.NewInt(1)
)

func newSupplyInfo() supplyInfo {
	s := supplyInfo{}
	s.Delta = big.NewInt(0)
	s.Issuance = &supplyInfoIssuance{
		GenesisAlloc: big.NewInt(0),
		Reward:       big.NewInt(0),
		Withdrawals:  big.NewInt(0),
	}
	s.Burn = &supplyInfoBurn{
		EIP1559: big.NewInt(0),
		Blob:    big.NewInt(0),
		Misc:    big.NewInt(0),
	}
	return s
}

func TestSetHead(t *testing.T) {
	s := NewState()

	supply := supplyInfo{
		Number:     10,
		Hash:       common.Hash{10},
		ParentHash: common.Hash{9},
	}

	s.setHead(&supply)

	if s.BlockNumber != 10 || s.Hash.Cmp(common.Hash{10}) != 0 || s.ParentHash.Cmp(common.Hash{9}) != 0 {
		t.Errorf("setHead failed to update state variables")
	}
}

func TestAdd(t *testing.T) {
	// Prepare state
	s := NewState()
	s.Issuance.GenesisAlloc = big.NewInt(9)
	s.Issuance.Reward = big.NewInt(9)
	s.Issuance.Withdrawals = big.NewInt(9)
	s.Burn.EIP1559 = big.NewInt(9)
	s.Burn.Blob = big.NewInt(9)
	s.Burn.Misc = big.NewInt(9)

	// Add a new supply
	supply := newSupplyInfo()
	supply.Number = 10
	supply.Hash = common.Hash{10}
	supply.ParentHash = common.Hash{9}
	supply.Issuance.GenesisAlloc = big1
	supply.Issuance.Reward = big1
	supply.Issuance.Withdrawals = big1
	supply.Burn.EIP1559 = big1
	supply.Burn.Blob = big1
	supply.Burn.Misc = big1

	s.add(&supply)

	big10 := big.NewInt(10)

	// Verify total supply
	if s.Delta.Cmp(big0) != 0 || s.Issuance.GenesisAlloc.Cmp(big10) != 0 || s.Issuance.Reward.Cmp(big10) != 0 || s.Issuance.Withdrawals.Cmp(big10) != 0 || s.Burn.EIP1559.Cmp(big10) != 0 || s.Burn.Blob.Cmp(big10) != 0 || s.Burn.Misc.Cmp(big10) != 0 {
		fmt.Printf("Delta want %s have %s\n", big10, s.Delta)
		fmt.Printf("Issuance.GenesisAlloc want %s have %s\n", big10, s.Issuance.GenesisAlloc)
		fmt.Printf("Issuance.Reward want %s have %s\n", big10, s.Issuance.Reward)
		fmt.Printf("Issuance.Withdrawals want %s have %s\n", big10, s.Issuance.Withdrawals)
		fmt.Printf("Burn.EIP1559 want %s have %s\n", big10, s.Burn.EIP1559)
		fmt.Printf("Burn.Blob want %s have %s\n", big10, s.Burn.Blob)
		fmt.Printf("Burn.Misc want %s have %s\n", big10, s.Burn.Misc)

		t.Errorf("forwardTo failed to update total supply")
	}
}

func TestSub(t *testing.T) {
	s := NewState()
	s.Issuance.GenesisAlloc = big.NewInt(9)
	s.Issuance.Reward = big.NewInt(9)
	s.Issuance.Withdrawals = big.NewInt(9)
	s.Burn.EIP1559 = big.NewInt(9)
	s.Burn.Blob = big.NewInt(9)
	s.Burn.Misc = big.NewInt(9)

	supply := newSupplyInfo()
	supply.Number = 10
	supply.Hash = common.Hash{10}
	supply.ParentHash = common.Hash{9}
	supply.Issuance.GenesisAlloc = big1
	supply.Issuance.Reward = big1
	supply.Issuance.Withdrawals = big1
	supply.Burn.EIP1559 = big1
	supply.Burn.Blob = big1
	supply.Burn.Misc = big1

	s.sub(&supply)

	big8 := big.NewInt(8)

	// Verify total supply
	if s.Delta.Cmp(big0) != 0 || s.Issuance.GenesisAlloc.Cmp(big8) != 0 || s.Issuance.Reward.Cmp(big8) != 0 || s.Issuance.Withdrawals.Cmp(big8) != 0 || s.Burn.EIP1559.Cmp(big8) != 0 || s.Burn.Blob.Cmp(big8) != 0 || s.Burn.Misc.Cmp(big8) != 0 {
		fmt.Printf("Delta want %s have %s\n", big0, s.Delta)
		fmt.Printf("Issuance.GenesisAlloc want %s have %s\n", big8, s.Issuance.GenesisAlloc)
		fmt.Printf("Issuance.Reward want %s have %s\n", big8, s.Issuance.Reward)
		fmt.Printf("Issuance.Withdrawals want %s have %s\n", big8, s.Issuance.Withdrawals)
		fmt.Printf("Burn.EIP1559 want %s have %s\n", big8, s.Burn.EIP1559)
		fmt.Printf("Burn.Blob want %s have %s\n", big8, s.Burn.Blob)
		fmt.Printf("Burn.Misc want %s have %s\n", big8, s.Burn.Misc)

		t.Errorf("sub failed to update total supply")
	}
}

func TestAddToHistory(t *testing.T) {
	s := NewState()

	entry := newSupplyInfo()
	entry.Number = 10
	entry.Hash = common.Hash{10}
	entry.ParentHash = common.Hash{9}
	entry.Issuance.Reward = big1
	entry.Issuance.Withdrawals = big1
	entry.Burn.EIP1559 = big1

	_, exists := s.HashHistory.Get(entry.Number)
	if exists {
		t.Errorf("addToHistory entry already exists")
	}

	s.addToHistory(entry)

	hashes, exists := s.HashHistory.Get(entry.Number)
	if !exists {
		t.Errorf("addToHistory failed to add entry to the history")
	}

	if _, ok := hashes[entry.Hash]; !ok {
		t.Errorf("addToHistory failed to add entry to the history")
	}
}

func TestCleanHistory(t *testing.T) {
	s := NewState()

	// Add more than 1024 blocks
	for i := uint64(0); i < 1030; i++ {
		entry := newSupplyInfo()
		entry.Number = i
		entry.Issuance.Reward = big1

		s.addToHistory(entry)
	}

	s.cleanHistory()

	if s.HashHistory.Len() != historyLimit {
		t.Errorf("cleanHistory failed to clean up hash history")
	}

	if _, exists := s.HashHistory.Get(uint64(0)); exists {
		t.Errorf("cleanHistory failed to delete the oldest pair")
	}
}

func TestHandleEntry(t *testing.T) {
	s := NewState()

	errCh := make(chan error, 16)

	for i := uint64(0); i < 2; i++ {
		block := newSupplyInfo()
		block.Number = i
		block.Issuance.Reward = big1
		block.Hash = common.Hash{byte(i)}
		block.ParentHash = common.Hash{byte(i - 1)}

		s.handleEntry(block, errCh)
	}

	if s.Delta.Cmp(big.NewInt(2)) != 0 || s.Issuance.Reward.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("HandleEntry failed to update total supply")
	}

	if s.BlockNumber != 1 || s.Hash.Cmp(common.Hash{1}) != 0 || s.ParentHash.Cmp(common.Hash{0}) != 0 {
		t.Errorf("HandleEntry failed to update head info in state")
	}
}

func TestRewindTo(t *testing.T) {
	s := NewState()
	s.BlockNumber = 3
	s.Hash = common.Hash{3}
	s.ParentHash = common.Hash{2}
	s.Delta = big.NewInt(4)
	s.Issuance.Reward = big.NewInt(4)
	s.canonicalChain = map[uint64]common.Hash{
		0: {0},
		1: {1},
		2: {2},
		3: {3},
	}
	s.HashHistory = orderedmap.New[uint64, map[common.Hash]supplyInfo](historyLimit)

	blocks := map[uint64]supplyInfo{}
	for i := uint64(0); i < 4; i++ {
		block := newSupplyInfo()
		block.Number = i
		block.Issuance.Reward = big1
		block.Hash = common.Hash{byte(i)}
		block.ParentHash = common.Hash{byte(i - 1)}

		blocks[i] = block

		s.HashHistory.Set(i, map[common.Hash]supplyInfo{block.Hash: block})
	}

	errCh := make(chan error, 1)

	s.rewindTo(common.Hash{1}, 1, errCh)

	// Verify block info
	if s.BlockNumber != 1 || s.Hash.Cmp(common.Hash{1}) != 0 || s.ParentHash.Cmp(common.Hash{}) != 0 {
		t.Errorf("rewindTo failed to update block info")
	}

	big2 := big.NewInt(2)

	// Verify total supply
	if s.Delta.Cmp(big.NewInt(2)) != 0 || s.Issuance.Reward.Cmp(big.NewInt(2)) != 0 {
		fmt.Printf("Delta want %s have %s\n", big2, s.Delta)
		fmt.Printf("Issuance.Reward want %s have %s\n", big2, s.Issuance.Reward)

		t.Errorf("rewindTo failed to update total supply")
	}

	if len(errCh) != 0 {
		err := <-errCh
		t.Errorf("rewindTo failed to update correctly: %v", err)
	}
}

func TestRewindToSameNumber(t *testing.T) {
	s := NewState()
	s.BlockNumber = 3
	s.Hash = common.Hash{3}
	s.ParentHash = common.Hash{2}
	s.Delta = big.NewInt(4)
	s.Issuance.Reward = big.NewInt(4)
	s.canonicalChain = map[uint64]common.Hash{
		0: {0},
		1: {1},
		2: {2},
		3: {3},
	}
	s.HashHistory = orderedmap.New[uint64, map[common.Hash]supplyInfo](historyLimit)

	blocks := map[uint64]supplyInfo{}
	for i := uint64(0); i < 4; i++ {
		block := newSupplyInfo()
		block.Number = i
		block.Issuance.Reward = big1
		block.Hash = common.Hash{byte(i)}
		block.ParentHash = common.Hash{byte(i - 1)}

		blocks[i] = block

		s.HashHistory.Set(i, map[common.Hash]supplyInfo{block.Hash: block})
	}

	// Add a new block with number 3, but different hash
	big2 := big.NewInt(2)
	block := newSupplyInfo()
	block.Number = 3
	block.Issuance.Reward = big2
	block.Hash = common.Hash{31}
	block.ParentHash = common.Hash{2}

	block3History, _ := s.HashHistory.Get(3)
	block3History[block.Hash] = block
	s.HashHistory.Set(3, block3History)

	errCh := make(chan error, 1)

	s.rewindTo(common.Hash{31}, 3, errCh)

	// Verify block info
	if s.BlockNumber != 3 || s.Hash.Cmp(common.Hash{31}) != 0 || s.ParentHash.Cmp(common.Hash{2}) != 0 {
		t.Errorf("rewindTo failed to update block info")
	}

	big5 := big.NewInt(5)

	// Verify total supply
	if s.Delta.Cmp(big5) != 0 || s.Issuance.Reward.Cmp(big5) != 0 {
		fmt.Printf("Delta want %s have %s\n", big5, s.Delta)
		fmt.Printf("Issuance.Reward want %s have %s\n", big5, s.Issuance.Reward)

		t.Errorf("rewindTo failed to update total supply")
	}

	if len(errCh) != 0 {
		err := <-errCh
		t.Errorf("rewindTo failed to update correctly: %v", err)
	}
}

func TestForwardTo(t *testing.T) {
	s := NewState()
	s.BlockNumber = 1
	s.Hash = common.Hash{1}
	s.ParentHash = common.Hash{0}
	s.Delta = big.NewInt(2)
	s.Issuance.Reward = big.NewInt(2)
	s.canonicalChain = map[uint64]common.Hash{
		0: {0},
		1: {1},
	}
	s.HashHistory = orderedmap.New[uint64, map[common.Hash]supplyInfo](historyLimit)

	big2 := big.NewInt(2)

	blocks := map[uint64]supplyInfo{}
	for i := uint64(0); i < 4; i++ {
		h := map[common.Hash]supplyInfo{}

		blockA := newSupplyInfo()
		blockA.Number = i
		blockA.Issuance.Reward = big1
		blockA.Hash = common.Hash{byte(i)}
		blockA.ParentHash = common.Hash{byte(i - 1)}

		blocks[i] = blockA

		h[blockA.Hash] = blockA

		if i > 0 {
			blockB := newSupplyInfo()
			blockB.Number = i
			blockB.Issuance.Reward = big2
			blockB.Hash = common.Hash{byte(i), byte(i)}

			if i == 1 {
				blockB.ParentHash = common.Hash{0}
			} else {
				blockB.ParentHash = common.Hash{byte(i - 1), byte(i - 1)}
			}

			blocks[i] = blockB

			h[blockB.Hash] = blockB
		}

		s.HashHistory.Set(i, h)
	}

	errCh := make(chan error, 1)

	s.forwardTo(3, common.Hash{3, 3}, errCh)

	// Verify block info
	if s.BlockNumber != 3 || s.Hash.Cmp(common.Hash{3, 3}) != 0 || s.ParentHash.Cmp(common.Hash{2, 2}) != 0 {
		t.Errorf("forwardTo failed to update block info")
	}

	big7 := big.NewInt(7)

	// Verify total supply
	if s.Delta.Cmp(big7) != 0 || s.Issuance.Reward.Cmp(big7) != 0 {
		fmt.Printf("Delta want %s have %s\n", big7, s.Delta)
		fmt.Printf("Issuance.Reward want %s have %s\n", big7, s.Issuance.Reward)

		t.Errorf("forwardTo failed to update total supply")
	}

	if len(errCh) != 0 {
		err := <-errCh
		t.Errorf("forwardTo failed to update correctly: %v", err)
	}
}

func TestBlockValidations(t *testing.T) {
	s := NewState()

	errCh := make(chan error, 16)

	blocks := map[uint64]supplyInfo{}
	for i := uint64(0); i < 3; i++ {
		block := newSupplyInfo()
		block.Number = i
		block.Issuance.Reward = big1
		block.Hash = common.Hash{byte(i)}
		block.ParentHash = common.Hash{byte(i - 1)}

		blocks[i] = block
	}

	for _, block := range blocks {
		s.handleEntry(block, errCh)
	}

	blockWithWrongParent := newSupplyInfo()
	blockWithWrongParent.Number = 3
	blockWithWrongParent.Issuance.Reward = big1
	blockWithWrongParent.Hash = common.Hash{3}
	blockWithWrongParent.ParentHash = common.Hash{1}
	s.handleEntry(blockWithWrongParent, errCh)

	err := <-errCh
	if !strings.HasPrefix(err.Error(), "skipping block 3 entry") {
		t.Errorf("HandleEntry failed to drop entry because of wrong parent hash: %v", err)
	}

	// Import next block that passes validations
	blockWithCorrectParent := newSupplyInfo()
	blockWithCorrectParent.Number = 3
	blockWithCorrectParent.Issuance.Reward = big1
	blockWithCorrectParent.Hash = common.Hash{4}
	blockWithCorrectParent.ParentHash = common.Hash{2}
	s.handleEntry(blockWithCorrectParent, errCh)

	if s.BlockNumber != 3 || s.Hash.Cmp(common.Hash{4}) != 0 || s.ParentHash.Cmp(common.Hash{2}) != 0 {
		err := <-errCh
		t.Errorf("HandleEntry failed to import next block, while it's correct: %v", err)
	}
}
