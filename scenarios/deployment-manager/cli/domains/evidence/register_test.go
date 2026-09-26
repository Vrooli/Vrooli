package evidence

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsManifestPrimitives(t *testing.T) {
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(app, manifest)
	if err != nil || group.Name != GroupName {
		t.Fatalf("group = %#v, err=%v", group, err)
	}
}

func TestDefaultReportPathUsesInjectedEnvironmentAndHomeDirectory(t *testing.T) {
	getenv := func(key string) string {
		if key == "DEPLOYMENT_MANAGER_REPORT_PATH" {
			return "/configured/report.json"
		}
		return ""
	}
	if got := defaultReportPath(getenv, func() (string, error) { return "/home/tester", nil }); got != "/configured/report.json" {
		t.Fatalf("configured report path = %q", got)
	}
	if got := defaultReportPath(func(string) string { return "" }, func() (string, error) { return "/home/tester", nil }); got != filepath.Join("/home/tester", ".vrooli", "data", "vrooli", "deployment-manager", "deployment", "deployment-report.json") {
		t.Fatalf("home report path = %q", got)
	}
	if got := defaultReportPath(func(string) string { return "" }, func() (string, error) { return "", os.ErrNotExist }); got != filepath.Join(".vrooli", "data", "vrooli", "deployment-manager", "deployment", "deployment-report.json") {
		t.Fatalf("fallback report path = %q", got)
	}
}
