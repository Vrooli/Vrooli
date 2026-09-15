package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	landinghttp "landing-page-business-suite-api/handlers/config"
	"landing-page-business-suite-api/internal/landing"
)

// The typed public route must fail closed until a published presentation is
// available; mutable legacy delivery rows are not a publication substitute.
func TestLandingConfigConnectCompositionFailsClosedWithoutPublication(t *testing.T) {
	db := setupTestDB(t)
	configStore := setupTestConfigStore(t)
	planService := NewPlanService(db)
	downloadService := NewDownloadService(db)
	if _, err := downloadService.UpsertApp(DownloadApp{BundleKey: planService.BundleKey(), AppKey: "unreleased-desktop", Name: "Unreleased Desktop"}); err != nil {
		t.Fatalf("seed unreleased download app: %v", err)
	}
	service := landing.NewLandingConfigServiceWithConfigStore(configStore, planService, downloadService)
	_, err := landinghttp.NewLandingConfigConnectHandler(service).GetLandingConfig(context.Background(), connect.NewRequest(&lpbsv1.GetLandingConfigRequest{VariantSlug: "control"}))
	if err == nil {
		t.Fatal("GetLandingConfig() exposed mutable delivery rows without publication")
	}
}
