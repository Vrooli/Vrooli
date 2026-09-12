package brands

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
)

const (
	// defaultListLimit caps the rows returned by Service.List when the caller
	// passes 0. Business policy (matches the `brands list` manifest default).
	defaultListLimit = 100
	// maxListLimit is the hard ceiling List clamps to (manifest "max 500").
	maxListLimit = 500
)

// Service is the application-layer surface the brands handlers depend on. Owns
// validation, default substitution, optimistic-lock enforcement, and the
// immutable version-snapshot lifecycle. The handler is intentionally thin
// around it: decode → call service → translate errors.
type Service interface {
	// Create validates in (Name required after trim), persists the brand at
	// version 1, and best-effort snapshots that version. Returns ErrInvalidBrand
	// on validation failure.
	Create(ctx context.Context, in CreateInput) (Brand, error)

	// Get is a thin pass-through to Repository.Get.
	Get(ctx context.Context, id string) (Brand, error)

	// List substitutes defaultListLimit when limit <= 0 and clamps to
	// maxListLimit, then passes the filter through.
	List(ctx context.Context, filter ListFilter) ([]Brand, error)

	// Update loads the brand, enforces the optimistic lock (when
	// in.ExpectedVersion > 0), merges the partial input, persists the new
	// version, and best-effort snapshots it. Returns ErrBrandNotFound,
	// ErrVersionConflict, or ErrInvalidBrand as appropriate.
	Update(ctx context.Context, in UpdateInput) (Brand, error)

	// Delete removes the brand. Idempotent: deleting a missing brand returns
	// nil (the not-found is swallowed at this layer).
	Delete(ctx context.Context, id string) error

	// ListVersions returns the brand's immutable version history, newest-first.
	ListVersions(ctx context.Context, brandID string) ([]BrandVersion, error)
}

type service struct {
	repo     Repository
	versions VersionRepository
	logger   *log.Logger
}

// NewService constructs the production Service. logger records best-effort
// version-snapshot failures (which never fail the mutation); a nil logger
// defaults to log.Default().
func NewService(repo Repository, versions VersionRepository, logger *log.Logger) Service {
	if logger == nil {
		logger = log.Default()
	}
	return &service{repo: repo, versions: versions, logger: logger}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Create(ctx context.Context, in CreateInput) (Brand, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Brand{}, ErrInvalidBrand{Field: "name", Reason: "required"}
	}
	created, err := s.repo.Create(ctx, Brand{
		Name:        name,
		Description: in.Description,
		Notes:       in.Notes,
		Identity:    in.Identity,
		Colors:      in.Colors,
		Typography:  in.Typography,
		Voice:       in.Voice,
	})
	if err != nil {
		return Brand{}, err
	}
	s.snapshot(ctx, created)
	return created, nil
}

func (s *service) Get(ctx context.Context, id string) (Brand, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) List(ctx context.Context, filter ListFilter) ([]Brand, error) {
	if filter.Limit <= 0 {
		filter.Limit = defaultListLimit
	}
	if filter.Limit > maxListLimit {
		filter.Limit = maxListLimit
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}

func (s *service) Update(ctx context.Context, in UpdateInput) (Brand, error) {
	current, err := s.repo.Get(ctx, in.ID)
	if err != nil {
		return Brand{}, err
	}
	if in.ExpectedVersion > 0 && in.ExpectedVersion != current.Version {
		return Brand{}, ErrVersionConflict{ID: in.ID, Expected: in.ExpectedVersion, Actual: current.Version}
	}

	merged := mergeBrand(current, in)
	if strings.TrimSpace(merged.Name) == "" {
		return Brand{}, ErrInvalidBrand{Field: "name", Reason: "required"}
	}

	updated, err := s.repo.Update(ctx, merged)
	if err != nil {
		return Brand{}, err
	}
	s.snapshot(ctx, updated)
	return updated, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if err == nil {
		return nil
	}
	// Idempotent: deleting a brand that does not exist is a success.
	var notFound ErrBrandNotFound
	if errors.As(err, &notFound) {
		return nil
	}
	return err
}

func (s *service) ListVersions(ctx context.Context, brandID string) ([]BrandVersion, error) {
	return s.versions.ListVersions(ctx, brandID)
}

// snapshot records the post-mutation brand state as an immutable version.
// Best-effort by design: a snapshot failure is logged but never fails the
// mutation the caller already committed (the authoring history degrades, the
// write does not).
func (s *service) snapshot(ctx context.Context, b Brand) {
	payload, err := json.Marshal(b)
	if err != nil {
		s.logger.Printf("brands: snapshot marshal brand %q v%d: %v", b.ID, b.Version, err)
		return
	}
	if _, err := s.versions.CreateVersion(ctx, BrandVersion{
		BrandID:  b.ID,
		Version:  b.Version,
		Snapshot: string(payload),
	}); err != nil {
		s.logger.Printf("brands: snapshot brand %q v%d: %v", b.ID, b.Version, err)
	}
}

// mergeBrand applies the partial UpdateInput onto current with field-level
// non-empty-overwrite semantics: an empty scalar or facet sub-field leaves the
// stored value unchanged. Mirrors the old domain ApplyPartialUpdate but at
// field granularity so updating one facet field does not wipe its siblings.
func mergeBrand(current Brand, in UpdateInput) Brand {
	out := current
	out.Name = orStr(in.Name, out.Name)
	out.Description = orStr(in.Description, out.Description)
	out.Notes = orStr(in.Notes, out.Notes)

	out.Identity.DisplayName = orStr(in.Identity.DisplayName, out.Identity.DisplayName)
	out.Identity.Tagline = orStr(in.Identity.Tagline, out.Identity.Tagline)
	out.Identity.LogoPath = orStr(in.Identity.LogoPath, out.Identity.LogoPath)
	out.Identity.FaviconPath = orStr(in.Identity.FaviconPath, out.Identity.FaviconPath)
	out.Identity.IconPath = orStr(in.Identity.IconPath, out.Identity.IconPath)

	out.Colors.Primary = orStr(in.Colors.Primary, out.Colors.Primary)
	out.Colors.Secondary = orStr(in.Colors.Secondary, out.Colors.Secondary)
	out.Colors.Accent = orStr(in.Colors.Accent, out.Colors.Accent)
	out.Colors.Background = orStr(in.Colors.Background, out.Colors.Background)
	out.Colors.Surface = orStr(in.Colors.Surface, out.Colors.Surface)
	out.Colors.Text = orStr(in.Colors.Text, out.Colors.Text)
	out.Colors.Error = orStr(in.Colors.Error, out.Colors.Error)

	out.Typography.HeadingFont = orStr(in.Typography.HeadingFont, out.Typography.HeadingFont)
	out.Typography.BodyFont = orStr(in.Typography.BodyFont, out.Typography.BodyFont)
	out.Typography.MonoFont = orStr(in.Typography.MonoFont, out.Typography.MonoFont)
	out.Typography.BaseFontSize = orStr(in.Typography.BaseFontSize, out.Typography.BaseFontSize)

	out.Voice.Tone = orStr(in.Voice.Tone, out.Voice.Tone)
	out.Voice.Style = orStr(in.Voice.Style, out.Voice.Style)
	if len(in.Voice.Keywords) > 0 {
		out.Voice.Keywords = append([]string(nil), in.Voice.Keywords...)
	}
	return out
}

// orStr returns next when it is non-empty after trimming, else prev.
func orStr(next, prev string) string {
	if strings.TrimSpace(next) != "" {
		return next
	}
	return prev
}
