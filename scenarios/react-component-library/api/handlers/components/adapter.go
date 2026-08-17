package components

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"react-component-library/internal/components"
	"react-component-library/internal/experience"

	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
)

func experienceToProto(snapshot experience.Snapshot) *componentsv1.ComponentExperience {
	out := &componentsv1.ComponentExperience{ComponentId: snapshot.ComponentID, LibraryId: snapshot.LibraryID, Version: snapshot.Version, ContractId: snapshot.ContractID, Title: snapshot.Title, Purpose: snapshot.Purpose, EvidenceStatus: snapshot.EvidenceStatus, EvidenceMessage: snapshot.EvidenceMessage}
	for _, state := range snapshot.States {
		out.States = append(out.States, &componentsv1.ComponentExperienceState{Id: state.ID, ExampleName: state.ExampleName, Description: state.Description})
	}
	for _, claim := range snapshot.Claims {
		out.Claims = append(out.Claims, &componentsv1.ComponentExperienceClaim{Id: claim.ID, Type: claim.Type, Statement: claim.Statement, Tier: claim.Tier, States: append([]string(nil), claim.States...)})
	}
	for _, evidence := range snapshot.Evidence {
		out.Evidence = append(out.Evidence, &componentsv1.ComponentExperienceEvidence{ClaimId: evidence.ClaimID, Verdict: evidence.Verdict, StateId: evidence.StateID, ExampleName: evidence.ExampleName, CaptureRef: evidence.CaptureRef, CheckedAt: evidence.CheckedAt, Message: evidence.Message, Viewport: evidence.Viewport, ViewportWidth: int32(evidence.ViewportWidth), ViewportHeight: int32(evidence.ViewportHeight)})
	}
	return out
}

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
		Id:                       c.ID,
		LibraryId:                c.LibraryID,
		DisplayName:              c.DisplayName,
		Description:              c.Description,
		Slot:                     c.Slot,
		Category:                 c.Category,
		SourcePath:               c.SourcePath,
		Version:                  c.Version,
		Tags:                     tags,
		IndexedAt:                timestamppb.New(c.IndexedAt.UTC()),
		UpdatedAt:                timestamppb.New(c.UpdatedAt.UTC()),
		Headers:                  headers,
		Slug:                     c.Slug,
		ManifestPath:             c.ManifestPath,
		DraftVersion:             c.DraftVersion,
		LatestVersion:            c.LatestVersion,
		DesignStyles:             designAffinitiesToProto(c.DesignStyles),
		AssetKind:                assetKindToProto(c.AssetKind),
		Dependencies:             assetDependenciesToProto(c.Dependencies),
		CatalogDomain:            c.CatalogDomain,
		CatalogDomainOrder:       int32(c.CatalogDomainOrder),
		CatalogRung:              int32(c.CatalogRung),
		CatalogRungName:          c.CatalogRungName,
		TransitiveDependentCount: int32(c.TransitiveDependentCount),
		CatalogId:                c.CatalogID,
		Metrics: &componentsv1.AssetMetrics{
			DirectAdoptionCount:    int32(c.Metrics.DirectAdoptionCount),
			EffectiveAdoptionCount: int32(c.Metrics.EffectiveAdoptionCount),
			VersionCount:           int32(c.Metrics.VersionCount),
			VersionAdoptions:       versionAdoptionsToProto(c.Metrics.VersionAdoptions),
		},
	}
}

func versionAdoptionsToProto(in []components.VersionAdoptionMetric) []*componentsv1.VersionAdoptionMetric {
	out := make([]*componentsv1.VersionAdoptionMetric, 0, len(in))
	for _, item := range in {
		out = append(out, &componentsv1.VersionAdoptionMetric{Version: item.Version, CurrentCount: int32(item.CurrentCount), PeakCount: int32(item.PeakCount)})
	}
	return out
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

func parityReportFromProto(report *componentsv1.IngestParityReport) *components.IngestParityReport {
	if report == nil {
		return nil
	}
	out := &components.IngestParityReport{
		OriginFiles:    append([]string(nil), report.OriginFiles...),
		HarvestedFiles: append([]string(nil), report.HarvestedFiles...),
		Acknowledged:   report.Acknowledged,
	}
	for _, finding := range report.Findings {
		if finding == nil {
			continue
		}
		out.Findings = append(out.Findings, components.IngestFinding{Code: finding.Code, Message: finding.Message, SourceFile: finding.SourceFile})
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

func storyToProto(story components.ComponentStory) *componentsv1.ComponentStory {
	return &componentsv1.ComponentStory{
		Id:              story.ID,
		ComponentId:     story.ComponentID,
		LibraryId:       story.LibraryID,
		Version:         story.Version,
		SchemaVersion:   int32(story.SchemaVersion),
		Kind:            string(story.Kind),
		Title:           story.Title,
		ArgsJson:        story.ArgsJSON,
		EnvironmentJson: story.EnvironmentJSON,
		StoriesJson:     story.StoriesJSON,
		ContractJson:    story.ContractJSON,
		SourcePath:      story.SourcePath,
		IndexedAt:       timestamppb.New(story.IndexedAt.UTC()),
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
