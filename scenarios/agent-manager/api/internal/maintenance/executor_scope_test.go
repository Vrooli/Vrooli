package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

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
		{name: "attached legacy session retains known harness", mode: domain.ExecutionModeAttached, kind: "claude-code", want: map[string]string{"harnessKind": "claude-code"}},
		{name: "attached missing kind retains canonical session", mode: domain.ExecutionModeAttached, sessionID: "native-session", want: map[string]string{"harnessSessionId": "native-session"}},
		{name: "attached legacy identity has no anchors", mode: domain.ExecutionModeAttached},
		{name: "attached malformed session cannot become a legacy request", mode: domain.ExecutionModeAttached, kind: "codex", sessionID: " ", want: map[string]string{"harnessKind": "codex", "harnessSessionId": " "}},
		{name: "attached malformed kind is retained for owner validation", mode: domain.ExecutionModeAttached, kind: " ", sessionID: "native-session", want: map[string]string{"harnessKind": " ", "harnessSessionId": "native-session"}},
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
			if reads != 1 || len(evidence) != 1 || err == nil || !strings.Contains(err.Error(), "exclusion is unknown") {
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
		{name: "attached key is not understood by an old owner", mode: domain.ExecutionModeAttached, kind: "codex", sessionID: "native-session"},
		{name: "attached key requires the exact identity version", mode: domain.ExecutionModeAttached, kind: "codex", sessionID: "native-session", version: "attached-key-v2"},
		{name: "attached key is understood by a current owner", mode: domain.ExecutionModeAttached, kind: "codex", sessionID: "native-session", version: "attached-key-v1", accepted: true},
		{name: "kind-only anchor also requires acknowledgement", mode: domain.ExecutionModeAttached, kind: "codex"},
		{name: "session-only anchor also requires acknowledgement", mode: domain.ExecutionModeAttached, sessionID: "native-session"},
		{name: "malformed nonempty anchor cannot bypass acknowledgement", mode: domain.ExecutionModeAttached, sessionID: " "},
		{name: "unanchored legacy attached request accepts old evidence", mode: domain.ExecutionModeAttached, accepted: true},
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
			} else if err == nil || !strings.Contains(err.Error(), "attached-key-v1") || len(evidence) != 0 {
				t.Fatalf("unacknowledged anchors became evidence: evidence=%+v err=%v", evidence, err)
			}
		})
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
	const count = 65536 // Full bounded protocol capacity, including retained history.
	runs := make([]*domain.Run, 0, count)
	refs := make([]ExecutorRef, 0, count)
	for i := 0; i < count; i++ {
		run := &domain.Run{ID: uuid.New(), RunnerPID: i + 1, RunnerPGID: i + 1}
		runs = append(runs, run)
		refs = append(refs, ExecutorRef{WorkRef: WorkRef{ID: run.ID.String(), Kind: "run", Status: "failed"}, PID: i + 1, PGID: i + 1})
	}
	lookups, reads := 0, 0
	owner := &ControlPlaneObserver{
		lookupBatch: func(context.Context, []ExecutorRef) ([]*domain.Run, error) {
			lookups++
			return runs, nil
		},
		read: func(_ context.Context, input []byte) ([]byte, error) {
			reads++
			var req struct {
				Executors []executorScopeRef `json:"executors"`
			}
			if err := json.Unmarshal(input, &req); err != nil || len(req.Executors) != count || len(input) <= 128<<10 || len(input) > 32<<20 {
				t.Fatalf("historical identities lost or request exceeded root bytes: refs=%d bytes=%d err=%v", len(req.Executors), len(input), err)
			}
			var out strings.Builder
			out.WriteString(`{"schemaVersion":"executor-scope-v1","complete":true,"executors":[`)
			for i, ref := range req.Executors {
				if i > 0 {
					out.WriteByte(',')
				}
				if i == count-1 {
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
		t.Fatalf("history fragmented or truncated: evidence=%d reads=%d lookups=%d err=%v", len(evidence), reads, lookups, err)
	}
	if last := evidence[count-1]; !last.Alive || len(last.PIDs) != 2 || last.PIDs[0] != 315576 {
		t.Fatalf("managed terminal residue at history tail lost: %+v", last)
	}
}
