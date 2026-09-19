package preflight

import (
	"strings"
	"testing"
)

func TestMailDNSApplyRequiresExplicitConfirmation(t *testing.T) {
	err := runMailDNSApply([]string{"manifest.json"})
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("error = %v, want explicit confirmation refusal", err)
	}
}
