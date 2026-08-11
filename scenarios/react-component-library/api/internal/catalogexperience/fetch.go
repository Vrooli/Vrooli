// Package catalogexperience adapts the live Experience Manager contract into
// catalog coverage evidence. It is intentionally best-effort: the catalog
// remains truthful when the manager is stopped by leaving these captures
// absent rather than manufacturing a pass.
package catalogexperience

import (
	"context"
	"strings"

	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/experience"
)

// Fetcher returns the latest exact-version reconciliation captures for one
// library implementation.
func Fetcher(repoRoot string) catalogcoverage.ExperienceCaptureFetcher {
	reader := experience.NewReader(repoRoot)
	return func(ctx context.Context, libraryID, version string) ([]catalogcoverage.ExperienceCapture, error) {
		name := strings.TrimPrefix(strings.TrimSpace(libraryID), "react-component-library:")
		snapshot, err := reader.Get(ctx, experience.Component{LibraryID: libraryID, Slug: name, Version: version})
		if err != nil || snapshot.EvidenceStatus != "available" {
			return nil, err
		}
		claimTypes := make(map[string]string, len(snapshot.Claims))
		for _, claim := range snapshot.Claims {
			claimTypes[claim.ID] = claim.Type
		}
		out := make([]catalogcoverage.ExperienceCapture, 0, len(snapshot.Evidence))
		for _, item := range snapshot.Evidence {
			claimType := item.ClaimType
			// Older Experience Manager records did not persist claim_type. The
			// exact version contract is the authoritative fallback, while
			// unknown inherited floor claims remain intentionally unmapped.
			if claimType == "" {
				claimType = claimTypes[item.ClaimID]
			}
			out = append(out, catalogcoverage.ExperienceCapture{
				ClaimID:     item.ClaimID,
				ClaimType:   claimType,
				Verdict:     item.Verdict,
				StateID:     item.StateID,
				ExampleName: item.ExampleName,
				Viewport:    item.Viewport,
				CaptureRef:  item.CaptureRef,
				CheckedAt:   item.CheckedAt,
			})
		}
		return out, nil
	}
}
