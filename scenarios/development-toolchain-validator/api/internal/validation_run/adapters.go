package validation_run

import (
	"context"
	"errors"

	golden "development-toolchain-validator/internal/golden"
	manifest "development-toolchain-validator/internal/manifest"
)

// GoldenSourceFromRepo adapts a golden.Repository into GoldenSource.
type GoldenSourceFromRepo struct {
	Repo golden.Repository
}

func (g GoldenSourceFromRepo) GoldenPath(ctx context.Context, goldenSlug string) (string, error) {
	gld, err := g.Repo.Get(ctx, goldenSlug)
	if err != nil {
		var nf golden.ErrGoldenNotFound
		if errors.As(err, &nf) {
			return "", ErrInvalidRun{Field: "golden_slug", Reason: "no such golden"}
		}
		return "", err
	}
	return gld.Path, nil
}

var _ GoldenSource = GoldenSourceFromRepo{}

// ManifestSourceFromRepo adapts a manifest.Repository into ManifestSource.
type ManifestSourceFromRepo struct {
	Repo manifest.Repository
}

func (m ManifestSourceFromRepo) GetManifest(ctx context.Context, skillID, goldenSlug string) (manifest.Manifest, error) {
	return m.Repo.Get(ctx, skillID, goldenSlug)
}

var _ ManifestSource = ManifestSourceFromRepo{}
