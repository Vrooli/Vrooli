package adoptions

import (
	"strings"
	"testing"
)

func TestValidateTokenMappingInjectiveRejectsCollision(t *testing.T) {
	mapping := map[string]string{
		"app-info":    "wc-accent",
		"app-primary": "wc-accent",
	}
	err := validateTokenMappingInjective("wc", mapping, []string{"app-info", "app-primary"})
	if err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("err = %v, want collision", err)
	}
}
