package taste

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	tastev1 "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/taste"
	tastev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/taste/taste_v1connect"
	"music-library/internal/catalog"
	"music-library/internal/decomposition"
	"music-library/internal/preference"
)

func TestTasteConnectRoundTripPersistsEvidenceAndReturnsHonestScores(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(catalog.Schema),
		apidb.SchemaProviderFunc(decomposition.Schema),
		apidb.SchemaProviderFunc(preference.Schema),
	))
	catalogStore := catalog.NewStore(db)
	leftID, rightID := strings.Repeat("a", 64), strings.Repeat("b", 64)
	_, err := catalogStore.Put(context.Background(), catalog.Track{SHA256: leftID, Title: "Bright Pulse", SourcePath: "/music/bright.wav", Tags: []string{"sparkly", "syncopated"}, Approval: "liked"})
	require.NoError(t, err)
	_, err = catalogStore.Put(context.Background(), catalog.Track{SHA256: rightID, Title: "Warm Drift", SourcePath: "/music/warm.wav", Tags: []string{"warm", "slow"}, Approval: "unreviewed"})
	require.NoError(t, err)
	decompositionStore := decomposition.NewStore(db)
	putDecomposition(t, decompositionStore, leftID, "Artist North")

	_, serviceHandler := tastev1connect.NewTasteServiceHandler(&handler{store: preference.NewStore(db), catalog: catalogStore, decompositions: decompositionStore})
	srv := httptest.NewServer(serviceHandler)
	defer srv.Close()
	client := tastev1connect.NewTasteServiceClient(srv.Client(), srv.URL)
	ctx := context.Background()

	selected, err := client.SelectPair(ctx, connect.NewRequest(&tastev1.SelectPairRequest{}))
	require.NoError(t, err)
	require.NotNil(t, selected.Msg.Left)
	require.NotEqual(t, selected.Msg.Left.TrackSha256, selected.Msg.Right.TrackSha256)
	require.Greater(t, selected.Msg.ExpectedInformation, 0.0)

	controlled, err := client.SetControl(ctx, connect.NewRequest(&tastev1.SetControlRequest{Control: &tastev1.TasteControl{Attribute: "tag:sparkly", Mode: "boost", Value: 0.4}}))
	require.NoError(t, err)
	require.Equal(t, int64(1), controlled.Msg.Profile.Revision)
	require.Len(t, controlled.Msg.Profile.Controls, 1)

	judged, err := client.RecordJudgment(ctx, connect.NewRequest(&tastev1.RecordJudgmentRequest{LeftTrackSha256: leftID, RightTrackSha256: rightID, Choice: "left", Reason: "the groove catches me"}))
	require.NoError(t, err)
	require.Equal(t, int64(2), judged.Msg.Profile.Revision)
	require.Len(t, judged.Msg.Profile.Judgments, 1)
	require.Equal(t, "operator", judged.Msg.Profile.Judgments[0].Source)
	require.Equal(t, leftID, judged.Msg.Profile.Judgments[0].LeftTrackSha256)

	read, err := client.GetProfile(ctx, connect.NewRequest(&tastev1.GetProfileRequest{}))
	require.NoError(t, err)
	require.Equal(t, judged.Msg.Profile.Revision, read.Msg.Profile.Revision)
	require.Len(t, read.Msg.Profile.Judgments, 1)
	var learned *tastev1.TasteWeight
	for _, weight := range read.Msg.Profile.Weights {
		if weight.Attribute == "tag:sparkly" {
			learned = weight
		}
	}
	require.NotNil(t, learned)
	require.Greater(t, learned.Mean, 0.0)

	recommendations, err := client.Recommend(ctx, connect.NewRequest(&tastev1.RecommendRequest{SeedTrackSha256: rightID, Limit: 10}))
	require.NoError(t, err)
	require.Equal(t, rightID, recommendations.Msg.Seed.TrackSha256)
	require.Len(t, recommendations.Msg.Recommendations, 1, "the API must report the real candidate count instead of padding to ten")
	item := recommendations.Msg.Recommendations[0]
	require.Equal(t, leftID, item.Candidate.TrackSha256)
	require.Equal(t, "Artist North", item.Candidate.Artist, "artist is joined from hash-matched decomposition source metadata")
	require.Equal(t, "operator_pairwise_evidence", recommendations.Msg.EvidenceState)
	require.Contains(t, recommendations.Msg.Mq2Brief, rightID)
	var reconstructed float64
	for _, part := range item.Contributions {
		reconstructed += part.Contribution
		require.NotEmpty(t, part.EvidenceSource)
	}
	require.InDelta(t, item.PredictedPreference, reconstructed, 1e-9)
	require.NotEmpty(t, recommendations.Msg.Provenance)
}

func TestTasteReturnsTenArtistAndGenreDiverseRecommendations(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(catalog.Schema),
		apidb.SchemaProviderFunc(decomposition.Schema),
		apidb.SchemaProviderFunc(preference.Schema),
	))
	catalogStore := catalog.NewStore(db)
	decompositionStore := decomposition.NewStore(db)
	seedID := strings.Repeat("a", 64)
	seed := catalog.Track{SHA256: seedID, Title: "Starting Point", SourcePath: "/music/seed.wav", Tags: []string{"genre:ambient", "tag:seed-texture"}, Approval: "liked"}
	_, err := catalogStore.Put(context.Background(), seed)
	require.NoError(t, err)
	putDecomposition(t, decompositionStore, seedID, "Seed Artist")
	genres := []string{"ambient", "ambient", "house", "house", "jazz", "jazz", "folk", "folk", "cinematic", "cinematic"}
	ids := make([]string, 0, len(genres))
	for i, genre := range genres {
		id := fmt.Sprintf("%064x", i+11)
		ids = append(ids, id)
		track := catalog.Track{SHA256: id, Title: fmt.Sprintf("Catalog Choice %02d", i+1), SourcePath: fmt.Sprintf("/music/choice-%02d.wav", i+1), Tags: []string{"genre:" + genre, fmt.Sprintf("texture-%02d", i+1)}, Approval: "unreviewed"}
		if i == 0 {
			track.Tags = append(track.Tags, "bright")
		}
		_, err := catalogStore.Put(context.Background(), track)
		require.NoError(t, err)
		putDecomposition(t, decompositionStore, id, fmt.Sprintf("Artist %02d", i+1))
	}
	h := &handler{store: preference.NewStore(db), catalog: catalogStore, decompositions: decompositionStore}
	loaded, err := h.candidates(context.Background(), includeRejectedTracks)
	require.NoError(t, err)
	var acousticSeen bool
	for _, candidate := range loaded {
		if candidate.ID != ids[0] {
			continue
		}
		for _, feature := range candidate.Features {
			if feature.Attribute == "acoustic:tempo_log2_120bpm" {
				acousticSeen = true
				require.Equal(t, "decomposition:"+ids[0]+":job-"+ids[0][56:], feature.Source)
				require.Equal(t, 1.0, feature.Confidence, "only explicitly confirmed analysis is projected")
			}
		}
	}
	require.True(t, acousticSeen, "confirmed decomposition features should reach the recommendation feature space")
	_, err = h.store.SetControl(context.Background(), "mq2-test", preference.Control{Attribute: "tag:bright", Mode: preference.ControlBoost, Value: 0.7})
	require.NoError(t, err)
	_, err = h.store.SetControl(context.Background(), "mq2-test", preference.Control{Attribute: "acoustic:tempo_log2_120bpm", Mode: preference.ControlPin, Value: -0.7})
	require.NoError(t, err)

	_, serviceHandler := tastev1connect.NewTasteServiceHandler(h)
	srv := httptest.NewServer(serviceHandler)
	defer srv.Close()
	client := tastev1connect.NewTasteServiceClient(srv.Client(), srv.URL)
	response, err := client.Recommend(context.Background(), connect.NewRequest(&tastev1.RecommendRequest{Context: "mq2-test", SeedTrackSha256: seedID, Limit: 10}))
	require.NoError(t, err)
	require.Len(t, response.Msg.Recommendations, 10)
	seenArtists, seenGenres := map[string]bool{}, map[string]bool{}
	genreCounts := map[string]int{}
	previousSelectionScore := 1e9
	for _, item := range response.Msg.Recommendations {
		require.NotEqual(t, "Seed Artist", item.Candidate.Artist)
		require.False(t, seenArtists[item.Candidate.Artist], "diversifier should keep one result per artist when enough artists exist")
		seenArtists[item.Candidate.Artist] = true
		require.NotEmpty(t, item.Candidate.Genre)
		seenGenres[item.Candidate.Genre] = true
		genreCounts[item.Candidate.Genre]++
		require.LessOrEqual(t, genreCounts[item.Candidate.Genre], 2)
		require.LessOrEqual(t, item.SelectionScore, previousSelectionScore, "the diverse result preserves score order")
		previousSelectionScore = item.SelectionScore
		var reconstructed float64
		var acousticContribution bool
		for _, contribution := range item.Contributions {
			reconstructed += contribution.Contribution
			if strings.HasPrefix(contribution.Attribute, "acoustic:") || strings.HasPrefix(contribution.Attribute, "key:") {
				require.Equal(t, "decomposition:"+item.Candidate.TrackSha256+":job-"+item.Candidate.TrackSha256[56:], contribution.EvidenceSource)
				if contribution.Attribute == "acoustic:tempo_log2_120bpm" {
					acousticContribution = true
					require.Greater(t, contribution.Contribution, 0.0)
					require.Equal(t, 1.0, contribution.FeatureConfidence)
				}
			} else {
				require.Equal(t, "catalog:"+item.Candidate.TrackSha256, contribution.EvidenceSource)
			}
		}
		require.True(t, acousticContribution, "the explicit acoustic control should flow through the score explanation")
		require.InDelta(t, item.PredictedPreference, reconstructed, 1e-9)
	}
	require.GreaterOrEqual(t, len(seenGenres), 5)
	require.Contains(t, response.Msg.ChosenFeatures, "tag:bright")
	require.Contains(t, response.Msg.ChosenFeatures, "acoustic:tempo_log2_120bpm")
	require.Contains(t, response.Msg.Mq2Brief, seedID)
	require.Contains(t, response.Msg.Mq2Brief, "tag:bright")
	require.Contains(t, response.Msg.Mq2Brief, "mq2-test")
	require.Contains(t, response.Msg.Mq2Brief, "uncertainty=")
	require.Contains(t, response.Msg.Mq2Brief, "confidence=")
	require.Contains(t, response.Msg.Mq2Brief, "evidence=decomposition:")
	require.Contains(t, response.Msg.Mq2Brief, "score_feature=acoustic:tempo_log2_120bpm")
	require.Contains(t, response.Msg.Mq2Brief, "clamp(log2(BPM/120)/log2(300/120), -1, 1)")
	for _, id := range ids {
		require.Contains(t, response.Msg.Provenance, "catalog:"+id)
		require.Contains(t, response.Msg.Provenance, "decomposition-source:"+id)
		require.Contains(t, response.Msg.Mq2Brief, id)
	}
}

func TestAcousticTasteProjectionRequiresLatestConfirmedReview(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(decomposition.Schema)))
	store := decomposition.NewStore(db)
	id := strings.Repeat("c", 64)
	putDecomposition(t, store, id, "Reviewed Artist")
	features, err := store.ListTasteFeatures(context.Background(), []string{id})
	require.NoError(t, err)
	require.Len(t, features[id], 3)
	byAttribute := map[string]decomposition.TasteFeature{}
	for _, feature := range features[id] {
		byAttribute[feature.Attribute] = feature
		require.Equal(t, "decomposition:"+id+":job-"+id[:8], feature.Source)
	}
	require.InDelta(t, 1, byAttribute["key:d_major"].Value, 1e-9)
	require.InDelta(t, 1, byAttribute["acoustic:duration_log2_180s"].Confidence, 1e-9)

	_, err = store.Review(context.Background(), decomposition.Review{ID: "needs-work-" + id[56:], TrackSHA256: id, AnalysisJob: "job-" + id[56:], Verdict: "needs_work"})
	require.NoError(t, err)
	features, err = store.ListTasteFeatures(context.Background(), []string{id})
	require.NoError(t, err)
	require.Empty(t, features[id], "a later review that withdraws confirmation hides the projection")

	_, err = store.Review(context.Background(), decomposition.Review{ID: "confirmed-again-" + id[56:], TrackSHA256: id, AnalysisJob: "job-" + id[56:], Verdict: "confirmed"})
	require.NoError(t, err)
	features, err = store.ListTasteFeatures(context.Background(), []string{id})
	require.NoError(t, err)
	require.Len(t, features[id], 3, "a later explicit confirmation restores exact job-bound projections")

	newJob := "job-newer"
	raw := json.RawMessage(fmt.Sprintf(`{"job_id":%q,"source_sha256":%q,"resource":"music-mir","operation":"analyze","analysis":{"source_sha256":%q,"media":{"duration_seconds":210},"beat":{"tempo_bpm_estimate":128},"key":{"primary_estimate":"F major"}}}`, newJob, id, id))
	_, err = store.Put(context.Background(), decomposition.Result{TrackSHA256: id, SourceSHA256: id, AnalysisJob: newJob, ToolResult: raw})
	require.NoError(t, err)
	features, err = store.ListTasteFeatures(context.Background(), []string{id})
	require.NoError(t, err)
	require.Empty(t, features[id], "an unreviewed newer decomposition makes an older projection stale")
	_, err = store.Review(context.Background(), decomposition.Review{ID: "confirm-newer-" + id[56:], TrackSHA256: id, AnalysisJob: newJob, Verdict: "confirmed"})
	require.NoError(t, err)
	features, err = store.ListTasteFeatures(context.Background(), []string{id})
	require.NoError(t, err)
	require.Len(t, features[id], 3)
	for _, feature := range features[id] {
		require.Equal(t, "decomposition:"+id+":"+newJob, feature.Source, "projection carries the newest exact analysis job")
	}
}

func putDecomposition(t *testing.T, store *decomposition.Store, id, artist string) {
	t.Helper()
	raw := json.RawMessage(fmt.Sprintf(`{"job_id":"job-%s","source_sha256":"%s","resource":"music-mir","operation":"analyze","analysis":{"schema_version":1,"status":"partial","source_sha256":"%s","media":{"duration_seconds":120},"beat":{"tempo_bpm_estimate":96},"key":{"primary_estimate":"D major"}}}`, id[56:], id, id))
	_, err := store.Put(context.Background(), decomposition.Result{
		TrackSHA256: id, SourceSHA256: id, AnalysisJob: "job-" + id[56:],
		Source:     decomposition.SourceMetadata{Title: "Fixture title", Artist: artist, FileName: "fixture.wav", MediaType: "audio/wav", SizeBytes: 1200},
		ToolResult: raw,
	})
	require.NoError(t, err)
	_, err = store.Review(context.Background(), decomposition.Review{ID: "review-" + id[56:], TrackSHA256: id, AnalysisJob: "job-" + id[56:], Verdict: "confirmed"})
	require.NoError(t, err)
}

func TestInformativeEvidenceExcludesSkipNeitherAndReversedRows(t *testing.T) {
	rows := []preference.Judgment{
		{ID: "old", Choice: preference.ChooseLeft},
		{ID: "skip", Choice: preference.ChooseSkip},
		{ID: "neither", Choice: preference.ChooseNeither},
		{ID: "tie", Choice: preference.ChooseTie},
		{ID: "correction", Choice: preference.ChooseRight, Reverses: "old"},
	}
	require.Equal(t, 2, informativeJudgmentCount(rows), "only the active correction and tie are informative")
}
