package findings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"web-search/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface the repository depends on. Both
// *sql.DB (unit tests) and *database.RoutedDB (production) satisfy it, so the
// production wiring participates in per-request routing without forcing the
// test fixture to wrap its handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// findingTimeFormat sorts lexicographically in time order for a fixed zone, so
// string-range comparisons on the TEXT timestamp columns are correct filters.
const findingTimeFormat = time.RFC3339Nano

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

func (s *sqliteRepository) now() time.Time { return s.clock.Now().UTC() }

func (s *sqliteRepository) writeAudit(ctx context.Context, findingID, mutation, reason, sourceBrief, actor string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO finding_audit (id, finding_id, mutation_type, reason, source_brief_id, actor, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), findingID, mutation, reason, sourceBrief, actor,
		s.now().Format(findingTimeFormat),
	)
	if err != nil {
		return fmt.Errorf("write audit (%s) for finding %q: %w", mutation, findingID, err)
	}
	return nil
}

func (s *sqliteRepository) Add(ctx context.Context, in NewFinding, actor string) (Finding, error) {
	now := s.now()
	f := Finding{
		ID:            uuid.NewString(),
		Claim:         in.Claim,
		BriefID:       in.BriefID,
		Confidence:    in.Confidence,
		Status:        StatusActive,
		RetrievalDate: now,
		Query:         in.Query,
		Source:        normalizeSource(in.Source),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO findings (id, claim, brief_id, confidence, status, retrieval_date, query, superseded_by, dispute_note, source, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, '', '', ?, ?, ?)`,
		f.ID, f.Claim, f.BriefID, f.Confidence, f.Status,
		f.RetrievalDate.Format(findingTimeFormat), f.Query, f.Source,
		f.CreatedAt.Format(findingTimeFormat), f.UpdatedAt.Format(findingTimeFormat),
	)
	if err != nil {
		return Finding{}, fmt.Errorf("insert finding %q: %w", f.ID, err)
	}
	for _, c := range in.Citations {
		cit := Citation{ID: uuid.NewString(), URL: c.URL, Title: c.Title, RetrievedAt: now}
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO finding_citations (id, finding_id, url, title, retrieved_at) VALUES (?, ?, ?, ?, ?)`,
			cit.ID, f.ID, cit.URL, cit.Title, cit.RetrievedAt.Format(findingTimeFormat),
		)
		if err != nil {
			return Finding{}, fmt.Errorf("insert citation for finding %q: %w", f.ID, err)
		}
		f.Citations = append(f.Citations, cit)
	}
	if err := s.writeAudit(ctx, f.ID, MutationCreate, "", in.BriefID, actor); err != nil {
		return Finding{}, err
	}
	return f, nil
}

const selectFindingColumns = `id, claim, brief_id, confidence, status, retrieval_date, query, superseded_by, dispute_note, source, created_at, updated_at`

func (s *sqliteRepository) Get(ctx context.Context, id string) (Finding, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+selectFindingColumns+` FROM findings WHERE id = ?`, id)
	f, err := scanFinding(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Finding{}, ErrFindingNotFound{ID: id}
	}
	if err != nil {
		return Finding{}, fmt.Errorf("get finding %q: %w", id, err)
	}
	cites, err := s.loadCitations(ctx, []string{id})
	if err != nil {
		return Finding{}, err
	}
	f.Citations = cites[id]
	return f, nil
}

func (s *sqliteRepository) GetMany(ctx context.Context, ids []string) (map[string]Finding, error) {
	out := make(map[string]Finding, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+selectFindingColumns+` FROM findings WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("get many findings: %w", err)
	}
	found, err := scanFindingRows(rows)
	if err != nil {
		return nil, err
	}
	idList := make([]string, 0, len(found))
	for _, f := range found {
		idList = append(idList, f.ID)
	}
	cites, err := s.loadCitations(ctx, idList)
	if err != nil {
		return nil, err
	}
	for _, f := range found {
		f.Citations = cites[f.ID]
		out[f.ID] = f
	}
	return out, nil
}

func (s *sqliteRepository) List(ctx context.Context, f ListFilter) ([]Finding, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	var where []string
	var args []any
	switch {
	case f.Status != "":
		where = append(where, "status = ?")
		args = append(args, f.Status)
	case !f.IncludeArchived:
		where = append(where, "status != ?")
		args = append(args, StatusSuperseded)
	}
	q := `SELECT ` + selectFindingColumns + ` FROM findings`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY created_at DESC, id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	return s.hydrate(ctx, rows)
}

func (s *sqliteRepository) Edit(ctx context.Context, id string, in EditInput, actor string) (Finding, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return Finding{}, err
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE findings SET claim = ?, confidence = ?, updated_at = ? WHERE id = ?`,
		in.Claim, in.Confidence, s.now().Format(findingTimeFormat), id,
	)
	if err != nil {
		return Finding{}, fmt.Errorf("edit finding %q: %w", id, err)
	}
	if err := s.writeAudit(ctx, id, MutationEdit, "", "", actor); err != nil {
		return Finding{}, err
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) Supersede(ctx context.Context, id, replacement, reason, actor string) (Finding, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return Finding{}, err
	}
	if replacement != "" {
		if _, err := s.Get(ctx, replacement); err != nil {
			if errors.As(err, &ErrFindingNotFound{}) {
				return Finding{}, ErrInvalidFinding{Field: "replacement", Reason: fmt.Sprintf("replacement finding %q not found", replacement)}
			}
			return Finding{}, err
		}
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE findings SET status = ?, superseded_by = ?, updated_at = ? WHERE id = ?`,
		StatusSuperseded, replacement, s.now().Format(findingTimeFormat), id,
	)
	if err != nil {
		return Finding{}, fmt.Errorf("supersede finding %q: %w", id, err)
	}
	if err := s.writeAudit(ctx, id, MutationSupersede, reason, "", actor); err != nil {
		return Finding{}, err
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) Flag(ctx context.Context, id, reason, actor string) (Finding, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return Finding{}, err
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE findings SET status = ?, dispute_note = ?, updated_at = ? WHERE id = ?`,
		StatusDisputed, reason, s.now().Format(findingTimeFormat), id,
	)
	if err != nil {
		return Finding{}, fmt.Errorf("flag finding %q: %w", id, err)
	}
	if err := s.writeAudit(ctx, id, MutationFlag, reason, "", actor); err != nil {
		return Finding{}, err
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) Resolve(ctx context.Context, id, reason, actor string) (Finding, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return Finding{}, err
	}
	if existing.Status != StatusDisputed {
		return Finding{}, ErrInvalidFinding{Field: "id", Reason: fmt.Sprintf("finding %q is not disputed (status=%s)", id, existing.Status)}
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE findings SET status = ?, dispute_note = '', updated_at = ? WHERE id = ?`,
		StatusActive, s.now().Format(findingTimeFormat), id,
	)
	if err != nil {
		return Finding{}, fmt.Errorf("resolve finding %q: %w", id, err)
	}
	if err := s.writeAudit(ctx, id, MutationResolve, reason, "", actor); err != nil {
		return Finding{}, err
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) Prune(ctx context.Context, dryRun bool, actor string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM findings WHERE status = ? ORDER BY updated_at ASC`, StatusSuperseded)
	if err != nil {
		return nil, fmt.Errorf("prune scan: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("prune scan id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("prune iterate: %w", err)
	}
	rows.Close()
	if dryRun {
		return ids, nil
	}
	for _, id := range ids {
		// Audit before delete so the trail records the prune even though the
		// finding row is gone afterward.
		if err := s.writeAudit(ctx, id, MutationPrune, "pruned superseded finding", "", actor); err != nil {
			return nil, err
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM finding_citations WHERE finding_id = ?`, id); err != nil {
			return nil, fmt.Errorf("prune delete citations %q: %w", id, err)
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM findings WHERE id = ?`, id); err != nil {
			return nil, fmt.Errorf("prune delete finding %q: %w", id, err)
		}
	}
	return ids, nil
}

func (s *sqliteRepository) Count(ctx context.Context, from, to time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM findings WHERE created_at >= ? AND created_at < ?`,
		from.UTC().Format(findingTimeFormat), to.UTC().Format(findingTimeFormat),
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count findings in [%s, %s): %w", from, to, err)
	}
	return n, nil
}

func (s *sqliteRepository) LoadIndexable(ctx context.Context) ([]Finding, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+selectFindingColumns+` FROM findings WHERE status != ? ORDER BY created_at DESC`, StatusSuperseded)
	if err != nil {
		return nil, fmt.Errorf("load indexable findings: %w", err)
	}
	return s.hydrate(ctx, rows)
}

func (s *sqliteRepository) SearchArchivedLike(ctx context.Context, query string, limit int) ([]Finding, error) {
	if limit <= 0 {
		limit = 20
	}
	like := "%" + strings.TrimSpace(query) + "%"
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+selectFindingColumns+` FROM findings WHERE status = ? AND claim LIKE ? ORDER BY updated_at DESC LIMIT ?`,
		StatusSuperseded, like, limit)
	if err != nil {
		return nil, fmt.Errorf("search archived findings: %w", err)
	}
	return s.hydrate(ctx, rows)
}

func (s *sqliteRepository) RecordSurfaced(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	stamp := s.now().Format(findingTimeFormat)
	// UPSERT: create the usage row at count 1 or bump an existing one. An id that
	// is not (or no longer) a real finding only ever produces an orphan usage row,
	// which is invisible to both read paths (GetUsage is keyed by live finding
	// ids; ListDecayCandidates LEFT JOINs from findings), so we do not pre-check
	// existence on this async hot-ish path.
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			continue
		}
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO finding_usage (finding_id, surfaced_count, used_count, last_surfaced_at)
			 VALUES (?, 1, 0, ?)
			 ON CONFLICT(finding_id) DO UPDATE SET
			   surfaced_count = surfaced_count + 1,
			   last_surfaced_at = excluded.last_surfaced_at`,
			id, stamp)
		if err != nil {
			return fmt.Errorf("record surfaced for finding %q: %w", id, err)
		}
	}
	return nil
}

func (s *sqliteRepository) RecordUsed(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalidFinding{Field: "id", Reason: "required"}
	}
	// Explicit "used" feedback validates the target exists (it is operator/UI
	// driven, not a high-volume async path), so a bogus id is reported rather
	// than silently creating an orphan counter.
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM findings WHERE id = ?`, id).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFindingNotFound{ID: id}
		}
		return fmt.Errorf("record used: lookup finding %q: %w", id, err)
	}
	stamp := s.now().Format(findingTimeFormat)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO finding_usage (finding_id, surfaced_count, used_count, last_surfaced_at)
		 VALUES (?, 0, 1, ?)
		 ON CONFLICT(finding_id) DO UPDATE SET
		   used_count = used_count + 1`,
		id, stamp)
	if err != nil {
		return fmt.Errorf("record used for finding %q: %w", id, err)
	}
	return nil
}

func (s *sqliteRepository) GetUsage(ctx context.Context, ids []string) (map[string]Usage, error) {
	out := make(map[string]Usage, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT finding_id, surfaced_count, used_count, last_surfaced_at FROM finding_usage WHERE finding_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("get usage: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var u Usage
		var lastRaw string
		if err := rows.Scan(&u.FindingID, &u.SurfacedCount, &u.UsedCount, &lastRaw); err != nil {
			return nil, fmt.Errorf("scan usage: %w", err)
		}
		u.LastSurfacedAt, _ = time.Parse(findingTimeFormat, lastRaw)
		out[u.FindingID] = u
	}
	return out, rows.Err()
}

func (s *sqliteRepository) ListDecayCandidates(ctx context.Context, minAge time.Duration, limit int) ([]Finding, error) {
	if limit <= 0 {
		limit = 100
	}
	// A candidate is ACTIVE, never surfaced (no usage row OR surfaced_count = 0),
	// and older than minAge measured from retrieval_date (falling back to
	// created_at via COALESCE on the empty string). The cutoff is a lexically-
	// comparable RFC3339Nano stamp.
	cutoff := s.now().Add(-minAge).Format(findingTimeFormat)
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+withPrefix("f", selectFindingColumns)+`
		   FROM findings f
		   LEFT JOIN finding_usage u ON u.finding_id = f.id
		  WHERE f.status = ?
		    AND COALESCE(u.surfaced_count, 0) = 0
		    AND (CASE WHEN f.retrieval_date != '' THEN f.retrieval_date ELSE f.created_at END) < ?
		  ORDER BY (CASE WHEN f.retrieval_date != '' THEN f.retrieval_date ELSE f.created_at END) ASC
		  LIMIT ?`,
		StatusActive, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("list decay candidates: %w", err)
	}
	return s.hydrate(ctx, rows)
}

func (s *sqliteRepository) ListOrphanedFindings(ctx context.Context) ([]Finding, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+withPrefix("f", selectFindingColumns)+`
		   FROM findings f
		   LEFT JOIN briefs b ON b.id = f.brief_id
		  WHERE f.brief_id != ''
		    AND b.id IS NULL
		  ORDER BY f.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list orphaned findings: %w", err)
	}
	return s.hydrate(ctx, rows)
}

// withPrefix qualifies a comma-separated column list with a table alias so it
// can be used in a JOIN select without ambiguity.
func withPrefix(alias, columns string) string {
	parts := strings.Split(columns, ", ")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}

// hydrate scans a finding result set and batch-loads citations for the rows.
func (s *sqliteRepository) hydrate(ctx context.Context, rows *sql.Rows) ([]Finding, error) {
	found, err := scanFindingRows(rows)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return found, nil
	}
	ids := make([]string, 0, len(found))
	for _, f := range found {
		ids = append(ids, f.ID)
	}
	cites, err := s.loadCitations(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range found {
		found[i].Citations = cites[found[i].ID]
	}
	return found, nil
}

func (s *sqliteRepository) loadCitations(ctx context.Context, ids []string) (map[string][]Citation, error) {
	out := make(map[string][]Citation, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, finding_id, url, title, retrieved_at FROM finding_citations
		 WHERE finding_id IN (`+placeholders+`) ORDER BY retrieved_at ASC, id ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("load citations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			c          Citation
			findingID  string
			retrievedR string
		)
		if err := rows.Scan(&c.ID, &findingID, &c.URL, &c.Title, &retrievedR); err != nil {
			return nil, fmt.Errorf("scan citation: %w", err)
		}
		if t, err := time.Parse(findingTimeFormat, retrievedR); err == nil {
			c.RetrievedAt = t
		}
		out[findingID] = append(out[findingID], c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate citations: %w", err)
	}
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFinding(sc rowScanner) (Finding, error) {
	var (
		f          Finding
		retrieval  string
		createdRaw string
		updatedRaw string
	)
	if err := sc.Scan(&f.ID, &f.Claim, &f.BriefID, &f.Confidence, &f.Status,
		&retrieval, &f.Query, &f.SupersededBy, &f.DisputeNote, &f.Source,
		&createdRaw, &updatedRaw); err != nil {
		return Finding{}, err
	}
	f.RetrievalDate, _ = time.Parse(findingTimeFormat, retrieval)
	f.CreatedAt, _ = time.Parse(findingTimeFormat, createdRaw)
	f.UpdatedAt, _ = time.Parse(findingTimeFormat, updatedRaw)
	return f, nil
}

func scanFindingRows(rows *sql.Rows) ([]Finding, error) {
	defer rows.Close()
	var out []Finding
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, fmt.Errorf("scan finding: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate findings: %w", err)
	}
	return out, nil
}
