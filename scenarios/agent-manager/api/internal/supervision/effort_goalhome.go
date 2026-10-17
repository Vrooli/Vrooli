package supervision

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/goalhome"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Goal-home board facts (DL-9). An enrollment registers a goal home with
// destination_ref "path:<repo-relative goal home>/GOAL.md". The board reads
// it through no-symlink, size-bounded reads under the project root and shows
// the facts as a typed EffortGoalHome. A fact that cannot be read is absent
// (unknown), never zero.
const (
	goalHomeAcceptedMeasured = 5
	goalHomeNeedsOperatorMax = 8
	goalHomeItemBytes        = 512
	goalHomeCanaryUnknown    = "unknown"
)

// EffortSpendReader reads weighted-token facts from the run owner. Weighted
// tokens are non-cache tokens times the model tier weight, the weighting the
// token caps use.
type EffortSpendReader interface {
	// WeightedRunTokens returns a run's weighted tokens; known is false when
	// its usage is not authoritatively metered.
	WeightedRunTokens(ctx context.Context, runID uuid.UUID) (weighted float64, known bool, err error)
	// TokenCapUsage returns an active run's weighted usage (nil when unknown),
	// its cap (0 for none) and the cap mode (off, report or enforce).
	TokenCapUsage(ctx context.Context, runID uuid.UUID) (weighted *float64, limit int64, mode string)
}

// SetSpendReader installs the weighted-token reader for goal-home spend and
// cap facts. Without one those facts read unknown.
func (s *EffortService) SetSpendReader(r EffortSpendReader) { s.spend = r }

// goalHomeCache keeps parsed goal homes and terminal runs' weighted spend
// between scan intervals.
type goalHomeCache struct {
	mu    sync.Mutex
	homes map[string]cachedGoalHome
	spent map[uuid.UUID]float64
}

type cachedGoalHome struct {
	at   time.Time
	home *goalhome.Home
	err  error
}

// goalHomeDestination reads an enrollment destination of the form
// "path:<repo-relative goal home>/GOAL.md" (DL-9 registration) and returns
// the goal-home directory.
func goalHomeDestination(destination string) (string, bool) {
	ref, ok := strings.CutPrefix(strings.TrimSpace(destination), "path:")
	if !ok {
		return "", false
	}
	ref, _, _ = strings.Cut(ref, "#")
	if path.Base(ref) != goalhome.GoalFile {
		return "", false
	}
	home, err := NormalizeGoalHome(path.Dir(ref))
	return home, err == nil
}

// safeGoalHomeFS serves goal-home reads through safeRead: no symlinks, no
// escape from the goal home, and a byte cap per file.
type safeGoalHomeFS struct {
	root  *os.Root
	limit int64
}

func (f safeGoalHomeFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
}

func (f safeGoalHomeFS) ReadFile(name string) ([]byte, error) {
	return safeRead(f.root, name, f.limit)
}

func (f safeGoalHomeFS) ReadDir(name string) ([]fs.DirEntry, error) {
	dir, err := safeChildRoot(f.root, name)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	handle, err := dir.Open(".")
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	entries, err := handle.ReadDir(-1)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, err
}

// readGoalHome parses a goal home under the project root, cached for one scan
// interval.
func (s *EffortService) readGoalHome(home string) (*goalhome.Home, error) {
	now := s.now()
	s.goalHomes.mu.Lock()
	cached, ok := s.goalHomes.homes[home]
	s.goalHomes.mu.Unlock()
	if ok && now.Sub(cached.at) < s.config.Interval {
		return cached.home, cached.err
	}
	parsed, err := s.loadGoalHome(home)
	s.goalHomes.mu.Lock()
	if s.goalHomes.homes == nil {
		s.goalHomes.homes = map[string]cachedGoalHome{}
	}
	s.goalHomes.homes[home] = cachedGoalHome{at: now, home: parsed, err: err}
	s.goalHomes.mu.Unlock()
	return parsed, err
}

func (s *EffortService) loadGoalHome(home string) (*goalhome.Home, error) {
	if strings.TrimSpace(s.config.ProjectRoot) == "" {
		return nil, errors.New("project root is not configured")
	}
	root, err := openSafeRoot(s.config.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("project root: %w", err)
	}
	defer root.Close()
	at := root
	for _, part := range strings.Split(filepath.FromSlash(home), string(filepath.Separator)) {
		next, err := safeChildRoot(at, part)
		if at != root {
			at.Close()
		}
		if err != nil {
			return nil, fmt.Errorf("goal home %s: %w", home, err)
		}
		at = next
	}
	defer at.Close()
	return goalhome.Load(safeGoalHomeFS{root: at, limit: s.config.FileBytes})
}

// GoalHomeForOrchestrator returns the goal home of the enrolled effort the run
// holds as orchestrator, and false when it holds none with a goal home.
func (s *EffortService) GoalHomeForOrchestrator(ctx context.Context, run *domain.Run) (*goalhome.Home, string, bool, error) {
	if run == nil {
		return nil, "", false, nil
	}
	for _, work := range run.WorkReferences {
		if !publicActiveEffortReference(work) || work.GetRelationship() != "orchestrator" {
			continue
		}
		e, _, err := s.repo.GetEffort(ctx, work.GetId())
		if errors.Is(err, ErrNotFound) || (err == nil && (e == nil || e.Withdrawn)) {
			continue
		}
		if err != nil {
			return nil, "", false, err
		}
		dir, ok := goalHomeDestination(e.GetDestinationRef())
		if !ok {
			continue
		}
		home, err := s.readGoalHome(dir)
		return home, dir, true, err
	}
	return nil, "", false, nil
}

// projectGoalHome builds the board's goal-home facts for an enrollment whose
// destination is a registered goal home, or nil for any other enrollment.
func (s *EffortService) projectGoalHome(ctx context.Context, e *pb.EffortEnrollment, row *pb.EffortBoardRow) *pb.EffortGoalHome {
	dir, ok := goalHomeDestination(e.GetDestinationRef())
	if !ok {
		return nil
	}
	out := &pb.EffortGoalHome{Path: dir, State: "available", CanaryStatus: goalHomeCanaryUnknown, ReadAt: timestamppb.New(s.now().UTC())}
	home, err := s.readGoalHome(dir)
	if err != nil {
		out.State = "unavailable"
		out.Limitations = append(out.Limitations, "goal home unreadable: "+err.Error())
	} else {
		s.goalHomeFileFacts(out, home)
		s.goalHomeSpendFacts(ctx, out, home)
		s.goalHomeCapFacts(ctx, out, home, row)
	}
	if decisions, err := s.repo.ListEffortDecisions(ctx, dir, "", 200); err != nil {
		out.Limitations = append(out.Limitations, "decisions unavailable: "+err.Error())
	} else {
		for _, d := range decisions {
			if d.RecordedAt == nil {
				out.OpenDecisions = append(out.OpenDecisions, d)
			}
		}
	}
	return out
}

func (s *EffortService) goalHomeFileFacts(out *pb.EffortGoalHome, home *goalhome.Home) {
	for _, missing := range home.Missing {
		out.Limitations = append(out.Limitations, missing+" is missing")
	}
	for _, warning := range home.Warnings {
		out.Limitations = append(out.Limitations, fmt.Sprintf("%s %s: %s", warning.Code, warning.File, warning.Detail))
	}
	if d := home.Goal.Destination; d != nil {
		out.DestinationMetric, out.DestinationOp, out.DestinationTarget = d.Metric, d.Op, proto.Float64(d.Target)
	} else {
		out.Limitations = append(out.Limitations, "GOAL.md has no '- Destination: <metric> <op> <n>' line; destination unknown")
	}
	if c := home.Goal.Current; c != nil {
		out.CurrentValue = proto.Float64(c.Value)
		if !c.At.IsZero() {
			out.CurrentAsOf = timestamppb.New(c.At)
		}
	}
	for i := len(home.Epochs) - 1; i >= 0; i-- {
		epoch := home.Epochs[i]
		if epoch.Epoch.IsAccepted() {
			if out.LastAcceptedEpoch == "" {
				out.LastAcceptedEpoch = fmt.Sprintf("E%d", epoch.Number)
				if accepted := epoch.Epoch.LastAcceptance(); accepted != nil && !accepted.At.IsZero() {
					out.LastAcceptedAt = timestamppb.New(accepted.At)
				}
				if accepted := epoch.Epoch.LastAcceptance(); accepted != nil && accepted.Yield != nil {
					out.LastYield = accepted.Yield.String()
				}
			}
			continue
		}
		if out.ActiveEpoch == "" && out.LastAcceptedEpoch == "" {
			out.ActiveEpoch = fmt.Sprintf("E%d", epoch.Number)
			out.ActiveStatus = epoch.Epoch.Fields["status"]
			out.ActiveWorkers = goalhome.WorkerRunIDs(epoch.Epoch.Fields["workers"])
		}
	}
	for i, item := range home.Queue.NeedsOperator {
		if i == goalHomeNeedsOperatorMax {
			out.Limitations = append(out.Limitations, fmt.Sprintf("%d more ## Needs operator items", len(home.Queue.NeedsOperator)-i))
			break
		}
		first, _, _ := strings.Cut(item.Text, "\n")
		if len(first) > goalHomeItemBytes {
			first = first[:goalHomeItemBytes]
		}
		out.NeedsOperator = append(out.NeedsOperator, first)
	}
	if !missingFile(home, goalhome.WorkaroundsFile) {
		open := uint32(0)
		for _, entry := range home.Workarounds.Entries {
			if entry.Open {
				open++
			}
		}
		out.OpenWorkarounds = proto.Uint32(open)
	}
	for _, entry := range home.Feedback.Entries {
		if entry.Open() {
			out.OpenFeedbackIds = append(out.OpenFeedbackIds, entry.ID)
		}
	}
	for _, finding := range goalhome.Lint(home).Findings {
		if finding.Blocking && !slices.Contains(out.LintFindings, finding.Code) {
			out.LintFindings = append(out.LintFindings, finding.Code)
		}
	}
}

func missingFile(home *goalhome.Home, name string) bool {
	for _, missing := range home.Missing {
		if missing == name {
			return true
		}
	}
	return false
}

// goalHomeSpendFacts averages the weighted spend of the last accepted epochs
// whose every Workers run is metered. An epoch with an unmetered or unknown
// worker is left out, never counted as zero.
func (s *EffortService) goalHomeSpendFacts(ctx context.Context, out *pb.EffortGoalHome, home *goalhome.Home) {
	if s.spend == nil {
		out.Limitations = append(out.Limitations, "weighted spend reader unavailable; spend per accepted epoch unknown")
		return
	}
	total, considered := 0.0, 0
	for i := len(home.Epochs) - 1; i >= 0 && considered < goalHomeAcceptedMeasured; i-- {
		epoch := home.Epochs[i]
		if !epoch.Epoch.IsAccepted() {
			continue
		}
		considered++
		spent, ok := s.epochSpend(ctx, goalhome.WorkerRunIDs(epoch.Epoch.Fields["workers"]))
		if !ok {
			continue
		}
		total += spent
		out.AcceptedEpochsMeasured++
	}
	if out.AcceptedEpochsMeasured > 0 {
		out.WeightedTokensPerAcceptedEpoch = proto.Float64(total / float64(out.AcceptedEpochsMeasured))
	}
	if unmeasured := considered - int(out.AcceptedEpochsMeasured); unmeasured > 0 {
		out.Limitations = append(out.Limitations, fmt.Sprintf("%d of the last %d accepted epochs have an unmetered or unknown worker; left out of spend", unmeasured, considered))
	}
}

func (s *EffortService) epochSpend(ctx context.Context, workers []string) (float64, bool) {
	if len(workers) == 0 {
		return 0, false
	}
	total := 0.0
	for _, raw := range workers {
		id, err := uuid.Parse(raw)
		if err != nil {
			return 0, false
		}
		s.goalHomes.mu.Lock()
		spent, ok := s.goalHomes.spent[id]
		s.goalHomes.mu.Unlock()
		if !ok {
			weighted, known, err := s.spend.WeightedRunTokens(ctx, id)
			if err != nil || !known {
				return 0, false
			}
			spent = weighted
			// A terminal run's metered spend never changes.
			if run, err := s.controller.GetRun(ctx, id); err == nil && run != nil && run.Status.IsTerminal() {
				s.goalHomes.mu.Lock()
				if s.goalHomes.spent == nil {
					s.goalHomes.spent = map[uuid.UUID]float64{}
				}
				s.goalHomes.spent[id] = spent
				s.goalHomes.mu.Unlock()
			}
		}
		total += spent
	}
	return total, true
}

// goalHomeCapFacts reads the cap usage of the effort's active runs: the row's
// active assignments and the active epoch's workers.
func (s *EffortService) goalHomeCapFacts(ctx context.Context, out *pb.EffortGoalHome, home *goalhome.Home, row *pb.EffortBoardRow) {
	if s.spend == nil || s.controller == nil {
		return
	}
	roles := map[string]string{}
	var order []string
	add := func(id, role string) {
		if _, seen := roles[id]; !seen {
			order = append(order, id)
		}
		if roles[id] == "" {
			roles[id] = role
		}
	}
	for _, a := range row.GetAssignments() {
		state := domain.RunStatus(a.GetRuntimeState())
		if a.GetSubject().GetRunId() != "" && (state.IsActive() || state == domain.RunStatusParked) {
			add(a.GetSubject().GetRunId(), a.GetSubject().GetRole())
		}
	}
	for _, id := range out.ActiveWorkers {
		add(id, "worker")
	}
	for _, raw := range order {
		id, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		run, err := s.controller.GetRun(ctx, id)
		if err != nil || run == nil || run.Status.IsTerminal() {
			continue
		}
		weighted, limit, mode := s.spend.TokenCapUsage(ctx, id)
		usage := &pb.EffortCapUsage{RunId: raw, Role: roles[raw], WeightedTokens: weighted, Mode: mode}
		if limit > 0 {
			usage.Cap = proto.Int64(limit)
		}
		out.CapUsage = append(out.CapUsage, usage)
	}
}
