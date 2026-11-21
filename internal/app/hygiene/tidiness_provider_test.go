package hygiene

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	plansapp "github.com/vrooli/vrooli/internal/app/plans"
	"github.com/vrooli/vrooli/internal/tidinessprovider"
)

type hygieneProviderFixture struct {
	Name      string `json:"name"`
	Structure struct {
		Success      bool                            `json:"success"`
		SchemaPassed bool                            `json:"schema_passed"`
		Required     bool                            `json:"required"`
		Error        string                          `json:"error"`
		Findings     []contractapp.StructuralFinding `json:"findings"`
	} `json:"structure"`
	Tidiness struct {
		Status   string `json:"status"`
		Required bool   `json:"required"`
		Findings []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
		} `json:"findings"`
	} `json:"tidiness"`
	SDA struct {
		Clean       bool                  `json:"clean"`
		Mode        string                `json:"mode"`
		Summary     sdaFreshnessSummary   `json:"summary"`
		Surfaces    []sdaFreshnessSurface `json:"surfaces"`
		NextActions []sdaFreshnessAction  `json:"next_actions"`
	} `json:"sda"`
	SDAError string `json:"sda_error"`
	Plans    struct {
		Required bool   `json:"required"`
		Error    string `json:"error"`
	} `json:"plans"`
	WantSuccess  bool     `json:"want_success"`
	WantBlocking int      `json:"want_blocking"`
	WantWarnings int      `json:"want_warnings"`
	WantCodes    []string `json:"want_codes"`
}

type fixtureStructureClient struct {
	output contractapp.ValidationOutput
	err    error
}

func (f fixtureStructureClient) Validate(context.Context, string) (contractapp.ValidationOutput, error) {
	return f.output, f.err
}

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

func TestTidinessProviderSharedContractCorpus(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "hygiene-provider-corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []hygieneProviderFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			root := t.TempDir()
			tidinessFindings := make([]tidinessprovider.Finding, 0, len(fixture.Tidiness.Findings))
			for _, finding := range fixture.Tidiness.Findings {
				tidinessFindings = append(tidinessFindings, tidinessprovider.Finding{Code: finding.Code, Severity: finding.Severity, Message: finding.Message})
			}
			structureOutput := contractapp.ValidationOutput{
				Success:  fixture.Structure.Success,
				Root:     root,
				Schema:   contractapp.ValidationCheck{Passed: fixture.Structure.SchemaPassed},
				Report:   contractapp.Report{Root: root, Success: fixture.Structure.Success},
				Findings: fixture.Structure.Findings,
			}
			report, err := (Service{
				Root:                  root,
				PathAuthorityProvider: passingPathAuthorityProvider{},
				StructureProvider: NewStructureProvider(root, fixtureStructureClient{
					output: structureOutput,
					err: func() error {
						if fixture.Structure.Error != "" {
							return errors.New(fixture.Structure.Error)
						}
						return nil
					}(),
				}),
				TidinessProvider: fakeTidinessProvider{result: tidinessprovider.Result{Status: fixture.Tidiness.Status, Findings: tidinessFindings}},
				PlanReconciler: &fakePlanReconciler{err: func() error {
					if fixture.Plans.Error != "" {
						return plansapp.ErrPlanManagerUnavailable
					}
					return nil
				}()},
				DependencyFreshnessRunner: stubDependencyFreshnessRunner{err: func() error {
					if fixture.SDAError != "" {
						return errors.New(fixture.SDAError)
					}
					return nil
				}(), report: sdaFreshnessReport{
					Clean: fixture.SDA.Clean, Mode: fixture.SDA.Mode, Summary: fixture.SDA.Summary,
					Surfaces: fixture.SDA.Surfaces, NextActions: fixture.SDA.NextActions,
				}},
			}).Run(Request{
				IncludeContract:         true,
				RequireContractProvider: fixture.Structure.Required,
				IncludePlans:            fixture.Plans.Required,
				RequirePlanProvider:     fixture.Plans.Required,
				IncludeTidiness:         true,
				RequireTidinessProvider: fixture.Tidiness.Required,
				IncludeDrift:            true,
			})
			if err != nil {
				t.Fatal(err)
			}
			if report.Success != fixture.WantSuccess || report.BlockingFailures != fixture.WantBlocking || report.Warnings != fixture.WantWarnings {
				t.Fatalf("report success/blocking/warnings = %v/%d/%d, want %v/%d/%d; findings=%+v", report.Success, report.BlockingFailures, report.Warnings, fixture.WantSuccess, fixture.WantBlocking, fixture.WantWarnings, report.Findings)
			}
			for _, code := range fixture.WantCodes {
				count := 0
				for _, finding := range report.Findings {
					if finding.Code == code {
						count++
					}
				}
				if count != 1 {
					t.Fatalf("finding %q appeared %d times, want once; findings=%+v", code, count, report.Findings)
				}
			}
		})
	}
}
