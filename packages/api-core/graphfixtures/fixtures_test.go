package graphfixtures

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListFiltersAndSortsProjectFixtures(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"z-project", "a-project", "no-project"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "z-project", "go.mod"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a-project", "go.mod"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a-project", ExpectedGraphFile), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "loose-file"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := List(root, "go.mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []Fixture{
		{Name: "a-project", Path: filepath.ToSlash(filepath.Join(root, "a-project")), HasExpected: true},
		{Name: "z-project", Path: filepath.ToSlash(filepath.Join(root, "z-project")), HasExpected: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fixtures = %#v, want %#v", got, want)
	}
}

func TestListMissingRootIsEmpty(t *testing.T) {
	got, err := List(filepath.Join(t.TempDir(), "missing"), "go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("fixtures = %#v, want non-nil empty slice", got)
	}
}

func TestListReturnsOtherReadErrors(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := List(file, "go.mod"); err == nil {
		t.Fatal("expected an error for a non-directory root")
	}
}
