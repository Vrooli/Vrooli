package execution

import (
	"context"
	"path/filepath"
	"testing"

	"swarm-manager/internal/planclient"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

type scopePolicyRenderer struct {
	*fakeMarkdownRenderer
	status *executionv1.GetStatusResponse
}

func (r *scopePolicyRenderer) GetStatus(context.Context, *executionv1.GetStatusRequest) (*executionv1.GetStatusResponse, error) {
	return r.status, nil
}

func TestReadScopeExtensionsProjectsRecordedAllowAndEffectiveWriteScope(t *testing.T) {
	renderer := &scopePolicyRenderer{
		fakeMarkdownRenderer: &fakeMarkdownRenderer{},
		status: &executionv1.GetStatusResponse{Execution: &executionv1.Execution{
			BoundaryExtensions: []*executionv1.BoundaryExtension{{
				AddedAllow: []string{"scenarios/extra/**"}, Reason: "phase needs the companion", Author: "agent", CreatedAt: "2026-09-11T00:00:00Z",
			}},
		}},
	}
	service := &Service{planRenderer: renderer}
	item := backlogItem{AcceptanceAllow: []string{"scenarios/base/**"}, ScopePolicy: scopePolicyExtendWithRecord}
	record := Record{PlanManagerExecutionID: "plan-exec-1"}

	extensions, err := service.readScopeExtensions(context.Background(), record, item)
	if err != nil {
		t.Fatalf("read scope extensions: %v", err)
	}
	if len(extensions) != 1 || len(extensions[0].Paths) != 1 || extensions[0].Paths[0] != "scenarios/extra/**" {
		t.Fatalf("extensions = %#v", extensions)
	}
	if got := effectiveWriteScope(item, extensions); len(got) != 2 || got[0] != "scenarios/base/**" || got[1] != "scenarios/extra/**" {
		t.Fatalf("effective write scope = %v", got)
	}

	snapshot, err := buildPhasedPlanSnapshotWithScope(item, record, "plan-1", filepath.FromSlash("/repo"), renderedPlanContent{Plan: &sharedv1.Plan{Id: "plan-1"}}, extensions)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	value, err := snapshot.input()
	if err != nil {
		t.Fatalf("snapshot input: %v", err)
	}
	constraints := value.AsInterface().(map[string]any)["constraints"].(map[string]any)
	if constraints["scopePolicy"] != scopePolicyExtendWithRecord {
		t.Fatalf("scope policy constraint = %#v", constraints["scopePolicy"])
	}
	if got := constraints["writeScope"].([]any); len(got) != 2 || got[1] != "scenarios/extra/**" {
		t.Fatalf("write scope constraint = %v", got)
	}
}

func TestReadScopeExtensionsFixedDoesNotWiden(t *testing.T) {
	service := &Service{planRenderer: &scopePolicyRenderer{status: &executionv1.GetStatusResponse{}}}
	item := backlogItem{AcceptanceAllow: []string{"scenarios/base/**"}, ScopePolicy: scopePolicyFixed}
	extensions, err := service.readScopeExtensions(context.Background(), Record{PlanManagerExecutionID: "plan-exec-1"}, item)
	if err != nil {
		t.Fatalf("fixed policy read: %v", err)
	}
	if extensions != nil {
		t.Fatalf("fixed policy returned extensions: %#v", extensions)
	}
	if got := effectiveWriteScope(item, extensions); len(got) != 1 || got[0] != "scenarios/base/**" {
		t.Fatalf("fixed effective scope = %v", got)
	}
}

func TestReadScopeExtensionsRejectsAcceptanceDenyOverlap(t *testing.T) {
	service := &Service{planRenderer: &scopePolicyRenderer{
		status: &executionv1.GetStatusResponse{Execution: &executionv1.Execution{
			BoundaryExtensions: []*executionv1.BoundaryExtension{{AddedAllow: []string{"scenarios/secret/new.go"}}},
		}},
	}}
	item := backlogItem{ScopePolicy: scopePolicyExtendWithRecord, AcceptanceDeny: []string{"scenarios/secret/**"}}
	if _, err := service.readScopeExtensions(context.Background(), Record{PlanManagerExecutionID: "plan-exec-1"}, item); err == nil {
		t.Fatal("deny-overlapping extension was accepted")
	}
}

func TestFinalizationScopeIncludesRecordedExtensions(t *testing.T) {
	service := &Service{}
	scope, err := service.resolveFinalizationScope(context.Background(), Record{
		ScopeExtensions: []ScopeExtension{{Paths: []string{"scenarios/extended/**"}}},
	}, backlogItem{
		AcceptanceAllow: []string{"scenarios/base/**"},
		ScopePolicy:     scopePolicyExtendWithRecord,
	})
	if err != nil {
		t.Fatalf("resolve finalization scope: %v", err)
	}
	if len(scope.affectedScenarios) != 2 || scope.affectedScenarios[0] != "base" || scope.affectedScenarios[1] != "extended" {
		t.Fatalf("affected scenarios = %v", scope.affectedScenarios)
	}
}

var _ planclient.MarkdownRenderer = (*scopePolicyRenderer)(nil)
