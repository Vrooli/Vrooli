package database

import (
	"agent-manager/internal/domain"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/eventbus"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func associationFixture(t *testing.T) (*DB, *Repositories, EvidenceAssociationProposal) {
	t.Helper()
	db, done := setupTestDB(t)
	t.Cleanup(done)
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	if _, err := terminalOutbox(t, r, codecAck).Dispatch(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	s := evidenceSelection(t, r, run)
	p := EvidenceAssociationProposal{Schema: "flywheel.native-evidence-association-proposal.v1", AssociationID: "association-fixture-1", OwnerID: "fixture-owner", EnvironmentID: "fixture-environment", WorkID: "fixture-work", WorkVersion: 1, ScopeVersion: "1", ComponentID: "fixture-component", OperationID: "fixture-operation", ExecutorReference: TerminalEvidenceReference{URI: "https://executor.invalid/work/fixture", Version: "1"}, Native: AssociationNativeLocator{Schema: s.Schema, OwnerID: s.OwnerID, RunID: s.RunID, SourceInstance: s.SourceInstance, SourceEpoch: s.SourceEpoch, LifecycleVersion: s.LifecycleVersion, TerminalObservationSHA256: s.TerminalObservationSHA256}, TerminalEventID: s.TerminalEventID, CanonicalEventSHA256: s.CanonicalEventSHA256, Purpose: "evidence-association-only"}
	p.Bridge.SourceInstance = "fixture-events"
	p.Bridge.ScopeEpoch = "fixture-epoch"
	p.Bridge.ScopeDigest = strings.Repeat("a", 64)
	return db, r, p
}
func allowAssociation(context.Context, EvidenceAssociationProposal, string) error { return nil }
func TestEvidenceAssociationPersistenceReplayAndRevocation(t *testing.T) {
	db, r, p := associationFixture(t)
	o, err := newTestAssociationOwner(t.Context(), r, allowAssociation)
	if err != nil {
		t.Fatal(err)
	}
	ack, err := o.Acknowledge(t.Context(), p, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Reconstruct composition to prove ledger ownership, not in-memory cache.
	o, err = newTestAssociationOwner(t.Context(), r, allowAssociation)
	if err != nil {
		t.Fatal(err)
	}
	again, err := o.Acknowledge(t.Context(), p, 0)
	if err != nil || ack != again {
		t.Fatalf("replay %#v %v", again, err)
	}
	if _, err = o.Select(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	conflict := p
	conflict.OperationID = "another-operation"
	if _, err = o.Acknowledge(t.Context(), conflict, 0); err == nil {
		t.Fatal("conflicting ID accepted")
	}
	if err = o.Revoke(t.Context(), p, 0); err == nil {
		t.Fatal("stale revoke accepted")
	}
	if err = o.Revoke(t.Context(), p, 1); err != nil {
		t.Fatal(err)
	}
	if _, err = o.Select(t.Context(), p); err == nil {
		t.Fatal("revoked association released")
	}
	if _, err = o.Acknowledge(t.Context(), p, 0); err == nil {
		t.Fatal("replay revived association")
	}
	var count int
	if err = db.QueryRowContext(t.Context(), `SELECT count(*) FROM run_evidence_associations`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("ledger rows %d %v", count, err)
	}
	if receiptCount(t, db) != 1 {
		t.Fatal("receipt history changed")
	}
}
func TestEvidenceAssociationManyToManyAndScope(t *testing.T) {
	_, r, p := associationFixture(t)
	o, _ := newTestAssociationOwner(t.Context(), r, allowAssociation)
	for _, name := range []string{"first", "second", "third"} {
		p.AssociationID = name
		p.OperationID = name
		if _, err := o.Acknowledge(t.Context(), p, 0); err != nil {
			t.Fatal(err)
		}
	}
	// Same receipt can bind separate Work/component/operation tuples, never aliases.
	p.AssociationID = "fourth"
	p.WorkID = "other-work"
	p.ComponentID = "other-component"
	if _, err := o.Acknowledge(t.Context(), p, 0); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*EvidenceAssociationProposal){func(p *EvidenceAssociationProposal) { p.ComponentID = "" }, func(p *EvidenceAssociationProposal) { p.OperationID = "" }, func(p *EvidenceAssociationProposal) { p.Native.SourceEpoch = "other" }, func(p *EvidenceAssociationProposal) { p.TerminalEventID = strings.Repeat("b", 64) }, func(p *EvidenceAssociationProposal) { p.ExecutorReference.URI = "agent-manager://runs/x" }, func(p *EvidenceAssociationProposal) { p.Native.TerminalObservationSHA256 = strings.Repeat("c", 64) }} {
		bad := p
		mutate(&bad)
		if _, err := o.Acknowledge(t.Context(), bad, 0); err == nil {
			t.Fatal("bad proposal admitted")
		}
	}
}
func TestEvidenceAssociationAuthorityAndRollback(t *testing.T) {
	for _, action := range []string{"acknowledge", "select", "revoke"} {
		t.Run(action, func(t *testing.T) {
			db, r, p := associationFixture(t)
			base, _ := newTestAssociationOwner(t.Context(), r, allowAssociation)
			if action != "acknowledge" {
				if _, err := base.Acknowledge(t.Context(), p, 0); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			denyAt := 5
			if action == "revoke" {
				denyAt = 3
			}
			o, _ := newTestAssociationOwner(t.Context(), r, func(_ context.Context, got EvidenceAssociationProposal, op string) error {
				if got != p || op != action {
					t.Fatal("authority tuple changed")
				}
				calls++
				if calls >= denyAt {
					return errors.New("revoked")
				}
				return nil
			})
			var err error
			if action == "revoke" {
				err = o.Revoke(t.Context(), p, 1)
			} else if action == "select" {
				_, err = o.Select(t.Context(), p)
			} else {
				_, err = o.Acknowledge(t.Context(), p, 0)
			}
			if err == nil {
				t.Fatal("revoked authorization released")
			}
			var count int
			if err = db.QueryRowContext(t.Context(), `SELECT count(*) FROM run_evidence_associations WHERE state='acknowledged'`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			want := 1
			if action == "acknowledge" {
				want = 0
			}
			if count != want {
				t.Fatalf("write escaped rollback: %d", count)
			}
		})
	}
}
func TestEvidenceAssociationCanonicalAndDisabled(t *testing.T) {
	_, r, p := associationFixture(t)
	raw, digest, err := p.canonical()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeEvidenceAssociationProposal(raw)
	if err != nil || decoded != p {
		t.Fatalf("decode %v", err)
	}
	for _, bad := range []string{raw + " {}", strings.Replace(raw, "{", "{\"extra\":1,", 1), strings.Replace(raw, "{", "{\"schema\":\"duplicate\",", 1)} {
		if _, err := DecodeEvidenceAssociationProposal(bad); err == nil {
			t.Fatal("ambiguous proposal decoded")
		}
	}
	if _, err := newTestAssociationOwner(t.Context(), r, nil); err == nil {
		t.Fatal("nil authority enabled")
	}
	if _, err := newTestAssociationOwner(t.Context(), NewRepositories(r.Runs.(*runRepository).db, logrus.New()), allowAssociation); err == nil {
		t.Fatal("production enabled")
	}
	o, _ := newTestAssociationOwner(t.Context(), r, allowAssociation)
	ack, err := o.Acknowledge(t.Context(), p, 0)
	if err != nil {
		t.Fatal(err)
	}
	ackRaw, _ := CanonicalAssociationJSON(ack)
	reader, _ := NewTerminalEvidenceReader(r, allowTerminalEvidence)
	evidence, err := reader.Select(t.Context(), p.selector())
	if err != nil {
		t.Fatal(err)
	}
	evidenceJSON, _ := CanonicalAssociationJSON(evidence)
	var intent string
	if err = r.Runs.(*runRepository).db.QueryRowContext(t.Context(), `SELECT event_intent_json FROM run_terminal_receipts WHERE run_id=? AND lifecycle_version=?`, p.Native.RunID, p.Native.LifecycleVersion).Scan(&intent); err != nil {
		t.Fatal(err)
	}
	_, eventBytes, err := eventbus.RetainedDomainEventEnvelope(intent, p.TerminalEventID)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _ := json.Marshal(map[string]string{"terminal": evidenceJSON, "event_base64": base64.StdEncoding.EncodeToString(eventBytes), "intent": intent, "proposal": raw, "ack": ackRaw, "proposal_sha256": digest})
	t.Logf("BUNDLE=%s", bundle)
	t.Logf("PROPOSAL=%s\nSHA256=%s\nACK=%s", raw, digest, ackRaw)
}

func TestEvidenceAssociationDiskReopenAndSelectedRevocation(t *testing.T) {
	db, r, p := associationFixture(t)
	o, _ := newTestAssociationOwner(t.Context(), r, allowAssociation)
	ack, err := o.Acknowledge(t.Context(), p, 0)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "association.sqlite")
	if _, err = db.ExecContext(t.Context(), `VACUUM INTO ?`, path); err != nil {
		t.Fatal(err)
	}
	reopened, err := sqlx.Connect("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	repos := receiptRepos(t, &DB{DB: reopened, log: logrus.New()})
	fresh, err := newTestAssociationOwner(t.Context(), repos, allowAssociation)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fresh.SelectAssociation(t.Context(), p)
	if err != nil || got.Ack != ack || got.State != "acknowledged" || got.Revision != 1 {
		t.Fatalf("disk read %#v %v", got, err)
	}
	if err = fresh.Revoke(t.Context(), p, 1); err != nil {
		t.Fatal(err)
	}
	got, err = fresh.SelectAssociation(t.Context(), p)
	if err != nil || got.Ack != ack || got.State != "revoked" || got.Revision != 2 {
		t.Fatalf("revoked read %#v %v", got, err)
	}
	// The identical association ID is isolated by independently authorized cloud owner.
	p.OwnerID = "other-fixture-owner"
	if _, err = fresh.Acknowledge(t.Context(), p, 0); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceAssociationIndependentAuthorityAndTamper(t *testing.T) {
	db, r, p := associationFixture(t)
	o, _ := newTestAssociationOwner(t.Context(), r, func(_ context.Context, got EvidenceAssociationProposal, _ string) error {
		if got != p {
			return errors.New("outside exact independently authorized tuple")
		}
		return nil
	})
	for _, change := range []func(*EvidenceAssociationProposal){
		func(q *EvidenceAssociationProposal) { q.OwnerID = "another-owner" }, func(q *EvidenceAssociationProposal) { q.WorkID = "another-work" }, func(q *EvidenceAssociationProposal) { q.ScopeVersion = "another-scope" }, func(q *EvidenceAssociationProposal) { q.ComponentID = "another-component" }, func(q *EvidenceAssociationProposal) { q.OperationID = "another-operation" }, func(q *EvidenceAssociationProposal) { q.Bridge.ScopeEpoch = "another-epoch" }, func(q *EvidenceAssociationProposal) { q.ExecutorReference.Version = "another-version" }, func(q *EvidenceAssociationProposal) { q.Native.RunID = "00000000-0000-4000-8000-000000000001" },
	} {
		q := p
		change(&q)
		if _, err := o.Acknowledge(t.Context(), q, 0); err == nil {
			t.Fatal("authority bypass")
		}
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := o.Acknowledge(cancelled, p, 0); err == nil {
		t.Fatal("cancelled write")
	}
	if _, err := o.Acknowledge(t.Context(), p, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(t.Context(), `DROP TRIGGER run_evidence_associations_immutable`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(t.Context(), `UPDATE run_evidence_associations SET acknowledgment_json='{}' WHERE owner_id=? AND association_id=?`, p.OwnerID, p.AssociationID); err != nil {
		t.Fatal(err)
	}
	if _, err := o.SelectAssociation(t.Context(), p); err == nil {
		t.Fatal("tampered ack released")
	}
}
func TestEvidenceAssociationConcurrentConflictingIDs(t *testing.T) {
	db, r, p := associationFixture(t)
	o, _ := newTestAssociationOwner(t.Context(), r, allowAssociation)
	q := p
	q.OperationID = "conflicting-operation"
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, proposal := range []EvidenceAssociationProposal{p, q} {
		go func(v EvidenceAssociationProposal) {
			<-start
			_, err := o.Acknowledge(t.Context(), v, 0)
			results <- err
		}(proposal)
	}
	close(start)
	first, second := <-results, <-results
	if first == nil && second == nil {
		t.Fatal("both conflicting proposals acknowledged")
	}
	// SQLite may return a lock conflict instead of choosing a winner. A bounded
	// caller retry can choose a winner, but never mutate the losing proposal ID.
	if first != nil && second != nil {
		if _, err := o.Acknowledge(t.Context(), p, 0); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := db.QueryRowContext(t.Context(), `SELECT count(*) FROM run_evidence_associations WHERE owner_id=? AND association_id=?`, p.OwnerID, p.AssociationID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("conflict rows %d %v", n, err)
	}
}

// This optional fixture reuses an actual immutable native receipt and retained
// Events acknowledgment. Replaying a supplied fixture never rewrites history.
func TestEvidenceAssociationCrossOwnerFixture(t *testing.T) {
	dir := os.Getenv("VROOLI_ASSOCIATION_FIXTURE_DIR")
	if dir == "" {
		t.Skip("explicit isolated fixture not requested")
	}
	db := outboxOpen(t, filepath.Join(dir, "agent-manager.db"))
	defer db.Close()
	r := receiptRepos(t, db)
	raw, err := os.ReadFile(filepath.Join(dir, "intent.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture terminalOwnerFixture
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	s := evidenceSelection(t, r, &domain.Run{ID: fixture.RunID, LifecycleVersion: fixture.LifecycleVersion})
	_, _, p := associationFixture(t)
	p.Native = AssociationNativeLocator{Schema: s.Schema, OwnerID: s.OwnerID, RunID: s.RunID, SourceInstance: s.SourceInstance, SourceEpoch: s.SourceEpoch, LifecycleVersion: s.LifecycleVersion, TerminalObservationSHA256: s.TerminalObservationSHA256}
	p.TerminalEventID = s.TerminalEventID
	p.CanonicalEventSHA256 = s.CanonicalEventSHA256
	p.AssociationID = "association-cross-owner-1"
	p.Bridge.SourceInstance = "fixture-native-events-epoch"
	p.Bridge.ScopeEpoch = "fixture-scope-001"
	p.Bridge.ScopeDigest = strings.Repeat("c", 64)
	o, err := newTestAssociationOwner(t.Context(), r, func(_ context.Context, got EvidenceAssociationProposal, _ string) error {
		if got != p {
			return errors.New("fixture tuple denied")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	ack, err := o.Acknowledge(t.Context(), p, 0)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := o.SelectAssociation(t.Context(), p)
	if err != nil {
		t.Fatal(err)
	}
	reader, _ := NewTerminalEvidenceReader(r, allowTerminalEvidence)
	terminal, err := reader.Select(t.Context(), s)
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]any{"proposal.json": p, "association-ack.json": ack, "selected-association.json": selected, "selected-terminal-evidence.json": terminal} {
		raw, err := CanonicalAssociationJSON(value)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, canonical, err := eventbus.RetainedDomainEventEnvelope(fixture.EventIntentJSON, fixture.EventIntentID)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "terminal-canonical.pb"), canonical, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "terminal-intent.json"), []byte(fixture.EventIntentJSON), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceAssociationStorageHistoryGuards(t *testing.T) {
	db, r, p := associationFixture(t)
	o, _ := newTestAssociationOwner(t.Context(), r, allowAssociation)
	if _, err := o.Acknowledge(t.Context(), p, 0); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{`UPDATE run_evidence_associations SET proposal_json='{}'`, `UPDATE run_evidence_associations SET acknowledgment_json='{}'`, `DELETE FROM run_evidence_associations`, `UPDATE run_evidence_associations SET revision=2`} {
		if _, err := db.ExecContext(t.Context(), statement); err == nil {
			t.Fatalf("history guard allowed %s", statement)
		}
	}
	if err := o.Revoke(t.Context(), p, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(t.Context(), `UPDATE run_evidence_associations SET revision=1,state='acknowledged'`); err == nil {
		t.Fatal("revocation reversed")
	}
	if _, err := db.ExecContext(t.Context(), `INSERT OR REPLACE INTO run_evidence_associations SELECT owner_id,association_id,proposal_json,proposal_sha256,acknowledgment_json,1,'acknowledged' FROM run_evidence_associations`); err == nil {
		t.Fatal("replacement reversed revocation")
	}
}

// Test-only authority gate. Never available to production composition.
type fixtureAssociationGate struct {
	mu    sync.Mutex
	check func(context.Context, EvidenceAssociationProposal, string) error
}

func (g *fixtureAssociationGate) WithAuthority(ctx context.Context, p EvidenceAssociationProposal, action string, fn func(func() error) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fn(func() error { return g.check(ctx, p, action) })
}
func newTestAssociationOwner(ctx context.Context, r *Repositories, check func(context.Context, EvidenceAssociationProposal, string) error) (*EvidenceAssociationOwner, error) {
	if check == nil {
		return NewEvidenceAssociationOwner(ctx, r, nil)
	}
	return NewEvidenceAssociationOwner(ctx, r, &fixtureAssociationGate{check: check})
}

// A real revocation owner shares the same mutex as the entire local transaction.
type revocableAssociationGate struct {
	mu      sync.Mutex
	revoked bool
	entered chan struct{}
	proceed chan struct{}
}

func (g *revocableAssociationGate) WithAuthority(_ context.Context, _ EvidenceAssociationProposal, _ string, fn func(func() error) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.revoked {
		return errors.New("authority revoked")
	}
	if g.entered != nil {
		close(g.entered)
		<-g.proceed
	}
	return fn(func() error {
		if g.revoked {
			return errors.New("authority revoked")
		}
		return nil
	})
}
func TestEvidenceAssociationSerializedAuthorityRevocation(t *testing.T) {
	t.Run("revoked-before-gate", func(t *testing.T) {
		db, r, p := associationFixture(t)
		g := &revocableAssociationGate{revoked: true}
		o, err := NewEvidenceAssociationOwner(t.Context(), r, g)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = o.Acknowledge(t.Context(), p, 0); err == nil {
			t.Fatal("revoked authority committed")
		}
		var n int
		if err = db.QueryRowContext(t.Context(), `SELECT count(*) FROM run_evidence_associations`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("unauthorized rows %d %v", n, err)
		}
	})
	t.Run("revocation-queued-during-commit", func(t *testing.T) {
		db, r, p := associationFixture(t)
		g := &revocableAssociationGate{entered: make(chan struct{}), proceed: make(chan struct{})}
		o, err := NewEvidenceAssociationOwner(t.Context(), r, g)
		if err != nil {
			t.Fatal(err)
		}
		committed := make(chan error, 1)
		go func() { _, err := o.Acknowledge(t.Context(), p, 0); committed <- err }()
		<-g.entered
		queued := make(chan struct{})
		revoked := make(chan error, 1)
		go func() {
			close(queued)
			g.mu.Lock()
			defer g.mu.Unlock()
			g.revoked = true
			var n int
			err := db.QueryRowContext(t.Context(), `SELECT count(*) FROM run_evidence_associations WHERE state='acknowledged'`).Scan(&n)
			if err == nil && n != 1 {
				err = errors.New("revocation acquired authority before local commit")
			}
			revoked <- err
		}()
		<-queued
		close(g.proceed)
		if err := <-committed; err != nil {
			t.Fatal(err)
		}
		if err := <-revoked; err != nil {
			t.Fatal(err)
		}
		if _, err := o.SelectAssociation(t.Context(), p); err == nil {
			t.Fatal("revoked read released")
		}
	})
}
func TestAssociationCanonicalJSONMatchesJavaScriptEscaping(t *testing.T) {
	got, err := CanonicalAssociationJSON(map[string]any{"z": "<&>\\\"", "a": "https://executor.invalid/path?a=1&b=<tag>"})
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"a":"https://executor.invalid/path?a=1&b=<tag>","z":"<&>\\\""}`
	if got != want {
		t.Fatalf("JSON.stringify mismatch: %s", got)
	}
	_, _, p := associationFixture(t)
	p.ExecutorReference.URI = "https://executor.invalid/path?a=1&b=<tag>"
	raw, _, err := p.canonical()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, `\u0026`) || strings.Contains(raw, `\u003c`) {
		t.Fatal("HTML escaping changed proposal digest")
	}
}
