// Tests that the sandbox-provenance v1.0.0 fields (schemaVersion, runOutcome,
// state, conversationId, costUsd) round-trip cleanly through the writer
// (RecordAppliedChanges) and the reader (GetPendingChangeFiles /
// GetPendingChangesByRun / GetFileProvenance).
//
// Pinning the wire shape here prevents drift against the shared package
// at packages/sandbox-provenance/go/schema.go.

package repository

import (
	"context"
	"testing"
	"time"

	sandboxprovenance "github.com/vrooli/sandbox-provenance"
	"workspace-sandbox/internal/types"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRecordAppliedChanges_PersistsProvenanceV1Fields(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewSandboxRepository(db)

	change := &types.AppliedChange{
		ID:                uuid.New(),
		SandboxID:         uuid.New(),
		SandboxOwner:      "agent-1",
		SandboxOwnerType:  string(types.OwnerTypeAgent),
		FilePath:          "/tmp/project/src/foo.go",
		ProjectRoot:       "/tmp/project",
		ChangeType:        "modified",
		FileSize:          42,
		AgentManagerRunID: "run-abc",
		SchemaVersion:     sandboxprovenance.SchemaVersion,
		RunOutcome:        string(sandboxprovenance.RunOutcomeSuccess),
		ProvenanceState:   string(sandboxprovenance.FileStateApplied),
		ConversationID:    "conv-xyz",
		CostUSD:           0.42,
	}

	mock.ExpectExec("INSERT INTO applied_changes").
		WithArgs(
			change.ID, change.SandboxID, change.SandboxOwner, change.SandboxOwnerType,
			change.FilePath, change.ProjectRoot, change.ChangeType, change.FileSize,
			change.AgentManagerRunID,
			change.SchemaVersion, change.RunOutcome, change.ProvenanceState,
			change.ConversationID, change.CostUSD,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.RecordAppliedChanges(context.Background(), []*types.AppliedChange{change}); err != nil {
		t.Fatalf("RecordAppliedChanges() = %v, want nil", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestRecordAppliedChanges_LegacyEmptyFieldsPersistAsNull(t *testing.T) {
	// Operator-driven Approve calls (not apply-at-run-end) leave the
	// schema-version'd fields empty. Persist them as SQL NULL so readers
	// can distinguish "legacy / pre-rollout" from "current with empty
	// values" once the contract evolves.
	db, mock := newMockDB(t)
	defer db.Close()
	repo := NewSandboxRepository(db)

	change := &types.AppliedChange{
		ID:               uuid.New(),
		SandboxID:        uuid.New(),
		SandboxOwner:     "user@example.com",
		SandboxOwnerType: string(types.OwnerTypeUser),
		FilePath:         "/tmp/project/x.txt",
		ProjectRoot:      "/tmp/project",
		ChangeType:       "added",
		FileSize:         1,
		// All five v1.0.0 fields are zero-valued.
	}

	mock.ExpectExec("INSERT INTO applied_changes").
		WithArgs(
			change.ID, change.SandboxID, change.SandboxOwner, change.SandboxOwnerType,
			change.FilePath, change.ProjectRoot, change.ChangeType, change.FileSize,
			nil, // agent_manager_run_id
			nil, // schema_version
			nil, // run_outcome
			nil, // provenance_state
			nil, // conversation_id
			nil, // cost_usd
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.RecordAppliedChanges(context.Background(), []*types.AppliedChange{change}); err != nil {
		t.Fatalf("RecordAppliedChanges() = %v, want nil", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestGetPendingChangesByRun_HydratesProvenanceV1Fields(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	repo := NewSandboxRepository(db)

	now := time.Now().UTC()
	cols := []string{
		"agent_manager_run_id", "sandbox_id", "sandbox_owner", "file_path",
		"change_type", "applied_at",
		"run_outcome", "conversation_id", "cost_usd", "provenance_state",
	}
	mock.ExpectQuery("FROM applied_changes").
		WithArgs("/tmp/project").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			"run-1", "11111111-1111-1111-1111-111111111111", "agent-1",
			"/tmp/project/src/foo.go", "modified", now,
			"success", "conv-1", 0.42, "applied",
		))

	groups, err := repo.GetPendingChangesByRun(context.Background(), "/tmp/project")
	if err != nil {
		t.Fatalf("GetPendingChangesByRun() = %v, want nil", err)
	}
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	g := groups[0]
	if g.RunOutcome != "success" {
		t.Errorf("RunOutcome = %q, want %q", g.RunOutcome, "success")
	}
	if g.ConversationID != "conv-1" {
		t.Errorf("ConversationID = %q, want %q", g.ConversationID, "conv-1")
	}
	if g.CostUSD != 0.42 {
		t.Errorf("CostUSD = %v, want 0.42", g.CostUSD)
	}
	if len(g.Files) != 1 || g.Files[0].State != types.ProvenanceFileStateApplied {
		t.Errorf("Files[0].State = %v, want %q", g.Files[0].State, types.ProvenanceFileStateApplied)
	}
}
