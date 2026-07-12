package lifecycle

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
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

type lifecycleRunner struct {
	result validationrunner.ValidateResult
}

func (r lifecycleRunner) ValidateTemplate(context.Context, validationrunner.ValidateRequest) (validationrunner.ValidateResult, error) {
	return r.result, nil
}

func (lifecycleRunner) RecordFleetDrift(context.Context) (validationrunner.DriftResult, error) {
	return validationrunner.DriftResult{}, nil
}
