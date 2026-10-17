package hygiene

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/internal/tidinessprovider"
)

type fakeTidinessProvider struct {
	result tidinessprovider.Result
	err    error
}

func (f fakeTidinessProvider) ID() string { return tidinessProviderID }

func (f fakeTidinessProvider) Run(_ context.Context, req Request, report *Report) error {
	return tidinessProvider{client: f}.Run(context.Background(), req, report)
}

func (f fakeTidinessProvider) Validate(context.Context, string) (tidinessprovider.Result, error) {
	return f.result, f.err
}

func TestTidinessProviderUnavailableIsBlockingWhenRequired(t *testing.T) {
	report, err := (Service{Root: "/repo", TidinessProvider: fakeTidinessProvider{err: tidinessprovider.ErrUnavailable}}).Run(Request{IncludeTidiness: true, RequireTidinessProvider: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Success {
		t.Fatal("unavailable required tidiness provider reported success")
	}
	if report.Findings[0].Code != "tidiness_provider_unavailable" {
		t.Fatalf("finding code = %q", report.Findings[0].Code)
	}
}

func TestTidinessProviderBudgetFindingIsBlocking(t *testing.T) {
	report, err := (Service{Root: "/repo", TidinessProvider: fakeTidinessProvider{result: tidinessprovider.Result{
		Status:   "VALIDATION_STATUS_FAILED",
		Findings: []tidinessprovider.Finding{{Code: "TIDINESS_BUDGET_EXCEEDED", Severity: "error", Message: "budget exceeded"}},
	}}}).Run(Request{IncludeTidiness: true, RequireTidinessProvider: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Success || report.BlockingFailures == 0 {
		t.Fatal("budget breach did not block hygiene")
	}
}

func TestTidinessProviderCleanResultPasses(t *testing.T) {
	report, err := (Service{Root: "/repo", TidinessProvider: fakeTidinessProvider{result: tidinessprovider.Result{Status: "VALIDATION_STATUS_PASSED"}}}).Run(Request{IncludeTidiness: true, RequireTidinessProvider: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !report.Success {
		t.Fatalf("clean provider result failed: %+v", report)
	}
}
