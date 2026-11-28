package operatordecision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const brief = "# W — Start three windows\n\n## Destination\n\nOne consent.\n\n## What each commission builds\n\n| Commission | What it builds |\n|---|---|\n| `alpha` (built) | Songs <b>loud</b> |\n| `beta` | **Keys** |\n\n## If nobody answers\n\nNothing launches.\n- a\n- b\n"

func TestReadBriefSplitsSectionsFromAFolderOrFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "BRIEF.md"), []byte(brief), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, filepath.Join(dir, "BRIEF.md")} {
		sections, err := ReadBrief(path)
		if err != nil || len(sections) != 3 || sections[1].Heading != "What each commission builds" || sections[2].Body != "Nothing launches.\n- a\n- b" {
			t.Fatalf("%s: %v %#v", path, err, sections)
		}
	}
}

func TestMarkdownRendersTablesListsAndEscapes(t *testing.T) {
	sections, _ := ReadBriefFromText(brief)
	page := NewPage("q").Markdown(sections[1].Body).Markdown(sections[2].Body).HTML()
	for _, want := range []string{`<table class="hub-tradeoffs">`, `<td data-label="Commission"><code>alpha</code> (built)</td>`, "Songs &lt;b&gt;loud&lt;/b&gt;", "<strong>Keys</strong>", "<p>Nothing launches.</p><ul><li>a</li><li>b</li></ul>"} {
		if !strings.Contains(page, want) {
			t.Fatalf("page lacks %q:\n%s", want, page)
		}
	}
}

func TestPacketLineIsTakenOutOfTheRequestersWords(t *testing.T) {
	path, rest := PacketLine("Launch three windows.\nPacket: ~/.vrooli/research/decisions/W/BRIEF.md\n")
	if path != "~/.vrooli/research/decisions/W/BRIEF.md" || rest != "Launch three windows." {
		t.Fatalf("%q %q", path, rest)
	}
}

// ReadBriefFromText is ReadBrief on text already read, for tests.
func ReadBriefFromText(text string) ([]BriefSection, error) {
	dir, err := os.MkdirTemp("", "brief")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "BRIEF.md"), []byte(text), 0o600); err != nil {
		return nil, err
	}
	return ReadBrief(dir)
}
