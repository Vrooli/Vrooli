package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"command-center/upstream"
	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	capreg "github.com/vrooli/vrooli/packages/capability-registry-go"
	walkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/walk"
	walkconnect "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/walk/walk_v1connect"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
)

func TestWalkRetainsSourceLedgerBriefingAcrossProviderLossAndRecovery(t *testing.T) {
	ledger, _ := walkTestLedger(t)
	runWalkProviderRecoveryJourney(t, ledger.ledger, false, nil, "")
}

// Invoked by Source Ledger's opt-in crossproduct test for each supported
// Journal owner selection mode against the same pre-seeded durable briefing.
func TestWalkRuntimeSourceLedgerConfigurationRecovery(t *testing.T) {
	mode := os.Getenv("SE4_WALK_RUNTIME_MODE")
	if mode == "" {
		t.Skip("run by Source Ledger's -tags=crossproduct owner integration")
	}
	if os.Getenv("SE4_EXPECTED_BRIEFING_ID") == "" {
		t.Fatal("cross-product owner must provide the seeded briefing ID")
	}
	briefingID := os.Getenv("SE4_EXPECTED_BRIEFING_ID")
	var resolveJournalBase func() string
	var lookups atomic.Int32
	switch mode {
	case "explicit-url":
		if os.Getenv("SOURCE_LEDGER_BASE_URL") == "" || os.Getenv("SOURCE_LEDGER_API_PORT") != "" {
			t.Fatal("explicit-url mode requires only SOURCE_LEDGER_BASE_URL")
		}
	case "port-fallback":
		if os.Getenv("SOURCE_LEDGER_BASE_URL") != "" || os.Getenv("SOURCE_LEDGER_API_PORT") == "" {
			t.Fatal("port-fallback mode requires only SOURCE_LEDGER_API_PORT")
		}
		resolver := discovery.NewResolver(discovery.ResolverConfig{
			VrooliPath: "isolated-vrooli",
			CommandRunner: func(_ context.Context, name string, args ...string) ([]byte, error) {
				if name != "isolated-vrooli" || len(args) != 4 || args[0] != "scenario" || args[1] != "port" || args[2] != "source-ledger" || args[3] != "API_PORT" {
					return nil, errors.New("unexpected api-core resolver invocation")
				}
				lookups.Add(1)
				return nil, errors.New("isolated scenario discovery unavailable")
			},
		})
		resolveJournalBase = resolveScenarioBaseURLUsing("source-ledger", "SOURCE_LEDGER_BASE_URL", "SOURCE_LEDGER_API_PORT", resolver.ResolveScenarioURLDefault)
		want := "http://localhost:" + os.Getenv("SOURCE_LEDGER_API_PORT")
		if got := resolveJournalBase(); got != want || lookups.Load() != 1 {
			t.Fatalf("api-core discovery failure fallback = %q after %d lookups, want %q after one discovery lookup", got, lookups.Load(), want)
		}
	default:
		t.Fatalf("unsupported Journal owner mode %q", mode)
	}
	if mode == "port-fallback" {
		// The preflight lookup plus both State reads must use the same fallback resolver.
		if lookups.Load() != 1 {
			t.Fatalf("resolver preflight calls = %d, want 1", lookups.Load())
		}
	}
	runWalkProviderRecoveryJourney(t, nil, true, resolveJournalBase, briefingID)
	if mode == "port-fallback" && lookups.Load() < 3 {
		t.Fatalf("runtime route resolved Journal owner only %d times, want preflight plus both State reads", lookups.Load())
	}
}

func runWalkProviderRecoveryJourney(t *testing.T, ledger journalconnect.JournalServiceClient, useSavedBriefing bool, resolveJournalBase func() string, expectedBriefingID string) {
	t.Helper()
	providerDown := atomic.Bool{}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if providerDown.Load() {
			http.Error(w, "authorization=synthetic-secret response=private", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/health" {
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
			return
		}
		if r.URL.Path != "/api/v1/stats" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprintf(w, `{"observed_at":%q,"throughput":{"completed_last_7_days":37}}`, time.Now().UTC().Format(time.RFC3339))
	}))
	defer provider.Close()

	metric := MetricEntry{
		ID: "throughput", Label: "Swarm throughput", Unit: "count", Coverage: CoverageNow, DataSource: StatusLive, UpstreamSource: SourceSwarm, TTLSeconds: 5,
		Source: SourceBinding{Binding: "scenario:swarm-manager", Read: "/api/v1/stats", IntegrationID: "swarm-manager", FeatureID: "swarm_throughput", Selector: "swarm_throughput", ContractVersion: "portfolio.v1", ExpectedUnit: "count", SourceTimePolicy: "producer_required", TTLSeconds: 5},
	}
	registry := &Registry{Version: "1.0.0", Metrics: []MetricEntry{metric}}
	s := NewServer(registry)
	s.instrumentProvider = nil
	s.objectiveProvider = nil
	providerClient := upstream.NewRESTResolved("swarm-manager", func() string { return provider.URL }, "")
	s.providers[SourceSwarm] = upstreamProvider{client: func() upstream.Client { return providerClient }, defaultPath: "/api/v1/stats"}
	def := capreg.Def{
		ID: "swarm-manager", Origin: "manifest", Name: "Swarm Manager", Description: "Throughput observation owner.",
		DependencyKind: capreg.DependencyScenario, DependencySlug: "swarm-manager", Features: []string{"swarm_throughput"}, Enabled: true,
		ActionKind: capreg.ActionKindOwnerGuidance, ActionLabel: "Review Swarm Manager", OperatorCommand: "vrooli scenario status swarm-manager --json",
	}
	checker := integrationChecker{client: providerClient, features: def.Features, slug: def.DependencySlug, label: def.Name, failureActionKind: capreg.ActionKindOwnerGuidance}
	s.integrationRegistry = capreg.New([]capreg.Def{def}, map[string]capreg.Checker{"swarm-manager": checker}, 20*time.Millisecond)
	s.integrationRegistry.SetLivenessCheckers(map[string]capreg.Checker{"swarm-manager": checker})

	_, handler := walkconnect.NewWalkServiceHandler(walkConnectService{server: s, ledger: ledger, resolveJournalBase: resolveJournalBase})
	server := httptest.NewServer(handler)
	defer server.Close()
	client := walkconnect.NewWalkServiceClient(server.Client(), server.URL)
	ctx := context.Background()

	read := func() *walkv1.ReadResponse {
		t.Helper()
		resp, err := client.Read(ctx, connect.NewRequest(&walkv1.ReadRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		return resp.Msg
	}
	state := func() *walkv1.StateResponse {
		t.Helper()
		resp, err := client.State(ctx, connect.NewRequest(&walkv1.StateRequest{Channel: "test"}))
		if err != nil {
			t.Fatal(err)
		}
		return resp.Msg
	}

	before := read()
	if len(before.GetReadings()) != 1 || before.GetReadings()[0].GetTrust() != string(TrustValid) || before.GetReadings()[0].GetValue().GetNumberValue() != 37 {
		t.Fatalf("healthy source read = %+v", before.GetReadings())
	}
	briefing := "Saved walk: investigate the throughput change and resume at phase 5.3."
	var entryID string
	if useSavedBriefing {
		saved := state().GetBriefing()
		if saved.GetEntryId() == "" || !strings.Contains(saved.GetBody(), briefing) || (expectedBriefingID != "" && saved.GetEntryId() != expectedBriefingID) {
			t.Fatalf("runtime port fallback did not select the pre-seeded Journal owner: %+v", saved)
		}
		entryID = saved.GetEntryId()
	} else {
		appendResponse, err := client.Publish(ctx, connect.NewRequest(walkJourneyPublishRequest(t, "durable-briefing", briefing)))
		if err != nil {
			t.Fatal(err)
		}
		entryID = appendResponse.Msg.GetEntryId()
		if entryID == "" {
			t.Fatal("Source Ledger did not return a saved briefing receipt")
		}
	}

	time.Sleep(5100 * time.Millisecond) // Expire the five-second metric cache while keeping producer time within its two-TTL stale window.
	providerDown.Store(true)
	degraded := read()
	if len(degraded.GetReadings()) != 1 {
		t.Fatalf("provider loss discarded the reading declaration: %+v", degraded.GetReadings())
	}
	reading := degraded.GetReadings()[0]
	if reading.GetTrust() != string(TrustCached) {
		t.Fatalf("provider loss trust = %q", reading.GetTrust())
	}
	if reading.GetValue().GetNumberValue() != 37 {
		t.Fatalf("provider loss discarded last qualified metric value: %+v", reading)
	}
	if reading.GetReason() != "swarm-manager unavailable (upstream_unavailable); showing last qualified observation" {
		t.Fatalf("provider loss reason = %q", reading.GetReason())
	}
	if strings.Contains(reading.GetReason(), "synthetic-secret") || strings.Contains(reading.GetReason(), "private") {
		t.Fatalf("provider diagnostics escaped into agent-facing read: %q", reading.GetReason())
	}
	savedDuringLoss := state().GetBriefing()
	if savedDuringLoss.GetEntryId() != entryID || !strings.Contains(savedDuringLoss.GetBody(), briefing) {
		t.Fatalf("saved Source Ledger work changed during provider loss: %+v", savedDuringLoss)
	}

	providerDown.Store(false)
	recovered := read()
	if len(recovered.GetReadings()) != 1 || recovered.GetReadings()[0].GetTrust() != string(TrustValid) || recovered.GetReadings()[0].GetValue().GetNumberValue() != 37 {
		t.Fatalf("provider recovery read = %+v", recovered.GetReadings())
	}
	savedAfterRecovery := state().GetBriefing()
	if savedAfterRecovery.GetEntryId() != entryID || savedAfterRecovery.GetBody() != savedDuringLoss.GetBody() {
		t.Fatalf("saved Source Ledger work changed after provider recovery: before=%+v after=%+v", savedDuringLoss, savedAfterRecovery)
	}
}

func walkJourneyPublishRequest(t *testing.T, requestKey, briefing string) *walkv1.PublishRequest {
	t.Helper()
	phases := make([]map[string]string, 0, len(walkPhases))
	for _, phase := range walkPhases {
		phases = append(phases, map[string]string{"phase": phase})
	}
	envelope, err := json.Marshal(map[string]any{
		"program": "command-center.vision-walk-prep", "status": "partial",
		"signals": map[string]any{"generated_at": time.Now().UTC().Format(time.RFC3339Nano), "phases": phases, "checkpoint": map[string]string{"status": "none"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &walkv1.PublishRequest{
		Channel: "test", RequestKey: requestKey, ProgramId: "prog_cross_product", EnvelopeJson: string(envelope),
		Briefing: briefing, FleetHealthJson: `{"status":"unavailable","reason":"provider transition test"}`,
	}
}
