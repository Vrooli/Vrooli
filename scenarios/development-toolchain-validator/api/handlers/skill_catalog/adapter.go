package skill_catalog

import (
	skillcatalog "development-toolchain-validator/internal/skill_catalog"

	skillcatalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/skill_catalog"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// domainToProto converts a domain Skill into the wire shape declared in
// skill_catalog.proto. Lives in the handler package by intent: the
// conversion is mechanical and only used at the transport edge.
func domainToProto(s skillcatalog.Skill) *skillcatalogv1.Skill {
	return &skillcatalogv1.Skill{
		Id:          s.ID,
		Version:     s.Version,
		ContentHash: s.ContentHash,
		SyncedAt:    timestamppb.New(s.SyncedAt.UTC()),
	}
}
