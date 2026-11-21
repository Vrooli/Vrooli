package home

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/researchhome"
)

// write puts one file under the fixture's research home.
func (f *fixture) write(t *testing.T, rel, content string) {
	t.Helper()
	path := filepath.Join(f.research, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func topicNamed(t *testing.T, snap *Snapshot, slug string) *Topic {
	t.Helper()
	for _, topic := range snap.Topics {
		if topic.Slug == slug {
			return topic
		}
	}
	t.Fatalf("topic %s not listed", slug)
	return nil
}

func byID(items []researchhome.Item) map[string]researchhome.Item {
	out := map[string]researchhome.Item{}
	for _, item := range items {
		out[item.ID] = item
	}
	return out
}

const presentSkill = "---\nname: present\ndescription: Present a decision.\n---\n# Present\n"

func TestTopicDocumentsReportResearchCapabilities(t *testing.T) {
	f := newFixture(t)
	f.write(t, "topics/local-model-performance/skills/present/SKILL.md", presentSkill)
	f.write(t, "topics/local-model-performance/skills/renamed/SKILL.md", presentSkill)
	f.write(t, "topics/local-model-performance/skills/empty/README.md", "no skill here\n")
	f.write(t, "topics/local-model-performance/programs/pulse.json", `{"name":"research/local-model-performance/pulse"}`)
	f.write(t, "topics/local-model-performance/programs/pulse.py", "program.report()\n")
	f.write(t, "topics/local-model-performance/programs/stray.py", "program.report()\n")

	caps := topicNamed(t, f.snapshot(t, Options{}), "local-model-performance").Documents.Capabilities
	if !caps.Known {
		t.Fatalf("capabilities unknown: %s", caps.Reason)
	}
	items := byID(caps.Items)
	present := items["research/local-model-performance/present"]
	if present.Kind != researchhome.KindSkill || len(present.Problems) != 0 || !strings.HasSuffix(present.Path, "skills/present/SKILL.md") {
		t.Fatalf("a valid topic skill resolves by its prefix id with its file: %+v", present)
	}
	if problems := strings.Join(items["research/local-model-performance/renamed"].Problems, "; "); !strings.Contains(problems, "present") {
		t.Fatalf("a frontmatter name that is not the folder name is a problem: %q", problems)
	}
	if problems := strings.Join(items["research/local-model-performance/empty"].Problems, "; "); !strings.Contains(problems, "SKILL.md") {
		t.Fatalf("a skill folder without SKILL.md is a problem: %q", problems)
	}
	if pulse := items["research/local-model-performance/pulse"]; pulse.Kind != researchhome.KindProgram || len(pulse.Problems) != 0 {
		t.Fatalf("a contract whose name is its research id resolves: %+v", pulse)
	}
	if len(items["research/local-model-performance/stray"].Problems) == 0 {
		t.Fatal("a program source without a contract is a problem")
	}

	other := topicNamed(t, f.snapshot(t, Options{}), "niche-book-production").Documents.Capabilities
	if !other.Known || len(other.Items) != 0 {
		t.Fatalf("a topic without skills/ or programs/ is known to have none: %+v", other)
	}
}

func TestSnapshotReportsHomeCapabilitiesFreshOnEveryRead(t *testing.T) {
	f := newFixture(t)
	f.write(t, "skills/research-review/SKILL.md", "---\nname: research-review\ndescription: Review topics.\n---\n")
	reader := NewReader(Config{ResearchDir: f.research, CodexDir: f.codex}, Deps{Clock: f.clock, Procs: f.procs})

	first, err := reader.Snapshot(context.Background(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := byID(first.Capabilities.Items)["research/research-review"]; got.Kind != researchhome.KindSkill || len(got.Problems) != 0 {
		t.Fatalf("a supervisor skill resolves by research/<name>: %+v", first.Capabilities)
	}

	f.write(t, "programs/run-freshness.json", `{"name":"research/wrong"}`)
	f.write(t, "programs/run-freshness.py", "program.report()\n")
	second, err := reader.Snapshot(context.Background(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	program := byID(second.Capabilities.Items)["research/run-freshness"]
	if program.Kind != researchhome.KindProgram || !strings.Contains(strings.Join(program.Problems, "; "), "research/wrong") {
		t.Fatalf("a program added after the first read is listed on the next, with its name problem: %+v", second.Capabilities)
	}
}

func TestCapabilitiesAreUnknownWhenTheFoldersCannotBeRead(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every folder")
	}
	f := newFixture(t)
	skills := filepath.Join(f.research, "topics", "local-model-performance", "skills")
	if err := os.Mkdir(skills, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(skills, 0o755) })
	caps := topicNamed(t, f.snapshot(t, Options{}), "local-model-performance").Documents.Capabilities
	if caps.Known || !strings.Contains(caps.Reason, "skills") {
		t.Fatalf("an unreadable skills folder makes the listing unknown, never empty: %+v", caps)
	}
}
