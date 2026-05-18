package golden

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"

	"development-toolchain-validator/internal/clock"
)

// RegeneratorRunner is the seam Service.Regenerate uses to invoke
// `vrooli scenario generate ... --force` (or any equivalent runner).
// Tests substitute a fake that records the call and returns either the
// expected new template version or an error. Keeping this as a seam
// means the service never reaches into os/exec directly and stays
// testable without spawning subprocesses.
type RegeneratorRunner interface {
	// Regenerate (re)materializes the golden at path on disk from
	// template templateID. Returns the template version produced by the
	// generator (often the same as the requested version, but the
	// generator is authoritative).
	Regenerate(ctx context.Context, in RegenerateRunnerInput) (RegenerateRunnerOutput, error)
}

// RegenerateRunnerInput is the explicit input DTO RegeneratorRunner
// implementations receive. Keeping it explicit means future fields
// (e.g. design kit override) land without changing the interface.
type RegenerateRunnerInput struct {
	Slug            string
	TemplateID      string
	TemplateVersion string
	Path            string
}

// RegenerateRunnerOutput is the explicit output DTO. TemplateVersion is
// the version actually produced (the generator owns version selection).
type RegenerateRunnerOutput struct {
	TemplateVersion string
}

// Service is the application-layer surface the golden handlers depend
// on. Owns input validation, slug uniqueness enforcement, regenerate
// orchestration, and any cross-handler policy.
type Service interface {
	List(ctx context.Context) ([]Golden, error)
	Get(ctx context.Context, slug string) (Golden, error)
	Register(ctx context.Context, in RegisterInput) (Golden, error)
	Update(ctx context.Context, in UpdateInput) (Golden, error)
	Delete(ctx context.Context, slug string) error
	Regenerate(ctx context.Context, slug string) (Golden, error)
}

type service struct {
	repo  Repository
	clock clock.Clock
	regen RegeneratorRunner
}

// NewService constructs the production Service.
func NewService(repo Repository, clk clock.Clock, regen RegeneratorRunner) Service {
	return &service{repo: repo, clock: clk, regen: regen}
}

var _ Service = (*service)(nil)

// slugPattern enforces a conservative kebab-case shape for goldens.
// Mirrors the slug shape vrooli scenario IDs use elsewhere; rejecting
// freeform strings at the boundary lets the rest of the system treat
// slugs as filesystem-safe.
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

func (s *service) List(ctx context.Context) ([]Golden, error) {
	return s.repo.List(ctx)
}

func (s *service) Get(ctx context.Context, slug string) (Golden, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return Golden{}, ErrInvalidGolden{Field: "slug", Reason: "required"}
	}
	return s.repo.Get(ctx, slug)
}

func (s *service) Register(ctx context.Context, in RegisterInput) (Golden, error) {
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		return Golden{}, ErrInvalidGolden{Field: "slug", Reason: "required"}
	}
	if !slugPattern.MatchString(slug) {
		return Golden{}, ErrInvalidGolden{Field: "slug", Reason: "must match [a-z0-9][a-z0-9-]{0,63}"}
	}
	if strings.TrimSpace(in.TemplateID) == "" {
		return Golden{}, ErrInvalidGolden{Field: "template_id", Reason: "required"}
	}
	if strings.TrimSpace(in.TemplateVersion) == "" {
		return Golden{}, ErrInvalidGolden{Field: "template_version", Reason: "required"}
	}
	if strings.TrimSpace(in.Path) == "" {
		return Golden{}, ErrInvalidGolden{Field: "path", Reason: "required"}
	}
	// Disallow absolute paths so callers can't accidentally register a
	// path outside the repository (which a later regenerate run could
	// overwrite). Repo-relative paths only.
	if filepath.IsAbs(in.Path) {
		return Golden{}, ErrInvalidGolden{Field: "path", Reason: "must be repository-relative"}
	}

	return s.repo.Create(ctx, Golden{
		Slug:                  slug,
		TemplateID:            strings.TrimSpace(in.TemplateID),
		TemplateVersionPinned: strings.TrimSpace(in.TemplateVersion),
		Path:                  strings.TrimSpace(in.Path),
	})
}

func (s *service) Update(ctx context.Context, in UpdateInput) (Golden, error) {
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		return Golden{}, ErrInvalidGolden{Field: "slug", Reason: "required"}
	}
	current, err := s.repo.Get(ctx, slug)
	if err != nil {
		return Golden{}, err
	}
	changed := false
	if p := strings.TrimSpace(in.Path); p != "" && p != current.Path {
		if filepath.IsAbs(p) {
			return Golden{}, ErrInvalidGolden{Field: "path", Reason: "must be repository-relative"}
		}
		current.Path = p
		changed = true
	}
	if v := strings.TrimSpace(in.TemplateVersion); v != "" && v != current.TemplateVersionPinned {
		current.TemplateVersionPinned = v
		changed = true
	}
	if changed {
		current.LastRegeneratedAt = s.clock.Now().UTC()
	}
	return s.repo.Update(ctx, current)
}

func (s *service) Delete(ctx context.Context, slug string) error {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ErrInvalidGolden{Field: "slug", Reason: "required"}
	}
	return s.repo.Delete(ctx, slug)
}

func (s *service) Regenerate(ctx context.Context, slug string) (Golden, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return Golden{}, ErrInvalidGolden{Field: "slug", Reason: "required"}
	}
	current, err := s.repo.Get(ctx, slug)
	if err != nil {
		return Golden{}, err
	}
	if s.regen == nil {
		return Golden{}, ErrRegenerateFailed{Slug: slug, Wrapped: errNoRegenerator}
	}
	out, err := s.regen.Regenerate(ctx, RegenerateRunnerInput{
		Slug:            current.Slug,
		TemplateID:      current.TemplateID,
		TemplateVersion: current.TemplateVersionPinned,
		Path:            current.Path,
	})
	if err != nil {
		return Golden{}, ErrRegenerateFailed{Slug: slug, Wrapped: err}
	}
	if v := strings.TrimSpace(out.TemplateVersion); v != "" {
		current.TemplateVersionPinned = v
	}
	current.LastRegeneratedAt = s.clock.Now().UTC()
	return s.repo.Update(ctx, current)
}
