package programs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"program-runtime/internal/sessions"
)

// SQLExecutor is shared with the sessions domain so repositories stay behind
// the same narrow, context-aware database seam.
type SQLExecutor = sessions.SQLExecutor

type Repository interface {
	Save(context.Context, *programsv1.Program) error
	Get(context.Context, string) (*programsv1.Program, error)
	List(context.Context, string, bool) ([]*programsv1.Program, error)
	MineFailures(context.Context, bool, time.Time) ([]*programsv1.FailureShape, error)
	MineRefusals(context.Context, bool) ([]*programsv1.RefusalShape, error)
	MineUnresolvedBindings(context.Context) ([]*programsv1.UnresolvedBindingShape, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Save(ctx context.Context, p *programsv1.Program) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO programs
	 (id, session_id, source, provenance, status, created_at, completed_at, stdout, context_bytes, agent_bytes, output_limit_bytes, failure_detail, failure_shape, failure_location, wall_time_millis, cpu_time_millis, library_version, failure_cause)
	 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	 ON CONFLICT(id) DO UPDATE SET status=excluded.status, completed_at=excluded.completed_at, stdout=excluded.stdout, context_bytes=excluded.context_bytes, agent_bytes=excluded.agent_bytes, output_limit_bytes=excluded.output_limit_bytes, failure_detail=excluded.failure_detail, failure_shape=excluded.failure_shape, failure_location=excluded.failure_location, wall_time_millis=excluded.wall_time_millis, cpu_time_millis=excluded.cpu_time_millis, library_version=excluded.library_version, failure_cause=excluded.failure_cause`,
		p.GetId(), p.GetSessionId(), p.GetSource(), strconv.Itoa(int(p.GetProvenance())), statusName(p.GetStatus()), p.GetCreatedAt(), p.GetCompletedAt(), p.GetStdout(), p.GetContextBytes(), p.GetAgentBytes(), p.GetOutputLimitBytes(), p.GetFailureDetail(), p.GetFailureShape(), failureLocation(p.GetFailureDetail()), p.GetWallTimeMillis(), p.GetCpuTimeMillis(), p.GetLibraryVersion(), p.GetFailureCause().String())
	if err != nil {
		return fmt.Errorf("save program %q: %w", p.GetId(), err)
	}
	return nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (*programsv1.Program, error) {
	p, err := r.scan(r.db.QueryRowContext(ctx, `SELECT id, session_id, source, provenance, status, created_at, completed_at, stdout, context_bytes, agent_bytes, output_limit_bytes, failure_detail, failure_shape, wall_time_millis, cpu_time_millis, library_version, failure_cause FROM programs WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProgramNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get program %q: %w", id, err)
	}
	return p, nil
}

func (r *sqliteRepository) List(ctx context.Context, sessionID string, includeOperator bool) ([]*programsv1.Program, error) {
	query := `SELECT id, session_id, source, provenance, status, created_at, completed_at, stdout, context_bytes, agent_bytes, output_limit_bytes, failure_detail, failure_shape, wall_time_millis, cpu_time_millis, library_version, failure_cause FROM programs WHERE 1=1`
	args := make([]any, 0, 2)
	if sessionID != "" {
		query += ` AND session_id = ?`
		args = append(args, sessionID)
	}
	if !includeOperator {
		query += ` AND provenance != ?`
		args = append(args, strconv.Itoa(int(programsv1.Provenance_PROVENANCE_OPERATOR)))
	}
	query += ` ORDER BY created_at, id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list programs: %w", err)
	}
	defer rows.Close()
	var out []*programsv1.Program
	for rows.Next() {
		p, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate programs: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) MineFailures(ctx context.Context, includeOperator bool, since time.Time) ([]*programsv1.FailureShape, error) {
	query := `SELECT failure_shape, COUNT(*), MIN(created_at), MAX(created_at), (SELECT p2.id FROM programs p2 WHERE p2.failure_shape = p.failure_shape AND p2.status = 'failed' ORDER BY p2.created_at DESC, p2.id DESC LIMIT 1) FROM programs p WHERE p.status = 'failed' AND p.failure_shape != ''`
	args := make([]any, 0, 2)
	if !includeOperator {
		query += ` AND provenance != ?`
		args = append(args, strconv.Itoa(int(programsv1.Provenance_PROVENANCE_OPERATOR)))
	}
	if !since.IsZero() {
		query += ` AND created_at >= ?`
		args = append(args, since.UTC().Format(time.RFC3339Nano))
	}
	query += ` GROUP BY failure_shape ORDER BY failure_shape`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mine program failures: %w", err)
	}
	defer rows.Close()
	var out []*programsv1.FailureShape
	for rows.Next() {
		var shape, firstSeen, lastSeen, sampleID string
		var count int64
		if err := rows.Scan(&shape, &count, &firstSeen, &lastSeen, &sampleID); err != nil {
			return nil, fmt.Errorf("scan failure shape: %w", err)
		}
		out = append(out, &programsv1.FailureShape{Shape: shape, Count: count, FirstSeen: firstSeen, LastSeen: lastSeen, SampleProgramId: sampleID})
	}
	return out, rows.Err()
}

func (r *sqliteRepository) MineRefusals(ctx context.Context, _ bool) ([]*programsv1.RefusalShape, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT binding_id, reason, COUNT(*), MAX(occurred_at) FROM refusals GROUP BY binding_id, reason ORDER BY COUNT(*) DESC, binding_id, reason`)
	if err != nil {
		return nil, fmt.Errorf("mine binding refusals: %w", err)
	}
	defer rows.Close()
	var out []*programsv1.RefusalShape
	for rows.Next() {
		shape := &programsv1.RefusalShape{}
		if err := rows.Scan(&shape.BindingId, &shape.Reason, &shape.Count, &shape.LastSeen); err != nil {
			return nil, fmt.Errorf("scan refusal shape: %w", err)
		}
		out = append(out, shape)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) MineUnresolvedBindings(ctx context.Context) ([]*programsv1.UnresolvedBindingShape, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT attempted_name, COUNT(*), MAX(occurred_at) FROM unresolved_binding_attempts GROUP BY attempted_name ORDER BY COUNT(*) DESC, attempted_name`)
	if err != nil {
		return nil, fmt.Errorf("mine unresolved binding attempts: %w", err)
	}
	defer rows.Close()
	var out []*programsv1.UnresolvedBindingShape
	for rows.Next() {
		shape := &programsv1.UnresolvedBindingShape{}
		if err := rows.Scan(&shape.AttemptedName, &shape.Count, &shape.LastSeen); err != nil {
			return nil, fmt.Errorf("scan unresolved binding shape: %w", err)
		}
		out = append(out, shape)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func (r *sqliteRepository) scan(row rowScanner) (*programsv1.Program, error) {
	var p programsv1.Program
	var provenance string
	var status, completedAt string
	var failureCause string
	if err := row.Scan(&p.Id, &p.SessionId, &p.Source, &provenance, &status, &p.CreatedAt, &completedAt, &p.Stdout, &p.ContextBytes, &p.AgentBytes, &p.OutputLimitBytes, &p.FailureDetail, &p.FailureShape, &p.WallTimeMillis, &p.CpuTimeMillis, &p.LibraryVersion, &failureCause); err != nil {
		return nil, err
	}
	p.Status = parseStatus(status)
	p.CompletedAt = completedAt
	p.FailureCause = parseFailureCause(failureCause)
	value, err := strconv.Atoi(provenance)
	if err != nil {
		return nil, fmt.Errorf("parse program provenance: %w", err)
	}
	p.Provenance = programsv1.Provenance(value)
	return &p, nil
}

func statusName(status programsv1.ProgramStatus) string {
	switch status {
	case programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED:
		return "accepted"
	case programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING:
		return "running"
	case programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED:
		return "succeeded"
	case programsv1.ProgramStatus_PROGRAM_STATUS_FAILED:
		return "failed"
	case programsv1.ProgramStatus_PROGRAM_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unspecified"
	}
}

func parseStatus(value string) programsv1.ProgramStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "accepted":
		return programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED
	case "running":
		return programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING
	case "succeeded":
		return programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED
	case "failed":
		return programsv1.ProgramStatus_PROGRAM_STATUS_FAILED
	case "cancelled", "canceled":
		return programsv1.ProgramStatus_PROGRAM_STATUS_CANCELLED
	default:
		return programsv1.ProgramStatus_PROGRAM_STATUS_UNSPECIFIED
	}
}

func parseFailureCause(value string) programsv1.FailureCause {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "failure_cause_unresolved_name", "unresolved_name":
		return programsv1.FailureCause_FAILURE_CAUSE_UNRESOLVED_NAME
	case "failure_cause_unknown_field", "unknown_field":
		return programsv1.FailureCause_FAILURE_CAUSE_UNKNOWN_FIELD
	case "failure_cause_ambiguous_response", "ambiguous_response":
		return programsv1.FailureCause_FAILURE_CAUSE_AMBIGUOUS_RESPONSE
	case "failure_cause_unreachable_scenario", "unreachable_scenario":
		return programsv1.FailureCause_FAILURE_CAUSE_UNREACHABLE_SCENARIO
	case "failure_cause_refused_no_grant", "refused_no_grant":
		return programsv1.FailureCause_FAILURE_CAUSE_REFUSED_NO_GRANT
	case "failure_cause_refused_not_run_eligible", "refused_not_run_eligible":
		return programsv1.FailureCause_FAILURE_CAUSE_REFUSED_NOT_RUN_ELIGIBLE
	case "failure_cause_inference_spend_exceeded", "inference_spend_exceeded":
		return programsv1.FailureCause_FAILURE_CAUSE_INFERENCE_SPEND_EXCEEDED
	case "failure_cause_delegated_run_spend_exceeded", "delegated_run_spend_exceeded":
		return programsv1.FailureCause_FAILURE_CAUSE_DELEGATED_RUN_SPEND_EXCEEDED
	case "failure_cause_deadline_exceeded", "deadline_exceeded":
		return programsv1.FailureCause_FAILURE_CAUSE_DEADLINE_EXCEEDED
	case "failure_cause_kernel_syntax", "kernel_syntax":
		return programsv1.FailureCause_FAILURE_CAUSE_KERNEL_SYNTAX
	case "failure_cause_kernel_runtime", "kernel_runtime":
		return programsv1.FailureCause_FAILURE_CAUSE_KERNEL_RUNTIME
	case "failure_cause_bridge_transport", "bridge_transport":
		return programsv1.FailureCause_FAILURE_CAUSE_BRIDGE_TRANSPORT
	case "failure_cause_unclassified", "unclassified":
		return programsv1.FailureCause_FAILURE_CAUSE_UNCLASSIFIED
	default:
		return programsv1.FailureCause_FAILURE_CAUSE_UNSPECIFIED
	}
}

type memoryRepository struct {
	mu       sync.RWMutex
	programs map[string]*programsv1.Program
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{programs: make(map[string]*programsv1.Program)}
}

func (r *memoryRepository) Save(_ context.Context, p *programsv1.Program) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.programs[p.GetId()] = clone(p)
	return nil
}

func (r *memoryRepository) Get(_ context.Context, id string) (*programsv1.Program, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.programs[id]
	if !ok {
		return nil, ErrProgramNotFound
	}
	return clone(p), nil
}

func (r *memoryRepository) List(_ context.Context, sessionID string, includeOperator bool) ([]*programsv1.Program, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*programsv1.Program
	for _, p := range r.programs {
		if sessionID != "" && p.GetSessionId() != sessionID {
			continue
		}
		if !includeOperator && p.GetProvenance() == programsv1.Provenance_PROVENANCE_OPERATOR {
			continue
		}
		out = append(out, clone(p))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GetCreatedAt() < out[j].GetCreatedAt() })
	return out, nil
}

func (r *memoryRepository) MineFailures(ctx context.Context, includeOperator bool, since time.Time) ([]*programsv1.FailureShape, error) {
	list, err := r.List(ctx, "", includeOperator)
	if err != nil {
		return nil, err
	}
	type aggregate struct {
		count       int64
		first, last string
		sample      string
	}
	counts := map[string]*aggregate{}
	for _, p := range list {
		if p.GetStatus() == programsv1.ProgramStatus_PROGRAM_STATUS_FAILED && p.GetFailureShape() != "" && (since.IsZero() || p.GetCreatedAt() >= since.UTC().Format(time.RFC3339Nano)) {
			item := counts[p.GetFailureShape()]
			if item == nil {
				item = &aggregate{first: p.GetCreatedAt(), last: p.GetCreatedAt(), sample: p.GetId()}
				counts[p.GetFailureShape()] = item
			}
			item.count++
			if p.GetCreatedAt() < item.first {
				item.first = p.GetCreatedAt()
			}
			if p.GetCreatedAt() >= item.last {
				item.last, item.sample = p.GetCreatedAt(), p.GetId()
			}
		}
	}
	out := make([]*programsv1.FailureShape, 0, len(counts))
	for shape, item := range counts {
		out = append(out, &programsv1.FailureShape{Shape: shape, Count: item.count, FirstSeen: item.first, LastSeen: item.last, SampleProgramId: item.sample})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GetShape() < out[j].GetShape() })
	return out, nil
}

func (r *memoryRepository) MineRefusals(context.Context, bool) ([]*programsv1.RefusalShape, error) {
	return nil, nil
}

func (r *memoryRepository) MineUnresolvedBindings(context.Context) ([]*programsv1.UnresolvedBindingShape, error) {
	return nil, nil
}
