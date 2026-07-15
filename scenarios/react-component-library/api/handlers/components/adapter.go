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
		Slot:          c.Slot,
		Category:      c.Category,
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
		DesignStyles:  designAffinitiesToProto(c.DesignStyles),
		AssetKind:     assetKindToProto(c.AssetKind),
		Dependencies:  assetDependenciesToProto(c.Dependencies),
		Metrics: &componentsv1.AssetMetrics{
			DirectAdoptionCount:    int32(c.Metrics.DirectAdoptionCount),
			EffectiveAdoptionCount: int32(c.Metrics.EffectiveAdoptionCount),
			VersionCount:           int32(c.Metrics.VersionCount),
		},
	}
}

func assetKindToProto(kind components.AssetKind) componentsv1.AssetKind {
	if kind == components.AssetKindHook {
		return componentsv1.AssetKind_ASSET_KIND_HOOK
	}
	return componentsv1.AssetKind_ASSET_KIND_COMPONENT
}

func protoAssetKindToDomain(kind componentsv1.AssetKind) components.AssetKind {
	if kind == componentsv1.AssetKind_ASSET_KIND_HOOK {
		return components.AssetKindHook
	}
	if kind == componentsv1.AssetKind_ASSET_KIND_COMPONENT {
		return components.AssetKindComponent
	}
	return ""
}

func assetDependenciesToProto(in []components.AssetDependency) []*componentsv1.AssetDependency {
	out := make([]*componentsv1.AssetDependency, 0, len(in))
	for _, dep := range in {
		out = append(out, &componentsv1.AssetDependency{LibraryId: dep.LibraryID, Version: dep.Version})
	}
	return out
}

func designAffinitiesToProto(in []components.ComponentDesignAffinity) []*componentsv1.ComponentDesignAffinity {
	out := make([]*componentsv1.ComponentDesignAffinity, 0, len(in))
	for _, affinity := range in {
		out = append(out, &componentsv1.ComponentDesignAffinity{
			StyleId:  affinity.StyleID,
			Affinity: designAffinityToProto(affinity.Affinity),
			Reason:   affinity.Reason,
		})
	}
	return out
}

func designAffinityToProto(affinity components.DesignAffinity) componentsv1.DesignAffinity {
	switch affinity {
	case components.DesignAffinityNative:
		return componentsv1.DesignAffinity_DESIGN_AFFINITY_NATIVE
	case components.DesignAffinityCompatible:
		return componentsv1.DesignAffinity_DESIGN_AFFINITY_COMPATIBLE
	case components.DesignAffinityDiscouraged:
		return componentsv1.DesignAffinity_DESIGN_AFFINITY_DISCOURAGED
	default:
		return componentsv1.DesignAffinity_DESIGN_AFFINITY_UNSPECIFIED
	}
}

func styleFitVerdictToProto(v components.StyleFitVerdict) *componentsv1.ValidateStyleFitResponse {
	return &componentsv1.ValidateStyleFitResponse{
		Kind:          styleFitVerdictKindToProto(v.Kind),
		ComponentId:   v.ComponentID,
		Version:       v.Version,
		Scenario:      v.Scenario,
		ScenarioStyle: v.ScenarioStyle,
		Affinity:      designAffinityToProto(v.Affinity),
		Detail:        v.Detail,
	}
}

func styleFitVerdictKindToProto(kind components.StyleFitVerdictKind) componentsv1.StyleFitVerdictKind {
	switch kind {
	case components.StyleFitVerdictOK:
		return componentsv1.StyleFitVerdictKind_STYLE_FIT_VERDICT_KIND_OK
	case components.StyleFitVerdictInfo:
		return componentsv1.StyleFitVerdictKind_STYLE_FIT_VERDICT_KIND_INFO
	case components.StyleFitVerdictWarn:
		return componentsv1.StyleFitVerdictKind_STYLE_FIT_VERDICT_KIND_WARN
	default:
		return componentsv1.StyleFitVerdictKind_STYLE_FIT_VERDICT_KIND_UNSPECIFIED
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
		Files:         versionFilesToProto(v.Files),
		ParityReport:  parityReportToProtoValue(v.ParityReport),
		IndexedAt:     timestamppb.New(v.IndexedAt.UTC()),
	}
	if !v.ReleasedAt.IsZero() {
		out.ReleasedAt = timestamppb.New(v.ReleasedAt.UTC())
	}
	return out
}

func parityReportToProto(report components.IngestParityReport) *componentsv1.IngestParityReport {
	return parityReportToProtoValue(&report)
}

func parityReportToProtoValue(report *components.IngestParityReport) *componentsv1.IngestParityReport {
	if report == nil {
		return nil
	}
	out := &componentsv1.IngestParityReport{OriginFiles: append([]string(nil), report.OriginFiles...), HarvestedFiles: append([]string(nil), report.HarvestedFiles...), Acknowledged: report.Acknowledged}
	for _, finding := range report.Findings {
		out.Findings = append(out.Findings, &componentsv1.IngestFinding{Code: finding.Code, Message: finding.Message, SourceFile: finding.SourceFile})
	}
	return out
}

func versionFilesToProto(files []components.ComponentVersionFile) []*componentsv1.ComponentVersionFile {
	out := make([]*componentsv1.ComponentVersionFile, 0, len(files))
	for _, file := range files {
		out = append(out, &componentsv1.ComponentVersionFile{Path: file.Path, ContentSha256: file.ContentSHA256, IsEntry: file.IsEntry, Slot: file.Slot})
	}
	return out
}

func exampleToProto(ex components.ComponentExample) *componentsv1.ComponentExample {
	return &componentsv1.ComponentExample{
		Id:          ex.ID,
		ComponentId: ex.ComponentID,
		LibraryId:   ex.LibraryID,
		Version:     ex.Version,
		Name:        ex.Name,
		DisplayName: ex.DisplayName,
		PropsJson:   ex.PropsJSON,
		SetupJson:   ex.SetupJSON,
		ExpectJson:  ex.ExpectJSON,
		SourcePath:  ex.SourcePath,
		IndexedAt:   timestamppb.New(ex.IndexedAt.UTC()),
	}
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
