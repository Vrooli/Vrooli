// Package mocks provides in-memory fakes for the skill_catalog seams.
// Production code never imports this package — the no_prod_import_test
// drift gate enforces that.
package mocks

import (
	"context"

	skillcatalog "development-toolchain-validator/internal/skill_catalog"
)

// FakeSource is a programmable fake for SkillCatalogSource. Tests
// populate Skills and (optionally) Err.
type FakeSource struct {
	Skills []skillcatalog.Skill
	Err    error
	Calls  int
}

var _ skillcatalog.SkillCatalogSource = (*FakeSource)(nil)

// Fetch returns the configured skills (or error).
func (f *FakeSource) Fetch(_ context.Context) ([]skillcatalog.Skill, error) {
	f.Calls++
	if f.Err != nil {
		return nil, f.Err
	}
	out := make([]skillcatalog.Skill, len(f.Skills))
	copy(out, f.Skills)
	return out, nil
}
