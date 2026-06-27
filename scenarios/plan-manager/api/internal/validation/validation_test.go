package validation_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

type fakeCommandValidator struct {
	results map[string]validation.CommandReferenceResult
	calls   []validation.CommandReferenceRequest
}

func (f *fakeCommandValidator) ValidateCommandReference(_ context.Context, req validation.CommandReferenceRequest) (validation.CommandReferenceResult, error) {
	f.calls = append(f.calls, req)
	if res, ok := f.results[req.CommandText]; ok {
		return res, nil
	}
	return validation.CommandReferenceResult{Verdict: "unknown", ValidationLevel: "parsed"}, nil
}

func planWith(refs []internalplans.Reference, phases []internalplans.Phase) internalplans.Plan {
	return internalplans.Plan{ID: "p1", Slug: "p1", Title: "P", References: refs, Phases: phases}
}

// --- tests ---

// [REQ:PM-REF-001]
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

// [REQ:PM-STALE-001]
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

// TestStalenessRefinesFreshToLightlyStaleViaGit pins the lightly-stale tier: a
// reference the existence floor calls FRESH (still present) is upgraded to
// LIGHTLY_STALE when git shows its code changed since the anchor HeadSha — with a
// non-zero change factor — while no change leaves it FRESH.
// [REQ:PM-STALE-001]
func TestStalenessRefinesFreshToLightlyStaleViaGit(t *testing.T) {
	plan := internalplans.Plan{
		ID: "p1", Slug: "p1", Title: "P",
		References:       []internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "a.go"}},
		RegressionAnchor: internalplans.RegressionAnchor{HeadSha: "abc123"},
	}

	t.Run("changed code => lightly stale", func(t *testing.T) {
		svc := validation.NewService(validation.Deps{
			Plans:     fakePlans{plan: plan},
			Resolver:  fakeResolver{resolution: internalplans.ResolutionResolved},
			Staleness: fakeStaleness{tier: internalplans.StalenessFresh}, // floor: present
			Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte("12\t3\ta.go\n"), nil // numstat: 12 added, 3 deleted
			},
		})
		report, err := svc.ComputeStaleness(context.Background(), "p1", "")
		require.NoError(t, err)
		require.Equal(t, internalplans.StalenessLightlyStale, report.Overall)
		require.Greater(t, report.References[0].ChangeFactor, 0.0)
	})

	t.Run("no change => stays fresh", func(t *testing.T) {
		svc := validation.NewService(validation.Deps{
			Plans:     fakePlans{plan: plan},
			Resolver:  fakeResolver{resolution: internalplans.ResolutionResolved},
			Staleness: fakeStaleness{tier: internalplans.StalenessFresh},
			Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte(""), nil // numstat empty => unchanged
			},
		})
		report, err := svc.ComputeStaleness(context.Background(), "p1", "")
		require.NoError(t, err)
		require.Equal(t, internalplans.StalenessFresh, report.Overall)
	})

	t.Run("tool absent => floor fresh stands", func(t *testing.T) {
		svc := validation.NewService(validation.Deps{
			Plans:     fakePlans{plan: plan},
			Resolver:  fakeResolver{resolution: internalplans.ResolutionResolved},
			Staleness: fakeStaleness{tier: internalplans.StalenessFresh},
			Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return nil, validation.ErrToolNotFound
			},
		})
		report, err := svc.ComputeStaleness(context.Background(), "p1", "")
		require.NoError(t, err)
		require.Equal(t, internalplans.StalenessFresh, report.Overall)
	})
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

// [REQ:PM-VALID-001]
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
		RegressionAnchor: internalplans.RegressionAnchor{
			BaselineName: "impl",
			Commands:     []string{"git-control-tower baseline diff --scenario foo --name impl"},
		},
	}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "")
	require.NoError(t, err)

	require.Contains(t, scope.Commands, "git-control-tower baseline snapshot status --scenario foo --name impl")
	require.Contains(t, scope.Commands, "git-control-tower baseline snapshot status --scenario bar --name impl")
	require.Contains(t, scope.Commands, "git-control-tower baseline diff --scenario foo --name impl")
	require.Contains(t, scope.Commands, "git-control-tower baseline diff --scenario bar --name impl")
	require.Contains(t, scope.Commands, "git diff --stat", "non-scenario code => repo-level diff")
	require.Contains(t, scope.Locations, "scenarios/foo")
	require.Contains(t, scope.Locations, "repo")
	// Anchor command is deduped, not duplicated.
	foo := 0
	for _, c := range scope.Commands {
		if c == "git-control-tower baseline diff --scenario foo --name impl" {
			foo++
		}
	}
	require.Equal(t, 1, foo, "anchor command deduped against derived command")
}

func TestDeriveBaselineScopeDoesNotFabricateGCTCommandWithoutName(t *testing.T) {
	plan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/x.go"}}, nil)
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Contains(t, scope.Locations, "scenarios/foo")
	require.Empty(t, scope.Commands, "GCT baseline diff requires a verified --name")
}

// [REQ:PM-VALID-002]
func TestRunValidationVerdicts(t *testing.T) {
	plan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/x.go"}}, nil)
	plan.RegressionAnchor = internalplans.RegressionAnchor{BaselineName: "impl"}

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

func TestRunValidationIncludesCommandReferenceFindings(t *testing.T) {
	plan := planWith(nil, []internalplans.Phase{{
		ID: "ph1",
		Intent: strings.Join([]string{
			"Run `cli:vrooli scenario test cli-health`.",
			"Fix `cli:knowledge-observatory docs healt cli-health`.",
			"Document `cli[future]:future-tool launch`.",
		}, "\n"),
	}})
	validator := &fakeCommandValidator{results: map[string]validation.CommandReferenceResult{
		"vrooli scenario test cli-health": {
			Verdict:         "partial",
			ValidationLevel: "command_exists",
			Issues:          []validation.CommandIssue{{Code: "argument_schema_unavailable", Message: "arguments unavailable"}},
		},
		"knowledge-observatory docs healt cli-health": {
			Verdict:         "invalid",
			ValidationLevel: "owner_identified",
			Issues:          []validation.CommandIssue{{Code: "unknown_command", Message: "command path was not found"}},
			Suggestions:     []string{"knowledge-observatory docs health"},
			Guidance:        []string{"Fix the command to a current catalog command."},
		},
	}}
	svc := validation.NewService(validation.Deps{
		Plans:    fakePlans{plan: plan},
		Commands: validator,
	})

	res, err := svc.RunValidation(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictFail, res.Verdict)
	require.Len(t, res.CommandFindings, 2, "future refs should be skipped")
	require.Equal(t, []string{"unknown_command"}, res.CommandFindings[1].IssueCodes)
	require.Equal(t, []string{"knowledge-observatory docs health"}, res.CommandFindings[1].Suggestions)
	require.Equal(t, []string{"Fix the command to a current catalog command."}, res.CommandFindings[1].Guidance)
	require.Len(t, validator.calls, 2)
	require.Contains(t, res.Detail, "command reference validation")
}

// TestVerdictHonestyByExitClass pins the corrected verdict model: a missing tool
// is UNKNOWN (not FAIL — git-control-tower being absent must not look like a
// regression), a baseline-diff exit 2 ("not comparable") is UNKNOWN, an exit 1 is
// FAIL, and an informational-only command set (a bare repo-level diff with no
// oracle) is UNKNOWN even when the command exits 0 — never a false PASS.
func TestVerdictHonestyByExitClass(t *testing.T) {
	scenarioPlan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/x.go"}}, nil)
	scenarioPlan.RegressionAnchor = internalplans.RegressionAnchor{BaselineName: "impl"}
	repoOnlyPlan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "packages/api-core/x.go"}}, nil)

	cases := []struct {
		name   string
		plan   internalplans.Plan
		runner validation.CommandRunner
		want   validation.Verdict
	}{
		{
			name:   "tool absent => UNKNOWN not FAIL",
			plan:   scenarioPlan,
			runner: func(context.Context, string, ...string) ([]byte, error) { return nil, validation.ErrToolNotFound },
			want:   validation.VerdictUnknown,
		},
		{
			name: "baseline exit 2 (not comparable) => UNKNOWN",
			plan: scenarioPlan,
			runner: func(context.Context, string, ...string) ([]byte, error) {
				return nil, validation.CommandExitError{Code: 2}
			},
			want: validation.VerdictUnknown,
		},
		{
			name: "baseline exit 1 (regression) => FAIL",
			plan: scenarioPlan,
			runner: func(context.Context, string, ...string) ([]byte, error) {
				return nil, validation.CommandExitError{Code: 1}
			},
			want: validation.VerdictFail,
		},
		{
			name:   "informational-only repo diff (exit 0, no oracle) => UNKNOWN",
			plan:   repoOnlyPlan,
			runner: func(context.Context, string, ...string) ([]byte, error) { return []byte("M x.go"), nil },
			want:   validation.VerdictUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: tc.plan}, Runner: tc.runner})
			res, err := svc.RunValidation(context.Background(), "p1", "")
			require.NoError(t, err)
			require.Equal(t, tc.want, res.Verdict)
		})
	}
}

// TestVerifyDoDDerivesFromReferences pins the authoring→DoD fix: a wizard-authored
// plan carries the anchor as captured prose with NO explicit commands, yet DoD
// must still verify against a real oracle derived from the plan's connected code
// (it used to always degrade to UNKNOWN/not-met).
// [REQ:PM-VALID-002]
func TestVerifyDoDDerivesFromReferences(t *testing.T) {
	wizardPlan := internalplans.Plan{
		ID:    "p1",
		Slug:  "p1",
		Title: "Wizard plan",
		References: []internalplans.Reference{
			{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/api/main.go"},
		},
		// Captured prose anchor, NO commands — exactly what the authoring wizard writes.
		RegressionAnchor: internalplans.RegressionAnchor{Strategy: "captured", BaselineName: "impl"},
	}
	var ran []string
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: wizardPlan},
		Runner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			ran = append(ran, name)
			return nil, nil // oracle exits 0 => DoD met
		},
	})
	res, ok, err := svc.VerifyDefinitionOfDone(context.Background(), "p1")
	require.NoError(t, err)
	require.True(t, ok, "DoD derives an oracle from references when the anchor has no commands")
	require.Equal(t, validation.VerdictPass, res.Verdict)
	require.NotEmpty(t, ran, "a baseline command was actually derived and run")
	require.Contains(t, res.CommandsRun, "git-control-tower baseline diff --scenario foo --name impl")
}

func TestVerifyDefinitionOfDone(t *testing.T) {
	plan := internalplans.Plan{
		ID:               "p1",
		Slug:             "p1",
		RegressionAnchor: internalplans.RegressionAnchor{Commands: []string{"git-control-tower baseline diff --scenario foo --name impl"}},
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
