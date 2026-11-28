package adoptions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"react-component-library/internal/components"
	"strings"
	"time"
)

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// adoptedSnapshotHash deliberately excludes the generated provenance header.
// A dependency can be shared by multiple root adoptions in one scenario; its
// provenance owner may change while the translated source remains identical.
// Drift must describe source edits, not which root wrote the shared file last.
func adoptedSnapshotHash(body string) string {
	return hashBytes([]byte(stripSourceHeader(body)))
}

func emptyOrVersion(v string) string {
	if v == "" {
		return "(no version)"
	}
	return v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func formatProvenance(v components.ComponentVersion, adoptionID string, appliedAt time.Time, driftHash, translationNote string) string {
	// JSDoc tag names align 1:1 with ui-health's ComponentProvenance proto:
	//   @vrooliComponentSource       -> library
	//   @vrooliComponentVersion      -> library_version
	//   @vrooliComponentAdoption     -> adoption_id
	//   @vrooliComponentAppliedAt    -> applied_at
	//   @vrooliComponentSourceSha256 -> source_sha256
	//   @vrooliComponentDriftHash    -> drift_hash (equal to source_sha256 at
	//                                   adoption time; recomputed at scan time)
	if driftHash == "" {
		driftHash = v.ContentSHA256
	}
	return fmt.Sprintf(`/**
 * @vrooliComponentSource %s
 * @vrooliComponentVersion %s
 * @vrooliComponentAdoption %s
 * @vrooliComponentAppliedAt %s
 * @vrooliComponentSourceSha256 %s
 * @vrooliComponentDriftHash %s
 * @vrooliComponentTokenTranslation %s
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
`, v.LibraryID, v.Version, adoptionID, appliedAt.UTC().Format(time.RFC3339), v.ContentSHA256, driftHash, translationNote)
}

func formatTokenTranslations(translations []TokenTranslation) string {
	if len(translations) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(translations))
	for _, translation := range translations {
		parts = append(parts, translation.From+"->"+translation.To)
	}
	return strings.Join(parts, ",")
}

func stripSourceHeader(src string) string {
	trimmed := strings.TrimLeft(src, "\ufeff \t\r\n")
	if strings.HasPrefix(trimmed, "// @vrooliComponent") || strings.HasPrefix(trimmed, "// @libraryId") {
		if end := strings.IndexByte(trimmed, '\n'); end >= 0 {
			return strings.TrimLeft(trimmed[end+1:], "\r\n")
		}
		return ""
	}
	if !strings.HasPrefix(trimmed, "/**") {
		return src
	}
	end := strings.Index(trimmed, "*/")
	if end < 0 {
		return src
	}
	body := trimmed[:end+2]
	if !strings.Contains(body, "@libraryId") && !strings.Contains(body, "@vrooliComponent") {
		return src
	}
	return strings.TrimLeft(trimmed[end+2:], "\r\n")
}
