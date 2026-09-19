package page

import (
	"github.com/stretchr/testify/require"
	pagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page"
	"strings"
	"testing"
)

func TestHumanInspectionReportsUnstampedAndUnresolvedHonestly(t *testing.T) {
	result := &pagev1.InspectResponse{Url: "http://example/coverage", ScreenshotPath: "capture.png", NodeCount: 3, StampedCount: 2, ResolvedCount: 1, Tree: &pagev1.PageNode{Source: &pagev1.AssetSource{Status: "unstamped"}, Children: []*pagev1.PageNode{
		{Source: &pagev1.AssetSource{Status: "resolved", StampedAsset: "controls.button", StampedVersion: "2", SourcePath: "Button.tsx", ResolutionRule: "consumer-major-alias"}},
		{Source: &pagev1.AssetSource{Status: "unresolved", StampedAsset: "unknown", StampedVersion: "1", Reason: "stamp has no unique catalog identity"}},
	}}}
	report := inspectReport(nil, result)
	require.Contains(t, strings.Join(report.Summary, "\n"), "1 unstamped")
	text := strings.Join(report.Results, "\n")
	require.Contains(t, text, "Screenshot: capture.png")
	require.Contains(t, text, "Button.tsx (consumer-major-alias)")
	require.Contains(t, text, "unknown@1 → stamp has no unique catalog identity")
}
