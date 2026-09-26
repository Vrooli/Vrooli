package sketch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
)

type critiqueDispatcher struct {
	captureDispatch
	url string
}

func (d *critiqueDispatcher) ResolveScreenshot(_ context.Context, op designcapture.Operation, a designcapture.Artifact) (designcapture.Screenshot, error) {
	return designcapture.Screenshot{Reference: a.Reference, URL: d.url, Width: 390, Height: 844, ContentType: "image/png"}, nil
}
func TestCritiqueRPCVerifiesEvidenceAndPreservesHistoricalRecord(t *testing.T) {
	ctx := apidb.WithTestMode(context.Background())
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(designcapture.Schema), apidb.SchemaProviderFunc(designcritique.Schema)))
	captures := designcapture.NewSQLiteRepository(db)
	critiques := designcritique.NewSQLiteRepository(db)
	html := "<main>fixture</main>"
	digest := sha256.Sum256([]byte(html))
	hash := hex.EncodeToString(digest[:])
	target := designcapture.Target{Scenario: "demo", DesignID: "home", Revision: strings.Repeat("a", 64), RenderHash: hash, HTMLSHA256: hash, InputsSHA256: hash, Kind: "preview", Kit: "test", Theme: "light", Direction: "ltr"}
	capture, err := captures.Create(ctx, "fixture", designcapture.Request{Target: target, HTML: html, Width: 390, Height: 844})
	require.NoError(t, err)
	capture, err = captures.Transition(ctx, capture.ID, capture.Version, designcapture.Dispatching, "", nil, "")
	require.NoError(t, err)
	capture, err = captures.Transition(ctx, capture.ID, capture.Version, designcapture.Running, "producer", nil, "")
	require.NoError(t, err)
	capture, err = captures.Transition(ctx, capture.ID, capture.Version, designcapture.Completed, "producer", []designcapture.Artifact{{Kind: "screenshot", Reference: "bas:producer:shot"}, {Kind: "target", Reference: "bas:producer:target", Evidence: &designcapture.TargetEvidence{RenderHash: hash, Width: 390, Height: 844}}}, "")
	require.NoError(t, err)
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 390, 844))))
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; _, _ = w.Write(encoded.Bytes()) }))
	defer server.Close()
	h := NewConnectHandler(Deps{CritiquesFor: func(c context.Context) (designcritique.Repository, error) {
		require.True(t, apidb.IsTestMode(c))
		return critiques, nil
	}, CapturesFor: func(c context.Context) (designcapture.Repository, error) {
		require.True(t, apidb.IsTestMode(c))
		return captures, nil
	}, CaptureDispatcher: &critiqueDispatcher{url: server.URL}})
	rubric, err := h.GetCritiqueRubric(ctx, connect.NewRequest(&sketchv1.GetCritiqueRubricRequest{}))
	require.NoError(t, err)
	require.False(t, rubric.Msg.Calibrated)
	require.Len(t, rubric.Msg.Dimensions, 8)
	require.Len(t, rubric.Msg.Anchors, 5)
	review := designcritique.Review{Target: designcritique.Target{Scenario: target.Scenario, DesignID: target.DesignID, Revision: target.Revision, RenderHash: hash}, RubricVersion: designcritique.RubricVersion, PolicyVersion: designcritique.PolicyVersion, Critic: designcritique.Critic{Kind: "model", ID: "test-critic", Version: "1", Model: "fixture", Profile: "unit-test"}}
	for _, d := range designcritique.Dimensions() {
		review.Ratings = append(review.Ratings, designcritique.Rating{Dimension: d, Score: 3, Rationale: "Fixture rating", Evidence: []designcritique.Evidence{{RenderHash: hash, CaptureID: capture.ID, Artifact: "bas:producer:shot", Region: "$page", State: "ready", Width: 390, Height: 844}}})
	}
	raw, err := json.Marshal(review)
	require.NoError(t, err)
	var input sketchv1.VisualCritique
	require.NoError(t, protojson.Unmarshal(raw, &input))
	request := connect.NewRequest(&sketchv1.RecordCritiqueRequest{IdempotencyKey: "review", Review: &input})
	first, err := h.RecordCritique(ctx, request)
	require.NoError(t, err)
	require.Equal(t, hash, first.Msg.Review.Ratings[0].Evidence[0].RenderHash)
	require.True(t, first.Msg.Assessment.VisualFloorMet)
	require.False(t, first.Msg.Assessment.AcceptanceEstablished)
	require.Equal(t, 1, calls)
	listed, err := h.ListCritiques(ctx, connect.NewRequest(&sketchv1.ListCritiquesRequest{Target: input.Target}))
	require.NoError(t, err)
	require.Len(t, listed.Msg.Reviews, 1)
	require.Equal(t, first.Msg.Id, listed.Msg.Reviews[0].Id)
	require.Equal(t, first.Msg.Hash, listed.Msg.Reviews[0].Hash)
	server.Close()
	again, err := h.RecordCritique(ctx, request)
	require.NoError(t, err)
	require.True(t, proto.Equal(first.Msg, again.Msg))
	require.Equal(t, 1, calls)
	read, err := h.GetCritique(ctx, connect.NewRequest(&sketchv1.GetCritiqueRequest{Id: first.Msg.Id}))
	require.NoError(t, err)
	require.True(t, proto.Equal(first.Msg, read.Msg))
	input.Ratings[0].Score = 2
	_, err = h.RecordCritique(ctx, request)
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	request.Msg.IdempotencyKey = "unavailable-image"
	_, err = h.RecordCritique(ctx, request)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	h.deps.CritiquesFor = func(context.Context) (designcritique.Repository, error) { return nil, errors.New("expired test lease") }
	_, err = h.GetCritique(ctx, connect.NewRequest(&sketchv1.GetCritiqueRequest{Id: first.Msg.Id}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}
