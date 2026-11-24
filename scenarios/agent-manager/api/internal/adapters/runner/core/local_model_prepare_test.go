package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

func TestPrepareManagedLocalOpenCodeUsesExistingSelectionBoundary(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cfg := &domain.RunConfig{
		RunnerType:   domain.RunnerTypeOpenCode,
		Model:        "llama-cpp/sensenova-loop-9b-q4_k_m",
		AllowedTools: []string{"read_file", "write_file"}, DeniedTools: []string{"git_push"},
		PolicySnapshot: &domain.ExecutionPolicySnapshot{SelectedCandidate: domain.ExecutionCandidate{
			RunnerType: domain.RunnerTypeOpenCode, ResourceRole: "code.local", Model: "llama-cpp/sensenova-loop-9b-q4_k_m",
		}},
	}
	before, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	err = prepareManagedLocalOpenCode(ctx, cfg, func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls++
		if name != "resource-opencode" || strings.Join(args, " ") != "local-model select --runtime llama-cpp --role code.local --json" {
			t.Fatalf("unexpected preparation command: %s %v", name, args)
		}
		return []byte(`{"status":"ready","runtime":"llama-cpp","role":"code.local","selected_model":"llama-cpp/sensenova-loop-9b-q4_k_m"}`), nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("prepareManagedLocalOpenCode() = %v, calls=%d", err, calls)
	}
	after, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("preparation changed Agent Manager's execution/permission config:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestPrepareManagedLocalOpenCodeLeavesOtherRunsAndPermissionConfigAlone(t *testing.T) {
	for _, cfg := range []*domain.RunConfig{
		{RunnerType: domain.RunnerTypeOpenCode, Model: "ollama/qwen3.5:9b"},
		{RunnerType: domain.RunnerTypeCodex, Model: "llama-cpp/sensenova-loop-9b-q4_k_m"},
	} {
		called := false
		if err := prepareManagedLocalOpenCode(context.Background(), cfg, func(context.Context, string, ...string) ([]byte, error) { called = true; return nil, nil }); err != nil || called {
			t.Fatalf("non-managed request changed by preparation: err=%v called=%t", err, called)
		}
	}
}

func TestPrepareManagedLocalOpenCodeRejectsWrongRoleAndReceipt(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cfg := &domain.RunConfig{RunnerType: domain.RunnerTypeOpenCode, Model: "llama-cpp/model", PolicySnapshot: &domain.ExecutionPolicySnapshot{SelectedCandidate: domain.ExecutionCandidate{ResourceRole: "code.default"}}}
	if err := prepareManagedLocalOpenCode(ctx, cfg, func(context.Context, string, ...string) ([]byte, error) { return nil, nil }); err == nil {
		t.Fatal("expected non-local role rejection")
	}
	cfg.PolicySnapshot.SelectedCandidate.ResourceRole = "code.local"
	if err := prepareManagedLocalOpenCode(ctx, cfg, func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"status":"ready","runtime":"llama-cpp","role":"code.local","selected_model":"llama-cpp/other"}`), nil
	}); err == nil {
		t.Fatal("expected selected identity mismatch rejection")
	}
}
