package capabilities

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/capabilities"
)

// TestFormatCapabilityExposesLinkedRelationships is the CLI-parity contract:
// a capability query must expose prerequisites, channels and the producing
// operation the API stores, not only the readiness dimensions. It fails if the
// CLI silently drops a stored link.
func TestFormatCapabilityExposesLinkedRelationships(t *testing.T) {
	rendered := formatCapability(&capabilitiesv1.Capability{
		Id:                       "abc-123",
		Name:                     "Short-form promotional video",
		Medium:                   "video",
		Aliases:                  []string{"short-form-video", "promo-video"},
		Channels:                 []string{"tiktok", "youtube-shorts"},
		AudienceApplicability:    "developer",
		DeliveryApplicability:    "organic",
		ProducingOperation:       "asset-studio",
		Prerequisites:            []string{"active post type", "paired craft skill"},
		Priority:                 8,
		PriorityReason:           "Launch needs one real playable demo.",
		PriorityScope:            "launch-2026-09-17",
		Owner:                    "asset-studio",
		DefinitionStatus:         "documented",
		ImplementationStatus:     "skeleton",
		OperationalReadiness:     "unverified",
		OutputQuality:            "unassessed",
		DistributionConnectivity: "connected",
		// The read-time connectivity verdict must be attributable to its owner
		// source, not reported as an unbound value.
		DistributionConnectivitySource: "channel-manager",
		SourceRefs:                     []string{"MARKETING.md"},
		NextAction:                     "Name a producing owner.",
	})

	for _, expected := range []string{
		"aliases=[short-form-video; promo-video]",
		"prerequisites=[active post type; paired craft skill]",
		"channels=[tiktok; youtube-shorts]",
		"audience=developer",
		"delivery=organic",
		"producer=asset-studio",
		"priority_reason=Launch needs one real playable demo.",
		"connectivity=connected",
		"connectivity_source=channel-manager",
		"sources=[MARKETING.md]",
		"next_action=Name a producing owner.",
	} {
		require.Contains(t, rendered, expected, "CLI output must expose %q", expected)
	}
}

// TestFormatCapabilityOmitsUnsetLinks proves an empty link is omitted rather
// than rendered as a fabricated value like channels=[] or audience=.
func TestFormatCapabilityOmitsUnsetLinks(t *testing.T) {
	rendered := formatCapability(&capabilitiesv1.Capability{
		Id:     "abc-123",
		Name:   "Bare capability",
		Medium: "governance",
	})

	require.NotContains(t, rendered, "aliases=")
	require.NotContains(t, rendered, "prerequisites=")
	require.NotContains(t, rendered, "channels=")
	require.NotContains(t, rendered, "audience=")
	require.NotContains(t, rendered, "delivery=")
	require.NotContains(t, rendered, "producer=")
	require.NotContains(t, rendered, "priority_reason=")
	require.NotContains(t, rendered, "sources=")
}

// TestFormatCapabilityNilIsHonest keeps the nil sentinel explicit instead of a
// blank string a caller could mistake for an empty capability.
func TestFormatCapabilityNilIsHonest(t *testing.T) {
	require.Equal(t, "(nil)", formatCapability(nil))
}

func TestFormatReadinessLimitations(t *testing.T) {
	require.Empty(t, formatReadinessLimitations(nil))
	require.Equal(t,
		" readiness_limitations=[no linked evidence; distribution owner read failed]",
		formatReadinessLimitations([]string{"no linked evidence", "distribution owner read failed"}))
}

// TestFormatCapabilityDoesNotDoubleSpace guards the report shape: inserting a
// link between qualification and next_action must not leave a run of spaces.
func TestFormatCapabilityDoesNotDoubleSpace(t *testing.T) {
	rendered := formatCapability(&capabilitiesv1.Capability{
		Id:                   "abc-123",
		Name:                 "Written marketing",
		Medium:               "text",
		Prerequisites:        []string{"active post type"},
		OperationalReadiness: "unverified",
		NextAction:           "Exercise review.",
	})
	require.False(t, strings.Contains(rendered, "  "), "report must not contain double spaces: %q", rendered)
}
