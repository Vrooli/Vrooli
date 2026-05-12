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
		Id:             a.ID,
		ComponentId:    a.ComponentID,
		LibraryId:      a.LibraryID,
		Scenario:       a.Scenario,
		AdoptedPath:    a.AdoptedPath,
		AdoptedVersion: a.AdoptedVersion,
		Status:         statusToProto(a.Status),
		StatusDetail:   a.StatusDetail,
		CreatedAt:      timestamppb.New(a.CreatedAt.UTC()),
	}
	if !a.RefreshedAt.IsZero() {
		out.RefreshedAt = timestamppb.New(a.RefreshedAt.UTC())
	}
	return out
}

func statusToProto(s adoptions.Status) adoptionsv1.AdoptionStatus {
	switch s {
	case adoptions.StatusCurrent:
		return adoptionsv1.AdoptionStatus_ADOPTION_STATUS_CURRENT
	case adoptions.StatusBehind:
		return adoptionsv1.AdoptionStatus_ADOPTION_STATUS_BEHIND
	case adoptions.StatusModified:
		return adoptionsv1.AdoptionStatus_ADOPTION_STATUS_MODIFIED
	case adoptions.StatusUnknown:
		return adoptionsv1.AdoptionStatus_ADOPTION_STATUS_UNKNOWN
	}
	return adoptionsv1.AdoptionStatus_ADOPTION_STATUS_UNSPECIFIED
}
