package agentsessions

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreReadsLegacyArtifactsForMigration(t *testing.T) {
	store := NewFileStore(t.TempDir())
	session := validStoredSession("sess_store")
	if err := store.CreateSession(session); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := store.AppendMessage(session.ID, Message{
		ID:        "msg-1",
		Role:      MessageRoleUser,
		Content:   "hello",
		CreatedAt: testTimestamp,
	}); err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}
	if err := store.SaveProposal(session.ID, Proposal{
		ID:          "prop-1",
		Kind:        ProposalBacklogBatchImport,
		Status:      ProposalStatusReady,
		Summary:     "Batch import",
		PayloadJSON: `{"items":[]}`,
		CreatedAt:   testTimestamp,
		UpdatedAt:   testTimestamp,
	}); err != nil {
		t.Fatalf("SaveProposal() error = %v", err)
	}
	writeLegacyArtifacts(t, store, Artifact{
		ID:           "art-1",
		SessionID:    session.ID,
		ArtifactType: ArtifactMilestone,
		Action:       ArtifactActionProposed,
		EntityRef:    "quality-gates",
		Title:        "Quality Gates",
		CreatedAt:    testTimestamp,
	})

	loaded, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if len(loaded.Messages) != 1 || len(loaded.Proposals) != 1 || len(loaded.Artifacts) != 1 {
		t.Fatalf("loaded counts = messages:%d proposals:%d artifacts:%d", len(loaded.Messages), len(loaded.Proposals), len(loaded.Artifacts))
	}
	if loaded.Status != StatusProposalReady {
		t.Fatalf("status = %q, want proposal_ready", loaded.Status)
	}

	artifacts, err := store.ListArtifactsByEntity(ArtifactMilestone, "quality-gates")
	if err != nil {
		t.Fatalf("ListArtifactsByEntity() error = %v", err)
	}
	if len(artifacts) != 1 || artifacts[0].ID != "art-1" {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}
}

func TestMigrateLegacySourceDataCopiesSessionsWithoutDeletingSource(t *testing.T) {
	scenarioRoot := t.TempDir()
	dataRoot := t.TempDir()
	legacyStore := NewFileStore(scenarioRoot)
	session := validStoredSession("sess_legacy_migration")
	if err := legacyStore.CreateSession(session); err != nil {
		t.Fatal(err)
	}
	if err := legacyStore.AppendMessage(session.ID, Message{ID: "msg-1", Role: MessageRoleUser, Content: "preserve me", CreatedAt: testTimestamp}); err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacySourceData(scenarioRoot, dataRoot); err != nil {
		t.Fatalf("MigrateLegacySourceData() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(scenarioRoot, "agent-sessions", session.ID, sessionFileName)); err != nil {
		t.Fatalf("legacy source data was removed: %v", err)
	}
	migrated, err := NewFileStore(dataRoot).LoadSession(session.ID)
	if err != nil {
		t.Fatalf("migrated session unavailable: %v", err)
	}
	if len(migrated.Messages) != 1 || migrated.Messages[0].Content != "preserve me" {
		t.Fatalf("migrated session = %+v", migrated)
	}
	if err := MigrateLegacySourceData(scenarioRoot, dataRoot); err != nil {
		t.Fatalf("idempotent migration error = %v", err)
	}
}

func TestMigrateLegacySourceDataRefusesToMergeIntoExistingData(t *testing.T) {
	scenarioRoot := t.TempDir()
	dataRoot := t.TempDir()
	if err := NewFileStore(scenarioRoot).CreateSession(validStoredSession("sess_legacy")); err != nil {
		t.Fatal(err)
	}
	if err := NewFileStore(dataRoot).CreateSession(validStoredSession("sess_existing")); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacySourceData(scenarioRoot, dataRoot); err == nil {
		t.Fatal("MigrateLegacySourceData() error = nil, want refusal to merge")
	}
}

func TestFileStoreListFiltersAndLimit(t *testing.T) {
	store := NewFileStore(t.TempDir())
	first := validStoredSession("sess_first")
	first.UpdatedAt = "2026-05-01T12:00:00Z"
	second := validStoredSession("sess_second")
	second.Kind = KindSwarmOperations
	second.SkillID = SkillSwarmOperations
	second.Status = StatusComplete
	second.UpdatedAt = "2026-05-01T13:00:00Z"
	for _, session := range []Session{first, second} {
		if err := store.CreateSession(session); err != nil {
			t.Fatalf("CreateSession(%s) error = %v", session.ID, err)
		}
	}

	active, err := store.ListSessions(ListFilters{ActiveOnly: true})
	if err != nil {
		t.Fatalf("ListSessions(active) error = %v", err)
	}
	if len(active) != 1 || active[0].ID != first.ID {
		t.Fatalf("active sessions = %+v", active)
	}

	limited, err := store.ListSessions(ListFilters{Limit: 1})
	if err != nil {
		t.Fatalf("ListSessions(limit) error = %v", err)
	}
	if len(limited) != 1 || limited[0].ID != second.ID {
		t.Fatalf("limited sessions = %+v", limited)
	}
}

func TestFileStoreListSkipsLegacyInvalidSession(t *testing.T) {
	store := NewFileStore(t.TempDir())
	valid := validStoredSession("sess_valid")
	if err := store.CreateSession(valid); err != nil {
		t.Fatal(err)
	}
	legacy := validStoredSession("sess_legacy")
	legacy.Kind = "operating_mode_authoring"
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.sessionDir(legacy.ID), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.sessionDir(legacy.ID), sessionFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}

	sessions, err := store.ListSessions(ListFilters{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != valid.ID {
		t.Fatalf("sessions = %+v, want only valid session", sessions)
	}
}

func TestFileStoreLoadMissingReturnsNotFound(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "missing-root"))
	if _, err := store.LoadSession("sess_missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadSession() error = %v, want ErrNotFound", err)
	}
}

func TestFileStoreDeleteSessionRemovesSessionOwnedFiles(t *testing.T) {
	root := t.TempDir()
	store := NewFileStore(root)
	session := validStoredSession("sess_delete")
	if err := store.CreateSession(session); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := store.AppendMessage(session.ID, Message{
		ID:        "msg-1",
		Role:      MessageRoleUser,
		Content:   "delete me",
		CreatedAt: testTimestamp,
	}); err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}
	if err := store.SaveProposal(session.ID, Proposal{
		ID:          "prop-delete",
		Kind:        ProposalBacklogBatchImport,
		Status:      ProposalStatusReady,
		Summary:     "Delete proposal",
		PayloadJSON: `{"items":[]}`,
		CreatedAt:   testTimestamp,
		UpdatedAt:   testTimestamp,
	}); err != nil {
		t.Fatalf("SaveProposal() error = %v", err)
	}
	writeLegacyArtifacts(t, store, Artifact{
		ID:           "art-delete",
		SessionID:    session.ID,
		ArtifactType: ArtifactFile,
		Action:       ArtifactActionLinked,
		EntityRef:    "README.md",
		CreatedAt:    testTimestamp,
	})

	sessionDir := filepath.Join(root, "agent-sessions", session.ID)
	if _, err := os.Stat(filepath.Join(sessionDir, sessionFileName)); err != nil {
		t.Fatalf("session file before delete: %v", err)
	}

	if err := store.DeleteSession(session.ID); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	if _, err := os.Stat(sessionDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("session dir after delete stat error = %v, want not exist", err)
	}
	if _, err := store.LoadSession(session.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadSession() after delete error = %v, want ErrNotFound", err)
	}
	sessions, err := store.ListSessions(ListFilters{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions after delete = %+v", sessions)
	}
}

func writeLegacyArtifacts(t *testing.T, store *FileStore, artifacts ...Artifact) {
	t.Helper()
	if len(artifacts) == 0 {
		return
	}
	for index, artifact := range artifacts {
		if err := artifact.Validate(); err != nil {
			t.Fatalf("legacy artifact %d validation: %v", index, err)
		}
	}
	if err := writeJSONLAtomic(filepath.Join(store.sessionDir(artifacts[0].SessionID), artifactsFileName), artifacts); err != nil {
		t.Fatalf("write legacy artifacts: %v", err)
	}
}

func TestFileStoreDeleteMissingAndUnsafeSessionIDs(t *testing.T) {
	root := t.TempDir()
	store := NewFileStore(root)
	outside := filepath.Join(root, "outside.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	if err := store.DeleteSession("sess_missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteSession(missing) error = %v, want ErrNotFound", err)
	}
	for _, id := range []string{"", "../outside", "sess_../outside", "bad_id", "sess_nested/path"} {
		if err := store.DeleteSession(id); !errors.Is(err, ErrValidation) {
			t.Fatalf("DeleteSession(%q) error = %v, want ErrValidation", id, err)
		}
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "keep" {
		t.Fatalf("outside file changed: data=%q err=%v", data, err)
	}
}

func validStoredSession(id string) Session {
	session := validSession()
	session.ID = id
	session.Status = StatusRunning
	session.Messages = nil
	session.Proposals = nil
	session.Artifacts = nil
	session.CreatedBy = &Attribution{Type: AttributionOperator}
	return session
}
