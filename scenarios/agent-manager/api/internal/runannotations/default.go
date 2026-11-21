package runannotations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent-manager/internal/config"
	"agent-manager/internal/sqlcompat"
)

// PackDirEnv overrides the annotation pack directory. Tests and operators use
// it; production resolves the scenario's canonical .vrooli/annotations.
const PackDirEnv = "AGENT_MANAGER_ANNOTATIONS_DIR"

// DefaultPackPath resolves the loaded pack file. It returns an error rather
// than silently selecting an empty vocabulary.
func DefaultPackPath() (string, error) {
	if dir := strings.TrimSpace(os.Getenv(PackDirEnv)); dir != "" {
		return filepath.Join(dir, "default.json"), nil
	}
	scenario, err := config.ScenarioSourcePath()
	if err != nil {
		return "", fmt.Errorf("resolve scenario root for annotations: %w", err)
	}
	return filepath.Join(scenario, ".vrooli", "annotations", "default.json"), nil
}

// NewDefaultService composes the repository-backed production service: the
// canonical pack, idempotent schema, SQLite feature/signal/selection adapters,
// and a discovery-backed decision client. Passing nil decisions uses ai-gateway
// lazily; an absent gateway records decision labels unavailable, never zero.
func NewDefaultService(ctx context.Context, db sqlcompat.DB, packPath string, decisions DecisionClient) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("annotation service requires a database")
	}
	if strings.TrimSpace(packPath) == "" {
		resolved, err := DefaultPackPath()
		if err != nil {
			return nil, err
		}
		packPath = resolved
	}
	pack, err := LoadCompiled(packPath)
	if err != nil {
		return nil, err
	}
	store := NewStore(db)
	if err := store.EnsureSchema(ctx); err != nil {
		return nil, err
	}
	if decisions == nil {
		decisions = NewGatewayDecisionClient()
	}
	return NewService(pack, Options{
		Store:          store,
		Features:       NewSQLiteFeatureSource(db),
		Signals:        NewSQLiteSignalReader(db),
		Selector:       NewSQLiteRunSelector(db),
		Decisions:      decisions,
		MaxDigestChars: pack.Pack.Runtime.DigestChars,
	})
}
