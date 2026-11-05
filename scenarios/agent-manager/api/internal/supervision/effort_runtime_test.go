package supervision

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

// ES-02/03/05: observation work remains visible and billable, while only
// unambiguous product executors establish runtime coverage. QA knw-1789218617635162514.
func TestEffortRuntimeRoleCoverage(t *testing.T) {
	type subject struct {
		role       string
		status     domain.RunStatus
		missing    bool
		ended      bool
		foreign    bool
		shared     bool
		supervisor bool
	}
	for _, tc := range []struct {
		name        string
		subjects    []subject
		wantRuntime string
		wantDetail  string
	}{
		{"supervisor complete only", []subject{{role: "supervisor", status: domain.RunStatusComplete}}, "unknown", "orchestrator runtime coverage: declared=0"},
		{"supervisor active only", []subject{{role: "supervisor", status: domain.RunStatusRunning}}, "unknown", "orchestrator runtime coverage: declared=0"},
		{"diagnostic activity only", []subject{{role: "reviewer", status: domain.RunStatusRunning}, {role: "investigator", status: domain.RunStatusComplete}}, "unknown", "worker runtime coverage: declared=0"},
		{"unclassified activity only", []subject{{role: "", status: domain.RunStatusRunning}, {role: "future-role", status: domain.RunStatusComplete}}, "unknown", "worker runtime coverage: declared=0"},
		{"repair observation only", []subject{{role: "repair", status: domain.RunStatusRunning}}, "unknown", "orchestrator runtime coverage: declared=0"},
		{"active supervisor failed orchestrator", []subject{{role: "supervisor", status: domain.RunStatusRunning}, {role: "orchestrator", status: domain.RunStatusFailed}}, "finished", "orchestrator runtime coverage: declared=1 active=0 terminal=1 unknown=0"},
		{"active supervisor stopped orchestrator", []subject{{role: "supervisor", status: domain.RunStatusRunning}, {role: "orchestrator", status: domain.RunStatusCancelled}}, "finished", "orchestrator runtime coverage: declared=1 active=0 terminal=1 unknown=0"},
		{"active supervisor missing orchestrator", []subject{{role: "supervisor", status: domain.RunStatusRunning}, {role: "orchestrator", missing: true}}, "unknown", "orchestrator runtime coverage: declared=1 active=0 terminal=0 unknown=1"},
		{"active worker stopped orchestrator", []subject{{role: "worker", status: domain.RunStatusRunning}, {role: "orchestrator", status: domain.RunStatusCancelled}}, "active", "orchestrator runtime coverage: declared=1 active=0 terminal=1 unknown=0"},
		{"active worker missing orchestrator", []subject{{role: "worker", status: domain.RunStatusRunning}, {role: "orchestrator", missing: true}}, "active", "orchestrator runtime coverage: declared=1 active=0 terminal=0 unknown=1"},
		{"active worker no declared orchestrator", []subject{{role: "worker", status: domain.RunStatusRunning}}, "active", "orchestrator runtime coverage: declared=0"},
		{"terminal worker no declared orchestrator", []subject{{role: "worker", status: domain.RunStatusComplete}}, "unknown", "orchestrator runtime coverage: declared=0"},
		{"orchestrator active worker coverage unknown", []subject{{role: "orchestrator", status: domain.RunStatusRunning}}, "active", "worker runtime coverage: declared=0"},
		{"complete orchestrator missing worker", []subject{{role: "orchestrator", status: domain.RunStatusComplete}, {role: "worker", missing: true}}, "unknown", "worker runtime coverage: declared=1 active=0 terminal=0 unknown=1"},
		{"complete orchestrator unknown worker", []subject{{role: "orchestrator", status: domain.RunStatusComplete}, {role: "worker", status: domain.RunStatusUnknown}}, "unknown", "worker runtime coverage: declared=1 active=0 terminal=0 unknown=1"},
		{"complete orchestrator unavailable supervisor", []subject{{role: "orchestrator", status: domain.RunStatusComplete}, {role: "supervisor", missing: true}}, "finished", "orchestrator runtime coverage: declared=1 active=0 terminal=1 unknown=0"},
		{"inconsistent active business owner", []subject{{role: "orchestrator", status: domain.RunStatusRunning, ended: true}, {role: "supervisor", status: domain.RunStatusRunning}}, "unknown", "orchestrator runtime coverage: declared=1 active=0 terminal=0 unknown=1"},
		{"foreign owner cannot use AM liveness", []subject{{role: "orchestrator", status: domain.RunStatusRunning, foreign: true}}, "unknown", "orchestrator runtime coverage: declared=1 active=0 terminal=0 unknown=1"},
		{"supervisor id overlaps worker", []subject{{role: "worker", status: domain.RunStatusRunning, supervisor: true}}, "unknown", "conflicting execution roles"},
		{"shared supervisor worker execution", []subject{{role: "worker", status: domain.RunStatusRunning, shared: true}, {role: "supervisor", status: domain.RunStatusRunning, shared: true}}, "unknown", "conflicting execution roles"},
		{"shared worker supervisor reversed", []subject{{role: "supervisor", status: domain.RunStatusRunning, shared: true}, {role: "worker", status: domain.RunStatusRunning, shared: true}}, "unknown", "conflicting execution roles"},
		{"shared orchestrator worker execution", []subject{{role: "orchestrator", status: domain.RunStatusRunning, shared: true}, {role: "worker", status: domain.RunStatusRunning, shared: true}}, "unknown", "conflicting execution roles"},
		{"parked business owner", []subject{{role: "orchestrator", status: domain.RunStatusParked}}, "active", "orchestrator runtime coverage: declared=1 active=1 terminal=0 unknown=0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, c := effortFixture(t)
			e := &pb.EffortEnrollment{EffortRef: "effort:role-coverage"}
			for _, spec := range tc.subjects {
				id := uuid.New()
				sub := &pb.EffortSubject{Owner: "agent-manager", Kind: "run", Reference: id.String(), RunId: id.String(), Role: spec.role}
				if spec.foreign {
					sub.Owner = "another-owner"
				}
				e.Subjects = append(e.Subjects, sub)
				if spec.supervisor {
					e.SupervisorRunId = id.String()
				}
				if !spec.missing {
					run := &domain.Run{ID: id, Status: spec.status}
					if spec.shared {
						run.SessionID, run.HarnessKind = "same-execution", "codex"
					}
					if spec.ended {
						ended := s.now().Add(-time.Minute)
						run.EndedAt = &ended
					}
					c.runs[id] = run
				}
			}
			ob := &pb.EffortBoardRow{RuntimeState: "active"}
			before := proto.Clone(e)
			row := s.projectEffort(context.Background(), e, ob, &pb.EffortDiscovery{})
			if row.RuntimeState != tc.wantRuntime {
				t.Errorf("runtime = %q; want %q", row.RuntimeState, tc.wantRuntime)
			}
			if !strings.Contains(strings.Join(row.Limitations, "\n"), tc.wantDetail) {
				t.Errorf("coverage missing %q: %v", tc.wantDetail, row.Limitations)
			}
			if row.OutcomeStanding.GetState() != "unknown" {
				t.Errorf("run status inferred product acceptance: %v", row.OutcomeStanding)
			}
			if len(row.Assignments) != len(tc.subjects) {
				t.Fatalf("lost role assignments: got %d, want %d", len(row.Assignments), len(tc.subjects))
			}
			for i, a := range row.Assignments {
				if !proto.Equal(a.Subject, e.Subjects[i]) {
					t.Errorf("assignment %d lost its declared owner/role", i)
				}
				if !tc.subjects[i].missing && !tc.subjects[i].ended && !tc.subjects[i].foreign && a.RuntimeState != string(tc.subjects[i].status) {
					t.Errorf("assignment %d lost owner status: %s", i, a.RuntimeState)
				}
			}
			if !proto.Equal(e, before) || ob.RuntimeState != "active" || ob.OutcomeStanding != nil {
				t.Fatal("projection mutated enrollment or source observation")
			}
		})
	}
}

func TestEffortRuntimeAllRoleUsageAndOutcomeAttribution(t *testing.T) {
	s, _, c := effortFixture(t)
	e := &pb.EffortEnrollment{EffortRef: "effort:all-role-cost"}
	start, end := s.now().Add(-time.Minute), s.now()
	for _, role := range []string{"orchestrator", "worker", "supervisor", "reviewer", "investigator", "repair"} {
		id := uuid.New()
		c.runs[id] = &domain.Run{ID: id, Status: domain.RunStatusComplete, StartedAt: &start, EndedAt: &end}
		if role == "supervisor" {
			e.SupervisorRunId = id.String() // implicit assignment must also remain billable
			continue
		}
		e.Subjects = append(e.Subjects, &pb.EffortSubject{Owner: "agent-manager", Kind: "run", Reference: id.String(), RunId: id.String(), Role: role})
	}
	ob := &pb.EffortBoardRow{OutcomeStanding: &pb.EffortOutcomeStanding{State: "unaccepted", Attribution: "workspace self-report", EvidenceRefs: []string{"checkpoint:cut-1"}}}
	row := s.projectEffort(context.Background(), e, ob, &pb.EffortDiscovery{})
	if row.RuntimeState != "finished" || !proto.Equal(row.OutcomeStanding, ob.OutcomeStanding) {
		t.Fatal("terminal runs changed attributed outcome standing", row)
	}
	if len(row.Assignments) != 6 || row.Usage.DeclaredRuns != 6 || row.Usage.ObservedRuns != 6 || row.Usage.AgentSeconds == nil || *row.Usage.AgentSeconds != 360 {
		t.Fatal("all-role time accounting lost an execution", row.Usage)
	}
	if row.Usage.Tokens != nil || row.Usage.ReportedCostUsd != nil || !row.Usage.Partial {
		t.Fatal("unknown metering became zero/complete", row.Usage)
	}
	// Two owner records for one provider execution cannot double charge time,
	// or let a supervisor masquerade as a second product executor.
	c.runs[uuid.MustParse(e.Subjects[1].RunId)].SessionID = "shared"
	c.runs[uuid.MustParse(e.SupervisorRunId)].SessionID = "shared"
	row = s.projectEffort(context.Background(), e, ob, &pb.EffortDiscovery{})
	if row.Usage.ObservedRuns != 5 || row.Usage.AgentSeconds == nil || *row.Usage.AgentSeconds != 300 || row.RuntimeState != "unknown" {
		t.Fatal("shared role execution was double counted or implied finished product runtime", row)
	}
}

func TestEffortRuntimeSupervisorReferenceWithoutBusinessOwner(t *testing.T) {
	for _, status := range []domain.RunStatus{domain.RunStatusRunning, domain.RunStatusComplete} {
		t.Run(string(status), func(t *testing.T) {
			s, _, c := effortFixture(t)
			id := uuid.New()
			c.runs[id] = &domain.Run{ID: id, Status: status}
			e := &pb.EffortEnrollment{EffortRef: "effort:supervisor-only", SupervisorRunId: id.String()}
			row := s.projectEffort(context.Background(), e, &pb.EffortBoardRow{}, &pb.EffortDiscovery{})
			if row.RuntimeState != "unknown" || row.OutcomeStanding.GetState() != "unknown" {
				t.Fatal("standalone supervisor implied business execution or acceptance", row)
			}
			if len(row.Assignments) != 1 || row.Assignments[0].Subject.Role != "supervisor" || row.Assignments[0].RuntimeState != string(status) || row.Usage.ObservedRuns != 1 {
				t.Fatal("standalone supervisor lost its activity/accounting", row)
			}
		})
	}
}

func TestEffortRuntimeUnavailableOwnerIdentity(t *testing.T) {
	for _, variant := range []string{"controller unavailable", "invalid identity", "wrong returned identity"} {
		t.Run(variant, func(t *testing.T) {
			s, _, c := effortFixture(t)
			id := uuid.New()
			sub := &pb.EffortSubject{Owner: "agent-manager", Kind: "run", Reference: id.String(), RunId: id.String(), Role: "orchestrator"}
			c.runs[id] = &domain.Run{ID: id, Status: domain.RunStatusRunning}
			switch variant {
			case "controller unavailable":
				s.controller = nil
			case "invalid identity":
				sub.RunId = "not-a-run-id"
			case "wrong returned identity":
				c.runs[id].ID = uuid.New()
			}
			e := &pb.EffortEnrollment{EffortRef: "effort:missing-owner", Subjects: []*pb.EffortSubject{sub}}
			row := s.projectEffort(context.Background(), e, &pb.EffortBoardRow{}, &pb.EffortDiscovery{})
			if row.RuntimeState != "unknown" || len(row.Assignments) != 1 || row.Assignments[0].UnavailableReason == "" || row.Usage.ObservedRuns != 0 {
				t.Fatal("unavailable identity supplied business runtime or hid the assignment", row)
			}
		})
	}
}
