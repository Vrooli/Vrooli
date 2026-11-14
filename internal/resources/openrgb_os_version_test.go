package resources

import (
	"testing"
)

func TestOpenRGBNativeMacBaselineRefusesUnknownAndOlderVersions(t *testing.T) {
	p, _ := openRGBReleaseProfileFor("darwin", "amd64")
	for _, value := range []string{"14.9", "10.15.7", "unknown", "15.beta", "15.0.9999", "", "+15.0"} {
		if openRGBDarwinVersionAtLeast(value, p.MinimumDarwinVersion) {
			t.Fatalf("unqualified OS version admitted: %q", value)
		}
	}
	for _, value := range []string{"15.0", "15.7.1", "26.1"} {
		if !openRGBDarwinVersionAtLeast(value, p.MinimumDarwinVersion) {
			t.Fatalf("supported OS metadata refused: %q", value)
		}
	}
}
