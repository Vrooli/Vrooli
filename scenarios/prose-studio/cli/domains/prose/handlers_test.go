package prose

import (
	"strings"
	"testing"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose"
)

func TestCLIRequestJSONPreservesBookBriefAndProductionMetadata(t *testing.T) {
	const payload = `{"document":{"title":"Synthetic book","profile_key":"prose-studio/book-manuscript-local","book_brief":{"schema_version":"book-brief/v1","title":"Synthetic book","creative_direction":"Quiet prose","audience":"Synthetic readers","reader_promise":"Notice details","structure":"Twelve slots","front_matter_intent":["Preface","Guide","Dedication"],"chapter_intent":["One","Two","Three","Four","Five","Six"],"back_matter_intent":["Thanks","Glossary","Sources"],"production_metadata":{"subtitle":"Synthetic subtitle","language":"en","trim_width_in":6,"trim_height_in":9,"margin_in":0.6,"content_origin":"synthetic","generated_with_ai":true}}}}`
	request, err := decodeRequestPayload(payload, &v1.CreateDocumentRequest{})
	if err != nil {
		t.Fatal(err)
	}
	brief := request.GetDocument().GetBookBrief()
	if len(brief.GetFrontMatterIntent()) != 3 || len(brief.GetChapterIntent()) != 6 || len(brief.GetBackMatterIntent()) != 3 {
		t.Fatalf("brief slots dropped at CLI request decode: %#v", brief)
	}
	if brief.GetProductionMetadata().GetSubtitle() != "Synthetic subtitle" || brief.GetProductionMetadata().GetTrimHeightIn() != 9 {
		t.Fatalf("production metadata dropped at CLI request decode: %#v", brief.GetProductionMetadata())
	}
}

func TestBookBriefReportIncludesProductionMetadataAndIntents(t *testing.T) {
	brief := &v1.BookBrief{
		Title: "Synthetic title", CreativeDirection: "Tactile prose", Audience: "Synthetic readers",
		ReaderPromise: "Notice small things", Structure: "Twelve ordered slots",
		FrontMatterIntent: []string{"Opening note"}, ChapterIntent: []string{"Chapter one"}, BackMatterIntent: []string{"Glossary"},
		ProductionMetadata: &v1.BookProductionMetadata{Subtitle: "Synthetic subtitle", Language: "en", TrimWidthIn: 6, TrimHeightIn: 9, MarginIn: 0.6, ContentOrigin: "synthetic", GeneratedWithAi: true},
	}
	joined := strings.Join(bookBriefReportLines(brief), "\n")
	for _, value := range []string{"Synthetic title", "Tactile prose", "Synthetic readers", "Notice small things", "Twelve ordered slots", "Opening note", "Chapter one", "Glossary", "Synthetic subtitle", "\"language\": \"en\"", "\"generated_with_ai\": true"} {
		if !strings.Contains(joined, value) {
			t.Errorf("book report omitted %q; output=%s", value, joined)
		}
	}
}
