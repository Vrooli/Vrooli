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

func TestEffortDecisionsPageUsesExactEffortAndCursor(t *testing.T) {
	services, server := decisionServer(t, decisionFixture("pending", "", "", "", time.Time{}))
	app := &App{services: services}
	if err := app.cmdEffort([]string{"decisions-page", "--effort-ref", "effort:example", "--page-size", "7", "--page-token", "opaque-token", "--json"}); err != nil {
		t.Fatal(err)
	}
	requests := server.Requests()
	if len(requests) != 1 || requests[0].Path != apiconnect.AgentManagerServiceListEffortDecisionsProcedure {
		t.Fatalf("expected exactly one owner page read: %+v", requests)
	}
	var req pb.ListEffortDecisionsRequest
	if err := protojson.Unmarshal([]byte(requests[0].Body), &req); err != nil {
		t.Fatal(err)
	}
	if req.GetEffortRef() != "effort:example" || req.GetLimit() != 7 || req.GetPageToken() != "opaque-token" {
		t.Fatalf("wrong page request: %+v", &req)
	}
	if err := app.cmdEffort([]string{"decisions-page", "--json"}); err == nil {
		t.Fatal("an unscoped decision-page read must be refused")
	}
}

func TestDecisionRaiseDryRunContactsNoServerAndWritesNothing(t *testing.T) {
	home := copyBASGoalHome(t, nil)
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
	home := copyBASGoalHome(t, nil)
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
			home := copyBASGoalHome(t, nil)
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
	home := copyBASGoalHome(t, nil)
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
	if _, err := os.Stat(filepath.Join(home, "QUEUE.md")); !os.IsNotExist(err) {
		t.Fatal("the v2 decision writer must not create a QUEUE.md")
	}
}

func TestDecisionSyncRecordsAnswersVerbatimAndIsIdempotent(t *testing.T) {
	answered := decisionFixture(effortdecision.Answered, "close", "keep the\nmacOS journeys", effortdecision.ViaPhone, time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC))
	answered.RecordedAt = timestamppb.New(time.Date(2026, 10, 8, 10, 1, 0, 0, time.UTC))
	defaulted := decisionFixture(effortdecision.Defaulted, "re-aim", "", effortdecision.ViaDefault, time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC))
	defaulted.DecisionId = "BAS-D-071"
	home := copyBASGoalHome(t, nil)
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

func TestDecisionSyncRecordsDismissalWithoutAnswerOrApproval(t *testing.T) {
	dismissed := decisionFixture(effortdecision.Dismissed, "", "", effortdecision.ViaPhone, time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC))
	dismissed.ResolutionReason = "No longer applicable"
	dismissed.ResolvedAt = timestamppb.New(time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC))
	home := copyBASGoalHome(t, nil)
	queuePath := filepath.Join(home, goalhome.StatusFile)
	queue, _ := os.ReadFile(queuePath)
	queue, _ = goalhome.AddNeedsOperatorItem(queue, effortdecision.View(dismissed))
	writeFile(t, queuePath, string(queue))
	services, server := decisionServer(t, dismissed)

	output, code := runEffort(t, &App{services: services}, "decision", "sync", home, "--home-ref", basGoalHomeRef)
	if code != 0 {
		t.Fatalf("dismissal sync failed (exit %d):\n%s", code, output)
	}
	feedback, err := os.ReadFile(filepath.Join(home, goalhome.FeedbackFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(feedback), "dismissed without answer") || !strings.Contains(string(feedback), "No longer applicable") {
		t.Fatalf("sync must record dismissal reason without inventing an answer:\n%s", feedback)
	}
	queue, _ = os.ReadFile(queuePath)
	if strings.Contains(string(queue), "[decision BAS-D-070]") {
		t.Fatalf("dismissed decision remains under Needs operator:\n%s", queue)
	}
	var recordedAt string
	for _, request := range server.Requests() {
		if request.Path != apiconnect.AgentManagerServiceMarkEffortDecisionRecordedProcedure {
			continue
		}
		var mark pb.MarkEffortDecisionRecordedRequest
		if err := protojson.Unmarshal([]byte(request.Body), &mark); err != nil {
			t.Fatal(err)
		}
		recordedAt = mark.GetResolvedAt().AsTime().UTC().Format(time.RFC3339)
	}
	if recordedAt != "2026-10-10T09:00:00Z" {
		t.Fatalf("sync fence must use resolved_at for dismissals, got %q", recordedAt)
	}
}

func TestRoutineDecisionSyncDoesNotRecreateHandledArchivedFeedback(t *testing.T) {
	home := copyBASGoalHome(t, nil)
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
			home := copyBASGoalHome(t, nil)
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

func cohortFolder(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "cohorts", "cohort-one")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# A research cohort\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func cohortRaiseArgs(dir string, extra ...string) []string {
	return append([]string{
		"decision", "raise", dir, "--id", "COH-1", "--question", "Keep the second study?",
		"--option", "keep=Keep it", "--option", "drop=Drop it", "--recommend", "keep",
	}, extra...)
}

// A research cohort lives under ~/.vrooli, outside the repository, and has no
// STATUS.md. Neither is a reason to refuse a decision: it is a key and an event.
func TestDecisionRaiseWorksInAFolderOutsideTheRepositoryWithNoStatusFile(t *testing.T) {
	dir := cohortFolder(t)
	services, server := decisionServer(t, &pb.EffortDecision{GoalHome: dir, DecisionId: "COH-1", State: effortdecision.Pending})
	output, code := runEffort(t, &App{services: services}, cohortRaiseArgs(dir)...)
	if code != 0 {
		t.Fatalf("raise in a cohort folder failed (exit %d):\n%s", code, output)
	}
	requests := server.Requests()
	if len(requests) < 1 || requests[len(requests)-1].Path != apiconnect.AgentManagerServiceRaiseEffortDecisionProcedure {
		t.Fatalf("the decision must reach Agent Manager: %+v", requests)
	}
	var sent pb.RaiseEffortDecisionRequest
	if err := protojson.Unmarshal([]byte(requests[len(requests)-1].Body), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.GetDecision().GetGoalHome() != dir {
		t.Fatalf("an outside home is keyed by its absolute directory: %q, want %q", sent.GetDecision().GetGoalHome(), dir)
	}
	if _, err := os.Stat(filepath.Join(dir, goalhome.StatusFile)); !os.IsNotExist(err) {
		t.Fatalf("raising must not create a STATUS.md in a folder that has none: %v", err)
	}
	for _, want := range []string{"has no STATUS.md", "effort decision list"} {
		if !strings.Contains(output, want) {
			t.Fatalf("the output must say where the answer is read; lacks %q:\n%s", want, output)
		}
	}
}

func TestDecisionSyncRecordsAnAnswerInAFolderWithNoStatusFile(t *testing.T) {
	dir := cohortFolder(t)
	answered := &pb.EffortDecision{
		GoalHome: dir, DecisionId: "COH-1", Question: "Keep the second study?", State: effortdecision.Answered, Answer: "keep", AnsweredVia: effortdecision.ViaPhone,
		Options:     []*pb.EffortDecisionOption{{Key: "keep", Label: "Keep it"}, {Key: "drop", Label: "Drop it"}},
		Recommended: "keep", AnsweredAt: timestamppb.New(time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)), CreatedAt: timestamppb.New(time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)),
	}
	services, _ := decisionServer(t, answered)
	output, code := runEffort(t, &App{services: services}, "decision", "sync", dir)
	if code != 0 {
		t.Fatalf("sync in a cohort folder failed (exit %d):\n%s", code, output)
	}
	feedback, err := os.ReadFile(filepath.Join(dir, goalhome.FeedbackFile))
	if err != nil || !strings.Contains(string(feedback), "COH-1") {
		t.Fatalf("the answer must be recorded in FEEDBACK.md: %v\n%s", err, feedback)
	}
	if _, err := os.Stat(filepath.Join(dir, goalhome.StatusFile)); !os.IsNotExist(err) {
		t.Fatalf("sync must not create a STATUS.md: %v", err)
	}
}

func TestDecisionRaiseNamesTheRevisionWhenAPendingDecisionIsChanged(t *testing.T) {
	dir := cohortFolder(t)
	before := &pb.EffortDecision{GoalHome: dir, DecisionId: "COH-1", State: effortdecision.Pending, EventId: "event-1", Question: "Keep the second study?"}
	after := &pb.EffortDecision{GoalHome: dir, DecisionId: "COH-1", State: effortdecision.Pending, EventId: "event-2", Question: "Keep the second study, given the new data?"}
	server := clitest.NewRecordingServerForRequests(t, func(r clitest.Request) string {
		var message proto.Message = &pb.EffortDecision{}
		switch r.Path {
		case apiconnect.AgentManagerServiceListEffortDecisionsProcedure:
			message = &pb.ListEffortDecisionsResponse{Decisions: []*pb.EffortDecision{before}}
		case apiconnect.AgentManagerServiceRaiseEffortDecisionProcedure:
			message = &pb.RaiseEffortDecisionResponse{Decision: after, Published: true, Revised: true}
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
	output, code := runEffort(t, &App{services: NewServices(api)}, cohortRaiseArgs(dir)...)
	if code != 0 {
		t.Fatalf("revising failed (exit %d):\n%s", code, output)
	}
	for _, want := range []string{"revised in place", "updates the operator's ask in place"} {
		if !strings.Contains(output, want) {
			t.Fatalf("the output must say the decision was revised; lacks %q:\n%s", want, output)
		}
	}
}

func TestDecisionRaiseExplainsADefaultThatIsNotReversible(t *testing.T) {
	dir := cohortFolder(t)
	services, server := decisionServer(t)
	err := (&App{services: services}).cmdEffort(cohortRaiseArgs(dir, "--default", "keep"))
	if err == nil || !strings.Contains(err.Error(), "--default needs --reversible") || len(server.Requests()) != 0 {
		t.Fatalf("a default without --reversible must be refused in plain words before any request: %v (%d requests)", err, len(server.Requests()))
	}
}

func TestDecisionLineShowsACohortHomeUnderTheUsersHomeReadably(t *testing.T) {
	user := t.TempDir()
	t.Setenv("HOME", user)
	for home, want := range map[string]string{
		filepath.Join(user, ".vrooli", "research", "cohort-1"): "COH-1  ~/.vrooli/research/cohort-1  pending",
		"/srv/elsewhere/cohort-2":                              "COH-1  /srv/elsewhere/cohort-2  pending",
		"scenarios/agent-manager":                              "COH-1  scenarios/agent-manager  pending",
		user + "-sibling/cohort":                               "COH-1  " + user + "-sibling/cohort  pending",
	} {
		line := decisionLine(&pb.EffortDecision{GoalHome: home, DecisionId: "COH-1", State: effortdecision.Pending, Question: "Keep it?"})
		if !strings.HasPrefix(line, want) {
			t.Errorf("home %q: line %q does not start with %q", home, line, want)
		}
	}
}
