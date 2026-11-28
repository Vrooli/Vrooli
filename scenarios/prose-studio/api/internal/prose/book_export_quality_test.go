package prose

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/textmetrics"
)

type part5FailFixture struct {
	SourceDocumentID string    `json:"source_document_id"`
	Document         Document  `json:"document"`
	Part5Candidate   Candidate `json:"part5_candidate"`
}

func TestExportBookSourceRejectsRetained560WordQualityFailureFromStoredParts(t *testing.T) {
	s, doc, _ := storeRetainedQualityFixture(t)
	_, err := s.ExportBookSource(context.Background(), doc.ID)
	require.ErrorContains(t, err, "stored part text fails quality boundary")
	require.ErrorContains(t, err, "unsupported named location detail")
}

func TestExportBookSourceRejectsParaphrasedNearDuplicatePartPair(t *testing.T) {
	s, doc, _ := storePositiveBookSource(t)
	candidate := loadCanonicalCandidate(t, s, doc)
	first, err := s.loadCandidate(context.Background(), candidate.DerivedFrom[0])
	require.NoError(t, err)
	paraphrase := "At the fictional Atrium North, a soft tone repeats twice before a quiet pause. This modest pattern gives the page a clear beginning without turning one observation into a story. The interval is brief; its value lies in the relation between repetition and silence. A reader can keep the fact intact and still notice its shape: first the return, then the gap, then the expectation that follows. Nothing here names a cause. The passage starts with what the record supports and leaves the rest open."
	require.Less(t, textmetrics.SectionNovelty(paraphrase, first.Text), 0.25, "this is a paraphrased near duplicate rather than a byte-identical copy")
	rewriteStoredPartText(t, s, doc, 1, paraphrase)
	_, err = s.ExportBookSource(context.Background(), doc.ID)
	require.ErrorContains(t, err, "part 2 repeats the substance of part 1")
}

func TestExportBookSourceRejectsPreservedPartFiveFailClosedDraft(t *testing.T) {
	raw, err := os.ReadFile("testdata/part5-fail-closed.json")
	require.NoError(t, err)
	var fixture part5FailFixture
	require.NoError(t, json.Unmarshal(raw, &fixture))
	require.Equal(t, "39c99716-d11a-4e8f-9811-5afd4efb1cea", fixture.SourceDocumentID)
	require.False(t, fixture.Part5Candidate.Committed, "the rejected part never became canonical source")
	require.True(t, fixture.Part5Candidate.Eligibility.Eligible, "generic length/format eligibility did not cover book-specific evidence quality")
	require.Contains(t, bookBriefViolation(fixture.Document.BookBrief, fixture.Part5Candidate.Text), "unsupported concrete source example or identity: engine")

	s, _ := newTestService(t)
	ctx := context.Background()
	require.NoError(t, s.saveJSON(ctx, "prose_documents", fixture.Document.ID, fixture.Document))
	_, err = s.ExportBookSource(ctx, fixture.SourceDocumentID)
	require.ErrorContains(t, err, "requires at least one committed section")
}

func TestExportBookSourceAcceptsGroundedPositiveChunkAndKeepsPreauthoredContract(t *testing.T) {
	s, doc, expected := storePositiveBookSource(t)
	source, err := s.ExportBookSource(context.Background(), doc.ID)
	require.NoError(t, err)
	require.Equal(t, "manuscript-source/v1", source.SchemaVersion)
	require.Equal(t, expected, source.Chapters[0].Text)
	require.Len(t, source.ProductionMetadata.GenerationRuns, 7)
	require.GreaterOrEqual(t, textmetrics.Analyze(source.Chapters[0].Text, nil).WordCount, 440)
	require.LessOrEqual(t, textmetrics.Analyze(source.Chapters[0].Text, nil).WordCount, 660)

	// The public source contract also permits caller-authored, non-AI content
	// with committed text and no chunk-generation records.
	brief := *syntheticBookBriefForChunks()
	brief.ProductionMetadata.GeneratedWithAI = false
	preauthored, err := ExportManuscript(Document{Title: brief.Title, BookBrief: &brief, Sections: []Section{{ID: "authored", Kind: "chapter", Intent: brief.ChapterIntent[0], CommittedCandidateID: "authored-candidate"}}}, map[string]string{"authored-candidate": "A complete, preauthored synthetic chapter."})
	require.NoError(t, err)
	require.Equal(t, "A complete, preauthored synthetic chapter.", preauthored.Chapters[0].Text)
}

func TestExportBookSourceRejectsSectionPartAndProvenanceMutations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, *Service, *Document)
		want   string
	}{
		{
			name: "section position drift",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				doc.Sections[0].Position = 1
				saveDocumentAndSections(t, s, *doc)
			},
			want: "section order/kind/intent drift",
		},
		{
			name: "section intent drift",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				doc.Sections[0].Intent = "A different chapter intent"
				saveDocumentAndSections(t, s, *doc)
			},
			want: "section order/kind/intent drift",
		},
		{
			name: "section kind drift",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				doc.Sections[0].Kind = "back_matter"
				saveDocumentAndSections(t, s, *doc)
			},
			want: "section order/kind/intent drift",
		},
		{
			name: "part candidate ordering drift",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				candidate := loadCanonicalCandidate(t, s, *doc)
				candidate.DerivedFrom[0], candidate.DerivedFrom[1] = candidate.DerivedFrom[1], candidate.DerivedFrom[0]
				require.NoError(t, s.saveJSON(context.Background(), "prose_candidates", candidate.ID, candidate))
			},
			want: "part 1 identity or order drift",
		},
		{
			name: "part purpose drift",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				doc.Sections[0].PartGenerationRuns[0].Purpose = "a different purpose"
				saveDocumentAndSections(t, s, *doc)
			},
			want: "part 1 purpose differs",
		},
		{
			name: "part text no longer matches its hash and measured count",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				candidate := loadCanonicalCandidate(t, s, *doc)
				part, err := s.loadCandidate(context.Background(), candidate.DerivedFrom[0])
				require.NoError(t, err)
				part.Text += " An extra synthetic sentence changes the committed part."
				require.NoError(t, s.saveJSON(context.Background(), "prose_candidates", part.ID, part))
			},
			want: "part 1 text/hash/metrics/generation run does not match",
		},
		{
			name: "canonical output differs from part derivation",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				candidate := loadCanonicalCandidate(t, s, *doc)
				candidate.Text += " Different canonical output."
				require.NoError(t, s.saveJSON(context.Background(), "prose_candidates", candidate.ID, candidate))
			},
			want: "canonical section text is not exactly derived",
		},
		{
			name: "missing prompt provenance",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				candidate := loadCanonicalCandidate(t, s, *doc)
				part, err := s.loadCandidate(context.Background(), candidate.DerivedFrom[0])
				require.NoError(t, err)
				part.Provenance.PromptSHA256 = ""
				require.NoError(t, s.saveJSON(context.Background(), "prose_candidates", part.ID, part))
			},
			want: "part 1 route/model/token/prompt provenance is incomplete",
		},
		{
			name: "provider no longer matches part evidence",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				candidate := loadCanonicalCandidate(t, s, *doc)
				part, err := s.loadCandidate(context.Background(), candidate.DerivedFrom[0])
				require.NoError(t, err)
				part.Provenance.Provider = "different-local-provider"
				require.NoError(t, s.saveJSON(context.Background(), "prose_candidates", part.ID, part))
			},
			want: "part 1 text/hash/metrics/generation run does not match",
		},
		{
			name: "part index provenance drift",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				doc.Sections[0].PartGenerationRuns[0].PartIndex = 2
				saveDocumentAndSections(t, s, *doc)
			},
			want: "part 1 identity or order drift",
		},
		{
			name: "unbound candidate round",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				candidate := loadCanonicalCandidate(t, s, *doc)
				part, err := s.loadCandidate(context.Background(), candidate.DerivedFrom[0])
				require.NoError(t, err)
				var round Round
				require.NoError(t, s.loadJSON(context.Background(), "prose_rounds", part.RoundID, &round))
				round.CandidateIDs = nil
				require.NoError(t, s.saveJSON(context.Background(), "prose_rounds", round.ID, round))
			},
			want: "candidate round does not bind",
		},
		{
			name: "stale chunk schema version",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				candidate := loadCanonicalCandidate(t, s, *doc)
				candidate.SectionAssembly.SchemaVersion = "book-section-chunks/v1"
				require.NoError(t, s.saveJSON(context.Background(), "prose_candidates", candidate.ID, candidate))
			},
			want: "stale chunk assembly contract",
		},
		{
			name: "stale generation date",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				doc.Sections[0].PartGenerationRuns[0].Date = doc.CreatedAt.Add(-time.Hour).Format(time.RFC3339)
				saveDocumentAndSections(t, s, *doc)
			},
			want: "generation date is outside the document's durable lifetime",
		},
		{
			name: "changed full section bound",
			mutate: func(t *testing.T, s *Service, doc *Document) {
				candidate := loadCanonicalCandidate(t, s, *doc)
				candidate.SectionAssembly.MinWords = 439
				require.NoError(t, s.saveJSON(context.Background(), "prose_candidates", candidate.ID, candidate))
			},
			want: "word bounds are stale or mismatched",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, doc, _ := storePositiveBookSource(t)
			tc.mutate(t, s, &doc)
			_, err := s.ExportBookSource(context.Background(), doc.ID)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func storeRetainedQualityFixture(t *testing.T) (*Service, Document, Candidate) {
	t.Helper()
	fixture, parts := loadFailedChunkFixture(t)
	s, _ := newTestService(t)
	ctx := context.Background()
	profileKey, version := splitProfileVersion(t, fixture.Document.SectionProfileVersion)
	createExportTestProfile(t, s, profileKey, version)
	doc := fixture.Document
	section := doc.Sections[0]
	partRoundIDs := make([]string, len(parts))
	for i := range parts {
		part := parts[i]
		part.RoundID = "fixture-round-" + part.ID
		part.Provenance.ProfileVersion = doc.SectionProfileVersion
		part.Provenance.GatewayRole = "write.default"
		part.Provenance.Strategy = samplerDirect
		part.Provenance.StrategyParameters = Sampler{Kind: samplerDirect, K: 1}
		part.Provenance.MaxOutputTokensEffective = 2048
		part.Provenance.MachineGenerated = true
		part.Provenance.Disclosure = "machine_generated"
		part.Measurements = textmetrics.Analyze(part.Text, nil)
		part.Eligibility = Eligibility{Eligible: true}
		parts[i] = part
		partRoundIDs[i] = part.RoundID
		require.NoError(t, s.saveJSON(ctx, "prose_candidates", part.ID, part))
		require.NoError(t, s.saveJSON(ctx, "prose_rounds", part.RoundID, Round{ID: part.RoundID, SessionID: section.SessionID, Strategy: Sampler{Kind: samplerDirect, K: 1}, CandidateIDs: []string{part.ID}}))
	}
	assembly := fixture.CanonicalCandidate
	roundIDs := append(partRoundIDs, assembly.RoundID)
	require.NoError(t, s.saveJSON(ctx, "prose_rounds", assembly.RoundID, Round{ID: assembly.RoundID, SessionID: section.SessionID, CompositionMode: assembly.SectionAssembly.SchemaVersion, Strategy: Sampler{Kind: "composed", K: 1}, CandidateIDs: []string{assembly.ID}}))
	require.NoError(t, s.saveJSON(ctx, "prose_sessions", section.SessionID, Session{ID: section.SessionID, ProfileKey: profileKey, Status: "active", RoundIDs: roundIDs}))
	require.NoError(t, s.saveJSON(ctx, "prose_candidates", assembly.ID, assembly))
	require.NoError(t, s.saveSection(ctx, section))
	require.NoError(t, s.saveJSON(ctx, "prose_documents", doc.ID, doc))
	return s, doc, assembly
}

func storePositiveBookSource(t *testing.T) (*Service, Document, string) {
	t.Helper()
	s, _ := newTestService(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second).Add(-time.Second)
	s.now = func() time.Time { return now }
	profileKey := "test/book-source-local"
	profile, err := s.CreateProfile(ctx, Profile{Key: profileKey, Version: 1, GatewayRole: "write.default", Locality: "local_only", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 2048}, Constraints: Constraints{RequiredFormat: "rich_prose"}, Composition: CompositionPolicy{TargetSectionWords: bookChapterWordTarget, SectionWordTolerance: 0.2, SectionFormat: "rich_prose", SectionMaxOutputTokens: 2048, MaxSections: 16}})
	require.NoError(t, err)
	brief := &BookBrief{
		SchemaVersion: "book-brief/v1", Title: "Synthetic Atrium Listening Note",
		CreativeDirection:   "Quiet, exact field prose with restrained sensory language.",
		Audience:            "Synthetic readers practicing careful observation.",
		ReaderPromise:       "Separate audible qualities from uncertain explanations.",
		Structure:           "One short chapter with a modest opening, distinct observation steps, a blank record card, and a compact close.",
		ChapterIntent:       []string{"Record the fictional atrium observation and distinguish quality from interpretation."},
		AdditionalDirection: "Fictional synthetic facts only. Do not invent source identities or personal stories.",
		EvidenceBoundary: &BookEvidenceBoundary{
			ApprovedFacts: []string{
				"At the fictional Atrium North, a soft tone repeats twice before a quiet pause.",
				"At the fictional Atrium North, the recorded interval lasted ten seconds.",
			},
			UnknownDetails: []string{UnknownDetailSourceIdentity},
		},
		ProductionMetadata: &BookProductionMetadata{Subtitle: "A Synthetic Observation Exercise", Language: "en", TrimWidthIn: 6, TrimHeightIn: 9, MarginIn: 0.6, ContentOrigin: "Synthetic test fixture; no external facts.", GeneratedWithAI: true},
	}
	partsText := []string{
		"At the fictional Atrium North, a soft tone repeats twice before a quiet pause. This small pattern gives the page a clear beginning without turning one observation into a story. The interval is modest; its value lies in the relation between repetition and silence. A reader can keep the fact intact and still notice its shape: first the return, then the gap, then the expectation that follows. Nothing here names a cause. The passage begins with what the record supports and leaves the rest open.",
		"At the fictional Atrium North, the recorded interval lasted ten seconds. Within that boundary, attention can widen from the repeated tone to the quiet around it. The broad bed is not a hidden object to identify; it is a set of audible qualities that can be compared without adding a source. One may note whether the texture feels smooth or broken, steady or changing, then keep those words close to the observation. The useful distinction is between a quality the listener records and an explanation the page does not authorize.",
		"A second pass separates nearer detail from the wider field by comparing qualities rather than naming origins. A sharper edge may seem close; a softer contour may seem distant, but the notebook records those as impressions, not coordinates. The reader can set two phrases side by side and ask what changed in clarity, rhythm, or apparent distance. This comparison adds a new layer to the first observation instead of repeating its introduction. If the distinction remains uncertain, the written note can preserve that uncertainty as part of the result.",
		"The record card has four blank spaces: time, position, audible qualities, and uncertainty. The first two remain empty until a reader supplies them from the actual visit. In the third, short descriptive phrases are more useful than a theory; in the fourth, a mark can show where confidence ends. A blank is an honest result when no value has been recorded. The card therefore supports careful attention without pretending that the synthetic example contains personal measurements or details beyond its approved sentences.",
		"Interpretation begins where direct description stops. A repeated tone and a quiet pause are the approved observations; a claim about what made them would exceed this record. The reader can preserve a verbatim fact, add a clearly labeled impression, and leave the proposed cause unresolved. This boundary does not diminish the note. It makes each sentence easier to inspect and keeps the difference between evidence and guess visible to a later reader who did not share the moment.",
		"The chapter closes by returning the reader to a practical habit: keep a compact description, separate it from interpretation, and leave unsupported values blank. Those steps make a small observation reusable without making it larger than the evidence. A future entry can add a measured time or a supplied location, but this sample does not presume either. The finished card remains legible because each detail has a place and each uncertainty remains explicit. Careful writing turns a brief encounter into a useful record through precision, not embellishment.",
	}
	section := Section{ID: "positive-section", DocumentID: "positive-document", Position: 0, Intent: brief.ChapterIntent[0], Summary: "A fictional observation and its evidence limits.", TargetWords: bookChapterWordTarget, ProfileKey: profile.Key, SessionID: "positive-session", Kind: "chapter"}
	plans := bookSectionPartPlans(section, bookChapterWordTarget, 440, nil)
	require.Len(t, plans, 6)
	var parts []Candidate
	roundIDs := make([]string, len(plans)+1)
	for i, text := range partsText {
		partID := fmt.Sprintf("positive-part-%d", i+1)
		roundID := fmt.Sprintf("positive-round-%d", i+1)
		roundIDs[i] = roundID
		part := Candidate{ID: partID, RoundID: roundID, Text: text, Measurements: textmetrics.Analyze(text, nil), Eligibility: Eligibility{Eligible: true}, Provenance: Provenance{
			ProfileVersion: fmt.Sprintf("%s@%d", profile.Key, profile.Version), GatewayRole: "write.default", Locality: "local_only", Strategy: samplerDirect,
			StrategyParameters: Sampler{Kind: samplerDirect, K: 1}, Provider: "ollama", ResolvedModelRef: "gemma4:12b", PromptSHA256: hash([]byte("synthetic-approved-prompt-" + partID)),
			MaxOutputTokensEffective: 2048, InputTokens: 150 + i, OutputTokens: 100 + i, CostMicros: 3, MachineGenerated: true, Disclosure: "machine_generated",
		}}
		parts = append(parts, part)
		require.NoError(t, s.saveJSON(ctx, "prose_candidates", part.ID, part))
		require.NoError(t, s.saveJSON(ctx, "prose_rounds", roundID, Round{ID: roundID, SessionID: section.SessionID, Strategy: Sampler{Kind: samplerDirect, K: 1}, CandidateIDs: []string{part.ID}}))
	}
	constraints := Constraints{MinWords: 440, MaxWords: 660, RequiredFormat: "rich_prose"}
	assembled, err := assembleBookSectionCandidate(brief, section, plans, parts, constraints, nil, nil)
	require.NoError(t, err)
	assembled.Committed = true
	assemblyRoundID := assembled.RoundID
	roundIDs[len(roundIDs)-1] = assemblyRoundID
	sectionTime := now.Format(time.RFC3339)
	partRuns := make([]BookGenerationRun, len(parts))
	for i, evidence := range assembled.SectionAssembly.Parts {
		partRuns[i] = BookGenerationRun{Date: sectionTime, Model: evidence.Model, Provider: evidence.Provider, PromptSHA256: evidence.PromptSHA256, OutputSHA256: evidence.OutputSHA256, SectionID: section.ID, CandidateID: evidence.CandidateID, PartIndex: evidence.Index, Purpose: evidence.Purpose, WordCount: evidence.WordCount, InputTokens: evidence.InputTokens, OutputTokens: evidence.OutputTokens}
	}
	section.PartGenerationRuns = partRuns
	section.GenerationRun = &BookGenerationRun{Date: sectionTime, Model: assembled.Provenance.ResolvedModelRef, Provider: assembled.Provenance.Provider, PromptSHA256: assembled.Provenance.PromptSHA256, OutputSHA256: hash([]byte(assembled.Text)), SectionID: section.ID, CandidateID: assembled.ID, Purpose: "canonical assembled manuscript section", WordCount: assembled.SectionAssembly.WordCount, InputTokens: assembled.Provenance.InputTokens, OutputTokens: assembled.Provenance.OutputTokens}
	section.CommittedCandidateID = assembled.ID
	doc := Document{ID: "positive-document", BookBrief: brief, Title: brief.Title, ProfileKey: profile.Key, SectionProfileVersion: fmt.Sprintf("%s@%d", profile.Key, profile.Version), SectionIDs: []string{section.ID}, Status: "assembled", CreatedAt: now.Add(-time.Minute), Outline: []OutlineSection{{Kind: "chapter", Intent: brief.ChapterIntent[0], Summary: section.Summary, TargetWords: bookChapterWordTarget}}, Sections: []Section{section}, AssembledText: assembled.Text}
	metrics := textmetrics.Analyze(assembled.Text, nil)
	doc.Provenance = DocumentProvenance{SectionCount: 1, WordCount: metrics.WordCount, TotalCostMicros: assembled.Provenance.CostMicros, InputTokens: assembled.Provenance.InputTokens, OutputTokens: assembled.Provenance.OutputTokens, Providers: []string{"ollama"}, Models: []string{"gemma4:12b"}}
	require.NoError(t, s.saveJSON(ctx, "prose_candidates", assembled.ID, assembled))
	require.NoError(t, s.saveJSON(ctx, "prose_rounds", assemblyRoundID, Round{ID: assemblyRoundID, SessionID: section.SessionID, CompositionMode: bookSectionChunkVersion, Strategy: Sampler{Kind: "composed", K: 1}, CandidateIDs: []string{assembled.ID}}))
	require.NoError(t, s.saveJSON(ctx, "prose_sessions", section.SessionID, Session{ID: section.SessionID, ProfileKey: profile.Key, Status: "active", RoundIDs: roundIDs}))
	require.NoError(t, s.saveSection(ctx, section))
	require.NoError(t, s.saveJSON(ctx, "prose_documents", doc.ID, doc))
	require.GreaterOrEqual(t, metrics.WordCount, 440)
	require.LessOrEqual(t, metrics.WordCount, 660)
	return s, doc, assembled.Text
}

func createExportTestProfile(t *testing.T, s *Service, key string, version int) {
	t.Helper()
	_, err := s.CreateProfile(context.Background(), Profile{Key: key, Version: version, GatewayRole: "write.default", Locality: "local_only", Sampler: Sampler{Kind: samplerDirect, K: 1}, Budget: Budget{MaxOutputTokens: 2048}, Composition: CompositionPolicy{TargetSectionWords: bookChapterWordTarget, SectionWordTolerance: 0.2, SectionFormat: "rich_prose", SectionMaxOutputTokens: 2048, MaxSections: 16}})
	require.NoError(t, err)
}

func splitProfileVersion(t *testing.T, ref string) (string, int) {
	t.Helper()
	index := strings.LastIndex(ref, "@")
	require.Greater(t, index, 0)
	version := 0
	_, err := fmt.Sscanf(ref[index+1:], "%d", &version)
	require.NoError(t, err)
	return ref[:index], version
}

func loadCanonicalCandidate(t *testing.T, s *Service, doc Document) Candidate {
	t.Helper()
	candidate, err := s.loadCandidate(context.Background(), doc.Sections[0].CommittedCandidateID)
	require.NoError(t, err)
	return candidate
}

func saveDocumentAndSections(t *testing.T, s *Service, doc Document) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, s.saveSection(ctx, doc.Sections[0]))
	require.NoError(t, s.saveJSON(ctx, "prose_documents", doc.ID, doc))
}

func rewriteStoredPartText(t *testing.T, s *Service, doc Document, index int, text string) {
	t.Helper()
	ctx := context.Background()
	canonical := loadCanonicalCandidate(t, s, doc)
	section := doc.Sections[0]
	parts := make([]Candidate, len(canonical.DerivedFrom))
	var combined strings.Builder
	var inputTokens, outputTokens int
	for i, id := range canonical.DerivedFrom {
		part, err := s.loadCandidate(ctx, id)
		require.NoError(t, err)
		if i == index {
			part.Text = text
			part.Measurements = textmetrics.Analyze(text, nil)
			part.Provenance.OutputTokens = len(strings.Fields(text)) + 8
		}
		if combined.Len() > 0 {
			combined.WriteString("\n\n")
		}
		combined.WriteString(strings.TrimSpace(part.Text))
		inputTokens += part.Provenance.InputTokens
		outputTokens += part.Provenance.OutputTokens
		parts[i] = part
		require.NoError(t, s.saveJSON(ctx, "prose_candidates", id, part))
		metrics := textmetrics.Analyze(strings.TrimSpace(part.Text), nil)
		evidence := &canonical.SectionAssembly.Parts[i]
		evidence.WordCount = metrics.WordCount
		evidence.OutputTokens = part.Provenance.OutputTokens
		evidence.OutputSHA256 = hash([]byte(strings.TrimSpace(part.Text)))
		run := &section.PartGenerationRuns[i]
		run.WordCount = metrics.WordCount
		run.OutputTokens = part.Provenance.OutputTokens
		run.OutputSHA256 = evidence.OutputSHA256
	}
	canonical.Text = combined.String()
	canonical.Measurements = textmetrics.Analyze(canonical.Text, nil)
	canonical.SectionAssembly.WordCount = textmetrics.Analyze(canonical.Text, nil).WordCount
	canonical.Provenance.InputTokens = inputTokens
	canonical.Provenance.OutputTokens = outputTokens
	section.CommittedCandidateID = canonical.ID
	section.GenerationRun.WordCount = canonical.SectionAssembly.WordCount
	section.GenerationRun.OutputTokens = outputTokens
	section.GenerationRun.OutputSHA256 = hash([]byte(canonical.Text))
	doc.Sections[0] = section
	doc.AssembledText = canonical.Text
	doc.Provenance.WordCount = canonical.SectionAssembly.WordCount
	doc.Provenance.InputTokens = inputTokens
	doc.Provenance.OutputTokens = outputTokens
	require.NoError(t, s.saveJSON(ctx, "prose_candidates", canonical.ID, canonical))
	saveDocumentAndSections(t, s, doc)
}
