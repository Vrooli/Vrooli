package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	deploymenthttp "landing-page-business-suite-api/handlers/deployment"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
)

func TestDeployReadiness_StorageNotConfigured(t *testing.T) {
	db := setupTestDB(t)
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadApps(t, db)

	hosting := NewDownloadHostingService(db)
	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "no_storage_bundle")
	handler := deployReadinessHandler(hosting, downloads, nil, plans)

	body, _ := json.Marshal(deploymenthttp.Request{AppKey: "any-app"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy-readiness", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 when storage missing, got %d: %s", w.Code, w.Body.String())
	}
	var resp deploymenthttp.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Ready {
		t.Errorf("expected ready=false; got true")
	}
	if !strings.Contains(resp.Error, "download_storage") {
		t.Errorf("expected error mentioning download_storage, got %q", resp.Error)
	}
}

func TestDeployReadiness_AppMissing(t *testing.T) {
	db := setupTestDB(t)
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadApps(t, db)
	if _, err := db.Exec(`INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('readiness_bundle', 's3', 'readiness-bucket', 'us-east-1', 900)`); err != nil {
		t.Fatalf("insert storage: %v", err)
	}
	hosting := NewDownloadHostingService(db)
	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "readiness_bundle")
	handler := deployReadinessHandler(hosting, downloads, nil, plans)

	body, _ := json.Marshal(deploymenthttp.Request{AppKey: "missing-app"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy-readiness", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for missing app, got %d: %s", w.Code, w.Body.String())
	}
	var resp deploymenthttp.Response
	_ = json.NewDecoder(w.Body).Decode(&resp)
	foundApp := false
	for _, g := range resp.Gates {
		if g.Name == "app_registered" && !g.Ready {
			foundApp = true
		}
	}
	if !foundApp {
		t.Errorf("expected app_registered gate to fail; gates=%+v", resp.Gates)
	}
}

func TestDeployReadiness_StorageAndAppOK(t *testing.T) {
	db := setupTestDB(t)
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadApps(t, db)
	if _, err := db.Exec(`INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('ready_bundle', 's3', 'ready-bucket', 'us-east-1', 900)`); err != nil {
		t.Fatalf("insert storage: %v", err)
	}
	insertTestDownloadApp(t, db, "ready_bundle", "ready-app", "Ready App")

	hosting := NewDownloadHostingService(db)
	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "ready_bundle")
	handler := deployReadinessHandler(hosting, downloads, nil, plans)

	body, _ := json.Marshal(deploymenthttp.Request{AppKey: "ready-app"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy-readiness", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp deploymenthttp.Response
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Ready {
		t.Errorf("expected ready=true; gates=%+v", resp.Gates)
	}
}

func deployReadinessHandler(storage *delivery.Service, catalog *delivery.CatalogService, profiles *administration.RemoteProfileService, plans *commerce.PlanService) http.HandlerFunc {
	return deploymenthttp.Readiness(deploymenthttp.Dependencies{
		Storage: storage, Catalog: catalog, RemoteProfiles: profiles, BundleKey: plans.BundleKey,
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			writeJSONError(w, status, message, kind)
		},
	})
}
