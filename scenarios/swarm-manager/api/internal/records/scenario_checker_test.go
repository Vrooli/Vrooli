package records

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryScenarioChecker(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	packages := filepath.Join(root, "packages")
	for _, dir := range []string{
		filepath.Join(scenarios, "web-console"),
		filepath.Join(scenarios, "audio-tools"),
		filepath.Join(packages, "cli-core"),
		filepath.Join(scenarios, ".hidden"),
	} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	check := NewDirectoryScenarioChecker([]string{scenarios, packages}, "vrooli")

	cases := []struct {
		slug        string
		wantKnown   bool
		wantNearest string
	}{
		{"web-console", true, ""},
		{"Web-Console", true, ""}, // case-insensitive
		{"cli-core", true, ""},    // from second root
		{"vrooli", true, ""},      // extra slug
		{"web-consol", false, "web-console"},
		{"audio-tool", false, "audio-tools"},
		{"totally-unknown-thing", false, ""}, // too far for a suggestion
		{".hidden", false, ""},               // hidden dirs are not slugs
		{"", true, ""},                       // emptiness is the service's error
	}
	for _, tc := range cases {
		known, nearest := check(tc.slug)
		if known != tc.wantKnown || nearest != tc.wantNearest {
			t.Errorf("check(%q) = (%v, %q), want (%v, %q)",
				tc.slug, known, nearest, tc.wantKnown, tc.wantNearest)
		}
	}
}

func TestDirectoryScenarioCheckerMissingRoot(t *testing.T) {
	check := NewDirectoryScenarioChecker([]string{"/nonexistent/path"}, "vrooli")
	if known, _ := check("vrooli"); !known {
		t.Error("extra slug should stay known when roots are unreadable")
	}
	if known, _ := check("anything"); known {
		t.Error("unknown slug reported known")
	}
}
