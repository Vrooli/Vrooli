package prose

import (
	"strings"
	"testing"
)

func TestBookBriefPromptAndManuscriptExport(t *testing.T) {
	brief := &BookBrief{SchemaVersion: "book-brief/v1", Title: "The Quiet Atlas", CreativeDirection: "Field-note essays with lucid, tactile language", Audience: "Curious urban walkers", ReaderPromise: "Notice overlooked city ecologies", Structure: "Seasonal braided essays", ChapterIntent: []string{"Rain routes", "Night pollinators"}, FrontMatterIntent: []string{"A short invitation"}, BackMatterIntent: []string{"A field glossary"}, AdditionalDirection: "Invented setting; no factual claims required", ProductionMetadata: &BookProductionMetadata{Subtitle: "Synthetic walks", Language: "en", TrimWidthIn: 6, TrimHeightIn: 9, MarginIn: 0.6, ContentOrigin: "synthetic", GeneratedWithAI: true, GenerationRuns: []BookGenerationRun{{Date: "2026-11-24T00:00:00Z", Model: "fixture", PromptSHA256: strings.Repeat("a", 64), OutputSHA256: strings.Repeat("b", 64)}}}}
	if err := brief.Validate(); err != nil {
		t.Fatal(err)
	}
	prompt := brief.PromptText()
	for _, value := range []string{brief.Title, brief.CreativeDirection, brief.Audience, brief.ReaderPromise, brief.Structure, brief.ChapterIntent[0], brief.FrontMatterIntent[0], brief.BackMatterIntent[0], brief.AdditionalDirection} {
		if !strings.Contains(prompt, value) {
			t.Fatalf("prompt omitted brief text %q", value)
		}
	}
	doc := Document{Title: brief.Title, BookBrief: brief, Sections: []Section{
		{ID: "fm", Kind: "front_matter", Intent: "A short invitation", CommittedCandidateID: "c0"},
		{ID: "s1", Kind: "chapter", Intent: "Rain routes", Summary: "A walk after a storm", CommittedCandidateID: "c1"},
		{ID: "s2", Kind: "chapter", Intent: "Night pollinators", CommittedCandidateID: "c2"},
		{ID: "bm", Kind: "back_matter", Intent: "A field glossary", CommittedCandidateID: "c3"},
	}}
	export, err := ExportManuscript(doc, map[string]string{"c0": "Begin with the invitation.", "c1": "Water threads the old pavement.", "c2": "Moths gather under amber lamps.", "c3": "Glossary prose goes here."})
	if err != nil {
		t.Fatal(err)
	}
	if export.SchemaVersion != "manuscript-source/v1" || export.Chapters[0].Text != "Water threads the old pavement." || export.FrontMatter[0].Text != "Begin with the invitation." || export.BackMatter[0].Text != "Glossary prose goes here." || len(export.ReadingOrder) != 4 || export.ReadingOrder[3].ID != "bm" {
		t.Fatalf("unexpected source export: %#v", export)
	}
}

func TestBookBriefRejectsIncompleteContract(t *testing.T) {
	if err := (BookBrief{SchemaVersion: "book-brief/v1", Title: "Untitled"}).Validate(); err == nil {
		t.Fatal("expected incomplete brief rejection")
	}
}

func TestExportManuscriptRejectsMissingSectionTextAndBrief(t *testing.T) {
	if _, err := ExportManuscript(Document{}, nil); err == nil {
		t.Fatal("expected missing brief rejection")
	}
	brief := &BookBrief{SchemaVersion: "book-brief/v1", Title: "T", CreativeDirection: "D", Audience: "A", ReaderPromise: "P", Structure: "S", ChapterIntent: []string{"C"}, ProductionMetadata: &BookProductionMetadata{Subtitle: "S", Language: "en", TrimWidthIn: 6, TrimHeightIn: 9, MarginIn: 0.5, ContentOrigin: "synthetic"}}
	_, err := ExportManuscript(Document{Title: "T", BookBrief: brief, Sections: []Section{{CommittedCandidateID: "missing"}}}, nil)
	if err == nil {
		t.Fatal("expected missing committed section text rejection")
	}
	brief.FrontMatterIntent = []string{"Unwritten invitation"}
	_, err = ExportManuscript(Document{Title: "T", BookBrief: brief, Sections: []Section{{Kind: "chapter", Intent: "C", CommittedCandidateID: "c1"}}}, map[string]string{"c1": "Chapter text"})
	if err == nil || !strings.Contains(err.Error(), "every brief intent") {
		t.Fatalf("expected authored-matter completeness refusal, got %v", err)
	}
}
