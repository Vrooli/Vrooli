package development

import (
	"errors"
	"testing"
)

func TestParseEffectRejectsRepeatedParameters(t *testing.T) {
	_, err := ParseEffect("filesystem.write[paths=scenarios/example/**,paths=scenarios/other/**]")
	if !errors.Is(err, ErrEffectAmbiguous) {
		t.Fatalf("ParseEffect error = %v, want ErrEffectAmbiguous", err)
	}
}

func TestValidateAllowedEffectsRequiresRegisteredOwnerSemantics(t *testing.T) {
	if err := validateAllowedEffects([]string{"filesystem.write[paths=scenarios/example/**]", "process.test[scope=example]"}); err != nil {
		t.Fatalf("registered effects rejected: %v", err)
	}
	if err := validateAllowedEffects([]string{"owner.operation[scope=example]"}); !errors.Is(err, ErrEffectUnsupported) {
		t.Fatalf("unsupported effect error = %v, want ErrEffectUnsupported", err)
	}
}

func TestValidateAllowedEffectsRejectsAmbiguousScopes(t *testing.T) {
	for _, effect := range []string{
		"filesystem.write[paths=../outside]",
		"process.test[scope=example/other]",
		"network.external[hosts=example.com/other]",
		"billing.reserve[account=fixture,mode=unknown]",
	} {
		if err := validateAllowedEffects([]string{effect}); err == nil {
			t.Errorf("validateAllowedEffects(%q) accepted an ambiguous scope", effect)
		}
	}
}
