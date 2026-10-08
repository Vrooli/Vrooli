package supervision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
)

func effortFixture(t *testing.T) (*EffortService, *Repository, *fakeActionController) {
	t.Helper()
	r, _ := testRepository(t)
	c := &fakeActionController{runs: map[uuid.UUID]*domain.Run{}}
	s := NewEffortService(r, c, EffortDiscoveryConfig{Root: t.TempDir(), ScanLimit: 2})
	s.now = r.now
	return s, r, c
}

func TestEffortRegistryRetainsReplacementCoordinatorAsRuntimeEvidence(t *testing.T) {
	s, _, c := effortFixture(t)
	ctx := context.Background()
	e := enrollmentFixture(t, s, c, domain.RunStatusRunning)
	replacement := uuid.New()
	c.runs[replacement] = &domain.Run{ID: replacement, Status: domain.RunStatusRunning, WorkReferences: []*eventpb.WorkReference{{
		Kind: "effort", Id: e.EffortRef, Revision: e.TargetRevision, Relationship: "orchestrator", Verified: true,
		Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
		State:      eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
	}}}
	s.SetRunRegistry(effortRegistryFixture{runs: []*domain.Run{c.runs[replacement]}})
	if _, err := s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	current, _, err := s.repo.GetEffort(ctx, e.EffortRef)
	if err != nil {
		t.Fatal(err)
	}
	if current.AuthorizedBy == "" || current.AuthorityRef != e.AuthorityRef {
		t.Fatalf("coordinator replacement dropped the verified owner: %+v", current)
	}
	if len(current.Subjects) != 2 || current.Subjects[1].RunId != replacement.String() {
		t.Fatalf("replacement coordinator was not retained as runtime evidence: %+v", current.Subjects)
	}
}

func TestEffortWorkspaceEvidenceTimestampAndDerivedJoinsStayStable(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	workspaceFixture(t, s.config.Root, "arbitrary", "effort:arbitrary", map[string]any{"target_revision": "", "execution": map[string]any{"approved_source_digest": "accepted-cut"}})
	if _, err := s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	e, ob, err := r.GetEffort(ctx, "effort:arbitrary")
	if err != nil || e.TargetRevision != "accepted-cut" {
		t.Fatal(e, err)
	}
	stamp := ob.ObservedAt.AsTime()
	s.now = func() time.Time { return stamp.Add(10 * time.Minute) }
	b, err := s.Board(ctx, nil)
	if err != nil || b.Rows[0].Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_STALE || !b.Rows[0].ObservedAt.AsTime().Equal(stamp) {
		t.Fatal("stale source stamped just now", b, err)
	}
	id := uuid.New()
	run := &domain.Run{ID: id, Status: domain.RunStatusRunning, WorkReferences: []*eventpb.WorkReference{{Kind: "effort", Id: e.EffortRef, Relationship: "supervisor", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}}}
	c.runs[id] = run
	s.SetRunRegistry(effortRegistryFixture{runs: []*domain.Run{run}})
	if _, err = s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	joined, _, _ := r.GetEffort(ctx, e.EffortRef)
	if _, err = s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	next, _, _ := r.GetEffort(ctx, e.EffortRef)
	if len(next.Subjects) != 1 || next.Revision != joined.Revision {
		t.Fatal("unchanged workspace erased/churned derived run join", next)
	}
}

func TestEffortContradictoryRunStateKeepsAccountingUnknown(t *testing.T) {
	s, _, c := effortFixture(t)
	e := enrollmentFixture(t, s, c, domain.RunStatusRunning)
	start := s.now().Add(-time.Hour)
	ended := s.now().Add(-time.Minute)
	heartbeat := s.now()
	run := c.runs[uuid.MustParse(e.Subjects[0].RunId)]
	run.StartedAt = &start
	run.EndedAt = &ended
	run.LastHeartbeat = &heartbeat
	run.ErrorMsg = "runner exited before terminal event"
	run.Summary = &domain.RunSummary{TokensUsed: 396462006}
	b, err := s.Board(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	a := b.Rows[0].Assignments[0]
	if a.RuntimeState != "unknown" || a.Usage.AgentSeconds != nil || a.Usage.Tokens != nil || a.UnavailableReason == "" || len(b.Rows[0].Blockers) == 0 {
		t.Fatal("contradictory lifecycle became settled state/accounting", a)
	}
	if c.continued != 0 || c.stopped != 0 {
		t.Fatal("observation repaired or replaced run")
	}
}

type effortRegistryFixture struct{ runs []*domain.Run }

func (f effortRegistryFixture) List(_ context.Context, filter repository.RunListFilter) ([]*domain.Run, error) {
	if filter.Offset >= len(f.runs) {
		return nil, nil
	}
	end := filter.Offset + filter.Limit
	if end > len(f.runs) {
		end = len(f.runs)
	}
	return f.runs[filter.Offset:end], nil
}

func TestEffortActualScanClearsNotScannedButRetainsCurrentFailure(t *testing.T) {
	s, _, _ := effortFixture(t)
	workspaceFixture(t, s.config.Root, "good", "effort:good", nil)
	workspaceFixture(t, s.config.Root, "bad", "effort:bad", map[string]any{"schema_version": 99})
	d, err := s.ReconcileDiscovery(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !d.Partial || d.LastScanAt == nil {
		t.Fatal(d)
	}
	for _, f := range d.Findings {
		if f.Code == "not_scanned" {
			t.Fatal("bootstrap finding persisted after scan", d)
		}
	}
}

func workspaceFixture(t *testing.T, root, name, ref string, changes map[string]any) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	m := map[string]any{"schema_version": 1, "slug": name, "repository": "repo:test", "effort_ref": ref, "destination_ref": "doc:accepted-target", "target_revision": "accepted-1", "work_shape": "bounded_task", "owners": map[string]any{}}
	for k, v := range changes {
		m[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "effort.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func enrollmentFixture(t *testing.T, s *EffortService, c *fakeActionController, status domain.RunStatus) *pb.EffortEnrollment {
	t.Helper()
	id := uuid.New()
	c.runs[id] = &domain.Run{ID: id, Status: status}
	e := &pb.EffortEnrollment{EffortRef: "effort:" + uuid.NewString(), DisplayName: "Arbitrary bounded effort", DestinationRef: "doc:target", TargetRevision: "accepted-1", AuthorityRef: "grant:operator-accepted", SupervisorRunId: uuid.NewString(), Subjects: []*pb.EffortSubject{{Owner: "agent-manager", Kind: "run", Reference: id.String(), RunId: id.String(), Role: "orchestrator"}}}
	supervisorID := uuid.MustParse(e.SupervisorRunId)
	c.runs[supervisorID] = &domain.Run{ID: supervisorID, Status: status}
	got, err := s.Enroll(context.Background(), &pb.EnrollEffortRequest{Enrollment: e, IdempotencyKey: uuid.NewString()}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestEffortDiscoveryRotatesAcrossRestartAndSuppressesUnchangedCuts(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	for _, name := range []string{"alpha", "beta", "gamma", "zeta"} {
		workspaceFixture(t, s.config.Root, name, "effort:"+name, nil)
	}
	first, err := s.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.ScannedCount != 2 || first.ScanCursor != "beta" || !first.Partial {
		t.Fatalf("bounded page=%v", first)
	}
	restart := NewEffortService(r, c, s.config)
	restart.now = s.now
	second, err := restart.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	list, _ := restart.List(ctx, nil)
	if len(list.Efforts) != 4 || second.ScanCursor != "zeta" {
		t.Fatalf("restart starved later sources: %v %v", second, list)
	}
	third, err := restart.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fourth, err := restart.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if third.Generation != second.Generation || fourth.Generation != second.Generation || third.ChangeIdentity != fourth.ChangeIdentity {
		t.Fatalf("unchanged rotations changed generation: %v %v %v", second, third, fourth)
	}
	workspaceFixture(t, s.config.Root, "omega", "effort:new-shape", map[string]any{"work_shape": "investigation"})
	for i := 0; i < 3; i++ {
		if _, err = restart.ReconcileDiscovery(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err = r.GetEffort(ctx, "effort:new-shape"); err != nil {
		t.Fatal("new arbitrary effort was not discovered", err)
	}
}

func TestEffortDiscoveryPathSchemaConflictAndRemoval(t *testing.T) {
	s, r, _ := effortFixture(t)
	s.config.ScanLimit = 100
	ctx := context.Background()
	workspaceFixture(t, s.config.Root, "first", "effort:stable", nil)
	if _, err := s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(s.config.Root, "first"), filepath.Join(s.config.Root, "moved")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	e, _, _ := r.GetEffort(ctx, "effort:stable")
	if e.Workspace != "moved" {
		t.Fatal("move changed identity or failed", e)
	}
	workspaceFixture(t, s.config.Root, "duplicate", "effort:stable", nil)
	report, err := s.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, o, _ := r.GetEffort(ctx, e.EffortRef)
	if !report.Partial || o.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE {
		t.Fatal("conflict not visible", report, o)
	}
	workspaceFixture(t, s.config.Root, "bad-version", "effort:bad", map[string]any{"schema_version": 99})
	workspaceFixture(t, s.config.Root, "unsafe", "effort:unsafe", map[string]any{"checkpoint": "../../credential.json"})
	if err = os.Symlink(t.TempDir(), filepath.Join(s.config.Root, "symlink")); err != nil {
		t.Fatal(err)
	}
	report, err = s.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) < 3 {
		t.Fatal("unsafe/incompatible sources accepted", report)
	}
	root, err := openSafeRoot(s.config.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err = safeRead(root, "../credentials", 100); err == nil {
		t.Fatal("path escape accepted")
	}
}

func TestEffortDiscoveryMissingAndMalformedNeverCompletes(t *testing.T) {
	s, r, _ := effortFixture(t)
	ctx := context.Background()
	workspaceFixture(t, s.config.Root, "one", "effort:one", nil)
	s.ReconcileDiscovery(ctx)
	if err := os.WriteFile(filepath.Join(s.config.Root, "one", "effort.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	s.ReconcileDiscovery(ctx)
	e, o, err := r.GetEffort(ctx, "effort:one")
	if err != nil || e.Withdrawn || o.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE {
		t.Fatalf("malformed erased enrollment: %v %v %v", e, o, err)
	}
	if err = os.Rename(filepath.Join(s.config.Root, "one"), filepath.Join(t.TempDir(), "retained")); err != nil {
		t.Fatal(err)
	}
	s.ReconcileDiscovery(ctx)
	board, err := s.Board(ctx, nil)
	if err != nil || len(board.Rows) != 1 || board.Rows[0].OutcomeStanding.State == "accepted" {
		t.Fatal("removal inferred successful completion", board, err)
	}
}

func TestEffortDiscoveryRetiresFindingsForRemovedWorkspaces(t *testing.T) {
	s, r, _ := effortFixture(t)
	ctx := context.Background()
	workspaceFixture(t, s.config.Root, "present", "effort:present", nil)
	if _, err := s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	d, err := r.GetEffortDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d.Findings = append(d.Findings, &pb.EffortDiscoveryFinding{Source: "removed-workspace", Code: "invalid_manifest", Reason: "stale"})
	if err := r.SaveEffortDiscovery(ctx, d); err != nil {
		t.Fatal(err)
	}
	refreshed, err := s.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range refreshed.Findings {
		if finding.Source == "removed-workspace" {
			t.Fatalf("stale removed-workspace finding retained: %v", finding)
		}
	}
}

func TestEffortBoardReadOnlyUnknownUsageAndDistinctOutcome(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	workspaceFixture(t, s.config.Root, "one", "effort:one", nil)
	board, err := s.Board(ctx, nil)
	if err != nil || len(board.Rows) != 0 {
		t.Fatal("read implicitly discovered", board, err)
	}
	e := enrollmentFixture(t, s, c, domain.RunStatusComplete)
	c.runs[uuid.MustParse(e.Subjects[0].RunId)].Summary = &domain.RunSummary{CostEstimate: 12}
	before, _, _ := r.GetEffort(ctx, e.EffortRef)
	board, err = s.Board(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	row := board.Rows[0]
	if row.RuntimeState != "finished" || row.OutcomeStanding.State != "unknown" || row.Usage.Tokens != nil || row.Usage.ReportedCostUsd != nil {
		t.Fatal("activity/cost confused with evidence", row)
	}
	after, _, _ := r.GetEffort(ctx, e.EffortRef)
	if before.Revision != after.Revision {
		t.Fatal("board mutated enrollment")
	}
	delete(c.runs, uuid.MustParse(e.Subjects[0].RunId))
	board, _ = s.Board(ctx, nil)
	if len(board.Rows) != 1 || board.Rows[0].Assignments[0].UnavailableReason == "" {
		t.Fatal("owner outage hid enrollment", board)
	}
}

func TestEffortEnrollmentIdempotencyScopeAndWithdrawalSurviveRediscovery(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	e := enrollmentFixture(t, s, c, domain.RunStatusRunning)
	req := &pb.EnrollEffortRequest{Enrollment: proto.Clone(e).(*pb.EffortEnrollment), ExpectedRevision: e.Revision, IdempotencyKey: "amend"}
	req.Enrollment.TargetRevision = "accepted-2"
	if _, err := s.Enroll(ctx, req, EffortActor{ID: e.SupervisorRunId}); err == nil {
		t.Fatal("supervisor amended its own grant")
	}
	first, err := s.Enroll(ctx, req, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Enroll(ctx, req, EffortActor{ID: "owner", Operator: true})
	if err != nil || first.Revision != again.Revision {
		t.Fatal("enrollment retry changed revision", err)
	}
	req.Enrollment.TargetRevision = "accepted-3"
	if _, err = s.Enroll(ctx, req, EffortActor{ID: "owner", Operator: true}); !errors.Is(err, ErrConflict) {
		t.Fatal("changed request reused key", err)
	}
	workspaceFixture(t, s.config.Root, "retired", "effort:retired", nil)
	s.ReconcileDiscovery(ctx)
	discovered, _, _ := r.GetEffort(ctx, "effort:retired")
	s.Withdraw(ctx, &pb.WithdrawEffortRequest{EffortRef: discovered.EffortRef, ExpectedRevision: discovered.Revision, IdempotencyKey: "retire", Reason: "explicit closure"}, EffortActor{ID: "owner", Operator: true})
	s.ReconcileDiscovery(ctx)
	discovered, _, _ = r.GetEffort(ctx, discovered.EffortRef)
	if !discovered.Withdrawn {
		t.Fatal("rediscovery revived retired effort")
	}
}

func TestEffortMetadataReconciliationIsScopedAndPreservesAuthority(t *testing.T) {
	s, _, c := effortFixture(t)
	ctx := context.Background()
	e := enrollmentFixture(t, s, c, domain.RunStatusRunning)
	request := func(key string) *pb.ReconcileEffortMetadataRequest {
		enrollment := proto.Clone(e).(*pb.EffortEnrollment)
		return &pb.ReconcileEffortMetadataRequest{
			Enrollment:       enrollment,
			ExpectedRevision: e.Revision,
			IdempotencyKey:   key,
		}
	}
	request("worker").Enrollment.DisplayName = "worker attempt"
	if _, err := s.ReconcileMetadata(ctx, request("worker"), EffortActor{ID: uuid.NewString(), MetadataReconciler: true, Scopes: []string{EffortMetadataReconcileScope}}); err == nil {
		t.Fatal("unrelated run reconciled effort metadata")
	}

	coordinator := EffortActor{ID: e.Subjects[0].RunId, MetadataReconciler: true, Scopes: []string{EffortMetadataReconcileScope}}
	coordinatorRequest := request("coordinator")
	coordinatorRequest.Enrollment.DisplayName = "reconciled metadata"
	updated, err := s.ReconcileMetadata(ctx, coordinatorRequest, coordinator)
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != "reconciled metadata" || updated.Revision != e.Revision+1 || updated.AuthorizedBy != e.AuthorizedBy {
		t.Fatalf("metadata reconciliation lost retained state: %+v", updated)
	}

	replay, err := s.ReconcileMetadata(ctx, coordinatorRequest, coordinator)
	if err != nil || replay.Revision != updated.Revision {
		t.Fatalf("metadata reconciliation was not idempotent: revision=%d err=%v", replay.GetRevision(), err)
	}

	forged := request("forged-authority")
	forged.Enrollment.AuthorityRef = "forged"
	if _, err = s.ReconcileMetadata(ctx, forged, coordinator); err == nil {
		t.Fatal("metadata reconciliation accepted an authority change")
	}

	stale := request("stale")
	stale.ExpectedRevision = e.Revision
	if _, err = s.ReconcileMetadata(ctx, stale, EffortActor{ID: "owner", Operator: true}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale metadata reconciliation returned %v, want conflict", err)
	}
}

func TestEffortDeclaredCheckpointKeepsIndependentBlocker(t *testing.T) {
	s, _, _ := effortFixture(t)
	workspaceFixture(t, s.config.Root, "mixed", "effort:mixed", map[string]any{"checkpoint": "handoffs/current.json"})
	dir := filepath.Join(s.config.Root, "mixed", "handoffs")
	os.Mkdir(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "current.json"), []byte(`{"next_action":"investigate outcome B","rationale":"fresh acceptance failure","blockers":["outcome B failed"],"pending_operations":["test-genie:A"]}`), 0o600)
	if _, err := s.ReconcileDiscovery(context.Background()); err != nil {
		t.Fatal(err)
	}
	board, _ := s.Board(context.Background(), nil)
	if len(board.Rows) != 1 || !strings.Contains(board.Rows[0].NextAction, "outcome B") || len(board.Rows[0].Blockers) != 1 {
		t.Fatal("independent deviation masked", board)
	}
}

func legacyDriverFixture(t *testing.T, root, name, body string) string {
	t.Helper()
	dir := filepath.Join(root, name, "handoffs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEffortLegacyAdapterDiscoversArbitraryDriversWithAndWithoutManifest(t *testing.T) {
	s, r, c := effortFixture(t)
	ctx := context.Background()
	worker := uuid.New()
	c.runs[worker] = &domain.Run{ID: worker, Status: domain.RunStatusRunning}
	workspaceFixture(t, s.config.Root, "copper-kite", "effort:canonical-kite", map[string]any{"destination_ref": "", "target_revision": ""})
	legacyDriverFixture(t, s.config.Root, "copper-kite", fmt.Sprintf(`{"schema_version":1,"effort":"driver label unrelated to directory","next_action":"wait for required validation","loop_state":"waiting","children":{"worker":{"run_id":%q,"execution_id":"not-a-run","status":"complete","attempt":4}},"authorized_by":"forged","target_revision":"not-accepted"}`, worker.String()))
	legacyDriverFixture(t, s.config.Root, "violet-bridge", `{"effort":"independent investigation","next_action":"inspect evidence gap","orchestrator":{"loop_state":"waiting on owner"},"assignment":"../../credentials/must-not-read","children":{"missing":{"execution_id":"not-an-AM-run"}}}`)
	first, err := s.ReconcileDiscovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Board(ctx, nil)
	if err != nil || len(b.Rows) != 2 {
		t.Fatal("arbitrary discovery missing rows", b, err)
	}
	for _, row := range b.Rows {
		if row.Enrollment.AuthorizedBy != "" || row.Enrollment.TargetRevision != "" || row.Enrollment.DestinationRef != "" || row.OutcomeStanding.State != "unknown" || row.Usage.Tokens != nil || row.Usage.ReportedCostUsd != nil {
			t.Fatal("driver declaration invented grant, acceptance or costs", row)
		}
		if row.NextAction == "" || !strings.Contains(row.Rationale, "self-report") || len(row.EvidenceRefs) == 0 {
			t.Fatal("names-only discovery", row)
		}
		if row.Enrollment.EffortRef == "effort:canonical-kite" {
			if len(row.Assignments) != 1 || row.Assignments[0].Subject.RunId != worker.String() || row.Assignments[0].RuntimeState != "running" {
				t.Fatal("child declaration replaced AM state", row)
			}
		} else if row.Enrollment.EffortRef != "legacy-effort:independent investigation" || len(row.Assignments) != 0 || !strings.Contains(strings.Join(row.Limitations, " "), "run reference is unknown") {
			t.Fatal("manifestless source not attributed", row)
		}
	}
	second, err := s.ReconcileDiscovery(ctx)
	if err != nil || second.Generation != first.Generation {
		t.Fatal("unchanged driver cut churned", first, second, err)
	}
	legacyDriverFixture(t, s.config.Root, "violet-bridge", `{"effort":"independent investigation","next_action":"reconcile changed evidence","loop_state":"reviewing"}`)
	if _, err = s.ReconcileDiscovery(ctx); err != nil {
		t.Fatal(err)
	}
	_, observation, err := r.GetEffort(ctx, "legacy-effort:independent investigation")
	if err != nil || observation.NextAction != "reconcile changed evidence" {
		t.Fatal("driver progress not observed", observation, err)
	}
}

func TestEffortLegacyAdapterPrefersDeclaredCheckpointAndExactSubjects(t *testing.T) {
	s, _, _ := effortFixture(t)
	id := uuid.NewString()
	workspaceFixture(t, s.config.Root, "silver-orbit", "effort:declared", map[string]any{"checkpoint": "handoffs/current.json", "supervision": map[string]any{"subjects": []any{map[string]any{"owner": "agent-manager", "kind": "run", "reference": id, "run_id": id, "role": "orchestrator"}}}})
	legacyDriverFixture(t, s.config.Root, "silver-orbit", `{"effort":"ignored fallback","next_action":"wrong action","loop_state":"wrong loop"}`)
	declared := fmt.Sprintf(`{"effort":"declared driver","next_action":"declared action","orchestrator":{"loop_state":"declared wait"},"children":{"duplicate":{"run_id":%q}}}`, id)
	path := filepath.Join(s.config.Root, "silver-orbit", "handoffs", "current.json")
	if err := os.WriteFile(path, []byte(declared), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReconcileDiscovery(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, err := s.Board(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	row := b.Rows[0]
	if row.NextAction != "declared action" || !strings.Contains(row.Rationale, "declared wait") || len(row.Enrollment.Subjects) != 1 || row.Enrollment.Subjects[0].Role != "orchestrator" {
		t.Fatal("fallback/child overrode explicit declaration", row)
	}
	if strings.Contains(strings.Join(row.EvidenceRefs, " "), "/state.json@") {
		t.Fatal("read undeclared fallback despite explicit checkpoint")
	}
}

func TestEffortLegacyAdapterPreservesMissingAndMalformedUncertainty(t *testing.T) {
	for _, manifest := range []bool{false, true} {
		t.Run(fmt.Sprint("manifest=", manifest), func(t *testing.T) {
			s, r, _ := effortFixture(t)
			ctx := context.Background()
			ref := "legacy-effort:recoverable"
			if manifest {
				ref = "effort:recoverable"
				workspaceFixture(t, s.config.Root, "amber-river", ref, nil)
			}
			path := legacyDriverFixture(t, s.config.Root, "amber-river", `{"effort":"recoverable","next_action":"wait for owner","loop_state":"waiting"}`)
			if _, err := s.ReconcileDiscovery(ctx); err != nil {
				t.Fatal(err)
			}
			for _, broken := range []string{"missing", "malformed"} {
				if broken == "missing" {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				} else {
					legacyDriverFixture(t, s.config.Root, "amber-river", "{broken")
				}
				d, err := s.ReconcileDiscovery(ctx)
				if err != nil {
					t.Fatal(err)
				}
				e, o, err := r.GetEffort(ctx, ref)
				if err != nil || e.Withdrawn || o.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE || !d.Partial || len(d.Findings) == 0 {
					t.Fatal("lost driver became success or disappeared", e, o, d, err)
				}
				if _, err = s.ReconcileDiscovery(ctx); err != nil {
					t.Fatal(err)
				}
				_, again, err := r.GetEffort(ctx, ref)
				if err != nil || again.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE {
					t.Fatal("repeated missing/malformed scan cleared uncertainty", again, err)
				}
			}
		})
	}
	// A missing manifest may use the fallback without duplicating its retained identity.
	s, _, _ := effortFixture(t)
	workspaceFixture(t, s.config.Root, "retained", "effort:retained", nil)
	legacyDriverFixture(t, s.config.Root, "retained", `{"effort":"different label","loop_state":"waiting"}`)
	if _, err := s.ReconcileDiscovery(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(s.config.Root, "retained", "effort.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReconcileDiscovery(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, err := s.Board(context.Background(), nil)
	if err != nil || len(b.Rows) != 1 || b.Rows[0].Enrollment.EffortRef != "effort:retained" || b.Rows[0].Enrollment.TargetRevision != "" {
		t.Fatal("manifest loss duplicated identity or preserved approval claim", b, err)
	}
}

func TestEffortLegacyAdapterBoundsAndManifestFailureAreNotBypassed(t *testing.T) {
	for _, mode := range []string{"version", "shape", "bytes", "symlink", "manifest"} {
		t.Run(mode, func(t *testing.T) {
			s, _, _ := effortFixture(t)
			body := `{"effort":"arbitrary","loop_state":"waiting"}`
			switch mode {
			case "version":
				body = `{"schema_version":2,"effort":"arbitrary","loop_state":"waiting"}`
			case "shape":
				body = `{"effort":"arbitrary","loop_state":{"state":"not-a-string"}}`
			case "bytes":
				body = strings.Repeat(" ", int(s.config.FileBytes)+1)
			case "manifest":
				workspaceFixture(t, s.config.Root, "sandboxed", "effort:no-bypass", map[string]any{"schema_version": 99})
			}
			path := legacyDriverFixture(t, s.config.Root, "sandboxed", body)
			if mode == "symlink" {
				outside := filepath.Join(t.TempDir(), "credential-file")
				if err := os.WriteFile(outside, []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			}
			d, err := s.ReconcileDiscovery(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			b, err := s.Board(context.Background(), nil)
			if err != nil || len(b.Rows) != 0 || !d.Partial || len(d.Findings) == 0 {
				t.Fatal("unsafe/incompatible source became a fresh row", b, d, err)
			}
		})
	}
}
