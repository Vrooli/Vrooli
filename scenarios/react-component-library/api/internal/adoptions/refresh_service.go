package adoptions

import (
	"context"
	"errors"
	"fmt"
	"log"
	"react-component-library/internal/components"
	"react-component-library/internal/versions"
	"strings"
)

func (s *service) Refresh(ctx context.Context, componentID string) ([]Adoption, RefreshSummary, error) {
	return s.refresh(ctx, componentID, true)
}

// ForkReport classifies the complete adoption registry using the same
// evidence as Refresh. Dry-run is the default so operators can inspect every
// row before persisting the four-way disposition; apply only updates the
// registry metadata and never rewrites scenario source files.
func (s *service) ForkReport(ctx context.Context, componentID string, apply bool) ([]Adoption, RefreshSummary, error) {
	return s.refresh(ctx, componentID, apply)
}

func (s *service) refresh(ctx context.Context, componentID string, apply bool) ([]Adoption, RefreshSummary, error) {
	rows, err := s.repo.List(ctx, ListQuery{ComponentID: strings.TrimSpace(componentID), Limit: refreshListLimit})
	if err != nil {
		return nil, RefreshSummary{}, err
	}
	now := s.clock.Now().UTC()
	updates := make([]RefreshUpdate, 0, len(rows))
	summary := RefreshSummary{}
	for i, row := range rows {
		libraryStatus, localStatus, detail := s.computeStatus(ctx, row)
		rows[i].LibraryVersionStatus = libraryStatus
		rows[i].LocalStatus = localStatus
		rows[i].StatusDetail = detail
		rows[i].RefreshedAt = now
		update := RefreshUpdate{
			ID: row.ID, LibraryVersionStatus: libraryStatus, LocalStatus: localStatus, StatusDetail: detail, RefreshedAt: now,
		}
		// Automated classifications are observations, not operator declarations.
		// Recompute them on refresh so improving the classifier repairs earlier
		// coarse results and every registry row has an explicit disposition. A
		// declared fork remains authoritative and is never overwritten here.
		if row.ForkStatus == ForkStatusDeclared {
			update.ForkStatus = ForkStatusDeclared
		} else if row.Mode == AdoptionModeLinked {
			update.ForkStatus = ForkStatusContractPreserved
		} else if localStatus == LocalStatusModified {
			update.ForkStatus = forkStatusForDisposition(s.classifyModified(ctx, row))
		} else if localStatus == LocalStatusClean && s.verifiedCleanAgainstLibrary(ctx, row) {
			update.ForkStatus = ForkStatusMechanicalTranslation
		} else {
			update.ForkStatus = ForkStatusLocalFork
		}
		rows[i].ForkStatus = update.ForkStatus
		// Drift policy:
		//   * status flips to behind/modified AND no backlog item filed
		//     yet → call the reporter and store the returned ref so the
		//     next refresh skips it.
		//   * status returns to current → clear the stored ref so a
		//     fresh drift files a new item rather than being silently
		//     swallowed.
		if apply && s.reporter != nil {
			switch {
			case (libraryStatus == LibraryVersionStatusBehind || localStatus == LocalStatusModified) && row.DriftBacklogRef == "":
				ev := s.buildDriftEvent(ctx, rows[i])
				report, err := s.reporter.Report(ctx, ev)
				if err != nil {
					s.logf("drift reporter for adoption %q failed: %v", row.ID, err)
				} else if ref := strings.TrimSpace(report.Ref); ref != "" {
					update.DriftBacklogRef = ref
					rows[i].DriftBacklogRef = ref
				}
			case libraryStatus == LibraryVersionStatusCurrent && localStatus == LocalStatusClean && row.DriftBacklogRef != "":
				update.ClearDriftBacklogRef = true
				rows[i].DriftBacklogRef = ""
			}
		}
		updates = append(updates, update)
		switch libraryStatus {
		case LibraryVersionStatusCurrent:
			summary.LibraryCurrent++
		case LibraryVersionStatusBehind:
			summary.LibraryBehind++
		case LibraryVersionStatusDeprecated:
			summary.LibraryDeprecated++
		case LibraryVersionStatusMissing:
			summary.LibraryMissing++
		case LibraryVersionStatusUnknown:
			summary.LibraryUnknown++
		case LibraryVersionStatusSourceDrifted:
			summary.LibrarySourceDrifted++
		}
		switch localStatus {
		case LocalStatusClean:
			summary.LocalClean++
		case LocalStatusModified:
			summary.LocalModified++
		case LocalStatusMissing:
			summary.LocalMissing++
		case LocalStatusUnknown:
			summary.LocalUnknown++
		}
	}
	if apply {
		if _, err := s.repo.ApplyRefresh(ctx, updates); err != nil {
			return nil, RefreshSummary{}, err
		}
	}
	return rows, summary, nil
}

// computeStatus is the AD-002 drift decision tree. Pure function of
// the row + the two reader seams.
func (s *service) computeStatus(ctx context.Context, row Adoption) (LibraryVersionStatus, LocalStatus, string) {
	cmp, err := s.library.Get(ctx, row.ComponentID)
	if err != nil {
		if errors.As(err, &components.ErrComponentNotFound{}) {
			return LibraryVersionStatusMissing, LocalStatusUnknown, "component removed from library"
		}
		return LibraryVersionStatusUnknown, LocalStatusUnknown, fmt.Sprintf("library lookup failed: %v", err)
	}
	files := row.Files
	if len(files) == 0 {
		files = []AdoptionFile{{AdoptedPath: row.AdoptedPath, AdoptedSnapshotSHA256: row.AdoptedSnapshotSHA256}}
	}
	localStatus := LocalStatusClean
	detail := ""
	for _, file := range files {
		adoptedBytes, err := s.files.Read(ctx, row.Scenario, file.AdoptedPath)
		if err == nil {
			if file.AdoptedSnapshotSHA256 != "" && adoptedSnapshotHash(string(adoptedBytes)) != file.AdoptedSnapshotSHA256 {
				localStatus, detail = LocalStatusModified, fmt.Sprintf("adopted file %s diverges from applied snapshot", file.AdoptedPath)
				break
			}
			continue
		}
		var missing ErrAdoptedFileMissing
		if errors.As(err, &missing) {
			libStatus, libDetail := s.libraryStatusFor(ctx, row, cmp)
			return libStatus, LocalStatusMissing, firstNonEmpty("adopted file missing: "+file.AdoptedPath, libDetail)
		}
		libStatus, libDetail := s.libraryStatusFor(ctx, row, cmp)
		return libStatus, LocalStatusUnknown, firstNonEmpty(fmt.Sprintf("adopted file read failed: %v", err), libDetail)
	}
	libStatus, libDetail := s.libraryStatusFor(ctx, row, cmp)
	if libDetail != "" {
		detail = libDetail
	}
	return libStatus, localStatus, detail
}

// buildDriftEvent assembles the payload handed to the reporter. The
// LibraryVersion lookup is best-effort: if the library reader fails
// (it shouldn't, since computeStatus just succeeded for this row), the
// event still goes out without a library version.
func (s *service) buildDriftEvent(ctx context.Context, row Adoption) DriftEvent {
	ev := DriftEvent{
		AdoptionID:           row.ID,
		ComponentID:          row.ComponentID,
		LibraryID:            row.LibraryID,
		Scenario:             row.Scenario,
		AdoptedPath:          row.AdoptedPath,
		AdoptedVersion:       row.AdoptedVersion,
		LibraryVersionStatus: row.LibraryVersionStatus,
		LocalStatus:          row.LocalStatus,
		StatusDetail:         row.StatusDetail,
	}
	if cmp, err := s.library.Get(ctx, row.ComponentID); err == nil {
		ev.LibraryVersion = firstNonEmpty(cmp.LatestVersion, cmp.Version)
		if ev.LibraryID == "" {
			ev.LibraryID = cmp.LibraryID
		}
	}
	return ev
}

func (s *service) logf(format string, args ...any) {
	if s.logger == nil {
		log.Default().Printf(format, args...)
		return
	}
	s.logger.Printf(format, args...)
}

func (s *service) libraryStatusFor(ctx context.Context, row Adoption, cmp components.Component) (LibraryVersionStatus, string) {
	status, detail, needsVersion := libraryStatusFor(row, cmp, components.ComponentVersionStatus(""))
	if !needsVersion {
		return status, detail
	}
	version, err := s.library.GetVersion(ctx, row.ComponentID, row.AdoptedVersion)
	if err != nil {
		return LibraryVersionStatusUnknown, fmt.Sprintf("adopted version %s not found in library", emptyOrVersion(row.AdoptedVersion))
	}
	if row.SourceSHA256 != "" && version.ContentSHA256 != "" && row.SourceSHA256 != version.ContentSHA256 {
		return LibraryVersionStatusSourceDrifted, fmt.Sprintf("source bytes for released version %s changed: recorded %s, current %s", row.AdoptedVersion, row.SourceSHA256, version.ContentSHA256)
	}
	status, detail, _ = libraryStatusFor(row, cmp, version.Status)
	return status, detail
}

func libraryStatusFor(row Adoption, cmp components.Component, adoptedStatus components.ComponentVersionStatus) (LibraryVersionStatus, string, bool) {
	latest := firstNonEmpty(cmp.LatestVersion, cmp.Version)
	if row.AdoptedVersion == "" || latest == "" {
		return LibraryVersionStatusUnknown, "", false
	}
	if adoptedStatus == "" {
		return LibraryVersionStatusUnknown, "", true
	}
	if adoptedStatus == components.VersionStatusDeprecated {
		return LibraryVersionStatusDeprecated, fmt.Sprintf("adopted version %s is deprecated", row.AdoptedVersion), false
	}
	if adoptedStatus == components.VersionStatusDraft {
		return LibraryVersionStatusUnknown, fmt.Sprintf("adopted version %s is a draft", row.AdoptedVersion), false
	}
	if row.AdoptedVersion == latest {
		return LibraryVersionStatusCurrent, "", false
	}
	cmpResult, ok := versions.CompareVersionCoreLabels(row.AdoptedVersion, latest)
	if ok && cmpResult >= 0 {
		return LibraryVersionStatusCurrent, "", false
	}
	return LibraryVersionStatusBehind, fmt.Sprintf("library at %s", emptyOrVersion(latest)), false
}
