// Package mocks provides in-memory fakes for staleness's three seams.
package mocks

import (
	"context"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
	staleness "development-toolchain-validator/internal/staleness"
)

// FakeManifestSource is an in-memory ManifestSource for tests.
type FakeManifestSource struct {
	Manifests []manifest.Manifest
	Overrides map[[2]string]time.Time
	Err       error
}

var _ staleness.ManifestSource = (*FakeManifestSource)(nil)

func (f *FakeManifestSource) List(_ context.Context) ([]manifest.Manifest, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	out := make([]manifest.Manifest, len(f.Manifests))
	copy(out, f.Manifests)
	return out, nil
}

func (f *FakeManifestSource) GetStaleOverride(_ context.Context, skillID, goldenSlug string) (time.Time, error) {
	if f.Overrides == nil {
		return time.Time{}, nil
	}
	return f.Overrides[[2]string{skillID, goldenSlug}], nil
}

// FakeGoldenSource is an in-memory GoldenSource for tests.
type FakeGoldenSource struct {
	Versions map[string]string
}

var _ staleness.GoldenSource = (*FakeGoldenSource)(nil)

func (f *FakeGoldenSource) CurrentTemplateVersion(_ context.Context, goldenSlug string) (string, error) {
	return f.Versions[goldenSlug], nil
}

// FakeSkillSource is an in-memory SkillSource for tests.
type FakeSkillSource struct {
	Versions map[string]string
}

var _ staleness.SkillSource = (*FakeSkillSource)(nil)

func (f *FakeSkillSource) CurrentSkillVersion(_ context.Context, skillID string) (string, error) {
	return f.Versions[skillID], nil
}
