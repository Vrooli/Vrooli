package model_eval

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

type approvalStub struct {
	answer, scope string
	err           error
}

func (s approvalStub) ReadDecision(context.Context, string) (Approval, error) {
	return Approval{DecisionID: AdmissionDecisionID, Answer: s.answer, Scope: s.scope, RecordedAt: time.Now().UTC()}, s.err
}
func (s approvalStub) ClaimDispatch(context.Context, Approval, string) error { return nil }

type sourceStub struct {
	refs   []SourceRef
	loads  int
	prompt string
}

func (s *sourceStub) Select(context.Context, int) ([]SourceRef, error) { return s.refs, nil }
func (s *sourceStub) VerifyAndLoad(_ context.Context, ref SourceRef) (VerifiedCase, error) {
	s.loads++
	timestamp, _ := time.Parse(time.RFC3339Nano, ref.Timestamp)
	return VerifiedCase{Ref: ref, Timestamp: timestamp, Source: "operator", Prompt: s.prompt}, nil
}

type privacyStub struct{ err error }

func (s privacyStub) ProveNoPromptOrOutputPersistence(context.Context) error { return s.err }

type diskStub struct {
	observation DiskObservation
	calls       int
}

func (s *diskStub) Measure(context.Context) (DiskObservation, error) {
	s.calls++
	return s.observation, nil
}

type capacityStub struct{ calls int }

func (s *capacityStub) MeasureAndCheck(_ context.Context, required int64) (CapacityObservation, error) {
	s.calls++
	return CapacityObservation{ObservedAt: time.Now().UTC(), Available: required, Fits: true}, nil
}

type gatewayStub struct {
	previews, executions int
	promptSeen           string
	output               string
}

func (s *gatewayStub) Preview(_ context.Context, _ *sharedv1.GatewayRequest) (*routingv1.PreviewRouteResponse, error) {
	s.previews++
	return &routingv1.PreviewRouteResponse{Valid: true, SelectedProvider: "ollama", Candidates: []*routingv1.RouteCandidate{{Provider: "ollama", Role: "chat.default", Locality: "local", Selected: true}}}, nil
}
func (s *gatewayStub) Execute(_ context.Context, req *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error) {
	s.executions++
	s.promptSeen = req.GetInputText()
	return &routingv1.ExecuteRouteResponse{Valid: true, OutputText: s.output, Evidence: &routingv1.RouteEvidence{
		EventId: "gateway-event", RequestId: req.GetRequest().GetRequestId(), TaskRole: req.GetRequest().GetTaskRole(), Status: "succeeded",
		SelectedLocality: "local", SelectedModel: "qwen3.5:9b", CapacityVerdict: "fit", CapacityClaimId: "claim-1",
		CapacityRequiredBytes: RequiredVRAMBytes, CapacityGrantedBytes: RequiredVRAMBytes,
	}}, nil
}

func testRef(t *testing.T) SourceRef {
	t.Helper()
	runID, eventID := uuid.NewString(), uuid.NewString()
	at := time.Date(2026, 11, 15, 3, 0, 0, 0, time.UTC)
	requestID, err := searcheval.AgentManagerReplayRequestID(runID, eventID, 7, "code.default", "agent-manager")
	if err != nil {
		t.Fatal(err)
	}
	return SourceRef{EventID: eventID, RunID: runID, Sequence: 7, Timestamp: at.Format(time.RFC3339Nano), TaskRole: "code.default", RequestID: requestID, PolicyRole: "chat.default", Consumer: "agent-manager"}
}
func testRunner(t *testing.T, answer string) (Runner, *sourceStub, *diskStub, *capacityStub, *gatewayStub) {
	t.Helper()
	ref := testRef(t)
	source := &sourceStub{refs: []SourceRef{ref}, prompt: "LM2_PROMPT_SENTINEL"}
	disk := &diskStub{observation: DiskObservation{ObservedAt: time.Now().UTC(), FreeBytes: MinimumDiskBytes, ArtifactHash: hashBytes([]byte("[]"))}}
	capacity := &capacityStub{}
	gateway := &gatewayStub{output: "LM2_OUTPUT_SENTINEL"}
	return Runner{Approval: approvalStub{answer: answer, scope: AdmissionScope}, Source: source, Selector: source, Privacy: privacyStub{}, Disk: disk, Capacity: capacity, Gateway: gateway}, source, disk, capacity, gateway
}

func TestModelEvalRunnerRefusesBeforeCurrentApprovalWithoutReadingSourceOrDispatching(t *testing.T) {
	for _, answer := range []string{"", "defer", "other"} {
		t.Run("answer_"+answer, func(t *testing.T) {
			runner, source, _, _, gateway := testRunner(t, answer)
			receipt, err := runner.Execute(context.Background(), Request{DecisionID: AdmissionDecisionID, MaxCases: 1})
			if !errors.Is(err, ErrAdmissionRefused) {
				t.Fatalf("Execute error = %v, want admission refusal", err)
			}
			if source.loads != 0 || gateway.executions != 0 {
				t.Fatalf("pre-approval touched source or dispatched: loads=%d calls=%d", source.loads, gateway.executions)
			}
			if receipt.Denominator != 0 {
				t.Fatalf("pre-approval receipt invented denominator: %+v", receipt)
			}
		})
	}
}

func TestModelEvalRunnerDispatchesSequentiallyAndReturnsMetadataOnlyReceipt(t *testing.T) {
	runner, _, _, capacity, gateway := testRunner(t, "approve")
	receipt, err := runner.Execute(context.Background(), Request{DecisionID: AdmissionDecisionID, MaxCases: 1})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gateway.executions != 1 || capacity.calls != 1 {
		t.Fatalf("calls=%d capacity=%d, want exactly one each", gateway.executions, capacity.calls)
	}
	if gateway.promptSeen != "LM2_PROMPT_SENTINEL" {
		t.Fatalf("prompt did not reach the public gateway: %q", gateway.promptSeen)
	}
	if receipt.Denominator != 1 || receipt.Succeeded != 1 || receipt.Failed+receipt.Refused+receipt.Unattempted != 0 {
		t.Fatalf("unexpected fixed denominator outcomes: %+v", receipt)
	}
	if receipt.ReceiptBytes <= 0 || !validHash(receipt.ReceiptSHA256) {
		t.Fatalf("receipt lacks a bounded metadata digest: %+v", receipt)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(encoded)) != receipt.ReceiptBytes || int64(len(encoded)) > MaximumArtifactSize {
		t.Fatalf("receipt bytes=%d encoded=%d", receipt.ReceiptBytes, len(encoded))
	}
	if strings.Contains(string(encoded), "LM2_PROMPT_SENTINEL") || strings.Contains(string(encoded), "LM2_OUTPUT_SENTINEL") {
		t.Fatalf("receipt leaked prompt/output: %s", encoded)
	}
	withoutDigest := receipt
	withoutDigest.ReceiptBytes = 0
	withoutDigest.ReceiptSHA256 = ""
	canonical, _ := json.Marshal(withoutDigest)
	if got := hashBytes(canonical); got != receipt.ReceiptSHA256 {
		t.Fatalf("receipt digest=%s want %s", receipt.ReceiptSHA256, got)
	}
}

func TestModelEvalRunnerFailsClosedWhenPrivacyProofIsUnavailable(t *testing.T) {
	runner, source, _, _, gateway := testRunner(t, "approve")
	runner.Privacy = privacyStub{err: errors.New("privacy proof unavailable")}
	receipt, err := runner.Execute(context.Background(), Request{DecisionID: AdmissionDecisionID, MaxCases: 1})
	if !errors.Is(err, ErrAdmissionRefused) {
		t.Fatalf("Execute error=%v", err)
	}
	if gateway.executions != 0 {
		t.Fatalf("privacy failure dispatched %d calls", gateway.executions)
	}
	if source.loads != 0 {
		t.Fatalf("privacy gate read %d source prompts before failing", source.loads)
	}
	if receipt.Denominator != 1 || receipt.Refused != 1 {
		t.Fatalf("privacy refusal did not preserve fixed denominator: %+v", receipt)
	}
}

func TestProductionGatewayPrivacyVerifierStaysClosedUntilCompleteBoundaryProof(t *testing.T) {
	if err := (GatewayPrivacyVerifier{}).ProveNoPromptOrOutputPersistence(context.Background()); err == nil {
		t.Fatal("production privacy verifier unexpectedly accepted an incomplete deployed-path proof")
	}
	if err := (GatewayPrivacyVerifier{Complete: true}).ProveNoPromptOrOutputPersistence(context.Background()); err != nil {
		t.Fatalf("complete proof should open the verifier: %v", err)
	}
}
