package runannotations

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"agent-manager/internal/config"
	"agent-manager/internal/sqlcompat"
	"agent-manager/internal/toolfacts"
)

// PackDirEnv overrides the annotation pack directory. Tests and operators use
// it; production resolves the scenario's canonical .vrooli/annotations.
const PackDirEnv = "AGENT_MANAGER_ANNOTATIONS_DIR"

// DefaultPackPath resolves the loaded pack file from the environment getenv
// reads (the process's os.Getenv in production, a map in tests). It returns an
// error rather than silently selecting an empty vocabulary.
func DefaultPackPath(getenv func(string) string) (string, error) {
	if dir := strings.TrimSpace(getenv(PackDirEnv)); dir != "" {
		return filepath.Join(dir, "default.json"), nil
	}
	scenario, err := config.ScenarioSourcePath()
	if err != nil {
		return "", fmt.Errorf("resolve scenario root for annotations: %w", err)
	}
	return filepath.Join(scenario, ".vrooli", "annotations", "default.json"), nil
}

// DefaultOptions are what the production composition supplies from outside
// the engine.
type DefaultOptions struct {
	// PackPath is the pack file to load (see DefaultPackPath). Required.
	PackPath string
	// StateRoot is the scenario's durable state directory; the gold root
	// resolves against it.
	StateRoot string
	// Decisions answers decision labels. Nil uses ai-gateway lazily; an absent
	// gateway records decision labels unavailable, never zero.
	Decisions DecisionClient
	// Parsers resolves a harness's transcript parser so the tool facts of a run
	// that kept only its transcript are read. Nil leaves such runs unobserved.
	Parsers toolfacts.ParserResolver
}

// NewDefaultService composes the repository-backed production service: the
// pack at opts.PackPath (never an ambient lookup), idempotent schema, SQLite
// feature/signal/selection adapters and a discovery-backed decision client.
func NewDefaultService(ctx context.Context, db sqlcompat.DB, opts DefaultOptions) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("annotation service requires a database")
	}
	if strings.TrimSpace(opts.PackPath) == "" {
		return nil, fmt.Errorf("annotation service requires a pack path; resolve it with DefaultPackPath")
	}
	pack, err := LoadCompiled(opts.PackPath)
	if err != nil {
		return nil, err
	}
	store := NewStore(db)
	if err := store.EnsureSchema(ctx); err != nil {
		return nil, err
	}
	decisions := opts.Decisions
	if decisions == nil {
		decisions = NewGatewayDecisionClient()
	}
	return NewService(pack, Options{
		Store:          store,
		Features:       NewSQLiteFeatureSource(db, opts.Parsers),
		Signals:        NewSQLiteSignalReader(db),
		Selector:       NewSQLiteRunSelector(db),
		Decisions:      decisions,
		Limits:         NewGatewayLimitsSource(),
		MaxDigestChars: pack.Pack.Runtime.DigestChars,
		StateRoot:      opts.StateRoot,
	})
}
