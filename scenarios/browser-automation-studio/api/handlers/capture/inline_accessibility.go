package capture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// defaultInlineAccessibilityMaxBytes caps CaptureResponse.accessibility_json so
// a pathological page cannot balloon the RPC response. Truncation is silent
// (documented on the proto field), mirroring inline_dom.
const defaultInlineAccessibilityMaxBytes = 2 << 20

// accessibilitySnapshotFile is the canonical filename the driver emits and
// ExportToFolder lands in the execution out dir. It is the same name the
// accessibility fileProducer reads.
const accessibilitySnapshotFile = "accessibility.json"

// InlineAccessibilityConfig holds the tunables for inline accessibility
// capture. Zero values fall back to the package defaults via withDefaults so
// callers can wire only the fields they want to override. It mirrors
// InlineDomConfig's shape, differing only in source: the AX snapshot is a real
// file on disk (written by the driver, landed by ExportToFolder), not an
// evaluate node's timeline result.
type InlineAccessibilityConfig struct {
	// MaxBytes caps the returned accessibility_json length (silent truncation).
	MaxBytes int
}

// withDefaults returns a copy with any unset field filled from the package
// defaults.
func (c InlineAccessibilityConfig) withDefaults() InlineAccessibilityConfig {
	if c.MaxBytes <= 0 {
		c.MaxBytes = defaultInlineAccessibilityMaxBytes
	}
	return c
}

// readInlineAccessibility reads the normalized AX-tree snapshot the driver
// wrote (accessibility.json) out of the execution out dir and caps it. An
// absent file is the documented degraded contract (capture succeeded, AX
// capture did not), surfaced as a typed error the caller logs and swallows.
func (c InlineAccessibilityConfig) readInlineAccessibility(outDir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(outDir, accessibilitySnapshotFile))
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("accessibility.json absent (AX capture unavailable this run)")
		}
		return "", fmt.Errorf("read accessibility.json: %w", err)
	}
	if len(raw) == 0 {
		return "", errors.New("accessibility.json is empty")
	}
	if len(raw) > c.MaxBytes {
		raw = raw[:c.MaxBytes]
	}
	return string(raw), nil
}
