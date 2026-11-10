package wiring

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/maintenance"
	"agent-manager/internal/orchestration/testutil"
	"github.com/google/uuid"
)

func TestMaintenanceBatchHydratesDurableIdentityWithoutActionOrConfig(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, _, _ := testutil.SetupTestReposWithDB(t, db)
	task := &domain.Task{ID: uuid.New(), Title: "scope fixture", ScopePath: "."}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	start, end := time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(-time.Minute)
	// Given a durable attached run with separate native and mutable sessions,
	// plus credentials and configuration that are not executor identity.
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Tag: "durable-tag", Status: domain.RunStatusFailed,
		RunnerPID: 315576, RunnerPGID: 315576, StartedAt: &start, EndedAt: &end,
		ExecutionMode: domain.ExecutionModeAttached, HarnessKind: "codex", HarnessSessionID: "immutable-native-session",
		SessionID: "mutable-transcript-session", IdentityTokenHash: "must-not-hydrate-token", IdentityRenewalAnchorHash: "must-not-hydrate-renewal",
		CustomEnv: map[string]string{"secret": "must-not-hydrate"}, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeOpenCode},
	}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	// When the inventory projection reads the run, then only durable identity is hydrated.
	identities, err := readMaintenanceExecutorIdentities(t.Context(), db, []maintenance.ExecutorRef{{WorkRef: maintenance.WorkRef{ID: run.ID.String()}}})
	if err != nil || len(identities) != 1 {
		t.Fatalf("scope identities=%+v err=%v", identities, err)
	}
	got := identities[0]
	if got.ID != run.ID || got.Tag != run.Tag || got.RunnerPID != run.RunnerPID || got.RunnerPGID != run.RunnerPGID || got.StartedAt == nil || !got.StartedAt.Equal(start) || got.EndedAt == nil || !got.EndedAt.Equal(end) {
		t.Fatalf("durable physical binding lost: %+v", got)
	}
	if got.ExecutionMode != domain.ExecutionModeAttached || got.HarnessKind != "codex" || got.HarnessSessionID != "immutable-native-session" {
		t.Fatalf("immutable attached identity lost: mode=%q harness=%q session=%q", got.ExecutionMode, got.HarnessKind, got.HarnessSessionID)
	}
	if got.ResolvedConfig != nil || got.CustomEnv != nil || got.Actions != nil || got.IdentityTokenHash != "" || got.IdentityRenewalAnchorHash != "" || got.SessionID != "" {
		t.Fatal("inventory hydrated unrelated action/config/secret data")
	}
	// A complete retained inventory must not become one SQL statement with
	// more bind parameters than the storage engine supports. Most requested
	// identities are deliberately missing: the observer must detect that later,
	// rather than the hydration query failing or fabricating run data.
	refs := make([]maintenance.ExecutorRef, 65536)
	for i := range refs {
		refs[i] = maintenance.ExecutorRef{WorkRef: maintenance.WorkRef{ID: uuid.NewString()}}
	}
	refs[len(refs)-1].ID = run.ID.String()
	identities, err = readMaintenanceExecutorIdentities(t.Context(), db, refs)
	if err != nil || len(identities) != 1 || identities[0].ID != run.ID || identities[0].HarnessSessionID != "immutable-native-session" {
		t.Fatalf("bounded hydration lost tail identity: identities=%d err=%v", len(identities), err)
	}
}

func TestMaintenanceBatchRetainsPartialAndLegacyAnchors(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		mode, kind, sessionID           any
		wantMode, wantKind, wantSession string
	}{
		{name: "legacy null anchors", mode: nil, kind: nil, sessionID: nil},
		{name: "attached kind without canonical session", mode: "attached", kind: "codex", sessionID: nil, wantMode: "attached", wantKind: "codex"},
		{name: "attached canonical session without kind", mode: "attached", kind: nil, sessionID: "native-session", wantMode: "attached", wantSession: "native-session"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, cleanup := testutil.SetupTestDB(t)
			t.Cleanup(cleanup)
			repos, _, _ := testutil.SetupTestReposWithDB(t, db)
			task := &domain.Task{ID: uuid.New(), Title: "legacy scope fixture", ScopePath: "."}
			if err := repos.Tasks.Create(t.Context(), task); err != nil {
				t.Fatal(err)
			}
			run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusFailed}
			if err := repos.Runs.Create(t.Context(), run); err != nil {
				t.Fatal(err)
			}
			// Given legacy or partially populated persisted identity, preserve
			// missing columns rather than substituting another session key.
			if _, err := db.ExecContext(t.Context(), `UPDATE runs SET execution_mode=?,harness_kind=?,harness_session_id=? WHERE id=?`, tc.mode, tc.kind, tc.sessionID, run.ID.String()); err != nil {
				t.Fatal(err)
			}
			// When the bounded projection hydrates the run, then all available
			// anchors survive and SQL NULL remains an absent anchor.
			runs, err := readMaintenanceExecutorIdentities(t.Context(), db, []maintenance.ExecutorRef{{WorkRef: maintenance.WorkRef{ID: run.ID.String()}}})
			if err != nil || len(runs) != 1 {
				t.Fatalf("identity count=%d err=%v", len(runs), err)
			}
			got := runs[0]
			if got.ID != run.ID || string(got.ExecutionMode) != tc.wantMode || got.HarnessKind != tc.wantKind || got.HarnessSessionID != tc.wantSession {
				t.Fatalf("identity changed: id=%s mode=%q harness=%q session=%q", got.ID, got.ExecutionMode, got.HarnessKind, got.HarnessSessionID)
			}
		})
	}
}
