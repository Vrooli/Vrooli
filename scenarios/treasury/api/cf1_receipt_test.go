package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"treasury/internal/baseusdc"
)

type cf1RPCReceiptLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

type cf1RPCReceipt struct {
	TransactionHash string             `json:"transactionHash"`
	BlockHash       string             `json:"blockHash"`
	BlockNumber     string             `json:"blockNumber"`
	Status          string             `json:"status"`
	From            string             `json:"from"`
	To              string             `json:"to"`
	Logs            []cf1RPCReceiptLog `json:"logs"`
}

const cf1ERC20TransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

func validateCF1RPCReceipt(raw []byte, transactionHash, walletAddress, counterparty string, amountMinor int64) (cf1RPCReceipt, error) {
	var receipt cf1RPCReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return receipt, fmt.Errorf("decode complete RPC response: %w", err)
	}
	if amountMinor <= 0 || receipt.Status != "0x1" || !strings.EqualFold(receipt.TransactionHash, transactionHash) || receipt.BlockHash == "" || receipt.BlockNumber == "" || !strings.EqualFold(receipt.From, walletAddress) || !strings.EqualFold(receipt.To, baseusdc.USDCAddress) {
		return receipt, errors.New("RPC receipt does not match a successful transaction from the derived budget wallet to Base USDC")
	}
	expectedAtoms := new(big.Int).Mul(big.NewInt(amountMinor), big.NewInt(10_000))
	for _, log := range receipt.Logs {
		if !strings.EqualFold(log.Address, baseusdc.USDCAddress) || len(log.Topics) != 3 || !strings.EqualFold(log.Topics[0], cf1ERC20TransferTopic) || len(log.Topics[1]) != 66 || len(log.Topics[2]) != 66 {
			continue
		}
		from := "0x" + strings.ToLower(log.Topics[1][26:])
		to := "0x" + strings.ToLower(log.Topics[2][26:])
		atoms, ok := new(big.Int).SetString(strings.TrimPrefix(log.Data, "0x"), 16)
		if ok && len(log.Data) == 66 && strings.HasPrefix(log.Data, "0x") && strings.EqualFold(from, walletAddress) && strings.EqualFold(to, counterparty) && atoms.Cmp(expectedAtoms) == 0 {
			return receipt, nil
		}
	}
	return receipt, errors.New("complete RPC receipt lacks the exact run-wallet USDC Transfer event")
}

func TestCF1ReceiptProofRequiresExactRawUSDCTransfer(t *testing.T) {
	walletAddress := "0x1111111111111111111111111111111111111111"
	counterparty := "0x2222222222222222222222222222222222222222"
	transactionHash := "0x" + strings.Repeat("a", 64)
	addressTopic := func(address string) string {
		return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(address), "0x")
	}
	makeReceipt := func() cf1RPCReceipt {
		return cf1RPCReceipt{
			TransactionHash: transactionHash, BlockHash: "0x" + strings.Repeat("b", 64), BlockNumber: "0x42",
			Status: "0x1", From: walletAddress, To: baseusdc.USDCAddress,
			Logs: []cf1RPCReceiptLog{{Address: baseusdc.USDCAddress, Topics: []string{cf1ERC20TransferTopic, addressTopic(walletAddress), addressTopic(counterparty)}, Data: fmt.Sprintf("0x%064x", 300_000)}},
		}
	}
	cases := []struct {
		name   string
		mutate func(*cf1RPCReceipt)
		valid  bool
	}{
		{name: "exact transfer", valid: true},
		{name: "wrong transaction", mutate: func(r *cf1RPCReceipt) { r.TransactionHash = "0x" + strings.Repeat("c", 64) }},
		{name: "wrong status", mutate: func(r *cf1RPCReceipt) { r.Status = "0x0" }},
		{name: "wrong signer", mutate: func(r *cf1RPCReceipt) { r.From = counterparty }},
		{name: "wrong token contract", mutate: func(r *cf1RPCReceipt) { r.Logs[0].Address = counterparty }},
		{name: "wrong recipient", mutate: func(r *cf1RPCReceipt) { r.Logs[0].Topics[2] = addressTopic(walletAddress) }},
		{name: "wrong amount", mutate: func(r *cf1RPCReceipt) { r.Logs[0].Data = fmt.Sprintf("0x%064x", 300_001) }},
		{name: "short amount encoding", mutate: func(r *cf1RPCReceipt) { r.Logs[0].Data = "0x493e0" }},
		{name: "no transfer", mutate: func(r *cf1RPCReceipt) { r.Logs = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			receipt := makeReceipt()
			if tc.mutate != nil {
				tc.mutate(&receipt)
			}
			raw, err := json.Marshal(receipt)
			if err != nil {
				t.Fatal(err)
			}
			_, err = validateCF1RPCReceipt(raw, transactionHash, walletAddress, counterparty, 30)
			if tc.valid && err != nil {
				t.Fatalf("valid receipt rejected: %v", err)
			}
			if !tc.valid && err == nil {
				t.Fatal("invalid receipt was accepted")
			}
		})
	}
}
