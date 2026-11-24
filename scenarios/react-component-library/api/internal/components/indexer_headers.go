package components

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

func parseManifestDesignStyles(path string, raw []manifestDesignAffinity) ([]ComponentDesignAffinity, error) {
	out := make([]ComponentDesignAffinity, 0, len(raw))
	seen := map[string]struct{}{}
	for _, entry := range raw {
		styleID := strings.TrimSpace(entry.StyleID)
		affinity := DesignAffinity(strings.ToLower(strings.TrimSpace(entry.Affinity)))
		if styleID == "" {
			return nil, ErrInvalidHeader{SourcePath: path, Field: "designStyles", Reason: "styleId required"}
		}
		if affinity == "" {
			affinity = DesignAffinityCompatible
		}
		if !isValidDesignAffinity(affinity) {
			return nil, ErrInvalidHeader{SourcePath: path, Field: "designStyles", Reason: fmt.Sprintf("invalid affinity %q for style %q", affinity, styleID)}
		}
		if _, ok := seen[styleID]; ok {
			return nil, ErrInvalidHeader{SourcePath: path, Field: "designStyles", Reason: fmt.Sprintf("duplicate style id %q", styleID)}
		}
		seen[styleID] = struct{}{}
		out = append(out, ComponentDesignAffinity{StyleID: styleID, Affinity: affinity, Reason: strings.TrimSpace(entry.Reason)})
	}
	return out, nil
}

func staleDesignStyleFindings(path string, affinities []ComponentDesignAffinity) ([]IndexFinding, error) {
	if len(affinities) == 0 {
		return nil, nil
	}
	root, err := defaultDesignRoot()
	if err != nil {
		return nil, err
	}
	styles, err := LoadDesignStyles(context.Background(), root)
	if err != nil {
		return nil, err
	}
	known := make(map[string]struct{}, len(styles))
	for _, style := range styles {
		known[style.ID] = struct{}{}
	}
	var findings []IndexFinding
	for _, affinity := range affinities {
		if _, ok := known[affinity.StyleID]; !ok {
			findings = append(findings, IndexFinding{
				Kind:       IndexFindingStaleDesignStyle,
				SourcePath: path,
				Field:      "designStyles",
				Expected:   "known design style",
				Actual:     affinity.StyleID,
				Detail:     fmt.Sprintf("component design affinity references unknown style id %q", affinity.StyleID),
			})
		}
	}
	return findings, nil
}

func isValidDesignAffinity(affinity DesignAffinity) bool {
	switch affinity {
	case DesignAffinityNative, DesignAffinityCompatible, DesignAffinityDiscouraged:
		return true
	default:
		return false
	}
}

func normalizeComponentSlot(raw string) (ComponentSlot, error) {
	slot := ComponentSlot(strings.ToLower(strings.TrimSpace(raw)))
	if slot == "" {
		return "", nil
	}
	switch slot {
	case ComponentSlotUIPrimitive, ComponentSlotUIPattern, ComponentSlotLayoutNav:
		return slot, nil
	default:
		return "", fmt.Errorf("invalid slot %q", raw)
	}
}

func isValidVersionStatus(status ComponentVersionStatus) bool {
	switch status {
	case VersionStatusDraft, VersionStatusReleased, VersionStatusDeprecated, VersionStatusArchived:
		return true
	default:
		return false
	}
}

func errInvalidHeader(path, field, reason string) ErrInvalidHeader {
	return ErrInvalidHeader{SourcePath: path, Field: field, Reason: reason}
}

// headerBlockRe captures the first /** … */ comment block in a file.
// (?s) flag makes . match newlines.
var headerBlockRe = regexp.MustCompile(`(?s)/\*\*(.*?)\*/`)

// headerFieldRe captures @field-name value pairs on each header line.
// The leading `*` and surrounding whitespace are stripped by the
// caller before this matches.
var headerFieldRe = regexp.MustCompile(`^@([A-Za-z][A-Za-z0-9_-]*)\s+(.*)$`)

// extractHeaderBlock returns the inner text of the first JSDoc-style
// header block, or ok=false if no such block exists. Only blocks that
// contain `@libraryId` are treated as library headers; otherwise the
// indexer would claim every JSDoc-commented file.
func extractHeaderBlock(src string) (string, bool) {
	matches := headerBlockRe.FindStringSubmatch(src)
	if len(matches) < 2 {
		return "", false
	}
	body := matches[1]
	if !strings.Contains(body, "@libraryId") {
		return "", false
	}
	return body, true
}

// parseHeader extracts field/value pairs from a header block. Multi-
// line values are folded onto the @field line until the next @field
// or end-of-block (so JSDoc continuation lines are tolerated).
func parseHeader(path, body string) (map[string]string, error) {
	out := map[string]string{}
	var currentField string
	var currentValue strings.Builder

	flush := func() {
		if currentField == "" {
			return
		}
		out[currentField] = strings.TrimSpace(currentValue.String())
		currentField = ""
		currentValue.Reset()
	}

	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := headerFieldRe.FindStringSubmatch(line); m != nil {
			flush()
			currentField = m[1]
			currentValue.WriteString(m[2])
			continue
		}
		if currentField != "" {
			currentValue.WriteByte(' ')
			currentValue.WriteString(line)
		}
	}
	flush()

	if _, ok := out["libraryId"]; !ok {
		return nil, ErrInvalidHeader{SourcePath: path, Field: "libraryId", Reason: "required"}
	}
	return out, nil
}

func digestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
