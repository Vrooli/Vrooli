package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectSymbolsIsFilenameIndependent(t *testing.T) {
	root := t.TempDir()
	writeGo(t, root, "first.go", `package fixture
type Alpha struct{}
const One = 1
func Build() {}
func (Alpha) Run() {}
`)
	writeGo(t, root, "second.go", `package fixture
var Ready bool
func helper() {}
`)
	want := []string{"const One", "func Build", "func helper", "method Alpha Run", "type Alpha", "var Ready"}
	got, err := collectSymbols(root)
	if err != nil {
		t.Fatalf("collectSymbols: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("symbols = %#v, want %#v", got, want)
	}
	os.Rename(filepath.Join(root, "first.go"), filepath.Join(root, "moved.go"))
	got, err = collectSymbols(root)
	if err != nil {
		t.Fatalf("collectSymbols after move: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("symbols changed after file move = %#v, want %#v", got, want)
	}
}

func TestCollectSymbolsReportsRenames(t *testing.T) {
	root := t.TempDir()
	writeGo(t, root, "fixture.go", "package fixture\nfunc Before() {}\n")
	before, err := collectSymbols(root)
	if err != nil {
		t.Fatalf("before: %v", err)
	}
	writeGo(t, root, "fixture.go", "package fixture\nfunc After() {}\n")
	after, err := collectSymbols(root)
	if err != nil {
		t.Fatalf("after: %v", err)
	}
	if reflect.DeepEqual(before, after) {
		t.Fatalf("rename was invisible: before=%#v after=%#v", before, after)
	}
}

func writeGo(t *testing.T, root, name, source string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
