package x402

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"treasury/internal/rail"

	"github.com/stretchr/testify/require"
)

// [REQ:TRS-P1-001] A local priced endpoint exercises the complete 402,
// operator-wallet signature, paid retry, and receipt cycle in one rail call.
func TestOutboundPayAndRetryReturnsFacilitatorReceipt(t *testing.T) {
	var endpoint string
	var attempts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts.Add(1)
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		payload, err := base64.StdEncoding.DecodeString(request.Header.Get(paymentSignatureHeader))
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(payload, &body))
		require.Equal(t, float64(2), body["x402Version"])
		require.NotNil(t, body["accepted"])
		receipt, err := json.Marshal(map[string]any{"success": true, "payer": "0x1111111111111111111111111111111111111111", "transaction": "0xabc123", "network": "eip155:8453"})
		require.NoError(t, err)
		writer.Header().Set(paymentResponseHeader, base64.StdEncoding.EncodeToString(receipt))
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"nodes":[{"name":"research-result"}]}`))
	}))
	defer server.Close()
	endpoint = server.URL + "/priced"

	signer := &fixtureSigner{}
	adapter, err := New(server.Client(), signer)
	require.NoError(t, err)
	adapter.now = func() time.Time { return time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC) }
	result, err := adapter.Settle(context.Background(), command(t, endpoint, 1))
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, result.Outcome)
	require.Equal(t, "0xabc123", result.ExternalID)
	require.Equal(t, "x402_facilitator_confirmation", result.Basis)
	require.NotEmpty(t, result.ReceiptReference)
	paidBody := []byte(`{"nodes":[{"name":"research-result"}]}`)
	digest := sha256.Sum256(paidBody)
	require.Contains(t, result.Detail, fmt.Sprintf("response_bytes=%d", len(paidBody)))
	require.Contains(t, result.Detail, "response_sha256="+hex.EncodeToString(digest[:]))
	require.NotContains(t, result.Detail, "research-result", "only a digest, never resource content, is retained")
	require.EqualValues(t, 1, signer.calls.Load())
	require.EqualValues(t, 2, attempts.Load())
	require.NotNil(t, signer.typed["message"])
}

// [REQ:TRS-P1-001] Oversized paid content cannot escape the evidence bound or
// be partially published as a complete result.
func TestOutboundPaidResponseEvidenceIsBounded(t *testing.T) {
	var endpoint string
	root := newPrivateEvidenceRoot(t)
	evidenceDir := filepath.Join(root, "agent-manager-cli")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		receipt, err := json.Marshal(map[string]any{"success": true, "payer": "0x1111111111111111111111111111111111111111", "transaction": "0xbounded", "network": "eip155:8453"})
		require.NoError(t, err)
		writer.Header().Set(paymentResponseHeader, base64.StdEncoding.EncodeToString(receipt))
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(strings.Repeat("x", maxResponseBytes+100)))
	}))
	defer server.Close()
	endpoint = server.URL + "/bounded"
	adapter, err := NewForLocalForkPaidBodyEvidence(server.Client(), &fixtureSigner{}, root, evidenceDir)
	require.NoError(t, err)
	input := command(t, endpoint, 1)
	input.SettlementID = "bounded-response-settlement"
	result, err := adapter.Settle(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, result.Outcome, "the facilitator receipt remains authoritative when result metadata is oversized")
	require.Contains(t, result.Detail, "response_evidence=truncated")
	require.Contains(t, result.Detail, fmt.Sprintf("response_bytes_at_least=%d", maxResponseBytes+1))
	require.NotContains(t, result.Detail, "response_sha256=", "truncated bodies must not claim a complete-content digest")
	require.Contains(t, result.Detail, "response_artifact=not_published")
	entries, err := os.ReadDir(evidenceDir)
	require.NoError(t, err)
	require.Empty(t, entries, "an oversized body must never be partially published")
}

// [REQ:TRS-P1-001] An explicitly enabled isolated local-fork sink preserves
// the exact bounded paid response while ordinary settlement detail stays
// content-free and carries the exact length and digest.
func TestLocalForkPaidBodyEvidenceWritesExactPrivateArtifact(t *testing.T) {
	var endpoint string
	body := []byte(`{"result":{"id":"measured-result","value":42}}`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		writePaidSuccessResponse(t, writer, body)
	}))
	defer server.Close()
	endpoint = server.URL + "/measured"
	root := newPrivateEvidenceRoot(t)
	evidenceDir := filepath.Join(root, "agent-manager-cli")
	adapter, err := NewForLocalForkPaidBodyEvidence(server.Client(), &fixtureSigner{}, root, evidenceDir)
	require.NoError(t, err)
	input := command(t, endpoint, 1)
	input.SettlementID = "settlement-measured"
	result, err := adapter.Settle(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, result.Outcome)
	digest := sha256.Sum256(body)
	artifactName := "x402-paid-response-settlement-measured.body"
	require.Contains(t, result.Detail, "response_artifact="+artifactName)
	require.Contains(t, result.Detail, fmt.Sprintf("response_bytes=%d", len(body)))
	require.Contains(t, result.Detail, "response_sha256="+hex.EncodeToString(digest[:]))
	require.NotContains(t, result.Detail, string(body))
	require.NotContains(t, result.Detail, root, "settlement metadata must not expose the absolute evidence path")
	artifactPath := filepath.Join(evidenceDir, artifactName)
	actual, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	require.Equal(t, body, actual)
	info, err := os.Stat(artifactPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

// [REQ:TRS-P1-001] Normal production construction ignores ambient evidence
// path variables and persists only bounded response metadata.
func TestDefaultAdapterNeverCapturesPaidBodyFromAmbientEnvironment(t *testing.T) {
	var endpoint string
	body := []byte(`sensitive-but-fixture-only-result`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		writePaidSuccessResponse(t, writer, body)
	}))
	defer server.Close()
	endpoint = server.URL + "/normal"
	root := newPrivateEvidenceRoot(t)
	evidenceDir := filepath.Join(root, "ambient-capture")
	t.Setenv("TREASURY_X402_PAID_BODY_EVIDENCE_ROOT", root)
	t.Setenv("TREASURY_X402_PAID_BODY_EVIDENCE_DIR", evidenceDir)
	adapter, err := New(server.Client(), &fixtureSigner{})
	require.NoError(t, err)
	result, err := adapter.Settle(context.Background(), command(t, endpoint, 1))
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, result.Outcome)
	require.Contains(t, result.Detail, "response_sha256=")
	require.NotContains(t, result.Detail, string(body))
	_, err = os.Lstat(evidenceDir)
	require.True(t, errors.Is(err, os.ErrNotExist), "the production constructor must ignore ambient artifact settings")
}

// [REQ:TRS-P1-001] Unsafe settlement IDs and paths outside the run evidence
// directory refuse artifact publication.
func TestLocalForkPaidBodyEvidenceRejectsUnsafeIDAndEscapingDirectories(t *testing.T) {
	root := newPrivateEvidenceRoot(t)
	if _, err := NewForLocalForkPaidBodyEvidence(&http.Client{}, &fixtureSigner{}, root, filepath.Join(filepath.Dir(root), "outside")); err == nil {
		t.Fatal("evidence directory outside current run root was accepted")
	}
	rootAlias := filepath.Join(filepath.Dir(root), "x402-evidence-root-alias")
	require.NoError(t, os.Symlink(root, rootAlias))
	if _, err := newPaidBodyEvidenceSink(rootAlias, filepath.Join(rootAlias, "aliased-child")); err == nil {
		t.Fatal("evidence root reached through a symlinked ancestor was accepted")
	}
	outside := filepath.Join(filepath.Dir(root), "x402-evidence-outside")
	require.NoError(t, os.Mkdir(outside, 0o700))
	symlink := filepath.Join(root, "agent-manager-cli")
	require.NoError(t, os.Symlink(outside, symlink))
	if _, err := NewForLocalForkPaidBodyEvidence(&http.Client{}, &fixtureSigner{}, root, symlink); err == nil {
		t.Fatal("evidence symlink was accepted")
	}
	_ = os.Remove(symlink)
	evidenceDir := filepath.Join(root, "agent-manager-cli")
	server, endpoint := newPaidResultServer(t, []byte(`{"ok":true}`))
	defer server.Close()
	adapter, err := NewForLocalForkPaidBodyEvidence(server.Client(), &fixtureSigner{}, root, evidenceDir)
	require.NoError(t, err)
	input := command(t, endpoint, 1)
	input.SettlementID = "../escape"
	result, err := adapter.Settle(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, result.Outcome, "capture refusal must preserve a proven settlement")
	require.Contains(t, result.Detail, "response_artifact=not_published")
	require.NotContains(t, result.Detail, "../escape")
	entries, err := os.ReadDir(evidenceDir)
	require.NoError(t, err)
	require.Empty(t, entries, "unsafe IDs must not escape or publish under the evidence directory")
}

// [REQ:TRS-P1-001] Exact retries are idempotent; a conflicting response for
// the same settlement cannot replace or silently rebind its evidence artifact.
func TestPaidBodyEvidenceRetryIsIdempotentAndConflictPreserving(t *testing.T) {
	root := newPrivateEvidenceRoot(t)
	dir := filepath.Join(root, "agent-manager-cli")
	sink, err := newPaidBodyEvidenceSink(root, dir)
	require.NoError(t, err)
	first := []byte(`{"receipt":"first"}`)
	name, err := sink.write("settlement-1", first)
	require.NoError(t, err)
	require.Equal(t, "x402-paid-response-settlement-1.body", name)
	name, err = sink.write("settlement-1", first)
	require.NoError(t, err, "a restart/retry with the same exact result should reuse the artifact")
	require.Equal(t, "x402-paid-response-settlement-1.body", name)
	if _, err := sink.write("settlement-1", []byte(`{"receipt":"different"}`)); err == nil {
		t.Fatal("conflicting response replaced an existing settlement artifact")
	}
	actual, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	require.Equal(t, first, actual)
}

// [REQ:TRS-P1-001] Replacing the configured child directory with a symlink
// after setup cannot redirect writes outside the run root.
func TestPaidBodyEvidenceRefusesReplacedDirectory(t *testing.T) {
	root := newPrivateEvidenceRoot(t)
	dir := filepath.Join(root, "agent-manager-cli")
	sink, err := newPaidBodyEvidenceSink(root, dir)
	require.NoError(t, err)
	parked := filepath.Join(root, "parked-directory")
	require.NoError(t, os.Rename(dir, parked))
	outside := filepath.Join(filepath.Dir(root), "x402-evidence-replacement-target")
	require.NoError(t, os.Mkdir(outside, 0o700))
	require.NoError(t, os.Symlink(outside, dir))
	if _, err := sink.write("replacement-case", []byte(`{"safe":true}`)); err == nil {
		t.Fatal("write accepted a replaced evidence directory")
	}
	entries, err := os.ReadDir(outside)
	require.NoError(t, err)
	require.Empty(t, entries, "directory replacement must not redirect artifact publication")
}

// [REQ:TRS-P1-001] A directory moved after the artifact link is detected
// through the pinned root descriptor and the new artifact is rolled back.
func TestPaidBodyEvidenceRollsBackWhenDirectoryMovesDuringPublish(t *testing.T) {
	root := newPrivateEvidenceRoot(t)
	dir := filepath.Join(root, "agent-manager-cli")
	sink, err := newPaidBodyEvidenceSink(root, dir)
	require.NoError(t, err)
	relocated := filepath.Join(filepath.Dir(root), "x402-evidence-relocated")
	sink.afterLink = func() {
		if renameErr := os.Rename(dir, relocated); renameErr != nil {
			t.Errorf("relocate evidence directory during publication: %v", renameErr)
		}
	}
	if _, err := sink.write("relocated-case", []byte(`{"private":true}`)); err == nil {
		t.Fatal("publication succeeded after the evidence directory moved out of its run root")
	}
	entries, err := os.ReadDir(relocated)
	require.NoError(t, err)
	require.Empty(t, entries, "the newly linked body must be rolled back through the pinned descriptor")
}

// [REQ:TRS-P1-001] Existing artifact retries open no-follow and reject both a
// swapped symlink and an oversized regular file before reading its content.
func TestPaidBodyEvidenceRefusesSwappedAndOversizedExistingArtifacts(t *testing.T) {
	root := newPrivateEvidenceRoot(t)
	dir := filepath.Join(root, "agent-manager-cli")
	sink, err := newPaidBodyEvidenceSink(root, dir)
	require.NoError(t, err)
	name := "x402-paid-response-swapped.body"
	external := filepath.Join(filepath.Dir(root), "x402-evidence-external-body")
	require.NoError(t, os.WriteFile(external, []byte("outside-content"), 0o600))
	require.NoError(t, os.Symlink(external, filepath.Join(dir, name)))
	if err := verifyExistingEvidenceAt(int(sink.dirFD.Fd()), name, []byte("outside-content")); err == nil {
		t.Fatal("existing artifact symlink was followed")
	}
	require.NoError(t, os.Remove(filepath.Join(dir, name)))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(strings.Repeat("z", maxResponseBytes+1)), 0o600))
	if err := verifyExistingEvidenceAt(int(sink.dirFD.Fd()), name, []byte("small")); err == nil {
		t.Fatal("oversized existing artifact was accepted")
	}
}

func newPaidResultServer(t *testing.T, body []byte) (*httptest.Server, string) {
	t.Helper()
	var endpoint string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		writePaidSuccessResponse(t, writer, body)
	}))
	endpoint = server.URL + "/priced"
	return server, endpoint
}

func writePaidSuccessResponse(t *testing.T, writer http.ResponseWriter, body []byte) {
	t.Helper()
	receipt, err := json.Marshal(map[string]any{"success": true, "payer": "0x1111111111111111111111111111111111111111", "transaction": "0xartifact", "network": "eip155:8453"})
	require.NoError(t, err)
	writer.Header().Set(paymentResponseHeader, base64.StdEncoding.EncodeToString(receipt))
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(body)
}

func newPrivateEvidenceRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "current-run")
	require.NoError(t, os.Mkdir(root, 0o700))
	return root
}

func TestOutboundPersistsExactPayloadBeforePaidDispatch(t *testing.T) {
	var endpoint string
	var recordedDigest, recordedPayload string
	var recordedMu sync.Mutex
	var paidRequests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		paidRequests.Add(1)
		recordedMu.Lock()
		digest, payload := recordedDigest, recordedPayload
		recordedMu.Unlock()
		require.NotEmpty(t, payload, "recovery payload must be durable before dispatch")
		require.NotEmpty(t, digest)
		require.Equal(t, payload, request.Header.Get(paymentSignatureHeader))
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	endpoint = server.URL + "/priced"
	adapter, err := New(server.Client(), &fixtureSigner{})
	require.NoError(t, err)
	cmd := command(t, endpoint, 1)
	cmd.RecordRecoveryInput = func(_ context.Context, digest, payload string) error {
		recordedMu.Lock()
		defer recordedMu.Unlock()
		recordedDigest, recordedPayload = digest, payload
		return nil
	}

	result, err := adapter.Settle(context.Background(), cmd)
	require.Error(t, err)
	require.Equal(t, rail.OutcomeUnknown, result.Outcome)
	recordedMu.Lock()
	digest, payload := recordedDigest, recordedPayload
	recordedMu.Unlock()
	require.Equal(t, digest, result.ExternalID)
	require.Equal(t, payload, result.ReceiptReference)
	require.EqualValues(t, 1, paidRequests.Load())
}

func TestOutboundDoesNotDispatchWhenRecoveryInputCannotBePersisted(t *testing.T) {
	var endpoint string
	var attempts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts.Add(1)
		if request.Header.Get(paymentSignatureHeader) != "" {
			t.Error("paid request was sent despite recovery persistence failure")
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
		writer.WriteHeader(http.StatusPaymentRequired)
	}))
	defer server.Close()
	endpoint = server.URL + "/priced"
	adapter, err := New(server.Client(), &fixtureSigner{})
	require.NoError(t, err)
	cmd := command(t, endpoint, 1)
	cmd.RecordRecoveryInput = func(context.Context, string, string) error { return errors.New("disk unavailable") }

	result, err := adapter.Settle(context.Background(), cmd)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeFailed, result.Outcome)
	require.Equal(t, "x402_recovery_not_durable", result.Basis)
	require.EqualValues(t, 1, attempts.Load(), "only the initial 402 price fetch is allowed")
}

func TestQueryOutcomeRejectsChangedEndpointPathBeforeReplay(t *testing.T) {
	var endpoint string
	var attempts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts.Add(1)
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	endpoint = server.URL + "/priced/original"
	adapter, err := New(server.Client(), &fixtureSigner{})
	require.NoError(t, err)
	cmd := command(t, endpoint, 1)
	first, err := adapter.Settle(context.Background(), cmd)
	require.Error(t, err)
	require.Equal(t, rail.OutcomeUnknown, first.Outcome)
	requestsBeforeQuery := attempts.Load()

	credential := Credential{
		EndpointURL: server.URL + "/priced/rotated", SignerURL: "http://127.0.0.1:8545",
		Account: "0x1111111111111111111111111111111111111111", Networks: []string{"eip155:8453"},
		Assets: map[string]AssetPolicy{"0x2222222222222222222222222222222222222222": {Decimals: 6, Currency: "USD"}},
	}
	credentialJSON, err := json.Marshal(credential)
	require.NoError(t, err)
	_, err = adapter.QueryOutcome(context.Background(), rail.Query{
		ExternalID: first.ExternalID, ReceiptReference: first.ReceiptReference,
		IdempotencyKey: cmd.IdempotencyKey, Counterparty: cmd.Counterparty, Credential: string(credentialJSON),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exact endpoint")
	require.Equal(t, requestsBeforeQuery, attempts.Load(), "changed path must be rejected before replay HTTP")
}

func TestQueryOutcomeKeepsMismatchedFacilitatorIdentityUnknown(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payer   string
		network string
	}{
		{name: "different payer", payer: "0x9999999999999999999999999999999999999999", network: "eip155:8453"},
		{name: "different network", payer: "0x1111111111111111111111111111111111111111", network: "eip155:84532"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var endpoint string
			var attempts atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Header.Get(paymentSignatureHeader) == "" {
					writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
					writer.WriteHeader(http.StatusPaymentRequired)
					return
				}
				if attempts.Add(1) == 1 {
					writer.WriteHeader(http.StatusInternalServerError)
					return
				}
				receipt, err := json.Marshal(map[string]any{"success": true, "payer": tc.payer, "transaction": "0xabc123", "network": tc.network})
				require.NoError(t, err)
				writer.Header().Set(paymentResponseHeader, base64.StdEncoding.EncodeToString(receipt))
				writer.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			endpoint = server.URL + "/priced"

			adapter, err := New(server.Client(), &fixtureSigner{})
			require.NoError(t, err)
			cmd := command(t, endpoint, 1)
			first, err := adapter.Settle(context.Background(), cmd)
			require.Error(t, err)
			require.Equal(t, rail.OutcomeUnknown, first.Outcome)

			credential, err := json.Marshal(Credential{
				EndpointURL: endpoint, SignerURL: "http://127.0.0.1:8545",
				Account: "0x1111111111111111111111111111111111111111", Networks: []string{"eip155:8453"},
				Assets: map[string]AssetPolicy{"0x2222222222222222222222222222222222222222": {Decimals: 6, Currency: "USD"}},
			})
			require.NoError(t, err)
			reconciled, err := adapter.QueryOutcome(context.Background(), rail.Query{
				ExternalID: first.ExternalID, ReceiptReference: first.ReceiptReference,
				IdempotencyKey: cmd.IdempotencyKey, Counterparty: cmd.Counterparty, Credential: string(credential),
			})
			require.NoError(t, err)
			require.Equal(t, rail.OutcomeUnknown, reconciled.Outcome)
			require.Equal(t, "x402_replay_inconclusive", reconciled.Basis)
		})
	}
}

// [REQ:TRS-P1-001] A success-shaped facilitator response naming another payer
// cannot turn a first payment into settled evidence.
func TestOutboundKeepsMismatchedFacilitatorPayerUnknown(t *testing.T) {
	var endpoint string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get(paymentSignatureHeader) == "" {
			writer.Header().Set(paymentRequiredHeader, paymentRequiredValue(t, endpoint, "10000"))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		receipt, err := json.Marshal(map[string]any{"success": true, "payer": "0x9999999999999999999999999999999999999999", "transaction": "0xabc123", "network": "eip155:8453"})
		require.NoError(t, err)
		writer.Header().Set(paymentResponseHeader, base64.StdEncoding.EncodeToString(receipt))
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	endpoint = server.URL + "/priced"

	adapter, err := New(server.Client(), &fixtureSigner{})
	require.NoError(t, err)
	result, err := adapter.Settle(context.Background(), command(t, endpoint, 1))
	require.Error(t, err)
	require.Equal(t, rail.OutcomeUnknown, result.Outcome, "an untrusted facilitator payer cannot be recorded as settled")
	require.Equal(t, "x402_payment_dispatched", result.Basis)
	require.NotEmpty(t, result.ReceiptReference, "retain the exact payment for safe reconciliation")
}
