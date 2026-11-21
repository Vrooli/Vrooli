package rootmetrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/statecontract"
)

func TestRootAttemptRecordsOnlyBoundedObservations(t *testing.T) {
	home := t.TempDir()
	a := Begin(home, "rebaseline")
	if a == nil {
		t.Fatal("attempt not created")
	}
	a.Identities(strings.Repeat("a", 64), "/private/source")
	a.Event("confirmation_started", "")
	a.Event("refused", "state_contract_changed")
	a.Event("/private/source", "sensitive error")
	b, err := os.ReadFile(a.path)
	if err != nil {
		t.Fatal(err)
	}
	var got rootAttemptRecord
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != rootAttemptSchema || got.Kind != "rebaseline" || got.ID == "" || len(got.Events) != 3 || got.Events[2].Reason != "state_contract_changed" {
		t.Fatalf("unexpected record: %+v", got)
	}
	if got.ServingContract != strings.Repeat("a", 64) || got.CandidateContract != "" || strings.Contains(string(b), "/private") {
		t.Fatalf("unexpected identity or leaked input: %s", b)
	}
	info, err := os.Stat(a.path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("record permissions: %v %v", info, err)
	}
	other := Begin(home, "rebaseline")
	if other == nil || a.path == other.path {
		t.Fatal("attempt IDs must be unique")
	}
}

func TestRootAttemptPrunesOnlyOldObservations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(state, "root-rebaseline", "attempts")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	oldAttempt := Begin(home, "install")
	old := oldAttempt.path
	recent := Begin(home, "install").path
	foreign := filepath.Join(dir, "keep.json")
	malformed := filepath.Join(dir, "20000101T000000.000000000Z-00000000000000000000000000000000.json")
	for _, p := range []string{foreign, malformed} {
		if err := os.WriteFile(p, []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-rootAttemptRetention - time.Hour)
	for _, p := range []string{old, foreign, malformed} {
		if err := os.Chtimes(p, past, past); err != nil {
			t.Fatal(err)
		}
	}
	pruneRootAttempts(dir, time.Now())
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old observation retained: %v", err)
	}
	for _, p := range []string{recent, foreign, malformed} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("removed %s: %v", p, err)
		}
	}
	// The observation's unique filename must not create an immortal lock for
	// every attempt, including after its record has expired.
	locks := filepath.Join(state, "locks", "owned-writes")
	if entries, err := os.ReadDir(locks); err == nil && len(entries) != 0 {
		t.Fatalf("observation left permanent owned-write locks: %v", entries)
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestRootAttemptFailureIsBestEffort(t *testing.T) {
	home := t.TempDir()
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(state, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "root-rebaseline"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	a := Begin(home, "install")
	if a != nil {
		t.Fatal("expected unavailable observation store")
	}
	a.Event("failed", "unavailable")
	a.Identities("", "")
}

func TestOwnerAttributionUsesManifestDigestsAndBoundIdentities(t *testing.T) {
	a := Begin(t.TempDir(), "install")
	p := statecontract.Manifest{Version: 1, Contract: strings.Repeat("a", 64), Toolchain: "go1", Owners: []statecontract.OwnerManifest{{Owner: "changed", Digest: "one"}, {Owner: "removed", Digest: "gone"}, {Owner: "same", Digest: "same"}}}
	c := statecontract.Manifest{Version: 1, Contract: strings.Repeat("b", 64), Toolchain: "go2", Owners: []statecontract.OwnerManifest{{Owner: "changed", Digest: "two"}, {Owner: "added", Digest: "new"}, {Owner: "same", Digest: "same"}}}
	a.Identities(p.Contract, c.Contract)
	a.attributeManifests(p, c)
	a.Event("refused", "state_contract_changed")
	if a.record.Attribution != "compared_toolchain_changed" || strings.Join(a.record.Owners, ",") != "added,changed,removed" {
		t.Fatalf("bad attribution: %+v", a.record)
	}
	a.Identities(strings.Repeat("c", 64), c.Contract)
	a.record.Owners = nil
	a.attributeManifests(p, c)
	if a.record.Attribution != "identity_mismatch" || len(a.record.Owners) != 0 {
		t.Fatalf("unbound evidence: %+v", a.record)
	}
}

func TestOperationIdentityIsBoundedAndPersisted(t *testing.T) {
	a := Begin(t.TempDir(), "activation_observation")
	a.Operation("/private/path")
	a.Event("terminal", "verified")
	if a.record.OperationID != "" {
		t.Fatal("accepted invalid operation identity")
	}
	id := strings.Repeat("a", 32)
	a.Operation(id)
	a.Event("terminal", "verified")
	b, err := os.ReadFile(a.path)
	if err != nil {
		t.Fatal(err)
	}
	var got rootAttemptRecord
	if err := json.Unmarshal(b, &got); err != nil || got.OperationID != id {
		t.Fatalf("operation identity: %+v %v", got, err)
	}
}

func TestAttributionRejectsUnsafeOwnerNames(t *testing.T) {
	a := Begin(t.TempDir(), "install")
	p := statecontract.Manifest{Version: 1, Contract: strings.Repeat("a", 64), Owners: []statecontract.OwnerManifest{{Owner: "/private/path", Digest: "one"}}}
	c := statecontract.Manifest{Version: 1, Contract: strings.Repeat("b", 64)}
	a.Identities(p.Contract, c.Contract)
	a.attributeManifests(p, c)
	a.Event("qualification", "state_contract_changed")
	data, err := os.ReadFile(a.path)
	if err != nil || strings.Contains(string(data), "/private/path") || a.record.Attribution != "invalid_or_excess_owners" {
		t.Fatalf("unsafe attribution: %s %v", data, err)
	}
}

func TestBundleEvidenceIsBoundedAndPrivate(t *testing.T) {
	a := Begin(t.TempDir(), "bundle_publication")
	hash := strings.Repeat("a", 64)
	a.CandidateImage(hash)
	a.Bundle(hash, map[string]string{"vrooli": hash, "vrooli-api": "/private/source", "/private/name": hash})
	a.Event("vrooli_api", "published")
	data, err := os.ReadFile(a.path)
	if err != nil {
		t.Fatal(err)
	}
	var got rootAttemptRecord
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.BundleManifest != hash || got.CandidateImage != hash || got.Images["vrooli"] != hash || len(got.Images) != 1 || strings.Contains(string(data), "/private") {
		t.Fatalf("unsafe evidence: %s", data)
	}
}

func TestObservationWriteReportsUnconfirmedPublication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "observation.json")
	if err := writeObservation(path, []byte("{}\n")); err == nil {
		t.Fatal("missing observation directory was reported durable")
	}
}

func TestAsyncObservationsFlushInOrder(t *testing.T) {
	a := Begin(t.TempDir(), "rebaseline")
	a.Async()
	a.Event("confirmation", "started")
	a.Event("terminal", "failed")
	a.Flush()
	data, err := os.ReadFile(a.path)
	var r rootAttemptRecord
	if err != nil || json.Unmarshal(data, &r) != nil || len(r.Events) != 3 || r.Events[2].Reason != "failed" {
		t.Fatalf("queued events not flushed in order: %s %v", data, err)
	}
}

func TestAsyncObservationDoesNotWaitForBlockedWrite(t *testing.T) {
	a := Begin(t.TempDir(), "install")
	entered, release := make(chan struct{}), make(chan struct{})
	first := true
	a.write = func(string, []byte) error {
		if first {
			first = false
			close(entered)
			<-release
		}
		return nil
	}
	a.Async()
	a.Event("admission_approved", "")
	<-entered
	returned := make(chan struct{})
	go func() { a.Event("terminal", "dispatched"); close(returned) }()
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("event waited for stalled observation write")
	}
	flushed := make(chan struct{})
	go func() { a.Flush(); close(flushed) }()
	select {
	case <-flushed:
	case <-time.After(3 * time.Second):
		t.Fatal("flush waited indefinitely for stalled observation write")
	}
	close(release)
	<-a.done
}

func TestAsyncEventRacingFlushCannotPanic(t *testing.T) {
	for i := 0; i < 100; i++ {
		a := Begin(t.TempDir(), "rebaseline")
		a.Async()
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			a.Event("terminal", "failed")
		}()
		go func() {
			defer wg.Done()
			<-start
			a.Flush()
		}()
		close(start)
		wg.Wait()
		data, err := os.ReadFile(a.path)
		if err != nil {
			t.Fatalf("flush did not persist a valid observation: %v", err)
		}
		var record rootAttemptRecord
		if err := json.Unmarshal(data, &record); err != nil || len(record.Events) == 0 {
			t.Fatalf("invalid observation after flush race: %s (%v)", data, err)
		}
	}
}

func TestPruneRemovesOnlyOldObservationTemps(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, ".observation-123")
	recent := filepath.Join(dir, ".observation-456")
	foreign := filepath.Join(dir, ".observation-foreign")
	for _, path := range []string{old, recent, foreign} {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-rootAttemptRetention - time.Hour)
	for _, path := range []string{old, foreign} {
		if err := os.Chtimes(path, past, past); err != nil {
			t.Fatal(err)
		}
	}
	pruneRootAttempts(dir, time.Now())
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old temp retained: %v", err)
	}
	for _, path := range []string{recent, foreign} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("removed %s: %v", path, err)
		}
	}
}
