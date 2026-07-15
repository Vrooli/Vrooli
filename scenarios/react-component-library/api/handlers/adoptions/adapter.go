package adoptions

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"react-component-library/internal/adoptions"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
)

// domainToProto converts an internal adoptions.Adoption into the wire
// shape the adoptions proto declares.
func domainToProto(a adoptions.Adoption) *adoptionsv1.Adoption {
	out := &adoptionsv1.Adoption{
		Id:                   a.ID,
		ComponentId:          a.ComponentID,
		LibraryId:            a.LibraryID,
		Scenario:             a.Scenario,
		AdoptedPath:          a.AdoptedPath,
		AdoptedVersion:       a.AdoptedVersion,
		LibraryVersionStatus: libraryStatusToProto(a.LibraryVersionStatus),
		LocalStatus:          localStatusToProto(a.LocalStatus),
		StatusDetail:         a.StatusDetail,
		CreatedAt:            timestamppb.New(a.CreatedAt.UTC()),
		SourceSha256:         a.SourceSHA256,
		Files:                adoptionFilesToProto(a.Files),
	}
	if !a.RefreshedAt.IsZero() {
		out.RefreshedAt = timestamppb.New(a.RefreshedAt.UTC())
	}
	if !a.AppliedAt.IsZero() {
		out.AppliedAt = a.AppliedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return out
}

func adoptionFilesToProto(files []adoptions.AdoptionFile) []*adoptionsv1.AdoptionFile {
	out := make([]*adoptionsv1.AdoptionFile, 0, len(files))
	for _, file := range files {
		out = append(out, &adoptionsv1.AdoptionFile{LibraryPath: file.LibraryPath, AdoptedPath: file.AdoptedPath, SourceSha256: file.SourceSHA256, AdoptedSnapshotSha256: file.AdoptedSnapshotSHA256})
	}
	return out
}

func reconvergeOutcomeToProto(o adoptions.ReconvergeOutcome) *adoptionsv1.ReconvergeOutcome {
	out := &adoptionsv1.ReconvergeOutcome{
		AdoptionId:           o.AdoptionID,
		Scenario:             o.Scenario,
		ComponentId:          o.ComponentID,
		LibraryId:            o.LibraryID,
		AdoptedVersion:       o.AdoptedVersion,
		TargetVersion:        o.TargetVersion,
		LibraryVersionStatus: libraryStatusToProto(o.LibraryVersionStatus),
		LocalStatus:          localStatusToProto(o.LocalStatus),
		Action:               reconvergeActionToProto(o.Action),
		Detail:               o.Detail,
	}
	for _, file := range o.Files {
		out.Files = append(out.Files, &adoptionsv1.ReconvergeFileOutcome{
			LibraryPath: file.LibraryPath,
			AdoptedPath: file.AdoptedPath,
			LocalStatus: localStatusToProto(file.LocalStatus),
		})
	}
	return out
}

func reconvergeActionToProto(a adoptions.ReconvergeAction) adoptionsv1.ReconvergeAction {
	switch a {
	case adoptions.ReconvergeActionReapplied:
		return adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_REAPPLIED
	case adoptions.ReconvergeActionWouldReapply:
		return adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_WOULD_REAPPLY
	case adoptions.ReconvergeActionFlaggedModified:
		return adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_FLAGGED_MODIFIED
	case adoptions.ReconvergeActionSkippedUnresolved:
		return adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_SKIPPED_UNRESOLVED
	case adoptions.ReconvergeActionError:
		return adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_ERROR
	}
	return adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_UNSPECIFIED
}

func libraryStatusToProto(s adoptions.LibraryVersionStatus) adoptionsv1.LibraryVersionStatus {
	switch s {
	case adoptions.LibraryVersionStatusCurrent:
		return adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_CURRENT
	case adoptions.LibraryVersionStatusBehind:
		return adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_BEHIND
	case adoptions.LibraryVersionStatusDeprecated:
		return adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_DEPRECATED
	case adoptions.LibraryVersionStatusMissing:
		return adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_MISSING
	case adoptions.LibraryVersionStatusUnknown:
		return adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_UNKNOWN
	}
	return adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_UNSPECIFIED
}

func localStatusToProto(s adoptions.LocalStatus) adoptionsv1.LocalStatus {
	switch s {
	case adoptions.LocalStatusClean:
		return adoptionsv1.LocalStatus_LOCAL_STATUS_CLEAN
	case adoptions.LocalStatusModified:
		return adoptionsv1.LocalStatus_LOCAL_STATUS_MODIFIED
	case adoptions.LocalStatusMissing:
		return adoptionsv1.LocalStatus_LOCAL_STATUS_MISSING
	case adoptions.LocalStatusUnknown:
		return adoptionsv1.LocalStatus_LOCAL_STATUS_UNKNOWN
	}
	return adoptionsv1.LocalStatus_LOCAL_STATUS_UNSPECIFIED
}
