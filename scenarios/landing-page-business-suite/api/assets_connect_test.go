package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	contenthttp "landing-page-business-suite-api/handlers/content"
	"landing-page-business-suite-api/internal/content"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

func TestAssetProtoPreservesAssetContract(t *testing.T) {
	thumbnail := "logos/thumb.png"
	asset := &content.Asset{
		ID:               7,
		Filename:         "logo.png",
		OriginalFilename: "original-logo.png",
		MimeType:         "image/png",
		SizeBytes:        42,
		StoragePath:      "logos/logo.png",
		ThumbnailPath:    &thumbnail,
		Category:         "logo",
		CreatedAt:        time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		URL:              "/api/v1/uploads/logos/logo.png",
		Derivatives:      map[string]string{"logo_256": "logos/logo-256.png"},
	}

	got := contenthttp.AssetProto(asset)
	if got.GetId() != 7 || got.GetOriginalFilename() != "original-logo.png" || got.GetThumbnailPath() != thumbnail {
		t.Fatalf("asset conversion lost fields: %#v", got)
	}
	if got.GetCreatedAt().AsTime() != asset.CreatedAt || got.GetDerivatives()["logo_256"] != "logos/logo-256.png" {
		t.Fatalf("asset conversion lost timestamp or derivatives: %#v", got)
	}
}

func TestAssetProto_MapsNilToAnEmptyMessage(t *testing.T) {
	if got := contenthttp.AssetProto(nil); got.GetId() != 0 || got.GetFilename() != "" {
		t.Fatalf("nil asset must map to an empty generated message, got %#v", got)
	}
}

func TestAssetsConnectRejectsInvalidAssetIDsBeforeServiceAccess(t *testing.T) {
	handler := contenthttp.NewAssetsConnectHandler(nil)
	for _, id := range []int64{0, -1} {
		_, err := handler.GetAsset(context.Background(), connect.NewRequest(&lpbsv1.GetAssetRequest{Id: id}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("GetAsset(%d) error code = %s, want invalid argument", id, connect.CodeOf(err))
		}
		_, err = handler.DeleteAsset(context.Background(), connect.NewRequest(&lpbsv1.DeleteAssetRequest{Id: id}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("DeleteAsset(%d) error code = %s, want invalid argument", id, connect.CodeOf(err))
		}
	}
}

func TestAssetsConnectListsGetsAndDeletesPersistedAssets(t *testing.T) {
	db := setupTestDB(t)
	service := setupTestAssetsService(t, db)
	createdAt := time.Now().UTC().Truncate(time.Second)
	var assetID int
	if err := db.QueryRow(`
		INSERT INTO assets (filename, original_filename, mime_type, size_bytes, storage_path, category, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "logo.png", "logo.png", "image/png", 42, "logos/logo.png", "logo", createdAt).Scan(&assetID); err != nil {
		t.Fatalf("seed asset: %v", err)
	}

	handler := contenthttp.NewAssetsConnectHandler(service)
	listed, err := handler.ListAssets(context.Background(), connect.NewRequest(&lpbsv1.ListAssetsRequest{Category: "logo"}))
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}
	if len(listed.Msg.GetAssets()) != 1 || listed.Msg.GetAssets()[0].GetId() != int64(assetID) {
		t.Fatalf("unexpected listed assets: %+v", listed.Msg.GetAssets())
	}

	got, err := handler.GetAsset(context.Background(), connect.NewRequest(&lpbsv1.GetAssetRequest{Id: int64(assetID)}))
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}
	if got.Msg.GetAsset().GetStoragePath() != "logos/logo.png" {
		t.Fatalf("unexpected asset payload: %+v", got.Msg.GetAsset())
	}

	deleted, err := handler.DeleteAsset(context.Background(), connect.NewRequest(&lpbsv1.DeleteAssetRequest{Id: int64(assetID)}))
	if err != nil || !deleted.Msg.GetDeleted() {
		t.Fatalf("DeleteAsset result = %+v, %v", deleted, err)
	}
	_, err = handler.GetAsset(context.Background(), connect.NewRequest(&lpbsv1.GetAssetRequest{Id: int64(assetID)}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("GetAsset after deletion error code = %s, want not found", connect.CodeOf(err))
	}
	_, err = handler.DeleteAsset(context.Background(), connect.NewRequest(&lpbsv1.DeleteAssetRequest{Id: int64(assetID)}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("DeleteAsset after deletion error code = %s, want not found", connect.CodeOf(err))
	}
}

func TestRegisterAssetsConnectRoutes_MountsEveryGeneratedProcedure(t *testing.T) {
	router := mux.NewRouter()
	contenthttp.RegisterAssetsConnectRoutes(router, nil, func(next http.HandlerFunc) http.HandlerFunc { return next })

	for _, procedure := range []string{
		lpbsconnect.AssetsServiceListAssetsProcedure,
		lpbsconnect.AssetsServiceGetAssetProcedure,
		lpbsconnect.AssetsServiceDeleteAssetProcedure,
	} {
		req := httptest.NewRequest(http.MethodPost, procedure, nil)
		if !router.Match(req, &mux.RouteMatch{}) {
			t.Fatalf("generated procedure %q was not mounted", procedure)
		}
	}
}
