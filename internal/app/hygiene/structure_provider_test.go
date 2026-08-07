package hygiene

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/structureprovider"
)

type fakeStructureProvider struct {
	calls int
	err   error
}

func (f *fakeStructureProvider) ID() string { return structureProviderID }

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
