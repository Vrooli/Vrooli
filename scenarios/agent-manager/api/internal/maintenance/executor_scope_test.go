package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

func TestControlPlaneExecutorScopeResolvesMissingAttachedPIDFromExactOwnerProof(t *testing.T) {
	attached := &domain.Run{
		ID:            uuid.MustParse("01c4bd4b-d27e-477a-a21c-16cc33d295cd"),
		ExecutionMode: domain.ExecutionModeAttached,
		HarnessKind:   "claude-code", HarnessSessionID: "claude-code-1ef8f018413dd54cd1be4a9445c98f17",
		WebConsoleSessionID: "0093bdf6-bedc-4c35-a61c-4019aa1e29a9",
		Tag:                 "attached-claude-code-01c4bd4b",
	}
	ordinary := &domain.Run{ID: uuid.New(), RunnerPID: 812, RunnerPGID: 812, Tag: "ordinary-run"}
	refs := []ExecutorRef{
		{WorkRef: WorkRef{ID: attached.ID.String(), Kind: "run", Status: "running"}},
		{WorkRef: WorkRef{ID: ordinary.ID.String(), Kind: "run", Status: "running"}, PID: 812, PGID: 812},
	}
	request := attachedBindingRequest{RunID: attached.ID.String(), HarnessSessionID: attached.HarnessSessionID, WebConsoleSessionID: attached.WebConsoleSessionID, HarnessKind: attached.HarnessKind}
	proof := cliutil.AgentSessionBindingProof{
		ProofVersion: cliutil.AgentSessionBindingProofVersion,
		RunID:        request.RunID, HarnessSessionID: request.HarnessSessionID,
		LeaseSessionID: request.HarnessSessionID, WebConsoleSessionID: request.WebConsoleSessionID,
		HarnessKind: request.HarnessKind, OwnerKind: "web-console-session",
		Binding: cliutil.ProcessBinding{PID: 913, UID: 1000, Start: "12345", Boot: "faad40a2-2026-4c3b-8b36-efc2d641ced6", Cgroup: "0::/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/tmux-spawn-test.scope"},
	}
	proofReads, scopeReads := 0, 0
	owner := &ControlPlaneObserver{
		lookupBatch: func(context.Context, []ExecutorRef) ([]*domain.Run, error) {
			return []*domain.Run{attached, ordinary}, nil
		},
		readBinding: func(_ context.Context, got attachedBindingRequest) (cliutil.AgentSessionBindingProof, error) {
			proofReads++
			if got != request {
				t.Fatalf("identity proof request=%+v want %+v", got, request)
			}
			return proof, nil
		},
		read: func(_ context.Context, input []byte) ([]byte, error) {
			scopeReads++
			var req struct {
				Executors []executorScopeRef `json:"executors"`
			}
			if err := json.Unmarshal(input, &req); err != nil || len(req.Executors) != 2 {
				t.Fatalf("invalid scope request: %s err=%v", input, err)
			}
			byRun := map[string]executorScopeRef{}
			for _, ref := range req.Executors {
				byRun[ref.RunID] = ref
			}
			attachedRef := byRun[attached.ID.String()]
			if attachedRef.PID != 0 || attachedRef.PGID != 0 || attachedRef.BindingProof == nil || attachedRef.BindingProof.Binding != proof.Binding || attachedRef.HarnessSessionID != request.HarnessSessionID || attachedRef.HarnessKind != request.HarnessKind {
				t.Errorf("attached owner proof was not forwarded exactly: %+v", attachedRef)
			}
			ordinaryRef := byRun[ordinary.ID.String()]
			if ordinaryRef.PID != ordinary.RunnerPID || ordinaryRef.PGID != ordinary.RunnerPGID || ordinaryRef.HarnessSessionID != "" || ordinaryRef.HarnessKind != "" {
				t.Errorf("ordinary durable identity changed: %+v", ordinaryRef)
			}
			return []byte(fmt.Sprintf(`{"schemaVersion":"executor-scope-v1","identityVersion":"attached-key-v1","complete":true,"executors":[{"runId":%q,"state":"present","pids":[%d],"reasons":[]},{"runId":%q,"state":"absent","pids":[],"reasons":[]}]}`, attached.ID, proof.Binding.PID, ordinary.ID)), nil
		},
	}
	evidence, err := owner.Observe(t.Context(), refs)
	if err != nil || proofReads != 1 || scopeReads != 1 || len(evidence) != 2 {
		t.Fatalf("owner proof evidence=%+v proofReads=%d scopeReads=%d err=%v", evidence, proofReads, scopeReads, err)
	}
	if !evidence[0].Alive || evidence[0].HasChildren || len(evidence[0].PIDs) != 1 || evidence[0].PIDs[0] != proof.Binding.PID {
		t.Fatalf("attached live root evidence=%+v", evidence[0])
	}
	if evidence[0].PID != 0 || evidence[0].PGID != 0 || evidence[1].Alive {
		t.Fatalf("ephemeral owner proof mutated durable row projection or ordinary absence: %+v", evidence)
	}
}

func TestControlPlaneExecutorScopeKeepsMissingAttachedPIDUnknownWithoutExactProof(t *testing.T) {
	attached := &domain.Run{
		ID: uuid.New(), ExecutionMode: domain.ExecutionModeAttached,
		HarnessKind: "claude-code", HarnessSessionID: "exact-harness-session",
		WebConsoleSessionID: "exact-web-console-session", Tag: "attached-run",
	}
	ref := ExecutorRef{WorkRef: WorkRef{ID: attached.ID.String(), Kind: "run", Status: "running"}}
	exact := attachedBindingRequest{RunID: attached.ID.String(), HarnessSessionID: attached.HarnessSessionID, WebConsoleSessionID: attached.WebConsoleSessionID, HarnessKind: attached.HarnessKind}
	for _, tc := range []struct {
		name       string
		reader     func(context.Context, attachedBindingRequest) (cliutil.AgentSessionBindingProof, error)
		wantReason string
	}{
		{name: "mismatch", reader: func(context.Context, attachedBindingRequest) (cliutil.AgentSessionBindingProof, error) {
			return cliutil.AgentSessionBindingProof{ProofVersion: cliutil.AgentSessionBindingProofVersion, RunID: "different-run"}, nil
		}, wantReason: "mismatched or incomplete"},
		{name: "unavailable", reader: func(context.Context, attachedBindingRequest) (cliutil.AgentSessionBindingProof, error) {
			return cliutil.AgentSessionBindingProof{}, errors.New("owner unavailable")
		}, wantReason: "proof unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner := &ControlPlaneObserver{
				lookupBatch: func(context.Context, []ExecutorRef) ([]*domain.Run, error) { return []*domain.Run{attached}, nil },
				readBinding: tc.reader,
				read: func(_ context.Context, input []byte) ([]byte, error) {
					var req struct {
						Executors []executorScopeRef `json:"executors"`
					}
					if err := json.Unmarshal(input, &req); err != nil || len(req.Executors) != 1 || req.Executors[0].PID != 0 || req.Executors[0].HarnessSessionID != "" {
						t.Fatalf("unproven attached identity leaked into scope request: %s err=%v", input, err)
					}
					return []byte(fmt.Sprintf(`{"schemaVersion":"executor-scope-v1","identityVersion":"attached-key-v1","complete":true,"executors":[{"runId":%q,"state":"absent","pids":[],"reasons":[]}]}`, attached.ID)), nil
				},
			}
			evidence, err := owner.Observe(t.Context(), []ExecutorRef{ref})
			if err == nil || !strings.Contains(err.Error(), tc.wantReason) || !strings.Contains(err.Error(), exact.RunID) || len(evidence) != 1 || evidence[0].Alive {
				t.Fatalf("unproven attached run became complete: evidence=%+v err=%v", evidence, err)
			}
		})
	}
}

func TestControlPlaneExecutorScopeUsesDurableIdentityAndCompleteProof(t *testing.T) {
	id := uuid.New()
	start, end := time.Now().Add(-time.Hour), time.Now().Add(-time.Minute)
	run := &domain.Run{ID: id, Tag: "retained-run-tag", RunnerPID: 315576, RunnerPGID: 315576, StartedAt: &start, EndedAt: &end}
	ref := ExecutorRef{WorkRef: WorkRef{ID: id.String(), Kind: "run", Status: "failed"}, PID: 315576, PGID: 315576}
	for _, tc := range []struct {
		name, report   string
		alive, unknown bool
	}{
		{"absent", `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"absent","pids":[],"reasons":[]}]}`, false, false},
		{"present-descendant", `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"present","pids":[567605]}]}`, true, false},
		{"unknown", `{"schemaVersion":"executor-scope-v1","complete":false,"executors":[{"runId":"%s","state":"unknown"}]}`, false, true},
		{"incomplete-absence", `{"schemaVersion":"executor-scope-v1","complete":false,"executors":[{"runId":"%s","state":"absent"}]}`, false, true},
		{"missing", `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[]}`, false, true},
		{"wrong-version", `{"schemaVersion":"old","complete":true,"executors":[{"runId":"%s","state":"absent"}]}`, false, true},
		{"different-run", `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"other","state":"absent"}]}`, false, true},
		{"duplicate", `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"absent"},{"runId":"%s","state":"absent"}]}`, false, true},
		{"contradictory", `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"absent","pids":[315576]}]}`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := &ControlPlaneObserver{lookup: func(context.Context, uuid.UUID) (*domain.Run, error) { return run, nil }, read: func(ctx context.Context, data []byte) ([]byte, error) {
				var req struct {
					Executors []executorScopeRef `json:"executors"`
				}
				if err := json.Unmarshal(data, &req); err != nil || len(req.Executors) != 1 {
					t.Fatalf("request=%s %v", data, err)
				}
				got := req.Executors[0]
				if got.RunID != id.String() || got.Tag != run.Tag || got.LegacyTag != "opencode-continue-"+id.String()[:8] || got.StartedAt == nil || !got.StartedAt.Equal(start) || got.EndedAt == nil || !got.EndedAt.Equal(end) || got.PID != 315576 {
					t.Fatalf("durable identity lost: %+v", got)
				}
				return []byte(strings.ReplaceAll(tc.report, "%s", id.String())), nil
			}}
			out, err := o.Observe(t.Context(), []ExecutorRef{ref})
			if (err != nil) != tc.unknown {
				t.Fatalf("proof=%+v err=%v", out, err)
			}
			if !tc.unknown && (len(out) != 1 || out[0].Alive != tc.alive || out[0].HasChildren != tc.alive) {
				t.Fatalf("physical evidence lost: %+v", out)
			}
			if tc.alive && (len(out[0].PIDs) != 1 || out[0].PIDs[0] != 567605) {
				t.Fatal("matched descendant PID evidence lost")
			}
		})
	}
}

func TestControlPlaneExecutorScopeRetainsOnlyAttachedHarnessAnchors(t *testing.T) {
	for _, tc := range []struct {
		name            string
		mode            domain.ExecutionMode
		kind, sessionID string
		want            map[string]string
	}{
		{name: "attached canonical session differs from run and mutable session", mode: domain.ExecutionModeAttached, kind: "codex", sessionID: "native-session", want: map[string]string{"harnessKind": "codex", "harnessSessionId": "native-session"}},
		{name: "attached legacy session does not send partial identity", mode: domain.ExecutionModeAttached, kind: "claude-code"},
		{name: "attached missing kind does not send partial identity", mode: domain.ExecutionModeAttached, sessionID: "native-session"},
		{name: "attached legacy identity has no anchors", mode: domain.ExecutionModeAttached},
		{name: "attached malformed session does not reach owner", mode: domain.ExecutionModeAttached, kind: "codex", sessionID: " "},
		{name: "attached malformed kind does not reach owner", mode: domain.ExecutionModeAttached, kind: " ", sessionID: "native-session"},
		{name: "managed codec metadata is not an attached claim", mode: domain.ExecutionModeCodecPipe, kind: "codex", sessionID: "native-session"},
		{name: "managed interactive metadata is not an attached claim", mode: domain.ExecutionModeInteractive, kind: "claude-code", sessionID: "native-session"},
		{name: "imported metadata is not an attached claim", mode: domain.ExecutionModeImported, kind: "codex", sessionID: "native-session"},
		{name: "default managed metadata is not an attached claim", kind: "codex", sessionID: "native-session"},
		{name: "unknown execution mode is not an attached claim", mode: "unknown", kind: "codex", sessionID: "native-session"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given durable run and mutable transcript identities that are not the
			// native lease key, only an attached run may assert that separate key.
			run := &domain.Run{
				ID: uuid.MustParse("dff7501c-2a17-4fe0-84c0-0bec7f24c397"), RunnerPID: 41, RunnerPGID: 41,
				ExecutionMode: tc.mode, HarnessKind: tc.kind, HarnessSessionID: tc.sessionID, SessionID: "mutable-transcript-session",
			}
			ref := ExecutorRef{WorkRef: WorkRef{ID: run.ID.String()}, PID: 41, PGID: 41}
			reads := 0
			owner := &ControlPlaneObserver{
				lookupBatch: func(context.Context, []ExecutorRef) ([]*domain.Run, error) { return []*domain.Run{run}, nil },
				read: func(_ context.Context, input []byte) ([]byte, error) {
					reads++
					var request struct {
						Executors []map[string]json.RawMessage `json:"executors"`
					}
					if err := json.Unmarshal(input, &request); err != nil || len(request.Executors) != 1 {
						t.Fatalf("invalid scope request: %s, %v", input, err)
					}
					// Then the wire retains partial/malformed anchors verbatim for
					// canonical validation, and omits non-attached or absent anchors.
					for _, field := range []string{"harnessKind", "harnessSessionId"} {
						value, present := request.Executors[0][field]
						want, wanted := tc.want[field]
						var got string
						if present != wanted || (present && (json.Unmarshal(value, &got) != nil || got != want)) {
							t.Errorf("%s=%s present=%t, want %q present=%t", field, value, present, want, wanted)
						}
					}
					return []byte(fmt.Sprintf(`{"schemaVersion":"executor-scope-v1","identityVersion":"attached-key-v1","complete":false,"executors":[{"runId":%q,"state":"unknown","reasons":["owner validation pending"]}]}`, run.ID)), nil
				},
			}
			// When the observer sends one bounded read, an unknown owner result
			// must remain unknown even if the attached identity is incomplete.
			evidence, err := owner.Observe(t.Context(), []ExecutorRef{ref})
			wantReason := "exclusion is unknown"
			if tc.mode == domain.ExecutionModeAttached && (tc.kind == "" || tc.sessionID == "" || strings.TrimSpace(tc.kind) != tc.kind || strings.TrimSpace(tc.sessionID) != tc.sessionID) {
				wantReason = "durable harness identity is incomplete"
			}
			if reads != 1 || len(evidence) != 1 || err == nil || !strings.Contains(err.Error(), wantReason) {
				t.Fatalf("scope read=%d evidence=%+v err=%v", reads, evidence, err)
			}
		})
	}
}

func TestControlPlaneExecutorScopeRequiresAcknowledgedAttachedIdentity(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		mode                     domain.ExecutionMode
		kind, sessionID, version string
		accepted                 bool
	}{
		{name: "old owner absence remains exact for ordinary ref but not attached ref", mode: domain.ExecutionModeAttached, kind: "codex", sessionID: "native-session"},
		{name: "attached key requires the exact identity version", mode: domain.ExecutionModeAttached, kind: "codex", sessionID: "native-session", version: "attached-key-v2"},
		{name: "attached key is understood by a current owner", mode: domain.ExecutionModeAttached, kind: "codex", sessionID: "native-session", version: "attached-key-v1", accepted: true},
		{name: "kind-only anchor also requires acknowledgement", mode: domain.ExecutionModeAttached, kind: "codex"},
		{name: "session-only anchor also requires acknowledgement", mode: domain.ExecutionModeAttached, sessionID: "native-session"},
		{name: "malformed nonempty anchor cannot bypass acknowledgement", mode: domain.ExecutionModeAttached, sessionID: " "},
		{name: "unanchored legacy attached request stays unknown", mode: domain.ExecutionModeAttached},
		{name: "managed harness metadata is not an attached request", mode: domain.ExecutionModeCodecPipe, kind: "codex", sessionID: "native-session", accepted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given an attached scope followed by an unanchored managed scope,
			// the owner must acknowledge every identity field the request uses.
			runs := []*domain.Run{
				{ID: uuid.MustParse("dff7501c-2a17-4fe0-84c0-0bec7f24c397"), RunnerPID: 41, RunnerPGID: 41, ExecutionMode: tc.mode, HarnessKind: tc.kind, HarnessSessionID: tc.sessionID},
				{ID: uuid.MustParse("dff7501c-2a17-4fe0-84c0-0bec7f24c398"), RunnerPID: 42, RunnerPGID: 42},
			}
			refs := []ExecutorRef{
				{WorkRef: WorkRef{ID: runs[0].ID.String()}, PID: 41, PGID: 41},
				{WorkRef: WorkRef{ID: runs[1].ID.String()}, PID: 42, PGID: 42},
			}
			reads := 0
			owner := &ControlPlaneObserver{
				lookupBatch: func(context.Context, []ExecutorRef) ([]*domain.Run, error) { return runs, nil },
				read: func(context.Context, []byte) ([]byte, error) {
					reads++
					versionField := ""
					if tc.version != "" {
						versionField = fmt.Sprintf(`,"identityVersion":%q`, tc.version)
					}
					return []byte(fmt.Sprintf(`{"schemaVersion":"executor-scope-v1"%s,"complete":true,"executors":[{"runId":%q,"state":"absent"},{"runId":%q,"state":"absent"}]}`, versionField, runs[0].ID, runs[1].ID)), nil
				},
			}
			// When complete absence is reported, then silently ignored anchors
			// cannot become an exclusion proof. Legacy-only reads remain valid.
			evidence, err := owner.Observe(t.Context(), refs)
			if reads != 1 {
				t.Fatalf("owner reads=%d, want one", reads)
			}
			if tc.accepted {
				if err != nil || len(evidence) != 2 || evidence[0].ID != refs[0].ID || evidence[1].ID != refs[1].ID || evidence[0].Alive || evidence[1].Alive {
					t.Fatalf("acknowledged absence lost: evidence=%+v err=%v", evidence, err)
				}
			} else if err == nil || (!strings.Contains(err.Error(), "attached-key-v1") && !strings.Contains(err.Error(), "durable harness identity is incomplete")) || !strings.Contains(err.Error(), runs[0].ID.String()) || strings.Contains(err.Error(), runs[1].ID.String()) || len(evidence) != 2 || evidence[0].ID != refs[0].ID || evidence[1].ID != refs[1].ID {
				t.Fatalf("old owner did not preserve exact ordinary coverage and isolate attached unknown: evidence=%+v err=%v", evidence, err)
			}
		})
	}
}

func TestControlPlaneExecutorScopeKeepsOrdinaryCoverageFromOldIncompleteOwner(t *testing.T) {
	ordinary := &domain.Run{ID: uuid.New(), RunnerPID: 41, RunnerPGID: 41}
	attached := &domain.Run{ID: uuid.New(), RunnerPID: 42, RunnerPGID: 42, ExecutionMode: domain.ExecutionModeAttached, HarnessKind: "claude-code", HarnessSessionID: "native-session"}
	refs := []ExecutorRef{
		{WorkRef: WorkRef{ID: ordinary.ID.String(), Kind: "run", Status: "failed"}, PID: 41, PGID: 41},
		{WorkRef: WorkRef{ID: attached.ID.String(), Kind: "run", Status: "running"}, PID: 42, PGID: 42},
	}
	reads := 0
	owner := &ControlPlaneObserver{
		lookupBatch: func(context.Context, []ExecutorRef) ([]*domain.Run, error) {
			return []*domain.Run{ordinary, attached}, nil
		},
		read: func(context.Context, []byte) ([]byte, error) {
			reads++
			return []byte(fmt.Sprintf(`{"schemaVersion":"executor-scope-v1","complete":false,"executors":[{"runId":%q,"state":"absent","pids":[],"reasons":[]},{"runId":%q,"state":"unknown","pids":[],"reasons":["legacy owner cannot prove attached identity"]}]}`, ordinary.ID, attached.ID)), errors.New("exit status 1")
		},
	}
	evidence, err := owner.Observe(t.Context(), refs)
	if reads != 1 || err == nil || !strings.Contains(err.Error(), attached.ID.String()) || strings.Contains(err.Error(), ordinary.ID.String()) || len(evidence) != 2 || evidence[0].ID != ordinary.ID.String() || evidence[1].ID != attached.ID.String() || evidence[0].Alive || evidence[1].Alive {
		t.Fatalf("old owner's structured partial proof was discarded or misattributed: reads=%d evidence=%+v err=%v", reads, evidence, err)
	}
}

func TestControlPlaneExecutorScopeRefusesUnstableOrUnavailableEvidence(t *testing.T) {
	id := uuid.New()
	ref := ExecutorRef{WorkRef: WorkRef{ID: id.String()}, PID: 3}
	for _, fail := range []string{"identity-change", "unavailable", "oversized", "cancelled"} {
		t.Run(fail, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			o := &ControlPlaneObserver{lookup: func(context.Context, uuid.UUID) (*domain.Run, error) {
				pid := 3
				if fail == "identity-change" {
					pid = 4
				}
				return &domain.Run{ID: id, RunnerPID: pid}, nil
			}, read: func(context.Context, []byte) ([]byte, error) {
				if fail == "unavailable" {
					return nil, errors.New("command unavailable")
				}
				if fail == "oversized" {
					return make([]byte, executorScopeMaxBytes+1), nil
				}
				if fail == "cancelled" {
					cancel()
				}
				return []byte(fmt.Sprintf(`{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":%q,"state":"absent"}]}`, id)), nil
			}}
			if _, err := o.Observe(ctx, []ExecutorRef{ref}); err == nil {
				t.Fatal("unknown became absent")
			}
		})
	}
	var out boundedScopeOutput
	if _, err := out.Write(make([]byte, executorScopeMaxBytes+1)); err == nil || out.Len() != 0 {
		t.Fatal("unbounded process output retained")
	}
}

func TestReadExecutorScopeIncludesBoundedStructuredCLIError(t *testing.T) {
	binDir := t.TempDir()
	command := filepath.Join(binDir, "vrooli")
	script := "#!/bin/sh\nprintf '%s\\n' '{\"success\":false,\"error\":\"executor-scope requires 1..65536 executors\",\"code\":\"untyped_cli_error\"}' >&2\nexit 1\n"
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	_, err := readExecutorScope(t.Context(), []byte(`{"executors":[]}`))
	if err == nil || !errors.Is(err, errExecutorScopeReferenceLimit) || !strings.Contains(err.Error(), "untyped_cli_error") || !strings.Contains(err.Error(), "executor-scope requires 1..65536 executors") {
		t.Fatalf("structured root CLI rejection was hidden: %v", err)
	}
	var bounded boundedScopeErrorOutput
	input := make([]byte, executorScopeMaxErrorBytes+100)
	n, writeErr := bounded.Write(input)
	if writeErr != nil || n != len(input) || bounded.Len() != executorScopeMaxErrorBytes {
		t.Fatalf("stderr capture was not consumed/bounded: written=%d retained=%d err=%v", n, bounded.Len(), writeErr)
	}
}

func TestExcludeExecutorRequiresCompleteExactAbsence(t *testing.T) {
	id := uuid.New()
	start, end := time.Now().Add(-time.Hour), time.Now().Add(-time.Minute)
	run := &domain.Run{ID: id, Tag: "managed-usi", Status: domain.RunStatusFailed, RunnerPID: 315576, RunnerPGID: 315576, StartedAt: &start, EndedAt: &end}
	for _, tc := range []struct {
		name, report string
		absent       bool
		readErr      error
		cancel       bool
	}{
		{name: "complete-absence", report: `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"absent","pids":[],"reasons":[]}]}`, absent: true},
		{name: "false-failed-live-root", report: `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"present","pids":[315576],"reasons":[]}]}`},
		{name: "live-descendant", report: `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"present","pids":[567605],"reasons":[]}]}`},
		{name: "unknown", report: `{"schemaVersion":"executor-scope-v1","complete":false,"executors":[{"runId":"%s","state":"unknown","reasons":["identity unavailable"]}]}`},
		{name: "incomplete", report: `{"schemaVersion":"executor-scope-v1","complete":false,"executors":[{"runId":"%s","state":"absent"}]}`},
		{name: "omitted", report: `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[]}`},
		{name: "duplicate", report: `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"absent"},{"runId":"%s","state":"absent"}]}`},
		{name: "wrong-run", report: `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"other","state":"absent"}]}`},
		{name: "wrong-version", report: `{"schemaVersion":"old","complete":true,"executors":[{"runId":"%s","state":"absent"}]}`},
		{name: "contradictory", report: `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"absent","reasons":["not excluded"]}]}`},
		{name: "malformed", report: `{`},
		{name: "nonzero-exit-with-absence", report: `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"absent"}]}`, readErr: errors.New("exit status 1")},
		{name: "cancelled-after-read", report: `{"schemaVersion":"executor-scope-v1","complete":true,"executors":[{"runId":"%s","state":"absent"}]}`, cancel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			reads := 0
			err := excludeExecutor(ctx, run, func(_ context.Context, input []byte) ([]byte, error) {
				reads++
				var req struct {
					Executors []executorScopeRef `json:"executors"`
				}
				if err := json.Unmarshal(input, &req); err != nil || len(req.Executors) != 1 {
					t.Fatalf("exclusion did not reuse the exact single-scope request: %s %v", input, err)
				}
				ref := req.Executors[0]
				if ref.RunID != id.String() || ref.Tag != run.GetTag() || ref.LegacyTag != "opencode-continue-"+id.String()[:8] || ref.PID != 315576 || ref.PGID != 315576 || ref.StartedAt == nil || !ref.StartedAt.Equal(start) || ref.EndedAt == nil || !ref.EndedAt.Equal(end) {
					t.Fatalf("exact durable identity lost: %+v", ref)
				}
				if tc.cancel {
					cancel()
				}
				return []byte(strings.ReplaceAll(tc.report, "%s", id.String())), tc.readErr
			})
			if (err == nil) != tc.absent || reads != 1 {
				t.Fatalf("exclusion=%v reads=%d wantAbsent=%v", err, reads, tc.absent)
			}
		})
	}
	if run.Status != domain.RunStatusFailed || run.RunnerPID != 315576 || run.RunnerPGID != 315576 {
		t.Fatal("executor observation changed durable accounting")
	}
}

func TestExcludeExecutorRequiresIdentityAndLiveContext(t *testing.T) {
	for _, run := range []*domain.Run{nil, {}} {
		if err := ExcludeExecutor(t.Context(), run); err == nil {
			t.Fatal("missing durable identity became exclusion")
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := ExcludeExecutor(ctx, &domain.Run{ID: uuid.New()}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled exclusion reached the host reader: %v", err)
	}
}

func TestControlPlaneExecutorScopeBatchesHistoricalReferences(t *testing.T) {
	const count = 65537 // Crosses the canonical per-request capacity.
	runs := make([]*domain.Run, 0, count)
	refs := make([]ExecutorRef, 0, count)
	for i := 0; i < count; i++ {
		run := &domain.Run{ID: uuid.New(), RunnerPID: i + 1, RunnerPGID: i + 1}
		runs = append(runs, run)
		refs = append(refs, ExecutorRef{WorkRef: WorkRef{ID: run.ID.String(), Kind: "run", Status: "failed"}, PID: i + 1, PGID: i + 1})
	}
	lookups, reads := 0, 0
	owner := &ControlPlaneObserver{
		lookupBatch: func(_ context.Context, page []ExecutorRef) ([]*domain.Run, error) {
			lookups++
			byID := make(map[string]*domain.Run, len(runs))
			for _, run := range runs {
				byID[run.ID.String()] = run
			}
			pageRuns := make([]*domain.Run, 0, len(page))
			for _, ref := range page {
				pageRuns = append(pageRuns, byID[ref.ID])
			}
			return pageRuns, nil
		},
		read: func(_ context.Context, input []byte) ([]byte, error) {
			reads++
			var req struct {
				Executors []executorScopeRef `json:"executors"`
			}
			if err := json.Unmarshal(input, &req); err != nil || len(req.Executors) != count || len(input) > executorScopeMaxBytes {
				t.Fatalf("historical identities lost or request exceeded root bytes: refs=%d bytes=%d err=%v", len(req.Executors), len(input), err)
			}
			var out strings.Builder
			out.WriteString(`{"schemaVersion":"executor-scope-v1","complete":true,"executors":[`)
			for i, ref := range req.Executors {
				if i > 0 {
					out.WriteByte(',')
				}
				if ref.RunID == refs[count-1].ID {
					fmt.Fprintf(&out, `{"runId":%q,"state":"present","pids":[315576,567605],"reasons":[]}`, ref.RunID)
				} else {
					fmt.Fprintf(&out, `{"runId":%q,"state":"absent","pids":[],"reasons":[]}`, ref.RunID)
				}
			}
			out.WriteString(`]}`)
			return []byte(out.String()), nil
		},
	}
	evidence, err := owner.Observe(t.Context(), refs)
	if err != nil || len(evidence) != count || reads != 1 || lookups != 1 {
		t.Fatalf("history not covered by one root snapshot request: evidence=%d reads=%d lookups=%d err=%v", len(evidence), reads, lookups, err)
	}
	if last := evidence[count-1]; !last.Alive || len(last.PIDs) != 2 || last.PIDs[0] != 315576 {
		t.Fatalf("managed terminal residue at history tail lost: %+v", last)
	}
}

func TestControlPlaneExecutorScopeOldCountLimitFallbackIsNonAtomic(t *testing.T) {
	const count = 65537
	runs := make(map[string]*domain.Run, count)
	refs := make([]ExecutorRef, 0, count)
	for i := 0; i < count; i++ {
		id := uuid.New()
		run := &domain.Run{ID: id, Tag: "run-" + id.String(), RunnerPID: i + 1, RunnerPGID: i + 1}
		runs[id.String()] = run
		refs = append(refs, ExecutorRef{WorkRef: WorkRef{ID: id.String(), Kind: "run", Status: "failed"}, PID: i + 1, PGID: i + 1})
	}
	reads, lookups := 0, 0
	owner := &ControlPlaneObserver{
		lookupBatch: func(_ context.Context, page []ExecutorRef) ([]*domain.Run, error) {
			lookups++
			out := make([]*domain.Run, 0, len(page))
			for _, ref := range page {
				out = append(out, runs[ref.ID])
			}
			return out, nil
		},
		read: func(_ context.Context, input []byte) ([]byte, error) {
			reads++
			var request struct {
				Executors []executorScopeRef `json:"executors"`
			}
			if err := json.Unmarshal(input, &request); err != nil {
				t.Fatal(err)
			}
			if len(input) > executorScopeMaxBytes {
				t.Fatalf("request exceeded byte limit: %d", len(input))
			}
			if len(request.Executors) > 65536 {
				// The installed legacy owner refuses before taking a host snapshot.
				return nil, fmt.Errorf("%w: old root CLI accepts at most 65536", errExecutorScopeReferenceLimit)
			}
			var report strings.Builder
			report.WriteString(`{"schemaVersion":"executor-scope-v1","complete":true,"executors":[`)
			for i, ref := range request.Executors {
				if i > 0 {
					report.WriteByte(',')
				}
				fmt.Fprintf(&report, `{"runId":%q,"state":"absent","pids":[],"reasons":[]}`, ref.RunID)
			}
			report.WriteString(`]}`)
			return []byte(report.String()), nil
		},
	}
	evidence, err := owner.Observe(t.Context(), refs)
	if err == nil || !strings.Contains(err.Error(), "non-atomic") || !strings.Contains(err.Error(), "old root CLI accepts at most 65536") || len(evidence) != count || reads != 3 || lookups != 3 {
		t.Fatalf("legacy count fallback did not preserve evidence as incomplete: evidence=%d reads=%d lookups=%d err=%v", len(evidence), reads, lookups, err)
	}
}

func TestControlPlaneExecutorScopeSplitsOversizedInventoryByJSONBytesAndMarksNonAtomic(t *testing.T) {
	const count = 4096
	longTag := strings.Repeat("x", 8193)
	runs := make(map[string]*domain.Run, count)
	refs := make([]ExecutorRef, 0, count)
	for i := 0; i < count; i++ {
		id := uuid.New()
		run := &domain.Run{ID: id, Tag: longTag, RunnerPID: i + 1, RunnerPGID: i + 1}
		runs[id.String()] = run
		refs = append(refs, ExecutorRef{WorkRef: WorkRef{ID: id.String(), Kind: "run", Status: "failed"}, PID: i + 1, PGID: i + 1})
	}
	reads := 0
	owner := &ControlPlaneObserver{
		lookupBatch: func(_ context.Context, page []ExecutorRef) ([]*domain.Run, error) {
			out := make([]*domain.Run, 0, len(page))
			for _, ref := range page {
				out = append(out, runs[ref.ID])
			}
			return out, nil
		},
		read: func(_ context.Context, input []byte) ([]byte, error) {
			reads++
			if len(input) > executorScopeMaxBytes {
				t.Fatalf("oversized request reached owner reader: %d bytes", len(input))
			}
			var request struct {
				Executors []executorScopeRef `json:"executors"`
			}
			if err := json.Unmarshal(input, &request); err != nil {
				t.Fatal(err)
			}
			var report strings.Builder
			report.WriteString(`{"schemaVersion":"executor-scope-v1","complete":true,"executors":[`)
			for i, ref := range request.Executors {
				if i > 0 {
					report.WriteByte(',')
				}
				fmt.Fprintf(&report, `{"runId":%q,"state":"absent","pids":[],"reasons":[]}`, ref.RunID)
			}
			report.WriteString(`]}`)
			return []byte(report.String()), nil
		},
	}
	evidence, err := owner.Observe(t.Context(), refs)
	if err == nil || !strings.Contains(err.Error(), "non-atomic") || len(evidence) != count || reads != 2 {
		t.Fatalf("oversized inventory did not report non-atomic bounded reads: evidence=%d reads=%d err=%v", len(evidence), reads, err)
	}
}
