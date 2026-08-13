package probes

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (repository
// unit tests via testutil/db.NewSQLite) and *database.RoutedDB (production
// main.go) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

const (
	// RFC3339Nano matches the wire format and the round-trip in scanProbe.
	probeTimeFormat = time.RFC3339Nano

	// defaultListLimit caps List when the caller passes a non-positive limit.
	defaultListLimit = 100

	probeColumns = `id, subdomain, kind, status, latency_ms, status_code, error_msg, created_at`

	insertProbeSQL = `
INSERT INTO probes (id, subdomain, kind, status, latency_ms, status_code, error_msg, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
)

func (s *sqliteRepository) Persist(ctx context.Context, r ProbeResult) (ProbeResult, error) {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = s.clock.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx, insertProbeSQL,
		r.ID, r.Subdomain, string(r.Kind), string(r.Status),
		r.LatencyMS, r.StatusCode, r.ErrorMsg,
		r.CreatedAt.Format(probeTimeFormat),
	)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("insert probe %q: %w", r.ID, err)
	}
	if err := s.pruneBefore(ctx, r.CreatedAt.Add(-HistoryRetentionWindow)); err != nil {
		return ProbeResult{}, err
	}
	return r, nil
}

func (s *sqliteRepository) pruneBefore(ctx context.Context, cutoff time.Time) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM probes WHERE created_at < ?", cutoff.UTC().Format(probeTimeFormat))
	if err != nil {
		return fmt.Errorf("prune probes before %s: %w", cutoff.UTC().Format(probeTimeFormat), err)
	}
	return nil
}

func (s *sqliteRepository) List(ctx context.Context, subdomain string, limit int) ([]ProbeResult, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	query := "SELECT " + probeColumns + " FROM probes"
	var args []any
	if subdomain != "" {
		query += " WHERE subdomain = ?"
		args = append(args, subdomain)
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list probes: %w", err)
	}
	defer rows.Close()

	var results []ProbeResult
	for rows.Next() {
		r, err := scanProbe(rows)
		if err != nil {
			return nil, fmt.Errorf("scan probe: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate probes: %w", err)
	}
	return results, nil
}

// latestPerRouteKindSQL selects the single newest probe for each
// (subdomain, kind) pair. The correlated subquery keys on created_at then
// id (the tie-break that matches List's ordering) so the result is
// deterministic even when two probes share a timestamp.
const latestPerRouteKindSQL = `
SELECT ` + probeColumns + ` FROM probes p
WHERE NOT EXISTS (
  SELECT 1 FROM probes q
  WHERE q.subdomain = p.subdomain AND q.kind = p.kind
    AND (q.created_at > p.created_at OR (q.created_at = p.created_at AND q.id > p.id))
)
ORDER BY subdomain ASC, kind ASC
`

func (s *sqliteRepository) LatestPerRoute(ctx context.Context) ([]LatestPair, error) {
	rows, err := s.db.QueryContext(ctx, latestPerRouteKindSQL)
	if err != nil {
		return nil, fmt.Errorf("latest probes per route: %w", err)
	}
	defer rows.Close()

	// Preserve first-seen subdomain order (rows are subdomain-sorted) so
	// Classify output is stable.
	bySub := make(map[string]*LatestPair)
	var order []string
	for rows.Next() {
		r, err := scanProbe(rows)
		if err != nil {
			return nil, fmt.Errorf("scan probe: %w", err)
		}
		pair, ok := bySub[r.Subdomain]
		if !ok {
			pair = &LatestPair{Subdomain: r.Subdomain}
			bySub[r.Subdomain] = pair
			order = append(order, r.Subdomain)
		}
		probe := r
		switch r.Kind {
		case ProbeKindInternal:
			pair.Internal = &probe
		case ProbeKindExternal:
			pair.External = &probe
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest probes: %w", err)
	}

	pairs := make([]LatestPair, 0, len(order))
	for _, sub := range order {
		pairs = append(pairs, *bySub[sub])
	}
	return pairs, nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan
// surface so scanProbe works for both single-row and multi-row reads.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanProbe(sc rowScanner) (ProbeResult, error) {
	var (
		r          ProbeResult
		kindRaw    string
		statusRaw  string
		createdRaw string
	)
	if err := sc.Scan(&r.ID, &r.Subdomain, &kindRaw, &statusRaw,
		&r.LatencyMS, &r.StatusCode, &r.ErrorMsg, &createdRaw); err != nil {
		return ProbeResult{}, err
	}
	r.Kind = ProbeKind(kindRaw)
	r.Status = ProbeStatus(statusRaw)
	created, err := time.Parse(probeTimeFormat, createdRaw)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	r.CreatedAt = created
	return r, nil
}
