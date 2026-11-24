package prose

import (
	"context"
	"database/sql"
	"encoding/json"
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
