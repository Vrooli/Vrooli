package providers

import (
	"context"
	"testing"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

type sequencedRunner struct {
	results  []cleanup.ProcessResult
	commands []cleanup.ProcessCommand
}

func (r *sequencedRunner) Run(_ context.Context, command cleanup.ProcessCommand) (cleanup.ProcessResult, error) {
	r.commands = append(r.commands, command)
	if len(r.results) == 0 {
		return cleanup.ProcessResult{}, nil
	}
	result := r.results[0]
	r.results = r.results[1:]
	return result, nil
}

func TestUndeclaredWorkloadProviderPreviewAndApplyRequireApproval(t *testing.T) {
	runner := &cleanupfakes.ProcessRunner{Result: cleanup.ProcessResult{Stdout: `{"Names":"airbyte-abctl-control-plane","Image":"kindest/node:v1.32.2","State":"running"}` + "\n" + `{"Names":"operator-owned","Image":"vrooli/test","State":"running"}`}}
	p := NewUndeclaredWorkloadProvider(runner, UndeclaredWorkloadProviderConfig{HistoricalNames: map[string]string{"airbyte-abctl-control-plane": "agent experiment"}})
	preview, err := p.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 || preview.Items[0].Path != "airbyte-abctl-control-plane" {
		t.Fatalf("preview=%+v", preview)
	}
	blocked, err := p.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "test", ApprovalMode: cleanup.ApprovalModeOwner, Preview: preview})
	if err != nil {
		t.Fatal(err)
	}
	if len(blocked.SkippedItems) != 1 {
		t.Fatalf("expected approval gate: %+v", blocked)
	}
	result, err := p.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "test", ApprovalMode: cleanup.ApprovalModeOperator, Preview: preview})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied {
		t.Fatalf("expected apply: %+v", result)
	}
	if len(runner.Commands) < 2 || runner.Commands[1].Name != "docker" || runner.Commands[1].Args[0] != "rm" {
		t.Fatalf("unexpected commands: %+v", runner.Commands)
	}
}

func TestUndeclaredWorkloadProviderDisabledPolicyStillReturnsBlockedPreview(t *testing.T) {
	p := NewUndeclaredWorkloadProvider(&cleanupfakes.ProcessRunner{}, UndeclaredWorkloadProviderConfig{
		Posture:  "whole_host",
		Findings: []ProviderFinding{{Kind: "container", Name: "airbyte-abctl-control-plane", Class: "abandoned", Finding: true, Evidence: []string{"historical agent experiment"}}},
	})
	preview, err := p.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: false, ApprovalMode: cleanup.ApprovalModeOperator}})
	if err != nil {
		t.Fatal(err)
	}
	if preview.BlockedReason != "provider disabled by policy" {
		t.Fatalf("blocked reason = %q, want disabled policy", preview.BlockedReason)
	}
	if len(preview.Items) != 1 || preview.Items[0].Path != "airbyte-abctl-control-plane" {
		t.Fatalf("disabled preview = %#v, want one read-only candidate", preview.Items)
	}
}

func TestUndeclaredWorkloadProviderBrakesApplyWhenHostSaturated(t *testing.T) {
	runner := &cleanupfakes.ProcessRunner{}
	p := NewUndeclaredWorkloadProvider(runner, UndeclaredWorkloadProviderConfig{
		Findings:  []ProviderFinding{{Kind: "container", Name: "airbyte-abctl-control-plane", Class: "abandoned", Finding: true, Evidence: []string{"agent-experiments/airbyte"}}},
		Saturated: func(context.Context) (bool, error) { return true, nil },
	})
	preview, err := p.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "saturated", ApprovalMode: cleanup.ApprovalModeOperator, Preview: preview})
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied || len(result.SkippedItems) != 1 || len(runner.Commands) != 0 {
		t.Fatalf("saturated host must brake disposal: result=%+v commands=%+v", result, runner.Commands)
	}
}

func TestUndeclaredWorkloadProviderUsesClassificationAndNeverPreviewsUnmanaged(t *testing.T) {
	runner := &cleanupfakes.ProcessRunner{}
	p := NewUndeclaredWorkloadProvider(runner, UndeclaredWorkloadProviderConfig{
		Posture: "vrooli_only",
		Findings: []ProviderFinding{
			{Kind: "container", Name: "airbyte-abctl-control-plane", Class: "abandoned", Finding: true, Evidence: []string{"agent-experiments/airbyte-abctl-control-plane absent manifest"}, Swapped: 123},
			{Kind: "container", Name: "operator-owned", Class: "unmanaged", Finding: true, Evidence: []string{"no Vrooli declaration matched"}},
			{Kind: "container", Name: "postgis-main", Class: "declared", Finding: true, Evidence: []string{"enabled resource manifest: resources/postgis/resource.json"}},
		},
	})
	preview, err := p.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 || preview.Items[0].Path != "airbyte-abctl-control-plane" {
		t.Fatalf("classification preview=%+v", preview)
	}
	if len(runner.Commands) != 0 {
		t.Fatalf("classification-backed preview should not enumerate Docker: %+v", runner.Commands)
	}

	declared := cleanup.Preview{ProviderID: "undeclared-workload", ProviderVersion: "v1", Items: []cleanup.PreviewItem{{ID: "declared", Path: "postgis-main"}}}
	result, err := p.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "declared", ApprovalMode: cleanup.ApprovalModeOperator, Preview: declared})
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied || len(result.SkippedItems) != 1 {
		t.Fatalf("declared workload must be refused: %+v", result)
	}
}

func TestUndeclaredWorkloadProviderRequiresPathEvidenceForVrooliOnly(t *testing.T) {
	p := NewUndeclaredWorkloadProvider(&cleanupfakes.ProcessRunner{Result: cleanup.ProcessResult{Stdout: `{"Names":"looks-vrooli","Image":"x","State":"running"}` + "\n"}}, UndeclaredWorkloadProviderConfig{
		Posture:         "vrooli_only",
		HistoricalNames: map[string]string{"looks-vrooli": "name pattern only"},
	})
	preview, err := p.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 0 || len(preview.Warnings) != 1 {
		t.Fatalf("name-only evidence must be refused: %+v", preview)
	}
}

func TestUndeclaredWorkloadProviderUsesNativeActionForServiceUnit(t *testing.T) {
	runner := &cleanupfakes.ProcessRunner{}
	p := NewUndeclaredWorkloadProvider(runner, UndeclaredWorkloadProviderConfig{
		Findings: []ProviderFinding{{Kind: "service-unit", Name: "old-vrooli.service", Class: "abandoned", Finding: true, Evidence: []string{"enabled resource manifest: resources/old/resource.json"}}},
	})
	preview, err := p.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}})
	if err != nil || len(preview.Items) != 1 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	result, err := p.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "service", ApprovalMode: cleanup.ApprovalModeOperator, Preview: preview})
	if err != nil || !result.Applied {
		t.Fatalf("apply=%+v err=%v", result, err)
	}
	if len(runner.Commands) != 1 || runner.Commands[0].Name != "systemctl" {
		t.Fatalf("expected native service command, got %+v", runner.Commands)
	}
}

func TestUndeclaredWorkloadProviderVerifiesContainerDisposalIndependently(t *testing.T) {
	runner := &sequencedRunner{results: []cleanup.ProcessResult{
		{Stdout: `{"Names":"airbyte-abctl-control-plane"}` + "\n"},
		{}, // docker rm -f
		{}, // post-disposal docker ps -a
	}}
	p := NewUndeclaredWorkloadProvider(runner, UndeclaredWorkloadProviderConfig{
		HistoricalNames: map[string]string{"airbyte-abctl-control-plane": "agent-experiments/airbyte"},
	})
	preview, err := p.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}})
	if err != nil || len(preview.Items) != 1 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	result, err := p.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: "v1", IdempotencyKey: "verify-container", ApprovalMode: cleanup.ApprovalModeOperator, Preview: preview,
	})
	if err != nil || !result.Applied {
		t.Fatalf("apply=%+v err=%v", result, err)
	}
	verified, err := p.Verify(context.Background(), cleanup.VerifyRequest{ApplyResult: result})
	if err != nil || !verified.Verified {
		t.Fatalf("verify=%+v err=%v", verified, err)
	}
	if len(runner.commands) != 3 || runner.commands[2].Name != "docker" {
		t.Fatalf("expected independent post-disposal enumeration, commands=%+v", runner.commands)
	}
}
