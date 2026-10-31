package staleness

import (
	"context"
	"errors"
	"time"

	golden "development-toolchain-validator/internal/golden"
	manifest "development-toolchain-validator/internal/manifest"
	skillcatalog "development-toolchain-validator/internal/skill_catalog"
)

// ManifestSourceFromService adapts a manifest.Service into the
// staleness ManifestSource seam. Lives in this package so the
// consumer-owned interface stays narrow.
type ManifestSourceFromService struct {
	Svc  manifest.Service
	Repo manifest.Repository // for GetStaleOverride (not on Service)
}

func (m ManifestSourceFromService) List(ctx context.Context) ([]manifest.Manifest, error) {
	return m.Svc.List(ctx)
}

func (m ManifestSourceFromService) GetStaleOverride(ctx context.Context, skillID, goldenSlug string) (time.Time, error) {
	return m.Repo.GetStaleOverride(ctx, skillID, goldenSlug)
}

var _ ManifestSource = ManifestSourceFromService{}

// GoldenSourceFromRepo adapts a golden.Repository into the staleness
// GoldenSource seam.
type GoldenSourceFromRepo struct {
	Repo golden.Repository
}

func (g GoldenSourceFromRepo) CurrentTemplateVersion(ctx context.Context, goldenSlug string) (string, error) {
	gld, err := g.Repo.Get(ctx, goldenSlug)
	if err != nil {
		var nf golden.ErrGoldenNotFound
		if errors.As(err, &nf) {
			return "", nil
		}
		return "", err
	}
	return gld.TemplateVersionPinned, nil
}

var _ GoldenSource = GoldenSourceFromRepo{}

// SkillSourceFromRepo adapts a skill_catalog.Repository into the
// staleness SkillSource seam.
type SkillSourceFromRepo struct {
	Repo skillcatalog.Repository
}

func (s SkillSourceFromRepo) CurrentSkillVersion(ctx context.Context, skillID string) (string, error) {
	sk, err := s.Repo.Get(ctx, skillID)
	if err != nil {
		var nf skillcatalog.ErrSkillNotFound
		if errors.As(err, &nf) {
			return "", nil
		}
		return "", err
	}
	return sk.Version, nil
}

var _ SkillSource = SkillSourceFromRepo{}
