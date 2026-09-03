package enroll

import (
	"strings"
	"testing"
	"time"
)

func TestRenderFirstBootCarriesPublicKeyAndExpiryOnly(t *testing.T) {
	config, err := RenderFirstBoot("ssh-ed25519 AAAATEST", time.Date(2026, 9, 3, 22, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(config, "ssh-ed25519 AAAATEST") || !strings.Contains(config, "2026-09-03 22:00:00 UTC") || strings.Contains(config, "PRIVATE KEY") {
		t.Fatalf("config = %q", config)
	}
}
