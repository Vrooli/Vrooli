package phases

import (
	"testing"
	"time"
)

func TestCredentialLeakProofIsRegisteredFromLPBSProviderDescriptor(t *testing.T) {
	catalog := NewDefaultCatalog(time.Minute)
	spec, ok := catalog.Lookup("credential-leak-proof")
	if !ok {
		t.Fatal("credential-leak-proof provider descriptor was not loaded")
	}
	if spec.Delegated == nil || spec.Delegated.ProviderScenario != "landing-page-business-suite" || !spec.Delegated.IncludeExecution {
		t.Fatalf("phase provider = %+v; want LPBS execution provider", spec.Delegated)
	}
	if spec.Runner == nil || spec.Policy.Unavailable != "fail" || !spec.Capabilities.NeedsAPI {
		t.Fatalf("phase runner/policy/capabilities = runner:%t policy:%+v capabilities:%+v", spec.Runner != nil, spec.Policy, spec.Capabilities)
	}
}
