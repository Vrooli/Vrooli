package main

import (
	"reflect"
	"testing"
)

func TestNormalizeArgs(t *testing.T) {
	app := &App{}
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "status becomes overview", in: []string{"status", "--verbose"}, want: []string{"overview", "--verbose"}},
		{name: "dependencies alias", in: []string{"dependencies", "--bottlenecks"}, want: []string{"graph", "dependencies", "--bottlenecks"}},
		{name: "trees default list", in: []string{"trees"}, want: []string{"trees", "list"}},
		{name: "progress list legacy", in: []string{"progress", "--list"}, want: []string{"progress", "list"}},
		{name: "progress status legacy", in: []string{"progress", "--scenario", "graph-studio", "--status", "completed"}, want: []string{"progress", "set-status", "--scenario", "graph-studio", "--status", "completed"}},
		{name: "graph default export", in: []string{"graph"}, want: []string{"graph", "export"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := app.normalizeArgs(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeArgs(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractTreeArgs(t *testing.T) {
	app := &App{}
	got, treeID, treeSlug, err := app.extractTreeArgs([]string{"--tree", "abc", "trees", "list"})
	if err != nil {
		t.Fatalf("extractTreeArgs returned error: %v", err)
	}
	if treeID != "abc" || treeSlug != "" {
		t.Fatalf("unexpected tree selection: id=%q slug=%q", treeID, treeSlug)
	}
	want := []string{"trees", "list"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractTreeArgs args = %v, want %v", got, want)
	}
}
