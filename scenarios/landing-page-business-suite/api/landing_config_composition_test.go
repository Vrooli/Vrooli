package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	landinghttp "landing-page-business-suite-api/handlers/config"
)

// This is intentionally an API-composition test: it verifies the concrete
// landing, pricing, and delivery services still assemble into the public
// generated response after the transport itself moved to handlers/content.
func TestLandingConfigConnectCompositionIncludesUnreleasedApps(t *testing.T) {
	db := setupTestDB(t)
	configStore := setupTestConfigStore(t)
	planService := NewPlanService(db)
	downloadService := NewDownloadService(db)
	if _, err := downloadService.UpsertApp(DownloadApp{BundleKey: planService.BundleKey(), AppKey: "unreleased-desktop", Name: "Unreleased Desktop"}); err != nil {
		t.Fatalf("seed unreleased download app: %v", err)
	}
	service := NewLandingConfigServiceWithConfigStore(configStore, planService, downloadService, nil)
	response, err := landinghttp.NewLandingConfigConnectHandler(service).GetLandingConfig(context.Background(), connect.NewRequest(&lpbsv1.GetLandingConfigRequest{VariantSlug: "control"}))
	if err != nil {
		t.Fatalf("GetLandingConfig() error = %v", err)
	}
	for _, app := range response.Msg.GetDownloads() {
		if app.GetAppKey() == "unreleased-desktop" {
			if app.GetPlatforms() == nil || len(app.GetPlatforms()) != 0 {
				t.Fatalf("expected empty platforms, got %#v", app.GetPlatforms())
			}
			return
		}
	}
	t.Fatalf("expected seeded unreleased app in response: %#v", response.Msg.GetDownloads())
}
