package capture

import (
	"fmt"
	"strings"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

// captureTypeMeta is the single CLI-side source of truth for a CaptureType:
// its canonical label (used for --json and the human report) and the
// aliases the --capture flag accepts. The two switches that used to live in
// command.go (parseCaptureType + captureTypeLabel) collapse into this table.
type captureTypeMeta struct {
	label   string
	aliases []string
}

// captureTypeMetadata is keyed by CaptureType. label is the canonical name;
// aliases are the additional --capture tokens that resolve to this type.
// The label is always an implicit alias and need not be repeated.
var captureTypeMetadata = map[capturev1.CaptureType]captureTypeMeta{
	capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT:    {label: "screenshot"},
	capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS:  {label: "console-logs", aliases: []string{"console", "logs"}},
	capturev1.CaptureType_CAPTURE_TYPE_NETWORK:       {label: "network"},
	capturev1.CaptureType_CAPTURE_TYPE_VIDEO:         {label: "video"},
	capturev1.CaptureType_CAPTURE_TYPE_DOM:           {label: "dom"},
	capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE:   {label: "performance", aliases: []string{"perf"}},
	capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY: {label: "accessibility", aliases: []string{"a11y", "ax"}},
}

// captureTypeByAlias is the reverse index from accepted token → CaptureType,
// built once from captureTypeMetadata.
var captureTypeByAlias = buildAliasIndex()

func buildAliasIndex() map[string]capturev1.CaptureType {
	idx := make(map[string]capturev1.CaptureType)
	for ct, meta := range captureTypeMetadata {
		idx[meta.label] = ct
		for _, a := range meta.aliases {
			idx[a] = ct
		}
	}
	return idx
}

// captureTypeLabel returns the canonical label for a CaptureType.
func captureTypeLabel(t capturev1.CaptureType) string {
	if meta, ok := captureTypeMetadata[t]; ok {
		return meta.label
	}
	return "unspecified"
}

// parseCaptureType resolves a --capture token (case-insensitive, with
// underscores treated as hyphens) into a CaptureType via the alias index.
func parseCaptureType(tok string) (capturev1.CaptureType, error) {
	key := strings.ToLower(strings.ReplaceAll(tok, "_", "-"))
	if ct, ok := captureTypeByAlias[key]; ok {
		return ct, nil
	}
	return capturev1.CaptureType_CAPTURE_TYPE_UNSPECIFIED,
		fmt.Errorf("unknown capture type %q (want one of: screenshot,console-logs,network,video,dom,performance,accessibility)", tok)
}
