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
