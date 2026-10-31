package testquality

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/vrooli/api-core/relationshiprefs"
)

func TestRetainedRequirementContextCalibration(t *testing.T) {
	data, err := os.ReadFile("testdata/context-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Group, ExpectedOutcome string
		Input                      struct {
			Registry []struct {
				ID              string
				ValidationKinds []string
			}
			RegistryError                     json.RawMessage
			RequestedKind, SuiteTitle, Source string
			Tests                             []struct {
				ID, Title, Status string
				Tags              []string
			}
			Runtime []struct{ TestID, Status string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	reports := map[string]TraceabilityReport{}
	for _, c := range cases {
		if c.Group != "Requirements" {
			continue
		}
		t.Run(c.ID, func(t *testing.T) {
			registry := RequirementRegistry{SchemaVersion: "requirement-registry/v1", Requirements: []RequirementDeclaration{}}
			for _, r := range c.Input.Registry {
				d := RequirementDeclaration{ID: r.ID}
				for _, kind := range r.ValidationKinds {
					d.Validations = append(d.Validations, RequirementResponsibility{Phase: kind})
				}
				registry.Requirements = append(registry.Requirements, d)
			}
			unavailable := ReasonNone
			if len(c.Input.RegistryError) > 0 {
				unavailable = OwnerUnavailable
			}
			var links []TestLinks
			for _, test := range c.Input.Tests {
				ids := append([]string(nil), test.Tags...)
				ids = append(ids, relationshiprefs.ExtractTestRequirementIDs(c.Input.SuiteTitle)...)
				ids = append(ids, relationshiprefs.ExtractTestRequirementIDs(test.Title)...)
				link := TestLinks{Target: Target{Workspace: "fixture", File: "fixture_test.go", TestID: test.ID}, IDs: ids, EvidenceKind: Static, Execution: ExecutionNotRun}
				state := test.Status
				for _, event := range c.Input.Runtime {
					if event.TestID == test.ID {
						state = event.Status
					}
				}
				if state != "" {
					link.EvidenceKind = Runtime
					link.Execution = ExecutionState(state)
					link.RunID = "retained-fixture"
					link.ExpectedRunID = "retained-fixture"
				}
				links = append(links, link)
			}
			phase := c.Input.RequestedKind
			if phase == "" {
				phase = "unit"
			}
			report := ReconcileRequirements(registry, unavailable, phase, links)
			reports[c.ID] = report
			switch c.ExpectedOutcome {
			case "linked":
				if len(report.Links) == 0 {
					t.Fatal("no links")
				}
				for _, link := range report.Links {
					if link.Registration != "registered" || link.Execution != ExecutionPassed {
						t.Fatalf("link/execute distinction: %+v", report)
					}
				}
			case "not_executed":
				if len(report.Links) != 0 {
					t.Fatalf("bare comment manufactured links: %+v", report)
				}
			case "not_applicable_unit":
				if len(report.Requirements) != 1 || report.Requirements[0].Applicability != "not_applicable" {
					t.Fatalf("ownership: %+v", report)
				}
			case "unknown":
				if report.UnavailableReason != OwnerUnavailable || len(report.Links) != 1 || report.Links[0].Registration != "unknown" {
					t.Fatalf("owner unavailable: %+v", report)
				}
			case "exact_match":
				if len(report.Links) != 1 || report.Links[0].ID != "UH-CORE-0010" || report.Links[0].Registration != "registered" {
					t.Fatalf("prefix collision: %+v", report)
				}
			default:
				t.Fatalf("unhandled retained outcome %q", c.ExpectedOutcome)
			}
		})
	}
	if len(reports) != 6 {
		t.Fatalf("retained requirement inventory changed: %d", len(reports))
	}
	if output := os.Getenv("UNIT_HEALTH_TRACE_CALIBRATION_OUTPUT"); output != "" {
		data, err := json.MarshalIndent(reports, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(output, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
