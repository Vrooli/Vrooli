package components

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"react-component-library/internal/components"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
)

// domainToProto converts an internal components.Component into the wire
// shape the components proto declares. Lives in the handler package by
// intent — the conversion is mechanical and only used at the transport
// edge; pulling it into a separate adapters package would create a
// one-import wrapper for no gain.
func domainToProto(c components.Component) *componentsv1.Component {
	tags := append([]string(nil), c.Tags...)
	if tags == nil {
		tags = []string{}
	}
	headers := make(map[string]string, len(c.Headers))
	for k, v := range c.Headers {
		headers[k] = v
	}
	return &componentsv1.Component{
		Id:            c.ID,
		LibraryId:     c.LibraryID,
		DisplayName:   c.DisplayName,
		Description:   c.Description,
		SourcePath:    c.SourcePath,
		Version:       c.Version,
		Tags:          tags,
		IndexedAt:     timestamppb.New(c.IndexedAt.UTC()),
		UpdatedAt:     timestamppb.New(c.UpdatedAt.UTC()),
		Headers:       headers,
		Slug:          c.Slug,
		ManifestPath:  c.ManifestPath,
		DraftVersion:  c.DraftVersion,
		LatestVersion: c.LatestVersion,
	}
}

func versionToProto(v components.ComponentVersion) *componentsv1.ComponentVersion {
	out := &componentsv1.ComponentVersion{
		Id:            v.ID,
		ComponentId:   v.ComponentID,
		LibraryId:     v.LibraryID,
		Version:       v.Version,
		Status:        versionStatusToProto(v.Status),
		SourcePath:    v.SourcePath,
		ContentSha256: v.ContentSHA256,
		ChangelogMd:   v.ChangelogMD,
		IndexedAt:     timestamppb.New(v.IndexedAt.UTC()),
	}
	if !v.ReleasedAt.IsZero() {
		out.ReleasedAt = timestamppb.New(v.ReleasedAt.UTC())
	}
	return out
}

func versionStatusToProto(s components.ComponentVersionStatus) componentsv1.ComponentVersionStatus {
	switch s {
	case components.VersionStatusDraft:
		return componentsv1.ComponentVersionStatus_COMPONENT_VERSION_STATUS_DRAFT
	case components.VersionStatusReleased:
		return componentsv1.ComponentVersionStatus_COMPONENT_VERSION_STATUS_RELEASED
	case components.VersionStatusDeprecated:
		return componentsv1.ComponentVersionStatus_COMPONENT_VERSION_STATUS_DEPRECATED
	case components.VersionStatusArchived:
		return componentsv1.ComponentVersionStatus_COMPONENT_VERSION_STATUS_ARCHIVED
	default:
		return componentsv1.ComponentVersionStatus_COMPONENT_VERSION_STATUS_UNSPECIFIED
	}
}
