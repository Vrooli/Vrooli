package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	clitest "agent-manager/cli/internal/testutil"
	"agent-manager/internal/effortdecision"
	"agent-manager/internal/goalhome"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const basGoalHomeRef = "scenarios/browser-automation-studio/docs/internal/goal"

func decisionFixture(state, answer, note, via string, answeredAt time.Time) *pb.EffortDecision {
	d := &pb.EffortDecision{
		GoalHome: basGoalHomeRef, DecisionId: "BAS-D-070", Question: "Close the BAS goal or re-aim at quality?",
		Options:     []*pb.EffortDecisionOption{{Key: "re-aim", Label: "Re-aim at quality"}, {Key: "close", Label: "Close the goal"}},
		Recommended: "re-aim", DefaultAnswer: "re-aim", Reversible: true, State: state, Answer: answer, AnswerNote: note, AnsweredVia: via,
		CreatedAt: timestamppb.New(time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)),
	}
	if !answeredAt.IsZero() {
		d.AnsweredAt = timestamppb.New(answeredAt)
	}
	return d
}

// decisionServer answers decision RPCs from fixed records.
func decisionServer(t *testing.T, decisions ...*pb.EffortDecision) (*Services, *clitest.RecordingServer) {
	t.Helper()
	server := clitest.NewRecordingServerForRequests(t, func(r clitest.Request) string {
		var message proto.Message = &pb.EffortDecision{}
		switch r.Path {
		case apiconnect.AgentManagerServiceListEffortDecisionsProcedure:
			message = &pb.ListEffortDecisionsResponse{Decisions: decisions}
		case apiconnect.AgentManagerServiceRaiseEffortDecisionProcedure:
			message = &pb.RaiseEffortDecisionResponse{Decision: decisions[0], Created: true, Published: true}
		}
		body, err := protojson.Marshal(message)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	return NewServices(api), server
}

func TestDecisionRaiseDryRunContactsNoServerAndWritesNothing(t *testing.T) {
	home := copyRuntimeBASGoalHome(t, nil)
	before, _ := os.ReadFile(filepath.Join(home, goalhome.StatusFile))
	services, server := decisionServer(t)
	output, code := runEffort(t, &App{services: services}, "decision", "raise", home, "--home-ref", basGoalHomeRef, "--id", "BAS-D-070",
		"--question", "Close the BAS goal or re-aim at quality?", "--option", "re-aim=Re-aim at quality", "--option", "close=Close the goal",
		"--recommend", "re-aim", "--default", "re-aim", "--reversible", "--deadline", "24h",
		"--big-picture", "This changes whether the original quality goal is acceptable.",
		"--technical-details", "The baseline artifact is absent and cannot be reconstructed.", "--dry-run")
	if code != 0 {
		t.Fatalf("dry run failed (exit %d):\n%s", code, output)
	}
	for _, want := range []string{"DRY RUN", goalhome.DecisionRequestedEventType, `"recommended": "re-aim"`, `"big_picture": "This changes whether the original quality goal is acceptable."`, `"technical_details": "The baseline artifact is absent and cannot be reconstructed."`, "[decision BAS-D-070] Close the BAS goal or re-aim at quality?", "`re-aim` Re-aim at quality (recommended)"} {
		if !strings.Contains(output, want) {
			t.Fatalf("dry run omitted %q:\n%s", want, output)
		}
	}
	after, _ := os.ReadFile(filepath.Join(home, goalhome.StatusFile))
	if string(after) != string(before) || len(server.Requests()) != 0 {
		t.Fatalf("a dry run wrote QUEUE.md or contacted the server (%d requests)", len(server.Requests()))
	}
}

func TestDecisionRaiseAddsTheNeedsOperatorItemOnce(t *testing.T) {
	home := copyRuntimeBASGoalHome(t, nil)
	services, _ := decisionServer(t, decisionFixture(effortdecision.Pending, "", "", "", time.Time{}))
	args := []string{
		"decision", "raise", home, "--home-ref", basGoalHomeRef, "--id", "BAS-D-070", "--question", "Close the BAS goal or re-aim at quality?",
		"--option", "re-aim=Re-aim at quality", "--option", "close=Close the goal", "--recommend", "re-aim",
	}
	for range 2 {
		if output, code := runEffort(t, &App{services: services}, args...); code != 0 {
			t.Fatalf("raise failed (exit %d):\n%s", code, output)
		}
	}
	queue, _ := os.ReadFile(filepath.Join(home, goalhome.StatusFile))
	if n := strings.Count(string(queue), "[decision BAS-D-070]"); n != 1 {
		t.Fatalf("a repeated raise must list the decision once under ## Needs operator, found %d:\n%s", n, queue)
	}
}

func TestDecisionReraiseDoesNotReopenAnsweredHumanDependency(t *testing.T) {
	for _, state := range []string{effortdecision.Answered, effortdecision.Defaulted} {
		t.Run(state, func(t *testing.T) {
			home := copyV2BASGoalHome(t, nil)
			d := decisionFixture(state, "close", "retain this answer", effortdecision.ViaPhone, time.Now())
			services, _ := decisionServer(t, d)
			path := filepath.Join(home, goalhome.StatusFile)
			queue, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			queue, _ = goalhome.AddNeedsOperatorItem(queue, effortdecision.View(d))
			writeFile(t, path, string(queue))
			for range 2 {
				output, code := runEffort(t, &App{services: services}, "decision", "raise", home,
					"--home-ref", basGoalHomeRef, "--id", d.DecisionId, "--question", d.Question,
					"--option", "re-aim=Re-aim at quality", "--option", "close=Close the goal", "--recommend", "re-aim")
				if code != 0 {
					t.Fatalf("raise: %s", output)
				}
			}
			queue, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(queue), "[decision BAS-D-070]") {
				t.Fatalf("answered decision recreated a human dependency: %s", queue)
			}
		})
	}
}

// TestDecisionRaiseTargetsStatusForV2Home proves the decision writer appends the
// ## Needs operator item to STATUS.md (not QUEUE.md) for a Phase-2 v2 home.
func TestDecisionRaiseTargetsStatusForV2Home(t *testing.T) {
	home := copyV2BASGoalHome(t, nil)
	services, _ := decisionServer(t, decisionFixture(effortdecision.Pending, "", "", "", time.Time{}))
	args := []string{
		"decision", "raise", home, "--home-ref", basGoalHomeRef, "--id", "BAS-D-070", "--question", "Close the BAS goal or re-aim at quality?",
		"--option", "re-aim=Re-aim at quality", "--option", "close=Close the goal", "--recommend", "re-aim",
	}
	for range 2 {
		if output, code := runEffort(t, &App{services: services}, args...); code != 0 {
			t.Fatalf("raise failed (exit %d):\n%s", code, output)
		}
	}
	status, _ := os.ReadFile(filepath.Join(home, goalhome.StatusFile))
	if n := strings.Count(string(status), "[decision BAS-D-070]"); n != 1 {
		t.Fatalf("a repeated raise must list the decision once under STATUS.md ## Needs operator, found %d:\n%s", n, status)
	}
	if _, err := os.Stat(filepath.Join(home, goalhome.QueueFile)); !os.IsNotExist(err) {
		t.Fatal("the v2 decision writer must not create a QUEUE.md")
	}
}

func TestDecisionSyncRecordsAnswersVerbatimAndIsIdempotent(t *testing.T) {
	answered := decisionFixture(effortdecision.Answered, "close", "keep the\nmacOS journeys", effortdecision.ViaPhone, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
	answered.RecordedAt = timestamppb.New(time.Date(2026, 10, 8, 10, 1, 0, 0, time.UTC))
	defaulted := decisionFixture(effortdecision.Defaulted, "re-aim", "", effortdecision.ViaDefault, time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC))
	defaulted.DecisionId = "BAS-D-071"
	home := copyRuntimeBASGoalHome(t, nil)
	queuePath := filepath.Join(home, goalhome.StatusFile)
	queue, _ := os.ReadFile(queuePath)
	for _, d := range []*pb.EffortDecision{answered, defaulted} {
		queue, _ = goalhome.AddNeedsOperatorItem(queue, effortdecision.View(d))
	}
	writeFile(t, queuePath, string(queue))
	services, server := decisionServer(t, defaulted, answered)

	for range 2 {
		if output, code := runEffort(t, &App{services: services}, "decision", "sync", home, "--home-ref", basGoalHomeRef, "--reconcile-recorded"); code != 0 {
			t.Fatalf("sync failed (exit %d):\n%s", code, output)
		}
	}
	feedback, _ := os.ReadFile(filepath.Join(home, goalhome.FeedbackFile))
	for _, want := range []string{
		"### BAS-D-070 — operator decision (2026-10-08T10:00:00Z, operator — answered via phone)",
		"> close — Close the goal\n> keep the\n> macOS journeys\n",
		"### BAS-D-071 — operator decision (2026-10-09T09:00:00Z, operator — applied by default)",
		"> re-aim — Re-aim at quality; applied by default at the deadline with no operator answer",
	} {
		if strings.Count(string(feedback), want) != 1 {
			t.Fatalf("FEEDBACK.md must hold %q exactly once:\n%s", want, feedback)
		}
	}
	queue, _ = os.ReadFile(queuePath)
	if strings.Contains(string(queue), "[decision BAS-D-07") {
		t.Fatalf("sync left a ## Needs operator item:\n%s", queue)
	}
	marks := 0
	for _, r := range server.Requests() {
		if r.Path == apiconnect.AgentManagerServiceListEffortDecisionsProcedure {
			var req pb.ListEffortDecisionsRequest
			if err := protojson.Unmarshal([]byte(r.Body), &req); err != nil {
				t.Fatal(err)
			}
			if req.UnrecordedOnly {
				t.Fatal("sync must restore answers already projected at a previous home")
			}
		}
		if r.Path == apiconnect.AgentManagerServiceMarkEffortDecisionRecordedProcedure {
			marks++
		}
	}
	if marks != 2 {
		t.Fatalf("only unrecorded answers need a native mark; prior recorded answers are re-projected locally: %d marks", marks)
	}
}

func TestRoutineDecisionSyncDoesNotRecreateHandledArchivedFeedback(t *testing.T) {
	home := copyV2BASGoalHome(t, nil)
	d := decisionFixture(effortdecision.Answered, "close", "handled before archival", effortdecision.ViaPhone, time.Now())
	d.RecordedAt = timestamppb.Now()
	services, server := decisionServer(t, d)
	feedbackPath := filepath.Join(home, goalhome.FeedbackFile)
	before, err := os.ReadFile(feedbackPath)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		output, code := runEffort(t, &App{services: services}, "decision", "sync", home, "--home-ref", basGoalHomeRef)
		if code != 0 {
			t.Fatalf("sync: %s", output)
		}
	}
	after, err := os.ReadFile(feedbackPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("routine sync recreated archived feedback: %s", after)
	}
	for _, r := range server.Requests() {
		if r.Path == apiconnect.AgentManagerServiceMarkEffortDecisionRecordedProcedure {
			t.Fatal("routine sync rewrote prior recording")
		}
	}
}

func TestDecisionSyncReadsOlderRecordedPolicyAndRefusesAmbiguityBeforeWrites(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprint(conflict), func(t *testing.T) {
			home := copyRuntimeBASGoalHome(t, nil)
			feedbackPath, statusPath := filepath.Join(home, goalhome.FeedbackFile), filepath.Join(home, goalhome.StatusFile)
			beforeFeedback, _ := os.ReadFile(feedbackPath)
			beforeStatus, _ := os.ReadFile(statusPath)
			var first []*pb.EffortDecision
			for i := range 200 {
				d := decisionFixture(effortdecision.Pending, "", "", "", time.Time{})
				d.DecisionId = fmt.Sprintf("new-%03d", i)
				first = append(first, d)
			}
			old := decisionFixture(effortdecision.Answered, "close", "original recorded policy", "phone", time.Now())
			old.DecisionId, old.RecordedAt = "OLDER-POLICY", timestamppb.Now()
			if conflict {
				first[0] = proto.Clone(old).(*pb.EffortDecision)
				first[0].Answer = "re-aim"
			}
			var pages int
			server := clitest.NewRecordingServerForRequests(t, func(r clitest.Request) string {
				if r.Path != apiconnect.AgentManagerServiceListEffortDecisionsProcedure {
					t.Errorf("unexpected native write: %s", r.Path)
				}
				request := &pb.ListEffortDecisionsRequest{}
				if err := protojson.Unmarshal([]byte(r.Body), request); err != nil {
					t.Error(err)
				}
				response := &pb.ListEffortDecisionsResponse{}
				if request.PageToken == "" {
					response.Decisions, response.NextPageToken = first, "older-page"
				} else {
					if request.PageToken != "older-page" {
						t.Errorf("wrong continuation: %q", request.PageToken)
					}
					response.Decisions = []*pb.EffortDecision{old}
				}
				pages++
				body, err := protojson.Marshal(response)
				if err != nil {
					t.Fatal(err)
				}
				return string(body)
			})
			api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL()} }, nil)
			app := &App{services: NewServices(api)}
			var output string
			var code int
			if conflict {
				if err := app.decisionSync([]string{home, "--home-ref", basGoalHomeRef, "--reconcile-recorded"}); err != nil {
					code, output = 1, err.Error()
				}
			} else {
				output, code = runEffort(t, app, "decision", "sync", home, "--home-ref", basGoalHomeRef, "--reconcile-recorded")
			}
			afterFeedback, _ := os.ReadFile(feedbackPath)
			afterStatus, _ := os.ReadFile(statusPath)
			if pages != 2 {
				t.Fatalf("did not enumerate older policy: pages=%d output=%s", pages, output)
			}
			if conflict {
				if code == 0 || !strings.Contains(output, "ambiguous decision") || string(beforeFeedback) != string(afterFeedback) || string(beforeStatus) != string(afterStatus) {
					t.Fatalf("ambiguous answers wrote local policy: exit=%d output=%s", code, output)
				}
			} else if code != 0 || !strings.Contains(string(afterFeedback), "OLDER-POLICY") || !strings.Contains(string(afterFeedback), "original recorded policy") {
				t.Fatalf("old recorded policy lost: exit=%d output=%s", code, output)
			}
		})
	}
}
