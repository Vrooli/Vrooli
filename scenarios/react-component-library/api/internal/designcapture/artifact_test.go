package designcapture

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func TestScreenshotResolutionChecksExactOwnerArtifactAndTarget(t *testing.T) {
	for _, name := range []string{"valid", "high density", "wrong producer", "missing artifact", "duplicate artifact", "wrong node", "wrong viewport", "unsafe content", "external URL", "other run URL", "traversal", "encoded traversal", "query"} {
		t.Run(name, func(t *testing.T) {
			shot := map[string]any{"artifactId": "shot", "url": "/api/v1/screenshots/producer/target.png", "width": 390, "height": 844, "contentType": "image/png", "path": "/private/owner/location"}
			entry := map[string]any{"nodeId": "target", "screenshot": shot}
			entries := []any{entry}
			producer := "producer"
			switch name {
			case "high density":
				shot["width"], shot["height"] = 780, 1688
			case "wrong producer":
				producer = "other"
			case "missing artifact":
				shot["artifactId"] = "other"
			case "duplicate artifact":
				entries = append(entries, entry)
			case "wrong node":
				entry["nodeId"] = "open"
			case "wrong viewport":
				shot["width"] = 1440
			case "unsafe content":
				shot["contentType"] = "image/svg+xml"
			case "external URL":
				shot["url"] = "https://example.com/image.png"
			case "other run URL":
				shot["url"] = "/api/v1/screenshots/other/target.png"
			case "traversal":
				shot["url"] = "/api/v1/screenshots/producer/../other/target.png"
			case "encoded traversal":
				shot["url"] = "/api/v1/screenshots/producer/%2e%2e/other/target.png"
			case "query":
				shot["url"] = "/api/v1/screenshots/producer/target.png?redirect=other"
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/browser_automation_studio.v1.ExecutionsService/GetExecutionScreenshots", r.URL.Path)
				var input map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
				require.Equal(t, "producer", input["executionId"])
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"executionId": producer, "screenshots": entries}))
			}))
			defer server.Close()
			op := Operation{ProducerID: "producer", Request: captureRequest()}
			result, err := (BASDispatcher{BASBaseURL: server.URL}).ResolveScreenshot(context.Background(), op, Artifact{Kind: "screenshot", Reference: "bas:producer:shot"})
			if name != "valid" && name != "high density" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, server.URL+"/api/v1/screenshots/producer/target.png", result.URL)
			raw, err := json.Marshal(result)
			require.NoError(t, err)
			require.NotContains(t, string(raw), "private")
		})
	}
}

type screenshotFunc func(context.Context, Operation, Artifact) (Screenshot, error)

func (f screenshotFunc) ResolveScreenshot(ctx context.Context, op Operation, a Artifact) (Screenshot, error) {
	return f(ctx, op, a)
}
func TestScreenshotRequiresCompletedReceiptMembership(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	svc := Service{Repository: repo, Dispatcher: dispatchFunc(func(context.Context, Operation) (string, error) { return "producer", nil })}
	op, err := svc.Start(ctx, "screenshot", captureRequest())
	require.NoError(t, err)
	calls := 0
	resolver := screenshotFunc(func(_ context.Context, _ Operation, a Artifact) (Screenshot, error) {
		calls++
		return Screenshot{Reference: a.Reference}, nil
	})
	_, err = svc.Screenshot(ctx, op.ID, "bas:producer:shot", resolver)
	require.Error(t, err)
	require.Zero(t, calls)
	op, err = repo.Transition(ctx, op.ID, op.Version, Completed, op.ProducerID, []Artifact{{Kind: "screenshot", Reference: "bas:producer:shot"}, {Kind: "target", Reference: "bas:producer:evidence", Evidence: &TargetEvidence{RenderHash: op.Request.Target.RenderHash, Width: 390, Height: 844}}}, "")
	require.NoError(t, err)
	for _, ref := range []string{"bas:other:shot", "bas:producer:evidence"} {
		_, err = svc.Screenshot(ctx, op.ID, ref, resolver)
		require.Error(t, err)
	}
	require.Zero(t, calls)
	result, err := svc.Screenshot(ctx, op.ID, "bas:producer:shot", resolver)
	require.NoError(t, err)
	require.Equal(t, "bas:producer:shot", result.Reference)
	require.Equal(t, 1, calls)
}
