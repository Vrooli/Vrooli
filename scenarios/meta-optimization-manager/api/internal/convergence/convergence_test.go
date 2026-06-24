package convergence

import (
	"context"
	"testing"
	"time"

	"meta-optimization-manager/internal/testutil/mocks"
)

// fakeFitness / fakeRefs / fakeRepo are in-memory seams for service tests.
type fakeFitness struct {
	out []TemplateFitness
	err error
}

func (f *fakeFitness) Scan(_ context.Context, template string) ([]TemplateFitness, error) {
	if f.err != nil {
		return nil, f.err
	}
	if template == "" {
		return f.out, nil
	}
	var filtered []TemplateFitness
	for _, tf := range f.out {
		if tf.Template == template {
			filtered = append(filtered, tf)
		}
	}
	return filtered, nil
}

type fakeRefs struct {
	out []ReferenceHealth
	err error
}

func (f *fakeRefs) Scan(_ context.Context) ([]ReferenceHealth, error) { return f.out, f.err }

type fakeRepo struct {
	fitnessSaves int
	refSaves     int
	trend        []FitnessTrendPoint
}

func (r *fakeRepo) SaveFitness(_ context.Context, _ []TemplateFitness, _ time.Time) error {
	r.fitnessSaves++
	return nil
}

func (r *fakeRepo) SaveReferences(_ context.Context, _ []ReferenceHealth, _ time.Time) error {
	r.refSaves++
	return nil
}

func (r *fakeRepo) Trend(_ context.Context, _ string) ([]FitnessTrendPoint, error) {
	return r.trend, nil
}

func newSvc(f FitnessScanner, refs ReferenceScanner, repo Repository) Service {
	return NewService(Deps{Fitness: f, References: refs, Repo: repo, Clock: mocks.NewFakeClock(time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC))})
}

func TestStatusComputesAndPersists(t *testing.T) {
	f := &fakeFitness{out: []TemplateFitness{{Template: "react-vite", PerReplicaCost: 1000, CoordinatedEditCount: 5, Tier: TierFair}}}
	refs := &fakeRefs{out: []ReferenceHealth{{Scenario: "reference-react-vite", Eligibility: EligibilityCandidate}}}
	repo := &fakeRepo{}
	svc := newSvc(f, refs, repo)

	status, err := svc.GetConvergenceStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(status.Templates) != 1 || status.Templates[0].Template != "react-vite" {
		t.Fatalf("templates = %+v", status.Templates)
	}
	if len(status.References) != 1 {
		t.Fatalf("references = %+v", status.References)
	}
	if repo.fitnessSaves != 1 || repo.refSaves != 1 {
		t.Fatalf("expected persistence: fitness=%d ref=%d", repo.fitnessSaves, repo.refSaves)
	}
}

func TestStatusDegradesWhenScannerErrors(t *testing.T) {
	svc := newSvc(&fakeFitness{err: context.DeadlineExceeded}, &fakeRefs{err: context.DeadlineExceeded}, &fakeRepo{})
	status, err := svc.GetConvergenceStatus(context.Background())
	if err != nil {
		t.Fatalf("status should degrade, got err=%v", err)
	}
	if len(status.Templates) != 0 || len(status.References) != 0 {
		t.Fatalf("expected empty on scanner failure, got %+v", status)
	}
}

func TestTemplateFitnessFilters(t *testing.T) {
	f := &fakeFitness{out: []TemplateFitness{
		{Template: "react-vite"}, {Template: "landing-page-react-vite"},
	}}
	svc := newSvc(f, &fakeRefs{}, &fakeRepo{})
	one, err := svc.GetTemplateFitness(context.Background(), "react-vite")
	if err != nil || len(one) != 1 || one[0].Template != "react-vite" {
		t.Fatalf("filter failed: %+v err=%v", one, err)
	}
}

func TestListReferencesEligibilityFilter(t *testing.T) {
	refs := &fakeRefs{out: []ReferenceHealth{
		{Scenario: "a", Eligibility: EligibilityEligible},
		{Scenario: "b", Eligibility: EligibilityIneligible},
	}}
	svc := newSvc(&fakeFitness{}, refs, &fakeRepo{})
	got, err := svc.ListReferences(context.Background(), EligibilityEligible)
	if err != nil || len(got) != 1 || got[0].Scenario != "a" {
		t.Fatalf("eligibility filter failed: %+v err=%v", got, err)
	}
}

func TestTrendReadsRepo(t *testing.T) {
	repo := &fakeRepo{trend: []FitnessTrendPoint{{Template: "react-vite", PerReplicaCost: 900}}}
	svc := newSvc(&fakeFitness{}, &fakeRefs{}, repo)
	pts, err := svc.GetConvergenceTrend(context.Background(), "react-vite")
	if err != nil || len(pts) != 1 || pts[0].PerReplicaCost != 900 {
		t.Fatalf("trend = %+v err=%v", pts, err)
	}
}

func TestTrendNilRepoEmpty(t *testing.T) {
	svc := NewService(Deps{Fitness: &fakeFitness{}, References: &fakeRefs{}})
	pts, err := svc.GetConvergenceTrend(context.Background(), "")
	if err != nil || len(pts) != 0 {
		t.Fatalf("nil repo should yield empty trend, got %+v err=%v", pts, err)
	}
}

func TestDeriveTier(t *testing.T) {
	cases := []struct {
		tf   TemplateFitness
		want FitnessTier
	}{
		{TemplateFitness{PerReplicaCost: 500, CoordinatedEditCount: 3}, TierStrong},
		{TemplateFitness{CommentOnlyContractCount: 10, CoordinatedEditCount: 6}, TierFair},
		{TemplateFitness{DriftSurfaceCount: 20, CommentOnlyContractCount: 5}, TierWeak},
	}
	for i, c := range cases {
		if got := deriveTier(c.tf); got != c.want {
			t.Errorf("case %d: deriveTier=%v want %v", i, got, c.want)
		}
	}
}

func TestDeriveEligibility(t *testing.T) {
	eligible := ReferenceHealth{CleanOnAllTools: true, StabilityDays: 90, Breadth: 4}
	if deriveEligibility(eligible) != EligibilityEligible {
		t.Errorf("expected eligible")
	}
	candidate := ReferenceHealth{StabilityDays: 70, Breadth: 2}
	if deriveEligibility(candidate) != EligibilityCandidate {
		t.Errorf("expected candidate, got %v", deriveEligibility(candidate))
	}
	ineligible := ReferenceHealth{StabilityDays: 10, Breadth: 1}
	if deriveEligibility(ineligible) != EligibilityIneligible {
		t.Errorf("expected ineligible")
	}
}
