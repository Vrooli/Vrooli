package packagecli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/packagegov"
)

func TestParseRefreshRequestCapturesTargetAndNoRestart(t *testing.T) {
	req, err := ParseRefreshRequest([]string{"api-core", "alpha", "--no-restart"})
	if err != nil {
		t.Fatalf("ParseRefreshRequest: %v", err)
	}
	if req.Name != "api-core" || req.Target != "alpha" || !req.NoRestart {
		t.Fatalf("req = %+v", req)
	}
}

func TestParseValidateRequestDefaultsToAll(t *testing.T) {
	req, err := ParseValidateRequest(nil)
	if err != nil {
		t.Fatalf("ParseValidateRequest: %v", err)
	}
	if !req.All || req.Name != "" {
		t.Fatalf("req = %+v", req)
	}
}

func TestParseRunRequestAllowsAllForTestLifecycle(t *testing.T) {
	req, err := ParseRunRequest("test", nil)
	if err != nil {
		t.Fatalf("ParseRunRequest: %v", err)
	}
	if req.Name != "" || req.Action != "test" {
		t.Fatalf("req = %+v", req)
	}
}

func TestParseAuditRequestRejectsUnknownFlag(t *testing.T) {
	if _, err := ParseAuditRequest([]string{"--bogus"}); err == nil || !strings.Contains(err.Error(), "unknown option for package audit") {
		t.Fatalf("ParseAuditRequest error = %v", err)
	}
}

func TestRenderCommandHelpIncludesRefresh(t *testing.T) {
	var stdout bytes.Buffer
	RenderCommandHelp(&stdout)
	if !strings.Contains(stdout.String(), "refresh") || !strings.Contains(stdout.String(), "propagate to affected consumers") {
		t.Fatalf("help = %q", stdout.String())
	}
}

func TestRenderRefreshHumanIncludesMergedClasses(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderRefresh(&stdout, cliout.FormatHuman, RefreshResponse{
		PackageName: "proto",
		Items: []RefreshItem{{
			Consumer: "desktop",
			Class:    packagegov.ConsumerClass("scenario_api"),
			Classes: []packagegov.ConsumerClass{
				packagegov.ConsumerClass("scenario_api"),
				packagegov.ConsumerClass("scenario_ui"),
			},
			Action: "setup_scenario",
			Status: "setup_only",
		}},
	})
	if err != nil {
		t.Fatalf("RenderRefresh: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "scenario_api,scenario_ui") || !strings.Contains(got, "setup_only") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRenderListJSONUsesSuccessEnvelope(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderList(&stdout, cliout.FormatJSON, ListResponse{
		Packages: []packagegov.Package{{Name: "alpha"}},
	})
	if err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, `"success": true`) || !strings.Contains(got, `"packages"`) {
		t.Fatalf("stdout = %q", got)
	}
}
