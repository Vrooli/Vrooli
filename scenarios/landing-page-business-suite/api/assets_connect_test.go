package main

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
)

func TestAssetProtoPreservesAssetContract(t *testing.T) {
	thumbnail := "logos/thumb.png"
	asset := &Asset{
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

	got := assetProto(asset)
	if got.GetId() != 7 || got.GetOriginalFilename() != "original-logo.png" || got.GetThumbnailPath() != thumbnail {
		t.Fatalf("asset conversion lost fields: %#v", got)
	}
	if got.GetCreatedAt().AsTime() != asset.CreatedAt || got.GetDerivatives()["logo_256"] != "logos/logo-256.png" {
		t.Fatalf("asset conversion lost timestamp or derivatives: %#v", got)
	}
}

func TestAssetsConnectRejectsInvalidAssetIDsBeforeServiceAccess(t *testing.T) {
	handler := assetsConnectHandler{}
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
