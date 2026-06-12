package autosteer

import (
	"reflect"
	"testing"

	"github.com/ecosystem-manager/api/pkg/skillmap"
)

type coverageProfileRepo struct {
	profile *AutoSteerProfile
}

func (r coverageProfileRepo) CreateProfile(profile *AutoSteerProfile) error { return nil }
func (r coverageProfileRepo) ListProfiles(tags []string) ([]*AutoSteerProfile, error) {
	return nil, nil
}

func (r coverageProfileRepo) GetProfile(id string) (*AutoSteerProfile, error) {
	return cloneProfile(r.profile), nil
}
func (r coverageProfileRepo) UpdateProfile(id string, updates *AutoSteerProfile) error { return nil }
func (r coverageProfileRepo) DeleteProfile(id string) error                            { return nil }

func TestCoverageReporterReportsKnownUncoveredDependencyGap(t *testing.T) {
	profile := &AutoSteerProfile{
		ID:   "production-ready",
		Name: "Production Ready",
		Objective: Objective{DimensionWeights: map[string]float64{
			"dependencies": 0.8,
			"standards":    1,
		}},
		Ladder: &LadderObjective{Enabled: true, TopRung: "R1"},
	}
	catalog := &skillmap.FakeCatalog{Declarations: []skillmap.SkillDeclaration{
		{ID: "lint-fix", Dimensions: []string{"standards"}},
		{ID: "invalid-skill", Dimensions: []string{"not-a-dimension"}},
	}}
	reporter := NewCoverageReporter(coverageProfileRepo{profile: profile}, catalog, CoveragePolicy{
		KnownUncovered: map[string]KnownUncovered{
			"dependencies": {Reason: "no dependencies skill yet", TrackingRef: "knw-test"},
		},
	})

	report, err := reporter.Report("production-ready", "demo-scenario")
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	if report.Scenario != "demo-scenario" {
		t.Fatalf("scenario = %q, want demo-scenario", report.Scenario)
	}
	if !reflect.DeepEqual(report.EffectiveAllowSet, []string{"lint-fix"}) {
		t.Fatalf("effective allow = %v, want [lint-fix]", report.EffectiveAllowSet)
	}
	if !reflect.DeepEqual(report.ExcludedSkills, []string{"invalid-skill"}) {
		t.Fatalf("excluded skills = %v, want [invalid-skill]", report.ExcludedSkills)
	}
	if len(report.WeightedUnactionable) != 1 || report.WeightedUnactionable[0].Dimension != "dependencies" {
		t.Fatalf("weighted gaps = %+v, want dependencies", report.WeightedUnactionable)
	}
	if report.WeightedUnactionable[0].TrackingRef != "knw-test" {
		t.Fatalf("tracking ref = %q, want knw-test", report.WeightedUnactionable[0].TrackingRef)
	}
	if len(report.KnownUncoveredInPlay) != 1 || report.KnownUncoveredInPlay[0].Dimension != "dependencies" {
		t.Fatalf("known entries = %+v, want dependencies", report.KnownUncoveredInPlay)
	}
}
