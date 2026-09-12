package findings

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	findingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings"
	findingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings/findings_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

// fakeFindingsService is a stateful in-memory FindingsServiceHandler. It
// mirrors the documented server semantics the CLI depends on (see
// api/internal/findings/service.go + sqlite.go): claim required on add,
// status filter wins over include-archived on list, default list excludes
// superseded, prune targets superseded rows and honours dry-run. CLI tests
// drive the real Connect transport (manifest schema → handler → HTTP →
// fake) so they exercise the full command flow, not just the formatter.
type fakeFindingsService struct {
	findingsconnect.UnimplementedFindingsServiceHandler

	mu     sync.Mutex
	seq    int
	store  map[string]*findingsv1.Finding
	order  []string
	audits []string // "<mutation>:<id>:<reason>"

	listReqs  []*findingsv1.ListFindingsRequest
	pruneReqs []*findingsv1.PruneFindingsRequest
}

func newFakeFindingsService() *fakeFindingsService {
	return &fakeFindingsService{store: map[string]*findingsv1.Finding{}}
}

// seed inserts a finding directly (no audit row), returning its id.
func (s *fakeFindingsService) seed(claim string, status findingsv1.FindingStatus) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.insertLocked(claim, 0.5, status, nil)
}

func (s *fakeFindingsService) insertLocked(claim string, confidence float64, status findingsv1.FindingStatus, cites []*findingsv1.Citation) string {
	s.seq++
	id := fmt.Sprintf("f-%d", s.seq)
	s.store[id] = &findingsv1.Finding{
		Id:         id,
		Claim:      claim,
		Confidence: confidence,
		Status:     status,
		Citations:  cites,
		CreatedAt:  timestamppb.Now(),
	}
	s.order = append(s.order, id)
	return id
}

func (s *fakeFindingsService) get(id string) *findingsv1.Finding {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store[id]
}

func (s *fakeFindingsService) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.store)
}

func (s *fakeFindingsService) ListFindings(_ context.Context, req *connect.Request[findingsv1.ListFindingsRequest]) (*connect.Response[findingsv1.ListFindingsResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listReqs = append(s.listReqs, req.Msg)
	resp := &findingsv1.ListFindingsResponse{}
	for _, id := range s.order {
		f, ok := s.store[id]
		if !ok {
			continue
		}
		switch {
		case req.Msg.GetStatus() != findingsv1.FindingStatus_FINDING_STATUS_UNSPECIFIED:
			if f.Status != req.Msg.GetStatus() {
				continue
			}
		case !req.Msg.GetIncludeArchived():
			if f.Status == findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED {
				continue
			}
		}
		resp.Findings = append(resp.Findings, f)
	}
	return connect.NewResponse(resp), nil
}

func (s *fakeFindingsService) GetFinding(_ context.Context, req *connect.Request[findingsv1.GetFindingRequest]) (*connect.Response[findingsv1.GetFindingResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.store[req.Msg.GetId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("finding %q not found", req.Msg.GetId()))
	}
	return connect.NewResponse(&findingsv1.GetFindingResponse{Finding: f}), nil
}

func (s *fakeFindingsService) AddFinding(_ context.Context, req *connect.Request[findingsv1.AddFindingRequest]) (*connect.Response[findingsv1.AddFindingResponse], error) {
	if strings.TrimSpace(req.Msg.GetClaim()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("claim: required"))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var cites []*findingsv1.Citation
	for i, c := range req.Msg.GetCitations() {
		cites = append(cites, &findingsv1.Citation{Id: fmt.Sprintf("c-%d", i+1), Url: c.GetUrl(), Title: c.GetTitle()})
	}
	id := s.insertLocked(req.Msg.GetClaim(), req.Msg.GetConfidence(), findingsv1.FindingStatus_FINDING_STATUS_ACTIVE, cites)
	s.audits = append(s.audits, "create:"+id+":")
	return connect.NewResponse(&findingsv1.AddFindingResponse{Finding: s.store[id]}), nil
}

func (s *fakeFindingsService) EditFinding(_ context.Context, req *connect.Request[findingsv1.EditFindingRequest]) (*connect.Response[findingsv1.EditFindingResponse], error) {
	if strings.TrimSpace(req.Msg.GetClaim()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("claim: required"))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.store[req.Msg.GetId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("finding %q not found", req.Msg.GetId()))
	}
	f.Claim = req.Msg.GetClaim()
	f.Confidence = req.Msg.GetConfidence()
	s.audits = append(s.audits, "edit:"+f.Id+":")
	return connect.NewResponse(&findingsv1.EditFindingResponse{Finding: f}), nil
}

func (s *fakeFindingsService) SupersedeFinding(_ context.Context, req *connect.Request[findingsv1.SupersedeFindingRequest]) (*connect.Response[findingsv1.SupersedeFindingResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.store[req.Msg.GetId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("finding %q not found", req.Msg.GetId()))
	}
	if rep := req.Msg.GetReplacement(); rep != "" {
		if _, ok := s.store[rep]; !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("replacement finding %q not found", rep))
		}
	}
	f.Status = findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED
	f.SupersededBy = req.Msg.GetReplacement()
	s.audits = append(s.audits, "supersede:"+f.Id+":"+req.Msg.GetReason())
	return connect.NewResponse(&findingsv1.SupersedeFindingResponse{Finding: f}), nil
}

func (s *fakeFindingsService) FlagFinding(_ context.Context, req *connect.Request[findingsv1.FlagFindingRequest]) (*connect.Response[findingsv1.FlagFindingResponse], error) {
	if strings.TrimSpace(req.Msg.GetReason()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("reason: required"))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.store[req.Msg.GetId()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("finding %q not found", req.Msg.GetId()))
	}
	f.Status = findingsv1.FindingStatus_FINDING_STATUS_DISPUTED
	f.DisputeNote = req.Msg.GetReason()
	s.audits = append(s.audits, "flag:"+f.Id+":"+req.Msg.GetReason())
	return connect.NewResponse(&findingsv1.FlagFindingResponse{Finding: f}), nil
}

func (s *fakeFindingsService) PruneFindings(_ context.Context, req *connect.Request[findingsv1.PruneFindingsRequest]) (*connect.Response[findingsv1.PruneFindingsResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneReqs = append(s.pruneReqs, req.Msg)
	var ids []string
	for _, id := range s.order {
		if f, ok := s.store[id]; ok && f.Status == findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED {
			ids = append(ids, id)
		}
	}
	if !req.Msg.GetDryRun() {
		for _, id := range ids {
			delete(s.store, id)
		}
	}
	return connect.NewResponse(&findingsv1.PruneFindingsResponse{Pruned: int32(len(ids)), FindingIds: ids}), nil
}

// findingsHarness wires the real manifest-built command group to a fake
// FindingsService over a real httptest Connect transport.
type findingsHarness struct {
	fake *fakeFindingsService
	core *cliapp.ScenarioApp
	cmds map[string]cliapp.Command
}

func newFindingsHarness(t *testing.T) *findingsHarness {
	t.Helper()
	fake := newFakeFindingsService()
	path, handler := findingsconnect.NewFindingsServiceHandler(fake)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	core := clitest.NewTestApp(t, mux)
	group, err := Register(core, readFindingsManifest(t))
	require.NoError(t, err, "Register must build the findings group from cli/manifest.json")
	cmds := make(map[string]cliapp.Command, len(group.Subcommands))
	for _, c := range group.Subcommands {
		cmds[c.Name] = c
	}
	return &findingsHarness{fake: fake, core: core, cmds: cmds}
}

// run dispatches a named findings subcommand through its manifest ArgSchema
// and bound handler, capturing stdout. opts carries flag/positional values.
func (h *findingsHarness) run(t *testing.T, name string, opts cliapptest.TestRunContextOptions) (string, error) {
	t.Helper()
	cmd, ok := h.cmds[name]
	require.True(t, ok, "manifest must declare findings %s", name)
	require.NotNil(t, cmd.RunCtx, "findings %s must have a bound handler", name)
	ctx, out := cliapptest.NewCapturedRunContext(h.core, cmd.Args, opts)
	err := cmd.RunCtx(ctx)
	return out.String(), err
}

// TestListStatusActiveFiltersToActiveOnly: `findings list --status=active`
// must send an ACTIVE status filter and render only active findings.
func TestListStatusActiveFiltersToActiveOnly(t *testing.T) {
	h := newFindingsHarness(t)
	h.fake.seed("active claim", findingsv1.FindingStatus_FINDING_STATUS_ACTIVE)
	h.fake.seed("disputed claim", findingsv1.FindingStatus_FINDING_STATUS_DISPUTED)
	h.fake.seed("superseded claim", findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED)

	out, err := h.run(t, "list", cliapptest.TestRunContextOptions{
		Flags: map[string]string{"status": "active"},
	})
	require.NoError(t, err)

	require.Len(t, h.fake.listReqs, 1)
	require.Equal(t, findingsv1.FindingStatus_FINDING_STATUS_ACTIVE, h.fake.listReqs[0].GetStatus())
	require.Contains(t, out, "Found 1 finding(s).")
	require.Contains(t, out, "active claim")
	require.NotContains(t, out, "disputed claim")
	require.NotContains(t, out, "superseded claim")
}

// TestListStatusSupersededFiltersToSupersededOnly: `findings list
// --status=superseded` must return only superseded findings (the explicit
// status filter wins over the default exclude-archived behavior).
func TestListStatusSupersededFiltersToSupersededOnly(t *testing.T) {
	h := newFindingsHarness(t)
	h.fake.seed("active claim", findingsv1.FindingStatus_FINDING_STATUS_ACTIVE)
	h.fake.seed("superseded claim", findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED)

	out, err := h.run(t, "list", cliapptest.TestRunContextOptions{
		Flags: map[string]string{"status": "superseded"},
	})
	require.NoError(t, err)

	require.Len(t, h.fake.listReqs, 1)
	require.Equal(t, findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED, h.fake.listReqs[0].GetStatus())
	require.Contains(t, out, "Found 1 finding(s).")
	require.Contains(t, out, "superseded claim")
	require.NotContains(t, out, "active claim")
}

// TestAddWithClaimAndCitationCreatesActiveFinding: `findings add` with a
// valid claim and one citation creates a finding with status=active.
func TestAddWithClaimAndCitationCreatesActiveFinding(t *testing.T) {
	h := newFindingsHarness(t)

	out, err := h.run(t, "add", cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"claim":      "Go 1.25 is the toolchain floor",
			"confidence": "0.9",
			"citations":  "https://go.dev/doc|Go docs",
		},
	})
	require.NoError(t, err)
	require.Contains(t, out, "Added finding f-1")
	require.Contains(t, out, "status=FINDING_STATUS_ACTIVE")
	require.Contains(t, out, "citations=1")

	created := h.fake.get("f-1")
	require.NotNil(t, created)
	require.Equal(t, findingsv1.FindingStatus_FINDING_STATUS_ACTIVE, created.Status)
	require.Equal(t, "Go 1.25 is the toolchain floor", created.Claim)
	require.InEpsilon(t, 0.9, created.Confidence, 1e-9)
	require.Len(t, created.Citations, 1)
	require.Equal(t, "https://go.dev/doc", created.Citations[0].Url)
	require.Equal(t, "Go docs", created.Citations[0].Title)
}

// TestAddMissingClaimReturnsValidationError covers both validation layers a
// missing claim can hit: the manifest parser (claim is Required:true, so the
// dispatcher rejects argv without --claim before any RPC fires) and the
// server (an empty claim value is rejected with a descriptive
// InvalidArgument the CLI must surface, mirroring
// api/handlers/findings/connect_handler_test.go::TestAddFindingValidatesClaim).
func TestAddMissingClaimReturnsValidationError(t *testing.T) {
	h := newFindingsHarness(t)
	addCmd, ok := h.cmds["add"]
	require.True(t, ok)

	t.Run("parser rejects missing required --claim", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		_, err := cliapp.NewTestRunContextFromArgs(addCmd.Args, nil, h.core, &stdout, &stderr)
		require.Error(t, err, "argv without --claim must not reach the handler")
		require.Contains(t, strings.ToLower(err.Error()), "claim", "error must name the missing field")
		require.Equal(t, 0, h.fake.count(), "no finding may be created")
	})

	t.Run("server validation error is surfaced descriptively", func(t *testing.T) {
		_, err := h.run(t, "add", cliapptest.TestRunContextOptions{
			Flags: map[string]string{"claim": "   "},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "claim", "error must carry the server's field-level message")
		require.Equal(t, 0, h.fake.count(), "no finding may be created")
	})
}

// TestPruneDryRunReportsCountWithoutMutating: `findings prune --dry-run`
// must send dry_run=true, report the matched count, and leave the store
// untouched.
func TestPruneDryRunReportsCountWithoutMutating(t *testing.T) {
	h := newFindingsHarness(t)
	h.fake.seed("keep me", findingsv1.FindingStatus_FINDING_STATUS_ACTIVE)
	gone := h.fake.seed("prune me", findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED)

	out, err := h.run(t, "prune", cliapptest.TestRunContextOptions{
		BoolFlags: map[string]bool{"dry-run": true},
	})
	require.NoError(t, err)

	require.Len(t, h.fake.pruneReqs, 1)
	require.True(t, h.fake.pruneReqs[0].GetDryRun(), "CLI must forward --dry-run")
	require.Contains(t, out, "Would prune 1 superseded finding(s).")
	require.Contains(t, out, gone)
	require.Equal(t, 2, h.fake.count(), "dry-run must not delete anything")
	require.NotNil(t, h.fake.get(gone), "the matched finding must survive a dry-run")
}

// TestListMixedStatusStoreFiltersCorrectly is the integration-shaped pass:
// against one store seeded with all three lifecycle states, the CLI's
// default / --include-archived / --status views must each return the
// correctly filtered slice end-to-end (manifest schema → handler → Connect
// transport → service → rendered output). The same filter semantics against
// the real SQLite store are pinned by
// api/internal/findings/sqlite_test.go::TestListExcludesSupersededByDefault.
func TestListMixedStatusStoreFiltersCorrectly(t *testing.T) {
	h := newFindingsHarness(t)
	h.fake.seed("active claim", findingsv1.FindingStatus_FINDING_STATUS_ACTIVE)
	h.fake.seed("disputed claim", findingsv1.FindingStatus_FINDING_STATUS_DISPUTED)
	h.fake.seed("superseded claim", findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED)

	t.Run("default excludes superseded only", func(t *testing.T) {
		out, err := h.run(t, "list", cliapptest.TestRunContextOptions{})
		require.NoError(t, err)
		require.Contains(t, out, "Found 2 finding(s).")
		require.Contains(t, out, "active claim")
		require.Contains(t, out, "disputed claim")
		require.NotContains(t, out, "superseded claim")
	})

	t.Run("include-archived returns all three", func(t *testing.T) {
		out, err := h.run(t, "list", cliapptest.TestRunContextOptions{
			BoolFlags: map[string]bool{"include-archived": true},
		})
		require.NoError(t, err)
		require.Contains(t, out, "Found 3 finding(s).")
		require.Contains(t, out, "superseded claim")
	})

	t.Run("status=disputed returns disputed only", func(t *testing.T) {
		out, err := h.run(t, "list", cliapptest.TestRunContextOptions{
			Flags: map[string]string{"status": "disputed"},
		})
		require.NoError(t, err)
		require.Contains(t, out, "Found 1 finding(s).")
		require.Contains(t, out, "disputed claim")
		require.NotContains(t, out, "active claim")
	})
}

// TestOperatorManagesFindingsLifecycleEndToEnd is the business-phase pass: a
// human operator drives the full lifecycle — add, review (list), correct
// (edit), flag, supersede, prune — exclusively through CLI commands, never
// touching the database directly. Every step asserts both the rendered
// output and the resulting store state.
func TestOperatorManagesFindingsLifecycleEndToEnd(t *testing.T) {
	h := newFindingsHarness(t)

	// Add two findings: a first draft and (later) its replacement.
	out, err := h.run(t, "add", cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"claim":      "GPT-5 ships in 2025",
			"confidence": "0.6",
			"citations":  "https://example.com/a|Rumor",
		},
	})
	require.NoError(t, err)
	require.Contains(t, out, "Added finding f-1")

	out, err = h.run(t, "add", cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"claim":      "GPT-5 shipped in August 2025",
			"confidence": "0.95",
			"citations":  "https://example.com/b|Launch post",
		},
	})
	require.NoError(t, err)
	require.Contains(t, out, "Added finding f-2")

	// Review: list shows both as active.
	out, err = h.run(t, "list", cliapptest.TestRunContextOptions{})
	require.NoError(t, err)
	require.Contains(t, out, "Found 2 finding(s).")

	// Correct: edit the first claim's text and confidence.
	out, err = h.run(t, "edit", cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "f-1"},
		Flags:       map[string]string{"claim": "GPT-5 expected in 2025", "confidence": "0.5"},
	})
	require.NoError(t, err)
	require.Contains(t, out, "Edited finding f-1")
	require.Equal(t, "GPT-5 expected in 2025", h.fake.get("f-1").Claim)

	// Flag: dispute the first finding with a reason.
	out, err = h.run(t, "flag", cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "f-1"},
		Flags:       map[string]string{"reason": "sources conflict on the date"},
	})
	require.NoError(t, err)
	require.Contains(t, out, "Flagged finding f-1 as disputed.")
	require.Equal(t, findingsv1.FindingStatus_FINDING_STATUS_DISPUTED, h.fake.get("f-1").Status)
	require.Equal(t, "sources conflict on the date", h.fake.get("f-1").DisputeNote)

	// Supersede: retire the first finding in favour of the second.
	out, err = h.run(t, "supersede", cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "f-1"},
		Flags:       map[string]string{"replacement": "f-2", "reason": "launch confirmed"},
	})
	require.NoError(t, err)
	require.Contains(t, out, "Superseded finding f-1.")
	require.Equal(t, findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED, h.fake.get("f-1").Status)
	require.Equal(t, "f-2", h.fake.get("f-1").SupersededBy)
	require.Contains(t, h.fake.audits, "supersede:f-1:launch confirmed",
		"the supersede reason must reach the service for its audit row")

	// Prune: the superseded original is removed; the replacement survives.
	// Deletion requires the explicit --force flag (REQ-P0-006).
	out, err = h.run(t, "prune", cliapptest.TestRunContextOptions{
		BoolFlags: map[string]bool{"force": true},
	})
	require.NoError(t, err)
	require.Contains(t, out, "Pruned 1 superseded finding(s).")
	require.Nil(t, h.fake.get("f-1"))
	require.NotNil(t, h.fake.get("f-2"))
}

// TestPruneWithoutForceRefuses: [REQ:REQ-P0-006] `findings prune` without
// --dry-run and without --force must refuse before any RPC is issued — the
// error names --force so the operator knows how to proceed. Interactive
// confirmation prompts are deliberately NOT used: every findings command
// must stay invocable programmatically (see
// TestManagementCommandsRunProgrammaticallyWithoutPrompts), so the
// destructive gate is an explicit flag, not a stdin prompt.
func TestPruneWithoutForceRefuses(t *testing.T) {
	h := newFindingsHarness(t)
	gone := h.fake.seed("prune me", findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED)

	_, err := h.run(t, "prune", cliapptest.TestRunContextOptions{})
	require.Error(t, err, "prune without --force must refuse")
	require.Contains(t, err.Error(), "--force", "the refusal must tell the operator about --force")
	require.Empty(t, h.fake.pruneReqs, "no PruneFindings RPC may be issued on refusal")
	require.NotNil(t, h.fake.get(gone), "nothing may be deleted on refusal")

	// With --force the same invocation executes the deletion.
	out, err := h.run(t, "prune", cliapptest.TestRunContextOptions{
		BoolFlags: map[string]bool{"force": true},
	})
	require.NoError(t, err)
	require.Contains(t, out, "Pruned 1 superseded finding(s).")
	require.Nil(t, h.fake.get(gone), "--force must execute the prune")
}

// TestManagementCommandsRunProgrammaticallyWithoutPrompts is the agent-side
// business pass: every findings management command (list, add, edit,
// supersede, flag, prune) must complete from a single non-interactive
// invocation — flags and positionals only, no stdin, no confirmation step —
// which is exactly how the L3 agent invokes them as tools (the agent prompt
// in api/internal/research/l3_agent.go issues these CLI commands verbatim).
func TestManagementCommandsRunProgrammaticallyWithoutPrompts(t *testing.T) {
	h := newFindingsHarness(t)
	replacement := h.fake.seed("replacement claim", findingsv1.FindingStatus_FINDING_STATUS_ACTIVE)
	target := h.fake.seed("target claim", findingsv1.FindingStatus_FINDING_STATUS_ACTIVE)

	steps := []struct {
		command string
		opts    cliapptest.TestRunContextOptions
	}{
		{"add", cliapptest.TestRunContextOptions{Flags: map[string]string{
			"claim": "agent-added claim", "confidence": "0.8", "source": "l3",
			"citations": "https://example.com|Source",
		}}},
		{"list", cliapptest.TestRunContextOptions{Flags: map[string]string{"status": "active"}}},
		{"edit", cliapptest.TestRunContextOptions{
			Positionals: map[string]string{"id": target},
			Flags:       map[string]string{"claim": "target claim, refined"},
		}},
		{"flag", cliapptest.TestRunContextOptions{
			Positionals: map[string]string{"id": target},
			Flags:       map[string]string{"reason": "contradicted by new source"},
		}},
		{"supersede", cliapptest.TestRunContextOptions{
			Positionals: map[string]string{"id": target},
			Flags:       map[string]string{"replacement": replacement, "reason": "agent reconcile"},
		}},
		{"prune", cliapptest.TestRunContextOptions{BoolFlags: map[string]bool{"dry-run": true}}},
	}
	for _, step := range steps {
		out, err := h.run(t, step.command, step.opts)
		require.NoError(t, err, "findings %s must succeed without human intervention", step.command)
		require.NotEmpty(t, out, "findings %s must render machine-readable output", step.command)
	}

	// The full tool sequence really happened: the target was edited, flagged,
	// then superseded by the replacement.
	require.Equal(t, findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED, h.fake.get(target).Status)
	require.Equal(t, replacement, h.fake.get(target).SupersededBy)
}
