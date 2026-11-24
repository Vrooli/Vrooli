package search

import (
	"context"
	"database/sql"
	"log"
	"os"
	"sync"
	"time"
)

type AvailabilityFunc func(ctx context.Context, catalogID string) (built bool, label string, err error)

// Semantic marks the optional retrieval leg. Production implements the shared
// ai-go/search SemanticSearcher and readiness seams; tests can still inject a
// minimal status fake without starting external model/vector resources.
type Semantic interface {
	Enabled() bool
}

// Service is the lexical retrieval service over the search projection.
type Service struct {
	db                  sqlExecutor
	scenarioRoot        string
	availability        AvailabilityFunc
	catalogAvailability CatalogAvailabilityFactory
	semantic            Semantic
	answerer            Answerer
	logger              interface{ Printf(string, ...any) }

	// reindexMu serializes rebuilds. Search reads keep seeing the previous
	// projection until the rebuild transaction commits.
	reindexMu sync.Mutex

	// Semantic reconciliation can be slow because it embeds the committed
	// projection. Keep one worker per service and coalesce changes that arrive
	// while a pass is running into at most one follow-up pass.
	semanticReconcileMu      sync.Mutex
	semanticReconcileRunning bool
	semanticReconcileDirty   bool
	semanticFailureMu        sync.RWMutex
	semanticFailure          string
}

// sqlExecutor is the request-aware database surface used by the lexical
// projection. Both *sql.DB and api-core's RoutedDB satisfy it.
type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// WithAnswerer attaches the explicit Ask path. A nil answerer keeps Ask
// unavailable while ordinary search remains usable.
func (s *Service) WithAnswerer(answerer Answerer) *Service {
	if s != nil {
		s.answerer = answerer
	}
	return s
}

// NewService builds the service for a scenario. scenarioRoot is the scenario
// directory (the parent of catalog/), not the repository root; callers that
// have only the repo root should pass `<repoRoot>/scenarios/react-component-library`.
func NewService(db sqlExecutor, scenarioRoot string, availability AvailabilityFunc) *Service {
	return &Service{db: db, scenarioRoot: scenarioRoot, availability: availability, logger: log.New(os.Stderr, "", log.LstdFlags)}
}

// WithLogger sets the diagnostic sink used for semantic reconciliation and
// query-time semantic degradation. A nil logger disables diagnostics without
// changing search availability.
func (s *Service) WithLogger(logger interface{ Printf(string, ...any) }) *Service {
	if s != nil {
		s.logger = logger
	}
	return s
}

// Reindex is the boot/setup helper for callers that do not otherwise need a
// Service value. It uses the same implementation as (*Service).Reindex so main
// wiring cannot drift from the RPC.
func Reindex(ctx context.Context, db sqlExecutor, scenarioRoot string, availability AvailabilityFunc, force bool) (int, time.Time, error) {
	return NewService(db, scenarioRoot, availability).Reindex(ctx, force)
}

// WithSemantic attaches the optional semantic leg. A nil leg means DISABLED.
func (s *Service) WithSemantic(semantic Semantic) *Service {
	if s != nil {
		s.semantic = semantic
	}
	return s
}

// DB exposes the handle for wiring tests; production callers already own it.
func (s *Service) DB() *sql.DB {
	if s == nil {
		return nil
	}
	db, _ := s.db.(*sql.DB)
	return db
}
