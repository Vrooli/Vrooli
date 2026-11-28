package phases

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReactConsumerContractsPhaseIsRegisteredFromItsDescriptor(t *testing.T) {
	catalog := NewDefaultCatalog(time.Minute)
	spec, ok := catalog.Lookup("consumer-contract")
	if !ok {
		t.Fatal("consumer-contract descriptor was not loaded into the effective Test Genie catalog")
	}
	if spec.Delegated == nil || spec.Delegated.ProviderScenario != "react-component-library" {
		t.Fatalf("phase provider = %+v, want react-component-library", spec.Delegated)
	}
	if spec.Runner == nil || spec.RuntimeClass != "static" || !spec.Capabilities.NeedsAPI {
		t.Fatalf("phase registration must expose its static provider runner and live validation API: %+v", spec)
	}

	const expectedDoc = "scenarios/test-genie/docs/phases/consumer-contract/README.md"
	if spec.Doc != expectedDoc {
		t.Fatalf("phase doc = %q, want %q", spec.Doc, expectedDoc)
	}
	repoRoot := filepath.Dir(filepath.Dir(scenarioRoot(t)))
	doc, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(spec.Doc)))
	if err != nil {
		t.Fatalf("read phase remediation doc %s: %v", spec.Doc, err)
	}
	for _, heading := range []string{
		"## North Star",
		"## The rungs and their gates",
		"## What each finding means",
		"## The canonical fix",
		"## How to verify",
	} {
		if !strings.Contains(string(doc), heading) {
			t.Errorf("phase remediation doc %s is missing required heading %q", spec.Doc, heading)
		}
	}
}
