package supervision

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

func TestEffortRegistryUnknownRelationshipDoesNotBecomeWorker(t *testing.T) {
	s, _, c := effortFixture(t)
	id := uuid.New()
	run := &domain.Run{ID: id, Status: domain.RunStatusRunning, WorkReferences: []*eventpb.WorkReference{{Kind: "effort", Id: "effort:unclassified", Relationship: "observer-new-role", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}}}
	c.runs[id] = run
	s.SetRunRegistry(effortRegistryFixture{runs: []*domain.Run{run}})
	if _, err := s.ReconcileDiscovery(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, err := s.Board(context.Background(), &pb.GetEffortBoardRequest{EffortRef: "effort:unclassified"})
	if err != nil || len(b.GetRows()) != 1 {
		t.Fatal(b, err)
	}
	row := b.Rows[0]
	if row.RuntimeState != "unknown" || row.Enrollment.Subjects[0].Role != "unknown" || row.Usage.ObservedRuns != 1 {
		t.Fatal("unknown relationship fabricated business activity or lost usage", row)
	}
}

func TestEffortResolutionSourcesReopenStoppedObservationWithoutGrant(t *testing.T) {
	for _, manifest := range []bool{false, true} {
		t.Run(map[bool]string{true: "declared", false: "legacy"}[manifest], func(t *testing.T) {
			s, _, _ := effortFixture(t)
			dir := filepath.Join(s.config.Root, "arbitrary-effort")
			if err := os.MkdirAll(filepath.Join(dir, "handoffs"), 0700); err != nil {
				t.Fatal(err)
			}
			write := func(path, value string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, path), []byte(value), 0600); err != nil {
					t.Fatal(err)
				}
			}
			write("handoffs/state.json", `{"effort":"arbitrary","next_action":"wait for credential repair","orchestrator":{"loop_state":"stopped"}}`)
			path := "handoffs/OPERATOR-ANSWERS.md"
			if manifest {
				path = "handoffs/owner-resolution.md"
				write("effort.json", `{"schema_version":1,"slug":"arbitrary","repository":"repo","effort_ref":"effort:arbitrary","observation_sources":["handoffs/owner-resolution.md"]}`)
			}
			write(path, "Repair pending; confidential text is never projected")
			board := func() *pb.EffortBoardRow {
				t.Helper()
				if _, err := s.ReconcileDiscovery(context.Background()); err != nil {
					t.Fatal(err)
				}
				b, err := s.Board(context.Background(), &pb.GetEffortBoardRequest{})
				if err != nil || len(b.Rows) != 1 {
					t.Fatalf("board=%v err=%v", b, err)
				}
				return b.Rows[0]
			}
			before := board()
			if after := board(); after.ChangeIdentity != before.ChangeIdentity {
				t.Fatal("unchanged evidence manufactured another wake")
			}
			write(path, "Repair qualified; owner receipt run:resolved; recheck old executor")
			after := board()
			if after.ChangeIdentity == before.ChangeIdentity {
				t.Fatal("new resolution did not reopen the stopped evidence cut")
			}
			if after.Enrollment.AuthorizedBy != "" || len(after.Enrollment.PermittedActions) != 0 {
				t.Fatal("resolution file manufactured a grant")
			}
			if strings.Contains(after.String(), "Repair qualified") {
				t.Fatal("source content leaked into board")
			}
			if !strings.Contains(strings.Join(after.EvidenceRefs, " "), path+"@") {
				t.Fatal("resolution source identity absent")
			}
			if after.NextAction != before.NextAction {
				t.Fatal("file contents replaced coordinator state")
			}
		})
	}
}

func TestEffortResolutionSourcesRejectUnsafeAndUnavailableEvidence(t *testing.T) {
	for _, source := range []string{"../escape", "handoffs/../../escape", "credentials/auth.json", "handoffs/missing.md", "handoffs/link.md", "handoffs/oversize.md"} {
		t.Run(source, func(t *testing.T) {
			s, _, _ := effortFixture(t)
			dir := filepath.Join(s.config.Root, "arbitrary")
			if err := os.MkdirAll(filepath.Join(dir, "handoffs"), 0700); err != nil {
				t.Fatal(err)
			}
			manifest := `{"schema_version":1,"slug":"arbitrary","repository":"repo","observation_sources":["` + source + `"]}`
			if err := os.WriteFile(filepath.Join(dir, "effort.json"), []byte(manifest), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("missing.md", filepath.Join(dir, "handoffs/link.md")); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "handoffs/oversize.md"), []byte(strings.Repeat("x", 2048)), 0600); err != nil {
				t.Fatal(err)
			}
			root, err := openSafeRoot(s.config.Root)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			item, err := readWorkspace(root, "arbitrary", 1024, s.now())
			if err == nil && item.observation.Freshness != pb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE {
				t.Fatal("unsafe/missing resolution was treated as fresh")
			}
		})
	}
}

func TestEffortFailedRecoveryNeedsContinueGrantAndSession(t *testing.T) {
	for _, variant := range []string{"recover", "nudge", "no-session", "no-hypothesis", "no-comparison", "no-expectation", "cancelled", "completed", "parked", "running"} {
		t.Run(variant, func(t *testing.T) {
			s, _, c := effortFixture(t)
			e := grantFixture(t, s, c, domain.RunStatusFailed)
			e.PermittedActions = append(e.PermittedActions, pb.WatchActionKind_WATCH_ACTION_KIND_CONTINUE)
			var err error
			e, err = s.Enroll(context.Background(), &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "recovery-grant"}, EffortActor{ID: "owner", Operator: true})
			if err != nil {
				t.Fatal(err)
			}
			run := c.runs[uuid.MustParse(e.Subjects[0].RunId)]
			run.SessionID = "retained-session"
			req := directiveFixture(s, e)
			req.Directive.Kind = pb.WatchActionKind_WATCH_ACTION_KIND_CONTINUE
			req.Directive.RecoveryExpectation = &pb.EffortRecoveryExpectation{ProgressCondition: "assigned owner regression passes with a retained receipt", BaselineEvidenceRefs: []string{"test-genie:before-recovery"}}
			switch variant {
			case "nudge":
				req.Directive.Kind = pb.WatchActionKind_WATCH_ACTION_KIND_NUDGE
			case "no-session":
				run.SessionID = ""
			case "no-hypothesis":
				req.Directive.Hypothesis = ""
			case "no-comparison":
				req.Directive.Comparison = ""
			case "no-expectation":
				req.Directive.RecoveryExpectation = nil
			case "cancelled":
				run.Status = domain.RunStatusCancelled
			case "completed":
				run.Status = domain.RunStatusComplete
			case "parked":
				run.Status = domain.RunStatusParked
			case "running":
				run.Status = domain.RunStatusRunning
			}
			d, err := s.RequestDirective(context.Background(), req, EffortActor{ID: e.SupervisorRunId})
			if err != nil {
				t.Fatal(err)
			}
			if variant == "recover" {
				if d.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED || c.continued != 1 {
					t.Fatalf("eligible owner recovery refused: %v", d)
				}
				if _, err := s.RequestDirective(context.Background(), req, EffortActor{ID: e.SupervisorRunId}); err != nil || c.continued != 1 {
					t.Fatal("replay duplicated recovery", err)
				}
			} else if c.continued != 0 {
				t.Fatal("ineligible run restarted")
			}
		})
	}
}
