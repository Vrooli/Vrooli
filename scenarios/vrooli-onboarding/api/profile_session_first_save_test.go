package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorstate"
	sessiondomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
)

type firstSaveEnvironment map[string]string

func (e firstSaveEnvironment) Getenv(key string) string { return e[key] }

func TestFirstProfileSaveRejectsMissingRevisionThenPersistsAnswersOnRetry(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	stateRoot := t.TempDir()
	statePath := filepath.Join(stateRoot, "operator-state.json")
	previousPath, previousRoots, previousEnvironment := operatorStatePath, operatorStateRoots, rootEnvironment
	operatorStatePath = func() (string, error) { return statePath, nil }
	operatorStateRoots = nil
	rootEnvironment = firstSaveEnvironment{"VROOLI_ROOT": repoRoot, "VROOLI_STORAGE_ROOT": stateRoot}
	t.Cleanup(func() {
		operatorStatePath, operatorStateRoots, rootEnvironment = previousPath, previousRoots, previousEnvironment
	})

	ctx := context.Background()
	initial, err := operatorStateService().Load(ctx)
	if err != nil {
		t.Fatalf("load fresh operator state: %v", err)
	}
	if initial.UpdatedAt != "" || operatorstate.Revision(initial) != initial.Version {
		t.Fatalf("fresh state revision = %q updated_at=%q version=%q; want schema-version fallback", operatorstate.Revision(initial), initial.UpdatedAt, initial.Version)
	}
	answers := map[string]json.RawMessage{"purpose": json.RawMessage(`"use-local-apps"`)}
	draft := sessiondomain.ProfileSession{
		Target: "local", Actor: "operator", Mode: "guided", ProfileID: "",
		BaseRevision: "", Answers: answers,
	}
	if _, err := (&Server{}).saveProfileSession(ctx, draft, ""); err == nil {
		t.Fatal("first save without a revision unexpectedly succeeded")
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("failed first save wrote operator state: stat err=%v", err)
	}

	// Retry the same user answers using the authoritative revision of the
	// untouched document, as the UI now does when updated_at is empty.
	baseRevision := operatorstate.Revision(initial)
	draft.BaseRevision = baseRevision
	saved, err := (&Server{}).saveProfileSession(ctx, draft, baseRevision)
	if err != nil {
		t.Fatalf("retry first profile save: %v", err)
	}
	if saved.Revision == "" || string(saved.Answers["purpose"]) != `"use-local-apps"` {
		t.Fatalf("saved first profile session = %+v", saved)
	}

	reloaded, err := operatorStateService().ProfileSession(ctx)
	if err != nil {
		t.Fatalf("reload first profile session: %v", err)
	}
	if reloaded == nil || reloaded.BaseRevision != baseRevision || string(reloaded.Answers["purpose"]) != `"use-local-apps"` {
		t.Fatalf("reloaded profile session = %+v, want retained first-save answer", reloaded)
	}
}
