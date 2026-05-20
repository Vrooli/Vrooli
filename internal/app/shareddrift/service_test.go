package shareddrift

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestParseLocalReplacesSingleLine(t *testing.T) {
	mod := `module foo

go 1.25

replace github.com/vrooli/api-core => ../../../packages/api-core
replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
replace github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go
replace github.com/vrooli/vrooli => ../../..
replace example.com/remote v1.2.3 => example.com/remote-fork v1.2.4
`
	got := parseLocalReplaces(mod)
	want := []replaceTarget{
		{Module: "github.com/vrooli/api-core", Local: "../../../packages/api-core"},
		{Module: "github.com/vrooli/vrooli/packages/proto", Local: "../../../packages/proto"},
		{Module: "github.com/vrooli/repo-contract-go", Local: "../../../packages/repo-contract-go"},
		{Module: "github.com/vrooli/vrooli", Local: "../../.."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseLocalReplaces single-line:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestParseLocalReplacesBlock(t *testing.T) {
	mod := `module foo

replace (
    github.com/vrooli/api-core => ../../../packages/api-core
    github.com/vrooli/repo-contract-go => ../../../packages/repo-contract-go
    example.com/remote v1.0.0 => example.com/remote-fork v1.0.1
)
`
	got := parseLocalReplaces(mod)
	want := []replaceTarget{
		{Module: "github.com/vrooli/api-core", Local: "../../../packages/api-core"},
		{Module: "github.com/vrooli/repo-contract-go", Local: "../../../packages/repo-contract-go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseLocalReplaces block:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestFilterByTouchedRootGoModFansOut(t *testing.T) {
	root := "/repo"
	scenarios := []scenarioInfo{
		{
			Path:   "scenarios/foo",
			APIDir: filepath.Join(root, "scenarios", "foo", "api"),
			Replaces: []replaceTarget{
				{Module: "github.com/vrooli/vrooli", Local: "../../.."},
			},
		},
	}
	got := filterByTouched(scenarios, []string{"go.mod"}, root)
	if len(got) != 1 {
		t.Fatalf("root go.mod change must fan out to all scenarios, got %d", len(got))
	}
}

func TestFilterByTouchedPackagePrefixSelectsOnlyDependents(t *testing.T) {
	root := "/repo"
	scenarios := []scenarioInfo{
		{
			Path:   "scenarios/uses-api-core",
			APIDir: filepath.Join(root, "scenarios", "uses-api-core", "api"),
			Replaces: []replaceTarget{
				{Module: "github.com/vrooli/api-core", Local: "../../../packages/api-core"},
			},
		},
		{
			Path:   "scenarios/uses-proto-only",
			APIDir: filepath.Join(root, "scenarios", "uses-proto-only", "api"),
			Replaces: []replaceTarget{
				{Module: "github.com/vrooli/vrooli/packages/proto", Local: "../../../packages/proto"},
			},
		},
	}
	got := filterByTouched(scenarios, []string{"packages/api-core/foo.go"}, root)
	if len(got) != 1 || got[0].Path != "scenarios/uses-api-core" {
		paths := []string{}
		for _, g := range got {
			paths = append(paths, g.Path)
		}
		sort.Strings(paths)
		t.Fatalf("expected only api-core dependent, got %v", paths)
	}
}

func TestFilterByTouchedEmptyTouchedReturnsNothing(t *testing.T) {
	root := "/repo"
	scenarios := []scenarioInfo{
		{
			Path:   "scenarios/foo",
			APIDir: filepath.Join(root, "scenarios", "foo", "api"),
			Replaces: []replaceTarget{
				{Module: "github.com/vrooli/api-core", Local: "../../../packages/api-core"},
			},
		},
	}
	if got := filterByTouched(scenarios, nil, root); len(got) != 0 {
		t.Fatalf("expected no scenarios for empty touched, got %d", len(got))
	}
}

func TestFilterByTouchedIgnoresUnrelatedPaths(t *testing.T) {
	root := "/repo"
	scenarios := []scenarioInfo{
		{
			Path:   "scenarios/foo",
			APIDir: filepath.Join(root, "scenarios", "foo", "api"),
			Replaces: []replaceTarget{
				{Module: "github.com/vrooli/api-core", Local: "../../../packages/api-core"},
			},
		},
	}
	got := filterByTouched(scenarios, []string{"docs/readme.md", "scenarios/foo/api/main.go"}, root)
	if len(got) != 0 {
		t.Fatalf("expected no scenarios for unrelated paths, got %d", len(got))
	}
}
