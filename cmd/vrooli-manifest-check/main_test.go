package main

import (
	"slices"
	"testing"
)

func TestManifestDriftNamesBothDirections(t *testing.T) {
	got := manifestDrift(
		[]string{"credentials list", "tree extra"},
		[]string{"credentials list", "manifest extra"},
	)
	want := []string{"manifest-only: manifest extra", "tree-only: tree extra"}
	if !slices.Equal(got, want) {
		t.Fatalf("manifestDrift() = %#v, want %#v", got, want)
	}
}
