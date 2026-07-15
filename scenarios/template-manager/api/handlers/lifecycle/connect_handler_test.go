package lifecycle

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templateengine"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/validationrunner"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"
	lifecyclev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/lifecycle"
	testdb "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/testutil/db"
)

func TestGenerateScenarioValuesMapsDisplayNamePlaceholder(t *testing.T) {
	values := generateScenarioValues(&lifecyclev1.GenerateScenarioRequest{
		Id:          "alpha",
		DisplayName: "Alpha App",
		Description: "Alpha description",
		Values: map[string]string{
			"AUTHOR": "Test Agent",
		},
	})

	if values["SCENARIO_ID"] != "alpha" {
		t.Fatalf("SCENARIO_ID = %q, want alpha", values["SCENARIO_ID"])
	}
	if values["SCENARIO_DISPLAY_NAME"] != "Alpha App" {
		t.Fatalf("SCENARIO_DISPLAY_NAME = %q, want Alpha App", values["SCENARIO_DISPLAY_NAME"])
	}
	if values["SCENARIO_DESCRIPTION"] != "Alpha description" {
		t.Fatalf("SCENARIO_DESCRIPTION = %q, want Alpha description", values["SCENARIO_DESCRIPTION"])
	}
	if values["AUTHOR"] != "Test Agent" {
		t.Fatalf("AUTHOR = %q, want Test Agent", values["AUTHOR"])
	}
	if _, ok := values["SCENARIO_NAME"]; ok {
		t.Fatalf("SCENARIO_NAME should not be emitted; templates consume SCENARIO_DISPLAY_NAME")
	}
}

func TestValidateTemplateRecordsLifecycleValidation(t *testing.T) {
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(catalog.Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := catalog.NewSQLiteRepository(db)
	service := validationrunner.NewService(repo, lifecycleRunner{result: validationrunner.ValidateResult{
		Success:    false,
		TemplateID: "react-vite",
		Mode:       catalog.ModeDeep,
		Findings: []catalog.ValidationFinding{{
			Key:      "react-vite.deep.failure",
			Severity: "warning",
			Summary:  "deep validation failed",
			Source:   "test",
		}},
	}})
	handler := NewConnectHandler(nil, service)

	resp, err := handler.ValidateTemplate(context.Background(), connect.NewRequest(&lifecyclev1.ValidateTemplateRequest{
		Template: "react-vite",
		Mode:     string(catalog.ModeDeep),
	}))
	if err != nil {
		t.Fatalf("ValidateTemplate: %v", err)
	}
	if len(resp.Msg.Issues) != 1 || resp.Msg.Issues[0].Message != "deep validation failed" {
		t.Fatalf("response issues = %#v", resp.Msg.Issues)
	}
	if resp.Msg.Status != "fail" {
		t.Fatalf("status = %q, want fail", resp.Msg.Status)
	}
	if resp.Msg.IssuesCount != 1 {
		t.Fatalf("issues_count = %d, want 1", resp.Msg.IssuesCount)
	}
	if resp.Msg.Issues[0].Path != "react-vite.deep.failure" {
		t.Fatalf("issue path = %q, want the finding key surfaced as the failure class", resp.Msg.Issues[0].Path)
	}
	runs, err := repo.ListValidationRuns(context.Background(), "react-vite")
	if err != nil {
		t.Fatalf("ListValidationRuns: %v", err)
	}
	var recorded *catalog.ValidationRun
	for i := range runs {
		if runs[i].Mode == catalog.ModeDeep {
			recorded = &runs[i]
			break
		}
	}
	if recorded == nil || recorded.Status != "failed" {
		t.Fatalf("runs = %#v", runs)
	}
}

func TestValidateTemplateWarnsRetainTempInShallowMode(t *testing.T) {
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(catalog.Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := catalog.NewSQLiteRepository(db)
	service := validationrunner.NewService(repo, lifecycleRunner{result: validationrunner.ValidateResult{
		Success:    true,
		TemplateID: "react-vite",
		Mode:       catalog.ModeShallow,
	}})
	handler := NewConnectHandler(nil, service)

	resp, err := handler.ValidateTemplate(context.Background(), connect.NewRequest(&lifecyclev1.ValidateTemplateRequest{
		Template:   "react-vite",
		RetainTemp: true,
	}))
	if err != nil {
		t.Fatalf("ValidateTemplate: %v", err)
	}
	if len(resp.Msg.Warnings) != 1 {
		t.Fatalf("warnings = %#v, want one retain-temp advisory", resp.Msg.Warnings)
	}
	if resp.Msg.Status != "pass" {
		t.Fatalf("status = %q, want pass", resp.Msg.Status)
	}
}

func TestRetainTempWarnings(t *testing.T) {
	if got := retainTempWarnings("", false); got != nil {
		t.Fatalf("retain=false warnings = %#v, want none", got)
	}
	if got := retainTempWarnings("deep", true); got != nil {
		t.Fatalf("deep+retain warnings = %#v, want none", got)
	}
	if got := retainTempWarnings("", true); len(got) != 1 {
		t.Fatalf("shallow(default)+retain warnings = %#v, want one", got)
	}
	if got := retainTempWarnings("shallow", true); len(got) != 1 {
		t.Fatalf("shallow+retain warnings = %#v, want one", got)
	}
}

func TestValidationVerdict(t *testing.T) {
	if validationVerdict("passed") != "pass" {
		t.Fatalf("passed should map to pass")
	}
	if validationVerdict("failed") != "fail" {
		t.Fatalf("failed should map to fail")
	}
	if validationVerdict("") != "fail" {
		t.Fatalf("unknown status should map to fail")
	}
}

func TestValidateDesignKitsReturnsPerKitResults(t *testing.T) {
	handler := NewConnectHandler(templateengine.MustNew(""), nil)
	resp, err := handler.ValidateDesignKits(context.Background(), connect.NewRequest(&lifecyclev1.ValidateDesignKitsRequest{All: true}))
	if err != nil {
		t.Fatalf("ValidateDesignKits: %v", err)
	}
	if resp.Msg.Count == 0 {
		t.Fatalf("expected at least one design kit validated")
	}
	if len(resp.Msg.Results) != int(resp.Msg.Count) {
		t.Fatalf("results = %d, want one per validated kit (%d)", len(resp.Msg.Results), resp.Msg.Count)
	}
	for _, result := range resp.Msg.Results {
		if result.Kit == "" {
			t.Fatalf("result missing kit id: %#v", result)
		}
		if result.Status != "pass" && result.Status != "fail" {
			t.Fatalf("result status = %q, want pass/fail", result.Status)
		}
		if result.Status == "pass" && len(result.Issues) != 0 {
			t.Fatalf("kit %s marked pass but carries issues", result.Kit)
		}
	}
	wantStatus := "pass"
	if resp.Msg.IssuesCount > 0 {
		wantStatus = "fail"
	}
	if resp.Msg.Status != wantStatus {
		t.Fatalf("overall status = %q, want %q for issues_count=%d", resp.Msg.Status, wantStatus, resp.Msg.IssuesCount)
	}
}

func TestDriftReportRequiresScenarioOrAll(t *testing.T) {
	handler := NewConnectHandler(templateengine.MustNew(""), nil)
	_, err := handler.DriftReport(context.Background(), connect.NewRequest(&lifecyclev1.DriftReportRequest{}))
	if err == nil {
		t.Fatalf("expected a usage error when neither a scenario nor --all is provided")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

type lifecycleRunner struct {
	result validationrunner.ValidateResult
}

func (r lifecycleRunner) ValidateTemplate(context.Context, validationrunner.ValidateRequest) (validationrunner.ValidateResult, error) {
	return r.result, nil
}

func (lifecycleRunner) RecordFleetDrift(context.Context) (validationrunner.DriftResult, error) {
	return validationrunner.DriftResult{}, nil
}
