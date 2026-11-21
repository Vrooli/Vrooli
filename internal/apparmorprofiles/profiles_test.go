package apparmorprofiles

import (
	"path/filepath"
	"testing"
)

func TestFixedProfilesOwnDistinctPathsContentAndReplacementRules(t *testing.T) {
	bubblewrap, ok := Lookup(string(BubblewrapUserns))
	if !ok || bubblewrap.Path != "/etc/apparmor.d/vrooli-bubblewrap" || bubblewrap.Replacement != RefuseDifferent || bubblewrap.Content == "" {
		t.Fatalf("bubblewrap profile = %+v, found=%v", bubblewrap, ok)
	}
	workspace, ok := Lookup(string(WorkspaceSandboxUserns))
	if !ok || workspace.Path != "/etc/apparmor.d/vrooli-workspace-sandbox" || workspace.Replacement != ReplaceManaged || workspace.Content == "" {
		t.Fatalf("workspace profile = %+v, found=%v", workspace, ok)
	}
	if bubblewrap.Path == workspace.Path || bubblewrap.Content == workspace.Content {
		t.Fatal("fixed AppArmor profiles unexpectedly share path or content")
	}
	for _, profile := range []Profile{bubblewrap, workspace} {
		if profile.Name == "" || filepath.Base(profile.Path) != profile.Name || len(profile.ParserArgs) != 1 || profile.ParserArgs[0] != ParserReloadFlag || len(profile.Parser) < 2 {
			t.Fatalf("profile parser policy = %+v", profile)
		}
		for _, candidate := range profile.Parser {
			if candidate != "apparmor_parser" && !filepath.IsAbs(candidate) {
				t.Fatalf("parser candidate is not fixed or absolute: %q", candidate)
			}
		}
	}
	if _, ok := Lookup("/etc/passwd"); ok {
		t.Fatal("arbitrary profile ID was accepted")
	}
}
