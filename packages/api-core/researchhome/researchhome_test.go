package researchhome

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func skill(name string) string {
	return "---\nname: " + name + "\ndescription: does " + name + "\nmetadata:\n  name: nested\n---\n# " + name + "\n"
}

func TestDirPrefersTheEnvironmentAndRefusesARelativeHome(t *testing.T) {
	got, err := Dir(func(string) string { return "/srv/research/" })
	if err != nil || got != "/srv/research" {
		t.Fatalf("Dir = %q, %v", got, err)
	}
	if _, err := Dir(func(string) string { return "relative" }); err == nil {
		t.Fatal("a relative research home must be refused")
	}
	t.Setenv("HOME", "/home/someone")
	got, err = Dir(func(string) string { return "" })
	if err != nil || got != "/home/someone/.vrooli/research" {
		t.Fatalf("default Dir = %q, %v", got, err)
	}
}

func TestParseAcceptsTopicAndHomeIDsAndRefusesTraversal(t *testing.T) {
	for id, want := range map[string]Ref{
		"research/experience-engineering/present-ux-decisions": {Topic: "experience-engineering", Name: "present-ux-decisions"},
		"research/topic-audit":                                 {Name: "topic-audit"},
	} {
		got, err := Parse(id)
		if err != nil || got != want || got.ID() != id {
			t.Fatalf("Parse(%q) = %+v, %v", id, got, err)
		}
	}
	if _, err := Parse("program-runtime.research-workspace-review"); !errors.Is(err, ErrNotResearch) {
		t.Fatalf("a scenario id is not a research id: %v", err)
	}
	for _, id := range []string{"research/", "research/../etc", "research/topic/../x", "research/a/b/c", "research/.hidden/x", "research/topic/Upper", "research//x", "research/topic/"} {
		if _, err := Parse(id); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Parse(%q) = %v, want ErrInvalid", id, err)
		}
	}
}

func TestReadSkillReadsTheFileFreshAndRefusesEscapes(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "topics", "ux", "skills", "present", "SKILL.md")
	write(t, path, skill("present"))
	ref, first, err := ReadSkill(home, "research/ux/present")
	if err != nil || ref != (Ref{Topic: "ux", Name: "present"}) || !strings.Contains(string(first.Content), "# present") {
		t.Fatalf("ReadSkill = %+v %q, %v", ref, first.Content, err)
	}
	write(t, path, skill("present")+"edited\n")
	if _, second, err := ReadSkill(home, "research/ux/present"); err != nil || !strings.Contains(string(second.Content), "edited") {
		t.Fatalf("an edit must be visible on the next read: %q, %v", second.Content, err)
	}

	outside := t.TempDir()
	write(t, filepath.Join(outside, "SKILL.md"), skill("leak"))
	if err := os.Symlink(outside, filepath.Join(home, "topics", "ux", "skills", "leak")); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{
		"research/ux/absent": "not_found",
		"research/ux/leak":   "outside_research_home",
		"research/../etc":    "invalid_research_id",
		"other/ux/present":   "invalid_research_id",
	} {
		if _, _, err := ReadSkill(home, id); Reason(err) != want {
			t.Errorf("ReadSkill(%q) = %v, reason %q; want %q", id, err, Reason(err), want)
		}
	}
	t.Setenv(EnvHome, "relative/home")
	if _, _, err := ReadSkill("", "research/present"); Reason(err) != "research_home_unavailable" {
		t.Errorf("a relative %s is refused: %v", EnvHome, err)
	}
}

func TestListReportsItemsWithTheirProblems(t *testing.T) {
	home := t.TempDir()
	topic := filepath.Join(home, "topics", "ux")
	write(t, filepath.Join(topic, "skills", "good", "SKILL.md"), skill("good"))
	write(t, filepath.Join(topic, "skills", "renamed", "SKILL.md"), skill("other"))
	write(t, filepath.Join(topic, "skills", "empty", "notes.md"), "x")
	write(t, filepath.Join(topic, "skills", "Bad_Name", "SKILL.md"), skill("Bad_Name"))
	write(t, filepath.Join(topic, "programs", "review.json"), `{"name":"research/ux/review"}`)
	write(t, filepath.Join(topic, "programs", "review.py"), "print(1)")
	write(t, filepath.Join(topic, "programs", "copied.json"), `{"name":"research/other/copied"}`)
	write(t, filepath.Join(topic, "programs", "copied.py"), "")
	write(t, filepath.Join(topic, "programs", "nosource.json"), `{"name":"research/ux/nosource"}`)
	write(t, filepath.Join(topic, "programs", "orphan.py"), "")
	write(t, filepath.Join(topic, "programs", "review.output.schema.json"), `{}`)
	write(t, filepath.Join(topic, "programs", "__pycache__", "x.pyc"), "")

	items, err := List(home, "ux")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, item := range items {
		got[item.ID] = strings.Join(item.Problems, "; ")
	}
	want := map[string]string{
		"research/ux/good":     "",
		"research/ux/renamed":  `SKILL.md name "other" must match its folder "renamed"`,
		"research/ux/empty":    "SKILL.md is missing",
		"research/ux/Bad_Name": "lowercase",
		"research/ux/review":   "",
		"research/ux/copied":   `contract name "research/other/copied" must be "research/ux/copied"`,
		"research/ux/nosource": "program source",
		"research/ux/orphan":   "program source has no orphan.json contract",
	}
	if len(got) != len(want) {
		t.Fatalf("items = %v", got)
	}
	for id, fragment := range want {
		problems, ok := got[id]
		if !ok || (fragment == "" && problems != "") || !strings.Contains(problems, fragment) {
			t.Errorf("%s problems = %q, want %q", id, problems, fragment)
		}
	}

	if items, err := List(home, ""); err != nil || len(items) != 0 {
		t.Fatalf("an empty research home lists nothing: %v, %v", items, err)
	}
}

func TestScopesSkipsTopicAliases(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "topics", "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, "topics", "real"), filepath.Join(home, "topics", "alias")); err != nil {
		t.Fatal(err)
	}
	scopes, err := Scopes(home)
	if err != nil || len(scopes) != 2 || scopes[0].Topic != "" || scopes[1].Topic != "real" {
		t.Fatalf("Scopes = %+v, %v", scopes, err)
	}
}
