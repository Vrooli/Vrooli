package visualhealth

import (
	"fmt"
	"strings"

	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

func networkFindings(step *visualpb.VisualStepArtifact) []*visualpb.VisualFinding {
	if step == nil {
		return nil
	}
	var out []*visualpb.VisualFinding
	for _, entry := range step.GetNetwork() {
		if entry == nil || !isBrokenVisualAsset(entry) {
			continue
		}
		evidence := entry.GetErrorText()
		if evidence == "" && entry.GetStatus() > 0 {
			evidence = fmt.Sprintf("HTTP %d", entry.GetStatus())
		}
		out = append(out, &visualpb.VisualFinding{
			Code:        "visual_broken_asset",
			Severity:    severityError,
			Category:    categoryAsset,
			Message:     "browser reported a failed visual asset request",
			Location:    firstNonEmpty(entry.GetUrl(), locationFor(step)),
			Evidence:    evidence,
			Remediation: "Resolve failed image, media, font, or stylesheet requests emitted by the browser.",
			StepId:      step.GetStepId(),
		})
	}
	return out
}

func isBrokenVisualAsset(entry *visualpb.NetworkEntry) bool {
	if entry.GetStatus() < 400 && strings.TrimSpace(entry.GetErrorText()) == "" {
		return false
	}
	t := strings.ToLower(strings.TrimSpace(entry.GetResourceType()))
	if t == "" {
		return looksVisualAsset(entry.GetUrl())
	}
	switch t {
	case "image", "media", "font", "stylesheet":
		return true
	default:
		return false
	}
}

func looksVisualAsset(url string) bool {
	u := strings.ToLower(strings.TrimSpace(url))
	for _, suffix := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".ico", ".css", ".woff", ".woff2", ".ttf", ".mp4", ".webm"} {
		if strings.Contains(u, suffix) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
