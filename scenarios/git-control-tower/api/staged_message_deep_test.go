package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/stagedmessages"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/types/known/structpb"
)

func stagedDeepInput(count int) stagedmessages.DraftInput {
	in := stagedmessages.DraftInput{FormatVersion: 3, JobID: "deep-job", ContinuationOfJobID: "old-job", Snapshot: &v1.StagedMessageSnapshot{RepositoryId: "repo", Digest: "deep-subject"}, PriorResult: &v1.StagedMessageSuggestion{Title: "Previous draft", Body: "Preserve previously described behavior.", Coverage: &v1.StagedMessageCoverage{Partial: true}}}
	for i := 0; i < count; i++ {
		path := fmt.Sprintf("scenarios/scope-%02d/file-%04d.go", i%12, i)
		in.Snapshot.Files = append(in.Snapshot.Files, &v1.StagedMessageFile{Path: path, EditCount: 1, OmissionReason: "quick sample omitted this patch"})
		in.PriorResult.Coverage.UncoveredPaths = append(in.PriorResult.Coverage.UncoveredPaths, path)
	}
	return in
}

func deepExecutionFixture(t *testing.T, in stagedmessages.DraftInput) *ampb.WorkflowExecution {
	t.Helper()
	packed, err := packageStagedMessageInput(in, "/workspace/registered-fixture")
	if err != nil {
		t.Fatal(err)
	}
	input, err := structpb.NewValue(packed)
	if err != nil {
		t.Fatal(err)
	}
	return &ampb.WorkflowExecution{Id: "deep-execution", Owner: "git-control-tower", WorkflowKey: stagedMessageDeepWorkflowKey, IdempotencyKey: "original-key", Input: input, Status: ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING}
}

func TestDeepResultCanCover4096OnlyWithActualExecutionEvidence(t *testing.T) {
	in := stagedDeepInput(4096)
	execution := deepExecutionFixture(t, in)
	ids := []any{}
	eligible := map[int]bool{}
	for i := 1; i <= 4096; i++ {
		ids = append(ids, i)
		eligible[i] = true
	}
	result := map[string]any{"snapshot_digest": in.Snapshot.Digest, "title": "Update behavior across the staged selection", "body": "Describe all examined behavior groups.", "trailers": []any{}, "covered_file_ids": ids, "source_ids": []any{}, "unknowns": []any{}}
	execution.Output, _ = structpb.NewValue(map[string]any{"result": result})
	got, err := decodeStagedMessageResultWithEvidence(execution, in, eligible)
	if err != nil || got.Coverage.Partial || len(got.Coverage.CoveredPaths) != 4096 {
		t.Fatalf("full evidence still capped at quick sample: %v %v", got, err)
	}
	delete(eligible, 4096)
	if _, err := decodeStagedMessageResultWithEvidence(execution, in, eligible); err == nil {
		t.Fatal("never-supplied tail file accepted as covered")
	}
	raw, _ := json.Marshal(execution.Input.AsInterface())
	output, _ := json.Marshal(execution.Output.AsInterface())
	if len(raw) > stagedMessageMaxWorkflowInputBytes || len(output) > 60<<10 {
		t.Fatalf("deep owner bounds exceeded: input=%d output=%d", len(raw), len(output))
	}
	t.Logf("complete files=4096; AM input=%d; model result=%d", len(raw), len(output))
}

func TestEvidenceAuthorizationBindsOwnerBeforeHandleIsPersisted(t *testing.T) {
	in := stagedDeepInput(285)
	for _, name := range []string{"valid", "nil execution", "wrong execution", "wrong key", "wrong subject"} {
		t.Run(name, func(t *testing.T) {
			execution := deepExecutionFixture(t, in)
			switch name {
			case "nil execution":
				execution = nil
			case "wrong execution":
				execution.Id = "foreign"
			case "wrong key":
				execution.IdempotencyKey = "foreign-key"
			case "wrong subject":
				execution.Input, _ = structpb.NewValue(map[string]any{})
			}
			owner := &stagedWorkflowFake{execution: execution}
			p := stagedAgentFixture(owner, nil)
			err := p.AuthorizeEvidence(stagedAgentOwnerContext(), in, "original-key", "deep-execution")
			if (name == "valid") != (err == nil) {
				t.Fatalf("authorization result: %v", err)
			}
			if name != "valid" && connect.CodeOf(err) != connect.CodePermissionDenied {
				t.Fatalf("wrong binding not denied: %v", err)
			}
		})
	}
}

func TestLifecycleAdapterRejectsForeignSiblingHandle(t *testing.T) {
	for _, action := range []string{"get", "wait", "cancel"} {
		t.Run(action, func(t *testing.T) {
			owner := &stagedWorkflowFake{execution: stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING)}
			owner.execution.Id = "foreign-sibling"
			p := stagedAgentFixture(owner, nil)
			var err error
			switch action {
			case "get":
				_, err = p.Get(context.Background(), "original", stagedmessages.DraftInput{})
			case "wait":
				_, err = p.Wait(context.Background(), "original", stagedmessages.DraftInput{}, time.Second)
			case "cancel":
				_, err = p.Cancel(context.Background(), "original", stagedmessages.DraftInput{})
			}
			if err == nil || owner.cancels != 0 {
				t.Fatalf("foreign sibling accepted or cancelled: %v", err)
			}
		})
	}
}

func TestSynthesisResultSeparatesWholeMessageFromSelectiveInspection(t *testing.T) {
	for _, count := range []int{285, 4096} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			in := stagedDeepInput(count)
			in.FormatVersion = 4
			execution := deepExecutionFixture(t, in)
			inventory := map[int]bool{}
			for i := 1; i <= count; i++ {
				inventory[i] = true
			}
			evidence := map[int]bool{1: true, count: true}
			result := map[string]any{"snapshot_digest": in.Snapshot.Digest, "title": "Summarize the full selection across affected themes", "body": "Use important details without claiming every patch was read.", "trailers": []any{}, "covered_file_ids": []any{1, count}, "source_ids": []any{}, "unknowns": []any{}, "message_scope": "whole_selection", "message_scope_explanation": "The full immutable inventory identifies all scope groups; two important patches support the summary."}
			execution.Output, _ = structpb.NewValue(map[string]any{"result": result})
			got, err := decodeStagedMessageSynthesisResult(execution, in, evidence, inventory)
			if err != nil {
				t.Fatal(err)
			}
			if got.MessageScope != v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_WHOLE_SELECTION || !got.Coverage.Partial || len(got.Coverage.CoveredPaths) != 2 || len(got.Unknowns) != 0 {
				t.Fatalf("message scope inflated inspection: %v", got.Coverage)
			}
			delete(inventory, count)
			if _, err := decodeStagedMessageSynthesisResult(execution, in, evidence, inventory); err == nil {
				t.Fatal("whole selection accepted absent inventory tail")
			}
			result["message_scope"] = "limited"
			result["message_scope_explanation"] = "The final scope group was not available; its intent remains unknown."
			execution.Output, _ = structpb.NewValue(map[string]any{"result": result})
			if _, err := decodeStagedMessageSynthesisResult(execution, in, evidence, inventory); err != nil {
				t.Fatalf("truthful limited synthesis rejected: %v", err)
			}
			result["covered_file_ids"] = []any{1, 2, count}
			execution.Output, _ = structpb.NewValue(map[string]any{"result": result})
			if _, err := decodeStagedMessageSynthesisResult(execution, in, evidence, inventory); err == nil {
				t.Fatal("inventory substituted for inspected patch")
			}
		})
	}
}

func TestEvidenceAuthorizationExplicitlyReadsRetainedInputBeforeHandle(t *testing.T) {
	in := stagedDeepInput(285)
	in.FormatVersion = 4
	execution := deepExecutionFixture(t, in)
	owner := &stagedWorkflowFake{execution: execution}
	p := stagedAgentFixture(owner, nil)
	if err := p.AuthorizeEvidence(stagedAgentOwnerContext(), in, "original-key", execution.Id); err != nil {
		t.Fatal(err)
	}
	if owner.results != 1 || owner.gets != 0 {
		t.Fatalf("evidence binding used metadata instead of explicit payload read: metadata=%d payload=%d", owner.gets, owner.results)
	}
}

// [REQ:GCT-OT-P1-002] Retained execution 03731f67-87aa-4f77-be8d-c0b06011dce6
// delivered all 285 inventory entries and ten new patches, then returned file
// IDs as context references despite having no selected context. Preserve its
// prior/read/claimed ID sets with sanitized paths and message text.
func TestSynthesisKeepsFileInspectionSeparateFromEmptyContextSources(t *testing.T) {
	in := stagedDeepInput(285)
	in.FormatVersion = 4
	priorIDs := []int{1, 20, 39, 64, 71, 90, 115, 128, 153, 160, 185, 204, 223, 255, 263, 265, 266, 268, 274, 277, 278, 279, 281, 282, 283, 284}
	readIDs := []int{45, 51, 109, 141, 170, 191, 207, 261, 273, 275}
	claimedIDs := []int{1, 20, 39, 45, 51, 64, 71, 90, 115, 128, 153, 160, 185, 204, 223, 255, 263, 265, 266, 268, 273, 274, 277, 278, 279, 281, 282, 283, 284}
	prior, evidence, inventory := map[int]bool{}, map[int]bool{}, map[int]bool{}
	for _, id := range priorIDs {
		prior[id] = true
	}
	for _, id := range readIDs {
		evidence[id] = true
	}
	in.PriorResult.Coverage.CoveredPaths = nil
	in.PriorResult.Coverage.UncoveredPaths = nil
	for i, file := range in.Snapshot.Files {
		scope := "agent-manager"
		if i >= 262 {
			scope = "ai-gateway"
		}
		file.Path = fmt.Sprintf("scenarios/%s/api/file-%03d.go", scope, i+1)
		inventory[i+1] = true
		if prior[i+1] {
			in.PriorResult.Coverage.CoveredPaths = append(in.PriorResult.Coverage.CoveredPaths, file.Path)
		} else {
			in.PriorResult.Coverage.UncoveredPaths = append(in.PriorResult.Coverage.UncoveredPaths, file.Path)
		}
	}
	execution := deepExecutionFixture(t, in)
	result := stagedMessageCompactResult{
		SnapshotDigest: in.Snapshot.Digest,
		Title:          "Improve agent execution and inference behavior",
		Body:           "Coordinate lifecycle, conversation, supervision and inference changes across the staged selection.",
		Trailers:       []any{}, Unknowns: []string{}, CoveredFileIDs: claimedIDs,
		SourceIDs:               []string{"45", "51", "109", "170", "191", "273", "275"},
		MessageScope:            "whole_selection",
		MessageScopeExplanation: "The complete inventory and selected important patches support a summary of both affected behavior groups.",
	}
	decode := func() (*v1.StagedMessageSuggestion, error) {
		setCompactOutput(t, execution, result)
		return decodeStagedMessageSynthesisResult(execution, in, evidence, inventory)
	}
	if _, err := decode(); err == nil {
		t.Fatal("file IDs became source references despite empty selected context")
	}
	// Correct only the source namespace; retain the entire actual inspection set.
	result.SourceIDs = []string{}
	got, err := decode()
	if err != nil || got.MessageScope != v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_WHOLE_SELECTION || len(got.Coverage.CoveredPaths) != 29 || len(got.Coverage.UncoveredPaths) != 256 || len(got.SourceIds) != 0 {
		t.Fatalf("whole-selection synthesis lost supported sparse inspection: result=%v err=%v", got, err)
	}
	delete(inventory, 285)
	if _, err := decode(); err == nil {
		t.Fatal("whole-selection synthesis accepted missing inventory tail")
	}
	inventory[285] = true
	result.CoveredFileIDs = append(append([]int{}, claimedIDs...), 285)
	if _, err := decode(); err == nil {
		t.Fatal("inventory-only file became inspected without a prior or whole-patch receipt")
	}
}
