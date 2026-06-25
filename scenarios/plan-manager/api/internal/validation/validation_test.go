package validation_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	internalplans "plan-manager/internal/plans"
	"plan-manager/internal/validation"

	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakePlans struct {
	plan internalplans.Plan
	err  error
}

func (f fakePlans) GetPlan(_ context.Context, _ string) (internalplans.Plan, error) {
	return f.plan, f.err
}

type fakeResolver struct {
	resolution internalplans.ReferenceResolution
	err        error
}

func (f fakeResolver) Resolve(_ context.Context, ref internalplans.Reference) (internalplans.Reference, error) {
	if f.err != nil {
		return ref, f.err
	}
	ref.Resolution = f.resolution
	return ref, nil
}

type fakeStaleness struct {
	tier   internalplans.StalenessTier
	factor float64
	err    error
}

func (f fakeStaleness) Compute(_ context.Context, _ internalplans.Reference) (internalplans.StalenessTier, float64, error) {
	return f.tier, f.factor, f.err
}

func planWith(refs []internalplans.Reference, phases []internalplans.Phase) internalplans.Plan {
	return internalplans.Plan{ID: "p1", Slug: "p1", Title: "P", References: refs, Phases: phases}
}

// --- tests ---

func TestResolveReferencesDegradesWhenResolverDown(t *testing.T) {
	plan := planWith([]internalplans.Reference{
		{Kind: internalplans.ReferenceCode, Target: "a.go"},
		{Kind: internalplans.ReferenceCode, Target: "b.go", Future: true},
	}, nil)
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}}) // nil resolver
	report, err := svc.ResolveReferences(context.Background(), "p1", "")
	require.NoError(t, err)
	require.True(t, report.Degraded, "nil resolver => degraded")
	require.Equal(t, internalplans.ResolutionUnresolved, report.References[0].Resolution)
	require.Equal(t, internalplans.ResolutionFuture, report.References[1].Resolution, "future refs are not flagged as deleted")
}

func TestResolveReferencesResolverError(t *testing.T) {
	plan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "a.go"}}, nil)
	svc := validation.NewService(validation.Deps{
		Plans:    fakePlans{plan: plan},
		Resolver: fakeResolver{err: errors.New("code-facts boom")},
	})
	report, err := svc.ResolveReferences(context.Background(), "p1", "")
	require.NoError(t, err)
	require.True(t, report.Degraded)
	require.Equal(t, internalplans.ResolutionUnresolved, report.References[0].Resolution)
}

func TestComputeStalenessTiering(t *testing.T) {
	plan := planWith([]internalplans.Reference{
		{Kind: internalplans.ReferenceCode, Target: "a.go"},
		{Kind: internalplans.ReferenceCode, Target: "b.go"},
	}, nil)
	svc := validation.NewService(validation.Deps{
		Plans:     fakePlans{plan: plan},
		Resolver:  fakeResolver{resolution: internalplans.ResolutionResolved},
		Staleness: fakeStaleness{tier: internalplans.StalenessLightlyStale, factor: 0.3},
	})
	report, err := svc.ComputeStaleness(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, internalplans.StalenessLightlyStale, report.Overall, "overall is the worst tier")
	require.InDelta(t, 0.3, report.References[0].ChangeFactor, 0.001)
}

func TestComputeStalenessUnknownWhenComputerDown(t *testing.T) {
	plan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "a.go"}}, nil)
	svc := validation.NewService(validation.Deps{
		Plans:    fakePlans{plan: plan},
		Resolver: fakeResolver{resolution: internalplans.ResolutionResolved},
		// nil staleness computer
	})
	report, err := svc.ComputeStaleness(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, internalplans.StalenessUnknown, report.Overall)
	require.True(t, report.Degraded)
}

func TestDeriveBaselineScope(t *testing.T) {
	plan := internalplans.Plan{
		ID:   "p1",
		Slug: "p1",
		References: []internalplans.Reference{
			{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/api/main.go"},
			{Kind: internalplans.ReferenceCode, Target: "scenarios/bar/cli/app.go"},
			{Kind: internalplans.ReferenceCode, Target: "packages/api-core/x.go"},
			{Kind: internalplans.ReferenceCode, Target: "scenarios/baz/x.go", Future: true}, // future excluded
			{Kind: internalplans.ReferenceReq, Target: "OT-P0-001"},                         // non-code excluded
		},
		RegressionAnchor: internalplans.RegressionAnchor{Commands: []string{"git-control-tower baseline diff --scenario foo"}},
	}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "")
	require.NoError(t, err)

	require.Contains(t, scope.Commands, "git-control-tower baseline diff --scenario foo")
	require.Contains(t, scope.Commands, "git-control-tower baseline diff --scenario bar")
	require.Contains(t, scope.Commands, "git diff --stat", "non-scenario code => repo-level diff")
	require.Contains(t, scope.Locations, "scenarios/foo")
	require.Contains(t, scope.Locations, "repo")
	// Anchor command is deduped, not duplicated.
	foo := 0
	for _, c := range scope.Commands {
		if c == "git-control-tower baseline diff --scenario foo" {
			foo++
		}
	}
	require.Equal(t, 1, foo, "anchor command deduped against derived command")
}

func TestRunValidationVerdicts(t *testing.T) {
	plan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/x.go"}}, nil)

	// PASS: runner returns no error for every command.
	pass := validation.NewService(validation.Deps{
		Plans:  fakePlans{plan: plan},
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) { return []byte("clean"), nil },
	})
	res, err := pass.RunValidation(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictPass, res.Verdict)

	// FAIL: runner errors (non-zero exit).
	fail := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan},
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("diff"), errors.New("exit 1")
		},
	})
	res, err = fail.RunValidation(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictFail, res.Verdict)

	// UNKNOWN: no runner configured — never a fabricated pass.
	unknown := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	res, err = unknown.RunValidation(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictUnknown, res.Verdict)
}

func TestVerifyDefinitionOfDone(t *testing.T) {
	plan := internalplans.Plan{
		ID:               "p1",
		Slug:             "p1",
		RegressionAnchor: internalplans.RegressionAnchor{Commands: []string{"git-control-tower baseline diff --scenario foo"}},
	}

	// DoD met: oracle exits 0.
	met := validation.NewService(validation.Deps{
		Plans:  fakePlans{plan: plan},
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) { return nil, nil },
	})
	res, ok, err := met.VerifyDefinitionOfDone(context.Background(), "p1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, validation.VerdictPass, res.Verdict)

	// Anchor unavailable => UNKNOWN, not met (never a false pass).
	noAnchor := internalplans.Plan{ID: "p1", Slug: "p1", RegressionAnchor: internalplans.RegressionAnchor{Unavailable: true}}
	degraded := validation.NewService(validation.Deps{Plans: fakePlans{plan: noAnchor}})
	res, ok, err = degraded.VerifyDefinitionOfDone(context.Background(), "p1")
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, validation.VerdictUnknown, res.Verdict)
}

func TestPhaseScopedReferencesAndNotFound(t *testing.T) {
	plan := planWith(
		[]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "plan-level.go"}},
		[]internalplans.Phase{{ID: "ph1", References: []internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "phase.go", Future: true}}}},
	)
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})

	report, err := svc.ResolveReferences(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Len(t, report.References, 1)
	require.Equal(t, "phase.go", report.References[0].Target)

	_, err = svc.ResolveReferences(context.Background(), "p1", "missing")
	require.Error(t, err)
}

func TestFileResolverAndExistenceStaleness(t *testing.T) {
	// FileResolver over a fake stat: a present file => resolved/fresh; an absent
	// one => missing/definitely-stale (the moved/deleted tier).
	present := map[string]bool{filepath.Join("/repo", "exists.go"): true}
	stat := func(path string) (os.FileInfo, error) {
		if present[path] {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}
	r := validation.FileResolver{Root: "/repo", Stat: stat}

	resolved, err := r.Resolve(context.Background(), internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "exists.go"})
	require.NoError(t, err)
	require.Equal(t, internalplans.ResolutionResolved, resolved.Resolution)

	missing, err := r.Resolve(context.Background(), internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "gone.go"})
	require.NoError(t, err)
	require.Equal(t, internalplans.ResolutionMissing, missing.Resolution)

	// REQ references aren't filesystem-resolvable; pass through unspecified.
	req, err := r.Resolve(context.Background(), internalplans.Reference{Kind: internalplans.ReferenceReq, Target: "OT-1"})
	require.NoError(t, err)
	require.Equal(t, internalplans.ResolutionUnspecified, req.Resolution)

	s := validation.NewExistenceStaleness(r)
	tier, factor, err := s.Compute(context.Background(), internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "gone.go"})
	require.NoError(t, err)
	require.Equal(t, internalplans.StalenessDefinitelyStale, tier)
	require.InDelta(t, 1.0, factor, 0.001)

	freshTier, _, err := s.Compute(context.Background(), internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "exists.go"})
	require.NoError(t, err)
	require.Equal(t, internalplans.StalenessFresh, freshTier)
}
