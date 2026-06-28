package assignments

import (
	"brand-manager/internal/assignments"

	assignmentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// assignmentToProto converts an internal Assignment into the wire shape. Lives
// in the handler package by intent — the conversion is mechanical and only used
// at the transport edge.
func assignmentToProto(a assignments.Assignment) *assignmentsv1.Assignment {
	return &assignmentsv1.Assignment{
		Id:           a.ID,
		BrandId:      a.BrandID,
		ScenarioName: a.ScenarioName,
		BrandVersion: int32(a.BrandVersion),
		Elements:     append([]string(nil), a.Elements...),
		AppliedAt:    timestamppb.New(a.AppliedAt.UTC()),
	}
}

// statusToProto converts an internal ScenarioStatus into the wire shape. When
// the scenario has no brand the optional fields stay zero-valued (applied_at is
// left nil rather than the Unix epoch).
func statusToProto(s assignments.ScenarioStatus) *assignmentsv1.ScenarioStatus {
	out := &assignmentsv1.ScenarioStatus{
		Scenario:     s.Scenario,
		HasBrand:     s.HasBrand,
		BrandId:      s.BrandID,
		BrandVersion: int32(s.BrandVersion),
		Elements:     append([]string(nil), s.Elements...),
	}
	if s.HasBrand && !s.AppliedAt.IsZero() {
		out.AppliedAt = timestamppb.New(s.AppliedAt.UTC())
	}
	return out
}
