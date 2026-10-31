package investigationpolicy

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func testPolicyRepository(t *testing.T) *Repository {
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	return NewRepository(db, schedule.System())
}

func TestPolicyAndIncidentPersistenceIsIdempotent(t *testing.T) {
	repo := testPolicyRepository(t)
	policy := testPolicy(ModeShadow)
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.ActivePolicy(context.Background())
	if err != nil || loaded.Version != policy.Version {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	decision, err := policy.Evaluate(Observation{ExecutionID: "exec-1", PhaseID: "phase-1", PhaseGeneration: "1", RuleVersion: policy.Version, Now: now, TerminalMismatch: true})
	if err != nil || !decision.Eligible {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	observation := Observation{ExecutionID: "exec-1", PhaseID: "phase-1", PhaseGeneration: "1", RuleVersion: policy.Version, Now: now, TerminalMismatch: true}
	first, reused, err := repo.RecordIncident(context.Background(), observation, decision)
	if err != nil || reused || first == nil {
		t.Fatalf("first=%+v reused=%v err=%v", first, reused, err)
	}
	second, reused, err := repo.RecordIncident(context.Background(), observation, decision)
	if err != nil || !reused || second.ID != first.ID {
		t.Fatalf("second=%+v reused=%v err=%v", second, reused, err)
	}
	if second.OccurrenceCount != 1 || len(second.Occurrences) != 1 {
		t.Fatalf("same observation should be one occurrence: count=%d occurrences=%d", second.OccurrenceCount, len(second.Occurrences))
	}
	newObservation := observation
	newObservation.Now = now.Add(time.Minute)
	third, reused, err := repo.RecordIncident(context.Background(), newObservation, decision)
	if err != nil || !reused || third.ID != first.ID || third.OccurrenceCount != 2 || len(third.Occurrences) != 2 {
		t.Fatalf("new observation should join incident history: incident=%+v reused=%v err=%v", third, reused, err)
	}
	incidents, err := repo.ListIncidents(context.Background(), "exec-1", "", "", 10)
	if err != nil || len(incidents) != 1 || incidents[0].OccurrenceCount != 2 {
		t.Fatalf("list=%+v err=%v", incidents, err)
	}
	if err := repo.LinkInvestigation(context.Background(), first.IncidentFingerprint, "inv-1"); err != nil {
		t.Fatal(err)
	}
	loadedIncident, found, err := repo.GetIncident(context.Background(), first.IncidentFingerprint)
	if err != nil || !found || loadedIncident.InvestigationID != "inv-1" || loadedIncident.State != "dispatched" {
		t.Fatalf("incident=%+v found=%v err=%v", loadedIncident, found, err)
	}
	_ = schedule.System // retain explicit schedule dependency in the test contract
}

func TestPolicyRevisionConflictPreservesActiveRevision(t *testing.T) {
	repo := testPolicyRepository(t)
	initial := testPolicy(ModeShadow)
	if err := repo.PutPolicy(context.Background(), initial, true); err != nil {
		t.Fatal(err)
	}
	changedSameVersion := initial
	changedSameVersion.Mode = ModeAutomatic
	if err := repo.PutPolicy(context.Background(), changedSameVersion, true); !errors.Is(err, ErrPolicyConflict) {
		t.Fatalf("same-version rewrite err=%v, want ErrPolicyConflict", err)
	}
	next := initial
	next.Version = "next.v1"
	if err := repo.PutPolicyIfVersion(context.Background(), next, true, "stale.v1"); !errors.Is(err, ErrPolicyConflict) {
		t.Fatalf("stale active edit err=%v, want ErrPolicyConflict", err)
	}
	active, err := repo.ActivePolicy(context.Background())
	if err != nil || active.Version != initial.Version || active.Mode != initial.Mode {
		t.Fatalf("active=%+v err=%v, want original policy", active, err)
	}
}

func TestScopedPolicyResolutionUsesPhaseExecutionFamilyPrecedence(t *testing.T) {
	repo := testPolicyRepository(t)
	ctx := context.Background()
	global := testPolicy(ModeShadow)
	if err := repo.PutPolicy(ctx, global, true); err != nil {
		t.Fatal(err)
	}
	family := global
	family.Version = "family.v1"
	family.Mode = ModeAutomatic
	family.Scope = PolicyScope{FamilyID: "family-1"}
	if err := repo.PutPolicy(ctx, family, true); err != nil {
		t.Fatal(err)
	}
	execution := global
	execution.Version = "execution.v1"
	execution.Scope = PolicyScope{ExecutionID: "exec-1"}
	if err := repo.PutPolicy(ctx, execution, true); err != nil {
		t.Fatal(err)
	}
	phase := global
	phase.Version = "phase.v1"
	phase.Scope = PolicyScope{PhaseID: "phase-1"}
	if err := repo.PutPolicy(ctx, phase, true); err != nil {
		t.Fatal(err)
	}
	resolved, source, err := repo.ResolvePolicy(ctx, PolicyScope{FamilyID: "family-1", ExecutionID: "exec-1", PhaseID: "phase-1"})
	if err != nil || source != "phase:phase-1" || resolved.Version != phase.Version {
		t.Fatalf("resolved=%+v source=%q err=%v", resolved, source, err)
	}
	if err := repo.PutPolicyIfVersion(ctx, phase, false, phase.Version); err != nil {
		t.Fatal(err)
	}
	resolved, source, err = repo.ResolvePolicy(ctx, PolicyScope{FamilyID: "family-1", ExecutionID: "exec-1", PhaseID: "phase-1"})
	if err != nil || source != "execution:exec-1" || resolved.Version != execution.Version {
		t.Fatalf("after disable resolved=%+v source=%q err=%v", resolved, source, err)
	}
}

func TestFamilyCorrelationCoalescesOnlyProvenCommonFailures(t *testing.T) {
	repo := testPolicyRepository(t)
	policy := testPolicy(ModeShadow)
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 6, 15, 0, 0, 0, time.UTC)
	makeObservation := func(executionID, sharedFailureRef string) Observation {
		return Observation{
			ExecutionID: executionID, PhaseID: "phase-1", PhaseGeneration: "1", FamilyID: "family-1",
			SharedFailureRef: sharedFailureRef, RuleVersion: policy.Version, Now: now, TerminalMismatch: true,
		}
	}
	firstObservation := makeObservation("child-1", "receipt://producer/42")
	secondObservation := makeObservation("child-2", "receipt://producer/42")
	firstDecision, err := policy.Evaluate(firstObservation)
	if err != nil {
		t.Fatal(err)
	}
	secondDecision, err := policy.Evaluate(secondObservation)
	if err != nil {
		t.Fatal(err)
	}
	if firstDecision.IncidentFingerprint != secondDecision.IncidentFingerprint {
		t.Fatalf("proven shared failure should coalesce: first=%s second=%s", firstDecision.IncidentFingerprint, secondDecision.IncidentFingerprint)
	}
	first, reused, err := repo.RecordIncident(context.Background(), firstObservation, firstDecision)
	if err != nil || reused || first == nil {
		t.Fatalf("first incident=%+v reused=%v err=%v", first, reused, err)
	}
	second, reused, err := repo.RecordIncident(context.Background(), secondObservation, secondDecision)
	if err != nil || !reused || second.ID != first.ID {
		t.Fatalf("shared child should reuse incident: second=%+v reused=%v err=%v", second, reused, err)
	}
	if len(second.SubjectExecutionIDs) != 2 || second.SubjectExecutionIDs[0] != "child-1" || second.SubjectExecutionIDs[1] != "child-2" {
		t.Fatalf("shared incident lost subject relations: %+v", second.SubjectExecutionIDs)
	}
	independentObservation := makeObservation("child-3", "receipt://producer/99")
	independentDecision, err := policy.Evaluate(independentObservation)
	if err != nil {
		t.Fatal(err)
	}
	if independentDecision.IncidentFingerprint == firstDecision.IncidentFingerprint {
		t.Fatal("unrelated producer failure must remain a separate incident")
	}
	independent, reused, err := repo.RecordIncident(context.Background(), independentObservation, independentDecision)
	if err != nil || reused || independent == nil || independent.ID == first.ID {
		t.Fatalf("independent incident=%+v reused=%v err=%v", independent, reused, err)
	}
	filtered, err := repo.ListIncidents(context.Background(), "child-2", "", "", 10)
	if err != nil || len(filtered) != 1 || filtered[0].ID != first.ID {
		t.Fatalf("subject relation filter=%+v err=%v", filtered, err)
	}
	familyFiltered, err := repo.ListIncidents(context.Background(), "", "family-1", "", 10)
	if err != nil || len(familyFiltered) != 2 {
		t.Fatalf("family filter=%+v err=%v", familyFiltered, err)
	}
}

func TestDispatchClaimHasOneDurableWinnerAndFailureCanRetry(t *testing.T) {
	repo := testPolicyRepository(t)
	policy := testPolicy(ModeAutomatic)
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 6, 16, 0, 0, 0, time.UTC)
	observation := Observation{ExecutionID: "exec-claim", PhaseID: "phase-1", PhaseGeneration: "1", RuleVersion: policy.Version, Now: now, TerminalMismatch: true}
	decision, err := policy.Evaluate(observation)
	if err != nil {
		t.Fatal(err)
	}
	incident, reused, err := repo.RecordIncident(context.Background(), observation, decision)
	if err != nil || reused || incident == nil {
		t.Fatalf("incident=%+v reused=%v err=%v", incident, reused, err)
	}
	first, err := repo.ClaimDispatch(context.Background(), incident.IncidentFingerprint, "caller-1")
	if err != nil || !first {
		t.Fatalf("first claim=%v err=%v", first, err)
	}
	second, err := repo.ClaimDispatch(context.Background(), incident.IncidentFingerprint, "caller-2")
	if err != nil || second {
		t.Fatalf("second claim=%v err=%v, want one winner", second, err)
	}
	claimed, found, err := repo.GetIncident(context.Background(), incident.IncidentFingerprint)
	if err != nil || !found || claimed.State != "dispatching" || claimed.DispatchClaimKey != "caller-1" {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	if err := repo.MarkDispatchFailed(context.Background(), incident.IncidentFingerprint, "provider unavailable"); err != nil {
		t.Fatal(err)
	}
	retry, err := repo.ClaimDispatch(context.Background(), incident.IncidentFingerprint, "caller-retry")
	if err != nil || !retry {
		t.Fatalf("retry claim=%v err=%v", retry, err)
	}
}

func TestStaleDispatchClaimCanBeRecoveredAfterOwnerRestart(t *testing.T) {
	repo := testPolicyRepository(t)
	policy := testPolicy(ModeAutomatic)
	if err := repo.PutPolicy(context.Background(), policy, true); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 6, 16, 0, 0, 0, time.UTC)
	observation := Observation{ExecutionID: "exec-crash-boundary", PhaseID: "phase-1", PhaseGeneration: "1", RuleVersion: policy.Version, Now: now, TerminalMismatch: true}
	decision, err := policy.Evaluate(observation)
	if err != nil {
		t.Fatal(err)
	}
	incident, reused, err := repo.RecordIncident(context.Background(), observation, decision)
	if err != nil || reused || incident == nil {
		t.Fatalf("incident=%+v reused=%v err=%v", incident, reused, err)
	}
	claimed, err := repo.ClaimDispatch(context.Background(), incident.IncidentFingerprint, "crashed-owner-claim")
	if err != nil || !claimed {
		t.Fatalf("initial claim=%v err=%v", claimed, err)
	}
	if recovered, err := repo.RecoverStaleDispatch(context.Background(), incident.IncidentFingerprint); err != nil || recovered {
		t.Fatalf("fresh claim recovered=%v err=%v, want active owner protection", recovered, err)
	}
	// Simulate the process dying after the claim commit but before the
	// external Program Runtime acknowledgement is persisted.
	if _, err := repo.db.ExecContext(context.Background(), `UPDATE plan_investigation_incidents SET dispatch_started_at=? WHERE incident_fingerprint=?`, formatTime(repo.clock.Now().UTC().Add(-dispatchClaimLease-time.Second)), incident.IncidentFingerprint); err != nil {
		t.Fatal(err)
	}
	recovered, err := repo.RecoverStaleDispatch(context.Background(), incident.IncidentFingerprint)
	if err != nil || !recovered {
		t.Fatalf("recovered=%v err=%v, want stale claim recovery", recovered, err)
	}
	loaded, found, err := repo.GetIncident(context.Background(), incident.IncidentFingerprint)
	if err != nil || !found || loaded.State != "dispatch_failed" || loaded.DispatchClaimKey != "" || loaded.DispatchStartedAt != "" {
		t.Fatalf("loaded=%+v found=%v err=%v, want retryable cleared claim", loaded, found, err)
	}
	reclaimed, err := repo.ClaimDispatch(context.Background(), incident.IncidentFingerprint, "restarted-owner-claim")
	if err != nil || !reclaimed {
		t.Fatalf("reclaimed=%v err=%v, want restarted owner to claim once", reclaimed, err)
	}
}

func TestEnsureMigrationsClosesMetadataRowsBeforeAddingColumns(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE plan_investigation_incidents (incident_id TEXT PRIMARY KEY, execution_id TEXT NOT NULL, phase_id TEXT NOT NULL, phase_generation TEXT NOT NULL, policy_version TEXT NOT NULL, incident_fingerprint TEXT NOT NULL UNIQUE, mode TEXT NOT NULL, state TEXT NOT NULL, decision_json TEXT NOT NULL, investigation_id TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
		CREATE TABLE plan_investigation_occurrences (occurrence_id TEXT PRIMARY KEY, incident_fingerprint TEXT NOT NULL, occurrence_key TEXT NOT NULL UNIQUE, decision_json TEXT NOT NULL, observed_at TEXT NOT NULL, created_at TEXT NOT NULL);
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var familyID, claimKey, startedAt, sharedFailureRef string
	if err := db.QueryRow(`SELECT family_id,dispatch_claim_key,dispatch_started_at FROM plan_investigation_incidents LIMIT 1`).Scan(&familyID, &claimKey, &startedAt); err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("incident migration columns unavailable: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO plan_investigation_occurrences (occurrence_id,incident_fingerprint,occurrence_key,decision_json,observed_at,created_at,family_id,shared_failure_ref) VALUES ('o','i','k','{}','2026-09-06T00:00:00Z','2026-09-06T00:00:00Z','','receipt://none')`); err != nil {
		t.Fatalf("occurrence migration columns unavailable: %v", err)
	}
	if err := db.QueryRow(`SELECT shared_failure_ref FROM plan_investigation_occurrences WHERE occurrence_id='o'`).Scan(&sharedFailureRef); err != nil || sharedFailureRef != "receipt://none" {
		t.Fatalf("shared failure ref=%q err=%v", sharedFailureRef, err)
	}
}
