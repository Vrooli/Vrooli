//go:build treasury_rehearsal

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"treasury/internal/rail/x402"
	"treasury/internal/testutil/basefork"
)

type cf2AgentManagerCapture struct {
	RunID             string `json:"run_id"`
	AgentManagerRunID string `json:"agent_manager_run_id"`
	OrderID           string `json:"order_id"`
	JobID             string `json:"job_id"`
	AuthorizationID   string `json:"authorization_id"`
	CaptureID         string `json:"capture_id"`
	ExternalID        string `json:"treasury_external_id"`
	State             string `json:"state_before_restart"`
}

type cf2AgentManagerReceipt struct {
	RunID             string `json:"run_id"`
	AgentManagerRunID string `json:"agent_manager_run_id"`
	OrderID           string `json:"order_id"`
	JobID             string `json:"job_id"`
	AuthorizationID   string `json:"authorization_id"`
	CaptureID         string `json:"capture_id"`
	ExternalID        string `json:"treasury_external_id"`
	PostingID         string `json:"posting_id"`
	ReceiptID         string `json:"receipt_id"`
	State             string `json:"state"`
}

// runCF2AgentManagerLPBSOrder makes the canonical x402 purchase from inside an
// ordinary Agent Manager run. The manager captures before Treasury restarts;
// this caller then reads Treasury's accepted receipt and lets that same run
// reconcile the LPBS order to the exact posting/receipt pair.
func runCF2AgentManagerLPBSOrder(t *testing.T, ctx context.Context, repo, runDir, lpbsDeliveryDir, treasuryName, treasurySource, treasuryURL, lpbsURL string, price x402.Price, offer cf2CanonicalOffer, buyerKey *ecdsa.PrivateKey, fork *basefork.Fork, receiptToken string) string {
	t.Helper()
	require.NoError(t, fork.Call(ctx, "evm_setNextBlockTimestamp", []any{time.Now().Unix()}, nil))
	require.NoError(t, fork.Call(ctx, "evm_mine", []any{}, nil))
	challengeURL := strings.TrimRight(treasuryURL, "/") + "/api/v1/x402/prices/" + url.PathEscape(price.ID) + "/admit"
	challengeRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, challengeURL, nil)
	require.NoError(t, err)
	challengeResponse, err := (&http.Client{Timeout: 10 * time.Second}).Do(challengeRequest)
	require.NoError(t, err)
	challengeResponse.Body.Close()
	require.Equal(t, http.StatusPaymentRequired, challengeResponse.StatusCode)
	paymentSignature := cf2SignLocalForkPayment(t, challengeResponse.Header.Get("Payment-Required"), buyerKey)
	buyer := gethcrypto.PubkeyToAddress(buyerKey.PublicKey)
	beforeBuyer := cf1WalletUSDCBalance(t, ctx, fork, buyer.Hex())
	beforePayee := cf1WalletUSDCBalance(t, ctx, fork, fork.Accounts[2])
	require.Equal(t, price.Amount, beforeBuyer.String(), "ephemeral x402 buyer must hold exactly the declared local-fork price")

	coord := filepath.Join(runDir, "agent-manager-ps2-order")
	require.NoError(t, os.MkdirAll(coord, 0o700))
	signaturePath := filepath.Join(coord, "one-use-local-fork-payment.signature")
	require.NoError(t, os.WriteFile(signaturePath, []byte(paymentSignature), 0o600))
	defer func() { _ = os.Remove(signaturePath) }()
	config := map[string]string{"lpbs_url": lpbsURL, "payment_signature_path": signaturePath, "delivery_dir": lpbsDeliveryDir}
	configBytes, err := json.Marshal(config)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(coord, "treasury-config.json"), configBytes, 0o600))

	amDir := filepath.Join(repo, "scenarios/agent-manager/api")
	amCommand := exec.CommandContext(ctx, "go", "test", "./internal/orchestration/integration", "-run", "^TestTreasuryCLIFromOrdinaryRun$", "-count=1", "-v")
	amCommand.Dir = amDir
	amCommand.Env = append(os.Environ(),
		"TREASURY_CF2_AGENT_MANAGER_ONLY=1",
		"TREASURY_CF2_AGENT_MANAGER_COORD_DIR="+coord,
	)
	outputPath := filepath.Join(coord, "agent-manager-go-test.log")
	output, err := os.Create(outputPath)
	require.NoError(t, err)
	amCommand.Stdout, amCommand.Stderr = output, output
	require.NoError(t, amCommand.Start())
	childDone := make(chan error, 1)
	go func() {
		waitErr := amCommand.Wait()
		_ = output.Close()
		childDone <- waitErr
	}()
	var runIdentity struct {
		RunID string `json:"run_id"`
	}
	require.NoError(t, waitForCF1FileOrChild(ctx, filepath.Join(coord, "run-id.json"), &runIdentity, childDone))
	var capture cf2AgentManagerCapture
	require.NoError(t, waitForCF1FileOrChild(ctx, filepath.Join(coord, "agent-manager-lpbs-captured-before-restart.json"), &capture, childDone))
	require.NotEmpty(t, runIdentity.RunID)
	require.Equal(t, runIdentity.RunID, capture.RunID)
	require.Equal(t, runIdentity.RunID, capture.AgentManagerRunID)
	require.NotEmpty(t, capture.OrderID)
	require.NotEmpty(t, capture.JobID)
	require.NotEmpty(t, capture.AuthorizationID)
	require.NotEmpty(t, capture.CaptureID)
	require.True(t, strings.HasPrefix(capture.ExternalID, "treasury-x402-inflow:"))
	require.Equal(t, "ledger_pending", capture.State)

	restarted := lifecycleRehearsal(t, ctx, runDir, "cf2-am-treasury-restart", map[string]string{"GOFLAGS": "-tags=treasury_rehearsal"}, "restart", treasuryName, "--path", treasurySource, "--json", "--timeout", "240")
	require.Equal(t, strings.TrimRight(treasuryURL, "/"), strings.TrimRight(rehearsalURL(t, restarted), "/"), "Treasury restart must retain its isolated API URL")
	receiptURL := strings.TrimRight(treasuryURL, "/") + "/api/v1/x402/receipts?" + url.Values{"external_id": []string{capture.ExternalID}}.Encode()
	var accepted struct {
		ExternalID  string `json:"external_id"`
		PostingID   string `json:"posting_id"`
		ReceiptID   string `json:"receipt_id"`
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
		Basis       string `json:"basis"`
	}
	var responseBody []byte
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, receiptURL, nil)
		require.NoError(t, requestErr)
		request.Header.Set("Authorization", "Bearer "+receiptToken)
		response, requestErr := (&http.Client{Timeout: 10 * time.Second}).Do(request)
		if requestErr == nil {
			responseBody, _ = io.ReadAll(io.LimitReader(response.Body, 64<<10))
			status := response.StatusCode
			_ = response.Body.Close()
			if status == http.StatusOK && json.Unmarshal(responseBody, &accepted) == nil && accepted.PostingID != "" && accepted.ReceiptID != "" {
				break
			}
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
	require.Equal(t, capture.ExternalID, accepted.ExternalID, "Treasury restart readback must return this AM order's exact accepted seller receipt: %s", strings.TrimSpace(string(responseBody)))
	require.EqualValues(t, 4900, accepted.AmountMinor)
	require.Equal(t, "USD", accepted.Currency)
	require.NotEmpty(t, accepted.Basis)
	restartEvidence := map[string]any{
		"run_id": runIdentity.RunID, "order_id": capture.OrderID, "job_id": capture.JobID,
		"authorization_id": capture.AuthorizationID, "transaction_id": capture.CaptureID,
		"external_id": accepted.ExternalID, "posting_id": accepted.PostingID, "receipt_id": accepted.ReceiptID,
		"treasury_restarted": true,
	}
	restartBytes, err := json.MarshalIndent(restartEvidence, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(coord, "treasury-restart-confirmed.json"), restartBytes, 0o600))

	select {
	case childErr := <-childDone:
		require.NoError(t, childErr, "Agent Manager ordinary run failed; inspect %s", outputPath)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var reconciled cf2AgentManagerReceipt
	readRehearsalJSON(t, filepath.Join(coord, "agent-manager-lpbs-order-reconciled.json"), &reconciled)
	require.Equal(t, runIdentity.RunID, reconciled.RunID)
	require.Equal(t, runIdentity.RunID, reconciled.AgentManagerRunID)
	require.Equal(t, capture.OrderID, reconciled.OrderID)
	require.Equal(t, capture.JobID, reconciled.JobID)
	require.Equal(t, capture.AuthorizationID, reconciled.AuthorizationID)
	require.Equal(t, capture.CaptureID, reconciled.CaptureID)
	require.Equal(t, accepted.ExternalID, reconciled.ExternalID)
	require.Equal(t, accepted.PostingID, reconciled.PostingID)
	require.Equal(t, accepted.ReceiptID, reconciled.ReceiptID)
	require.Equal(t, "reconciled", reconciled.State)

	afterBuyer := cf1WalletUSDCBalance(t, ctx, fork, buyer.Hex())
	afterPayee := cf1WalletUSDCBalance(t, ctx, fork, fork.Accounts[2])
	amountAtoms, ok := new(big.Int).SetString(price.Amount, 10)
	require.True(t, ok)
	require.Equal(t, new(big.Int).Sub(new(big.Int).Set(beforeBuyer), amountAtoms).String(), afterBuyer.String())
	require.Equal(t, new(big.Int).Add(new(big.Int).Set(beforePayee), amountAtoms).String(), afterPayee.String())
	var chainReceipt struct {
		Status          string `json:"status"`
		TransactionHash string `json:"transactionHash"`
		Logs            []struct {
			Address string   `json:"address"`
			Topics  []string `json:"topics"`
			Data    string   `json:"data"`
		} `json:"logs"`
	}
	require.NoError(t, fork.Call(ctx, "eth_getTransactionReceipt", []any{capture.CaptureID}, &chainReceipt))
	require.Equal(t, "0x1", chainReceipt.Status)
	require.True(t, strings.EqualFold(chainReceipt.TransactionHash, capture.CaptureID))
	transferTopic := gethcrypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex()
	addressTopic := func(value string) string {
		return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(value), "0x")
	}
	transferCount := 0
	for _, log := range chainReceipt.Logs {
		if !strings.EqualFold(log.Address, basefork.USDC) || len(log.Topics) < 3 || !strings.EqualFold(log.Topics[0], transferTopic) {
			continue
		}
		if !strings.EqualFold(log.Topics[1], addressTopic(buyer.Hex())) || !strings.EqualFold(log.Topics[2], addressTopic(fork.Accounts[2])) {
			continue
		}
		atoms, parsed := new(big.Int).SetString(strings.TrimPrefix(log.Data, "0x"), 16)
		if parsed && atoms.String() == price.Amount {
			transferCount++
		}
	}
	require.Equal(t, 1, transferCount, "Agent Manager's exact x402 transaction must contain one buyer-to-payee Transfer")
	writeRehearsalJSON(t, filepath.Join(runDir, "cf2-agent-manager-lpbs-order-result.json"), map[string]any{
		"run_id": reconciled.RunID, "agent_manager_run_id": reconciled.AgentManagerRunID,
		"order_id": reconciled.OrderID, "job_id": reconciled.JobID, "authorization_id": reconciled.AuthorizationID,
		"capture_id": reconciled.CaptureID, "treasury_external_id": reconciled.ExternalID,
		"posting_id": reconciled.PostingID, "receipt_id": reconciled.ReceiptID,
		"one_exact_transfer": transferCount == 1, "transfer_atoms": price.Amount,
	})
	return reconciled.OrderID
}
