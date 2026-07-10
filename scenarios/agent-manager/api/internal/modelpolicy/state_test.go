package modelpolicy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStateInitialLoadAndStatus(t *testing.T) {
	path := writeCatalogFile(t, validCatalog())
	now := time.Date(2026, 7, 9, 20, 0, 0, 0, time.FixedZone("EDT", -4*60*60))
	state, err := newState(path, Requirement{Required: true, Reason: "test profile uses policy"}, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new state: %v", err)
	}

	status := state.Status()
	if !status.Ready || status.ActiveDigest == "" {
		t.Fatalf("unexpected ready status: %+v", status)
	}
	if status.ActivatedAt == nil || !status.ActivatedAt.Equal(now.UTC()) {
		t.Fatalf("activatedAt = %v, want %v", status.ActivatedAt, now.UTC())
	}
	if status.LastReloadAttempt == nil || !status.LastReloadAttempt.Succeeded {
		t.Fatalf("reload attempt = %+v, want success", status.LastReloadAttempt)
	}
	if err := state.ReadinessError(); err != nil {
		t.Fatalf("readiness error: %v", err)
	}
	if got := state.ModelIDs("codex"); len(got) != 1 || got[0] != "gpt-current" {
		t.Fatalf("codex models = %v", got)
	}
}

func TestStateRequiredInvalidCatalogFailsReadinessWithPathAndCause(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1}`), 0o644); err != nil {
		t.Fatalf("write invalid catalog: %v", err)
	}

	state, loadErr := NewState(path, Requirement{Required: true, Reason: "policy-backed profile exists"})
	if loadErr == nil {
		t.Fatal("expected initial load failure")
	}
	status := state.Status()
	if status.Ready || status.ActiveDigest != "" {
		t.Fatalf("unexpected invalid status: %+v", status)
	}
	if status.LastReloadAttempt == nil || status.LastReloadAttempt.Diagnostic == nil {
		t.Fatalf("missing diagnostic: %+v", status.LastReloadAttempt)
	}
	if status.LastReloadAttempt.Diagnostic.Code != DiagnosticCodeCatalogInvalid {
		t.Fatalf("diagnostic code = %q", status.LastReloadAttempt.Diagnostic.Code)
	}
	readyErr := state.ReadinessError()
	if readyErr == nil || !strings.Contains(readyErr.Error(), path) || !strings.Contains(readyErr.Error(), "modelPolicyCatalog.metadata.catalogId") {
		t.Fatalf("readiness error = %v, want path and validation cause", readyErr)
	}
}

func TestStateOptionalInvalidCatalogRemainsReady(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	state, loadErr := NewState(path, Requirement{Required: false})
	if loadErr == nil {
		t.Fatal("expected initial load failure")
	}
	if status := state.Status(); !status.Ready || status.LastReloadAttempt == nil || status.LastReloadAttempt.Diagnostic == nil {
		t.Fatalf("optional status = %+v", status)
	}
	if err := state.ReadinessError(); err != nil {
		t.Fatalf("optional catalog blocked readiness: %v", err)
	}
}

func TestStateFailedReloadRetainsActiveRevision(t *testing.T) {
	path := writeCatalogFile(t, validCatalog())
	state, err := NewState(path, Requirement{Required: true})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	before := state.Status()

	if err := os.WriteFile(path, []byte(`{"schemaVersion":99}`), 0o644); err != nil {
		t.Fatalf("write invalid catalog: %v", err)
	}
	if _, err := state.Reload(); err == nil {
		t.Fatal("expected reload failure")
	}

	after := state.Status()
	if !after.Ready || after.ActiveDigest != before.ActiveDigest {
		t.Fatalf("failed reload changed active state: before=%+v after=%+v", before, after)
	}
	if after.LastReloadAttempt == nil || after.LastReloadAttempt.Succeeded || after.LastReloadAttempt.Diagnostic == nil {
		t.Fatalf("failed reload diagnostic = %+v", after.LastReloadAttempt)
	}
}

func TestStateSuccessfulReloadAtomicallyActivatesNewRevision(t *testing.T) {
	path := writeCatalogFile(t, validCatalog())
	state, err := NewState(path, Requirement{Required: true})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	before := state.Status()

	updated := validCatalog()
	inventory := updated.Runners["codex"]
	inventory.Models = append(inventory.Models, Model{ID: "gpt-next", Description: "Next test model"})
	updated.Runners["codex"] = inventory
	writeCatalogAtPath(t, path, updated)
	if _, err := state.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}

	after := state.Status()
	if !after.Ready || after.ActiveDigest == before.ActiveDigest {
		t.Fatalf("successful reload did not activate new digest: before=%+v after=%+v", before, after)
	}
	models := state.ModelIDs("codex")
	if len(models) != 2 || models[1] != "gpt-next" {
		t.Fatalf("active models = %v", models)
	}
}

func TestStateValidateDoesNotActivateCandidate(t *testing.T) {
	path := writeCatalogFile(t, validCatalog())
	state, err := NewState(path, Requirement{Required: true})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}
	before := state.Status()

	updated := validCatalog()
	updated.Metadata.CatalogID = "validated-not-activated"
	writeCatalogAtPath(t, path, updated)
	candidate, err := state.Validate()
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if candidate.Digest() == before.ActiveDigest {
		t.Fatal("test candidate did not change digest")
	}
	if after := state.Status(); after.ActiveDigest != before.ActiveDigest {
		t.Fatalf("validate changed active digest: before=%s after=%s", before.ActiveDigest, after.ActiveDigest)
	}
}

func TestStateConcurrentReadersObserveOnlyCompleteRevisions(t *testing.T) {
	path := writeCatalogFile(t, validCatalog())
	state, err := NewState(path, Requirement{Required: true})
	if err != nil {
		t.Fatalf("new state: %v", err)
	}

	var stop atomic.Bool
	var invalid atomic.Bool
	var readers sync.WaitGroup
	for range 8 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for !stop.Load() {
				models := state.ModelIDs("codex")
				if len(models) != 1 && len(models) != 2 {
					invalid.Store(true)
					return
				}
			}
		}()
	}

	updated := validCatalog()
	inventory := updated.Runners["codex"]
	inventory.Models = append(inventory.Models, Model{ID: "gpt-next", Description: "Next test model"})
	updated.Runners["codex"] = inventory
	writeCatalogAtPath(t, path, updated)
	if _, err := state.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	stop.Store(true)
	readers.Wait()
	if invalid.Load() {
		t.Fatal("reader observed a partial catalog revision")
	}
}

func writeCatalogFile(t *testing.T, catalog *Catalog) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model-policy-catalog.json")
	writeCatalogAtPath(t, path, catalog)
	return path
}

func writeCatalogAtPath(t *testing.T, path string, catalog *Catalog) {
	t.Helper()
	data, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
}
