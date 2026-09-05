package hygiene

import (
	"context"
	"errors"
	"testing"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/structureprovider"
)

type fakeStructureProvider struct {
	calls int
	err   error
	id    string
}

func (f *fakeStructureProvider) ID() string {
	if f.id != "" {
		return f.id
	}
	return structureProviderID
}

func (f *fakeStructureProvider) Run(_ context.Context, _ Request, report *Report) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	report.addCheck("repo_contract", true, SeverityInfo, "passed")
	return nil
}

func TestServiceContractSurfaceDelegatesOnce(t *testing.T) {
	fake := &fakeStructureProvider{}
	report, err := (Service{Root: "/repo", StructureProvider: fake}).Run(Request{
		IncludeContract: true,
		FailOn:          SeverityError,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", fake.calls)
	}
	if len(report.Findings) != 0 || !report.Success {
		t.Fatalf("report = %#v", report)
	}
}

type fakeStructureClient struct {
	err error
}

func (f fakeStructureClient) Validate(context.Context, string) (contractapp.ValidationOutput, error) {
	return contractapp.ValidationOutput{}, f.err
}

// An unreachable structure-health says nothing about plan hygiene. It must be
// visible as an unguarded contract while allowing independent hygiene lanes to
// continue and --fail-on error to report only actual errors.
func TestStructureProviderUnavailableStillRunsLaterProviders(t *testing.T) {
	structure := structureProvider{
		root:   "/repo",
		client: fakeStructureClient{err: errors.New("deadline_exceeded: context deadline exceeded")},
	}
	plans := &fakeStructureProvider{}
	plans.id = plansProviderID

	report := Report{Root: "/repo", Success: true}
	registry := NewRegistry(structure, plans)
	if err := registry.Run(context.Background(), Request{}, &report, structureProviderID, plansProviderID); err != nil {
		t.Fatalf("Run returned %v, want the run to continue past the unavailable provider", err)
	}
	if plans.calls != 1 {
		t.Fatalf("later provider ran %d times, want 1", plans.calls)
	}

	report.finish(SeverityError)
	if !report.Success {
		t.Fatal("report.Success = false, want true: an unavailable provider is unguarded, not an error")
	}
	var found bool
	for _, finding := range report.Findings {
		if finding.Code == "repo_contract_unguarded" && finding.Severity == SeverityWarning {
			found = true
		}
	}
	if !found {
		t.Fatalf("findings = %#v, want a warning-severity repo_contract_unguarded finding", report.Findings)
	}
}

func TestServiceDoesNotConvertUnavailableProviderToPass(t *testing.T) {
	fake := &fakeStructureProvider{err: errors.Join(structureprovider.ErrUnavailable, errors.New("offline"))}
	_, err := (Service{Root: "/repo", StructureProvider: fake}).Run(Request{
		IncludeContract: true,
		FailOn:          SeverityError,
	})
	if !errors.Is(err, structureprovider.ErrUnavailable) {
		t.Fatalf("error = %v, want unavailable error", err)
	}
}
