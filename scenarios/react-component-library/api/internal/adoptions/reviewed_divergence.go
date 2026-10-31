package adoptions

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ReviewedDivergence is one explicitly-accepted mismatch between a vendored
// component copy (typically under templates/**, which is outside the RCL write
// boundary) and the catalog latest. It is an audit record, not a blanket
// suppression: it names the exact file, library, on-disk version, catalog
// version, and status it excuses, plus a human reason. A divergence is only
// tolerated when a reviewer has recorded one of these — mirroring the
// --accept-behavior-loss override semantics used by the ingest gate.
type ReviewedDivergence struct {
	Path            string `json:"path"`
	LibraryID       string `json:"libraryId"`
	VendoredVersion string `json:"vendoredVersion"`
	CatalogVersion  string `json:"catalogVersion"`
	Status          string `json:"status"`
	Reason          string `json:"reason"`
}

// ReviewedDivergenceAllowlist is the parsed allowlist file: a dated set of
// reviewed divergences scoped to a specific scan root, with a pointer to the
// review report that justifies them.
type ReviewedDivergenceAllowlist struct {
	ReviewedAt  string               `json:"reviewedAt"`
	Report      string               `json:"report"`
	ScanRoot    string               `json:"scanRoot"`
	Divergences []ReviewedDivergence `json:"divergences"`
}

// LoadReviewedDivergences reads and parses a reviewed-divergence allowlist file.
func LoadReviewedDivergences(path string) (ReviewedDivergenceAllowlist, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ReviewedDivergenceAllowlist{}, fmt.Errorf("read reviewed-divergence allowlist %s: %w", path, err)
	}
	var allow ReviewedDivergenceAllowlist
	if err := json.Unmarshal(raw, &allow); err != nil {
		return ReviewedDivergenceAllowlist{}, fmt.Errorf("parse reviewed-divergence allowlist %s: %w", path, err)
	}
	return allow, nil
}

// matches reports whether entry excuses finding. Every axis must line up: an
// allowlist entry for one version pair does not excuse a copy that has since
// drifted to a different version, and a "behind" entry does not excuse a
// "deprecated" finding.
func (entry ReviewedDivergence) matches(finding FileScanFinding) bool {
	return strings.EqualFold(strings.TrimSpace(entry.Path), strings.TrimSpace(finding.Path)) &&
		strings.EqualFold(strings.TrimSpace(entry.LibraryID), strings.TrimSpace(finding.LibraryID)) &&
		strings.EqualFold(strings.TrimSpace(entry.VendoredVersion), strings.TrimSpace(finding.AdoptedVersion)) &&
		strings.EqualFold(strings.TrimSpace(entry.CatalogVersion), strings.TrimSpace(finding.LatestVersion)) &&
		strings.EqualFold(strings.TrimSpace(entry.Status), strings.TrimSpace(string(finding.Status)))
}

// PartitionReviewedFindings splits scan findings against the allowlist.
//   - unreviewed: findings not excused by any allowlist entry — these must still
//     fail the parity gate.
//   - staleEntries: allowlist entries that matched no finding — the divergence
//     they excused is gone (e.g. the template was reconverged), so the entry is
//     dead audit weight and must be removed. Enforcing this keeps the allowlist
//     from silently rotting into a permanent suppression list.
func PartitionReviewedFindings(
	findings []FileScanFinding,
	allow ReviewedDivergenceAllowlist,
) (unreviewed []FileScanFinding, staleEntries []ReviewedDivergence) {
	matchedEntry := make([]bool, len(allow.Divergences))
	for _, finding := range findings {
		covered := false
		for i, entry := range allow.Divergences {
			if entry.matches(finding) {
				matchedEntry[i] = true
				covered = true
			}
		}
		if !covered {
			unreviewed = append(unreviewed, finding)
		}
	}
	for i, entry := range allow.Divergences {
		if !matchedEntry[i] {
			staleEntries = append(staleEntries, entry)
		}
	}
	return unreviewed, staleEntries
}
