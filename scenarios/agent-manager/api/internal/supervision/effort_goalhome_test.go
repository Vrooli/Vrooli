package supervision

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

const goalHomeRef = "scenarios/browser-automation-studio/docs/internal/goal"

// fakeSpend meters the runs it knows; any other run is unmetered.
type fakeSpend struct {
	weighted map[uuid.UUID]float64
	limit    int64
}

func (f fakeSpend) WeightedRunTokens(_ context.Context, id uuid.UUID) (float64, bool, error) {
	value, ok := f.weighted[id]
	return value, ok, nil
}

func (f fakeSpend) TokenCapUsage(_ context.Context, id uuid.UUID) (*float64, int64, string) {
	if value, ok := f.weighted[id]; ok {
		return &value, f.limit, "report"
	}
	return nil, f.limit, "report"
}

// goalHomeTree copies the trimmed BAS goal home under root/goalHomeRef,
// applying edits by file name.
func goalHomeTree(t *testing.T, root string, edits map[string]func(string) string) string {
	t.Helper()
	source, home := filepath.Join("..", "goalhome", "testdata", "bas"), filepath.Join(root, filepath.FromSlash(goalHomeRef))
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		name, _ := filepath.Rel(source, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		if edit := edits[name]; edit != nil {
			text = edit(text)
		}
		if err := os.MkdirAll(filepath.Join(home, filepath.Dir(name)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(home, name), []byte(text), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, edit := range edits {
		if _, err := os.Stat(filepath.Join(home, name)); os.IsNotExist(err) {
			if err := os.WriteFile(filepath.Join(home, name), []byte(edit("")), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return home
}

// goalHomeBoardRow registers the goal home on an enrollment and reads its row.
func goalHomeBoardRow(t *testing.T, s *EffortService, c *fakeActionController) *pb.EffortBoardRow {
	t.Helper()
	ctx := context.Background()
	e := enrollmentFixture(t, s, c, domain.RunStatusRunning)
	e.DestinationRef = "path:" + goalHomeRef + "/GOAL.md"
	if _, err := s.ReconcileMetadata(ctx, &pb.ReconcileEffortMetadataRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "goal-home"}, EffortActor{ID: "owner", Operator: true}); err != nil {
		t.Fatal(err)
	}
	board, err := s.Board(ctx, &pb.GetEffortBoardRequest{EffortRef: e.EffortRef})
	if err != nil || len(board.Rows) != 1 {
		t.Fatalf("board: %v %v", board, err)
	}
	return board.Rows[0]
}

func TestEffortGoalHomeProjectsRegisteredHome(t *testing.T) {
	s, _, c := effortFixture(t)
	s.config.ProjectRoot = t.TempDir()
	measured, active := uuid.MustParse("a7240424-43e7-4d36-87f6-2b2e83d0c959"), uuid.New()
	goalHomeTree(t, s.config.ProjectRoot, map[string]func(string) string{
		"GOAL.md": func(text string) string {
			title, rest, _ := strings.Cut(text, "\n")
			return title + "\n\n- Destination: journeys >= 24\n- Current: 12 @ 2026-10-07\n" + rest
		},
		"QUEUE.md": func(text string) string {
			return strings.Replace(text, "(none; the operator answered BAS-FB-063 on 2026-10-06)", "- BAS-D-070 Close the goal or re-aim at quality?", 1)
		},
		"epochs/E27.md": func(string) string {
			return "# E27\n\n- Status: in-progress\n- Workers: " + active.String() + " (task 1; worker)\n\n## Slice log\n"
		},
	})
	c.runs[measured] = &domain.Run{ID: measured, Status: domain.RunStatusComplete}
	c.runs[active] = &domain.Run{ID: active, Status: domain.RunStatusRunning}
	s.SetSpendReader(fakeSpend{weighted: map[uuid.UUID]float64{measured: 640000, active: 120000}, limit: 1000000})

	home := goalHomeBoardRow(t, s, c).GetGoalHome()
	if home == nil || home.State != "available" || home.Path != goalHomeRef {
		t.Fatalf("goal home not projected: %+v", home)
	}
	if home.DestinationMetric != "journeys" || home.DestinationOp != ">=" || home.GetDestinationTarget() != 24 || home.GetCurrentValue() != 12 {
		t.Fatalf("destination facts: %+v", home)
	}
	if home.ActiveEpoch != "E27" || home.ActiveStatus != "in-progress" || len(home.ActiveWorkers) != 1 || home.ActiveWorkers[0] != active.String() {
		t.Fatalf("active epoch facts: %+v", home)
	}
	if home.LastAcceptedEpoch != "E25" || home.LastAcceptedAt == nil {
		t.Fatalf("last acceptance: %+v", home)
	}
	if len(home.NeedsOperator) != 1 || !strings.Contains(home.NeedsOperator[0], "BAS-D-070") {
		t.Fatalf("needs operator: %q", home.NeedsOperator)
	}
	if home.OpenWorkarounds == nil || home.GetOpenWorkarounds() != 3 || len(home.OpenFeedbackIds) == 0 {
		t.Fatalf("workaround and feedback facts: %+v", home)
	}
	// Of the accepted E23-E25 only E25's worker is metered, so only E25 is measured.
	if home.AcceptedEpochsMeasured != 1 || home.GetWeightedTokensPerAcceptedEpoch() != 640000 {
		t.Fatalf("spend per accepted epoch: measured=%d mean=%v", home.AcceptedEpochsMeasured, home.WeightedTokensPerAcceptedEpoch)
	}
	if !strings.Contains(strings.Join(home.Limitations, "; "), "unmetered or unknown worker") {
		t.Fatalf("an unmeasured accepted epoch must be named: %q", home.Limitations)
	}
	var capped *pb.EffortCapUsage
	for _, usage := range home.CapUsage {
		if usage.RunId == active.String() {
			capped = usage
		}
	}
	if capped == nil || capped.Role != "worker" || capped.GetWeightedTokens() != 120000 || capped.GetCap() != 1000000 || capped.Mode != "report" {
		t.Fatalf("active worker cap usage: %+v", home.CapUsage)
	}
	if home.CanaryStatus != "unknown" {
		t.Fatalf("canary status = %q", home.CanaryStatus)
	}
}

func TestEffortGoalHomeProjectsActiveEpochIndependentlyOfLatestAcceptance(t *testing.T) {
	s, _, c := effortFixture(t)
	s.config.ProjectRoot = t.TempDir()
	home := filepath.Join(s.config.ProjectRoot, filepath.FromSlash(goalHomeRef))
	epochs := filepath.Join(home, "epochs")
	for _, dir := range []string{home, epochs} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"GOAL.md":        "# Goal\n\n- Destination: journeys >= 24\n- Current: 12 @ 2026-10-07\n",
		"QUEUE.md":       "# Queue\n\n## Handoff\n\nContinue the admitted work.\n",
		"FEEDBACK.md":    "# Feedback\n\n## Resolved\n",
		"WORKAROUNDS.md": "# Workarounds\n\n## Resolved\n",
		"epochs/E1.md":   "# E1\n\n- Status: active — D35 same-run correction continuation admitted\n- Workers: 11111111-1111-4111-8111-111111111111\n\n## Slice log\n",
		"epochs/E2.md":   "# E2\n\n- Status: A190 validation complete; G1 failed, E2 remains unaccepted\n\n## Slice log\n",
		"epochs/E3.md":   "# E3\n\n- Status: accepted by the original owner\n\n## Slice log\nACCEPTED 2026-10-18T10:03:08Z delivered; gates=G1,G2,G3\n",
	}
	for name, data := range files {
		path := filepath.Join(home, filepath.FromSlash(name))
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.ActiveEpoch != "E1" || facts.ActiveStatus != "active — D35 same-run correction continuation admitted" {
		t.Fatalf("the explicit active epoch must remain visible beside a later accepted epoch: %+v", facts)
	}
	if facts.LastAcceptedEpoch != "E3" || facts.LastAcceptedAt == nil {
		t.Fatalf("latest acceptance must be reported independently: %+v", facts)
	}
	if len(facts.ActiveWorkers) != 1 || facts.ActiveWorkers[0] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("workers must come from the selected active epoch: %q", facts.ActiveWorkers)
	}
}

func TestEffortGoalHomeProjectsReopenedEpochDespiteStaleAcceptedStatus(t *testing.T) {
	s, _, c := effortFixture(t)
	s.config.ProjectRoot = t.TempDir()
	home := filepath.Join(s.config.ProjectRoot, filepath.FromSlash(goalHomeRef))
	epochs := filepath.Join(home, "epochs")
	for _, dir := range []string{home, epochs} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"GOAL.md":        "# Goal\n\n- Destination: journeys >= 24\n- Current: 12 @ 2026-10-07\n",
		"QUEUE.md":       "# Queue\n\n## Handoff\n\nContinue the reopened work.\n",
		"FEEDBACK.md":    "# Feedback\n\n## Resolved\n",
		"WORKAROUNDS.md": "# Workarounds\n\n## Resolved\n",
		"epochs/E1.md":   "# E1\n\n- Status: in-progress\n\n## Slice log\n",
		"epochs/E2.md":   "# E2\n\n- Status: A190 validation complete; G1 failed, E2 remains unaccepted\n\n## Slice log\n",
		"epochs/E3.md":   "# E3\n\n- Status: accepted by the original owner\n\n## Directives\n- D1 2026-10-18T10:10:00Z resume after new directive\n\n## Slice log\nACCEPTED 2026-10-18T10:03:08Z delivered; gates=G1,G2,G3\n",
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(home, filepath.FromSlash(name)), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.ActiveEpoch != "E3" || facts.ActiveStatus != "accepted by the original owner" {
		t.Fatalf("a later directive must reopen E3 even when its status text is stale: %+v", facts)
	}
	if facts.LastAcceptedEpoch != "" || facts.LastAcceptedAt != nil {
		t.Fatalf("reopened E3 must not remain the latest accepted epoch: %+v", facts)
	}
	if len(facts.ActiveWorkers) != 0 {
		t.Fatalf("workers must come only from the selected reopened epoch: %q", facts.ActiveWorkers)
	}
}

func TestEffortGoalHomeNewestActiveEpochWinsReopenedTie(t *testing.T) {
	s, _, c := effortFixture(t)
	s.config.ProjectRoot = t.TempDir()
	home := filepath.Join(s.config.ProjectRoot, filepath.FromSlash(goalHomeRef))
	epochs := filepath.Join(home, "epochs")
	for _, dir := range []string{home, epochs} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"GOAL.md":        "# Goal\n\n- Destination: journeys >= 24\n- Current: 12 @ 2026-10-07\n",
		"QUEUE.md":       "# Queue\n\n## Handoff\n\nContinue the current work.\n",
		"FEEDBACK.md":    "# Feedback\n\n## Resolved\n",
		"WORKAROUNDS.md": "# Workarounds\n\n## Resolved\n",
		"epochs/E1.md":   "# E1\n\n- Status: accepted by the original owner\n\n## Directives\n- D1 2026-10-18T10:10:00Z resume after new directive\n\n## Slice log\nACCEPTED 2026-10-18T10:03:08Z delivered; gates=G1,G2,G3\n",
		"epochs/E2.md":   "# E2\n\n- Status: in-progress\n\n## Slice log\n",
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(home, filepath.FromSlash(name)), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.ActiveEpoch != "E2" || facts.ActiveStatus != "in-progress" {
		t.Fatalf("newer explicitly active E2 must win the shared active-priority tie: %+v", facts)
	}
	if facts.LastAcceptedEpoch != "" {
		t.Fatalf("reopened E1 must not be reported as accepted: %+v", facts)
	}
}

func TestEffortGoalHomeLeavesProseWorkersUnknown(t *testing.T) {
	s, _, c := effortFixture(t)
	s.config.ProjectRoot = t.TempDir()
	goalHomeTree(t, s.config.ProjectRoot, map[string]func(string) string{
		"epochs/E27.md": func(string) string {
			return "# E27\n\n- Status: in-progress\n- Workers: attempt 1 task 8e9bda82-a42e-4bbe-aec8-9ac0a6957122 / run 7e762c65-72ee-413c-b051-e498d384baaa\n\n## Slice log\n"
		},
	})

	home := goalHomeBoardRow(t, s, c).GetGoalHome()
	if home == nil || home.ActiveEpoch != "E27" || len(home.ActiveWorkers) != 0 {
		t.Fatalf("prose must not become active workers: %+v", home)
	}
	if !strings.Contains(strings.Join(home.Limitations, "; "), "Workers field is not a well-formed UUID list") {
		t.Fatalf("malformed worker field must remain an explicit limitation: %q", home.Limitations)
	}
}

func TestEffortGoalHomeMissingFactsReadUnknownNotZero(t *testing.T) {
	s, _, c := effortFixture(t)
	s.config.ProjectRoot = t.TempDir()
	home := filepath.Join(s.config.ProjectRoot, filepath.FromSlash(goalHomeRef))
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "QUEUE.md"), []byte("# Queue\n\n## Needs operator\n\n(none)\n\n## Handoff\n\nStart E1.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.State != "available" {
		t.Fatalf("goal home: %+v", facts)
	}
	if facts.DestinationTarget != nil || facts.CurrentValue != nil || facts.OpenWorkarounds != nil || facts.WeightedTokensPerAcceptedEpoch != nil || facts.LastAcceptedEpoch != "" {
		t.Fatalf("missing facts must stay unknown, not zero: %+v", facts)
	}
	limitations := strings.Join(facts.Limitations, "; ")
	for _, want := range []string{"GOAL.md is missing", "WORKAROUNDS.md is missing", "destination unknown", "spend per accepted epoch unknown"} {
		if !strings.Contains(limitations, want) {
			t.Fatalf("limitations omit %q: %s", want, limitations)
		}
	}
}

func TestEffortGoalHomeRefusesSymlinkedHome(t *testing.T) {
	s, _, c := effortFixture(t)
	s.config.ProjectRoot = t.TempDir()
	outside := t.TempDir()
	goalHomeTree(t, outside, nil)
	link := filepath.Join(s.config.ProjectRoot, filepath.FromSlash(goalHomeRef))
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, filepath.FromSlash(goalHomeRef)), link); err != nil {
		t.Fatal(err)
	}
	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.State != "unavailable" || facts.ActiveEpoch != "" || len(facts.NeedsOperator) != 0 {
		t.Fatalf("a symlinked goal home must read unavailable: %+v", facts)
	}
}

func TestGoalHomeDestinationAcceptsOnlyRepoRelativeGoalFiles(t *testing.T) {
	for raw, want := range map[string]string{
		"path:" + goalHomeRef + "/GOAL.md":          goalHomeRef,
		"path:" + goalHomeRef + "/GOAL.md#frontier": goalHomeRef,
		"path:" + goalHomeRef + "/QUEUE.md":         "",
		"path:/etc/GOAL.md":                         "",
		"path:../outside/GOAL.md":                   "",
		"doc:target":                                "",
	} {
		got, ok := goalHomeDestination(raw)
		if got != want || ok != (want != "") {
			t.Fatalf("%s: got %q %v, want %q", raw, got, ok, want)
		}
	}
}
