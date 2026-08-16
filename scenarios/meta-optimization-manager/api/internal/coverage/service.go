package coverage

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/spacedoc"
)

const coverageMethodVersion = "answer-tiered-corpus-end-to-end-v3"

// TrendProvider supplies the latest empirical trials trend for the scoreboard.
// The trials domain (Phase 4) implements it; until then GetStatus surfaces no
// trend. Optional — a nil provider simply omits the trend.
type TrendProvider interface {
	LatestTrend(ctx context.Context) (*EmpiricalTrendPoint, bool)
}

// Service is the coverage application surface.
type Service interface {
	GetStatus(ctx context.Context, projection Projection) (Status, error)
	ListCells(ctx context.Context, projection Projection, status spacedoc.CellStatus) ([]Cell, error)
	ExplainCell(ctx context.Context, cellID string) (Cell, error)
	ValidateBaseDocs(ctx context.Context, projection Projection) (BaseDocReport, error)
}

type service struct {
	reader    SpaceReader
	joiner    NumeratorJoiner
	snaps     SnapshotRepository
	trend     TrendProvider
	clock     schedule.Clock
	adherence AdherenceReader
}

// Deps wires the coverage Service. Snapshots and Trend are optional (nil-safe).
type Deps struct {
	Reader    SpaceReader
	Joiner    NumeratorJoiner
	Snapshots SnapshotRepository
	Trend     TrendProvider
	Clock     schedule.Clock
	Adherence AdherenceReader
}

// NewService constructs the coverage Service, defaulting the production seams.
func NewService(d Deps) Service {
	if d.Reader == nil {
		d.Reader = NewSpaceReader()
	}
	if d.Joiner == nil {
		d.Joiner = NewNumeratorJoiner()
	}
	if d.Clock == nil {
		d.Clock = schedule.System()
	}
	return &service{reader: d.Reader, joiner: d.Joiner, snaps: d.Snapshots, trend: d.Trend, clock: d.Clock, adherence: d.Adherence}
}

var _ Service = (*service)(nil)

// GetStatus computes per-projection coverage, degrading gracefully where an
// owner is unreachable. A short-TTL snapshot absorbs bursts. projection == ""
// (or UNSPECIFIED) computes all three.
func (s *service) GetStatus(ctx context.Context, projection Projection) (Status, error) {
	all := projection == "" || projection == spacedoc.Projection("")
	now := s.clock.Now().UTC()
	var previous Status
	var havePrevious bool
	if all && s.snaps != nil {
		previous, havePrevious = s.snaps.Latest(ctx, 365*24*time.Hour, now)
		if havePrevious && previous.CoverageMethodVersion != coverageMethodVersion {
			havePrevious = false
		}
	}

	targets := AllProjections
	if !all {
		if OwnerFor(projection) == "" {
			return Status{}, fmt.Errorf("coverage: unknown projection %q", projection)
		}
		targets = []Projection{projection}
	}

	// Compute the projections concurrently: each owner read is independent and
	// never returns an error (an unreachable/slow owner degrades to an honest
	// Available=false), so one slow owner can never fail or stall the others.
	// Total board latency is the slowest single projection, capped by the
	// per-owner read deadlines — not the serial sum that let one ~30s hang stall
	// the whole scoreboard.
	out := Status{ComputedAt: now, CoverageMethodVersion: coverageMethodVersion}
	out.Projections = make([]ProjectionCoverage, len(targets))
	var wg sync.WaitGroup
	for i, p := range targets {
		wg.Add(1)
		go func(i int, p Projection) {
			defer wg.Done()
			first := s.coverageFor(ctx, p)
			if p != ProjectionAnswer {
				out.Projections[i] = first
				return
			}
			second := s.coverageFor(ctx, p)
			out.Projections[i] = first
			out.DeterminismChecked = true
			out.Deterministic = reflect.DeepEqual(first, second)
			if out.Deterministic {
				out.DeterminismEvidence = "two uncached Answer joins produced identical effective coverage"
			} else {
				out.DeterminismEvidence = fmt.Sprintf("uncached Answer joins disagreed: first=%+v second=%+v", first, second)
			}
		}(i, p)
	}
	wg.Wait()
	if s.trend != nil {
		if t, ok := s.trend.LatestTrend(ctx); ok {
			out.LatestTrialTrend = t
		}
	}
	if havePrevious {
		previousByProjection := make(map[Projection]ProjectionCoverage, len(previous.Projections))
		for _, pc := range previous.Projections {
			previousByProjection[pc.Projection] = pc
		}
		for _, pc := range out.Projections {
			if old, ok := previousByProjection[pc.Projection]; ok {
				out.Deltas = append(out.Deltas, ProjectionDelta{Projection: pc.Projection, PreviousRatio: old.CoverageRatio, CurrentRatio: pc.CoverageRatio, Delta: pc.CoverageRatio - old.CoverageRatio})
			}
		}
	}
	if all && s.snaps != nil {
		_ = s.snaps.Save(ctx, out) // best-effort cache; never fail the read on a cache miss
	}
	return out, nil
}

// coverageFor computes one projection's ProjectionCoverage, never returning an
// error: an unreachable owner yields an honest Available=false result.
func (s *service) coverageFor(ctx context.Context, p Projection) ProjectionCoverage {
	pc := ProjectionCoverage{Projection: p}
	def, err := s.reader.Read(ctx, p)
	if err != nil {
		pc.Available = false
		pc.UnavailableReason = fmt.Sprintf("%s denominator unavailable: %v", OwnerFor(p), err)
		return pc
	}
	pc.TotalCells = len(def.Cells)

	join := s.joiner.Join(ctx, p, def.Cells)
	pc.Available = join.Available
	pc.UnavailableReason = join.Reason
	conditionCounts := map[ConditionVerdict]int{}
	if pc.Available {
		pc.DenominatorConfidence = def.DenominatorConfidence
		pc.ConfidenceRationale = def.ConfidenceRationale
		if join.DenominatorConfidence != "" {
			pc.DenominatorConfidence = join.DenominatorConfidence
		}
	}
	for _, c := range def.Cells {
		switch effectiveStatus(c, join) {
		case spacedoc.StatusNow:
			pc.NowCount++
		case spacedoc.StatusInReach:
			pc.InReachCount++
		default:
			pc.MissingCount++
		}
		if condition, ok := join.Conditions[c.ID]; ok {
			conditionCounts[condition]++
		}
	}
	for _, condition := range []ConditionVerdict{ConditionOK, ConditionDegraded, ConditionUninstrumented, ConditionDormant} {
		if count := conditionCounts[condition]; count > 0 {
			pc.ConditionCounts = append(pc.ConditionCounts, ConditionCount{Condition: condition, Count: count})
		}
	}
	if pc.Available && pc.TotalCells > 0 {
		pc.CoverageRatio = float64(pc.NowCount) / float64(pc.TotalCells)
	}
	if p == ProjectionAnswer && pc.Available {
		pc.CorpusCapableNowCount = join.AnswerCorpusCapableNowCount
		pc.CorpusCapableTotalCells = join.AnswerCorpusCapableTotalCells
		pc.EndToEndAnswerableNowCount = join.AnswerEndToEndNowCount
		pc.EndToEndAnswerableTotalCells = join.AnswerEndToEndTotalCells
		if pc.CorpusCapableTotalCells > 0 {
			pc.CorpusCapableRatio = float64(pc.CorpusCapableNowCount) / float64(pc.CorpusCapableTotalCells)
		}
		if pc.EndToEndAnswerableTotalCells > 0 {
			pc.EndToEndAnswerableRatio = float64(pc.EndToEndAnswerableNowCount) / float64(pc.EndToEndAnswerableTotalCells)
		}
	}
	return pc
}

// effectiveStatus returns a cell's live status: the joiner's per-cell status
// when present, else the authored denominator status.
func effectiveStatus(c spacedoc.Cell, join JoinResult) spacedoc.CellStatus {
	if join.Statuses != nil {
		if st, ok := join.Statuses[c.ID]; ok {
			return st
		}
	}
	return c.Status
}

// ListCells returns the denominator cells (across all projections, or one),
// with live status applied, optionally filtered by status. Cell IDs are
// namespaced "<projection>/<rawid>" so they are unique across projections.
func (s *service) ListCells(ctx context.Context, projection Projection, status spacedoc.CellStatus) ([]Cell, error) {
	targets := AllProjections
	if projection != "" {
		if OwnerFor(projection) == "" {
			return nil, fmt.Errorf("coverage: unknown projection %q", projection)
		}
		targets = []Projection{projection}
	}
	var out []Cell
	for _, p := range targets {
		def, err := s.reader.Read(ctx, p)
		if err != nil {
			continue // degrade: skip an unreachable projection's cells
		}
		join := s.joiner.Join(ctx, p, def.Cells)
		for _, c := range def.Cells {
			eff := effectiveStatus(c, join)
			if status != "" && eff != status {
				continue
			}
			out = append(out, s.toCellWithJoin(ctx, p, c, eff, join))
		}
	}
	return out, nil
}

// ExplainCell returns one cell (by namespaced id "<projection>/<rawid>") with
// its provenance citations.
func (s *service) ExplainCell(ctx context.Context, cellID string) (Cell, error) {
	p, rawID, err := splitCellID(cellID)
	if err != nil {
		return Cell{}, err
	}
	def, err := s.reader.Read(ctx, p)
	if err != nil {
		return Cell{}, fmt.Errorf("coverage: %s unavailable: %w", OwnerFor(p), err)
	}
	join := s.joiner.Join(ctx, p, def.Cells)
	for _, c := range def.Cells {
		if c.ID != rawID {
			continue
		}
		cell := s.toCellWithJoin(ctx, p, c, effectiveStatus(c, join), join)
		cell.Citations = citationsFor(def, c)
		return cell, nil
	}
	return Cell{}, fmt.Errorf("coverage: cell %q not found", cellID)
}

// citationsFor builds provenance pointers for a cell: the space doc it came from
// and its declared owner provider.
func citationsFor(def *spacedoc.SpaceDefinition, c spacedoc.Cell) []Citation {
	cites := []Citation{}
	if def.Source != "" {
		cites = append(cites, Citation{Locator: def.Source, Kind: "doc", Note: "denominator cell " + c.ID})
	}
	if c.Owner != "" && !strings.Contains(strings.ToLower(c.Owner), "none") {
		cites = append(cites, Citation{Locator: c.Owner, Kind: "runtime", Note: "declared owner/provider"})
	}
	return cites
}

func toCell(p Projection, c spacedoc.Cell, eff spacedoc.CellStatus) Cell {
	return Cell{
		ID:          string(p) + "/" + c.ID,
		Projection:  p,
		Group:       c.Group,
		Question:    c.Question,
		Owner:       c.Owner,
		Status:      eff,
		Basis:       c.Basis,
		Sufficiency: c.Sufficiency,
		Notes:       c.Notes,
	}
}

func (s *service) toCell(ctx context.Context, p Projection, c spacedoc.Cell, eff spacedoc.CellStatus) Cell {
	cell := toCell(p, c, eff)
	if s.adherence == nil {
		cell.ObservedAdherence = ObservedAdherence{State: "unavailable", Reason: "agent-run adherence source is not configured"}
	} else {
		observed, err := s.adherence.ReadAdherence(ctx, p, c)
		if err != nil {
			observed = ObservedAdherence{State: "unavailable", Reason: err.Error()}
		}
		cell.ObservedAdherence = observed
	}
	cell.Notes = append(cell.Notes, observedAdherenceNote(cell.ObservedAdherence))
	return cell
}

func (s *service) toCellWithJoin(ctx context.Context, p Projection, c spacedoc.Cell, eff spacedoc.CellStatus, join JoinResult) Cell {
	cell := s.toCell(ctx, p, c, eff)
	cell.SignalEvidence = append(cell.SignalEvidence, join.Evidence[c.ID]...)
	cell.Condition = join.Conditions[c.ID]
	if p == ProjectionAnswer {
		if !join.Available {
			reason := join.Reason
			if reason == "" {
				reason = "search-hub signal read unavailable"
			}
			for _, signal := range []string{"active", "reachable", "eval_fresh"} {
				cell.SignalEvidence = append(cell.SignalEvidence, SignalEvidence{Signal: signal, Verdict: "unavailable", Evidence: reason})
			}
		}
		for _, signal := range cell.SignalEvidence {
			cell.Notes = append(cell.Notes, fmt.Sprintf("answer signal %s: %s (%s)", signal.Signal, signal.Verdict, signal.Evidence))
		}
	}
	return cell
}

func observedAdherenceNote(observed ObservedAdherence) string {
	if observed.State == "available" {
		return fmt.Sprintf("observed_adherence: %d/%d (%.1f%%)", observed.Numerator, observed.Denominator, observed.Ratio*100)
	}
	if observed.Reason == "" {
		return "observed_adherence: " + observed.State
	}
	return "observed_adherence: " + observed.State + " (" + observed.Reason + ")"
}

func splitCellID(id string) (Projection, string, error) {
	i := strings.IndexByte(id, '/')
	if i <= 0 || i == len(id)-1 {
		return "", "", fmt.Errorf("coverage: cell id %q must be '<projection>/<id>'", id)
	}
	p := Projection(id[:i])
	if OwnerFor(p) == "" {
		return "", "", fmt.Errorf("coverage: cell id %q has unknown projection", id)
	}
	return p, id[i+1:], nil
}
