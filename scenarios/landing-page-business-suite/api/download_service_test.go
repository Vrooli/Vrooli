package main

import (
	"errors"
	"testing"
)

func TestDownloadServiceDeleteAppRemovesAssets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewDownloadService(db)
	bundleKey := "bundle_delete_test"
	appKey := "delete_me_app"

	created, err := service.UpsertDownloadApp(DownloadApp{
		BundleKey: bundleKey,
		AppKey:    appKey,
		Name:      "Delete Me",
		Platforms: []DownloadAsset{
			{
				BundleKey:      bundleKey,
				AppKey:         appKey,
				Platform:       "windows",
				ArtifactURL:    "https://example.com/win.exe",
				ReleaseVersion: "1.0.0",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to seed download app: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected created download app to have an ID")
	}

	if err := service.DeleteApp(bundleKey, appKey); err != nil {
		t.Fatalf("DeleteApp returned error: %v", err)
	}

	if _, err := service.GetApp(bundleKey, appKey); !errors.Is(err, ErrDownloadAppNotFound) {
		t.Fatalf("expected ErrDownloadAppNotFound after delete, got %v", err)
	}

	var assetCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM download_assets WHERE bundle_key = $1 AND app_key = $2`, bundleKey, appKey).Scan(&assetCount); err != nil {
		t.Fatalf("failed counting assets: %v", err)
	}
	if assetCount != 0 {
		t.Fatalf("expected assets to be deleted with app, found %d", assetCount)
	}
}

func TestDownloadServiceDeleteAppNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewDownloadService(db)
	err := service.DeleteApp("missing_bundle", "missing_app")
	if !errors.Is(err, ErrDownloadAppNotFound) {
		t.Fatalf("expected ErrDownloadAppNotFound, got %v", err)
	}
}
