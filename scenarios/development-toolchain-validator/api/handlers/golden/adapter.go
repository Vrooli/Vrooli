package golden

import (
	"development-toolchain-validator/internal/golden"

	goldenv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/golden"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// domainToProto converts an internal golden.Golden into the wire shape
// the golden proto declares. Lives in the handler package by intent:
// the conversion is mechanical and only used at the transport edge.
func domainToProto(g golden.Golden) *goldenv1.Golden {
	return &goldenv1.Golden{
		Id:                     g.ID,
		Slug:                   g.Slug,
		TemplateId:             g.TemplateID,
		TemplateVersionPinned:  g.TemplateVersionPinned,
		Path:                   g.Path,
		CreatedAt:              timestamppb.New(g.CreatedAt.UTC()),
		LastRegeneratedAt:      timestamppb.New(g.LastRegeneratedAt.UTC()),
		GenerationOptionsJson:  g.GenerationOptionsJSON,
		MaterializationMode:    g.MaterializationMode,
		LogicalRoot:            g.LogicalRoot,
		LastMaterializedPath:   g.LastMaterializedPath,
		LastMaterializedStatus: g.LastMaterializedStatus,
	}
}
