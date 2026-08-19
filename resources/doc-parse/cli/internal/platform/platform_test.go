package platform

import "testing"

func TestCheckSupportedDesktopPairs(t *testing.T) {
	for _, pair := range [][2]string{{"linux", "amd64"}, {"linux", "arm64"}, {"macos", "amd64"}, {"macos", "arm64"}, {"windows", "amd64"}} {
		if err := Check(pair[0], pair[1]); err != nil {
			t.Fatalf("Check(%q, %q): %v", pair[0], pair[1], err)
		}
	}
}

func TestCheckNamesUnsupportedTarget(t *testing.T) {
	if err := Check("freebsd", "amd64"); err == nil {
		t.Fatal("expected unsupported operating system")
	}
	if err := Check("windows", "arm64"); err == nil {
		t.Fatal("expected unsupported Windows architecture")
	}
}
