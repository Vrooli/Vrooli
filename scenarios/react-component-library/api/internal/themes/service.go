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
	if err := s.repo.ReplaceBuiltins(ctx, builtinThemes()); err != nil {
		return fmt.Errorf("migrate builtin themes: %w", err)
	}
	return nil
}

// builtinThemes is the canonical seed set. Keep token keys aligned
// with the DESIGN.md projection so the harness applies them uniformly
// regardless of source.
func builtinThemes() []Theme {
	return []Theme{
		{
			ID: "light", Name: "Light", Source: "builtin",
			Tokens: map[string]string{
				"--color-background": "#f8fafc", "--color-surface": "#ffffff", "--color-surface-muted": "#f1f5f9", "--color-surface-raised": "#ffffff", "--color-foreground": "#0f172a", "--color-muted-foreground": "#475569", "--color-border": "#cbd5e1", "--color-primary": "#2563eb", "--color-primary-foreground": "#ffffff", "--color-accent": "#0891b2", "--color-success": "#16a34a", "--color-danger": "#dc2626", "--color-warning": "#d97706", "--color-info": "#0284c7", "--color-focus": "#2563eb", "--radius-control": "0.375rem", "--radius-panel": "0.5rem", "--radius-sheet": "0.75rem", "--radius-pill": "9999px",
			},
		},
		{
			ID: "dark", Name: "Dark", Source: "builtin",
			Tokens: map[string]string{
				"--color-background": "#020617", "--color-surface": "#0f172a", "--color-surface-muted": "#1e293b", "--color-surface-raised": "#1e293b", "--color-foreground": "#f8fafc", "--color-muted-foreground": "#cbd5e1", "--color-border": "#334155", "--color-primary": "#60a5fa", "--color-primary-foreground": "#0f172a", "--color-accent": "#67e8f9", "--color-success": "#4ade80", "--color-danger": "#f87171", "--color-warning": "#fbbf24", "--color-info": "#7dd3fc", "--color-focus": "#93c5fd", "--radius-control": "0.375rem", "--radius-panel": "0.5rem", "--radius-sheet": "0.75rem", "--radius-pill": "9999px",
			},
		},
		{
			ID: "high-contrast", Name: "High Contrast", Source: "builtin",
			Tokens: map[string]string{
				"--color-background": "#ffffff", "--color-surface": "#ffffff", "--color-surface-muted": "#f5f5f5", "--color-surface-raised": "#ffffff", "--color-foreground": "#000000", "--color-muted-foreground": "#1f2937", "--color-border": "#000000", "--color-primary": "#0000ee", "--color-primary-foreground": "#ffffff", "--color-accent": "#000000", "--color-success": "#006600", "--color-danger": "#b00020", "--color-warning": "#7c2d12", "--color-info": "#003f8c", "--color-focus": "#0000ee", "--radius-control": "0", "--radius-panel": "0", "--radius-sheet": "0", "--radius-pill": "0",
			},
		},
	}
}
