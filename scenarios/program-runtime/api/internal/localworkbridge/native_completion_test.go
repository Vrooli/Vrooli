package localworkbridge

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type nativeTerminalFixture struct {
	result SourceBrokerProgram
	before func()
	after  func() error
}

func (f *nativeTerminalFixture) WithTerminal(ctx context.Context, _ *SourcePermit, effect func(context.Context, SourceBrokerProgram) error) error {
	if f.before != nil {
		f.before()
	}
	if err := effect(ctx, f.result); err != nil {
		return err
	}
	if f.after != nil {
		return f.after()
	}
	return nil
}
func completionFixture(t *testing.T, db *sql.DB) (*NativeReceiptOwner, *NativeSelection, *SourcePermit, *nativeTerminalFixture, *sourceBindingStub) {
	t.Helper()
	g, _, bindings := sourceFixture(t, SourceCollector)
	old, _, _ := nativeFixture(t, db)
	scope := old.scopeCopy()
	scope.Owner, scope.Subject = g.policy.Owner, g.policy.Subject
	terminal := &nativeTerminalFixture{}
	owner, err := NewNativeReceiptOwner(NativeReceiptConfig{Enabled: true, Scope: scope, Credentials: old.credentials, Guard: old.guard, Store: old.store, CollectorReconciler: terminal})
	if err != nil {
		t.Fatal(err)
	}
	owner.clock = old.clock
	if ok, err := owner.Accept(context.Background(), nativeRequest("event-1")); !ok || err != nil {
		t.Fatal(ok, err)
	}
	selection, err := owner.Select(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p, err := g.Admit(context.Background(), sourceRequest("POST", sourceLibrary+"GetDeclaredExecution", `{"name":"program-runtime.local-work-snapshot","idempotencyKey":"`+selection.key+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	events, _, _ := sourceFixture(t, SourceEvents)
	events.policy.EventIDs = selection.EventIDs()
	data, _ := json.Marshal(events.policy)
	hash := sha256.Sum256(data)
	events.digest = hex.EncodeToString(hash[:])
	if _, err := owner.BindSelection(context.Background(), selection, events, g); err != nil {
		t.Fatal(err)
	}
	terminal.result = SourceBrokerProgram{Found: true, Status: "PROGRAM_STATUS_SUCCEEDED", Stdout: `{"snapshot":true}`, Identity: SourceProgramIdentity{ID: "program-1", Name: SourceCollectorName, Digest: g.policy.CollectorDigest, RequestKey: selection.key, Subject: g.policy.Subject, Owner: g.policy.Owner, BindingID: g.policy.BindingID, BindingVersion: g.policy.BindingVersion, ScopeSHA256: g.digest}}
	return owner, selection, p, terminal, bindings
}
func assertNativeActive(t *testing.T, o *NativeReceiptOwner, want *NativeSelection) {
	t.Helper()
	got, err := o.Select(context.Background())
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("active changed: got=%+v err=%v", got, err)
	}
}
func TestNativeCompletionRetainsReceiptsAndNeverRenewsOldEvents(t *testing.T) {
	o, s, p, f, _ := completionFixture(t, nil)
	if err := o.ReconcileCollector(context.Background(), s, p); err != nil {
		t.Fatal(err)
	}
	if got, err := o.Select(context.Background()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if ok, err := o.Accept(context.Background(), nativeRequest("event-1")); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if ok, err := o.Accept(context.Background(), nativeRequest("event-2")); !ok || err != nil {
		t.Fatal(ok, err)
	}
	later, err := o.Select(context.Background())
	if err != nil || !reflect.DeepEqual(later.EventIDs(), []string{"event-2"}) || later.key == s.key {
		t.Fatal(later, err)
	}
	if err = o.ReconcileCollector(context.Background(), s, p); err != nil {
		t.Fatal(err)
	}
	assertNativeActive(t, o, later)
	f.result.Stdout = `{"changed":true}`
	if err = o.ReconcileCollector(context.Background(), s, p); err == nil {
		t.Fatal("changed terminal result accepted")
	}
	assertNativeActive(t, o, later)
	var n int
	if err = o.store.db.QueryRow("SELECT count(*) FROM local_work_native_receipts").Scan(&n); err != nil || n != 2 {
		t.Fatal(n, err)
	}
}
func TestNativeCompletionRejectsNonterminalAndChangedIdentity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*SourceBrokerProgram)
	}{
		{"missing", func(r *SourceBrokerProgram) { r.Found = false }},
		{"running", func(r *SourceBrokerProgram) { r.Status = "PROGRAM_STATUS_RUNNING"; r.Stdout = "" }},
		{"wrong-key", func(r *SourceBrokerProgram) { r.Identity.RequestKey = "local-work-" + strings.Repeat("c", 64) }},
		{"wrong-owner", func(r *SourceBrokerProgram) { r.Identity.Owner = "other" }},
		{"wrong-digest", func(r *SourceBrokerProgram) { r.Identity.Digest = strings.Repeat("f", 64) }},
		{"wrong-binding", func(r *SourceBrokerProgram) { r.Identity.BindingVersion = "v2" }},
		{"partial-failure", func(r *SourceBrokerProgram) { r.Status = "PROGRAM_STATUS_FAILED" }},
		{"oversize", func(r *SourceBrokerProgram) { r.Stdout = strings.Repeat("x", 65537) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, s, p, f, _ := completionFixture(t, nil)
			tc.change(&f.result)
			if err := o.ReconcileCollector(context.Background(), s, p); err == nil {
				t.Fatal("invalid terminal proof accepted")
			}
			assertNativeActive(t, o, s)
		})
	}
}
func TestNativeCompletionAllowsTerminalFailureWithoutRetry(t *testing.T) {
	for _, status := range []string{"PROGRAM_STATUS_FAILED", "PROGRAM_STATUS_CANCELLED"} {
		t.Run(status, func(t *testing.T) {
			o, s, p, f, _ := completionFixture(t, nil)
			f.result.Status = status
			f.result.Stdout = ""
			if err := o.ReconcileCollector(context.Background(), s, p); err != nil {
				t.Fatal(err)
			}
			if got, err := o.Select(context.Background()); got != nil || err != nil {
				t.Fatal(got, err)
			}
		})
	}
}
func TestNativeCompletionExpiryAndRevocationFailClosed(t *testing.T) {
	for _, mode := range []string{"source-revoked", "source-expired", "native-revoked", "cancelled", "commit-expired"} {
		t.Run(mode, func(t *testing.T) {
			o, s, p, f, b := completionFixture(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "source-revoked":
				f.before = func() { b.b.Revoked = true }
			case "source-expired":
				f.before = func() { p.gate.clock = func() time.Time { return time.Date(2026, 11, 7, 15, 0, 0, 0, time.UTC) } }
			case "native-revoked":
				o.guard.(*nativeTestGuard).revoked = true
			case "cancelled":
				f.before = cancel
			case "commit-expired":
				reads := 0
				b.onRead = func() {
					reads++
					if reads == 3 {
						b.b.Revoked = true
					}
				}
			}
			if err := o.ReconcileCollector(ctx, s, p); err == nil {
				t.Fatal("expired/revoked completion accepted")
			}
			o.guard.(*nativeTestGuard).revoked = false
			assertNativeActive(t, o, s)
		})
	}
}
func TestNativeCompletionAfterAdmissionDeadlineDoesNotRenewIt(t *testing.T) {
	o, s, p, _, _ := completionFixture(t, nil)
	p.gate.clock = func() time.Time { return time.Date(2026, 11, 7, 12, 6, 0, 0, time.UTC) }
	if err := o.ReconcileCollector(context.Background(), s, p); err != nil {
		t.Fatal(err)
	}
	var deadline string
	if err := o.store.db.QueryRow("SELECT admission_deadline FROM local_work_native_completed").Scan(&deadline); err != nil || deadline != "2026-11-07T12:05:00Z" {
		t.Fatal(deadline, err)
	}
}
func TestNativeCompletionLostResponseAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	o, s, p, f, _ := completionFixture(t, db)
	f.after = func() error { return errors.New("lost response") }
	if err = o.ReconcileCollector(context.Background(), s, p); err == nil {
		t.Fatal("missing lost response error")
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	o.store = NewNativeReceiptStore(db)
	f.after = nil
	if err = o.ReconcileCollector(context.Background(), s, p); err != nil {
		t.Fatal(err)
	}
	if got, err := o.Select(context.Background()); got != nil || err != nil {
		t.Fatal(got, err)
	}
}
func TestNativeCompletionConcurrentSelectAndRetry(t *testing.T) {
	o, s, p, _, _ := completionFixture(t, nil)
	if ok, err := o.Accept(context.Background(), nativeRequest("event-2")); !ok || err != nil {
		t.Fatal(ok, err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				if err := o.ReconcileCollector(context.Background(), s, p); err != nil {
					t.Error(err)
				}
			} else {
				got, err := o.Select(context.Background())
				if err != nil || got == nil {
					t.Error(got, err)
				}
			}
		}(i)
	}
	wg.Wait()
	later, err := o.Select(context.Background())
	if err != nil || !reflect.DeepEqual(later.EventIDs(), []string{"event-2"}) {
		t.Fatal(later, err)
	}
}
func TestNativeCompletionDisabledAndMissingReconciler(t *testing.T) {
	o, _ := NewNativeReceiptOwner(NativeReceiptConfig{})
	if err := o.ReconcileCollector(nil, nil, nil); err != ErrDisabled {
		t.Fatal(err)
	}
	o, s, p, _, _ := completionFixture(t, nil)
	o.collectorReconciler = nil
	if err := o.ReconcileCollector(context.Background(), s, p); err == nil {
		t.Fatal("missing owner accepted")
	}
	assertNativeActive(t, o, s)
}

func TestNativeCompletionIndependentHandlesSerializeRetirement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native.sqlite")
	type attempt struct {
		o *NativeReceiptOwner
		s *NativeSelection
		p *SourcePermit
	}
	attempts := make([]attempt, 4)
	for i := range attempts {
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		o, s, p, _, _ := completionFixture(t, db)
		attempts[i] = attempt{o, s, p}
	}
	o := attempts[0].o
	if ok, err := o.Accept(context.Background(), nativeRequest("event-2")); !ok || err != nil {
		t.Fatal(ok, err)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, a := range attempts {
		wg.Add(1)
		go func(a attempt) { defer wg.Done(); <-start; _ = a.o.ReconcileCollector(context.Background(), a.s, a.p) }(a)
	}
	close(start)
	wg.Wait()
	// A busy loser is allowed to fail closed; every retry must converge without
	// touching the next active selection, even on a different database handle.
	for _, a := range attempts {
		if err := a.o.ReconcileCollector(context.Background(), a.s, a.p); err != nil {
			t.Fatal(err)
		}
	}
	later, err := o.Select(context.Background())
	if err != nil || !reflect.DeepEqual(later.EventIDs(), []string{"event-2"}) {
		t.Fatal(later, err)
	}
	for _, a := range attempts {
		if err := a.o.ReconcileCollector(context.Background(), a.s, a.p); err != nil {
			t.Fatal(err)
		}
	}
	assertNativeActive(t, o, later)
	var n int
	if err = o.store.db.QueryRow("SELECT count(*) FROM local_work_native_completed").Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
}

func TestNativeCompletionRequiresDurableGateBinding(t *testing.T) {
	o, s, p, _, _ := completionFixture(t, nil)
	if _, err := o.store.db.Exec("DELETE FROM local_work_native_bindings"); err != nil {
		t.Fatal(err)
	}
	if err := o.ReconcileCollector(context.Background(), s, p); err == nil {
		t.Fatal("unbound selection retired")
	}
	assertNativeActive(t, o, s)
}
func TestNativeBindingCannotChangeAcrossRestart(t *testing.T) {
	o, s, p, _, _ := completionFixture(t, nil)
	events, _, _ := sourceFixture(t, SourceEvents)
	events.policy.EventIDs = s.EventIDs()
	raw, _ := json.Marshal(events.policy)
	sum := sha256.Sum256(raw)
	events.digest = hex.EncodeToString(sum[:])
	if _, err := o.BindSelection(context.Background(), s, events, p.gate); err != nil {
		t.Fatal(err)
	}
	// A newly constructed owner over the same durable store must not rebind.
	newer, err := NewNativeReceiptOwner(NativeReceiptConfig{Enabled: true, Scope: o.scopeCopy(), Credentials: o.credentials, Guard: o.guard, Store: o.store, CollectorReconciler: o.collectorReconciler})
	if err != nil {
		t.Fatal(err)
	}
	replacement := *p.gate
	replacement.digest = strings.Repeat("d", 64)
	if _, err = newer.BindSelection(context.Background(), s, events, &replacement); err == nil {
		t.Fatal("collector gate rebound")
	}
	changedEvents := *events
	changedEvents.digest = strings.Repeat("e", 64)
	if _, err = newer.BindSelection(context.Background(), s, &changedEvents, p.gate); err == nil {
		t.Fatal("events gate rebound")
	}
	if _, err = newer.BindSelection(context.Background(), s, events, p.gate); err != nil {
		t.Fatal(err)
	}
	if err = newer.ReconcileCollector(context.Background(), s, p); err != nil {
		t.Fatal(err)
	}
	if _, err = newer.BindSelection(context.Background(), s, events, p.gate); err == nil {
		t.Fatal("retired binding renewed")
	}
}
func TestNativeCompletionRejectsDifferentlyBoundCollector(t *testing.T) {
	o, s, p, _, _ := completionFixture(t, nil)
	if _, err := o.store.db.Exec("UPDATE local_work_native_bindings SET collector_sha256=?", strings.Repeat("f", 64)); err != nil {
		t.Fatal(err)
	}
	if err := o.ReconcileCollector(context.Background(), s, p); err == nil {
		t.Fatal("different approved collector retired selection")
	}
	assertNativeActive(t, o, s)
}

func TestNativeBindingConcurrentChoicesCannotReplaceWinner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	o, s, p, _, _ := completionFixture(t, db)
	if _, err = db.Exec("DELETE FROM local_work_native_bindings"); err != nil {
		t.Fatal(err)
	}
	db2, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	other, err := NewNativeReceiptOwner(NativeReceiptConfig{Enabled: true, Scope: o.scopeCopy(), Credentials: &nativeTestKey{}, Guard: &nativeTestGuard{}, Store: NewNativeReceiptStore(db2)})
	if err != nil {
		t.Fatal(err)
	}
	events, _, _ := sourceFixture(t, SourceEvents)
	events.policy.EventIDs = s.EventIDs()
	raw, _ := json.Marshal(events.policy)
	sum := sha256.Sum256(raw)
	events.digest = hex.EncodeToString(sum[:])
	replacement := *p.gate
	replacement.digest = strings.Repeat("d", 64)
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() { <-start; _, err := o.BindSelection(context.Background(), s, events, p.gate); results <- err }()
	go func() {
		<-start
		_, err := other.BindSelection(context.Background(), s, events, &replacement)
		results <- err
	}()
	close(start)
	first, second := <-results, <-results
	if first != nil && second != nil {
		t.Fatal("neither binding committed", first, second)
	}
	if first == nil && second == nil {
		t.Fatal("conflicting bindings both accepted")
	}
	var winner string
	if err = db.QueryRow("SELECT collector_sha256 FROM local_work_native_bindings").Scan(&winner); err != nil {
		t.Fatal(err)
	}
	winning, losing := p.gate, &replacement
	if winner == replacement.digest {
		winning, losing = losing, winning
	}
	if _, err = other.BindSelection(context.Background(), s, events, winning); err != nil {
		t.Fatal(err)
	}
	if _, err = o.BindSelection(context.Background(), s, events, losing); err == nil {
		t.Fatal("loser replaced winner")
	}
}
