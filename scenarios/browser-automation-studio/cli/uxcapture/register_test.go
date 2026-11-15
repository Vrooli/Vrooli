package uxcapture

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func TestValidateWaitSelectorIsScenarioOnlyAndBounded(t *testing.T) {
	if got, err := validateWaitSelector("web-console", ` [data-testid="pane-grid"] `); err != nil || got != `[data-testid="pane-grid"]` {
		t.Fatalf("validate scenario selector = %q, %v", got, err)
	}
	if _, err := validateWaitSelector("", `[data-testid="pane-grid"]`); err == nil {
		t.Fatal("expected a public capture to reject a scenario wait selector")
	}
	if _, err := validateWaitSelector("web-console", strings.Repeat("x", 513)); err == nil {
		t.Fatal("expected an overlong selector to be rejected")
	}
	if got, err := validateWaitSelector("web-console", "   "); err != nil || got != "" {
		t.Fatalf("blank selector = %q, %v", got, err)
	}
}

func TestBuildScenarioRequestCarriesWaitSelector(t *testing.T) {
	got := buildScenarioRequest("web-console", "/", nil, 1280, 720, "light", `[data-testid="pane-grid"]`, "", "")
	want := map[string]any{
		"scenario":      "web-console",
		"route":         "/",
		"steps":         []string(nil),
		"width":         1280,
		"height":        720,
		"theme":         "light",
		"wait_selector": `[data-testid="pane-grid"]`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scenario request = %#v, want %#v", got, want)
	}
}

func TestValidateRegionSelectorRequiresComponentTestScenarioAndIsBounded(t *testing.T) {
	if got, err := validateRegionSelector("react-component-library", "componenttests", `[data-preview-capture-boundary]`); err != nil || got != `[data-preview-capture-boundary]` {
		t.Fatalf("validate region selector = %q, %v", got, err)
	}
	if _, err := validateRegionSelector("react-component-library", "", `[data-preview-capture-boundary]`); err == nil {
		t.Fatal("expected an evidence profile error")
	}
	if _, err := validateRegionSelector("", "componenttests", `[data-preview-capture-boundary]`); err == nil {
		t.Fatal("expected a scenario-only error")
	}
	if _, err := validateRegionSelector("react-component-library", "componenttests", strings.Repeat("x", 257)); err == nil {
		t.Fatal("expected an overlong selector to be rejected")
	}
}

func TestBuildScenarioRequestCarriesComponentEvidenceProfile(t *testing.T) {
	got := buildScenarioRequest("react-component-library", "/preview/button/harness.html?version=1.2.3", nil, 1280, 900, "light", "", "componenttests", `[data-preview-capture-boundary]`)
	if got["evidence_profile"] != "componenttests" || got["region_selector"] != `[data-preview-capture-boundary]` {
		t.Fatalf("component evidence request = %#v", got)
	}
}

// Regression: the command passed /api/v1/ux-capture/... to api.Do, which adds
// the scenario's /api/v1 prefix itself, so every capture returned HTTP 404.
func TestUXCaptureCommandCallsTheVersionedEndpointOnce(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"bundle_dir":"/x","manifest":{}}`))
	}))
	t.Cleanup(server.Close)
	t.Setenv("BAS_UXCAPTURE_CLI_TEST_CONFIG", t.TempDir())
	app, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:             "bas-uxcapture-cli-test",
		DefaultAPIBase:   server.URL,
		APIPrefix:        "/api/v1",
		AllowAnonymous:   true,
		ConfigDirEnvVars: []string{"BAS_UXCAPTURE_CLI_TEST_CONFIG"},
	})
	if err != nil {
		t.Fatalf("new CLI app: %v", err)
	}
	command := Commands(&appctx.Context{Core: app}).Commands[0]
	var stdout bytes.Buffer
	rc, err := cliapp.NewTestRunContextFromArgs(command.Args, []string{"--scenario", "image-tools", "--width", "390", "--height", "844"}, app, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}
	if err := command.RunCtx(rc); err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/api/v1/ux-capture/scenario" {
		t.Fatalf("request path = %q, want /api/v1/ux-capture/scenario", gotPath)
	}
}
