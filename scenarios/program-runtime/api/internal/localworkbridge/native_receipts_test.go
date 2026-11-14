package localworkbridge

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/databasetest"
)

type nativeTestGuard struct {
	mu      sync.Mutex
	revoked bool
	calls   int
	enter   func()
}

func (g *nativeTestGuard) WithCurrent(ctx context.Context, _ NativeReceiptScope, f func(context.Context) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++
	if g.revoked || ctx.Err() != nil {
		return ErrSourceAuthority
	}
	if g.enter != nil {
		g.enter()
	}
	return f(ctx)
}

type nativeTestKey struct {
	calls int
	last  []byte
	scope NativeReceiptScope
	fail  bool
}

func (k *nativeTestKey) VerificationKey(_ context.Context, s NativeReceiptScope) ([]byte, error) {
	k.calls++
	k.scope = s
	k.last = bytes.Repeat([]byte{7}, 32)
	if k.fail {
		return k.last, ErrSourceAuthority
	}
	return k.last, nil
}
func nativeFixture(t *testing.T, db *sql.DB) (*NativeReceiptOwner, *nativeTestGuard, *nativeTestKey) {
	t.Helper()
	if db == nil {
		db = databasetest.NewSQLite(t)
	}
	guard, key := &nativeTestGuard{}, &nativeTestKey{}
	o, err := NewNativeReceiptOwner(NativeReceiptConfig{Enabled: true, Scope: NativeReceiptScope{Owner: "owner", Subject: "subject", SourceInstance: "swarminator", SourceEpoch: "epoch-1", AllowedTypes: nativeEventTypes}, Store: NewNativeReceiptStore(db), Guard: guard, Credentials: key})
	if err != nil {
		t.Fatal(err)
	}
	o.clock = func() time.Time { return time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) }
	return o, guard, key
}
func nativeRequest(id string) Request {
	body, _ := json.Marshal(map[string]any{"event_id": id, "event_type": nativeEventTypes[0], "source_scenario": "agent-manager", "delivered_at": "2026-11-07T11:59:59Z", "payload": map[string]any{"secret": "never retained"}})
	return signNative(Request{Operation: "accept_native", Body: body, EventID: id, IdempotencyKey: id})
}
func signNative(r Request) Request {
	m := hmac.New(sha256.New, bytes.Repeat([]byte{7}, 32))
	m.Write(r.Body)
	r.Signature = "sha256=" + hex.EncodeToString(m.Sum(nil))
	return r
}
func noNativeTables(t *testing.T, db *sql.DB) {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name LIKE 'local_work_native_%'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("unexpected durable access count=%d err=%v", n, err)
	}
}

func TestNativeReceiptDisabledHasNoDependencies(t *testing.T) {
	o, err := NewNativeReceiptOwner(NativeReceiptConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.Accept(nil, Request{}); err != ErrDisabled {
		t.Fatal(err)
	}
	if _, err = o.Select(nil); err != ErrDisabled {
		t.Fatal(err)
	}
}
func TestNativeReceiptAuthenticatesBeforeStorage(t *testing.T) {
	cases := map[string]func(*Request){
		"signature":    func(r *Request) { r.Signature = "sha256=" + strings.Repeat("0", 64) },
		"wrong_header": func(r *Request) { r.IdempotencyKey = "other" },
		"duplicate": func(r *Request) {
			r.Body = []byte(`{"event_id":"event-1","event_id":"event-1","event_type":"agent_manager.run.notification.v1","source_scenario":"agent-manager","delivered_at":"2026-11-07T11:59:59Z"}`)
			*r = signNative(*r)
		},
		"nested_duplicate": func(r *Request) {
			r.Body = bytes.Replace(r.Body, []byte(`"secret":"never retained"`), []byte(`"secret":1,"secret":2`), 1)
			*r = signNative(*r)
		},
		"unknown_field": func(r *Request) {
			r.Body = bytes.Replace(r.Body, []byte(`"payload":`), []byte(`"worker_key":`), 1)
			*r = signNative(*r)
		},
		"wrong_id": func(r *Request) {
			r.Body = bytes.ReplaceAll(r.Body, []byte("event-1"), []byte("event-2"))
			*r = signNative(*r)
		},
		"wrong_time": func(r *Request) {
			r.Body = bytes.ReplaceAll(r.Body, []byte("2026-11-07T11:59:59Z"), []byte("tomorrow"))
			*r = signNative(*r)
		},
		"trailing": func(r *Request) { r.Body = append(r.Body, []byte(` {}`)...); *r = signNative(*r) },
		"oversize": func(r *Request) { r.Body = bytes.Repeat([]byte("x"), int(MaxBodyBytes)+1); *r = signNative(*r) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			o, _, key := nativeFixture(t, nil)
			r := nativeRequest("event-1")
			mutate(&r)
			if accepted, err := o.Accept(context.Background(), r); err == nil || accepted {
				t.Fatal("accepted invalid receipt")
			}
			noNativeTables(t, o.store.db)
			if key.last != nil && !bytes.Equal(key.last, make([]byte, len(key.last))) {
				t.Fatal("key not cleared")
			}
		})
	}
}
func TestNativeReceiptOutOfScopeNeverCreatesStorage(t *testing.T) {
	o, _, _ := nativeFixture(t, nil)
	r := nativeRequest("event-1")
	r.Body = bytes.ReplaceAll(r.Body, []byte(nativeEventTypes[0]), []byte("agent_manager.other.v1"))
	r = signNative(r)
	if accepted, err := o.Accept(context.Background(), r); err != nil || accepted {
		t.Fatalf("%t %v", accepted, err)
	}
	noNativeTables(t, o.store.db)
}
func TestNativeReceiptDedupeConflictAndKeyCustody(t *testing.T) {
	o, _, key := nativeFixture(t, nil)
	ctx := context.Background()
	r := nativeRequest("event-1")
	for i := 0; i < 2; i++ {
		if ok, err := o.Accept(ctx, r); err != nil || !ok {
			t.Fatalf("accept %v %v", ok, err)
		}
	}
	if key.scope.SourceEpoch != "epoch-1" || !bytes.Equal(key.last, make([]byte, 32)) {
		t.Fatal("key scope/custody")
	}
	r.Body = bytes.ReplaceAll(r.Body, []byte("never retained"), []byte("changed"))
	r = signNative(r)
	if ok, err := o.Accept(ctx, r); err == nil || ok {
		t.Fatal("conflicting duplicate accepted")
	}
	var n int
	o.store.db.QueryRow("SELECT count(*) FROM local_work_native_receipts").Scan(&n)
	if n != 1 {
		t.Fatal(n)
	}
}
func TestNativeSelectionSurvivesRestartExpiryAndLaterReceipt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owner.sqlite")
	open := func() *sql.DB {
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		return db
	}
	db := open()
	o, _, _ := nativeFixture(t, db)
	ctx := context.Background()
	for _, id := range []string{"b", "a"} {
		if _, err := o.Accept(ctx, nativeRequest(id)); err != nil {
			t.Fatal(err)
		}
	}
	first, err := o.Select(ctx)
	if err != nil || first == nil {
		t.Fatalf("select %v", err)
	}
	if !reflect.DeepEqual(first.EventIDs(), []string{"a", "b"}) {
		t.Fatal(first.EventIDs())
	}
	db.Close()
	db = open()
	defer db.Close()
	o, _, _ = nativeFixture(t, db)
	o.clock = func() time.Time { return time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC) }
	if _, err = o.Accept(ctx, nativeRequest("c")); err != nil {
		t.Fatal(err)
	}
	second, err := o.Select(ctx)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("selection renewed after restart/expiry: %#v %v", second, err)
	}
	ids := second.EventIDs()
	ids[0] = "mutated"
	if second.EventIDs()[0] != "a" {
		t.Fatal("mutable selection")
	}
}
func TestNativeSelectionRejectsRecommissioningAndCorruption(t *testing.T) {
	for _, kind := range []string{"epoch", "deadline", "receipt", "json"} {
		t.Run(kind, func(t *testing.T) {
			o, _, _ := nativeFixture(t, nil)
			ctx := context.Background()
			o.Accept(ctx, nativeRequest("a"))
			o.Select(ctx)
			switch kind {
			case "epoch":
				o.scope.SourceEpoch = "epoch-2"
				o.digest = strings.Repeat("d", 64)
			case "deadline":
				o.store.db.Exec("UPDATE local_work_native_selection SET admission_deadline='2026-11-08T00:00:00Z'")
			case "receipt":
				o.store.db.Exec("UPDATE local_work_native_receipts SET body_sha256=?", strings.Repeat("a", 64))
			case "json":
				o.store.db.Exec("UPDATE local_work_native_selection SET receipts_json='[]'")
			}
			if selection, err := o.Select(ctx); err == nil || selection != nil {
				t.Fatal("accepted changed durable identity")
			}
		})
	}
}
func TestNativeSelectionBoundedAndStableAcrossConcurrentOwners(t *testing.T) {
	o, g, _ := nativeFixture(t, nil)
	ctx := context.Background()
	for i := 0; i < 66; i++ {
		if _, err := o.Accept(ctx, nativeRequest(fmt.Sprintf("event-%03d", i))); err != nil {
			t.Fatal(err)
		}
	}
	second, _, key := nativeFixture(t, o.store.db)
	second.guard = g
	second.credentials = key
	var wg sync.WaitGroup
	got := make([]*NativeSelection, 2)
	errs := make([]error, 2)
	for i, owner := range []*NativeReceiptOwner{o, second} {
		wg.Add(1)
		go func(i int, owner *NativeReceiptOwner) { defer wg.Done(); got[i], errs[i] = owner.Select(ctx) }(i, owner)
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil || len(got[0].EventIDs()) != 64 || !reflect.DeepEqual(got[0], got[1]) {
		t.Fatalf("concurrent selection: %v", errs)
	}
}
func TestNativeReceiptRevocationSerializesCommit(t *testing.T) {
	o, g, _ := nativeFixture(t, nil)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	g.enter = func() { close(entered); <-release }
	go func() { _, err := o.Accept(context.Background(), nativeRequest("a")); done <- err }()
	<-entered
	revoked := make(chan struct{})
	go func() { g.mu.Lock(); g.revoked = true; g.mu.Unlock(); close(revoked) }()
	select {
	case <-revoked:
		t.Fatal("revocation crossed transaction lock")
	default:
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	<-revoked
	if ok, err := o.Accept(context.Background(), nativeRequest("b")); ok || err == nil {
		t.Fatal("accepted after revocation")
	}
	if s, err := o.Select(context.Background()); err == nil || s != nil {
		t.Fatal("released selection after revocation")
	}
}
func TestNativeSelectionRequiresMatchingSourceGates(t *testing.T) {
	o, _, _ := nativeFixture(t, nil)
	ctx := context.Background()
	o.Accept(ctx, nativeRequest("event-1"))
	s, err := o.Select(ctx)
	if err != nil {
		t.Fatal(err)
	}
	e, _, _ := sourceFixture(t, SourceEvents)
	c, _, _ := sourceFixture(t, SourceCollector)
	for _, g := range []*SourceGate{e, c} {
		g.policy.Owner = "owner"
		g.policy.Subject = "subject"
		g.policy.SourceEpoch = "epoch-1"
	}
	e.policy.EventIDs = []string{"event-1"}
	if _, err = s.BrokerSelection(e, c); err != nil {
		t.Fatal(err)
	}
	e.policy.EventIDs = []string{"worker-id"}
	if _, err = s.BrokerSelection(e, c); err == nil {
		t.Fatal("worker selection accepted")
	}
	e.policy.EventIDs = []string{"event-1"}
	c.policy.SourceEpoch = "other"
	if _, err = s.BrokerSelection(e, c); err == nil {
		t.Fatal("cross epoch accepted")
	}
}
func TestNativeReceiptCancelledAndUnavailableAuthorityNoStorage(t *testing.T) {
	for _, kind := range []string{"cancelled", "revoked", "key"} {
		t.Run(kind, func(t *testing.T) {
			o, g, key := nativeFixture(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "cancelled":
				cancel()
			case "revoked":
				g.revoked = true
			case "key":
				key.fail = true
			}
			if ok, err := o.Accept(ctx, nativeRequest("a")); ok || err == nil {
				t.Fatal("accepted without authority")
			}
			noNativeTables(t, o.store.db)
		})
	}
}

func TestNativeReceiptScopeIsDefensivelyCopied(t *testing.T) {
	o, _, _ := nativeFixture(t, nil)
	s := o.scopeCopy()
	s.AllowedTypes[0] = "wrong"
	if o.scope.AllowedTypes[0] == "wrong" {
		t.Fatal("mutable scope escaped")
	}
	if _, err := NewNativeReceiptOwner(NativeReceiptConfig{Enabled: true, Scope: NativeReceiptScope{Owner: "owner", Subject: "subject", SourceInstance: "swarminator", SourceEpoch: "uncommissioned", AllowedTypes: nativeEventTypes}, Store: o.store, Guard: o.guard, Credentials: o.credentials}); err == nil {
		t.Fatal("uncommissioned owner accepted")
	}
}

func TestNativeSelectionEmptyAndCapacity(t *testing.T) {
	o, _, _ := nativeFixture(t, nil)
	ctx := context.Background()
	if s, err := o.Select(ctx); s != nil || err != nil {
		t.Fatalf("empty result %v %v", s, err)
	}
	if ok, err := o.Accept(ctx, nativeRequest("kept")); err != nil || !ok {
		t.Fatal(err)
	}
	// Pressure fixture remains test-owned; no automatic prune or eviction.
	_, err := o.store.db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<9999) INSERT INTO local_work_native_receipts SELECT 'capacity-'||i, 'agent_manager.run.notification.v1', 'agent-manager', ?, '2026-11-07T12:00:00Z' FROM n`, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := o.Accept(ctx, nativeRequest("overflow")); err == nil || ok {
		t.Fatal("capacity did not fail closed")
	}
	if ok, err := o.Accept(ctx, nativeRequest("kept")); err != nil || !ok {
		t.Fatal("exact duplicate should still acknowledge at capacity", err)
	}
}

func TestNativeSelectionIndependentSQLiteConnections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owner.sqlite")
	open := func() *sql.DB {
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		t.Cleanup(func() { db.Close() })
		return db
	}
	first, g1, _ := nativeFixture(t, open())
	second, g2, _ := nativeFixture(t, open())
	if g1 == g2 || first.store.db == second.store.db {
		t.Fatal("test must not share guard or connection")
	}
	ctx := context.Background()
	for _, id := range []string{"a", "b"} {
		if _, err := first.Accept(ctx, nativeRequest(id)); err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	selections := make([]*NativeSelection, 2)
	errs := make([]error, 2)
	for i, owner := range []*NativeReceiptOwner{first, second} {
		wg.Add(1)
		go func(i int, owner *NativeReceiptOwner) {
			defer wg.Done()
			<-start
			selections[i], errs[i] = owner.Select(ctx)
		}(i, owner)
	}
	close(start)
	wg.Wait()
	// SQLite may reject a lock contender; it must never acknowledge a different
	// selection. A retry after both completed observes the unique retained row.
	if errs[0] != nil && errs[1] != nil {
		t.Fatalf("no successful contender: %v", errs)
	}
	retained, err := first.Select(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for i, owner := range []*NativeReceiptOwner{first, second} {
		retry, err := owner.Select(ctx)
		if err != nil || !reflect.DeepEqual(retained, retry) {
			t.Fatalf("connection %d retry changed selection: %v", i, err)
		}
		if errs[i] == nil && !reflect.DeepEqual(retained, selections[i]) {
			t.Fatalf("connection %d acknowledged different selection", i)
		}
	}
	var n int
	if err = first.store.db.QueryRow("SELECT count(*) FROM local_work_native_selection").Scan(&n); err != nil || n != 1 {
		t.Fatalf("retained selections %d: %v", n, err)
	}
}

func TestNativeSelectionPinsCanonicalEventIdentityInBroker(t *testing.T) {
	broker, fence, credentials, owners, _, _ := brokerFixture(t)
	native, _, _ := nativeFixture(t, nil)
	// Commission fixture native scope to exactly the already approved gate pair.
	native.scope.Owner = broker.events.policy.Owner
	native.scope.Subject = broker.events.policy.Subject
	native.scope.SourceEpoch = broker.events.policy.SourceEpoch
	scopeJSON, _ := json.Marshal(native.scope)
	sum := sha256.Sum256(scopeJSON)
	native.digest = hex.EncodeToString(sum[:])
	if ok, err := native.Accept(context.Background(), nativeRequest("event-1")); err != nil || !ok {
		t.Fatal(err)
	}
	if ok, err := native.Accept(context.Background(), nativeRequest("event-2")); err != nil || !ok {
		t.Fatal(err)
	}
	selection, err := native.Select(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	selected, err := selection.BrokerSelection(broker.events, broker.collector)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewSourceBroker(SourceBrokerConfig{Enabled: true, LaunchGeneration: "launch-1", Selection: selected, Events: broker.events, Collector: broker.collector, Credentials: credentials, Fence: fence, Owners: owners})
	if err != nil {
		t.Fatal(err)
	}
	// notification and decision_requested are both allowed globally, but this
	// receipt authenticates only decision_requested for the exact selected ID.
	request := []byte(`{"schema":"flywheel.local-work-source-broker.v1","operation":"read_event","event_id":"event-1"}`)
	if result, err := b.Execute(context.Background(), request); err == nil || len(result) != 0 {
		t.Fatal("released canonical row with changed type")
	}
	owners.event = bytes.ReplaceAll(owners.event, []byte("agent_manager.run.notification.v1"), []byte(nativeEventTypes[0]))
	if result, err := b.Execute(context.Background(), request); err != nil || len(result) == 0 {
		t.Fatalf("exact canonical event refused: %s %v", result, err)
	}
	// Construction copies native metadata, so even a trusted caller's later
	// mutation of its config value cannot weaken the broker's retained identity.
	selected.nativeEvents[0].Type = "agent_manager.run.notification.v1"
	if result, err := b.Execute(context.Background(), request); err != nil || len(result) == 0 {
		t.Fatal("broker retained mutable config", err)
	}
}

func TestNativeSelectionRejectsChangedBrokerBinding(t *testing.T) {
	broker, fence, credentials, owners, _, _ := brokerFixture(t)
	selected := SourceBrokerSelection{RequestKey: broker.scope.RequestKey, AdmissionDeadline: broker.scope.AdmissionDeadline, nativeEvents: []SourceEventIdentity{{ID: "event-1", Type: nativeEventTypes[0], Scenario: "agent-manager"}, {ID: "event-2", Type: nativeEventTypes[0], Scenario: "agent-manager"}}, nativeBinding: &nativeSelectionBinding{requestKey: broker.scope.RequestKey, admissionDeadline: broker.scope.AdmissionDeadline, eventsDigest: broker.events.digest, collectorDigest: broker.collector.digest}}
	base := SourceBrokerConfig{Enabled: true, LaunchGeneration: "launch-1", Selection: selected, Events: broker.events, Collector: broker.collector, Credentials: credentials, Fence: fence, Owners: owners}
	if _, err := NewSourceBroker(base); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"key", "deadline", "gate", "missing"} {
		t.Run(kind, func(t *testing.T) {
			c := base
			switch kind {
			case "key":
				c.Selection.RequestKey = "local-work-" + strings.Repeat("f", 64)
			case "deadline":
				c.Selection.AdmissionDeadline = "2026-11-07T12:06:00Z"
			case "gate":
				g := *broker.events
				g.digest = strings.Repeat("f", 64)
				c.Events = &g
			case "missing":
				c.Selection.nativeBinding = nil
			}
			if _, err := NewSourceBroker(c); err == nil {
				t.Fatal("accepted changed native binding")
			}
		})
	}
}
