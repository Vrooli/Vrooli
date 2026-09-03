package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"command-center/upstream"
	"connectrpc.com/connect"
	capreg "github.com/vrooli/vrooli/packages/capability-registry-go"
	integrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/integrations/integrations_v1connect"
	integrationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type integrationActionRunner struct{}

func (integrationActionRunner) Run(context.Context, string, ...string) (capreg.CommandResult, error) {
	return capreg.CommandResult{Stdout: []byte(`{"success":true,"verdict":"ready"}`)}, nil
}

type unavailableIntegrationClient struct{}

func (unavailableIntegrationClient) Name() string { return "unavailable" }

func (unavailableIntegrationClient) Fetch(context.Context, string) (json.RawMessage, error) {
	return nil, upstream.ErrNotAvailable
}

func TestIntegrationSnapshotSeparatesLifecycleAndFeatures(t *testing.T) {
	setUnavailableUpstreams(t)
	s := NewServer(testRegistry())
	snap := s.integrationSnapshot(context.Background(), false)
	if len(snap.States) != 4 {
		t.Fatalf("states = %d, want 4", len(snap.States))
	}
	for _, state := range snap.States {
		if state.ID == "prompt-manager" && state.Status != "unavailable" {
			t.Fatalf("prompt state = %+v", state)
		}
		if state.CheckedAt == "" {
			t.Fatalf("state %q has no checked time", state.ID)
		}
	}
}

func TestControlPlaneOutageOffersOperatorGuidanceNotScenarioRecovery(t *testing.T) {
	setUnavailableUpstreams(t)
	s := NewServer(testRegistry())
	for _, state := range s.integrationSnapshot(context.Background(), true).States {
		if state.ID != "vrooli-core" {
			continue
		}
		if state.ActionKind != capreg.ActionKindOwnerGuidance {
			t.Fatalf("control-plane action kind = %q, want owner guidance", state.ActionKind)
		}
		if state.OperatorCommand != "vrooli status --json" {
			t.Fatalf("control-plane operator command = %q", state.OperatorCommand)
		}
		return
	}
	t.Fatal("vrooli-core state not found")
}

func TestIntegrationCheckerDefaultsToOwnerGuidance(t *testing.T) {
	result := (integrationChecker{client: unavailableIntegrationClient{}, label: "Unconfigured"}).CheckResult(context.Background())
	if result.ActionKind != capreg.ActionKindOwnerGuidance {
		t.Fatalf("default failure action = %q, want owner guidance", result.ActionKind)
	}
	if result.ActionLabel != "Review Unconfigured" || result.OperatorCommand != "vrooli status --json" {
		t.Fatalf("default guidance = label %q command %q", result.ActionLabel, result.OperatorCommand)
	}
}

func TestIntegrationRegistryFailsClosedOnInvalidManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COMMAND_CENTER_SERVICE_MANIFEST", path)
	defer func() {
		if recover() == nil {
			t.Fatal("expected invalid manifest to fail startup")
		}
	}()
	commandCenterIntegrationRegistry(&Server{})
}

func TestOutcomeBindingsReferenceDeclaredIntegrationFeatures(t *testing.T) {
	reg, err := LoadRegistry("../config/outcome-registry.json")
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer(reg)
	if err := validateOutcomeBindings(s.integrationRegistry.Definitions(), reg.Metrics); err != nil {
		t.Fatal(err)
	}
	if s.integrationRegistry.CacheTTL() != 5*time.Second {
		t.Fatalf("unexpected integration cache ttl: %v", s.integrationRegistry.CacheTTL())
	}
}

func TestOutcomeBindingsRejectLiveCoverageWithoutSource(t *testing.T) {
	defs := []capreg.Def{{ID: "swarm-manager", Features: []string{"throughput_stats"}}}
	metrics := []MetricEntry{{
		ID:       "unwired",
		Coverage: CoverageInReach,
		Source: SourceBinding{
			IntegrationID: "none",
			FeatureID:     "throughput_stats",
		},
	}}
	if err := validateOutcomeBindings(defs, metrics); err == nil || !strings.Contains(err.Error(), "has no source") {
		t.Fatalf("validateOutcomeBindings() error = %v, want no-source coverage error", err)
	}
}

func TestIntegrationsConnectServiceUsesGeneratedContract(t *testing.T) {
	s := httptest.NewServer(NewServer(testRegistry()).Handler())
	defer s.Close()
	client := integrationconnect.NewIntegrationsServiceClient(http.DefaultClient, s.URL)
	resp, err := client.List(context.Background(), &connect.Request[integrationv1.ListIntegrationsRequest]{Msg: &integrationv1.ListIntegrationsRequest{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetIntegrations()) != 4 {
		t.Fatalf("integrations = %d, want 4", len(resp.Msg.GetIntegrations()))
	}
	for _, integration := range resp.Msg.GetIntegrations() {
		if integration.GetOrigin() == "" {
			t.Fatalf("integration %q has no shared-contract origin", integration.GetId())
		}
	}
}

func TestIntegrationsConnectGetIncludesIndependentFeatureState(t *testing.T) {
	setUnavailableUpstreams(t)
	s := httptest.NewServer(NewServer(testRegistry()).Handler())
	defer s.Close()
	client := integrationconnect.NewIntegrationsServiceClient(http.DefaultClient, s.URL)
	resp, err := client.Get(context.Background(), connect.NewRequest(&integrationv1.GetIntegrationRequest{IntegrationId: "swarm-manager"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetFeatures()) == 0 || resp.Msg.GetFeatures()[0].GetStatus() != integrationv1.FeatureStatus_FEATURE_STATUS_UNKNOWN {
		t.Fatalf("feature state = %+v, want independently unknown until a producer feature probe exists", resp.Msg.GetFeatures())
	}
}

func TestIntegrationWireActionIsNotEligibleWhileScenarioIsHealthy(t *testing.T) {
	state := capreg.State{
		Def:        capreg.Def{ID: "swarm-manager", DependencyKind: capreg.DependencyScenario, DependencySlug: "swarm-manager"},
		Status:     capreg.StatusAvailable,
		ActionKind: capreg.ActionKindScenarioStart,
	}
	message := messageFromState(state)
	if message.GetAction() == nil || message.GetAction().GetEligible() || message.GetAction().GetRequiresConfirmation() {
		t.Fatalf("healthy scenario action policy=%+v, want ineligible without confirmation", message.GetAction())
	}
}

func TestActionKindNameMapsSharedEnumsToLifecycleKinds(t *testing.T) {
	cases := []struct {
		name string
		wire integrationv1.ActionKind
		want capreg.ActionKind
	}{
		{name: "owner guidance", wire: integrationv1.ActionKind_ACTION_KIND_OWNER_GUIDANCE, want: capreg.ActionKindOwnerGuidance},
		{name: "scenario start", wire: integrationv1.ActionKind_ACTION_KIND_SCENARIO_START, want: capreg.ActionKindScenarioStart},
		{name: "scenario restart", wire: integrationv1.ActionKind_ACTION_KIND_SCENARIO_RESTART, want: capreg.ActionKindScenarioRestart},
		{name: "operator command", wire: integrationv1.ActionKind_ACTION_KIND_OPERATOR_COMMAND, want: capreg.ActionKindOperatorCommand},
		{name: "unspecified", wire: integrationv1.ActionKind_ACTION_KIND_UNSPECIFIED, want: capreg.ActionKindNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := actionKindName(tc.wire); got != tc.want {
				t.Fatalf("actionKindName(%v) = %q, want %q", tc.wire, got, tc.want)
			}
		})
	}
}

func TestIntegrationFeatureStateIsIndependentFromLifecycle(t *testing.T) {
	s := NewServer(testRegistry())
	r := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/swarm-manager/features/throughput_stats", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"lifecycleStatus"`) || !strings.Contains(w.Body.String(), `"featureId":"throughput_stats"`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestIntegrationActionRequiresConfirmationAndCannotMutateBusinessState(t *testing.T) {
	setUnavailableUpstreams(t)
	s := NewServer(testRegistry())
	s.actionService.Runner = integrationActionRunner{}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/swarm-manager/action", strings.NewReader(`{"action":"scenario_start"}`))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "confirmation") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/swarm-manager/action", strings.NewReader(`{"action":"scenario_start","confirm":true}`))
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "\"status\":\"ready\"") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/swarm-manager/action", strings.NewReader(`{"action":"arbitrary_command","confirm":true}`))
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "action_not_allowed") {
		t.Fatalf("arbitrary action status=%d body=%s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/not-declared/action", strings.NewReader(`{"action":"scenario_start","confirm":true}`))
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "integration_not_found") {
		t.Fatalf("undeclared integration status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestIntegrationActionRejectsHealthyScenarioRecovery(t *testing.T) {
	setUnavailableUpstreams(t)
	s := NewServer(testRegistry())
	s.swarm = staticUpstreamClient{
		name: "swarm",
		body: json.RawMessage(`{"status":"ok"}`),
	}
	s.integrationRegistry = commandCenterIntegrationRegistry(s)
	s.actionService.Defs = s.integrationRegistry.Definitions()
	s.actionService.Runner = integrationActionRunner{}

	r := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/swarm-manager/action", strings.NewReader(`{"action":"scenario_start","confirm":true}`))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "action_not_allowed") {
		t.Fatalf("healthy lifecycle action status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestIntegrationsConnectRejectsHealthyScenarioRecovery(t *testing.T) {
	setUnavailableUpstreams(t)
	s := NewServer(testRegistry())
	s.swarm = staticUpstreamClient{name: "swarm", body: json.RawMessage(`{"status":"ok"}`)}
	s.integrationRegistry = commandCenterIntegrationRegistry(s)
	s.actionService.Defs = s.integrationRegistry.Definitions()
	s.actionService.Runner = integrationActionRunner{}
	server := httptest.NewServer(s.Handler())
	defer server.Close()
	client := integrationconnect.NewIntegrationsServiceClient(http.DefaultClient, server.URL)
	_, err := client.RunAction(context.Background(), connect.NewRequest(&integrationv1.RunIntegrationActionRequest{
		IntegrationId: "swarm-manager",
		Action:        integrationv1.ActionKind_ACTION_KIND_SCENARIO_START,
		Confirmed:     true,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("healthy Connect lifecycle action error=%v, want permission denied", err)
	}
}

func setUnavailableUpstreams(t *testing.T) {
	t.Helper()
	t.Setenv("SWARM_MANAGER_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("LPBS_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("PROMPT_MANAGER_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("VROOLI_CORE_BASE_URL", "http://127.0.0.1:1")
}

func TestScenarioAddressResolutionHonorsExplicitRuntimeOverrides(t *testing.T) {
	t.Setenv("TEST_SCENARIO_BASE_URL", "http://scenario.example")
	resolve := resolveScenarioBaseURL("test-scenario", "TEST_SCENARIO_BASE_URL", "TEST_SCENARIO_API_PORT")
	if got := resolve(); got != "http://scenario.example" {
		t.Fatalf("base URL = %q, want explicit endpoint", got)
	}
	t.Setenv("TEST_SCENARIO_BASE_URL", "")
	t.Setenv("TEST_SCENARIO_API_PORT", "19321")
	if got := resolve(); got != "http://localhost:19321" {
		t.Fatalf("port override = %q, want explicit port", got)
	}
}

func TestControlPlaneAddressDoesNotGuessAStalePort(t *testing.T) {
	t.Setenv("VROOLI_CORE_BASE_URL", "")
	if got := resolveVrooliBaseURL(); got != "" {
		t.Fatalf("default control-plane URL = %q, want empty until configured", got)
	}
}
