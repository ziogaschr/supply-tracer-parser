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
	big1 = big.NewInt(1)
)

func TestSetHead(t *testing.T) {
	s := NewState()

	supply := supplyInfo{Number: 10, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{10}, ParentHash: common.Hash{9}}

	s.setHead(&supply)

	if s.BlockNumber != 10 || s.Hash.Cmp(common.Hash{10}) != 0 || s.ParentHash.Cmp(common.Hash{9}) != 0 {
		t.Errorf("setHead failed to update state variables")
	}
}

func TestAdd(t *testing.T) {
	s := NewState()

	s.TotalDelta = big.NewInt(9)
	s.TotalReward = big.NewInt(9)
	s.TotalWithdrawals = big.NewInt(9)
	s.TotalBurn = big.NewInt(9)

	supply := supplyInfo{Number: 10, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{10}, ParentHash: common.Hash{9}}

	s.add(&supply)

	big10 := big.NewInt(10)

	// Verify total supply
	if s.TotalDelta.Cmp(big10) != 0 || s.TotalReward.Cmp(big10) != 0 || s.TotalWithdrawals.Cmp(big10) != 0 || s.TotalBurn.Cmp(big10) != 0 {
		fmt.Printf("TotalDelta want %s have %s\n", big10, s.TotalDelta)
		fmt.Printf("TotalReward want %s have %s\n", big10, s.TotalReward)
		fmt.Printf("TotalWithdrawals want %s have %s\n", big10, s.TotalWithdrawals)
		fmt.Printf("TotalBurn want %s have %s\n", big10, s.TotalBurn)

		t.Errorf("forwardTo failed to update total supply")
	}
}

func TestSub(t *testing.T) {
	s := NewState()

	s.TotalDelta = big.NewInt(10)
	s.TotalReward = big.NewInt(10)
	s.TotalWithdrawals = big.NewInt(10)
	s.TotalBurn = big.NewInt(10)

	supply := supplyInfo{Number: 10, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{10}, ParentHash: common.Hash{9}}

	s.sub(&supply)

	big9 := big.NewInt(9)

	// Verify total supply
	if s.TotalDelta.Cmp(big9) != 0 || s.TotalReward.Cmp(big9) != 0 || s.TotalWithdrawals.Cmp(big9) != 0 || s.TotalBurn.Cmp(big9) != 0 {
		fmt.Printf("TotalDelta want %s have %s\n", big9, s.TotalDelta)
		fmt.Printf("TotalReward want %s have %s\n", big9, s.TotalReward)
		fmt.Printf("TotalWithdrawals want %s have %s\n", big9, s.TotalWithdrawals)
		fmt.Printf("TotalBurn want %s have %s\n", big9, s.TotalBurn)

		t.Errorf("sub failed to update total supply")
	}
}

func TestAddToHistory(t *testing.T) {
	s := NewState()

	entry := supplyInfo{
		Number:      10,
		Delta:       big1,
		Reward:      big1,
		Withdrawals: big1,
		Burn:        big1,
		Hash:        common.Hash{10},
		ParentHash:  common.Hash{9},
	}

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
		entry := supplyInfo{Number: i, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{0}, ParentHash: common.Hash{0}}

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

	blocks := []supplyInfo{
		{Number: 0, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{0}, ParentHash: common.Hash{0}},
		{Number: 1, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{1}, ParentHash: common.Hash{0}},
	}

	for _, block := range blocks {
		s.handleEntry(block, errCh)
	}

	if s.TotalDelta.Cmp(big.NewInt(2)) != 0 || s.TotalReward.Cmp(big.NewInt(2)) != 0 || s.TotalWithdrawals.Cmp(big.NewInt(2)) != 0 || s.TotalBurn.Cmp(big.NewInt(2)) != 0 {
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
	s.TotalDelta = big.NewInt(4)
	s.TotalReward = big.NewInt(4)
	s.TotalBurn = big.NewInt(4)
	s.TotalWithdrawals = big.NewInt(4)
	s.canonicalChain = map[uint64]common.Hash{
		0: {0},
		1: {1},
		2: {2},
		3: {3},
	}
	s.HashHistory = orderedmap.New[uint64, map[common.Hash]supplyInfo](historyLimit)

	s.HashHistory.Set(0, map[common.Hash]supplyInfo{
		common.Hash{0}: {Number: 0, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{0}, ParentHash: common.Hash{}},
	})
	s.HashHistory.Set(1, map[common.Hash]supplyInfo{
		common.Hash{1}: {Number: 1, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{1}, ParentHash: common.Hash{0}},
	})
	s.HashHistory.Set(2, map[common.Hash]supplyInfo{
		common.Hash{2}: {Number: 2, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{2}, ParentHash: common.Hash{1}},
	})
	s.HashHistory.Set(3, map[common.Hash]supplyInfo{
		common.Hash{3}: {Number: 3, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{3}, ParentHash: common.Hash{2}},
	})

	errCh := make(chan error, 1)

	s.rewindTo(common.Hash{1}, 1, errCh)

	// Verify block info
	if s.BlockNumber != 1 || s.Hash.Cmp(common.Hash{1}) != 0 || s.ParentHash.Cmp(common.Hash{}) != 0 {
		t.Errorf("rewindTo failed to update block info")
	}

	big2 := big.NewInt(2)

	// Verify total supply
	if s.TotalDelta.Cmp(big2) != 0 || s.TotalReward.Cmp(big2) != 0 || s.TotalWithdrawals.Cmp(big2) != 0 || s.TotalBurn.Cmp(big2) != 0 {
		fmt.Printf("TotalDelta want %s have %s\n", big2, s.TotalDelta)
		fmt.Printf("TotalReward want %s have %s\n", big2, s.TotalReward)
		fmt.Printf("TotalWithdrawals want %s have %s\n", big2, s.TotalWithdrawals)
		fmt.Printf("TotalBurn want %s have %s\n", big2, s.TotalBurn)

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
	s.TotalDelta = big.NewInt(4)
	s.TotalReward = big.NewInt(4)
	s.TotalBurn = big.NewInt(4)
	s.TotalWithdrawals = big.NewInt(4)
	s.canonicalChain = map[uint64]common.Hash{
		0: {0},
		1: {1},
		2: {2},
		3: {3},
	}
	s.HashHistory = orderedmap.New[uint64, map[common.Hash]supplyInfo](historyLimit)

	s.HashHistory.Set(0, map[common.Hash]supplyInfo{
		common.Hash{0}: {Number: 0, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{0}, ParentHash: common.Hash{}},
	})
	s.HashHistory.Set(1, map[common.Hash]supplyInfo{
		common.Hash{1}: {Number: 1, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{1}, ParentHash: common.Hash{0}},
	})
	s.HashHistory.Set(2, map[common.Hash]supplyInfo{
		common.Hash{2}: {Number: 2, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{2}, ParentHash: common.Hash{1}},
	})

	big2 := big.NewInt(2)
	s.HashHistory.Set(3, map[common.Hash]supplyInfo{
		common.Hash{3}:  {Number: 3, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{3}, ParentHash: common.Hash{2}},
		common.Hash{31}: {Number: 3, Delta: big2, Reward: big2, Withdrawals: big2, Burn: big2, Hash: common.Hash{31}, ParentHash: common.Hash{2}},
	})

	errCh := make(chan error, 1)

	s.rewindTo(common.Hash{31}, 3, errCh)

	// Verify block info
	if s.BlockNumber != 3 || s.Hash.Cmp(common.Hash{31}) != 0 || s.ParentHash.Cmp(common.Hash{2}) != 0 {
		t.Errorf("rewindTo failed to update block info")
	}

	big5 := big.NewInt(5)

	// Verify total supply
	if s.TotalDelta.Cmp(big5) != 0 || s.TotalReward.Cmp(big5) != 0 || s.TotalWithdrawals.Cmp(big5) != 0 || s.TotalBurn.Cmp(big5) != 0 {
		fmt.Printf("TotalDelta want %s have %s\n", big5, s.TotalDelta)
		fmt.Printf("TotalReward want %s have %s\n", big5, s.TotalReward)
		fmt.Printf("TotalWithdrawals want %s have %s\n", big5, s.TotalWithdrawals)
		fmt.Printf("TotalBurn want %s have %s\n", big5, s.TotalBurn)

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
	s.TotalDelta = big.NewInt(2)
	s.TotalReward = big.NewInt(2)
	s.TotalBurn = big.NewInt(2)
	s.TotalWithdrawals = big.NewInt(2)
	s.canonicalChain = map[uint64]common.Hash{
		0: {0},
		1: {1},
	}
	s.HashHistory = orderedmap.New[uint64, map[common.Hash]supplyInfo](historyLimit)

	big2 := big.NewInt(2)

	s.HashHistory.Set(0, map[common.Hash]supplyInfo{
		common.Hash{0}: {Number: 0, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{0}, ParentHash: common.Hash{}},
	})
	s.HashHistory.Set(1, map[common.Hash]supplyInfo{
		common.Hash{1}:  {Number: 1, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{1}, ParentHash: common.Hash{0}},
		common.Hash{11}: {Number: 1, Delta: big2, Reward: big2, Withdrawals: big2, Burn: big2, Hash: common.Hash{11}, ParentHash: common.Hash{0}},
	})
	s.HashHistory.Set(2, map[common.Hash]supplyInfo{
		common.Hash{2}:  {Number: 2, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{2}, ParentHash: common.Hash{1}},
		common.Hash{22}: {Number: 2, Delta: big2, Reward: big2, Withdrawals: big2, Burn: big2, Hash: common.Hash{22}, ParentHash: common.Hash{11}},
	})
	s.HashHistory.Set(3, map[common.Hash]supplyInfo{
		common.Hash{3}:  {Number: 3, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{3}, ParentHash: common.Hash{2}},
		common.Hash{33}: {Number: 3, Delta: big2, Reward: big2, Withdrawals: big2, Burn: big2, Hash: common.Hash{33}, ParentHash: common.Hash{22}},
	})

	errCh := make(chan error, 1)

	s.forwardTo(3, common.Hash{33}, errCh)

	// Verify block info
	if s.BlockNumber != 3 || s.Hash.Cmp(common.Hash{33}) != 0 || s.ParentHash.Cmp(common.Hash{22}) != 0 {
		t.Errorf("forwardTo failed to update block info")
	}

	big7 := big.NewInt(7)

	// Verify total supply
	if s.TotalDelta.Cmp(big7) != 0 || s.TotalReward.Cmp(big7) != 0 || s.TotalWithdrawals.Cmp(big7) != 0 || s.TotalBurn.Cmp(big7) != 0 {
		fmt.Printf("TotalDelta want %s have %s\n", big7, s.TotalDelta)
		fmt.Printf("TotalReward want %s have %s\n", big7, s.TotalReward)
		fmt.Printf("TotalWithdrawals want %s have %s\n", big7, s.TotalWithdrawals)
		fmt.Printf("TotalBurn want %s have %s\n", big7, s.TotalBurn)

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

	blocks := []supplyInfo{
		{Number: 0, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{0}, ParentHash: common.Hash{}},
		{Number: 1, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{1}, ParentHash: common.Hash{0}},
		{Number: 2, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{2}, ParentHash: common.Hash{1}},
	}

	for _, block := range blocks {
		s.handleEntry(block, errCh)
	}

	withWrongParent := supplyInfo{Number: 3, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{3}, ParentHash: common.Hash{1}}
	s.handleEntry(withWrongParent, errCh)

	err := <-errCh
	if !strings.HasPrefix(err.Error(), "skipping block 3 entry") {
		t.Errorf("HandleEntry failed to drop entry because of wrong parent hash: %v", err)
	}

	// Import next block that passes validations
	withCorrectParent := supplyInfo{Number: 3, Delta: big1, Reward: big1, Withdrawals: big1, Burn: big1, Hash: common.Hash{4}, ParentHash: common.Hash{2}}
	s.handleEntry(withCorrectParent, errCh)

	if s.BlockNumber != 3 || s.Hash.Cmp(common.Hash{4}) != 0 || s.ParentHash.Cmp(common.Hash{2}) != 0 {
		err := <-errCh
		t.Errorf("HandleEntry failed to import next block, while it's correct: %v", err)
	}
}
