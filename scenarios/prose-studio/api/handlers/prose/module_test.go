package prose

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose"
	_ "modernc.org/sqlite"
	internal "prose-studio/internal/prose"
)

type applicationFixtureGateway struct{ requests []internal.GatewayRequest }

func (g *applicationFixtureGateway) Generate(_ context.Context, request internal.GatewayRequest) ([]internal.GatewayCandidate, error) {
	g.requests = append(g.requests, request)
	return []internal.GatewayCandidate{{
		Text:     `{"sentences":[{"text":"I led a synthetic migration review with two partner teams.","source_ids":["fixture:FACT-02","fixture:REQ-02"]}]}`,
		Provider: "fixture-only", Model: "fixture-only",
	}}, nil
}

// This test exercises the registered REST route with an in-memory, local_only
// profile and fake inference seam. Set J11_PROSE_RESPONSE_PATH to retain its
// fixture response in a private research run folder for the document packet.
func TestApplicationDocumentRESTUsesProtoJSONAndPrivateLocalRoute(t *testing.T) {
	db, err := sql.Open("sqlite", "file:j11-application-document?mode=memory&cache=shared")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, internal.EnsureSchema(db))

	gateway := &applicationFixtureGateway{}
	service := internal.NewWithGateway(db, gateway)
	_, err = service.CreateProfile(context.Background(), internal.Profile{
		Key: "fixture/local-only", Locality: "local_only", GatewayRole: "write.default",
		Sampler:         internal.Sampler{Kind: "direct", K: 1, TemperatureStance: "ignored", MaxOutputTokens: 128},
		SelectionPolicy: "take_first", Budget: internal.Budget{MaxOutputTokens: 128},
	})
	require.NoError(t, err)
	_, err = service.CreateProfile(context.Background(), internal.Profile{
		Key: "fixture/not-local", Locality: "local_first", GatewayRole: "write.default",
		Sampler:         internal.Sampler{Kind: "direct", K: 1, TemperatureStance: "ignored", MaxOutputTokens: 128},
		SelectionPolicy: "take_first", Budget: internal.Budget{MaxOutputTokens: 128},
	})
	require.NoError(t, err)

	router := mux.NewRouter()
	module := Module(service)
	module.Mount(router)
	body := `{"contract_version":1,"kind":"APPLICATION_DOCUMENT_KIND_SHORT_ANSWER","profile_key":"fixture/local-only","instruction":"Draft one synthetic answer and cite every source.","sources":[{"id":"fixture:FACT-02","kind":"APPLICATION_SOURCE_KIND_CONFIRMED_FACT","literal":"Led a synthetic migration review with two partner teams.","confirmed":true,"confirmation_ref":"fixture:confirmed:FACT-02"},{"id":"fixture:REQ-02","kind":"APPLICATION_SOURCE_KIND_LISTING_EVIDENCE","literal":"Describe a cross-team engineering decision.","confirmed":true,"confirmation_ref":"fixture:confirmed:REQ-02"}]}`
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/prose/generate/application-document", strings.NewReader(body)))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Len(t, gateway.requests, 1)
	require.Equal(t, "local_only", gateway.requests[0].Locality)

	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "APPLICATION_DOCUMENT_KIND_SHORT_ANSWER", response["kind"])
	require.Equal(t, "draft", response["status"])
	require.Equal(t, float64(1), response["contract_version"])
	sentences, ok := response["sentences"].([]any)
	require.True(t, ok)
	require.Len(t, sentences, 1)
	first := sentences[0].(map[string]any)
	require.Equal(t, []any{"fixture:FACT-02", "fixture:REQ-02"}, first["source_ids"])

	for _, table := range []string{"prose_sessions", "prose_rounds", "prose_candidates", "prose_selection_events"} {
		var count int
		require.NoError(t, db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count))
		require.Zero(t, count, "private application generation must not persist %s", table)
	}
	if outputPath := os.Getenv("J11_PROSE_RESPONSE_PATH"); outputPath != "" {
		require.NoError(t, os.WriteFile(outputPath, recorder.Body.Bytes(), 0o600))
	}

	coverLetter := `{"contract_version":1,"kind":"APPLICATION_DOCUMENT_KIND_COVER_LETTER","profile_key":"fixture/local-only","instruction":"Draft a synthetic cover letter.","sources":[{"id":"fixture:FACT-03","kind":"APPLICATION_SOURCE_KIND_CONFIRMED_FACT","literal":"Unverified synthetic claim.","confirmed":false,"confirmation_ref":"fixture:missing"}]}`
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/prose/generate/application-document", strings.NewReader(coverLetter)))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var refusedCover map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &refusedCover))
	require.Equal(t, "refused", refusedCover["status"])
	require.Empty(t, refusedCover["sentences"])

	nonLocal := strings.Replace(body, "fixture/local-only", "fixture/not-local", 1)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/prose/generate/application-document", strings.NewReader(nonLocal)))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var refusedNonLocal map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &refusedNonLocal))
	require.Equal(t, "refused", refusedNonLocal["status"])
	require.Empty(t, refusedNonLocal["sentences"])
	require.Len(t, gateway.requests, 1, "refused requests must not reach inference")
}

func TestGenerateDecodePreservesPersistenceMode(t *testing.T) {
	var request internal.GenerateRequest
	err := decodeMessage(&v1.GenerateRequest{
		ProfileKey:            "synthetic-local",
		Query:                 "SYNTHETIC_PRIVATE_MARKER",
		PersistenceMode:       "private_ephemeral",
		SupportedClaimPhrases: []string{"I contributed to 3 shipped features."},
	}, &request)
	require.NoError(t, err)
	require.Equal(t, "private_ephemeral", request.PersistenceMode)
	require.Equal(t, "SYNTHETIC_PRIVATE_MARKER", request.Query)
	require.Equal(t, []string{"I contributed to 3 shipped features."}, request.SupportedClaimPhrases)
}

func TestCreateDocumentProtoPreservesBookBrief(t *testing.T) {
	var in struct {
		Document internal.Document `json:"document"`
	}
	err := decodeMessage(&v1.CreateDocumentRequest{Document: &v1.Document{Title: "Brief title", BookBrief: &v1.BookBrief{
		SchemaVersion: "book-brief/v1", Title: "Brief title", CreativeDirection: "Synthetic direction", Audience: "Synthetic audience",
		ReaderPromise: "Synthetic promise", Structure: "Braided vignettes", ChapterIntent: []string{"First", "Second"},
		FrontMatterIntent: []string{"Invitation"}, BackMatterIntent: []string{"Glossary"}, AdditionalDirection: "Invented content only",
		EvidenceBoundary:   &v1.BookEvidenceBoundary{ApprovedFacts: []string{"A synthetic observation is recorded."}, UnknownDetails: []string{"time", "location", "source_identity"}},
		ProductionMetadata: &v1.BookProductionMetadata{Subtitle: "Synthetic subtitle", Language: "en", TrimWidthIn: 6, TrimHeightIn: 9, MarginIn: 0.6, ContentOrigin: "synthetic", GeneratedWithAi: true},
	}}}, &in)
	require.NoError(t, err)
	require.Equal(t, "book-brief/v1", in.Document.BookBrief.SchemaVersion)
	require.Equal(t, "Synthetic direction", in.Document.BookBrief.CreativeDirection)
	require.Equal(t, []string{"First", "Second"}, in.Document.BookBrief.ChapterIntent)
	require.Equal(t, []string{"Invitation"}, in.Document.BookBrief.FrontMatterIntent)
	require.Equal(t, []string{"A synthetic observation is recorded."}, in.Document.BookBrief.EvidenceBoundary.ApprovedFacts)
	require.Equal(t, []string{"time", "location", "source_identity"}, in.Document.BookBrief.EvidenceBoundary.UnknownDetails)
	require.Equal(t, "Synthetic subtitle", in.Document.BookBrief.ProductionMetadata.Subtitle)
}

func TestManuscriptSourceProtoPreservesChunkGenerationEvidence(t *testing.T) {
	source := internal.ManuscriptSource{ProductionMetadata: &internal.BookProductionMetadata{
		GeneratedWithAI: true,
		GenerationRuns: []internal.BookGenerationRun{{
			Date: "2026-11-24T00:00:00Z", Model: "gemma4:12b", Provider: "ollama",
			PromptSHA256: strings.Repeat("a", 64), OutputSHA256: strings.Repeat("b", 64),
			SectionID: "synthetic-section", CandidateID: "synthetic-part-candidate", PartIndex: 2,
			Purpose: "distinct synthetic part purpose", WordCount: 92, InputTokens: 700, OutputTokens: 101,
		}},
	}}
	var encoded v1.ManuscriptSource
	require.NoError(t, encodeResponse(source, &encoded))
	require.Len(t, encoded.GetProductionMetadata().GetGenerationRuns(), 1)
	run := encoded.GetProductionMetadata().GetGenerationRuns()[0]
	require.Equal(t, "ollama", run.GetProvider())
	require.Equal(t, "synthetic-section", run.GetSectionId())
	require.Equal(t, "synthetic-part-candidate", run.GetCandidateId())
	require.Equal(t, int32(2), run.GetPartIndex())
	require.Equal(t, "distinct synthetic part purpose", run.GetPurpose())
	require.Equal(t, int32(92), run.GetWordCount())
	require.Equal(t, int32(700), run.GetInputTokens())
	require.Equal(t, int32(101), run.GetOutputTokens())
}

func TestBookCreateReadExportRoundTripUsesDurableBriefAndAuthoredMatter(t *testing.T) {
	db, err := sql.Open("sqlite", "file:book-brief-round-trip?mode=memory&cache=shared")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, internal.EnsureSchema(db))
	gateway := &applicationFixtureGateway{}
	h := &handler{service: internal.NewWithGateway(db, gateway)}

	brief := &v1.BookBrief{SchemaVersion: "book-brief/v1", Title: "Synthetic Round Trip", CreativeDirection: "Quiet, observant prose", Audience: "Synthetic readers", ReaderPromise: "Notice details", Structure: "Twelve ordered slots", FrontMatterIntent: []string{"Preface", "How to use this book", "Dedication"}, ChapterIntent: []string{"Chapter 1", "Chapter 2", "Chapter 3", "Chapter 4", "Chapter 5", "Chapter 6"}, BackMatterIntent: []string{"Acknowledgements", "Glossary", "Further reading"}, EvidenceBoundary: &v1.BookEvidenceBoundary{ApprovedFacts: []string{"Fixture fact"}, UnknownDetails: []string{"time", "location", "source_identity"}}, ProductionMetadata: &v1.BookProductionMetadata{Subtitle: "Synthetic subtitle", Language: "en", TrimWidthIn: 6, TrimHeightIn: 9, MarginIn: 0.6, ContentOrigin: "synthetic", GeneratedWithAi: false}}
	kinds := []string{"front_matter", "front_matter", "front_matter", "chapter", "chapter", "chapter", "chapter", "chapter", "chapter", "back_matter", "back_matter", "back_matter"}
	intents := []string{"Preface", "How to use this book", "Dedication", "Chapter 1", "Chapter 2", "Chapter 3", "Chapter 4", "Chapter 5", "Chapter 6", "Acknowledgements", "Glossary", "Further reading"}
	sections := make([]*v1.Section, 0, len(kinds))
	for i := range kinds {
		candidate := internal.Candidate{ID: fmt.Sprintf("candidate-%d", i), Text: fmt.Sprintf("Synthetic authored section %d.", i), Committed: true}
		payload, err := json.Marshal(candidate)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO prose_candidates(id,payload) VALUES(?,?)`, candidate.ID, string(payload))
		require.NoError(t, err)
		sections = append(sections, &v1.Section{Id: fmt.Sprintf("section-%d", i), Kind: kinds[i], Position: int32(i), Intent: intents[i], CommittedCandidateId: candidate.ID})
	}
	created, err := h.CreateDocument(context.Background(), connect.NewRequest(&v1.CreateDocumentRequest{Document: &v1.Document{Title: "stale title", BookBrief: brief}, Sections: sections}))
	require.NoError(t, err)
	require.Equal(t, brief.Title, created.Msg.GetDocument().GetTitle())
	require.Equal(t, brief.GetReaderPromise(), created.Msg.GetDocument().GetBookBrief().GetReaderPromise())
	require.Equal(t, "Synthetic subtitle", created.Msg.GetDocument().GetBookBrief().GetProductionMetadata().GetSubtitle())
	require.Equal(t, []string{"Fixture fact"}, created.Msg.GetDocument().GetBookBrief().GetEvidenceBoundary().GetApprovedFacts())

	read, err := h.GetDocument(context.Background(), connect.NewRequest(&v1.GetDocumentRequest{Id: created.Msg.GetDocument().GetId()}))
	require.NoError(t, err)
	require.Equal(t, brief.GetTitle(), read.Msg.GetDocument().GetBookBrief().GetTitle())
	require.Equal(t, []string{"time", "location", "source_identity"}, read.Msg.GetDocument().GetBookBrief().GetEvidenceBoundary().GetUnknownDetails())
	require.Len(t, read.Msg.GetDocument().GetSections(), 12)
	require.Equal(t, "back_matter", read.Msg.GetDocument().GetSections()[11].GetKind())

	exported, err := h.ExportBookSource(context.Background(), connect.NewRequest(&v1.ExportBookSourceRequest{Id: created.Msg.GetDocument().GetId()}))
	require.NoError(t, err)
	require.Equal(t, "manuscript-source/v1", exported.Msg.GetSource().GetSchemaVersion())
	require.Equal(t, "Synthetic subtitle", exported.Msg.GetSource().GetProductionMetadata().GetSubtitle())
	require.Len(t, exported.Msg.GetSource().GetFrontMatter(), 3)
	require.Len(t, exported.Msg.GetSource().GetChapters(), 6)
	require.Len(t, exported.Msg.GetSource().GetBackMatter(), 3)
	require.Len(t, exported.Msg.GetSource().GetReadingOrder(), 12)
	require.Equal(t, "candidate-0", exported.Msg.GetSource().GetReadingOrder()[0].GetCandidateId())
	require.Equal(t, "candidate-11", exported.Msg.GetSource().GetReadingOrder()[11].GetCandidateId())
	require.Empty(t, gateway.requests, "create/read/export must not call inference")
}

func TestApplicationDocumentContractRefusesUnconfirmedOrMissingSources(t *testing.T) {
	h := &handler{}
	kind := v1.ApplicationDocumentKind_APPLICATION_DOCUMENT_KIND_COVER_LETTER
	response, err := h.GenerateApplicationDocument(context.Background(), connect.NewRequest(&v1.GenerateApplicationDocumentRequest{
		ContractVersion: 1, Kind: kind, ProfileKey: "synthetic-local", Instruction: "Draft a short letter",
		Sources: []*v1.ApplicationSource{{Id: "fact-1", Kind: v1.ApplicationSourceKind_APPLICATION_SOURCE_KIND_CONFIRMED_FACT, Literal: "Synthetic fact", Confirmed: false}},
	}))
	require.NoError(t, err)
	require.Equal(t, "refused", response.Msg.GetStatus())
	require.Empty(t, response.Msg.GetSentences())
	response, err = h.GenerateApplicationDocument(context.Background(), connect.NewRequest(&v1.GenerateApplicationDocumentRequest{
		ContractVersion: 1, Kind: kind, ProfileKey: "synthetic-local", Instruction: "Draft a short letter",
	}))
	require.NoError(t, err)
	require.Equal(t, "refused", response.Msg.GetStatus())
	require.Contains(t, response.Msg.GetRefusalReason(), "no confirmed sources")
}

func TestApplicationDocumentContractRejectsUnknownVersion(t *testing.T) {
	h := &handler{}
	_, err := h.GenerateApplicationDocument(context.Background(), connect.NewRequest(&v1.GenerateApplicationDocumentRequest{
		ContractVersion: 2, Kind: v1.ApplicationDocumentKind_APPLICATION_DOCUMENT_KIND_SHORT_ANSWER,
		ProfileKey: "synthetic-local", Instruction: "Answer the question",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestApplicationDocumentOutputRequiresDeclaredSourceReferences(t *testing.T) {
	sources := map[string]*v1.ApplicationSource{
		"fact-1": {Id: "fact-1", Kind: v1.ApplicationSourceKind_APPLICATION_SOURCE_KIND_CONFIRMED_FACT, Literal: "Synthetic fact", Confirmed: true, ConfirmationRef: "fixture-receipt-1"},
	}
	sentences, reason := validateApplicationOutput(&internal.Candidate{Text: `{"sentences":[{"text":"I did the work.","source_ids":["other-fact"]}]}`}, sources)
	require.Empty(t, sentences)
	require.Contains(t, reason, "absent")
	sentences, reason = validateApplicationOutput(&internal.Candidate{Text: `{"sentences":[{"text":"I did the work.","source_ids":["fact-1"]}]}`}, sources)
	require.Empty(t, reason)
	require.Len(t, sentences, 1)
	require.Equal(t, []string{"fact-1"}, sentences[0].GetSourceIds())
}

func TestProfileDecodePreservesExactClaimPhrasePolicy(t *testing.T) {
	var profile internal.Profile
	err := decodeMessage(&v1.Profile{Key: "synthetic-claim-only", Constraints: &v1.Constraints{RequireExactClaimPhrases: true}}, &profile)
	require.NoError(t, err)
	require.True(t, profile.Constraints.RequireExactClaimPhrases)
}
