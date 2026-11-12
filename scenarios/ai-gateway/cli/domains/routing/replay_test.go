package routing

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestReadReplayCasesRequiresRealSourceProvenance(t *testing.T) {
	valid := `{"case_id":"c1","source_run_id":"11111111-1111-4111-8111-111111111111","source_event_id":"22222222-2222-4222-8222-222222222222","source_sequence":3,"role":"chat.default","profile":"local-only","privacy_class":"internal","input_text":"Summarize this request"}`
	cases, err := ReadReplayCases(strings.NewReader(valid + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].SourceEventID != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("unexpected parsed cases: %#v", cases)
	}

	missingSource := `{"case_id":"c2","role":"chat.default","input_text":"generated prompt"}`
	if _, err := ReadReplayCases(strings.NewReader(missingSource)); err == nil || !strings.Contains(err.Error(), "source_run_id") {
		t.Fatalf("missing provenance error = %v, want source_run_id requirement", err)
	}
}

func TestReadReplayCasesAcceptsAgentManagerSequenceZeroOnly(t *testing.T) {
	row := `{"case_id":"zero","source_run_id":"11111111-1111-4111-8111-111111111111","source_event_id":"22222222-2222-4222-8222-222222222222","source_sequence":0,"role":"chat.default","profile":"local-only","privacy_class":"internal","input_text":"sequence zero caller"}`
	cases, err := ReadReplayCases(strings.NewReader(row))
	if err != nil || len(cases) != 1 || cases[0].SourceSequence != 0 {
		t.Fatalf("sequence-zero replay case = %#v, %v", cases, err)
	}
	negative := strings.Replace(row, `"source_sequence":0`, `"source_sequence":-1`, 1)
	if _, err := ReadReplayCases(strings.NewReader(negative)); err == nil || !strings.Contains(err.Error(), "nonnegative source_sequence") {
		t.Fatalf("negative source sequence error = %v", err)
	}
}

func TestReadReplayCasesRejectsDuplicateAndInvalidPolicy(t *testing.T) {
	row := `{"case_id":"same","source_run_id":"11111111-1111-4111-8111-111111111111","source_event_id":"22222222-2222-4222-8222-222222222222","source_sequence":3,"role":"chat.default","profile":"local-only","privacy_class":"internal","input_text":"q"}`
	if _, err := ReadReplayCases(strings.NewReader(row + "\n" + row)); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate id error = %v", err)
	}
	invalid := strings.Replace(row, "local-only", "not-a-profile", 1)
	if _, err := ReadReplayCases(strings.NewReader(invalid)); err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("invalid profile error = %v", err)
	}
}

func TestReplayCallerCasesBoundsConcurrencyAndPreservesInputOrder(t *testing.T) {
	cases := []ReplayCase{
		{CaseID: "slow", SourceRunID: "r1", SourceEventID: "e1", SourceSequence: 1, Role: "chat.default", InputText: "first", RequiredVRAMBytes: 1 << 20, CapacityEstimateEvidenceURI: "capacity-footprint://ollama/dynamic"},
		{CaseID: "fast", SourceRunID: "r2", SourceEventID: "e2", SourceSequence: 2, Role: "chat.small", InputText: "second", RequiredVRAMBytes: 1 << 20, CapacityEstimateEvidenceURI: "capacity-footprint://ollama/dynamic"},
	}
	var active, maxActive atomic.Int32
	execute := func(_ context.Context, req *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error) {
		if req.GetRequest().GetRequiredVramBytes() != 1<<20 {
			t.Fatalf("required vram = %d, want explicit 1 MiB admission", req.GetRequest().GetRequiredVramBytes())
		}
		current := active.Add(1)
		if current > maxActive.Load() {
			maxActive.Store(current)
		}
		for prev := maxActive.Load(); current > prev && !maxActive.CompareAndSwap(prev, current); prev = maxActive.Load() {
		}
		defer active.Add(-1)
		if req.GetInputText() == "first" {
			time.Sleep(15 * time.Millisecond)
		}
		return &routingv1.ExecuteRouteResponse{Valid: true, Evidence: &routingv1.RouteEvidence{
			EventId: "gateway-event-" + req.GetRequest().GetRole(), RequestId: "gateway-request-" + req.GetRequest().GetRole(),
			Status: "succeeded", SelectedModel: req.GetRequest().GetRole(), CapacityVerdict: "fit",
			CapacityClaimId: "claim-" + req.GetRequest().GetRole(), CapacityRequiredBytes: 1 << 20, CapacityGrantedBytes: 1 << 20,
		}}, nil
	}
	receipts, err := ReplayCallerCases(context.Background(), cases, 2, func(context.Context, ReplayCase) error { return nil }, execute)
	if err != nil {
		t.Fatal(err)
	}
	if maxActive.Load() != 2 {
		t.Fatalf("max concurrency = %d, want 2", maxActive.Load())
	}
	if receipts[0].CaseID != "slow" || receipts[1].CaseID != "fast" {
		t.Fatalf("replay order = %q, %q", receipts[0].CaseID, receipts[1].CaseID)
	}
	if receipts[0].SelectedModel != "chat.default" || receipts[1].Status != "succeeded" || receipts[0].GatewayEventID != "gateway-event-chat.default" || receipts[0].GatewayRequestID != "gateway-request-chat.default" || receipts[0].GatewayRoleEvidenceURI != "gateway-ledger://events/gateway-event-chat.default" || receipts[0].CapacityVerdict != "fit" || receipts[0].CapacityClaimID != "claim-chat.default" || receipts[0].CapacityRequestedBytes != 1<<20 {
		t.Fatalf("unexpected receipts: %#v", receipts)
	}
	if parallelErr := func() error {
		_, err := ReplayCallerCases(context.Background(), cases, maxReplayParallel+1, func(context.Context, ReplayCase) error { return nil }, execute)
		return err
	}(); parallelErr == nil {
		t.Fatal("parallelism above safety cap should fail")
	}
}

func TestReplayRequiresGatewayRouteIdentityToCaptureSourceLink(t *testing.T) {
	cases := []ReplayCase{{CaseID: "c", SourceRunID: "r", SourceEventID: "e", SourceSequence: 1, Role: "chat.default", InputText: "q", RequiredVRAMBytes: 1 << 20, CapacityEstimateEvidenceURI: "capacity-footprint://ollama/dynamic"}}
	receipts, err := ReplayCallerCases(context.Background(), cases, 1, func(context.Context, ReplayCase) error { return nil }, func(context.Context, *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error) {
		return &routingv1.ExecuteRouteResponse{Valid: true, Evidence: &routingv1.RouteEvidence{Status: "succeeded"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipts[0].Status != "error" || receipts[0].Error != "gateway route receipt omitted event or request identity" {
		t.Fatalf("receipt without Gateway identity was treated as captured: %+v", receipts[0])
	}
}

func TestReplayRequiresDeclaredCapacityBeforeVerifyingOrDispatching(t *testing.T) {
	cases := []ReplayCase{{CaseID: "c", SourceRunID: "r", SourceEventID: "e", SourceSequence: 1, Role: "chat.default", InputText: "q"}}
	verified, dispatched := 0, 0
	_, err := ReplayCallerCases(context.Background(), cases, 1, func(context.Context, ReplayCase) error { verified++; return nil }, func(context.Context, *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error) {
		dispatched++
		return nil, nil
	})
	if err == nil || verified != 0 || dispatched != 0 {
		t.Fatalf("err=%v verified=%d dispatched=%d; missing reservation must stop preflight", err, verified, dispatched)
	}
}

func TestReplayRequiresGatewayCapacityReceiptForLocalExecution(t *testing.T) {
	cases := []ReplayCase{{CaseID: "c", SourceRunID: "r", SourceEventID: "e", SourceSequence: 1, Role: "chat.default", InputText: "q", RequiredVRAMBytes: 1 << 20, CapacityEstimateEvidenceURI: "capacity-footprint://ollama/dynamic"}}
	receipts, err := ReplayCallerCases(context.Background(), cases, 1, func(context.Context, ReplayCase) error { return nil }, func(context.Context, *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error) {
		return &routingv1.ExecuteRouteResponse{Valid: true, Evidence: &routingv1.RouteEvidence{EventId: "route", RequestId: "request", Status: "succeeded"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipts[0].Status != "error" || receipts[0].Error != "gateway route receipt omitted verified capacity admission" {
		t.Fatalf("execution without admission was captured: %+v", receipts[0])
	}
}

func TestReplayCallerCasesVerifiesEverySourceBeforeDispatch(t *testing.T) {
	cases := []ReplayCase{{CaseID: "c1", RequiredVRAMBytes: 1 << 20, CapacityEstimateEvidenceURI: "capacity-footprint://ollama/dynamic"}, {CaseID: "c2", RequiredVRAMBytes: 1 << 20, CapacityEstimateEvidenceURI: "capacity-footprint://ollama/dynamic"}}
	verified := 0
	dispatched := 0
	_, err := ReplayCallerCases(context.Background(), cases, 2, func(_ context.Context, c ReplayCase) error {
		verified++
		if c.CaseID == "c2" {
			return context.DeadlineExceeded
		}
		return nil
	}, func(context.Context, *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error) {
		dispatched++
		return nil, nil
	})
	if err == nil || verified != 2 || dispatched != 0 {
		t.Fatalf("err=%v verified=%d dispatched=%d, want source failure before any dispatch", err, verified, dispatched)
	}
}

func TestPreviewCallerCasesVerifiesSourcesAndUsesNoModelExecution(t *testing.T) {
	cases := []ReplayCase{
		{CaseID: "first", SourceRunID: "11111111-1111-4111-8111-111111111111", SourceEventID: "22222222-2222-4222-8222-222222222222", SourceSequence: 1, Role: "chat.default", InputText: "first"},
		{CaseID: "second", SourceRunID: "33333333-3333-4333-8333-333333333333", SourceEventID: "44444444-4444-4444-8444-444444444444", SourceSequence: 2, Role: "chat.small", InputText: "second"},
	}
	var verified, previewed atomic.Int32
	receipts, err := PreviewCallerCases(context.Background(), cases, 2, func(_ context.Context, c ReplayCase) error {
		verified.Add(1)
		if c.SourceSequence < 0 {
			t.Fatalf("invalid source sequence: %#v", c)
		}
		return nil
	}, func(_ context.Context, req *routingv1.PreviewRouteRequest) (*routingv1.PreviewRouteResponse, error) {
		previewed.Add(1)
		if req.GetRequest().GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY {
			t.Fatalf("preview profile = %v, want local-only", req.GetRequest().GetProfile())
		}
		return &routingv1.PreviewRouteResponse{Valid: true, SelectedProvider: "ollama", RoutePlanId: req.GetRequest().GetRole()}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if int(verified.Load()) != len(cases) || int(previewed.Load()) != len(cases) {
		t.Fatalf("verified=%d previewed=%d", verified.Load(), previewed.Load())
	}
	if receipts[0].Mode != "preview" || receipts[0].LatencyKind != "route_preview" || receipts[0].SelectedProvider != "ollama" || receipts[0].Status != "valid" || receipts[0].SelectedModel != "" {
		t.Fatalf("unexpected preview receipt: %#v", receipts[0])
	}
}

func TestPreviewCallerCasesRejectsUnverifiedSourceBeforeAnyPreview(t *testing.T) {
	cases := []ReplayCase{{CaseID: "c1"}, {CaseID: "c2"}}
	dispatched := 0
	_, err := PreviewCallerCases(context.Background(), cases, 2, func(_ context.Context, c ReplayCase) error {
		if c.CaseID == "c2" {
			return context.DeadlineExceeded
		}
		return nil
	}, func(context.Context, *routingv1.PreviewRouteRequest) (*routingv1.PreviewRouteResponse, error) {
		dispatched++
		return &routingv1.PreviewRouteResponse{Valid: true}, nil
	})
	if err == nil || dispatched != 0 {
		t.Fatalf("err=%v previews=%d; want source rejection before dispatch", err, dispatched)
	}
}

func TestReplayProfileAndPrivacyDefaults(t *testing.T) {
	if got := replayProfile(""); got != sharedv1.Profile_PROFILE_LOCAL_FIRST {
		t.Fatalf("default profile = %v", got)
	}
	if got := replayPrivacy(""); got != sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL {
		t.Fatalf("default privacy = %v", got)
	}
	if replayProfile("invalid") != sharedv1.Profile_PROFILE_UNSPECIFIED {
		t.Fatal("invalid profile should remain unspecified")
	}
}
