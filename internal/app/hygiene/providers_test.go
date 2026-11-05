package hygiene

import (
	"context"
	"testing"
	"time"
)

type overBudgetProvider struct{}

func (overBudgetProvider) ID() string            { return "slow" }
func (overBudgetProvider) Budget() time.Duration { return time.Millisecond }
func (overBudgetProvider) Run(context.Context, Request, *Report) error {
	time.Sleep(5 * time.Millisecond)
	return nil
}

func TestRegistryReportsProviderBudgetOverrunAsWarning(t *testing.T) {
	var report Report
	if err := NewRegistry(overBudgetProvider{}).Run(context.Background(), Request{}, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 || report.Findings[0].Severity != SeverityWarning {
		t.Fatalf("findings = %#v", report.Findings)
	}
	if report.Findings[0].Code != "hygiene_provider_budget" {
		t.Fatalf("finding = %#v", report.Findings[0])
	}
}
