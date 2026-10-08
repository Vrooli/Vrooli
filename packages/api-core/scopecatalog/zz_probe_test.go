package scopecatalog

import (
	"fmt"
	"testing"
)

func TestProbeOnboardingScopes(t *testing.T) {
	c, err := BuildResilient("/home/matthalloran8/Vrooli")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	fmt.Printf("manifests=%d governed=%d rpc=%d\n", c.ManifestCount, c.GovernedCommandCount, c.RPCScopeCount)
	n := 0
	for _, s := range c.Scopes {
		if s.Scenario == "vrooli-onboarding" {
			n++
			fmt.Printf("  scope=%s cmd=%q service=%q method=%q run_eligible=%v\n", s.Value, s.Command, s.Service, s.Method, s.RunEligible)
		}
	}
	fmt.Printf("vrooli-onboarding scopes=%d\n", n)
	for _, o := range c.OmittedResolutions {
		if o.Scenario == "vrooli-onboarding" {
			fmt.Printf("  omitted service=%q method=%q scope=%s\n", o.Service, o.Method, o.Scope)
		}
	}
}
