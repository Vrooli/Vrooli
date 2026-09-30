package assets

import (
	"brand-manager/internal/assets"

	assetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assets"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// domainToProto converts an internal assets.Asset into the wire shape the
// assets proto declares. The on-disk FilePath is intentionally omitted — it is
// a server-internal detail, and callers fetch bytes via DownloadAsset.
func domainToProto(a assets.Asset) *assetsv1.Asset {
	return &assetsv1.Asset{
		Id:        a.ID,
		BrandId:   a.BrandID,
		Filename:  a.Filename,
		MimeType:  a.MimeType,
		Size:      a.Size,
		CreatedAt: timestamppb.New(a.CreatedAt.UTC()),
	}
}
