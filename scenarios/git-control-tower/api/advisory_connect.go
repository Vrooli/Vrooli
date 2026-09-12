package main

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	domainadvisory "git-control-tower/internal/advisory"
	advisoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	advisoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory/advisory_v1connect"
)

var _ advisoryconnect.AdvisoryServiceHandler = (*advisoryConnectServer)(nil)

type advisoryConnectServer struct{}

func (advisoryConnectServer) Draft(_ context.Context, req *connect.Request[advisoryv1.DraftRequest]) (*connect.Response[advisoryv1.DraftResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("draft request is required"))
	}

	response, err := buildAdvisoryDraft(advisoryDraftRequest{
		Kind:     req.Msg.GetKind(),
		Subject:  advisorySubjectFromProto(req.Msg.GetSubject()),
		Evidence: advisoryEvidenceFromProto(req.Msg.GetEvidence()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&advisoryv1.DraftResponse{
		Status:        response.Status,
		Kind:          response.Kind,
		SubjectDigest: response.SubjectDigest,
		Title:         response.Title,
		Body:          response.Body,
		Unknowns:      append([]string(nil), response.Unknowns...),
		Coverage:      advisoryCoverageToProto(response.Coverage),
		EvidenceRefs:  append([]string(nil), response.EvidenceRefs...),
	}), nil
}

func advisorySubjectFromProto(subject *advisoryv1.ChangeSubject) domainadvisory.ChangeSubject {
	if subject == nil {
		return domainadvisory.ChangeSubject{}
	}
	result := domainadvisory.ChangeSubject{
		RepositoryID:   subject.GetRepositoryId(),
		Kind:           domainadvisory.SubjectKind(subject.GetKind()),
		BaseRevision:   subject.GetBaseRevision(),
		HeadRevision:   subject.GetHeadRevision(),
		ParentRevision: subject.GetParentRevision(),
		SnapshotDigest: subject.GetSnapshotDigest(),
	}
	if scope := subject.GetScope(); scope != nil {
		result.Scope = domainadvisory.Scope{
			Paths:           append([]string(nil), scope.GetPaths()...),
			SelectionDigest: scope.GetSelectionDigest(),
		}
	}
	if host := subject.GetHost(); host != nil {
		result.Host = &domainadvisory.HostSubject{
			Provider:     host.GetProvider(),
			InstanceID:   host.GetInstanceId(),
			RepositoryID: host.GetRepositoryId(),
			ChangeNumber: host.GetChangeNumber(),
		}
	}
	return result
}

func advisoryEvidenceFromProto(evidence *advisoryv1.EvidenceBundle) domainadvisory.EvidenceBundle {
	if evidence == nil {
		return domainadvisory.EvidenceBundle{}
	}
	result := domainadvisory.EvidenceBundle{
		OperationID:   evidence.GetOperationId(),
		SubjectDigest: evidence.GetSubjectDigest(),
		Status:        evidence.GetStatus(),
		Unknowns:      append([]string(nil), evidence.GetUnknowns()...),
		Versions:      evidence.GetVersions(),
	}
	for _, claim := range evidence.GetClaims() {
		if claim == nil {
			continue
		}
		result.Claims = append(result.Claims, domainadvisory.Claim{
			Text:         claim.GetText(),
			EvidenceRefs: append([]string(nil), claim.GetEvidenceRefs()...),
		})
	}
	if coverage := evidence.GetCoverage(); coverage != nil {
		result.Coverage = domainadvisory.Coverage{
			IncludedFiles: int(coverage.GetIncludedFiles()),
			OmittedFiles:  int(coverage.GetOmittedFiles()),
		}
		for _, omission := range coverage.GetOmissions() {
			if omission == nil {
				continue
			}
			result.Coverage.Omissions = append(result.Coverage.Omissions, domainadvisory.Omission{Path: omission.GetPath(), Reason: omission.GetReason()})
		}
	}
	for _, validation := range evidence.GetValidation() {
		if validation == nil {
			continue
		}
		result.Validation = append(result.Validation, domainadvisory.ValidationRef{
			ExecutionID:  validation.GetExecutionId(),
			Availability: validation.GetAvailability(),
			Verdict:      validation.GetVerdict(),
		})
	}
	return result
}

func advisoryCoverageToProto(coverage domainadvisory.Coverage) *advisoryv1.Coverage {
	result := &advisoryv1.Coverage{
		IncludedFiles: int32(coverage.IncludedFiles),
		OmittedFiles:  int32(coverage.OmittedFiles),
	}
	for _, omission := range coverage.Omissions {
		result.Omissions = append(result.Omissions, &advisoryv1.Omission{Path: omission.Path, Reason: omission.Reason})
	}
	return result
}
