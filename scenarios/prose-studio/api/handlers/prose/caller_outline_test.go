package prose

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	internal "prose-studio/internal/prose"
)

type narrationFixtureGateway struct {
	calls []internal.GatewayRequest
}

func (g *narrationFixtureGateway) Generate(_ context.Context, request internal.GatewayRequest) ([]internal.GatewayCandidate, error) {
	g.calls = append(g.calls, request)
	text := "The empty hook swayed gently beside the front door."
	if strings.Contains(request.Query, "Current section: 2 of 2") {
		text = "Under the letter, the brass key caught the light."
	}
	return []internal.GatewayCandidate{{Text: text, Provider: "fixture", Model: "fixture"}}, nil
}

func TestCreateDocumentConnectWithNarrationDeclaration(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "prose.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, internal.EnsureSchema(db))
	gateway := &narrationFixtureGateway{}
	service := internal.NewWithGateway(db, gateway)
	root, err := filepath.Abs("../../..")
	require.NoError(t, err)
	_, err = service.Reindex(ctx, root)
	require.NoError(t, err)

	router := mux.NewRouter()
	Module(service).Mount(router)
	body := `{"document":{"id":"synthetic-narration","profile_key":"prose-studio/video-narration-local","title":"Fixture","outline":[{"intent":"Opening","summary":"Establish the missing key.","target_words":10},{"intent":"Reveal","summary":"Find it under a letter.","target_words":10}]}}`
	url := "/vrooli.prose_studio.v1.prose.ProseStudioService/CreateDocument"
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), "The empty hook swayed gently beside the front door.")
	require.Contains(t, recorder.Body.String(), "Under the letter, the brass key caught the light.")
	require.Len(t, gateway.calls, 2)
	for _, call := range gateway.calls {
		require.Equal(t, "local_only", call.Locality)
		require.Equal(t, "write.default", call.Role)
		require.Contains(t, call.Instruction, "Write only spoken narration")
	}
}
