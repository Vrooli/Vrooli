package validation

import (
	"context"
	"slices"
	"testing"
	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

type combinedEvidenceStub struct {
	nativeQualityStub
	reads int
}

func (s *combinedEvidenceStub) CollectQuality(adapters.QualityInput) ([]testquality.Result, testquality.Reason) {
	panic("legacy collector must not run beside combined collector")
}
func (s *combinedEvidenceStub) CollectQualityEvidence(input adapters.QualityInput) ([]testquality.Result, []testquality.TestLinks, testquality.Reason) {
	s.reads++
	return s.rows, []testquality.TestLinks{{Target: s.rows[0].Target, IDs: []string{"UH-CORE-001"}, EvidenceKind: testquality.Runtime, Execution: testquality.ExecutionSkipped, RunID: input.RunID, ExpectedRunID: input.RunID}}, testquality.ReasonNone
}
func TestNativeQualityCollectsRequirementEvidenceOnce(t *testing.T) {
	collector := &combinedEvidenceStub{nativeQualityStub: nativeQualityStub{rows: []testquality.Result{{RuleID: "assertion-observation", RuleVersion: "1", Target: testquality.Target{Workspace: "ui", File: "a.test.ts", TestID: "native"}, SupportProfile: "react-vitest-v2", Status: testquality.Unknown, Reason: testquality.Skipped}}}}
	report, links, reason := collectNativeQualityEvidence(context.Background(), []qualityCollection{{collector: collector, input: adapters.QualityInput{RunID: "current"}}}, true, nil)
	if collector.reads != 1 || reason != testquality.ReasonNone || len(report.Results) != 1 || len(links) != 1 || links[0].Execution != testquality.ExecutionSkipped || links[0].ExpectedRunID != "current" {
		t.Fatalf("combined evidence: reads=%d report=%+v links=%+v reason=%s", collector.reads, report, links, reason)
	}
}

type nativeQualityStub struct {
	rows   []testquality.Result
	reason testquality.Reason
}

func (s nativeQualityStub) QualityArtifact(string, string) adapters.Artifact {
	return adapters.Artifact{}
}
func (s nativeQualityStub) CollectQuality(adapters.QualityInput) ([]testquality.Result, testquality.Reason) {
	return s.rows, s.reason
}

func TestNativeQualityAggregateCannotHideUnavailableWorkspace(t *testing.T) {
	row := testquality.Result{RuleID: "assertion-observation", RuleVersion: "1", Target: testquality.Target{Workspace: "ui", File: "a.test.ts", TestID: "native"}, SupportProfile: "react-vitest-v2", Status: testquality.CheckedClean, Reason: testquality.ReasonNone}
	collections := []qualityCollection{{collector: nativeQualityStub{rows: []testquality.Result{row}, reason: testquality.ReasonNone}}, {collector: nativeQualityStub{reason: testquality.MissingInput}}}
	report := collectNativeQuality(collections, true).Normalized()
	if report.UnavailableReason != "" || len(report.IncompleteReasons) != 1 || report.IncompleteReasons[0] != testquality.MissingInput || len(report.Coverage) != 1 || len(report.Results) != 1 || report.Results[0].Status != testquality.CheckedClean {
		t.Fatalf("partial collection lost independent observations or concealed missing scope: %+v", report)
	}
	multiple := append(append([]qualityCollection(nil), collections...), qualityCollection{collector: nativeQualityStub{reason: testquality.UnsupportedVersion}})
	partial := collectNativeQuality(multiple, true).Normalized()
	if len(partial.IncompleteReasons) != 2 || partial.IncompleteReasons[0] != testquality.MissingInput || partial.IncompleteReasons[1] != testquality.UnsupportedVersion {
		t.Fatalf("collector reasons overwritten: %+v", partial.IncompleteReasons)
	}
	collections[1].collector = nativeQualityStub{rows: []testquality.Result{row}, reason: testquality.ReasonNone}
	if report := collectNativeQuality(collections, true); report.UnavailableReason != testquality.MissingInput {
		t.Fatalf("duplicate identity became valid: %+v", report)
	}
}

func TestNativeQualityPreparationIsCommandUniqueAndNonMutating(t *testing.T) {
	sharedEnv := map[string]string{"KEEP": "yes"}
	ws := []Workspace{{ID: "ui", RootPath: t.TempDir(), AdapterID: "react-vitest", Language: "typescript", Framework: "vitest"}}
	plan := ExecutionPlan{Commands: []PlannedCommand{{WorkspaceID: "ui", TestKind: "unit", Env: sharedEnv}, {WorkspaceID: "ui", TestKind: "typecheck", Env: sharedEnv}}}
	first, err := prepareNativeQuality(&plan, ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(sharedEnv) != 1 || len(plan.Commands[1].Env) != 1 {
		t.Fatalf("mutation or wrong scope: %+v", plan)
	}
	if plan.Commands[0].Env["VROOLI_TEST_RUN_ID"] != first[0].input.RunID || plan.Commands[0].Env["KEEP"] != "yes" {
		t.Fatal("handoff env missing")
	}
	secondPlan := ExecutionPlan{Commands: []PlannedCommand{{WorkspaceID: "ui", TestKind: "unit"}}}
	second, err := prepareNativeQuality(&secondPlan, ws)
	if err != nil {
		t.Fatal(err)
	}
	if first[0].input.RunID == second[0].input.RunID || first[0].input.Artifact.Path == second[0].input.Artifact.Path {
		t.Fatal("run identity reused")
	}
	// This empty workspace lacks syntax-owner inputs and did not execute its
	// native command. Both limitations must survive; neither is a passing row.
	if got := collectNativeQuality(first, false); got.UnavailableReason == "" || got.TotalResults != 0 || len(got.Results) != 0 || len(got.Coverage) != 0 || len(got.IncompleteReasons) != 2 || !slices.Contains(got.IncompleteReasons, testquality.NotExecuted) || !slices.Contains(got.IncompleteReasons, testquality.MissingInput) {
		t.Fatalf("dry run fabricated evidence or discarded a collection limitation: %+v", got)
	}
	if got := collectNativeQuality(first, true); got.UnavailableReason != testquality.MissingInput {
		t.Fatalf("missing reporter artifact became clean: %+v", got)
	}
}
