package adoptions

import (
	"context"
	"errors"
	"strings"
)

// Reconverge batch-reconverges BEHIND adoptions to the current library version
// — it closes the drift loop the Refresh machinery already detects. For each
// BEHIND adoption it re-applies only CLEAN copies (unmodified vs their recorded
// snapshot) and flags MODIFIED copies for human review rather than overwriting
// them. It never bypasses server-side Apply validation: every re-apply goes
// through the same Reapply primitive, so the dependency (block) and style-fit
// (warn) gates still run.
//
// Dry-run is the default; Apply is the only mode that writes scenario files.
// The result reports per-adoption and per-file outcomes so an operator can see
// exactly what moved and what needs review.
func (s *service) Reconverge(ctx context.Context, in ReconvergeInput) (ReconvergeResult, error) {
	rows, err := s.repo.List(ctx, ListQuery{Scenario: strings.TrimSpace(in.Scenario), Limit: 100000})
	if err != nil {
		return ReconvergeResult{}, err
	}
	result := ReconvergeResult{}
	for _, row := range rows {
		result.Scanned++
		libStatus, localStatus, detail := s.computeStatus(ctx, row)
		// Only BEHIND adoptions are reconverge candidates. current / deprecated
		// / missing / unknown fall outside the drift-reconverge contract.
		if libStatus != LibraryVersionStatusBehind {
			continue
		}
		result.Behind++
		outcome := ReconvergeOutcome{
			AdoptionID:           row.ID,
			Scenario:             row.Scenario,
			ComponentID:          row.ComponentID,
			LibraryID:            row.LibraryID,
			AdoptedVersion:       row.AdoptedVersion,
			TargetVersion:        s.currentLibraryVersion(ctx, row.ComponentID),
			LibraryVersionStatus: libStatus,
			LocalStatus:          localStatus,
			Detail:               detail,
			Files:                s.reconvergeFileOutcomes(ctx, row),
		}
		switch localStatus {
		case LocalStatusModified:
			// Never auto-overwrite a locally edited copy. Surface it instead.
			outcome.Action = ReconvergeActionFlaggedModified
			result.Flagged++
		case LocalStatusClean:
			if !in.Apply {
				outcome.Action = ReconvergeActionWouldReapply
			} else {
				// Reapply without ConfirmLocalOverwrite/OverrideValidation keeps
				// the modified-guard and both validation gates authoritative.
				updated, _, rerr := s.Reapply(ctx, ReapplyInput{ID: row.ID})
				if rerr != nil {
					outcome.Action = ReconvergeActionError
					outcome.Detail = rerr.Error()
					result.Errored++
				} else {
					outcome.Action = ReconvergeActionReapplied
					outcome.TargetVersion = updated.AdoptedVersion
					result.Reapplied++
				}
			}
		default: // missing / unknown
			outcome.Action = ReconvergeActionSkippedUnresolved
			result.Skipped++
		}
		result.Outcomes = append(result.Outcomes, outcome)
	}
	return result, nil
}

// currentLibraryVersion returns the library's current version for a component,
// or "" when the component cannot be resolved (so the caller can render it as
// unknown without failing the whole batch).
func (s *service) currentLibraryVersion(ctx context.Context, componentID string) string {
	cmp, err := s.library.Get(ctx, componentID)
	if err != nil {
		return ""
	}
	return firstNonEmpty(cmp.LatestVersion, cmp.Version)
}

// reconvergeFileOutcomes computes the per-file CLEAN/MODIFIED/MISSING drift of
// a vendored unit by comparing each adopted file against its recorded snapshot.
// It mirrors the file loop in computeStatus but reports every file rather than
// short-circuiting on the first divergence, so the operator sees the full unit.
func (s *service) reconvergeFileOutcomes(ctx context.Context, row Adoption) []ReconvergeFileOutcome {
	files := row.Files
	if len(files) == 0 {
		files = []AdoptionFile{{AdoptedPath: row.AdoptedPath, AdoptedSnapshotSHA256: row.AdoptedSnapshotSHA256}}
	}
	out := make([]ReconvergeFileOutcome, 0, len(files))
	for _, file := range files {
		status := LocalStatusClean
		body, err := s.files.Read(ctx, row.Scenario, file.AdoptedPath)
		switch {
		case err != nil:
			var missing ErrAdoptedFileMissing
			if errors.As(err, &missing) {
				status = LocalStatusMissing
			} else {
				status = LocalStatusUnknown
			}
		case file.AdoptedSnapshotSHA256 != "" && hashBytes(body) != file.AdoptedSnapshotSHA256:
			status = LocalStatusModified
		}
		out = append(out, ReconvergeFileOutcome{LibraryPath: file.LibraryPath, AdoptedPath: file.AdoptedPath, LocalStatus: status})
	}
	return out
}
