package components

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
	"react-component-library/internal/components"
	"react-component-library/internal/experience"
)

func TestAdapterParityReportRoundTrip(t *testing.T) {
	require.Nil(t, parityReportToProtoValue(nil))
	require.Nil(t, parityReportFromProto(nil))
	in := &components.IngestParityReport{OriginFiles: []string{"src/Button.tsx"}, HarvestedFiles: []string{"Button.tsx"}, Acknowledged: true, Findings: []components.IngestFinding{{Code: "missing", Message: "not copied", SourceFile: "src/Button.tsx"}}}
	wire := parityReportToProtoValue(in)
	require.Equal(t, "missing", wire.Findings[0].Code)
	wire.Findings = append(wire.Findings, nil)
	out := parityReportFromProto(wire)
	require.Equal(t, in, out)
	require.Nil(t, parityReportToProto(components.IngestParityReport{}).Findings)
}

func TestAdapterStoriesAndEnumFallbacks(t *testing.T) {
	now := time.Date(2025, 2, 3, 4, 5, 6, 0, time.FixedZone("offset", 3600))
	story := storyToProto(components.ComponentStory{ID: "story-id", ComponentID: "button", LibraryID: "lib:button", Version: "1.2.0", SchemaVersion: 2, Kind: "storybook", Title: "Primary", ArgsJSON: "{}", EnvironmentJSON: "{}", StoriesJSON: "[]", ContractJSON: "{}", SourcePath: "story.json", IndexedAt: now})
	require.Equal(t, "story-id", story.Id)
	require.Equal(t, int32(2), story.SchemaVersion)
	require.Equal(t, now.UTC(), story.IndexedAt.AsTime())

	require.Equal(t, componentsv1.AssetKind_ASSET_KIND_COMPONENT, assetKindToProto("unknown"))
	require.Equal(t, components.AssetKind(""), protoAssetKindToDomain(componentsv1.AssetKind_ASSET_KIND_UNSPECIFIED))
	require.Equal(t, componentsv1.DesignAffinity_DESIGN_AFFINITY_UNSPECIFIED, designAffinityToProto("unknown"))
	require.Equal(t, componentsv1.StyleFitVerdictKind_STYLE_FIT_VERDICT_KIND_UNSPECIFIED, styleFitVerdictKindToProto("unknown"))
	require.Equal(t, componentsv1.ComponentVersionStatus_COMPONENT_VERSION_STATUS_UNSPECIFIED, versionStatusToProto("unknown"))
}

func TestAdapterExperienceCopiesNestedFields(t *testing.T) {
	got := experienceToProto(experience.Snapshot{
		ComponentID: "id", LibraryID: "lib:id", Version: "1.0.0", ContractID: "contract", Title: "Title", Purpose: "Purpose", EvidenceStatus: "verified", EvidenceMessage: "ok",
		States:   []experience.State{{ID: "empty", ExampleName: "Empty", Description: "No rows"}},
		Claims:   []experience.Claim{{ID: "keyboard", Type: "accessibility", Statement: "Keyboard works", Tier: "required", States: []string{"empty"}}},
		Evidence: []experience.Evidence{{ClaimID: "keyboard", Verdict: "pass", StateID: "empty", ExampleName: "Empty", CaptureRef: "capture.png", CheckedAt: "now", Message: "works", Viewport: "desktop", ViewportWidth: 1280, ViewportHeight: 800}},
	})
	require.Equal(t, "verified", got.EvidenceStatus)
	require.Equal(t, "empty", got.States[0].Id)
	require.Equal(t, []string{"empty"}, got.Claims[0].States)
	require.Equal(t, int32(1280), got.Evidence[0].ViewportWidth)
}
