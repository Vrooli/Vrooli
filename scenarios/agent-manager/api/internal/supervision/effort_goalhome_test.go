package supervision

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/goalhome"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
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
	migrateGoalHomeFixture(t, home)
	return home
}

func migrateGoalHomeFixture(t *testing.T, home string) {
	t.Helper()
	v1, err := goalhome.LoadLegacy(os.DirFS(home))
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte { b, _ := os.ReadFile(filepath.Join(home, name)); return b }
	conv := goalhome.ConvertHome(v1, read(goalhome.GoalFile), read(goalhome.QueueFile), read(goalhome.WorkaroundsFile))
	if !conv.Equivalent {
		t.Fatal(conv.Divergence)
	}
	for name, body := range map[string][]byte{goalhome.CharterFile: conv.Charter, goalhome.StatusFile: conv.Status, goalhome.LogFile: conv.Log} {
		if err := os.WriteFile(filepath.Join(home, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
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
	if home.ActiveUnit != "E27" || home.ActiveStatus != "in-progress" || len(home.ActiveWorkers) != 1 || home.ActiveWorkers[0] != active.String() {
		t.Fatalf("active epoch facts: %+v", home)
	}
	if home.LastAcceptedUnit != "E25" || home.LastAcceptedAt == nil {
		t.Fatalf("last acceptance: %+v", home)
	}
	if len(home.NeedsOperator) != 1 || !strings.Contains(home.NeedsOperator[0], "BAS-D-070") {
		t.Fatalf("needs operator: %q", home.NeedsOperator)
	}
	if home.OpenWorkarounds == nil || home.GetOpenWorkarounds() != 3 || len(home.OpenFeedbackIds) == 0 {
		t.Fatalf("workaround and feedback facts: %+v", home)
	}
	// Of the accepted E23-E25 only E25's worker is metered, so only E25 is measured.
	if home.AcceptedUnitsMeasured != 1 || home.GetWeightedTokensPerAcceptedUnit() != 640000 {
		t.Fatalf("spend per accepted epoch: measured=%d mean=%v", home.AcceptedUnitsMeasured, home.WeightedTokensPerAcceptedUnit)
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

	migrateGoalHomeFixture(t, home)
	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.ActiveUnit != "E1" || facts.ActiveStatus != "active — D35 same-run correction continuation admitted" {
		t.Fatalf("the explicit active epoch must remain visible beside a later accepted epoch: %+v", facts)
	}
	if facts.LastAcceptedUnit != "E3" || facts.LastAcceptedAt == nil {
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

	migrateGoalHomeFixture(t, home)
	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.ActiveUnit != "E3" || facts.ActiveStatus != "accepted by the original owner" {
		t.Fatalf("a later directive must reopen E3 even when its status text is stale: %+v", facts)
	}
	if facts.LastAcceptedUnit != "" || facts.LastAcceptedAt != nil {
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

	migrateGoalHomeFixture(t, home)
	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.ActiveUnit != "E2" || facts.ActiveStatus != "in-progress" {
		t.Fatalf("newer explicitly active E2 must win the shared active-priority tie: %+v", facts)
	}
	if facts.LastAcceptedUnit != "" {
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
	if home == nil || home.ActiveUnit != "E27" || len(home.ActiveWorkers) != 0 {
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
	if err := os.WriteFile(filepath.Join(home, "STATUS.md"), []byte("# Queue\n\n## Needs operator\n\n(none)\n\n## Handoff\n\nStart E1.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.State != "partial" {
		t.Fatalf("goal home: %+v", facts)
	}
	if facts.DestinationTarget != nil || facts.CurrentValue != nil || facts.OpenWorkarounds != nil || facts.WeightedTokensPerAcceptedUnit != nil || facts.LastAcceptedUnit != "" {
		t.Fatalf("missing facts must stay unknown, not zero: %+v", facts)
	}
	limitations := strings.Join(facts.Limitations, "; ")
	for _, want := range []string{"CHARTER.md is missing", "LOG.md is missing", "destination unknown", "epoch-derived spend and cap facts are unknown"} {
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
	if facts == nil || facts.State != "unavailable" || facts.ActiveUnit != "" || len(facts.NeedsOperator) != 0 {
		t.Fatalf("a symlinked goal home must read unavailable: %+v", facts)
	}
}

func TestEffortGoalHomeSharesDefaultFileCapAndHonorsConfiguredOverride(t *testing.T) {
	root := t.TempDir()
	epoch := "# E99\n\n- Status: in-progress\n\n## Slice log\n" + strings.Repeat("x\n", goalhome.DefaultBoardFileBytes/2+2)
	home := goalHomeTree(t, root, map[string]func(string) string{
		"GOAL.md": func(text string) string {
			title, rest, _ := strings.Cut(text, "\n")
			return title + "\n\n- Destination: journeys >= 24\n- Current: 12 @ 2026-10-20\n" + rest
		},
		"QUEUE.md": func(text string) string {
			return strings.Replace(text, "(none; the operator answered BAS-FB-063 on 2026-10-06)", "- Review the current handoff", 1)
		},
		"epochs/E99.md": func(string) string { return epoch },
	})

	logPath := filepath.Join(home, goalhome.LogFile)
	logBytes, _ := os.ReadFile(logPath)
	logBytes = append(logBytes, []byte("\n"+epoch)...)
	if err := os.WriteFile(logPath, logBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	defaultService, _, defaultController := effortFixture(t)
	defaultService.config.ProjectRoot = root
	if defaultService.config.FileBytes != int64(goalhome.DefaultBoardFileBytes) {
		t.Fatalf("board default cap = %d, want shared default %d", defaultService.config.FileBytes, goalhome.DefaultBoardFileBytes)
	}
	defaultFacts := goalHomeBoardRow(t, defaultService, defaultController).GetGoalHome()
	if defaultFacts == nil || defaultFacts.State != "partial" {
		t.Fatalf("an oversized epoch should preserve a partial bounded projection: %+v", defaultFacts)
	}
	if defaultFacts.DestinationMetric != "journeys" || defaultFacts.CurrentValue == nil || len(defaultFacts.NeedsOperator) == 0 {
		t.Fatalf("bounded GOAL and QUEUE facts should remain visible: %+v", defaultFacts)
	}
	if defaultFacts.ActiveUnit != "" || defaultFacts.LastAcceptedUnit != "" || defaultFacts.WeightedTokensPerAcceptedUnit != nil || len(defaultFacts.CapUsage) != 0 {
		t.Fatalf("incomplete epoch evidence must not project epoch-derived claims: %+v", defaultFacts)
	}
	if !strings.Contains(strings.Join(defaultFacts.Limitations, "\n"), "LOG.md") || !strings.Contains(strings.Join(defaultFacts.Limitations, "\n"), "review before acceptance") {
		t.Fatalf("partial projection must identify the unknown epoch and require review: %+v", defaultFacts.Limitations)
	}
	board, err := defaultService.Board(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Rows) != 1 || board.Rows[0].NextAction != "review goal-home evidence" || !board.Partial {
		t.Fatalf("an otherwise quiet row must direct review of incomplete epoch evidence: %+v", board.Rows)
	}
	orchestrator := enrollmentFixture(t, defaultService, defaultController, domain.RunStatusRunning)
	orchestrator.DestinationRef = "path:" + goalHomeRef + "/GOAL.md"
	if _, err := defaultService.ReconcileMetadata(context.Background(), &pb.ReconcileEffortMetadataRequest{
		Enrollment: orchestrator, ExpectedRevision: orchestrator.Revision, IdempotencyKey: "goal-home-partial-orchestrator",
	}, EffortActor{ID: "owner", Operator: true}); err != nil {
		t.Fatal(err)
	}
	partialRun := &domain.Run{WorkReferences: []*eventpb.WorkReference{{
		Kind: "effort", Id: orchestrator.EffortRef, Relationship: "orchestrator", Verified: true,
		Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
		State:      eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
	}}}
	partialHome, _, found, err := defaultService.GoalHomeForOrchestrator(context.Background(), partialRun)
	if err == nil || !found || partialHome != nil {
		t.Fatalf("orchestrator policy reader must fail closed on cached partial epoch evidence: home=%+v found=%t err=%v", partialHome, found, err)
	}
	rootHandle, err := openSafeRoot(filepath.Join(root, filepath.FromSlash(goalHomeRef)))
	if err != nil {
		t.Fatal(err)
	}
	_, strictErr := goalhome.LoadV2(safeGoalHomeFS{root: rootHandle, limit: defaultService.config.FileBytes})
	rootHandle.Close()
	if strictErr == nil {
		t.Fatal("strict goal-home loading must still reject the oversized epoch")
	}

	defaultService.config.FileBytes = int64(len(logBytes) + 1)
	defaultService.goalHomes.mu.Lock()
	delete(defaultService.goalHomes.homes, goalHomeRef)
	defaultService.goalHomes.mu.Unlock()
	customBoard, err := defaultService.Board(context.Background(), nil)
	if err != nil || len(customBoard.Rows) < 1 {
		t.Fatalf("board with configured larger cap: %v %+v", err, customBoard)
	}
	var customFacts *pb.EffortGoalHome
	for _, row := range customBoard.Rows {
		if row.GetGoalHome() != nil && row.GetGoalHome().GetPath() == goalHomeRef {
			customFacts = row.GetGoalHome()
			break
		}
	}
	if customFacts == nil || customFacts.State != "available" || customFacts.ActiveUnit != "E99" {
		t.Fatalf("a configured larger safe-read cap must remain authoritative and preserve typed epoch parsing: %+v", customFacts)
	}
}

func TestEffortGoalHomePartialProjectionDoesNotFollowEpochSymlinks(t *testing.T) {
	s, _, c := effortFixture(t)
	s.config.ProjectRoot = t.TempDir()
	home := goalHomeTree(t, s.config.ProjectRoot, map[string]func(string) string{
		"GOAL.md": func(text string) string {
			title, rest, _ := strings.Cut(text, "\n")
			return title + "\n\n- Destination: journeys >= 24\n- Current: 12 @ 2026-10-20\n" + rest
		},
	})
	epoch := filepath.Join(home, goalhome.LogFile)
	if err := os.Remove(epoch); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "E25.md")
	if err := os.WriteFile(outside, []byte("# E25\n\n- Status: accepted\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, epoch); err != nil {
		t.Fatal(err)
	}
	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.State != "partial" || facts.DestinationMetric == "" || facts.LastAcceptedUnit != "" {
		t.Fatalf("a symlinked epoch must remain unread and leave only known root facts visible: %+v", facts)
	}
	if !strings.Contains(strings.Join(facts.Limitations, "\n"), "LOG.md") {
		t.Fatalf("the unsafe epoch must be named as incomplete: %+v", facts.Limitations)
	}
}

func TestGoalHomeDestinationAcceptsOnlyRepoRelativeGoalFiles(t *testing.T) {
	for raw, want := range map[string]string{
		"path:" + goalHomeRef + "/GOAL.md":          goalHomeRef,
		"path:" + goalHomeRef + "/GOAL.md#frontier": goalHomeRef,
		"path:" + goalHomeRef + "/CHARTER.md":       goalHomeRef,
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

// copyTree copies every file under source into dest, preserving relative paths.
func copyTree(t *testing.T, source, dest string) {
	t.Helper()
	err := filepath.WalkDir(source, func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(source, p)
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(dest, filepath.Dir(rel)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestEffortGoalHomeReadsPhase2FormatThroughSniff registers a Phase-2 home
// (CHARTER/STATUS/LOG/FEEDBACK) and asserts the board reads its load-bearing
// facts through the v2 sniff, including the new budget_seconds and scoreboard.
func TestEffortGoalHomeReadsPhase2FormatThroughSniff(t *testing.T) {
	s, _, c := effortFixture(t)
	s.config.ProjectRoot = t.TempDir()
	copyTree(t, filepath.Join("..", "goalhome", "testdata", "v2-bas"), filepath.Join(s.config.ProjectRoot, filepath.FromSlash(goalHomeRef)))

	owner := uuid.MustParse("e8c322ca-72e7-434c-9d37-1d2cf853961f")    // STATUS ## Active worker + export surface owner
	measured := uuid.MustParse("a7240424-43e7-4d36-87f6-2b2e83d0c959") // LOG E25's sole metered worker
	c.runs[owner] = &domain.Run{ID: owner, Status: domain.RunStatusRunning}
	c.runs[measured] = &domain.Run{ID: measured, Status: domain.RunStatusComplete}
	s.SetSpendReader(fakeSpend{weighted: map[uuid.UUID]float64{owner: 120000, measured: 640000}, limit: 1000000})

	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.State != "available" || facts.Path != goalHomeRef {
		t.Fatalf("v2 home not projected through the sniff: %+v", facts)
	}
	// CHARTER: destination + budget.
	if facts.DestinationMetric != "runtime lines" || facts.DestinationOp != "<=" || facts.GetDestinationTarget() != 205000 {
		t.Fatalf("destination from CHARTER: %+v", facts)
	}
	if facts.BudgetSeconds != 72*3600 {
		t.Fatalf("budget_seconds from CHARTER '- Budget: 72h': got %d", facts.BudgetSeconds)
	}
	// STATUS: current + scoreboard + active.
	if facts.GetCurrentValue() != 253452 {
		t.Fatalf("current from STATUS: %v", facts.GetCurrentValue())
	}
	if len(facts.SurfaceScoreboard) != 3 {
		t.Fatalf("scoreboard rows from STATUS: %+v", facts.SurfaceScoreboard)
	}
	if first := facts.SurfaceScoreboard[0]; first.Surface != "Workflow codec" || first.State != "accepted" || first.OwnerRun != "" {
		t.Fatalf("accepted surface row (owner '-' means none): %+v", first)
	}
	if export := facts.SurfaceScoreboard[2]; export.Surface != "Export replay" || export.State != "rough" || export.OwnerRun != owner.String() {
		t.Fatalf("export surface row: %+v", export)
	}
	if len(facts.ActiveWorkers) != 1 || facts.ActiveWorkers[0] != owner.String() || facts.ActiveStatus == "" {
		t.Fatalf("active unit from STATUS ## Active: %+v", facts)
	}
	// LOG: accepted units drive last-accepted + spend. Only E25's worker is metered.
	if facts.LastAcceptedAt == nil {
		t.Fatalf("last accepted from LOG ## Accepted units: %+v", facts)
	}
	if facts.AcceptedUnitsMeasured != 1 || facts.GetWeightedTokensPerAcceptedUnit() != 640000 {
		t.Fatalf("spend from LOG: measured=%d mean=%v", facts.AcceptedUnitsMeasured, facts.GetWeightedTokensPerAcceptedUnit())
	}
}

func TestEffortGoalHomeReportsUnmigratedHomeUnknown(t *testing.T) {
	s, _, c := effortFixture(t)
	s.config.ProjectRoot = t.TempDir()
	home := goalHomeTree(t, s.config.ProjectRoot, nil)
	for _, name := range []string{goalhome.CharterFile, goalhome.StatusFile, goalhome.LogFile} {
		if err := os.Remove(filepath.Join(home, name)); err != nil {
			t.Fatal(err)
		}
	}
	facts := goalHomeBoardRow(t, s, c).GetGoalHome()
	if facts == nil || facts.State != "partial" || facts.DestinationTarget != nil || facts.ActiveUnit != "" || facts.LastAcceptedUnit != "" {
		t.Fatalf("unmigrated home must stay unknown pending explicit import: %+v", facts)
	}
}

func TestOrchestratorParkReadsSteeringFreshAfterBoardCache(t *testing.T) {
	s, _, _, controller := decisionFixture(t)
	s.config.ProjectRoot = t.TempDir()
	homeDir := goalHomeTree(t, s.config.ProjectRoot, nil)
	s.config.Interval = time.Hour
	if _, err := s.readGoalHome(goalHomeRef); err != nil {
		t.Fatal(err)
	}
	var run *domain.Run
	for _, candidate := range controller.runs {
		if len(candidate.WorkReferences) > 0 {
			run = candidate
			break
		}
	}
	if run == nil {
		t.Fatal("fixture has no enrolled orchestrator")
	}
	feedbackPath := filepath.Join(homeDir, goalhome.FeedbackFile)
	feedback := []byte("# Feedback\n\n## Open directives\n\n### FRESH-POLICY — updated scope (2026-10-25, supervisor)\n\nStatus: open. Use the revised product target.\n")
	if err := os.WriteFile(feedbackPath, feedback, 0o600); err != nil {
		t.Fatal(err)
	}
	home, _, applies, err := s.GoalHomeForOrchestrator(t.Context(), run)
	if err != nil || !applies {
		t.Fatalf("fresh read: applies=%t err=%v", applies, err)
	}
	if d := goalhome.DecideFeedbackPark(home); d.Allowed || !strings.Contains(d.Reason, "FRESH-POLICY") {
		t.Fatalf("cached board hid new steering: %+v", d)
	}
	statusPath := filepath.Join(homeDir, goalhome.StatusFile)
	status, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	status = []byte(strings.Replace(string(status), "## Handoff", "## Handoff\n\nFRESH-POLICY: amended owned scope and continued product implementation.", 1))
	if err := os.WriteFile(statusPath, status, 0o600); err != nil {
		t.Fatal(err)
	}
	home, _, _, err = s.GoalHomeForOrchestrator(t.Context(), run)
	if err != nil {
		t.Fatal(err)
	}
	if d := goalhome.DecideFeedbackPark(home); !d.Allowed {
		t.Fatalf("cached board hid fresh acknowledgement: %+v", d)
	}
	if err := os.Remove(feedbackPath); err != nil {
		t.Fatal(err)
	}
	if _, _, applies, err := s.GoalHomeForOrchestrator(t.Context(), run); !applies || err == nil || !strings.Contains(err.Error(), goalhome.FeedbackFile) {
		t.Fatalf("missing steering file became empty policy: applies=%t err=%v", applies, err)
	}
}

func TestEffortGoalHomeCapturesOnlyUnambiguousHandoff(t *testing.T) {
	s, _, _ := effortFixture(t)
	for _, tc := range []struct {
		name     string
		handoffs []goalhome.HandoffSection
		want     string
	}{
		{"single", []goalhome.HandoffSection{{Body: "A tested partial result.\n\nThirty-one visual states and current-source validation remain open."}}, "Thirty-one visual states"},
		{"missing", nil, ""},
		{"ambiguous", []goalhome.HandoffSection{{Body: "First"}, {Body: "Second"}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := &pb.EffortGoalHome{}
			s.goalHomeFileFacts(out, &goalhome.Home{Queue: goalhome.Queue{Handoffs: tc.handoffs}})
			if tc.want != "" {
				if !strings.Contains(out.Handoff, tc.want) || !strings.Contains(out.Handoff, "\n\n") {
					t.Fatalf("handoff narrative or paragraphs lost: %q", out.Handoff)
				}
			} else if out.Handoff != "" || !strings.Contains(strings.Join(out.Limitations, " "), "remaining-work narrative is unknown") {
				t.Fatalf("unavailable handoff was guessed: %+v", out)
			}
		})
	}
}
