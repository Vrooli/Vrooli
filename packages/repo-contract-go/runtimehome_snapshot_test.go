package repocontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeHomeSnapshotEntryResolvesWithoutPinOrSource(t *testing.T) {
	isolatedRuntimeHomePin(t)
	c := validContract(t)
	data, err := c.RuntimeHomeSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	oldRead := readFile
	readFile = func(string) ([]byte, error) { t.Fatal("snapshot entry read source"); return nil, nil }
	t.Cleanup(func() { readFile = oldRead })
	home := t.TempDir()
	got, err := RuntimeHomeSnapshotEntry(data, home, HomeKeyState)
	want, wantErr := c.RuntimeHomeEntry(home, HomeKeyState)
	if err != nil || wantErr != nil || got.AbsPath != want.AbsPath {
		t.Fatalf("entry differs: %+v %v", got, err)
	}
	if pinnedRuntimeHome.Load() != nil {
		t.Fatal("read-only entry resolution pinned process authority")
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		t.Fatal("snapshot resolution wrote runtime state")
	}
	if _, err := RuntimeHomeSnapshotEntry([]byte("{}"), home, HomeKeyState); err == nil {
		t.Fatal("invalid snapshot accepted")
	}
}

func isolatedRuntimeHomePin(t *testing.T) {
	t.Helper()
	prior := pinnedRuntimeHome.Swap(nil)
	t.Cleanup(func() { pinnedRuntimeHome.Store(prior) })
}

func TestPinnedRuntimeHomeSnapshotWorksWithoutSourceAndCreatesNoFiles(t *testing.T) {
	isolatedRuntimeHomePin(t)
	home := t.TempDir()
	c := validContract(t)
	data, err := c.RuntimeHomeSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := PinRuntimeHomeSnapshot(data); err != nil {
		t.Fatal(err)
	}
	oldRead := readFile
	readFile = func(string) ([]byte, error) {
		t.Fatal("pinned runtime-home path attempted source I/O")
		return nil, nil
	}
	t.Cleanup(func() { readFile = oldRead })
	for key := range c.doc.RuntimeHome.Entries {
		want, err := c.RuntimeHomeEntry(home, key)
		if err != nil {
			t.Fatal(err)
		}
		got, err := RuntimeHomeEntryPath(home, key)
		if err != nil || got != want.AbsPath {
			t.Fatalf("snapshot path for %s differs: %s %v", key, got, err)
		}
	}
	batch, err := RuntimeHomeEntries(home)
	if err != nil || len(batch) != len(c.doc.RuntimeHome.Entries) {
		t.Fatalf("pinned batch requires source: %+v %v", batch, err)
	}
	for _, entry := range batch {
		want, err := c.RuntimeHomeEntry(home, entry.Key)
		if err != nil || entry.AbsPath != want.AbsPath {
			t.Fatalf("pinned batch changed entry %s", entry.Key)
		}
	}
	root, err := VrooliUserRoot(home)
	if err != nil || root != filepath.Join(home, ".vrooli") {
		t.Fatalf("runtime root: %s %v", root, err)
	}
	if err := PinRuntimeHomeSnapshot(data); err != nil {
		t.Fatalf("identical snapshot pin not idempotent: %v", err)
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		t.Fatalf("snapshot/path resolution initialized user state: %v %v", entries, err)
	}
}

func TestPinnedRuntimeHomeSnapshotCannotBeReplacedOrMutatedByCaller(t *testing.T) {
	isolatedRuntimeHomePin(t)
	c := validContract(t)
	data, err := c.RuntimeHomeSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := PinRuntimeHomeSnapshot(data); err != nil {
		t.Fatal(err)
	}
	before, err := RuntimeHomeEntryPath(t.TempDir(), HomeKeyRuntimeDB)
	if err != nil {
		t.Fatal(err)
	}
	for i := range data {
		data[i] = 'x'
	}
	modified := validContract(t)
	entry := modified.doc.RuntimeHome.Entries[HomeKeyRuntimeDB]
	entry.Path = "state/other.db"
	modified.doc.RuntimeHome.Entries[HomeKeyRuntimeDB] = entry
	other, err := modified.RuntimeHomeSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := PinRuntimeHomeSnapshot(other); err == nil {
		t.Fatal("process runtime-home authority was replaced")
	}
	got := pinnedRuntimeHome.Load()
	path, err := got.RuntimeHomeEntry(filepath.Dir(filepath.Dir(filepath.Dir(before))), HomeKeyRuntimeDB)
	if err != nil || path.RelPath != "state/runtime.db" {
		t.Fatalf("pinned snapshot shared mutable input: %+v %v", path, err)
	}
}

func TestRuntimeHomeSnapshotRejectsAmbiguousInvalidAndUnboundedInput(t *testing.T) {
	c := validContract(t)
	canonical, err := c.RuntimeHomeSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"empty", "unknown", "duplicate", "case_alias", "nested_alias", "trailing", "whitespace", "oversized", "traversal", "unsupported_version"} {
		t.Run(kind, func(t *testing.T) {
			isolatedRuntimeHomePin(t)
			data := string(canonical)
			switch kind {
			case "empty":
				data = ""
			case "unknown":
				data = strings.Replace(data, `"version":1`, `"version":1,"unknown":true`, 1)
			case "duplicate":
				data = strings.Replace(data, `"version":1`, `"version":1,"version":1`, 1)
			case "case_alias":
				data = strings.Replace(data, `"version":1`, `"Version":1`, 1)
			case "nested_alias":
				data = strings.Replace(data, `"dir_name":`, `"DIR_NAME":`, 1)
			case "trailing":
				data += "{}"
			case "whitespace":
				data += "\n"
			case "oversized":
				data = strings.Repeat(" ", maxRuntimeHomeSnapshotBytes+1)
			case "unsupported_version":
				data = strings.Replace(data, `"version":1`, `"version":2`, 1)
			case "traversal":
				var s runtimeHomeSnapshot
				if err := json.Unmarshal(canonical, &s); err != nil {
					t.Fatal(err)
				}
				entry := s.RuntimeHome.Entries[HomeKeyRuntimeDB]
				entry.Path = "../escape.db"
				s.RuntimeHome.Entries[HomeKeyRuntimeDB] = entry
				b, _ := json.Marshal(s)
				data = string(b)
			}
			if err := PinRuntimeHomeSnapshot([]byte(data)); err == nil {
				t.Fatal("invalid compiled snapshot accepted")
			}
			if pinnedRuntimeHome.Load() != nil {
				t.Fatal("invalid snapshot changed authority")
			}
		})
	}
}
