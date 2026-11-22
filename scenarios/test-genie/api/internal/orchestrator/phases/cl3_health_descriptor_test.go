package phases

import (
	"testing"
	"time"
)

func TestCloudProductionHealthPhaseIsRegisteredFromProviderDescriptor(t *testing.T) {
	catalog := NewDefaultCatalog(time.Minute)
	spec, ok := catalog.Lookup("cloud-production-health")
	if !ok {
		t.Fatal("cloud-production-health provider descriptor was not loaded")
	}
	if spec.Delegated == nil || spec.Delegated.ProviderScenario != "landing-page-business-suite" || !spec.Delegated.IncludeExecution {
		t.Fatalf("phase provider = %+v; want LPBS execution provider", spec.Delegated)
	}
	if spec.Runner == nil || spec.Policy.Unavailable != "fail" || !spec.Capabilities.NeedsAPI {
		t.Fatalf("phase runner/policy/capabilities = runner:%t policy:%+v capabilities:%+v", spec.Runner != nil, spec.Policy, spec.Capabilities)
	}
	if spec.Doc != "scenarios/test-genie/docs/phases/cloud-production-health/README.md" {
		t.Fatalf("phase doc = %q", spec.Doc)
	}
}
