package envx

import "testing"

func TestGetReadsNamedEnvironmentValue(t *testing.T) {
	const key = "AUDIO_TOOLS_ENV_SEAM_TEST"
	t.Setenv(key, "configured")

	if got := Get(key); got != "configured" {
		t.Fatalf("Get(%q) = %q, want %q", key, got, "configured")
	}
}
