package releases_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	"music-library/handlers/releases"
	"music-library/internal/catalog"
	releasedomain "music-library/internal/releases"
)

func TestReleaseRESTCreatesImmutableDraftAndExposesBlockedDependencies(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(catalog.Schema),
		apidb.SchemaProviderFunc(releasedomain.Schema),
	))
	hashes := []string{strings.Repeat("1", 64), strings.Repeat("2", 64), strings.Repeat("3", 64), strings.Repeat("4", 64)}
	catalogStore := catalog.NewStore(db)
	for _, hash := range hashes {
		_, err := catalogStore.Put(context.Background(), catalog.Track{SHA256: hash, Title: "Instrumental", SourcePath: "/fixture/audio.wav", TakeID: "take", ReceiptPath: "receipt", Approval: "liked"})
		require.NoError(t, err)
	}
	router := releases.NewHandler(releasedomain.NewStore(db))
	edition := releasedomain.Edition{
		ReleaseID: "release-rest-test", Revision: 1, Kind: "ep", Title: "Future EP", DefaultLocale: "en",
		ArtistOwner: "subjects", ArtistSubjectID: "fictional-artist", ArtistSubjectVersion: "v1",
		LocalizedMetadata: map[string]releasedomain.LocalizedText{"en": {Title: "Future EP"}, "ja": {Title: "未来 EP"}},
	}
	for i, hash := range hashes {
		edition.Tracks = append(edition.Tracks, releasedomain.TrackOccurrence{Order: i + 1, TrackSHA256: hash})
	}
	body, err := json.Marshal(edition)
	require.NoError(t, err)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/releases", bytes.NewReader(body)))
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/releases/release-rest-test", nil))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var read struct {
		Edition     releasedomain.Edition     `json:"edition"`
		Eligibility releasedomain.Eligibility `json:"eligibility"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &read))
	require.Equal(t, int64(1), read.Edition.Revision)
	require.Len(t, read.Edition.Tracks, 4)
	require.False(t, read.Eligibility.ReadyForOperatorJudgment, "missing external owner evidence must be visible as a blocker")

	response = httptest.NewRecorder()
	rightsBody := map[string]any{"decision_id": "rights-rest", "revision": 1, "track_sha256": hashes[0], "source_license": "claimed-license", "usage_scope": "personal_only", "decision": "confirmed", "evidence_ref": "unrelated-receipt"}
	rightsJSON, err := json.Marshal(rightsBody)
	require.NoError(t, err)
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/releases/release-rest-test/rights-decisions", bytes.NewReader(rightsJSON)))
	require.Equal(t, http.StatusPreconditionFailed, response.Code, "draft metadata alone cannot stand in for rights and exact receipt provenance")

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/releases/release-rest-test/reviews", strings.NewReader("{\"review_id\":\"approval\",\"revision\":1,\"decision\":\"approved\"}")))
	require.Equal(t, http.StatusPreconditionFailed, response.Code, "operator review cannot bypass missing subject/artwork/rights/lineage evidence")
}

func TestReleaseRESTRejectsUnknownFieldsAndMismatchedReviewRoute(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(catalog.Schema),
		apidb.SchemaProviderFunc(releasedomain.Schema),
	))
	router := releases.NewHandler(releasedomain.NewStore(db))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/releases", strings.NewReader("{\"release_id\":\"x\",\"unexpected\":true}")))
	require.Equal(t, http.StatusBadRequest, response.Code)

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/releases/route-id/reviews", strings.NewReader("{\"release_id\":\"other-id\",\"review_id\":\"x\",\"revision\":1,\"decision\":\"approved\"}")))
	require.Equal(t, http.StatusBadRequest, response.Code)
}
