package prose

import (
	"encoding/json"
	"errors"
	"strings"
)

// BookBrief is the durable, typed creative request supplied by a book consumer.
// Its text is preserved verbatim so a caller can recover or revise the intent.
type BookBrief struct {
	SchemaVersion       string                  `json:"schema_version"`
	Title               string                  `json:"title"`
	CreativeDirection   string                  `json:"creative_direction"`
	Audience            string                  `json:"audience"`
	ReaderPromise       string                  `json:"reader_promise"`
	Structure           string                  `json:"structure"`
	ChapterIntent       []string                `json:"chapter_intent"`
	FrontMatterIntent   []string                `json:"front_matter_intent,omitempty"`
	BackMatterIntent    []string                `json:"back_matter_intent,omitempty"`
	AdditionalDirection string                  `json:"additional_direction,omitempty"`
	EvidenceBoundary    *BookEvidenceBoundary   `json:"evidence_boundary,omitempty"`
	ProductionMetadata  *BookProductionMetadata `json:"production_metadata"`
}

// BookEvidenceBoundary distinguishes creative direction from evidence. Only
// ApprovedFacts can support concrete claims; omitted evidence boundaries
// conservatively mean that time, location and source identity are unknown.
type BookEvidenceBoundary struct {
	ApprovedFacts  []string `json:"approved_facts,omitempty"`
	UnknownDetails []string `json:"unknown_details,omitempty"`
}

const (
	UnknownDetailTime           = "time"
	UnknownDetailLocation       = "location"
	UnknownDetailSourceIdentity = "source_identity"
)

var defaultUnknownBookDetails = []string{UnknownDetailTime, UnknownDetailLocation, UnknownDetailSourceIdentity}

// BookProductionMetadata is consumer-supplied packaging context. Export refuses
// missing required fields instead of guessing production values.
type BookProductionMetadata struct {
	Subtitle          string              `json:"subtitle"`
	Language          string              `json:"language"`
	TrimWidthIn       float64             `json:"trim_width_in"`
	TrimHeightIn      float64             `json:"trim_height_in"`
	MarginIn          float64             `json:"margin_in"`
	ContentOrigin     string              `json:"content_origin"`
	GeneratedWithAI   bool                `json:"generated_with_ai"`
	GenerationRuns    []BookGenerationRun `json:"generation_runs,omitempty"`
	ExternalAssetRefs []string            `json:"external_asset_refs,omitempty"`
}

type BookGenerationRun struct {
	Date         string `json:"date"`
	Model        string `json:"model"`
	PromptSHA256 string `json:"prompt_sha256"`
	OutputSHA256 string `json:"output_sha256"`
	Provider     string `json:"provider,omitempty"`
	SectionID    string `json:"section_id,omitempty"`
	CandidateID  string `json:"candidate_id,omitempty"`
	PartIndex    int    `json:"part_index,omitempty"`
	Purpose      string `json:"purpose,omitempty"`
	WordCount    int    `json:"word_count,omitempty"`
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
}

// ManuscriptSource is the stable source export consumed by book-system
// adapters. It maps to the shared book schema's title/sections/source text;
// NB9 remains responsible for figure validation, rendering and packaging.
type ManuscriptSource struct {
	SchemaVersion      string                  `json:"schema_version"`
	SourceKind         string                  `json:"source_kind"`
	Title              string                  `json:"title"`
	Brief              *BookBrief              `json:"brief,omitempty"`
	Chapters           []ManuscriptPart        `json:"chapters"`
	FrontMatter        []ManuscriptPart        `json:"front_matter,omitempty"`
	BackMatter         []ManuscriptPart        `json:"back_matter,omitempty"`
	ReadingOrder       []ManuscriptPart        `json:"reading_order"`
	PlainText          string                  `json:"plain_text"`
	FrontMatterIntent  []string                `json:"front_matter_intent,omitempty"`
	BackMatterIntent   []string                `json:"back_matter_intent,omitempty"`
	ProductionMetadata *BookProductionMetadata `json:"production_metadata"`
}

type ManuscriptPart struct {
	ID          string `json:"id"`
	CandidateID string `json:"candidate_id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Intent      string `json:"intent,omitempty"`
	Text        string `json:"text"`
	Position    int    `json:"position"`
}

var ErrInvalidBookBrief = errors.New("invalid_book_brief")

func (b BookBrief) Validate() error {
	if b.SchemaVersion != "book-brief/v1" || strings.TrimSpace(b.Title) == "" ||
		strings.TrimSpace(b.CreativeDirection) == "" || strings.TrimSpace(b.Audience) == "" ||
		strings.TrimSpace(b.ReaderPromise) == "" || strings.TrimSpace(b.Structure) == "" || len(b.ChapterIntent) == 0 {
		return ErrInvalidBookBrief
	}
	if b.ProductionMetadata == nil || strings.TrimSpace(b.ProductionMetadata.Subtitle) == "" || strings.TrimSpace(b.ProductionMetadata.Language) == "" || b.ProductionMetadata.TrimWidthIn <= 0 || b.ProductionMetadata.TrimHeightIn <= 0 || b.ProductionMetadata.MarginIn <= 0 || strings.TrimSpace(b.ProductionMetadata.ContentOrigin) == "" {
		return ErrInvalidBookBrief
	}
	if !validGenerationRuns(b.ProductionMetadata.GenerationRuns) {
		return ErrInvalidBookBrief
	}
	if b.EvidenceBoundary != nil {
		for _, fact := range b.EvidenceBoundary.ApprovedFacts {
			if strings.TrimSpace(fact) == "" {
				return ErrInvalidBookBrief
			}
		}
		seenUnknown := map[string]bool{}
		for _, detail := range b.EvidenceBoundary.UnknownDetails {
			switch detail {
			case UnknownDetailTime, UnknownDetailLocation, UnknownDetailSourceIdentity:
			default:
				return ErrInvalidBookBrief
			}
			if seenUnknown[detail] {
				return ErrInvalidBookBrief
			}
			seenUnknown[detail] = true
		}
	}
	for _, intent := range b.ChapterIntent {
		if strings.TrimSpace(intent) == "" {
			return ErrInvalidBookBrief
		}
	}
	return nil
}

func validGenerationRuns(runs []BookGenerationRun) bool {
	for _, run := range runs {
		if strings.TrimSpace(run.Date) == "" || strings.TrimSpace(run.Model) == "" || !isSHA256(run.PromptSHA256) || !isSHA256(run.OutputSHA256) {
			return false
		}
	}
	return true
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// PromptText serializes every approved brief field without paraphrase and
// makes explicit that creative direction is not factual source material.
func (b BookBrief) PromptText() string {
	raw, _ := json.MarshalIndent(b, "", "  ")
	boundary := b.EvidenceBoundary
	unknown := append([]string(nil), defaultUnknownBookDetails...)
	if boundary != nil && len(boundary.UnknownDetails) > 0 {
		unknown = append([]string(nil), boundary.UnknownDetails...)
	}
	approved := []string(nil)
	if boundary != nil {
		approved = boundary.ApprovedFacts
	}
	approvedText := "none"
	if len(approved) > 0 {
		approvedText = strings.Join(approved, " | ")
	}
	unknownText := "none declared"
	if len(unknown) > 0 {
		unknownText = strings.Join(unknown, ", ")
	}
	return "Approved book brief (preserve its audience, promise, structure, and matter intents):\n" + string(raw) +
		"\n\nSOURCE-OF-TRUTH BOUNDARY: The brief's creative direction and intent are instructions, not evidence. Only Approved facts below support concrete factual claims; preserve a supported fact as an exact sentence and do not paraphrase or extend it. Do not invent a specific time, named location, source identity, or source example. Keep unknown details unknown; do not convert requested fields (such as time or position on a field card) into observed values." +
		"\nApproved facts: " + approvedText + ".\nUnknown details: " + unknownText + "."
}

// ExportManuscript produces a stable, render-independent JSON source document.
// Existing section order and committed text are retained exactly.
func ExportManuscript(doc Document, sectionText map[string]string) (ManuscriptSource, error) {
	if doc.BookBrief == nil {
		return ManuscriptSource{}, ErrInvalidBookBrief
	}
	if err := doc.BookBrief.Validate(); err != nil {
		return ManuscriptSource{}, err
	}
	title := doc.Title
	title = doc.BookBrief.Title
	expected := make([]ManuscriptPart, 0, len(doc.BookBrief.FrontMatterIntent)+len(doc.BookBrief.ChapterIntent)+len(doc.BookBrief.BackMatterIntent))
	for _, intent := range doc.BookBrief.FrontMatterIntent {
		expected = append(expected, ManuscriptPart{Kind: "front_matter", Title: intent})
	}
	for _, intent := range doc.BookBrief.ChapterIntent {
		expected = append(expected, ManuscriptPart{Kind: "chapter", Title: intent})
	}
	for _, intent := range doc.BookBrief.BackMatterIntent {
		expected = append(expected, ManuscriptPart{Kind: "back_matter", Title: intent})
	}
	if len(expected) != len(doc.Sections) {
		return ManuscriptSource{}, errors.New("book source export requires authored sections for every brief intent")
	}
	out := ManuscriptSource{SchemaVersion: "manuscript-source/v1", SourceKind: "prose-studio", Title: title, Brief: doc.BookBrief, Chapters: []ManuscriptPart{}}
	var full strings.Builder
	for i, s := range doc.Sections {
		kind := s.Kind
		if kind == "" {
			kind = "chapter"
		}
		if kind != expected[i].Kind || s.Intent != expected[i].Title {
			return ManuscriptSource{}, errors.New("book source export section order or intent does not match brief")
		}
		body, ok := sectionText[s.CommittedCandidateID]
		if !ok || strings.TrimSpace(body) == "" {
			return ManuscriptSource{}, errors.New("missing committed section text")
		}
		part := ManuscriptPart{ID: s.ID, CandidateID: s.CommittedCandidateID, Kind: kind, Title: s.Intent, Intent: s.Summary, Text: body, Position: i}
		switch kind {
		case "front_matter":
			out.FrontMatter = append(out.FrontMatter, part)
		case "chapter":
			out.Chapters = append(out.Chapters, part)
		case "back_matter":
			out.BackMatter = append(out.BackMatter, part)
		default:
			return ManuscriptSource{}, errors.New("unknown manuscript section kind")
		}
		out.ReadingOrder = append(out.ReadingOrder, part)
		if full.Len() > 0 {
			full.WriteString("\n\n")
		}
		full.WriteString(body)
	}
	out.FrontMatterIntent = append([]string(nil), doc.BookBrief.FrontMatterIntent...)
	out.BackMatterIntent = append([]string(nil), doc.BookBrief.BackMatterIntent...)
	metadata := *doc.BookBrief.ProductionMetadata
	metadata.GenerationRuns = append(metadata.GenerationRuns, doc.GenerationRuns...)
	for _, section := range doc.Sections {
		metadata.GenerationRuns = append(metadata.GenerationRuns, section.PartGenerationRuns...)
		if section.GenerationRun != nil {
			metadata.GenerationRuns = append(metadata.GenerationRuns, *section.GenerationRun)
		}
	}
	if !validGenerationRuns(metadata.GenerationRuns) || metadata.GeneratedWithAI && len(metadata.GenerationRuns) == 0 {
		return ManuscriptSource{}, errors.New("book source export requires complete AI generation provenance")
	}
	out.ProductionMetadata = &metadata
	out.PlainText = full.String()
	return out, nil
}
