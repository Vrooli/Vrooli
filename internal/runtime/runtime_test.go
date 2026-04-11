package runtime

import (
	goruntime "runtime"
	"testing"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=1 | LAST: 2026-04-10

func TestCurrentMatchesRuntimeGOOS(t *testing.T) {
	host := Current()
	if host.OS != goruntime.GOOS {
		t.Fatalf("host.OS = %q, want %q", host.OS, goruntime.GOOS)
	}
}

func TestValidateSupportMatchesCapabilityFlags(t *testing.T) {
	host := Current()
	if host.SupportsSetup {
		if err := host.ValidateSetup(); err != nil {
			t.Fatalf("ValidateSetup: %v", err)
		}
	} else if err := host.ValidateSetup(); err == nil {
		t.Fatal("ValidateSetup succeeded on unsupported host")
	}

	if host.SupportsDevelop {
		if err := host.ValidateDevelop(); err != nil {
			t.Fatalf("ValidateDevelop: %v", err)
		}
	} else if err := host.ValidateDevelop(); err == nil {
		t.Fatal("ValidateDevelop succeeded on unsupported host")
	}
}
