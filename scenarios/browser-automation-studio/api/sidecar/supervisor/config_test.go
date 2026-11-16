package supervisor

import "testing"

func TestDefaultConfigUsesBuiltTypeScriptDriverEntrypoint(t *testing.T) {
	if got := DefaultConfig().DriverScript; got != "dist/server.js" {
		t.Fatalf("DriverScript = %q, want API-supervised TypeScript build entrypoint dist/server.js", got)
	}
}

func TestLoadConfigUsesLifecycleAssignedDriverPort(t *testing.T) {
	t.Setenv("PLAYWRIGHT_DRIVER_PORT", "24485")

	cfg := LoadConfig(nil)
	if cfg.DriverPort != 24485 {
		t.Fatalf("DriverPort = %d, want lifecycle-assigned port 24485", cfg.DriverPort)
	}
}

func TestLoadConfigUsesLifecycleSelectedVariantDriverArtifact(t *testing.T) {
	t.Setenv("PLAYWRIGHT_DRIVER_SCRIPT", "dist/mb3-shadow/server.js")

	cfg := LoadConfig(nil)
	if cfg.DriverScript != "dist/mb3-shadow/server.js" {
		t.Fatalf("DriverScript = %q, want lifecycle-selected variant artifact", cfg.DriverScript)
	}
}

func TestLoadConfigIgnoresInvalidLifecycleDriverPort(t *testing.T) {
	for _, value := range []string{"", "not-a-port", "0", "65536"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("PLAYWRIGHT_DRIVER_PORT", value)
			cfg := LoadConfig(nil)
			if cfg.DriverPort != DefaultConfig().DriverPort {
				t.Fatalf("DriverPort = %d for invalid value %q, want default %d", cfg.DriverPort, value, DefaultConfig().DriverPort)
			}
		})
	}
}
