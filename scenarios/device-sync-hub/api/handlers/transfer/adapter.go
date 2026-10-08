package transfer

import (
	"device-sync-hub/internal/transfer"

	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// itemToProto converts an internal transfer.Item into the wire shape the
// transfer proto declares. Mechanical, used only at the transport edge — the
// domain layer never imports proto. The blob and thumbnail storage keys are
// deliberately NOT surfaced (internal storage detail); has_thumbnail conveys
// only that a thumbnail exists.
func itemToProto(i transfer.Item) *transferv1.Item {
	out := &transferv1.Item{
		Id:             i.ID,
		OwnerId:        i.OwnerID,
		OriginDeviceId: i.OriginDeviceID,
		Kind:           kindToProto(i.Kind),
		Name:           i.Name,
		Mime:           i.MIME,
		SizeBytes:      i.SizeBytes,
		Text:           i.Text,
		HasThumbnail:   i.HasThumbnail(),
		Retention:      retentionToProto(i.Retention),
		TargetDeviceId: i.TargetDeviceID,
		CreatedAt:      timestamppb.New(i.CreatedAt.UTC()),
	}
	if !i.ExpiresAt.IsZero() {
		out.ExpiresAt = timestamppb.New(i.ExpiresAt.UTC())
	}
	return out
}

func kindToProto(k transfer.Kind) transferv1.ItemKind {
	switch k {
	case transfer.KindText:
		return transferv1.ItemKind_ITEM_KIND_TEXT
	case transfer.KindFile:
		return transferv1.ItemKind_ITEM_KIND_FILE
	default:
		return transferv1.ItemKind_ITEM_KIND_UNSPECIFIED
	}
}

func kindFromProto(k transferv1.ItemKind) transfer.Kind {
	switch k {
	case transferv1.ItemKind_ITEM_KIND_TEXT:
		return transfer.KindText
	case transferv1.ItemKind_ITEM_KIND_FILE:
		return transfer.KindFile
	default:
		return "" // unspecified — list filter treats this as "both kinds"
	}
}

func retentionToProto(r transfer.Retention) transferv1.Retention {
	switch r {
	case transfer.RetentionLive:
		return transferv1.Retention_RETENTION_LIVE
	case transfer.RetentionHeld:
		return transferv1.Retention_RETENTION_HELD
	case transfer.RetentionPinned:
		return transferv1.Retention_RETENTION_PINNED
	default:
		return transferv1.Retention_RETENTION_UNSPECIFIED
	}
}

// retentionFromProto maps the wire enum to the domain string. UNSPECIFIED maps
// to "" so the service applies the configured global default.
func retentionFromProto(r transferv1.Retention) transfer.Retention {
	switch r {
	case transferv1.Retention_RETENTION_LIVE:
		return transfer.RetentionLive
	case transferv1.Retention_RETENTION_HELD:
		return transfer.RetentionHeld
	case transferv1.Retention_RETENTION_PINNED:
		return transfer.RetentionPinned
	default:
		return ""
	}
}
