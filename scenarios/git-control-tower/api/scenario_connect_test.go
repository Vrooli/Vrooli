package main

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/scenarios"
)

func TestScenarioConnectListUsesTypedScenarioStatusContract(t *testing.T) {
	swapCLIClient(t, stubRunner{out: []byte(`{"success":true,"scenarios":[{"name":"alpha","display_name":"Alpha","description":"first","status":"running","tags":["a"],"runtime":"registry"}]}`)})
	h := scenarioConnectServer{server: &Server{scenarioLocator: NewScenarioLocator(time.Minute)}}
	resp, err := h.ListScenarios(context.Background(), connect.NewRequest(&scenariosv1.ListScenariosRequest{}))
	if err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	if len(resp.Msg.GetScenarios()) != 1 {
		t.Fatalf("want one typed scenario, got %+v", resp.Msg.GetScenarios())
	}
	got := resp.Msg.GetScenarios()[0]
	if got.GetName() != "alpha" || got.GetDisplayName() != "Alpha" || got.GetStatus() != "running" || got.GetTags()[0] != "a" || got.HealthStatus != nil {
		t.Fatalf("scenario fields/presence changed: %+v", got)
	}
}

func TestScenarioConnectEnvelopeReadFromCachePreservesTypedFields(t *testing.T) {
	envelope := &ScenarioEnvelopeResponse{
		Name: "alpha", DisplayName: "Alpha", Description: "first", Path: "scenarios/alpha", Tags: []string{"tag"},
		Dependencies: ScenarioEnvelopeDependencies{Scenarios: map[string]string{"one": "scenario dep"}, Resources: map[string]string{"db": "database"}},
		Lifecycle:    ScenarioEnvelopeLifecycle{TestCommand: "go test ./...", BuildCommand: "make build"},
	}
	cache := NewEnvelopeCache(time.Minute)
	cache.Set("alpha", envelope)
	h := scenarioConnectServer{server: &Server{envelopeCache: cache}}
	resp, err := h.GetScenarioEnvelope(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioEnvelopeRequest{Slug: "alpha"}))
	if err != nil {
		t.Fatalf("GetScenarioEnvelope: %v", err)
	}
	got := resp.Msg
	if got.GetPath() != "scenarios/alpha" || got.GetDependencies().GetScenarios()["one"] != "scenario dep" || got.GetDependencies().GetResources()["db"] != "database" || got.GetLifecycle().GetTestCommand() != "go test ./..." || got.GetLifecycle().GetBuildCommand() != "make build" {
		t.Fatalf("envelope fields were not preserved: %+v", got)
	}
}

func TestScenarioConnectIsolationReadFromCachePreservesTypedViolations(t *testing.T) {
	cache := NewIsolationCache(time.Minute)
	cache.Put("alpha", IsolationResponse{Status: IsolationStatusNotRouted, Reasons: []string{"unmapped"}, Violations: []IsolationViolation{{RuleID: "rule-1", Severity: "error", File: "a.go", Line: 7, Excerpt: "line"}}})
	h := scenarioConnectServer{server: &Server{isolationCache: cache}}
	resp, err := h.GetScenarioIsolation(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioIsolationRequest{Slug: "alpha"}))
	if err != nil {
		t.Fatalf("GetScenarioIsolation: %v", err)
	}
	got := resp.Msg
	if got.GetStatus() != "not_routed" || got.GetReasons()[0] != "unmapped" || got.GetViolations()[0].GetRuleId() != "rule-1" || got.GetViolations()[0].GetFile() != "a.go" || got.GetViolations()[0].GetLine() != 7 || got.GetViolations()[0].GetExcerpt() != "line" {
		t.Fatalf("isolation details lost: %+v", got)
	}
}

func TestScenarioConnectIsolationRetainsUnknownWhenTestGenieIsUnavailable(t *testing.T) {
	h := scenarioConnectServer{server: &Server{}}
	resp, err := h.GetScenarioIsolation(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioIsolationRequest{Slug: "alpha"}))
	if err != nil {
		t.Fatalf("GetScenarioIsolation: %v", err)
	}
	if resp.Msg.GetStatus() != "unknown" || len(resp.Msg.GetReasons()) != 1 || resp.Msg.GetReasons()[0] != "test-genie eligibility client is not configured" {
		t.Fatalf("unavailable dependency did not remain explicit: %+v", resp.Msg)
	}
}

func TestScenarioConnectMethodsRejectMissingRequests(t *testing.T) {
	h := scenarioConnectServer{}
	if _, err := h.ListScenarios(context.Background(), nil); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("ListScenarios(nil) code = %v, want InvalidArgument", connect.CodeOf(err))
	}
	if _, err := h.GetScenarioEnvelope(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioEnvelopeRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetScenarioEnvelope(empty) code = %v, want InvalidArgument", connect.CodeOf(err))
	}
	if _, err := h.GetScenarioIsolation(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioIsolationRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetScenarioIsolation(empty) code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}
