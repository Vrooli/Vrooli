package findings

import (
	"web-search/internal/findings"

	"google.golang.org/protobuf/types/known/timestamppb"

	findingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings"
)

// statusToProto maps a domain status string to the proto enum.
func statusToProto(s string) findingsv1.FindingStatus {
	switch s {
	case findings.StatusActive:
		return findingsv1.FindingStatus_FINDING_STATUS_ACTIVE
	case findings.StatusDisputed:
		return findingsv1.FindingStatus_FINDING_STATUS_DISPUTED
	case findings.StatusSuperseded:
		return findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED
	default:
		return findingsv1.FindingStatus_FINDING_STATUS_UNSPECIFIED
	}
}

// statusFromProto maps a proto status enum to a domain status string. UNSPECIFIED
// maps to "" so callers treat it as "no filter".
func statusFromProto(s findingsv1.FindingStatus) string {
	switch s {
	case findingsv1.FindingStatus_FINDING_STATUS_ACTIVE:
		return findings.StatusActive
	case findingsv1.FindingStatus_FINDING_STATUS_DISPUTED:
		return findings.StatusDisputed
	case findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED:
		return findings.StatusSuperseded
	default:
		return ""
	}
}

func sourceToProto(s string) findingsv1.FindingSource {
	switch s {
	case findings.SourceManual:
		return findingsv1.FindingSource_FINDING_SOURCE_MANUAL
	case findings.SourceL2:
		return findingsv1.FindingSource_FINDING_SOURCE_L2
	case findings.SourceL3:
		return findingsv1.FindingSource_FINDING_SOURCE_L3
	default:
		return findingsv1.FindingSource_FINDING_SOURCE_UNSPECIFIED
	}
}

func sourceFromProto(s findingsv1.FindingSource) string {
	switch s {
	case findingsv1.FindingSource_FINDING_SOURCE_L2:
		return findings.SourceL2
	case findingsv1.FindingSource_FINDING_SOURCE_L3:
		return findings.SourceL3
	default:
		return findings.SourceManual
	}
}

func citationToProto(c findings.Citation) *findingsv1.Citation {
	out := &findingsv1.Citation{Id: c.ID, Url: c.URL, Title: c.Title}
	if !c.RetrievedAt.IsZero() {
		out.RetrievedAt = timestamppb.New(c.RetrievedAt)
	}
	return out
}

// domainToProto projects an internal Finding onto the wire shape.
func domainToProto(f findings.Finding) *findingsv1.Finding {
	out := &findingsv1.Finding{
		Id:           f.ID,
		Claim:        f.Claim,
		BriefId:      f.BriefID,
		Confidence:   f.Confidence,
		Status:       statusToProto(f.Status),
		Query:        f.Query,
		SupersededBy: f.SupersededBy,
		DisputeNote:  f.DisputeNote,
		Source:       sourceToProto(f.Source),
	}
	if !f.RetrievalDate.IsZero() {
		out.RetrievalDate = timestamppb.New(f.RetrievalDate)
	}
	if !f.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(f.CreatedAt)
	}
	if !f.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(f.UpdatedAt)
	}
	for _, c := range f.Citations {
		out.Citations = append(out.Citations, citationToProto(c))
	}
	return out
}
