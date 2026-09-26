package pathhygiene

import (
	"os"
	"strings"
	"testing"
)

func joinPath(dirs ...string) string {
	return strings.Join(dirs, string(os.PathListSeparator))
}

func TestDuplicateEntriesRanksWorstFirst(t *testing.T) {
	dups := DuplicateEntries(joinPath("/a", "/b", "/a", "/c", "/b", "/a"))
	if len(dups) != 2 {
		t.Fatalf("got %d duplicate dirs, want 2: %+v", len(dups), dups)
	}
	if dups[0].Dir != "/a" || dups[0].Count != 3 {
		t.Errorf("worst = %+v, want /a x3", dups[0])
	}
	if dups[1].Dir != "/b" || dups[1].Count != 2 {
		t.Errorf("second = %+v, want /b x2", dups[1])
	}
}

func TestEntryCountsIgnoreEmptySegments(t *testing.T) {
	p := joinPath("/a", "", "/b", "/a")
	if got := EntryCount(p); got != 3 {
		t.Errorf("EntryCount = %d, want 3", got)
	}
	if got := UniqueEntryCount(p); got != 2 {
		t.Errorf("UniqueEntryCount = %d, want 2", got)
	}
}

// Only directories AHEAD of the canonical one can shadow it. A copy behind
// it is harmless and must not be reported, or the note would cry wolf on
// every host that has ever run `go install`.
func TestShadowingBinariesOnlyLooksAheadOfCanonical(t *testing.T) {
	orig := isExecutableFileFn
	isExecutableFileFn = func(path string) bool { return strings.HasSuffix(path, "/vrooli") }
	t.Cleanup(func() { isExecutableFileFn = orig })

	t.Run("ahead is reported", func(t *testing.T) {
		got := ShadowingBinaries(joinPath("/early", "/canon", "/late"), "/canon", "vrooli")
		if len(got) != 1 || got[0] != "/early/vrooli" {
			t.Errorf("got %v, want [/early/vrooli]", got)
		}
	})
	t.Run("behind is ignored", func(t *testing.T) {
		if got := ShadowingBinaries(joinPath("/canon", "/late"), "/canon", "vrooli"); len(got) != 0 {
			t.Errorf("got %v, want none — /late is behind the canonical entry", got)
		}
	})
	t.Run("canonical absent reports every hit", func(t *testing.T) {
		got := ShadowingBinaries(joinPath("/early", "/other"), "/canon", "vrooli")
		if len(got) != 2 {
			t.Errorf("got %v, want both dirs reported when canonical is not on PATH", got)
		}
	})
}
