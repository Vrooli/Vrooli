package review

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const findingColumns = `id, subject_kind, subject_id, gate, severity, status, message, remediation, evidence_json, docs_ref, owner, created_at`

// Service is the review queue owner. It refreshes findings from a Projector,
// but status and owner are human state: the upsert never overwrites them.
type Service struct {
	db        *sql.DB
	projector Projector
	clock     func() time.Time
	random    io.Reader
	ttl       time.Duration

	refreshMu   sync.Mutex
	lastRefresh time.Time
	refreshing  atomic.Bool
}

// NewService wires the durable queue to a projection source. A nil projector
// leaves the stored queue untouched; that is useful for read-only consumers
// and tests, not for production wiring.
func NewService(db *sql.DB, projector Projector) *Service {
	return &Service{
		db:        db,
		projector: projector,
		clock:     time.Now,
		random:    rand.Reader,
		ttl:       defaultProjectionTTL,
	}
}

// WithClock replaces the time source. It is used by tests and by callers that
// need deterministic generated timestamps.
func (s *Service) WithClock(clock func() time.Time) *Service {
	if clock != nil {
		s.clock = clock
	}
	return s
}

// WithTTL replaces the projection refresh window. A non-positive duration
// forces a refresh on every read that asks for freshness.
func (s *Service) WithTTL(ttl time.Duration) *Service {
	s.ttl = ttl
	return s
}

// List returns one page of findings newest-first plus the filtered total. It
// refreshes the projection first when the TTL has elapsed or the queue is
// empty. The count and the page use the same WHERE clause so the total always
// describes the filter, not the window.
func (s *Service) List(ctx context.Context, filter Filter, limit, offset int) (Page, error) {
	if s == nil || s.db == nil {
		return Page{}, errors.New("review service is not configured")
	}
	if filter.Status != "" && !ValidStatus(filter.Status) {
		return Page{}, fmt.Errorf("%w: status %q is not recognized", ErrInvalid, filter.Status)
	}
	if filter.Severity != "" && !ValidSeverity(filter.Severity) {
		return Page{}, fmt.Errorf("%w: severity %q is not recognized", ErrInvalid, filter.Severity)
	}
	if offset < 0 {
		return Page{}, fmt.Errorf("%w: offset must not be negative", ErrInvalid)
	}
	if err := s.ensureFresh(ctx); err != nil {
		return Page{}, err
	}

	limit = clampLimit(limit)
	conditions := make([]string, 0, 4)
	args := make([]any, 0, 6)
	if status := strings.TrimSpace(filter.Status); status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	if severity := strings.TrimSpace(filter.Severity); severity != "" {
		conditions = append(conditions, "severity = ?")
		args = append(args, severity)
	}
	if kind := strings.TrimSpace(filter.SubjectKind); kind != "" {
		conditions = append(conditions, "subject_kind = ?")
		args = append(args, kind)
	}
	if search := strings.TrimSpace(filter.Query); search != "" {
		pattern := "%" + escapeLike(search) + "%"
		conditions = append(conditions, `(message LIKE ? ESCAPE '\' OR gate LIKE ? ESCAPE '\' OR subject_id LIKE ? ESCAPE '\' OR remediation LIKE ? ESCAPE '\')`)
		args = append(args, pattern, pattern, pattern, pattern)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM findings`+where, args...).Scan(&total); err != nil {
		return Page{}, fmt.Errorf("count findings: %w", err)
	}

	pageArgs := append(append([]any{}, args...), limit, offset)
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+findingColumns+` FROM findings`+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`,
		pageArgs...)
	if err != nil {
		return Page{}, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()

	findings := make([]Finding, 0)
	ids := make([]string, 0)
	for rows.Next() {
		finding, err := scanFinding(rows)
		if err != nil {
			return Page{}, fmt.Errorf("scan finding: %w", err)
		}
		findings = append(findings, finding)
		ids = append(ids, finding.ID)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("iterate findings: %w", err)
	}
	byFinding, err := s.dispositionsFor(ctx, ids)
	if err != nil {
		return Page{}, err
	}
	for index := range findings {
		findings[index].Dispositions = byFinding[findings[index].ID]
	}
	return Page{Findings: findings, Total: total}, nil
}

// Get returns one finding with its full disposition history.
func (s *Service) Get(ctx context.Context, id string) (Finding, error) {
	if s == nil || s.db == nil {
		return Finding{}, errors.New("review service is not configured")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Finding{}, fmt.Errorf("%w: id is required", ErrInvalid)
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+findingColumns+` FROM findings WHERE id = ?`, id)
	finding, err := scanFinding(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Finding{}, ErrNotFound
	}
	if err != nil {
		return Finding{}, fmt.Errorf("get finding %q: %w", id, err)
	}
	byFinding, err := s.dispositionsFor(ctx, []string{id})
	if err != nil {
		return Finding{}, err
	}
	finding.Dispositions = byFinding[id]
	return finding, nil
}

// RecordDisposition appends the decision and applies its status transition in
// one transaction. Reason and actor are required: an unexplained triage write
// is not durable authority.
func (s *Service) RecordDisposition(ctx context.Context, id, decision, reason, actor string) (Finding, error) {
	if s == nil || s.db == nil {
		return Finding{}, errors.New("review service is not configured")
	}
	id = strings.TrimSpace(id)
	decision = strings.TrimSpace(decision)
	reason = strings.TrimSpace(reason)
	actor = strings.TrimSpace(actor)
	switch {
	case id == "":
		return Finding{}, fmt.Errorf("%w: id is required", ErrInvalid)
	case !ValidDecision(decision):
		return Finding{}, fmt.Errorf("%w: decision %q is not recognized", ErrInvalid, decision)
	case reason == "":
		return Finding{}, fmt.Errorf("%w: reason is required", ErrInvalid)
	case actor == "":
		return Finding{}, fmt.Errorf("%w: actor is required", ErrInvalid)
	}
	status := StatusForDecision(decision)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Finding{}, fmt.Errorf("begin disposition: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM findings WHERE id = ?`, id).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Finding{}, ErrNotFound
		}
		return Finding{}, fmt.Errorf("check finding %q: %w", id, err)
	}
	at := s.now()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO finding_dispositions(id, finding_id, decision, reason, actor, at) VALUES (?, ?, ?, ?, ?, ?)`,
		s.newID(at), id, decision, reason, actor, at.Format(timestampLayout)); err != nil {
		return Finding{}, fmt.Errorf("record disposition: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE findings SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return Finding{}, fmt.Errorf("apply disposition status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Finding{}, fmt.Errorf("apply disposition status rows affected: %w", err)
	}
	if affected == 0 {
		return Finding{}, ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return Finding{}, fmt.Errorf("commit disposition: %w", err)
	}
	return s.Get(ctx, id)
}

// Assign sets the accountable owner. An empty owner clears the assignment;
// both forms are explicit writes so the UI never has to guess.
func (s *Service) Assign(ctx context.Context, id, owner string) (Finding, error) {
	if s == nil || s.db == nil {
		return Finding{}, errors.New("review service is not configured")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Finding{}, fmt.Errorf("%w: id is required", ErrInvalid)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE findings SET owner = ? WHERE id = ?`, strings.TrimSpace(owner), id)
	if err != nil {
		return Finding{}, fmt.Errorf("assign finding %q: %w", id, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Finding{}, fmt.Errorf("assign finding rows affected: %w", err)
	}
	if affected == 0 {
		return Finding{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

// Summary returns queue counts. It refreshes like List so Stats does not show
// a stale queue while the TTL is being observed.
func (s *Service) Summary(ctx context.Context) (Summary, error) {
	if s == nil || s.db == nil {
		return Summary{}, errors.New("review service is not configured")
	}
	if err := s.ensureFresh(ctx); err != nil {
		return Summary{}, err
	}
	var summary Summary
	err := s.db.QueryRowContext(ctx, `
SELECT
  COALESCE(SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN severity = 'blocker' AND status IN ('open', 'acknowledged') THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN severity = 'high' AND status IN ('open', 'acknowledged') THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'acknowledged' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'fixed' THEN 1 ELSE 0 END), 0)
FROM findings`).Scan(
		&summary.Open, &summary.Blockers, &summary.High,
		&summary.Acknowledged, &summary.Accepted, &summary.Fixed,
	)
	if err != nil {
		return Summary{}, fmt.Errorf("summarize findings: %w", err)
	}
	return summary, nil
}

// ensureFresh is the only projection trigger. It refreshes when the TTL has
// elapsed or when the queue has no rows, so a first page load does not depend
// on a background job having run.
// ensureFresh keeps reads fast on a large backlog. A first projection and an
// explicitly fresh read (TTL <= 0) run inline; a later TTL expiry refreshes in
// the background and returns the last projection immediately. Projecting
// ~14k rows inside a request made every first page load after the TTL pay a
// multi-second projection.
func (s *Service) ensureFresh(ctx context.Context) error {
	if s.projector == nil {
		return nil
	}
	s.refreshMu.Lock()
	last := s.lastRefresh
	s.refreshMu.Unlock()

	now := s.now()
	if last.IsZero() || s.ttl <= 0 {
		return s.refresh(ctx)
	}
	if now.Sub(last) < s.ttl {
		var count int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM findings`).Scan(&count); err != nil {
			return fmt.Errorf("count findings before refresh: %w", err)
		}
		if count > 0 {
			return nil
		}
		return s.refresh(ctx)
	}
	s.refreshInBackground()
	return nil
}

// refresh projects and upserts once. It re-checks the TTL inside the lock so a
// queued background run and a request cannot both project the same corpus.
func (s *Service) refresh(ctx context.Context) error {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	now := s.now()
	if !s.lastRefresh.IsZero() && s.ttl > 0 && now.Sub(s.lastRefresh) < s.ttl {
		return nil
	}
	projected, err := s.projector.Project(ctx)
	if err != nil {
		return fmt.Errorf("project findings: %w", err)
	}
	if err := s.upsert(ctx, projected, now); err != nil {
		return err
	}
	s.lastRefresh = now
	return nil
}

func (s *Service) refreshInBackground() {
	if !s.refreshing.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer s.refreshing.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		// A failed background projection leaves the previous queue in place and
		// the next TTL window retries; a request must never see a partial queue.
		_ = s.refresh(ctx)
	}()
}

// upsert applies only projection-owned columns on conflict. status and owner
// are deliberately absent from the DO UPDATE clause: a gate rerun must not
// erase an acknowledgement or an assignment.
func (s *Service) upsert(ctx context.Context, projected []Finding, now time.Time) error {
	if len(projected) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin finding projection: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, candidate := range projected {
		finding, err := normalizeProjected(candidate, now)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO findings (id, subject_kind, subject_id, gate, severity, status, message, remediation, evidence_json, docs_ref, owner, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  subject_kind = excluded.subject_kind,
  subject_id = excluded.subject_id,
  gate = excluded.gate,
  severity = excluded.severity,
  message = excluded.message,
  remediation = excluded.remediation,
  evidence_json = excluded.evidence_json,
  docs_ref = excluded.docs_ref`,
			finding.ID, finding.SubjectKind, finding.SubjectID, finding.Gate,
			finding.Severity, finding.Status, finding.Message, finding.Remediation,
			finding.EvidenceJSON, finding.DocsRef, finding.Owner, finding.CreatedAt); err != nil {
			return fmt.Errorf("upsert finding %q: %w", finding.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit finding projection: %w", err)
	}
	return nil
}

func normalizeProjected(finding Finding, now time.Time) (Finding, error) {
	finding.ID = strings.TrimSpace(finding.ID)
	finding.SubjectKind = strings.TrimSpace(finding.SubjectKind)
	finding.SubjectID = strings.TrimSpace(finding.SubjectID)
	finding.Gate = strings.TrimSpace(finding.Gate)
	finding.Severity = strings.TrimSpace(finding.Severity)
	finding.Status = strings.TrimSpace(finding.Status)
	finding.Message = strings.TrimSpace(finding.Message)
	finding.Owner = strings.TrimSpace(finding.Owner)
	finding.DocsRef = strings.TrimSpace(finding.DocsRef)

	switch {
	case finding.ID == "":
		return Finding{}, errors.New("projector returned a finding without id")
	case finding.SubjectKind == "":
		return Finding{}, fmt.Errorf("projector returned finding %q without subject_kind", finding.ID)
	case finding.SubjectID == "":
		return Finding{}, fmt.Errorf("projector returned finding %q without subject_id", finding.ID)
	case finding.Gate == "":
		return Finding{}, fmt.Errorf("projector returned finding %q without gate", finding.ID)
	case finding.Message == "":
		return Finding{}, fmt.Errorf("projector returned finding %q without message", finding.ID)
	}
	if finding.Severity == "" {
		finding.Severity = SeverityMedium
	}
	if !ValidSeverity(finding.Severity) {
		return Finding{}, fmt.Errorf("projector returned finding %q with unknown severity %q", finding.ID, finding.Severity)
	}
	if finding.Status == "" {
		finding.Status = StatusOpen
	}
	if !ValidStatus(finding.Status) {
		return Finding{}, fmt.Errorf("projector returned finding %q with unknown status %q", finding.ID, finding.Status)
	}
	if finding.EvidenceJSON == "" {
		finding.EvidenceJSON = "{}"
	}
	if finding.CreatedAt == "" {
		finding.CreatedAt = now.Format(timestampLayout)
	} else if parsed, err := time.Parse(time.RFC3339Nano, finding.CreatedAt); err == nil {
		finding.CreatedAt = parsed.UTC().Format(timestampLayout)
	}
	return finding, nil
}

func (s *Service) dispositionsFor(ctx context.Context, ids []string) (map[string][]Disposition, error) {
	byFinding := make(map[string][]Disposition, len(ids))
	if len(ids) == 0 {
		return byFinding, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for index, id := range ids {
		args[index] = id
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, finding_id, decision, reason, actor, at FROM finding_dispositions
		 WHERE finding_id IN (`+placeholders+`)
		 ORDER BY at DESC, id DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list finding dispositions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var disposition Disposition
		if err := rows.Scan(&disposition.ID, &disposition.FindingID, &disposition.Decision,
			&disposition.Reason, &disposition.Actor, &disposition.At); err != nil {
			return nil, fmt.Errorf("scan finding disposition: %w", err)
		}
		byFinding[disposition.FindingID] = append(byFinding[disposition.FindingID], disposition)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate finding dispositions: %w", err)
	}
	return byFinding, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFinding(scanner rowScanner) (Finding, error) {
	var finding Finding
	err := scanner.Scan(
		&finding.ID, &finding.SubjectKind, &finding.SubjectID, &finding.Gate,
		&finding.Severity, &finding.Status, &finding.Message, &finding.Remediation,
		&finding.EvidenceJSON, &finding.DocsRef, &finding.Owner, &finding.CreatedAt,
	)
	return finding, err
}

func (s *Service) now() time.Time {
	if s.clock != nil {
		return s.clock().UTC()
	}
	return time.Now().UTC()
}

// newID combines a fixed-width timestamp with random hex. Disposition writes
// are not required to be idempotent by the proto, but the deterministic prefix
// keeps the history ordered under lexical id ties.
func (s *Service) newID(at time.Time) string {
	suffix := make([]byte, dispositionIDBytes)
	if s.random != nil {
		_, _ = io.ReadFull(s.random, suffix)
	} else {
		_, _ = rand.Read(suffix)
	}
	return fmt.Sprintf("dsp_%016x%s", uint64(at.UnixNano()), hex.EncodeToString(suffix))
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
