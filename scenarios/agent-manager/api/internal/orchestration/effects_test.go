package orchestration

import "testing"

func TestValidateEffectGrantRequiresOwnerScopedParameters(t *testing.T) {
	if err := validateEffectGrant([]string{
		"filesystem.write[paths=scenarios/example/**]",
		"process.test[scope=scenarios/example]",
		"network.external[hosts=api.example.com;localhost]",
	}); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []string{
		"filesystem.write[paths=../outside]",
		"process.test[scope=scenarios/example,extra=x]",
		"owner.operation[scope=example]",
		"credentials.use[account=main]",
	} {
		if err := validateEffectGrant([]string{effect}); err == nil {
			t.Errorf("validateEffectGrant(%q) accepted an invalid grant", effect)
		}
	}
}
