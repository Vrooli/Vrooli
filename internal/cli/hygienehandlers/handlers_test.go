package hygienehandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	hygieneapp "github.com/vrooli/vrooli/internal/app/hygiene"
	plansapp "github.com/vrooli/vrooli/internal/app/plans"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/tidinessprovider"
)

type fixtureFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type providerFixture struct {
	Name      string `json:"name"`
	Structure struct {
		Success      bool                            `json:"success"`
		SchemaPassed bool                            `json:"schema_passed"`
		Required     bool                            `json:"required"`
		Error        string                          `json:"error"`
		Findings     []contractapp.StructuralFinding `json:"findings"`
	} `json:"structure"`
	Tidiness struct {
		Status   string           `json:"status"`
		Required bool             `json:"required"`
		Findings []fixtureFinding `json:"findings"`
	} `json:"tidiness"`
	SDA      json.RawMessage `json:"sda"`
	SDAError string          `json:"sda_error"`
	Plans    struct {
		Required bool   `json:"required"`
		Error    string `json:"error"`
	} `json:"plans"`
	WantSuccess  bool     `json:"want_success"`
	WantBlocking int      `json:"want_blocking"`
	WantWarnings int      `json:"want_warnings"`
	WantCodes    []string `json:"want_codes"`
}

type fixtureClient struct{ result tidinessprovider.Result }

func (f fixtureClient) Validate(context.Context, string) (tidinessprovider.Result, error) {
	return f.result, nil
}

type fixtureStructureClient struct {
	output contractapp.ValidationOutput
	err    error
}

type fixturePathAuthorityProvider struct{}

func (fixturePathAuthorityProvider) ID() string { return "runtime-path-authority" }
func (fixturePathAuthorityProvider) Run(_ context.Context, _ hygieneapp.Request, report *hygieneapp.Report) error {
	report.Checks = append(report.Checks, hygieneapp.Check{Name: "runtime-path-authority", Passed: true, Severity: hygieneapp.SeverityInfo})
	return nil
}

func (f fixtureStructureClient) Validate(context.Context, string) (contractapp.ValidationOutput, error) {
	return f.output, f.err
}

type emptyPlanReconciler struct{ err error }

func (r emptyPlanReconciler) ReconcilePlans(context.Context, hygieneapp.PlanReconcileRequest) (hygieneapp.PlanReconcileReport, error) {
	return hygieneapp.PlanReconcileReport{}, r.err
}

func TestHandlerConstructsServiceBeforeParsing(t *testing.T) {
	wantErr := errors.New("output format unavailable")
	handler := Handler(rootcli.HandlerDeps[string]{
		Stdout: func(string) io.Writer { return &bytes.Buffer{} },
		OutputFormat: func(string) (cliout.Format, error) {
			return "", wantErr
		},
		HomeDir: func(string) (string, error) {
			t.Fatal("home lookup ran after output format failed")
			return "", nil
		},
	})

	if err := handler("ctx", []string{"--invalid"}); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestPublicHygieneCLIUsesSharedProviderContractCorpus(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "hygiene-provider-corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []providerFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	binDir := t.TempDir()
	sdaFixturePath := filepath.Join(t.TempDir(), "sda.json")
	scriptPath := filepath.Join(binDir, "scenario-dependency-analyzer")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nif [ -n \"$HYGIENE_SDA_FAIL\" ]; then echo provider unavailable >&2; exit 1; fi\ncat \"$HYGIENE_SDA_FIXTURE\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	previousPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+previousPath)
	t.Setenv("HYGIENE_SDA_FIXTURE", sdaFixturePath)
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			if fixture.SDAError == "" {
				t.Setenv("HYGIENE_SDA_FAIL", "")
				var sdaReport map[string]any
				if err := json.Unmarshal(fixture.SDA, &sdaReport); err != nil {
					t.Fatal(err)
				}
				sdaReport["root"] = root
				fixture.SDA, err = json.Marshal(sdaReport)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				t.Setenv("HYGIENE_SDA_FAIL", "1")
			}
			if err := os.WriteFile(sdaFixturePath, append(fixture.SDA, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			findings := make([]tidinessprovider.Finding, 0, len(fixture.Tidiness.Findings))
			for _, finding := range fixture.Tidiness.Findings {
				findings = append(findings, tidinessprovider.Finding{Code: finding.Code, Severity: finding.Severity, Message: finding.Message})
			}
			result := tidinessprovider.Result{Status: fixture.Tidiness.Status, Findings: findings}
			var planErr error
			if fixture.Plans.Error != "" {
				planErr = plansapp.ErrPlanManagerUnavailable
			}
			structureOutput := contractapp.ValidationOutput{
				Success:  fixture.Structure.Success,
				Root:     root,
				Schema:   contractapp.ValidationCheck{Passed: fixture.Structure.SchemaPassed},
				Report:   contractapp.Report{Root: root, Success: fixture.Structure.Success},
				Findings: fixture.Structure.Findings,
			}
			var stdout bytes.Buffer
			outputFormat := cliout.FormatJSON
			deps := rootcli.HandlerDeps[string]{
				Stdout:       func(string) io.Writer { return &stdout },
				Root:         func(string) string { return root },
				HomeDir:      func(string) (string, error) { return t.TempDir(), nil },
				OutputFormat: func(string) (cliout.Format, error) { return outputFormat, nil },
			}
			handler := handlerWithService(deps, func(root, home string) hygieneapp.Service {
				return hygieneapp.Service{
					Root: root, Home: home, PathAuthorityProvider: fixturePathAuthorityProvider{},
					StructureProvider: hygieneapp.NewStructureProvider(root, fixtureStructureClient{
						output: structureOutput,
						err: func() error {
							if fixture.Structure.Error != "" {
								return errors.New(fixture.Structure.Error)
							}
							return nil
						}(),
					}),
					TidinessProvider: hygieneapp.NewTidinessProvider(fixtureClient{result: result}),
					PlanReconciler:   emptyPlanReconciler{err: planErr},
				}
			})
			args := []string{"--no-freshness", "--json"}
			if fixture.Structure.Required {
				args = []string{"--contract-only", "--json"}
			} else if fixture.Tidiness.Required {
				args = []string{"--tidiness-only", "--json"}
			} else if fixture.Plans.Required {
				args = []string{"--plans-only", "--json"}
			}
			err := handler("ctx", args)
			if fixture.WantSuccess && err != nil {
				t.Fatalf("handler returned %v for successful fixture", err)
			}
			if !fixture.WantSuccess && err == nil {
				t.Fatal("blocking fixture returned success")
			}
			if !fixture.WantSuccess {
				var exitErr rootcli.ExitCodeError
				if !errors.As(err, &exitErr) || exitErr.Code != 1 {
					t.Fatalf("handler error = %v, want exit code 1", err)
				}
			}
			var actual struct {
				Success  bool `json:"success"`
				Blocking int  `json:"blocking_failures"`
				Warnings int  `json:"warnings"`
				Findings []struct {
					Code string `json:"code"`
				} `json:"findings"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &actual); err != nil {
				t.Fatal(err)
			}
			if actual.Success != fixture.WantSuccess || actual.Blocking != fixture.WantBlocking || actual.Warnings != fixture.WantWarnings {
				t.Fatalf("CLI success/blocking/warnings = %v/%d/%d, want %v/%d/%d (%s)", actual.Success, actual.Blocking, actual.Warnings, fixture.WantSuccess, fixture.WantBlocking, fixture.WantWarnings, stdout.String())
			}
			for _, code := range fixture.WantCodes {
				count := 0
				for _, finding := range actual.Findings {
					if finding.Code == code {
						count++
					}
				}
				if count != 1 {
					t.Fatalf("CLI finding %s count = %d, want once", code, count)
				}
			}

			// Run the same corpus through the public human CLI renderer too. The
			// machine pass above proves complete JSON evidence; this pass proves
			// bounded operator output and one visible attribution per blocker.
			stdout.Reset()
			outputFormat = cliout.FormatHuman
			humanArgs := args[:len(args)-1]
			err = handler("ctx", humanArgs)
			if fixture.WantSuccess && err != nil {
				t.Fatalf("human handler returned %v for successful fixture", err)
			}
			if !fixture.WantSuccess {
				var exitErr rootcli.ExitCodeError
				if !errors.As(err, &exitErr) || exitErr.Code != 1 {
					t.Fatalf("human handler error = %v, want exit code 1", err)
				}
			}
			if stdout.Len() > 32<<10 {
				t.Fatalf("human CLI output = %d bytes, limit 32 KiB", stdout.Len())
			}
			for _, code := range fixture.WantCodes {
				if count := strings.Count(stdout.String(), code); count != 1 {
					t.Fatalf("human CLI finding %s count = %d, want once; output=%s", code, count, stdout.String())
				}
			}
		})
	}
}
