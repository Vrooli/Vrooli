package skills

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func researchRead(t *testing.T, home string, req ReadRequest) ReadResponse {
	t.Helper()
	store := NewMockStore()
	addMockSkill(store, "core", "catalog-skill", "catalog-skill.md", "Catalog Skill", "catalog body")
	handlers := NewHandlers(store, &MockMetricsService{}, "/test/store")
	handlers.SetResearchHome(home)
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	handlers.Read(w, httptest.NewRequest("POST", "/skills/read", bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("read: status %d: %s", w.Code, w.Body.String())
	}
	var resp ReadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp
}

func writeResearchSkill(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + filepath.Base(filepath.Dir(path)) + "\ndescription: Present UX decisions.\n---\n" + body
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadResolvesResearchSkillsFromFilesOnEveryRead(t *testing.T) {
	home := t.TempDir()
	topicSkill := filepath.Join(home, "topics", "experience-engineering", "skills", "present-ux-decisions", "SKILL.md")
	writeResearchSkill(t, topicSkill, "# Present {{WHAT}}\n")
	writeResearchSkill(t, filepath.Join(home, "skills", "topic-audit", "SKILL.md"), "# Audit\n")

	resp := researchRead(t, home, ReadRequest{
		Identifiers: []string{"research/experience-engineering/present-ux-decisions", "research/topic-audit", "catalog-skill"},
		Output:      "skills",
		Variables:   map[string]string{"WHAT": "redesigns"},
	})
	if len(resp.Skills) != 3 || len(resp.Missing) != 0 {
		t.Fatalf("skills=%+v missing=%+v", resp.Skills, resp.Missing)
	}
	got := resp.Skills[0]
	if got.ID != "research/experience-engineering/present-ux-decisions" || got.Folder != "research" ||
		got.Content != "# Present redesigns\n" || got.Description != "Present UX decisions." {
		t.Fatalf("topic skill = %+v", got)
	}
	if real, _ := filepath.EvalSymlinks(topicSkill); got.ContentPath != real {
		t.Fatalf("content path = %q, want %q", got.ContentPath, real)
	}
	if resp.Skills[1].ID != "research/topic-audit" || resp.Skills[1].Content != "# Audit\n" {
		t.Fatalf("home skill = %+v", resp.Skills[1])
	}
	if resp.Skills[2].ID != "catalog-skill" {
		t.Fatalf("catalog reads are unchanged: %+v", resp.Skills[2])
	}

	writeResearchSkill(t, topicSkill, "# Edited\n")
	again := researchRead(t, home, ReadRequest{Identifiers: []string{"research/experience-engineering/present-ux-decisions"}})
	if len(again.Skills) != 1 || again.Skills[0].Content != "# Edited\n" {
		t.Fatalf("an edit must be served on the next read: %+v", again.Skills)
	}
}

func TestReadReportsMissingAndRefusedResearchSkills(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	writeResearchSkill(t, filepath.Join(outside, "leak", "SKILL.md"), "secret\n")
	if err := os.MkdirAll(filepath.Join(home, "topics", "t", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "leak"), filepath.Join(home, "topics", "t", "skills", "leak")); err != nil {
		t.Fatal(err)
	}

	resp := researchRead(t, home, ReadRequest{Identifiers: []string{"research/t/absent", "research/../etc/passwd", "research/t/leak"}})
	reasons := map[string]string{}
	for _, miss := range resp.Missing {
		reasons[miss.Identifier] = miss.Reason
	}
	want := map[string]string{"research/t/absent": "not_found", "research/../etc/passwd": "invalid_research_id", "research/t/leak": "outside_research_home"}
	for id, reason := range want {
		if reasons[id] != reason {
			t.Errorf("%s: reason %q, want %q", id, reasons[id], reason)
		}
	}
	for _, skill := range resp.Skills {
		if strings.Contains(skill.Content, "secret") {
			t.Fatal("content outside the research home was served")
		}
	}
}
