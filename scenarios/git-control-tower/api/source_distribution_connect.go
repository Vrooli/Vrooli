package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/source_distributions"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/source_distributions/source_distributions_v1connect"
)

var _ sourceconnect.SourceDistributionServiceHandler = (*sourceDistributionConnectServer)(nil)

type sourceDistributionConnectServer struct{ server *Server }

func (s sourceDistributionConnectServer) ListDistributions(ctx context.Context, req *connect.Request[sourcev1.ListDistributionsRequest]) (*connect.Response[sourcev1.ListDistributionsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("distribution list request is required"))
	}
	if s.server.sourceDistributions == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("source-ramp reader is not configured"))
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	items, owner, freshness, err := s.server.sourceDistributions.List(ctx, req.Msg.GetScenario(), req.Msg.GetRepositoryContext(), limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("source-ramp list unavailable: %s", safeSourceError(err)))
	}
	resp := &sourcev1.ListDistributionsResponse{Available: true, SourceOfTruth: firstNonEmpty(owner, "scenario-to-repository"), Freshness: firstNonEmpty(freshness, "unknown"), Distributions: make([]*sourcev1.Distribution, 0, len(items))}
	for _, item := range items {
		resp.Distributions = append(resp.Distributions, distributionToConnect(mapDistribution(item, owner, freshness)))
	}
	return connect.NewResponse(resp), nil
}

func (s sourceDistributionConnectServer) GetDistributionDetails(ctx context.Context, req *connect.Request[sourcev1.GetDistributionDetailsRequest]) (*connect.Response[sourcev1.DistributionDetails], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetDistributionId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("distribution id is required"))
	}
	if s.server.sourceDistributions == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("source-ramp reader is not configured"))
	}
	id := strings.TrimSpace(req.Msg.GetDistributionId())
	d, err := s.server.sourceDistributions.Get(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("source-ramp distribution unavailable: %s", safeSourceError(err)))
	}
	detail := &sourcev1.DistributionDetails{Distribution: distributionToConnect(mapDistribution(d, d.GetSourceOfTruth(), d.GetFreshness())), Available: true}
	if detail.Distribution.SourceOfTruth == "" {
		detail.Distribution.SourceOfTruth = "scenario-to-repository"
	}
	if contents, e := s.server.sourceDistributions.Contents(ctx, id); e == nil {
		for _, x := range contents.GetContents() {
			detail.Contents = append(detail.Contents, &sourcev1.DistributionContent{Path: x.GetPath(), SourcePath: x.GetSourcePath(), Category: x.GetCategory(), Digest: x.GetDigest(), SizeBytes: x.GetSizeBytes()})
		}
		for _, x := range contents.GetExclusions() {
			detail.Exclusions = append(detail.Exclusions, &sourcev1.DistributionExclusion{Path: x.GetPath(), Category: x.GetCategory(), SafeReason: x.GetSafeReason()})
		}
		detail.UnresolvedObligations = contents.GetUnresolvedObligations()
		detail.RuntimeRequirements = contents.GetRuntimeRequirements()
	} else {
		detail.UnavailableReason = "contents unavailable: " + safeSourceError(e)
	}
	if handoff, e := s.server.sourceDistributions.Handoff(ctx, id); e == nil {
		detail.Handoff = handoffToConnect(mapHandoff(handoff))
	} else if detail.UnavailableReason == "" {
		detail.UnavailableReason = "publication handoff unavailable: " + safeSourceError(e)
	}
	if drift, e := s.server.sourceDistributions.Drift(ctx, id); e == nil {
		detail.Drift = driftToConnect(mapDrift(drift))
	} else if detail.UnavailableReason == "" {
		detail.UnavailableReason = "drift unavailable: " + safeSourceError(e)
	}
	return connect.NewResponse(detail), nil
}

func distributionToConnect(d SourceDistribution) *sourcev1.Distribution {
	return &sourcev1.Distribution{DistributionId: d.DistributionID, Scenario: d.Scenario, SourceDigest: d.SourceDigest, ClosureDigest: d.ClosureDigest, RecipeDigest: d.RecipeDigest, PolicyDigest: d.PolicyDigest, ArtifactId: d.ArtifactID, ArtifactDigest: d.ArtifactDigest, VerificationStatus: d.VerificationStatus, VerificationReceipt: d.VerificationReceipt, DeploymentManagerDecision: d.DeploymentManagerDecision, PublicationStatus: d.PublicationStatus, Destination: d.Destination, DestinationRevision: d.DestinationRevision, ReadbackReceipt: d.ReadbackReceipt, DriftState: d.DriftState, SourceOfTruth: d.SourceOfTruth, SourceTimestamp: d.SourceTimestamp, Freshness: d.Freshness, UpdatedAt: d.UpdatedAt, WorkflowUrl: d.WorkflowURL}
}

func handoffToConnect(h *PublicationHandoff) *sourcev1.PublicationHandoff {
	if h == nil {
		return nil
	}
	return &sourcev1.PublicationHandoff{Status: h.Status, ArtifactId: h.ArtifactID, ArtifactDigest: h.ArtifactDigest, Destination: h.Destination, HumanAction: h.HumanAction, ReadbackOracle: h.ReadbackOracle, Preconditions: h.Preconditions, ApprovalReference: h.ApprovalReference, SourceOfTruth: h.SourceOfTruth, Freshness: h.Freshness}
}

func driftToConnect(d *DistributionDrift) *sourcev1.DistributionDrift {
	if d == nil {
		return nil
	}
	return &sourcev1.DistributionDrift{State: d.State, SourceChanged: d.SourceChanged, DestinationChanged: d.DestinationChanged, CurrentSourceDigest: d.CurrentSourceDigest, RecordedSourceDigest: d.RecordedSourceDigest, CurrentArtifactDigest: d.CurrentArtifactDigest, RecordedArtifactDigest: d.RecordedArtifactDigest, Actions: d.Actions, SourceOfTruth: d.SourceOfTruth, Freshness: d.Freshness}
}
