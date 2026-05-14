package themes

import (
	"context"
	"fmt"
	"strings"
)

// DesignMDReader is the target-scenario-tree seam ResolveFromScenario
// uses to read a scenario's DESIGN.md. Production walks the configured
// scenarios root with a traversal guard.
type DesignMDReader interface {
	Read(ctx context.Context, scenario string) ([]byte, error)
}

// Service is the application-layer surface handlers depend on.
type Service interface {
	ListBuiltins(ctx context.Context) ([]Theme, error)
	GetBuiltin(ctx context.Context, id string) (Theme, error)
	ResolveFromScenario(ctx context.Context, scenario string) (Theme, error)
	// EnsureBuiltinsSeeded inserts the canonical built-in themes when
	// the table is empty. Idempotent — safe to call on every boot.
	EnsureBuiltinsSeeded(ctx context.Context) error
}

type service struct {
	repo    Repository
	designs DesignMDReader
}

func NewService(repo Repository, designs DesignMDReader) Service {
	return &service{repo: repo, designs: designs}
}

var _ Service = (*service)(nil)

func (s *service) ListBuiltins(ctx context.Context) ([]Theme, error) {
	return s.repo.ListBuiltins(ctx)
}

func (s *service) GetBuiltin(ctx context.Context, id string) (Theme, error) {
	return s.repo.GetBuiltin(ctx, id)
}

func (s *service) ResolveFromScenario(ctx context.Context, scenario string) (Theme, error) {
	scn := strings.TrimSpace(scenario)
	if scn == "" {
		return Theme{}, fmt.Errorf("scenario required")
	}
	if s.designs == nil {
		return Theme{}, fmt.Errorf("design.md reader not configured")
	}
	raw, err := s.designs.Read(ctx, scn)
	if err != nil {
		return Theme{}, ErrScenarioDesignMDMissing{Scenario: scn, Cause: err}
	}
	return ParseDesignMDToTheme(raw, scn)
}

func (s *service) EnsureBuiltinsSeeded(ctx context.Context) error {
	n, err := s.repo.CountBuiltins(ctx)
	if err != nil {
		return fmt.Errorf("count builtins: %w", err)
	}
	if n > 0 {
		return nil
	}
	for _, t := range builtinThemes() {
		if err := s.repo.UpsertBuiltin(ctx, t); err != nil {
			return fmt.Errorf("seed %s: %w", t.ID, err)
		}
	}
	return nil
}

// builtinThemes is the canonical seed set. Keep token keys aligned
// with the DESIGN.md projection so the harness applies them uniformly
// regardless of source.
func builtinThemes() []Theme {
	return []Theme{
		{
			ID: "vrooli-default", Name: "Vrooli Default", Source: "builtin",
			Tokens: map[string]string{
				"--color-primary":     "#2563eb",
				"--color-secondary":   "#0891b2",
				"--color-neutral":     "#f8fafc",
				"--color-surface":     "#ffffff",
				"--color-on-surface":  "#0f172a",
				"--color-error":       "#dc2626",
				"--color-success":     "#16a34a",
				"--color-warning":     "#d97706",
				"--rounded-sm":        "0.375rem",
				"--rounded-md":        "0.5rem",
				"--rounded-lg":        "1rem",
				"--rounded-full":      "9999px",
				"--spacing-unit":      "0.25rem",
				"--spacing-touch":     "44px",
			},
		},
		{
			ID: "neutral-light", Name: "Neutral Light", Source: "builtin",
			Tokens: map[string]string{
				"--color-primary":    "#475569",
				"--color-secondary":  "#64748b",
				"--color-neutral":    "#ffffff",
				"--color-surface":    "#ffffff",
				"--color-on-surface": "#0f172a",
				"--color-error":      "#b91c1c",
				"--color-success":    "#15803d",
				"--color-warning":    "#a16207",
				"--rounded-sm":       "0.25rem",
				"--rounded-md":       "0.375rem",
				"--rounded-lg":       "0.75rem",
				"--rounded-full":     "9999px",
				"--spacing-unit":     "0.25rem",
			},
		},
		{
			ID: "neutral-dark", Name: "Neutral Dark", Source: "builtin",
			Tokens: map[string]string{
				"--color-primary":    "#94a3b8",
				"--color-secondary":  "#cbd5e1",
				"--color-neutral":    "#0f172a",
				"--color-surface":    "#1e293b",
				"--color-on-surface": "#f1f5f9",
				"--color-error":      "#f87171",
				"--color-success":    "#4ade80",
				"--color-warning":    "#fbbf24",
				"--rounded-sm":       "0.25rem",
				"--rounded-md":       "0.375rem",
				"--rounded-lg":       "0.75rem",
				"--rounded-full":     "9999px",
				"--spacing-unit":     "0.25rem",
			},
		},
		{
			ID: "high-contrast", Name: "High Contrast", Source: "builtin",
			Tokens: map[string]string{
				"--color-primary":    "#000000",
				"--color-secondary":  "#000000",
				"--color-neutral":    "#ffffff",
				"--color-surface":    "#ffffff",
				"--color-on-surface": "#000000",
				"--color-error":      "#cc0000",
				"--color-success":    "#006600",
				"--color-warning":    "#cc6600",
				"--rounded-sm":       "0",
				"--rounded-md":       "0",
				"--rounded-lg":       "0",
				"--rounded-full":     "0",
				"--spacing-unit":     "0.25rem",
			},
		},
	}
}
