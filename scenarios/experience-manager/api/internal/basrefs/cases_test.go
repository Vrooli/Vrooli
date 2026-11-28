package basrefs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCaseFilesReturnsSupportedDepthsSorted(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"bas/cases/root.json",
		"bas/cases/group/child.json",
		"bas/cases/group/nested/grandchild.json",
		"bas/cases/group/nested/deeper/ignored.json",
		"bas/cases/group/not-json.txt",
	} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	got := CaseFiles(root)
	want := []string{
		filepath.Join(root, "bas/cases/group/child.json"),
		filepath.Join(root, "bas/cases/group/nested/grandchild.json"),
		filepath.Join(root, "bas/cases/root.json"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CaseFiles() = %#v, want %#v", got, want)
	}
}
