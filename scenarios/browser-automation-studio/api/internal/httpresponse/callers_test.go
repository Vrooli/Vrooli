package httpresponse_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/usecases/import/assets"
	"github.com/vrooli/browser-automation-studio/usecases/import/projects"
	"github.com/vrooli/browser-automation-studio/usecases/import/scan"
)

func TestImportHTTPCallersShareLegacyJSONErrorWriter(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	projectHandler := projects.NewHandler(nil, logger)
	scanHandler := scan.NewHandler(nil, logger)
	assetHandler := assets.NewHandler(nil, logger)
	projectID := "00000000-0000-4000-8000-000000000001"
	assetRoute := chi.NewRouteContext()
	assetRoute.URLParams.Add("projectID", projectID)
	cases := []struct {
		name string
		run  func(http.ResponseWriter, *http.Request)
		req  *http.Request
	}{
		{name: "project inspect", run: projectHandler.InspectProjectFolder, req: httptest.NewRequest(http.MethodPost, "/projects/inspect-folder", strings.NewReader("{"))},
		{name: "filesystem scan", run: scanHandler.Scan, req: httptest.NewRequest(http.MethodPost, "/fs/scan", strings.NewReader("{"))},
		{name: "asset delete", run: assetHandler.DeleteAsset, req: httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/assets/delete", strings.NewReader("{"))},
	}
	var canonicalBody string
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			if tc.name == "asset delete" {
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, assetRoute))
			}
			response := httptest.NewRecorder()
			tc.run(response, req)
			if response.Code != http.StatusBadRequest || response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("response status/content type = %d/%q, body: %s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body["error"] != true || body["message"] != "invalid request body" {
				t.Fatalf("error envelope = %#v", body)
			}
			if canonicalBody == "" {
				canonicalBody = response.Body.String()
			} else if response.Body.String() != canonicalBody {
				t.Fatalf("caller response differs:\nfirst: %s\nnext:  %s", canonicalBody, response.Body.String())
			}
		})
	}
}
