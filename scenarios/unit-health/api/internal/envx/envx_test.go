package envx

import (
	"strings"
	"testing"
)

func TestOSReadsProcessEnvironment(t *testing.T) {
	t.Setenv("UNIT_HEALTH_ENVX", "visible")
	var env OS
	if got := env.Getenv("UNIT_HEALTH_ENVX"); got != "visible" {
		t.Fatalf("Getenv = %q", got)
	}
	got, ok := env.LookupEnv("UNIT_HEALTH_ENVX")
	if !ok || got != "visible" {
		t.Fatalf("LookupEnv = %q,%v", got, ok)
	}
	found := false
	for _, value := range env.Environ() {
		if strings.HasPrefix(value, "UNIT_HEALTH_ENVX=") {
			found = true
		}
	}
	if !found {
		t.Fatal("Environ omitted test variable")
	}
}
