package spacecli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/spacedoc"
)

const doc = `# Guide Space

## This Space

| | |
|---|---|
| Projection | Guide |
| Owner | ` + "`prompt-manager`" + ` (owns the skill graph) |
| Denominator confidence | ` + "`SKETCH`" + ` — first cut. |

## Coverage Grid

| # | SWE task | Guiding skill(s) | Status | Gate | Notes |
|---|---|---|---|---|---|
| G1 | Explore a codebase | ` + "`explore`" + ` | COVERED | — | helps. |
`

func writeDoc(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "guide-space.md")
	if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSpaceCommandJSON(t *testing.T) {
	var out bytes.Buffer
	grp := CommandGroup(Config{
		Owner:      "prompt-manager",
		Projection: spacedoc.ProjectionGuide,
		DocPath:    writeDoc(t),
		Out:        &out,
	})
	if grp.Commands[0].Name != "space" {
		t.Fatalf("command name = %q", grp.Commands[0].Name)
	}
	if err := grp.Commands[0].Run([]string{"--projection", "guide", "--json"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	var def spacedoc.SpaceDefinition
	if err := json.Unmarshal(out.Bytes(), &def); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out.String())
	}
	if def.Projection != spacedoc.ProjectionGuide || def.Owner != "prompt-manager" {
		t.Errorf("def = %+v", def)
	}
	if len(def.Cells) != 1 || def.Cells[0].ID != "G1" {
		t.Errorf("cells = %+v", def.Cells)
	}
}

func TestSpaceCommandText(t *testing.T) {
	var out bytes.Buffer
	grp := CommandGroup(Config{
		Owner:      "prompt-manager",
		Projection: spacedoc.ProjectionGuide,
		DocPath:    writeDoc(t),
		Out:        &out,
	})
	if err := grp.Commands[0].Run(nil); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "Projection:  guide") {
		t.Errorf("text output missing projection:\n%s", out.String())
	}
}

func TestSpaceCommandProjectionMismatch(t *testing.T) {
	grp := CommandGroup(Config{
		Owner:      "prompt-manager",
		Projection: spacedoc.ProjectionGuide,
		DocPath:    writeDoc(t),
		Out:        new(bytes.Buffer),
	})
	if err := grp.Commands[0].Run([]string{"--projection", "answer"}); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestSpaceCommandMultipleProjections(t *testing.T) {
	var out bytes.Buffer
	answerDoc := filepath.Join(t.TempDir(), "answer-space.md")
	answer := strings.Replace(doc, "Projection | Guide", "Projection | Answer", 1)
	if err := os.WriteFile(answerDoc, []byte(answer), 0o644); err != nil {
		t.Fatal(err)
	}
	grp := CommandGroup(Config{
		Owner:       "prompt-manager",
		Projections: []spacedoc.Projection{spacedoc.ProjectionGuide, spacedoc.ProjectionAnswer},
		Out:         &out,
	})
	// A direct DocPath is intentionally shared by the single-projection test;
	// exercise the multi-projection resolver with a temporary repo-independent
	// path by using the command's normal document shape for the first route.
	grp = CommandGroup(Config{
		Owner:       "prompt-manager",
		Projections: []spacedoc.Projection{spacedoc.ProjectionGuide, spacedoc.ProjectionAnswer},
		DocPath:     answerDoc,
		Out:         &out,
	})
	if err := grp.Commands[0].Run([]string{"--projection", "answer", "--json"}); err != nil {
		t.Fatalf("run alternate projection: %v", err)
	}
	var def spacedoc.SpaceDefinition
	if err := json.Unmarshal(out.Bytes(), &def); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out.String())
	}
	if def.Projection != spacedoc.ProjectionAnswer {
		t.Fatalf("projection = %q, want answer", def.Projection)
	}
}
