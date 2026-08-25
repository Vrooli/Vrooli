package adoptions

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"react-component-library/internal/components"
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
			ForkStatus:           row.ForkStatus,
		}
		if row.ForkStatus == ForkStatusDeclared {
			outcome.Action = ReconvergeActionFlaggedModified
			outcome.Disposition = ReconvergeDispositionLocalFork
			outcome.Detail = "declared fork: " + row.ForkReason
			result.LocalFork++
			result.Flagged++
			result.Outcomes = append(result.Outcomes, outcome)
			continue
		}
		switch localStatus {
		case LocalStatusModified:
			// Never auto-overwrite a locally edited copy. Surface it instead.
			outcome.Action = ReconvergeActionFlaggedModified
			outcome.Disposition = s.classifyModified(ctx, row)
			switch outcome.Disposition {
			case ReconvergeDispositionTranslationOnly:
				result.TranslationOnly++
			case ReconvergeDispositionLocalAddition:
				result.LocalAddition++
			default:
				result.LocalFork++
			}
			outcome.Detail = string(outcome.Disposition)
			if outcome.Disposition == ReconvergeDispositionTranslationOnly {
				outcome.ForkStatus = ForkStatusMechanicalTranslation
			} else {
				outcome.ForkStatus = ForkStatusUnintendedDrift
			}
			result.Flagged++
		case LocalStatusClean:
			// Snapshot-based CLEAN is necessary but not sufficient to overwrite:
			// a snapshot captured from a locally-modified copy (the backfill
			// hash-masking defect) reads CLEAN yet the bytes are not the library's.
			// Re-derive the pristine body from the adopted library version and
			// require an exact body match before touching the file. A poisoned or
			// otherwise unverifiable copy is flagged for human review, never
			// silently overwritten.
			if !s.verifiedCleanAgainstLibrary(ctx, row) {
				outcome.Action = ReconvergeActionFlaggedModified
				outcome.LocalStatus = LocalStatusModified
				outcome.Disposition = ReconvergeDispositionLocalFork
				outcome.Detail = "on-disk body diverges from the adopted library version; refusing to overwrite (stale or poisoned snapshot)"
				result.LocalFork++
				result.Flagged++
			} else if !in.Apply {
				outcome.Action = ReconvergeActionWouldReapply
			} else {
				// Reapply without ConfirmLocalOverwrite/OverrideValidation keeps
				// the modified-guard and both validation gates authoritative.
				updated, _, rerr := s.Reapply(ctx, ReapplyInput{ID: row.ID})
				if rerr != nil {
					outcome.Action = ReconvergeActionError
					outcome.Detail = rerr.Error()
					var tokenErr ErrAdoptionTokensUnsatisfied
					if errors.As(rerr, &tokenErr) {
						outcome.Action = ReconvergeActionBlockedTokens
						outcome.Disposition = ReconvergeDispositionTokenBlocked
						result.TokenBlocked++
					}
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

// classifyModified compares a locally modified copy with the exact body the
// old adoption would have produced. Token translations and relative-import
// rewrites are mechanical; an exact mechanical match is translation_only.
// If every source line remains present and extra lines exist, it is a
// local_addition. Ambiguous content is a local_fork.
func (s *service) classifyModified(ctx context.Context, row Adoption) ReconvergeDisposition {
	v, err := s.library.GetVersion(ctx, row.ComponentID, row.AdoptedVersion)
	if err != nil {
		return ReconvergeDispositionLocalFork
	}
	files := row.Files
	if len(files) == 0 {
		files = []AdoptionFile{{LibraryPath: filepath.Base(row.AdoptedPath), AdoptedPath: row.AdoptedPath}}
	}
	targets := make(map[string]string, len(files))
	for _, file := range files {
		targets[moduleKey(firstNonEmpty(file.LibraryPath, filepath.Base(file.AdoptedPath)))] = file.AdoptedPath
	}
	mapping, err := s.resolveTokenMapping(ctx, row.Scenario)
	if err != nil {
		return ReconvergeDispositionLocalFork
	}
	addition := false
	for _, file := range files {
		libraryFile, ok := libraryFileByPath(v, file.LibraryPath)
		if !ok {
			return ReconvergeDispositionLocalFork
		}
		disk, err := s.files.Read(ctx, row.Scenario, file.AdoptedPath)
		if err != nil {
			return ReconvergeDispositionLocalFork
		}
		expected := expectedAppliedBody(libraryFile, file.AdoptedPath, targets)
		expected, _, err = TranslateDesignTokens(expected, mapping.Namespace, mapping)
		if err != nil {
			return ReconvergeDispositionLocalFork
		}
		local := stripSourceHeader(string(disk))
		if local == expected {
			continue
		}
		if containsEverySourceLine(local, expected) {
			addition = true
			continue
		}
		return ReconvergeDispositionLocalFork
	}
	if addition {
		return ReconvergeDispositionLocalAddition
	}
	return ReconvergeDispositionTranslationOnly
}

func containsEverySourceLine(local, source string) bool {
	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(local, line) {
			return false
		}
	}
	return true
}

// verifiedCleanAgainstLibrary re-derives, from the adopted library version, the
// exact bytes an apply would have written for every file in the unit and compares
// them (header ignored, import specifiers rewritten to the on-disk layout) to the
// current on-disk bytes. It returns true only when every file provably matches —
// the guarantee reconverge needs before overwriting a copy. A snapshot captured
// from a locally-modified copy (which makes LocalStatus read CLEAN), a file whose
// library body cannot be loaded, or an unreadable file all fail here, so a
// poisoned or unverifiable record can never be silently overwritten.
func (s *service) verifiedCleanAgainstLibrary(ctx context.Context, row Adoption) bool {
	v, err := s.library.GetVersion(ctx, row.ComponentID, row.AdoptedVersion)
	if err != nil {
		return false
	}
	files := row.Files
	if len(files) == 0 {
		files = []AdoptionFile{{LibraryPath: filepath.Base(row.AdoptedPath), AdoptedPath: row.AdoptedPath}}
	}
	targets := make(map[string]string, len(files))
	for _, f := range files {
		targets[moduleKey(firstNonEmpty(f.LibraryPath, filepath.Base(f.AdoptedPath)))] = f.AdoptedPath
	}
	for _, f := range files {
		libFile, ok := libraryFileByPath(v, f.LibraryPath)
		if !ok || strings.TrimSpace(libFile.Content) == "" {
			return false
		}
		onDisk, err := s.files.Read(ctx, row.Scenario, f.AdoptedPath)
		if err != nil {
			return false
		}
		if stripSourceHeader(string(onDisk)) != expectedAppliedBody(libFile, f.AdoptedPath, targets) {
			return false
		}
	}
	return true
}

// libraryFileByPath resolves the version file whose path matches libraryPath,
// falling back to the entry / single mirrored body when the version was stored
// without per-file rows. It mirrors matchReconciledLibraryFile's resolution so
// verification and reconciliation agree on which library body a unit maps to.
func libraryFileByPath(v components.ComponentVersion, libraryPath string) (components.ComponentVersionFile, bool) {
	name := strings.TrimSpace(libraryPath)
	for _, f := range v.Files {
		if f.Path == name {
			return f, true
		}
	}
	if len(v.Files) == 1 {
		return v.Files[0], true
	}
	for _, f := range v.Files {
		if f.IsEntry {
			return f, true
		}
	}
	if strings.TrimSpace(v.Content) != "" {
		return components.ComponentVersionFile{Path: firstNonEmpty(name, filepath.Base(v.SourcePath)), Content: v.Content, ContentSHA256: v.ContentSHA256, IsEntry: true}, true
	}
	return components.ComponentVersionFile{}, false
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
