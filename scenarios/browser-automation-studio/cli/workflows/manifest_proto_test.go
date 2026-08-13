package workflows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

// TestWorkflowsManifestCoversWorkflowsService asserts every RPC on
// WorkflowsService has a matching manifest command binding.
//
// Per-domain parity test added in Phase 7 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc).
func TestWorkflowsManifestCoversWorkflowsService(t *testing.T) {
	manifest := readBASManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, apiv1.File_browser_automation_studio_v1_api_service_proto, "WorkflowsService")
}

func TestWorkflowsAdhocBindingNormalizesShortExecutionMode(t *testing.T) {
	svc := apiv1.File_browser_automation_studio_v1_api_service_proto.Services().ByName("WorkflowsService")
	normalize := protoBindingOptions(string(svc.Name())).Normalize["WorkflowsService.ExecuteAdhocWorkflow"]
	if normalize == nil {
		t.Fatal("adhoc workflow binding has no compatibility normalizer")
	}
	got, err := normalize([]byte(`{"metadata":{"execution_mode":"mutating"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "EXECUTION_MODE_MUTATING") {
		t.Fatalf("normalized flow = %s", got)
	}
}

func TestNormalizeAdhocFlowFileExtractsOnboardingCase(t *testing.T) {
	got, err := normalizeAdhocFlowFile([]byte(`{
		"metadata": {"execution_mode": "mutating"},
		"description": "case wrapper metadata",
		"flow_definition": {"metadata": {"execution_mode": "mutating"}}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "case wrapper metadata") {
		t.Fatalf("case wrapper was not removed: %s", got)
	}
	if !strings.Contains(string(got), "EXECUTION_MODE_MUTATING") {
		t.Fatalf("normalized flow = %s", got)
	}
}

func TestFindAdhocProjectRootFindsScenarioSelectorManifest(t *testing.T) {
	caseFile := filepath.Join("..", "..", "..", "vrooli-onboarding", "bas", "cases", "experience", "first-run-to-applied-install.json")
	root := findAdhocProjectRoot(caseFile)
	if root == "" || !strings.HasSuffix(filepath.ToSlash(root), "/scenarios/vrooli-onboarding") {
		t.Fatalf("findAdhocProjectRoot(%q) = %q", caseFile, root)
	}
}

func readBASManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
