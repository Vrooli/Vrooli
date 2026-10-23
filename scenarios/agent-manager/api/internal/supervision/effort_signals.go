package supervision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxEffortSignalCheckinsPerTick = 100

// EffortSignalWakeController can resume only a retained parked run, with the
// exact queued signal high-water committed alongside its running transition.
// It is intentionally separate from the general ActionController surface.
type EffortSignalWakeController interface {
	WakeEffortCheckin(context.Context, uuid.UUID, string, int64) error
}

// Effort signal kinds (DL-2). A check-in dispatcher starts one orchestrator
// check-in when the log advances past its cursor.
const (
	SignalChildRunEnded  = "child_run_ended"
	SignalStepBack       = "step_back"
	SignalFeedback       = "feedback"
	SignalOperatorAnswer = "operator_answer"
	SignalCapStop        = "cap_stop"
)

const (
	defaultSignalPage = 100
	maxSignalPage     = 500
	maxSignalDetail   = 2048
	// signalSweepBackfill is how far back the first terminal-run sweep looks.
	signalSweepBackfill = 24 * time.Hour
	// signalSweepOverlap re-reads the end of the previous window, so a run
	// whose ended_at was written late is not missed; dedupe absorbs repeats.
	signalSweepOverlap = 2 * time.Minute
	signalSweepPage    = 500
	signalSweepPages   = 20
)

// ErrSignalKindNotAccepted refuses a caller-recorded kind that Agent Manager
// derives itself.
var ErrSignalKindNotAccepted = errors.New("only step_back and feedback signals are recorded by callers; Agent Manager derives the other kinds")

// AppendEffortSignal records one signal. A repeated dedupe key returns the
// stored signal and false. seq is allocated in the same statement, under
// signalMu, so it increases monotonically per effort.
func (r *Repository) AppendEffortSignal(ctx context.Context, ref, kind, runID, detail, dedupeKey string) (*pb.EffortSignal, bool, error) {
	if len(detail) > maxSignalDetail {
		detail = detail[:maxSignalDetail]
	}
	if dedupeKey == "" {
		dedupeKey = kind + ":" + runID + ":" + detail
	}
	r.signalMu.Lock()
	defer r.signalMu.Unlock()
	result, err := r.db.ExecContext(ctx, `INSERT INTO supervision_effort_signals(effort_ref,seq,kind,run_id,detail,dedupe_key,created_at)
SELECT ?, COALESCE(MAX(seq),0)+1, ?, ?, ?, ?, ? FROM supervision_effort_signals WHERE effort_ref=?
ON CONFLICT(effort_ref, dedupe_key) DO NOTHING`, ref, kind, runID, detail, dedupeKey, formatTime(r.now()), ref)
	if err != nil {
		return nil, false, err
	}
	inserted, _ := result.RowsAffected()
	signal, err := r.effortSignalByKey(ctx, ref, dedupeKey)
	return signal, inserted == 1, err
}

func (r *Repository) effortSignalByKey(ctx context.Context, ref, key string) (*pb.EffortSignal, error) {
	row := r.db.QueryRowContext(ctx, `SELECT seq,kind,run_id,detail,created_at FROM supervision_effort_signals WHERE effort_ref=? AND dedupe_key=?`, ref, key)
	return scanEffortSignal(ref, row.Scan)
}

func scanEffortSignal(ref string, scan func(...any) error) (*pb.EffortSignal, error) {
	signal := &pb.EffortSignal{EffortRef: ref}
	var created string
	if err := scan(&signal.Seq, &signal.Kind, &signal.RunId, &signal.Detail, &created); err != nil {
		return nil, err
	}
	if at, err := time.Parse(time.RFC3339Nano, created); err == nil {
		signal.CreatedAt = timestamppb.New(at)
	}
	return signal, nil
}

// ListEffortSignals returns signals after the cursor in seq order, and the
// effort's latest seq.
func (r *Repository) ListEffortSignals(ctx context.Context, ref string, after int64, limit int) ([]*pb.EffortSignal, int64, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT seq,kind,run_id,detail,created_at FROM supervision_effort_signals WHERE effort_ref=? AND seq>? ORDER BY seq LIMIT ?`, ref, after, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	signals := []*pb.EffortSignal{}
	for rows.Next() {
		signal, err := scanEffortSignal(ref, rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		signals = append(signals, signal)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	var latest int64
	err = r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),0) FROM supervision_effort_signals WHERE effort_ref=?`, ref).Scan(&latest)
	return signals, latest, err
}

// EffortSignalCursor is the terminal-run sweep watermark; zero before the
// first sweep.
func (r *Repository) EffortSignalCursor(ctx context.Context) (time.Time, error) {
	var through string
	err := r.db.QueryRowContext(ctx, `SELECT ended_through FROM supervision_effort_signal_cursor WHERE singleton=1`).Scan(&through)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, through)
}

func (r *Repository) SaveEffortSignalCursor(ctx context.Context, through time.Time) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO supervision_effort_signal_cursor(singleton,ended_through) VALUES(1,?) ON CONFLICT(singleton) DO UPDATE SET ended_through=excluded.ended_through`, formatTime(through))
	return err
}

// ListSignals is the check-in dispatcher's read: signals after after_seq.
func (s *EffortService) ListSignals(ctx context.Context, req *pb.ListEffortSignalsRequest) (*pb.ListEffortSignalsResponse, error) {
	if req == nil || strings.TrimSpace(req.EffortRef) == "" {
		return nil, errors.New("effort reference required")
	}
	limit := int(req.Limit)
	if limit <= 0 {
		limit = defaultSignalPage
	}
	if limit > maxSignalPage {
		limit = maxSignalPage
	}
	signals, latest, err := s.repo.ListEffortSignals(ctx, req.EffortRef, req.AfterSeq, limit)
	if err != nil {
		return nil, err
	}
	return &pb.ListEffortSignalsResponse{Signals: signals, LatestSeq: latest}, nil
}

// dispatchEffortSignalCheckins walks a bounded, rotating keyset of enrolled
// orchestrator subjects. Per-run cursors make signal delivery durable; this
// in-memory scan position only prevents a large registry from monopolizing a
// single scheduler tick and safely resets after restart.
func (s *EffortService) dispatchEffortSignalCheckins(ctx context.Context) error {
	if s.signalCursorStore == nil || s.controller == nil {
		return nil
	}
	waker, ok := s.controller.(EffortSignalWakeController)
	if !ok {
		return errors.New("retained effort signal wake controller is unavailable")
	}
	processed, effortsRead := 0, 0
	var dispatchErrors []error
	for processed < maxEffortSignalCheckinsPerTick && effortsRead < maxEffortSignalCheckinsPerTick {
		var enrollment *pb.EffortEnrollment
		var err error
		if s.signalScanSubject != "" {
			enrollment, _, err = s.repo.GetEffort(ctx, s.signalScanAfter)
			if errors.Is(err, ErrNotFound) {
				s.signalScanSubject = ""
				continue
			}
		} else {
			var rows []*pb.EffortEnrollment
			rows, err = s.repo.ListEfforts(ctx, s.signalScanAfter, 1)
			if err == nil && len(rows) > 0 {
				enrollment = rows[0]
			}
		}
		if err != nil {
			return errors.Join(append(dispatchErrors, fmt.Errorf("list effort signal owners: %w", err))...)
		}
		if enrollment == nil {
			s.signalScanAfter, s.signalScanSubject = "", ""
			return errors.Join(dispatchErrors...)
		}
		effortsRead++
		if enrollment.GetWithdrawn() {
			s.signalScanAfter, s.signalScanSubject = enrollment.GetEffortRef(), ""
			continue
		}
		subjects := orchestratorSignalSubjects(enrollment)
		next := -1
		for i, subject := range subjects {
			if subject.GetRunId() > s.signalScanSubject {
				next = i
				break
			}
		}
		if next < 0 {
			s.signalScanAfter, s.signalScanSubject = enrollment.GetEffortRef(), ""
			continue
		}
		subject := subjects[next]
		if err := s.dispatchEffortSignalSubject(ctx, waker, enrollment, subject); err != nil {
			// Keep this exact owner's durable queued cursor pending, but advance
			// the in-memory scan so one unavailable owner cannot starve later
			// enrolled goals. Wrapping the keyset retries it on a later tick.
			dispatchErrors = append(dispatchErrors, err)
		}
		s.signalScanAfter, s.signalScanSubject = enrollment.GetEffortRef(), subject.GetRunId()
		processed++
	}
	return errors.Join(dispatchErrors...)
}

func orchestratorSignalSubjects(e *pb.EffortEnrollment) []*pb.EffortSubject {
	if e == nil {
		return nil
	}
	seen := map[string]bool{}
	result := make([]*pb.EffortSubject, 0, len(e.GetSubjects()))
	for _, subject := range e.GetSubjects() {
		if subject == nil || subject.GetOwner() != "agent-manager" || subject.GetKind() != "run" || subject.GetRole() != "orchestrator" || subject.GetRunId() == "" || subject.GetReference() != subject.GetRunId() || seen[subject.GetRunId()] {
			continue
		}
		if _, err := uuid.Parse(subject.GetRunId()); err != nil {
			continue
		}
		seen[subject.GetRunId()] = true
		result = append(result, subject)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].GetRunId() < result[j].GetRunId() })
	return result
}

func (s *EffortService) dispatchEffortSignalSubject(ctx context.Context, waker EffortSignalWakeController, enrollment *pb.EffortEnrollment, subject *pb.EffortSubject) error {
	if enrollment == nil || enrollment.GetWithdrawn() {
		return nil
	}
	id, err := uuid.Parse(subject.GetRunId())
	if err != nil {
		return nil
	}
	run, err := s.controller.GetRun(ctx, id)
	if err != nil {
		return fmt.Errorf("read enrolled effort orchestrator %s: %w", id, err)
	}
	if run == nil || run.ID != id || !run.Status.IsValid() || run.Status.IsTerminal() || !runCarriesEffortOrchestrator(run, enrollment.GetEffortRef()) {
		return nil
	}
	through := int64(0)
	cursor, hasCursor := run.EffortSignalCursors[enrollment.GetEffortRef()]
	if hasCursor && cursor.QueuedSeq > cursor.DeliveredSeq {
		through = cursor.QueuedSeq
	} else {
		var signals []*pb.EffortSignal
		if hasCursor {
			signals, _, err = s.repo.ListEffortSignals(ctx, enrollment.GetEffortRef(), cursor.QueuedSeq, defaultSignalPage)
		} else {
			// Old enrollments have no proof that their retained history was read.
			// Coalesce one bounded latest-row catch-up; the neutral wake only asks
			// the owner to reread current canonical feedback and does not replay text.
			_, latest, listErr := s.repo.ListEffortSignals(ctx, enrollment.GetEffortRef(), 0, 1)
			if listErr != nil {
				return fmt.Errorf("read latest effort signal for %s: %w", enrollment.GetEffortRef(), listErr)
			}
			through = latest
		}
		if err != nil {
			return fmt.Errorf("read effort signals for %s: %w", enrollment.GetEffortRef(), err)
		}
		if hasCursor {
			if len(signals) == 0 {
				return nil
			}
			through = signals[len(signals)-1].GetSeq()
		}
		cursor, eligible, queueErr := s.signalCursorStore.QueueEffortSignalCursor(ctx, id, enrollment.GetEffortRef(), through)
		if queueErr != nil {
			return fmt.Errorf("queue effort signal check-in for %s: %w", id, queueErr)
		}
		if !eligible {
			return nil
		}
		if cursor.QueuedSeq <= cursor.DeliveredSeq {
			return nil
		}
		through = cursor.QueuedSeq
	}
	if run.Status != domain.RunStatusParked || through <= 0 {
		return nil
	}
	if err := waker.WakeEffortCheckin(ctx, id, enrollment.GetEffortRef(), through); err != nil {
		return fmt.Errorf("wake retained effort orchestrator %s through signal %d: %w", id, through, err)
	}
	return nil
}

func runCarriesEffortOrchestrator(run *domain.Run, effortRef string) bool {
	if run == nil || effortRef == "" {
		return false
	}
	for _, ref := range run.WorkReferences {
		if domain.IsPublicActiveEffortOrchestratorReference(ref) && ref.GetId() == effortRef {
			return true
		}
	}
	return false
}

// RecordCallerSignal records a step_back from a worker's epoch-check or a
// feedback signal from whoever wrote a FEEDBACK.md entry. A named run must
// exist and carry the effort reference, so a signal can only name a run of
// that effort.
func (s *EffortService) RecordCallerSignal(ctx context.Context, req *pb.RecordEffortSignalRequest) (*pb.EffortSignal, error) {
	if req == nil || strings.TrimSpace(req.EffortRef) == "" {
		return nil, errors.New("effort reference required")
	}
	if req.Kind != SignalStepBack && req.Kind != SignalFeedback {
		return nil, ErrSignalKindNotAccepted
	}
	if req.RunId != "" {
		id, err := uuid.Parse(req.RunId)
		if err != nil {
			return nil, errors.New("invalid run UUID")
		}
		if s.controller == nil {
			return nil, errors.New("run owner unavailable")
		}
		run, err := s.controller.GetRun(ctx, id)
		if err != nil || run == nil {
			return nil, fmt.Errorf("run %s not found", req.RunId)
		}
		if _, ok := effortRelationship(run, req.EffortRef); !ok {
			return nil, fmt.Errorf("run %s does not carry effort reference %s", req.RunId, req.EffortRef)
		}
	}
	signal, inserted, err := s.repo.AppendEffortSignal(ctx, req.EffortRef, req.Kind, req.RunId, req.Detail, req.DedupeKey)
	if err != nil {
		return nil, err
	}
	signal.NewlyRecorded = &inserted
	return signal, nil
}

// effortRelationship returns the run's relationship to the effort when it
// carries a public, active, verified reference to it.
func effortRelationship(run *domain.Run, ref string) (string, bool) {
	for _, work := range run.WorkReferences {
		if publicActiveEffortReference(work) && work.GetId() == ref {
			return work.GetRelationship(), true
		}
	}
	return "", false
}

// sweepEndedRuns derives child_run_ended and cap_stop signals from terminal
// runs that carry an effort reference. It reads the run owner's ended_at
// window, so every terminal path (executor, stop, reconciler, interactive)
// is covered after a restart too. A run stopped at its token cap gives one
// cap_stop for every effort it names, including an orchestrator's own; other
// terminal runs give child_run_ended unless they are the effort's
// orchestrator or supervisor, whose ends are not child events.
func (s *EffortService) sweepEndedRuns(ctx context.Context) error {
	if s.runRegistry == nil {
		return nil
	}
	now := s.now().UTC()
	cursor, err := s.repo.EffortSignalCursor(ctx)
	if err != nil {
		return err
	}
	from := now.Add(-signalSweepBackfill)
	if !cursor.IsZero() {
		from = cursor.Add(-signalSweepOverlap)
	}
	for page := 0; page < signalSweepPages; page++ {
		runs, err := s.runRegistry.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: signalSweepPage, Offset: page * signalSweepPage}, EndedFrom: &from})
		if err != nil {
			return err
		}
		for _, run := range runs {
			if err := s.signalEndedRun(ctx, run); err != nil {
				return err
			}
		}
		if len(runs) < signalSweepPage {
			break
		}
	}
	return s.repo.SaveEffortSignalCursor(ctx, now)
}

func (s *EffortService) signalEndedRun(ctx context.Context, run *domain.Run) error {
	if run == nil || run.EndedAt == nil || !run.Status.IsTerminal() {
		return nil
	}
	id := run.ID.String()
	seen := map[string]bool{}
	for _, work := range run.WorkReferences {
		if !publicActiveEffortReference(work) || seen[work.GetId()] {
			continue
		}
		seen[work.GetId()] = true
		role := work.GetRelationship()
		if run.StopReason == domain.RunStopReasonTokenCap {
			detail := fmt.Sprintf("%s run %s stopped at its weighted-token cap (stop_reason=%s)", role, id, run.StopReason)
			if _, _, err := s.repo.AppendEffortSignal(ctx, work.GetId(), SignalCapStop, id, detail, SignalCapStop+":"+id); err != nil {
				return err
			}
			continue
		}
		if role == "orchestrator" || role == "supervisor" {
			continue
		}
		detail := fmt.Sprintf("%s run %s ended %s", role, id, run.Status)
		if run.StopReason != "" {
			detail += " (stop_reason=" + string(run.StopReason) + ")"
		}
		if _, _, err := s.repo.AppendEffortSignal(ctx, work.GetId(), SignalChildRunEnded, id, detail, SignalChildRunEnded+":"+id); err != nil {
			return err
		}
	}
	return nil
}
