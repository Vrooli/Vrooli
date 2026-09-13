package heartbeat

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	amapi "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	amconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type effortBoardHandlerFake struct {
	amconnect.UnimplementedAgentManagerServiceHandler
	request *ampb.GetEffortBoardRequest
	board   *ampb.EffortBoard
}

func TestStandingSupervisorDoesNotJudgeDispatchAuthorizationAnchor(t *testing.T) {
	cut := mapEffortBoard(&ampb.EffortBoard{Rows: []*ampb.EffortBoardRow{{Enrollment: &ampb.EffortEnrollment{EffortRef: "arbitrary-authorization-id"}, ChangeIdentity: "grant-cut", Freshness: ampb.EffortFreshness_EFFORT_FRESHNESS_FRESH, NextAction: "authorization-only"}}})
	if len(cut.Efforts) != 1 || cut.Efforts[0].Eligible || cut.Efforts[0].Retired {
		t.Fatal("authorization anchor became a judgment target or was falsely withdrawn", cut)
	}
}

func TestStandingSupervisorFreshBoardWithoutGrantRemainsObservationOnly(t *testing.T) {
	row := &ampb.EffortBoardRow{Enrollment: &ampb.EffortEnrollment{EffortRef: "any-effort", TargetRevision: "accepted"}, Freshness: ampb.EffortFreshness_EFFORT_FRESHNESS_FRESH, ChangeIdentity: "source"}
	for _, variant := range []string{"missing", "expired", "valid"} {
		t.Run(variant, func(t *testing.T) {
			e := row.Enrollment
			if variant != "missing" {
				e.AuthorizedBy = "owner"
				e.PermittedActions = []ampb.WatchActionKind{ampb.WatchActionKind_WATCH_ACTION_KIND_CONTINUE}
				e.AuthorityExpiresAt = timestamppb.New(time.Now().Add(-time.Hour))
				if variant == "valid" {
					e.AuthorityExpiresAt = timestamppb.New(time.Now().Add(time.Hour))
				}
			}
			cut := mapEffortBoard(&ampb.EffortBoard{Rows: []*ampb.EffortBoardRow{row}, ObservedAt: timestamppb.Now()})
			if !cut.Efforts[0].Eligible || cut.Efforts[0].ObservationOnly != (variant != "valid") {
				t.Fatal("freshness was confused with permission", cut)
			}
		})
	}
}

func (h *effortBoardHandlerFake) GetEffortBoard(_ context.Context, req *connect.Request[ampb.GetEffortBoardRequest]) (*connect.Response[ampb.EffortBoard], error) {
	h.request = req.Msg
	return connect.NewResponse(h.board), nil
}

func TestStandingSupervisorRecoveryFollowupUsesOwnerDirectiveState(t *testing.T) {
	for _, state := range []string{"pending", "progress-observed", "owner-wait", "failed", "superseded", "wrong-revision", "undelivered", "legacy"} {
		t.Run(state, func(t *testing.T) {
			d := &ampb.EffortDirective{TargetRevision: "accepted", Delivery: ampb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED, RecoveryExpectation: &ampb.EffortRecoveryExpectation{ProgressCondition: "owner receipt"}, RecoveryVerification: &ampb.EffortRecoveryVerification{State: state}}
			switch state {
			case "superseded":
				d.RecoveryVerification.State, d.SupersededBy = "pending", "replacement-directive"
			case "wrong-revision":
				d.RecoveryVerification.State, d.TargetRevision = "pending", "old"
			case "undelivered":
				d.RecoveryVerification.State = "pending"
				d.Delivery = ampb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN
			case "legacy":
				d.RecoveryVerification, d.RecoveryExpectation = nil, nil
			}
			row := &ampb.EffortBoardRow{Enrollment: &ampb.EffortEnrollment{EffortRef: "any-effort", TargetRevision: "accepted"}, ChangeIdentity: "cut", Directives: []*ampb.EffortDirective{d}}
			cut := mapEffortBoard(&ampb.EffortBoard{Rows: []*ampb.EffortBoardRow{row}})
			if cut.Efforts[0].RecoveryVerificationPending != (state == "pending") {
				t.Fatal("follow-up inferred from activity rather than current pending verification", cut)
			}
		})
	}
}

func TestStandingSupervisorGeneratedAMBoardContract(t *testing.T) {
	owner := &effortBoardHandlerFake{board: &ampb.EffortBoard{NextPageToken: "next", Rows: []*ampb.EffortBoardRow{
		{Enrollment: &ampb.EffortEnrollment{EffortRef: "arbitrary/effort", TargetRevision: "target", Revision: 2}, ChangeIdentity: "meaningful-cut", Freshness: ampb.EffortFreshness_EFFORT_FRESHNESS_FRESH, RuntimeState: "running", PendingOperations: []string{"subject-wait"}},
		{Enrollment: &ampb.EffortEnrollment{EffortRef: "retired", Withdrawn: true}, Freshness: ampb.EffortFreshness_EFFORT_FRESHNESS_FRESH},
		{Enrollment: &ampb.EffortEnrollment{EffortRef: "missing"}, Freshness: ampb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE},
	}}}
	_, handler := amconnect.NewAgentManagerServiceHandler(owner)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := NewAgentManagerClient(time.Second)
	client.testBaseURL = server.URL
	cut, err := client.DiscoverEfforts(context.Background(), 3, "page")
	if err != nil {
		t.Fatal(err)
	}
	if owner.request.PageSize != 3 || owner.request.PageToken != "page" || cut.NextCursor != "next" {
		t.Fatal("typed pagination not preserved")
	}
	if !cut.Efforts[0].Eligible || cut.Efforts[0].WaitRef != "subject-wait" || !cut.Efforts[1].Retired || cut.Efforts[2].Eligible {
		t.Fatal("owner eligibility/retirement/subject-wait semantics lost")
	}
	before := cut.Efforts[0].revision()
	owner.board.ObservedAt = timestamppb.Now()
	owner.board.Rows[0].ObservedAt = timestamppb.Now()
	owner.board.Rows[0].Enrollment.Revision++ // Fresh join CAS is not subject evidence.
	cut, err = client.DiscoverEfforts(context.Background(), 3, "page")
	if err != nil || cut.Efforts[0].revision() != before {
		t.Fatal("observation timestamp made unchanged evidence eligible")
	}
	if _, err := client.DiscoverEfforts(context.Background(), 101, ""); err == nil {
		t.Fatal("wire page limit not enforced")
	}
	owner.board.Rows[0].RuntimeState = "unknown"
	owner.board.Rows[0].Limitations = []string{"owner lifecycle inconsistency; reconcile original run identity"}
	cut, err = client.DiscoverEfforts(context.Background(), 3, "page")
	if err != nil || !cut.Efforts[0].Eligible {
		t.Fatal("inconsistent subject lifecycle blocked bounded diagnosis")
	}
	owner.board.Rows[0].LastAssessment = &ampb.EffortAssessment{AssessmentId: "receipt", EffortRefs: []string{"arbitrary/effort"}, TargetRevisions: map[string]string{"arbitrary/effort": "target"}, SupervisorRunId: "supervisor", IdempotencyKey: "wake", Disposition: "sample"}
	receipt, err := client.GetEffortAssessment(context.Background(), "arbitrary/effort")
	if err != nil || receipt == nil || receipt.ID != "receipt" || receipt.RunID != "supervisor" || receipt.WakeID != "wake" || receipt.TargetRevisions["arbitrary/effort"] != "target" || receipt.Disposition != "sample" || owner.request.EffortRef != "arbitrary/effort" || owner.request.PageSize != 1 {
		t.Fatal("typed exact-effort assessment receipt contract lost")
	}
}

func TestStandingSupervisorAMAssessmentOnlyChangeDoesNotWakeOrSwallowSubjectChange(t *testing.T) {
	for _, subjectChanged := range []bool{false, true} {
		t.Run(map[bool]string{false: "assessment-only", true: "subject-changed-during-wake"}[subjectChanged], func(t *testing.T) {
			f := newSupervisionFixture(t)
			row := &ampb.EffortBoardRow{Enrollment: &ampb.EffortEnrollment{EffortRef: "a", TargetRevision: "target", Revision: 1}, ChangeIdentity: "subject-cut-1", Freshness: ampb.EffortFreshness_EFFORT_FRESHNESS_FRESH}
			owner := &effortBoardHandlerFake{board: &ampb.EffortBoard{Rows: []*ampb.EffortBoardRow{row}}}
			_, handler := amconnect.NewAgentManagerServiceHandler(owner)
			server := httptest.NewServer(handler)
			defer server.Close()
			client := NewAgentManagerClient(time.Second)
			client.testBaseURL = server.URL
			f.s.Owner = client
			wake := f.tick(t).Pending
			row.Enrollment.Revision++ // Revalidation must survive a fresh join.
			f.dispatch(t)
			row.LastAssessment = &ampb.EffortAssessment{AssessmentId: "receipt", EffortRefs: []string{"a"}, TargetRevisions: map[string]string{"a": "target"}, SupervisorRunId: "wake-run-1", IdempotencyKey: wake.ID, Disposition: "quiet", ObservedUsage: &ampb.EffortUsage{}}
			// AM's subject identity must exclude self-assessment/accounting. The
			// full board display identity may change for those fields independently.
			owner.board.ChangeIdentity = "new-full-display-cut"
			row.Enrollment.Revision++
			if subjectChanged {
				row.ChangeIdentity = "subject-cut-2"
			}
			f.agent.getRuns["wake-run-1"].Status = "complete"
			f.now = f.now.Add(time.Minute)
			state := f.tick(t)
			if (state.Pending != nil) != subjectChanged {
				t.Fatal("self assessment woke again or a real concurrent subject change was swallowed")
			}
			if state.Efforts["a"].ServedRevision != wake.Efforts[0].revision() {
				t.Fatal("receipt marked the latest rather than assessed cut served")
			}
			if !subjectChanged {
				for i := 0; i < 3; i++ {
					row.Enrollment.Revision++
					if f.tick(t).Pending != nil {
						t.Fatal("fresh joins created self-feedback wake")
					}
				}
				row.Enrollment.TargetRevision = "new-target"
				if f.tick(t).Pending == nil {
					t.Fatal("target revision change was swallowed")
				}
			}
		})
	}
}

func TestStandingSupervisorObservationWorkReferencesAndTypedAssessment(t *testing.T) {
	f := newSupervisionFixture(t)
	f.cfg.Supervision.MaxEffortsPerWake = 2
	f.owner.rows = []EffortObservation{effort("arbitrary/a"), effort("other:b")}
	wake := f.tick(t).Pending
	f.dispatch(t)
	req := f.agent.createRunCalls[0]
	if len(req.WorkReferences) != 2 {
		t.Fatal("observation omitted selected effort work references")
	}
	for i, ref := range req.WorkReferences {
		if ref.Kind != "effort" || ref.Id != wake.Efforts[i].ID || ref.Revision != wake.Efforts[i].TargetRevision || ref.Relationship != "supervisor" || !ref.Verified || ref.Visibility != eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC || ref.State != eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE {
			t.Fatal("work reference is not an exact owner-observed supervisor relationship")
		}
	}
	for key := range req.Environment {
		if strings.Contains(key, "TOKEN") || strings.Contains(key, "SCOPE") {
			t.Fatal("PM supplied token or scope authority in worker environment")
		}
	}
	prompt := f.agent.createTaskCalls[0].Description
	if !strings.Contains(prompt, "agent-manager effort assess --request-file <request.json> --json") || !strings.Contains(prompt, "VROOLI_AGENT_IDENTITY_TOKEN") {
		t.Fatal("prompt omitted actual signed assessment operation")
	}
	if !strings.Contains(prompt, "pending periodic board membership join does not require waiting") || !strings.Contains(prompt, "retain the exact refusal and finish without claiming a receipt") {
		t.Fatal("prompt lost immediate owner verification or explicit refusal handling")
	}
	if !strings.Contains(prompt, `prompt-manager team knowledge-add "supervisors" --topic="supervision-assessment/`+wake.ID+`"`) || !strings.Contains(prompt, "Retain any link failure separately from the accepted AM receipt and finish") {
		t.Fatal("prompt lost the selected team's typed link operation or bounded post-receipt exit")
	}
	var assessment ampb.RecordEffortAssessmentRequest
	if err := protojson.Unmarshal([]byte(prompt[strings.LastIndex(prompt, "\n")+1:]), &assessment); err != nil {
		t.Fatal(err)
	}
	a := assessment.Assessment
	if a == nil || a.IdempotencyKey != wake.ID || a.SharedOperationRef != wake.ID || a.AllowanceRef != wake.AccountingRef || len(a.EffortRefs) != 2 || a.TargetRevisions["arbitrary/a"] != "target-1" || a.SupervisorRunId != "" || !a.ObservedUsage.Partial || assessment.Authority != ampb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT {
		t.Fatal("assessment skeleton lost receipt identity or fabricated caller authority")
	}
}

func TestStandingSupervisorCreateRunGeneratedWorkReferenceContract(t *testing.T) {
	f := newSupervisionFixture(t)
	f.owner.rows = []EffortObservation{effort("arbitrary/effort")}
	f.tick(t)
	f.dispatch(t)
	b, err := json.Marshal(f.agent.createRunCalls[0])
	if err != nil {
		t.Fatal(err)
	}
	var request amapi.CreateRunRequest
	if err := protojson.Unmarshal(b, &request); err != nil {
		t.Fatalf("AM CreateRun wire cannot retain observation membership; adopt the owner transport before activation: %v", err)
	}
}

func TestStandingSupervisorStaleTransitionAssessedOnceWithoutSteering(t *testing.T) {
	for _, freshness := range []ampb.EffortFreshness{ampb.EffortFreshness_EFFORT_FRESHNESS_STALE, ampb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE} {
		t.Run(freshness.String(), func(t *testing.T) {
			f := newSupervisionFixture(t)
			f.cfg.Supervision.HealthySampleIntervalSeconds, f.cfg.Supervision.MaxHealthySamplesPerWake = 60, 1
			row := &ampb.EffortBoardRow{Enrollment: &ampb.EffortEnrollment{EffortRef: "a", TargetRevision: "target-1"}, Freshness: ampb.EffortFreshness_EFFORT_FRESHNESS_FRESH, ChangeIdentity: "fresh-cut"}
			f.owner.rows = mapEffortBoard(&ampb.EffortBoard{Rows: []*ampb.EffortBoardRow{row}}).Efforts
			f.tick(t)
			f.dispatch(t)
			f.receipt(t, "quiet")
			f.agent.getRuns["wake-run-1"].Status = "complete"
			f.tick(t)
			row.Freshness, row.ChangeIdentity = freshness, "stale-or-unavailable-cut"
			f.owner.rows = mapEffortBoard(&ampb.EffortBoard{Rows: []*ampb.EffortBoardRow{row}}).Efforts
			f.now = f.now.Add(time.Minute)
			state := f.tick(t)
			if state.Pending == nil || !state.Pending.Efforts[0].ObservationOnly || !strings.Contains(state.Pending.Efforts[0].Reason, "no steering") {
				t.Fatal("valid stale identity could not get bounded uncertainty assessment")
			}
			f.dispatch(t)
			f.receipt(t, "unknown")
			if f.tick(t).Pending != nil {
				t.Fatal("assessed stale cut woke again")
			}
			f.now = f.now.Add(time.Hour)
			if f.tick(t).Pending != nil || len(f.agent.createRunCalls) != 2 {
				t.Fatal("unchanged unavailable evidence was not coalesced")
			}
		})
	}
}

func TestStandingSupervisorMalformedObservationExcludedWithoutLosingValidSibling(t *testing.T) {
	cut := mapEffortBoard(&ampb.EffortBoard{Rows: []*ampb.EffortBoardRow{
		{Enrollment: &ampb.EffortEnrollment{}, Freshness: ampb.EffortFreshness_EFFORT_FRESHNESS_STALE},
		{Enrollment: &ampb.EffortEnrollment{EffortRef: "missing-evidence"}},
		{Enrollment: &ampb.EffortEnrollment{EffortRef: "valid", TargetRevision: "target"}, ChangeIdentity: "cut", Freshness: ampb.EffortFreshness_EFFORT_FRESHNESS_UNAVAILABLE},
		{Enrollment: &ampb.EffortEnrollment{EffortRef: "withdrawn", TargetRevision: "target", Withdrawn: true}, ChangeIdentity: "cut"},
	}})
	if cut.Coverage != "partial" || cut.Error == "" || len(cut.Efforts) != 3 || cut.Efforts[0].Eligible || !cut.Efforts[1].Eligible || cut.Efforts[2].Eligible {
		t.Fatal("malformed identity was admitted or hid the valid uncertainty cut")
	}
}

func TestStandingSupervisorUnknownTargetAssessmentRequiresExplicitEmptyReceipt(t *testing.T) {
	for _, includeTarget := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing-map-key", true: "exact-empty-target"}[includeTarget], func(t *testing.T) {
			f := newSupervisionFixture(t)
			f.owner.rows = mapEffortBoard(&ampb.EffortBoard{Rows: []*ampb.EffortBoardRow{{Enrollment: &ampb.EffortEnrollment{EffortRef: "legacy-effort:arbitrary", SourceRevision: "retained-source"}, ChangeIdentity: "external-input-cut", Freshness: ampb.EffortFreshness_EFFORT_FRESHNESS_STALE, NextAction: "external-input", Blockers: []string{"owner input pending"}}}}).Efforts
			wake := f.tick(t).Pending
			if wake == nil || !wake.Efforts[0].ObservationOnly || wake.Efforts[0].TargetRevision != "" {
				t.Fatal("unknown accepted target suppressed observation or fabricated revision")
			}
			f.dispatch(t)
			prompt := f.agent.createTaskCalls[0].Description
			var req ampb.RecordEffortAssessmentRequest
			if err := protojson.Unmarshal([]byte(prompt[strings.LastIndex(prompt, "\n")+1:]), &req); err != nil {
				t.Fatal(err)
			}
			if value, ok := req.Assessment.TargetRevisions["legacy-effort:arbitrary"]; !ok || value != "" {
				t.Fatal("typed assessment omitted the explicit unknown target")
			}
			f.receipt(t, "unknown")
			if !includeTarget {
				delete(f.owner.receipts["legacy-effort:arbitrary"].TargetRevisions, "legacy-effort:arbitrary")
			}
			f.agent.getRuns["wake-run-1"].Status = "complete"
			state := f.tick(t)
			if got := state.Efforts["legacy-effort:arbitrary"].AssessmentID != ""; got != includeTarget {
				t.Fatal("coverage did not require an explicit exact-empty target receipt")
			}
			if includeTarget {
				f.now = f.now.Add(time.Hour)
				if f.tick(t).Pending != nil {
					t.Fatal("unchanged unknown target cut did not coalesce")
				}
			}
		})
	}
}
