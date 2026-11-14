package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"git-control-tower/internal/stagedmessages"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func stagedLargeInput() stagedmessages.DraftInput {
	input := stagedmessages.DraftInput{FormatVersion: 2, Snapshot: &v1.StagedMessageSnapshot{Digest: "sha256:entire-4096-file-subject", RepositoryId: "registered-repository"}, OperatorNote: "Explain the complete coordinated change, including the generated outputs; disclose uninspected changes."}
	for i := 0; i < 4096; i++ {
		path := fmt.Sprintf("scenarios/product-%02d/%s/source-%04d.go", i%40, strings.Repeat("long-realistic-directory/", 20), i)
		file := &v1.StagedMessageFile{Path: path, OldBlobId: strings.Repeat("a", 40), NewBlobId: strings.Repeat("b", 40), EditCount: 2, OmissionReason: "staged content not sampled within bounded capture"}
		if i%64 == 0 {
			file.Patch = "@@ -1,2 +1,2 @@\n-return oldValue\n+return boundValue\n" + strings.Repeat(" unchanged context\n", 60)
			file.OmissionReason = ""
		}
		input.Snapshot.Files = append(input.Snapshot.Files, file)
	}
	for i := 0; i < 20; i++ {
		source := &v1.StagedMessageContext{Id: fmt.Sprintf("retained-source-%02d", i), Revision: fmt.Sprintf("sha256:source-revision-%02d", i), Kind: "proposal", Status: "retained_edits_included", FullMatch: true, Body: strings.Repeat("Retained purpose: support a coordinated change. ", 200)}
		for _, file := range input.Snapshot.Files[:64] {
			source.MatchedPaths = append(source.MatchedPaths, file.Path)
		}
		input.Contexts = append(input.Contexts, source)
	}
	return input
}

func compactExecution(t *testing.T, input stagedmessages.DraftInput) (*ampb.WorkflowExecution, stagedMessageCompactInput) {
	t.Helper()
	packaged, err := packageStagedMessageInput(input, "/workspace/registered-fixture")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(packaged)
	var projection stagedMessageCompactInput
	if err := json.Unmarshal(raw, &projection); err != nil {
		t.Fatal(err)
	}
	execution := stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED)
	execution.Input, _ = structpb.NewValue(packaged)
	setCompactOutput(t, execution, stagedMessageCompactResult{SnapshotDigest: input.Snapshot.Digest, Title: "Update coordinated repository behavior and generated outputs", Body: "Apply the retained changes across the complete staged selection. Representative patches support this summary; uninspected files remain explicitly uncovered.", Trailers: []any{}, CoveredFileIDs: []int{}, SourceIDs: []string{input.Contexts[0].Id}, Unknowns: []string{"Per-file implementation details are unavailable for the unsampled changes."}})
	return execution, projection
}

func setCompactOutput(t *testing.T, execution *ampb.WorkflowExecution, result stagedMessageCompactResult) {
	t.Helper()
	raw, _ := json.Marshal(result)
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	execution.Output, _ = structpb.NewValue(map[string]any{"result": value})
}

func TestStagedMessage4096FilesKeepCompleteSubjectAndBoundModelEnvelope(t *testing.T) {
	canonical := stagedLargeInput()
	before, _ := json.Marshal(canonical)
	execution, projection := compactExecution(t, canonical)
	input, _ := json.Marshal(execution.Input.AsInterface())
	// Optional synthetic contract handoff to the AM owner's declaration test.
	// The fixture contains no repository or conversation data from this host.
	if path := os.Getenv("GCT_STAGED_MESSAGE_INPUT_FIXTURE"); path != "" {
		if err := os.WriteFile(path, input, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	prompt, _ := json.Marshal(projection.DraftContext)
	output, _ := json.Marshal(execution.Output.AsInterface())
	after, _ := json.Marshal(canonical)
	if string(before) != string(after) || len(before) < 2<<20 {
		t.Fatal("fixture did not retain the large canonical subject or packaging mutated it")
	}
	if len(input) > 56<<10 || len(prompt) > 48<<10 || len(output) > 64<<10 {
		t.Fatalf("owner envelope exceeded: input=%d prompt=%d output=%d", len(input), len(prompt), len(output))
	}
	if projection.DraftContext.FileCount != 4096 || len(projection.DraftContext.Files) < 2 || len(projection.DraftContext.Files) > 64 || len(projection.DraftContext.Contexts) != 20 {
		t.Fatalf("bounded projection lost whole-selection identity: %+v", projection.DraftContext)
	}
	count := 0
	for _, scope := range projection.DraftContext.ScopeCounts {
		count += scope.Files
	}
	if count != 4096 {
		t.Fatal("scope summary dropped files")
	}
	for i, source := range projection.DraftContext.Contexts {
		if source.ID != canonical.Contexts[i].Id || source.Revision != canonical.Contexts[i].Revision || !source.ExcerptTruncated || source.Status != "retained_edits_included" || !source.FullMatch {
			t.Fatal("selected source identity, exact retained status, or excerpt limitation lost")
		}
	}
	result, err := decodeBoundStagedMessageResult(execution, canonical)
	if err != nil || !result.GetCoverage().GetPartial() || len(result.GetCoverage().GetUncoveredPaths()) != 4096 || result.GetCoverage().GetTotalEdits() != 8192 || result.GetCoverage().GetMatchedEdits() != 0 {
		t.Fatalf("uninspected files gained coverage or whole subject lost: %v %v", result, err)
	}
	expanded, _ := json.Marshal(result)
	if len(expanded) < 256<<10 || len(result.Unknowns) == 0 || !strings.Contains(result.Unknowns[0], "4096") {
		t.Fatal("server expansion did not pass the old result bound or omitted sampling disclosure")
	}
	t.Logf("canonical=%d AM envelope=%d prompt=%d model result=%d expanded result=%d detailed files=%d", len(before), len(input), len(prompt), len(output), len(expanded), len(projection.DraftContext.Files))
}

func TestStagedMessageCompactCoverageUsesActualSuppliedEvidence(t *testing.T) {
	canonical := stagedLargeInput()
	for _, test := range []string{"valid", "invented", "not supplied", "clipped patch", "wrong path", "changed canonical", "source revision", "other source", "wrong digest", "duplicate"} {
		t.Run(test, func(t *testing.T) {
			input := canonical
			input.Snapshot = proto.Clone(canonical.Snapshot).(*v1.StagedMessageSnapshot)
			if test == "clipped patch" {
				for _, file := range input.Snapshot.Files {
					if file.Patch != "" {
						file.Patch += strings.Repeat("staged context\n", 5000)
					}
				}
			}
			execution, projection := compactExecution(t, input)
			covered := projection.DraftContext.Files[0].ID
			result := stagedMessageCompactResult{SnapshotDigest: input.Snapshot.Digest, Title: "Update coordinated behavior", Trailers: []any{}, CoveredFileIDs: []int{covered}, SourceIDs: []string{input.Contexts[0].Id}, Unknowns: []string{}}
			switch test {
			case "invented":
				result.CoveredFileIDs = []int{4097}
			case "not supplied":
				// A real full-manifest ID still cannot be claimed if absent from
				// the ACTUAL immutable workflow projection, irrespective of sampling.
				projection.DraftContext.Files = projection.DraftContext.Files[1:]
			case "wrong path":
				projection.DraftContext.Files[0].Path = "invented/path.go"
			case "changed canonical":
				input.OperatorNote += " changed after admission"
			case "source revision":
				projection.DraftContext.Contexts[0].Revision = "other-revision"
			case "other source":
				result.SourceIDs = []string{"unselected-source"}
			case "wrong digest":
				result.SnapshotDigest = "other-selection"
			case "duplicate":
				result.CoveredFileIDs = []int{covered, covered}
			}
			raw, _ := json.Marshal(projection)
			var value any
			_ = json.Unmarshal(raw, &value)
			execution.Input, _ = structpb.NewValue(value)
			setCompactOutput(t, execution, result)
			got, err := decodeBoundStagedMessageResult(execution, input)
			if test != "valid" {
				if err == nil {
					t.Fatalf("unsupported coverage accepted: %s", test)
				}
				return
			}
			if err != nil || len(got.GetCoverage().GetCoveredPaths()) != 1 || len(got.Coverage.UncoveredPaths) != 4095 || got.Coverage.MatchedEdits != 2 {
				t.Fatalf("valid ID not mapped to exact complete canonical coverage: %v %v", got, err)
			}
			workflow := &stagedWorkflowFake{execution: execution}
			if _, err := stagedAgentFixture(workflow, nil).Get(context.Background(), execution.Id, input); err != nil || workflow.starts != 0 {
				t.Fatal("reconnect changed the immutable subject or launched another run")
			}
		})
	}
}

func TestStagedMessagePrefersWholeModeratePatchesDespiteHugeEdgeFiles(t *testing.T) {
	canonical := stagedLargeInput()
	var captured []*v1.StagedMessageFile
	for _, file := range canonical.Snapshot.Files {
		if file.Patch != "" {
			file.Patch = "@@ staged change @@\n" + strings.Repeat("+coordinated implementation\n", 100)
			captured = append(captured, file)
		}
	}
	captured[0].Patch = strings.Repeat("+huge edge change\n", 5000)
	captured[len(captured)-1].Patch = strings.Repeat("+huge last change\n", 5000)
	_, projection := compactExecution(t, canonical)
	complete, groups := 0, map[string]bool{}
	for _, detail := range projection.DraftContext.Files {
		if detail.CompletePatch {
			complete++
			groups[stagedMessageScope(detail.Path)] = true
			if detail.Patch != canonical.Snapshot.Files[detail.ID-1].Patch || len(detail.Patch) < 2500 {
				t.Fatal("moderate whole patch clipped before coverage could be supported")
			}
		}
	}
	if complete < 4 || len(groups) < 4 {
		t.Fatalf("huge edge files crowded out useful complete evidence: complete=%d groups=%d", complete, len(groups))
	}
	t.Logf("64 available patches with two huge lexical edges: complete supplied=%d scope groups=%d", complete, len(groups))
}

func TestStagedMessageBudgetTrimsTwentySourceExcerptsBeforeRejectingSelection(t *testing.T) {
	canonical := stagedLargeInput()
	for _, source := range canonical.Contexts {
		source.Body = strings.Repeat("\x01", 8192)
		source.Unknowns = []string{strings.Repeat("\x02", 512), strings.Repeat("\x03", 512)}
	}
	value, err := packageStagedMessageInput(canonical, "/registered/"+strings.Repeat("\x01", 3000))
	if err != nil {
		t.Fatalf("explanatory source bytes moved the capacity failure: %v", err)
	}
	raw, _ := json.Marshal(value)
	var input stagedMessageCompactInput
	_ = json.Unmarshal(raw, &input)
	if len(raw) > 56<<10 || len(input.DraftContext.Contexts) != 20 {
		t.Fatal("escaped root or selected source identities exceeded envelope")
	}
	for i, source := range input.DraftContext.Contexts {
		if source.ID != canonical.Contexts[i].Id || source.Revision != canonical.Contexts[i].Revision || !source.ExcerptTruncated {
			t.Fatal("budget reduction dropped selected identity or hid excerpt truncation")
		}
	}
	t.Logf("twenty large selected sources and escaped registered root: AM envelope=%d", len(raw))
}

func TestStagedMessageLegacyResultCannotBeReinterpretedAsCompactDraft(t *testing.T) {
	execution := stagedAgentComplete(t)
	canonical := stagedLegacyInput(t, execution)
	if _, err := decodeBoundStagedMessageResult(execution, canonical); err != nil {
		t.Fatal(err)
	}
	canonical.FormatVersion = 2
	if _, err := decodeBoundStagedMessageResult(execution, canonical); err == nil {
		t.Fatal("legacy workflow silently adopted for a new-format subject")
	}
	canonical.FormatVersion = 0
	canonical.OperatorNote = "different original request"
	if _, err := decodeBoundStagedMessageResult(execution, canonical); err == nil {
		t.Fatal("legacy result detached from its original canonical subject")
	}
}

func TestStagedMessageCompactStartAndLegacyLostAcknowledgementKeepTheirEncoding(t *testing.T) {
	canonical := stagedLargeInput()
	owner := &stagedWorkflowFake{execution: stagedAgentExecution(ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_PENDING)}
	provider := stagedAgentFixture(owner, nil)
	for i := 0; i < 2; i++ {
		if _, err := provider.Start(stagedAgentOwnerContext(), canonical, "same-admission-key"); err != nil {
			t.Fatal(err)
		}
		fields := owner.start.GetInput().GetStructValue().GetFields()
		if fields["format_version"].GetNumberValue() != 2 || fields["snapshot"] != nil || fields["draft_context"] == nil || owner.start.IdempotencyKey != "same-admission-key" {
			t.Fatal("compact start transmitted the full subject or changed its admission key")
		}
	}
	legacy := stagedAgentComplete(t)
	legacyInput := stagedLegacyInput(t, legacy)
	owner.execution = legacy
	if got, err := provider.Start(stagedAgentOwnerContext(), legacyInput, "original-legacy-key"); err != nil || got.Status != "succeeded" {
		t.Fatalf("lost legacy acknowledgement did not reconnect to original encoding: %v %v", got, err)
	}
	fields := owner.start.GetInput().GetStructValue().GetFields()
	if fields["format_version"] != nil || fields["snapshot"] == nil || owner.start.IdempotencyKey != "original-legacy-key" {
		t.Fatal("legacy admission was silently repackaged")
	}
	owner.startErr = connect.NewError(connect.CodeInternal, errors.New("workflow input: missing format_version"))
	if _, err := provider.Start(stagedAgentOwnerContext(), legacyInput, "not-yet-dispatched-legacy-key"); connect.CodeOf(err) != connect.CodeFailedPrecondition || !strings.Contains(err.Error(), "legacy") {
		t.Fatalf("unsupported legacy admission did not return a permanent truthful refresh instruction: %v", err)
	}
}
