package langrecover

import "testing"

const goModBefore = `module example.com/thing

go 1.25.0

require (
	github.com/gorilla/websocket v1.5.1
	github.com/google/uuid v1.6.0
)

require github.com/vrooli/envkit-go v0.0.0 // indirect
`

const goModAfter = `module example.com/thing

go 1.25.0

require (
	github.com/gorilla/websocket v1.5.3
	github.com/google/uuid v1.6.0
)

require (
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/vrooli/envkit-go v0.0.0 // indirect
)
`

// The 2026-09-01 regression this guards: healing app-monitor's missing go.sum
// entry silently moved a direct dependency from v1.5.1 to v1.5.3 via MVS. The
// bump must appear as a Changed delta, distinct from the merely-added indirect.
func TestDiffGoModVersionsSeparatesBumpFromAddition(t *testing.T) {
	deltas := diffGoModVersions(goModBefore, goModAfter)

	byModule := map[string]VersionDelta{}
	for _, d := range deltas {
		byModule[d.Module] = d
	}

	bump, ok := byModule["github.com/gorilla/websocket"]
	if !ok {
		t.Fatalf("expected websocket delta, got %+v", deltas)
	}
	if !bump.Changed() {
		t.Errorf("websocket bump must report Changed(): %+v", bump)
	}
	if bump.From != "v1.5.1" || bump.To != "v1.5.3" {
		t.Errorf("want v1.5.1 -> v1.5.3, got %s -> %s", bump.From, bump.To)
	}

	added, ok := byModule["github.com/santhosh-tekuri/jsonschema/v5"]
	if !ok {
		t.Fatalf("expected jsonschema delta, got %+v", deltas)
	}
	if !added.Added() {
		t.Errorf("new indirect must report Added(), not Changed(): %+v", added)
	}

	if _, ok := byModule["github.com/google/uuid"]; ok {
		t.Errorf("unchanged module must produce no delta")
	}

	changed := ChangedVersionDeltas(deltas)
	if len(changed) != 1 || changed[0].Module != "github.com/gorilla/websocket" {
		t.Errorf("only the real bump should escalate, got %+v", changed)
	}
}

func TestDiffGoModVersionsHandlesSingleLineRequire(t *testing.T) {
	before := "module m\n\nrequire example.com/a v1.0.0\n"
	after := "module m\n\nrequire example.com/a v1.1.0\n"

	deltas := diffGoModVersions(before, after)
	if len(deltas) != 1 || !deltas[0].Changed() {
		t.Fatalf("want one changed delta, got %+v", deltas)
	}
	if deltas[0].From != "v1.0.0" || deltas[0].To != "v1.1.0" {
		t.Errorf("want v1.0.0 -> v1.1.0, got %+v", deltas[0])
	}
}

func TestDiffGoModVersionsIgnoresNonRequireLines(t *testing.T) {
	// replace/module/go lines must never be read as requirements.
	body := `module example.com/thing

go 1.25.0

replace example.com/other => ../other

require example.com/a v1.0.0
`
	requires := parseGoModRequires(body)
	if len(requires) != 1 {
		t.Fatalf("want exactly one requirement, got %+v", requires)
	}
	if requires["example.com/a"] != "v1.0.0" {
		t.Errorf("unexpected parse: %+v", requires)
	}
}

func TestDiffGoModVersionsNoChange(t *testing.T) {
	if deltas := diffGoModVersions(goModBefore, goModBefore); len(deltas) != 0 {
		t.Errorf("identical files must produce no deltas, got %+v", deltas)
	}
}

func TestFormatVersionDeltas(t *testing.T) {
	if got := FormatVersionDeltas(nil); got != "" {
		t.Errorf("empty deltas must format empty, got %q", got)
	}
	got := FormatVersionDeltas([]VersionDelta{
		{Module: "m/a", From: "v1", To: "v2"},
		{Module: "m/b", To: "v3"},
		{Module: "m/c", From: "v4"},
	})
	want := "m/a v1 -> v2, m/b +v3, m/c -v4"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
