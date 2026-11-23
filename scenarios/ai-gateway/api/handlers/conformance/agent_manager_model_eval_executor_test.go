package conformance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type modelEvalRoundTripper func(*http.Request) (*http.Response, error)

func (f modelEvalRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDefaultModelEvalWiringInstallsOwnerExecutor(t *testing.T) {
	deps := withDefaultModelEvalSource(Deps{})
	if _, ok := deps.ModelEvalExecutor.(*DiscoveredAgentManagerModelEvalExecutor); !ok {
		t.Fatalf("default model-eval executor = %T", deps.ModelEvalExecutor)
	}
}

func ownerReceiptResponse(t *testing.T, status string, receipt agentManagerModelEvalReceipt) []byte {
	t.Helper()
	unsigned := receipt
	unsigned.ReceiptBytes, unsigned.ReceiptSHA256 = 0, ""
	data, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	receipt.ReceiptSHA256 = hex.EncodeToString(digest[:])
	for i := 0; i < 4; i++ {
		data, err = json.Marshal(receipt)
		if err != nil {
			t.Fatal(err)
		}
		if receipt.ReceiptBytes == int64(len(data)) {
			break
		}
		receipt.ReceiptBytes = int64(len(data))
	}
	data, err = json.Marshal(agentManagerModelEvalResponse{State: status, Receipt: &receipt})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAgentManagerModelEvalExecutorForwardsOnlyDecisionAndPreservesMetadataReceipt(t *testing.T) {
	cases := []agentManagerModelEvalCaseReceipt{{
		SourceEventID: "11111111-1111-1111-1111-111111111111", SourceRunID: "22222222-2222-2222-2222-222222222222",
		SourceSequence: 3, EventTimestamp: "2026-11-15T03:00:00Z", RequestID: "request",
		TaskRole: "task", PolicyRole: "chat.default", Consumer: "agent-manager",
		EffectiveModel: "installed", Locality: "local", LatencyMillis: 12, RightsResult: "allowed",
		CapacityClaimID: "claim", RequiredBytes: agentManagerMinClaimBytes, GrantedBytes: agentManagerMinClaimBytes,
		Status: "succeeded",
	}}
	caseSetDigest, err := ownerCaseSetDigest(cases)
	if err != nil {
		t.Fatal(err)
	}
	wireReceipt := agentManagerModelEvalReceipt{
		DecisionID: agentManagerModelEvalID, DecisionScope: agentManagerModelEvalScope,
		CaseSetSHA256: caseSetDigest, Denominator: 1, Succeeded: 1,
		EffectiveModel: "installed", DiskFreeBefore: agentManagerMinDiskBytes,
		DiskFreeAfter: agentManagerMinDiskBytes, ArtifactHashBefore: strings.Repeat("b", 64),
		ArtifactHashAfter: strings.Repeat("c", 64),
		Cases:             cases,
	}
	client := &http.Client{Transport: modelEvalRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != agentManagerModelEvalPath {
			return nil, fmt.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		if got, want := string(body), `{"decision_id":"LM2-INFERENCE-ADMISSION-20261116","max_cases":20}`; got != want {
			return nil, fmt.Errorf("request body = %s, want %s", got, want)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(ownerReceiptResponse(t, "succeeded", wireReceipt)))}, nil
	})}
	executor := &DiscoveredAgentManagerModelEvalExecutor{HTTPClient: client, ResolveURL: func(context.Context) (string, error) { return "http://agent-manager.invalid", nil }}
	receipt, err := executor.ExecuteAgentManagerModelEval(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.DecisionID != agentManagerModelEvalID || receipt.DecisionScope != agentManagerModelEvalScope || receipt.Denominator != 1 || receipt.DiskFreeBytesBefore != agentManagerMinDiskBytes {
		t.Fatalf("receipt top-level metadata = %+v", receipt)
	}
	if len(receipt.Cases) != 1 || receipt.Cases[0].SourceSequence != 3 || receipt.Cases[0].CapacityGrantedBytes != agentManagerMinClaimBytes || receipt.Cases[0].Locality != "local" {
		t.Fatalf("receipt case metadata = %+v", receipt.Cases)
	}
}

func TestAgentManagerModelEvalExecutorReturnsRefusalReceiptWithoutSensitiveBody(t *testing.T) {
	cases := []agentManagerModelEvalCaseReceipt{{
		SourceEventID: "11111111-1111-1111-1111-111111111111", SourceRunID: "22222222-2222-2222-2222-222222222222",
		SourceSequence: 3, EventTimestamp: "2026-11-15T03:00:00Z", RequestID: "request",
		TaskRole: "task", PolicyRole: "chat.default", Consumer: "agent-manager",
		Status: "refused", RightsResult: "unknown", FailureCode: "prompt_output_non_persistence_unproven",
	}}
	caseSetDigest, err := ownerCaseSetDigest(cases)
	if err != nil {
		t.Fatal(err)
	}
	wireReceipt := agentManagerModelEvalReceipt{
		DecisionID: agentManagerModelEvalID, DecisionScope: agentManagerModelEvalScope,
		CaseSetSHA256: caseSetDigest, Denominator: 1, Refused: 1, Cases: cases,
	}
	client := &http.Client{Transport: modelEvalRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusPreconditionFailed, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(ownerReceiptResponse(t, "refused", wireReceipt)))}, nil
	})}
	executor := &DiscoveredAgentManagerModelEvalExecutor{HTTPClient: client, ResolveURL: func(context.Context) (string, error) { return "http://agent-manager.invalid", nil }}
	receipt, err := executor.ExecuteAgentManagerModelEval(context.Background())
	if err == nil || strings.Contains(err.Error(), "prompt") || receipt.Denominator != 1 || receipt.Refused != 1 || len(receipt.Cases) != 1 {
		t.Fatalf("refusal result receipt=%+v err=%v", receipt, err)
	}
}
