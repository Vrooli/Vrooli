package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithinHandlesEqualDescendantAndPrefixSibling(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name   string
		target string
		want   bool
	}{
		{"equal", root, true},
		{"descendant", filepath.Join(root, "nested", "file"), true},
		{"prefix sibling", root + "-other", false},
		{"parent", filepath.Dir(root), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Within(root, tc.target)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("Within(%q, %q) = %t, want %t", root, tc.target, got, tc.want)
			}
		})
	}
}

func TestWithinRejectsEmptyAndResolvedSymlinkEscape(t *testing.T) {
	if _, err := Within("", "/tmp"); err == nil {
		t.Fatal("expected empty root error")
	}
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "file"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if got, err := WithinResolved(root, filepath.Join(link, "file")); err == nil && got {
		t.Fatal("resolved symlink escape was accepted")
	}
}

func TestExistsDistinguishesAbsence(t *testing.T) {
	got, err := Exists(filepath.Join(t.TempDir(), "missing"))
	if err != nil || got {
		t.Fatalf("missing Exists = %t, %v", got, err)
	}
	if _, err := Exists(""); err == nil {
		t.Fatal("expected empty path error")
	}
}

func TestJSONRoundTripRejectsEmptyAndTrailingValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "value.json")
	data, err := MarshalJSON(map[string]int{"answer": 42})
	if err != nil {
		t.Fatal(err)
	}
	if string(data[len(data)-1:]) != "\n" {
		t.Fatalf("JSON has no trailing newline: %q", data)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	var got map[string]int
	if err := ReadJSON(path, &got); err != nil || got["answer"] != 42 {
		t.Fatalf("ReadJSON = %#v, %v", got, err)
	}
	if err := os.WriteFile(path, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ReadJSON(path, &got); err == nil {
		t.Fatal("expected empty JSON error")
	}
	if err := os.WriteFile(path, []byte("{} {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	trailingErr := ReadJSON(path, &got)
	if trailingErr == nil {
		t.Fatal("expected trailing JSON value error")
	}
}
