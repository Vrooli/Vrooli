// Package capabilities mounts the marketing-capability catalog Connect surface.
package capabilities

import (
	"context"
	"errors"
	"time"

	internalcapabilities "content-desk/internal/capabilities"
	"content-desk/internal/module"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/capabilities/capabilities_v1connect"
)

type handler struct {
	repository   internalcapabilities.Repository
	evidence     internalcapabilities.EvidenceResolver
	distribution internalcapabilities.DistributionResolver
}

var _ capabilitiesconnect.CapabilitiesServiceHandler = handler{}

func (h handler) ListCapabilities(ctx context.Context, request *connect.Request[capabilitiesv1.ListCapabilitiesRequest]) (*connect.Response[capabilitiesv1.ListCapabilitiesResponse], error) {
	capabilities, err := h.repository.List(ctx, request.Msg.Medium)
	if err != nil {
		if errors.Is(err, internalcapabilities.ErrInvalidMedium) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &capabilitiesv1.ListCapabilitiesResponse{}
	for _, capability := range capabilities {
		response.Capabilities = append(response.Capabilities, h.capabilityMessage(ctx, capability))
	}
	return connect.NewResponse(response), nil
}

func (h handler) GetCapability(ctx context.Context, request *connect.Request[capabilitiesv1.GetCapabilityRequest]) (*connect.Response[capabilitiesv1.GetCapabilityResponse], error) {
	if request.Msg.Id == "" && request.Msg.Alias == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalcapabilities.ErrReferenceRequired)
	}
	capability, err := h.repository.Get(ctx, request.Msg.Id, request.Msg.Alias)
	if err != nil {
		if errors.Is(err, internalcapabilities.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&capabilitiesv1.GetCapabilityResponse{Capability: h.capabilityMessage(ctx, capability)}), nil
}

func (h handler) UpsertCapability(ctx context.Context, request *connect.Request[capabilitiesv1.UpsertCapabilityRequest]) (*connect.Response[capabilitiesv1.UpsertCapabilityResponse], error) {
	if request.Msg.Capability == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("capability is required"))
	}
	capability, err := h.repository.Upsert(ctx, capabilityRecord(request.Msg.Capability))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&capabilitiesv1.UpsertCapabilityResponse{Capability: h.capabilityMessage(ctx, capability)}), nil
}

func (h handler) RecordQualification(ctx context.Context, request *connect.Request[capabilitiesv1.RecordQualificationRequest]) (*connect.Response[capabilitiesv1.RecordQualificationResponse], error) {
	if request.Msg.Qualification == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("qualification is required"))
	}
	qualification, err := h.repository.RecordQualification(ctx, qualificationRecord(request.Msg.Qualification))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&capabilitiesv1.RecordQualificationResponse{Qualification: qualificationMessage(qualification)}), nil
}

func (h handler) LinkCapability(ctx context.Context, request *connect.Request[capabilitiesv1.LinkCapabilityRequest]) (*connect.Response[capabilitiesv1.LinkCapabilityResponse], error) {
	if request.Msg.Link == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("link is required"))
	}
	link, err := h.repository.Link(ctx, internalcapabilities.CapabilityLink{
		ID:           request.Msg.Link.Id,
		CapabilityID: request.Msg.Link.CapabilityId,
		Relation:     request.Msg.Link.Relation,
		TargetID:     request.Msg.Link.TargetId,
	})
	if err != nil {
		if errors.Is(err, internalcapabilities.ErrInvalidRelation) || errors.Is(err, internalcapabilities.ErrInvalidTarget) || errors.Is(err, internalcapabilities.ErrReferenceRequired) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&capabilitiesv1.LinkCapabilityResponse{Link: linkMessage(link)}), nil
}

func (h handler) capabilityMessage(ctx context.Context, capability internalcapabilities.Capability) *capabilitiesv1.Capability {
	qualification := capability.LatestQualification
	if qualification == nil {
		unknown := internalcapabilities.UnknownQualification(capability.ID)
		qualification = &unknown
	}
	quality := h.outputQuality(ctx, capability)
	readiness := internalcapabilities.RehydrateOperationalReadiness(capability.OperationalReadiness, capability, time.Now().UTC())
	connectivity := internalcapabilities.RehydrateDistributionConnectivity(ctx, capability.DistributionConnectivity, capability, h.distribution)
	limitations := make([]string, 0, len(quality.Limitations)+len(readiness.Limitations)+len(connectivity.Limitations))
	limitations = append(limitations, quality.Limitations...)
	limitations = append(limitations, readiness.Limitations...)
	limitations = append(limitations, connectivity.Limitations...)
	message := &capabilitiesv1.Capability{
		Id:                             capability.ID,
		Name:                           capability.Name,
		Medium:                         capability.Medium,
		Aliases:                        capability.Aliases,
		Channels:                       capability.Channels,
		AudienceApplicability:          capability.AudienceApplicability,
		DeliveryApplicability:          capability.DeliveryApplicability,
		ProducingOperation:             capability.ProducingOperation,
		Prerequisites:                  capability.Prerequisites,
		Priority:                       capability.Priority,
		PriorityReason:                 capability.PriorityReason,
		PriorityScope:                  capability.PriorityScope,
		DefinitionStatus:               capability.DefinitionStatus,
		ImplementationStatus:           capability.ImplementationStatus,
		OperationalReadiness:           readiness.Readiness,
		OperationalReadinessSource:     readiness.Source,
		OutputQuality:                  quality.Quality,
		OutputQualitySource:            quality.Source,
		ReadinessLimitations:           limitations,
		DistributionConnectivity:       connectivity.Connectivity,
		DistributionConnectivitySource: connectivity.Source,
		Owner:                          capability.Owner,
		SourceRefs:                     capability.SourceRefs,
		LatestQualification:            qualificationMessage(*qualification),
		NextAction:                     capability.NextAction,
	}
	if !capability.CreatedAt.IsZero() {
		message.CreatedAt = capability.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if !capability.UpdatedAt.IsZero() {
		message.UpdatedAt = capability.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	return message
}

// outputQuality projects output_quality from evidence-linked owner artifacts.
// A link-read failure degrades to the stored value with a named limitation
// instead of failing the whole catalog response.
func (h handler) outputQuality(ctx context.Context, capability internalcapabilities.Capability) internalcapabilities.OutputQualityProjection {
	links, err := h.repository.LinksForCapability(ctx, capability.ID)
	if err != nil {
		return internalcapabilities.OutputQualityProjection{
			Quality:     capability.OutputQuality,
			Source:      internalcapabilities.Unknown,
			Limitations: []string{"evidence links could not be read; output_quality is the stored inventory value"},
		}
	}
	return internalcapabilities.RehydrateOutputQuality(ctx, capability.OutputQuality, links, h.evidence)
}

func qualificationMessage(qualification internalcapabilities.CapabilityQualification) *capabilitiesv1.CapabilityQualification {
	message := &capabilitiesv1.CapabilityQualification{
		Id:                qualification.ID,
		CapabilityId:      qualification.CapabilityID,
		LatestArtifactId:  qualification.LatestArtifactID,
		LatestRunId:       qualification.LatestRunID,
		Environment:       qualification.Environment,
		ValidatedAt:       qualification.ValidatedAt,
		ObservedAt:        qualification.ObservedAt,
		FreshnessBasis:    qualification.FreshnessBasis,
		CandidateIdentity: qualification.CandidateIdentity,
		MaxAgeSeconds:     qualification.MaxAgeSeconds,
		Limitation:        qualification.Limitation,
		NextAction:        qualification.NextAction,
	}
	if !qualification.CreatedAt.IsZero() {
		message.CreatedAt = qualification.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return message
}

func linkMessage(link internalcapabilities.CapabilityLink) *capabilitiesv1.CapabilityLink {
	message := &capabilitiesv1.CapabilityLink{
		Id:           link.ID,
		CapabilityId: link.CapabilityID,
		Relation:     link.Relation,
		TargetId:     link.TargetID,
	}
	if !link.CreatedAt.IsZero() {
		message.CreatedAt = link.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return message
}

func capabilityRecord(message *capabilitiesv1.Capability) internalcapabilities.Capability {
	return internalcapabilities.Capability{
		ID:                       message.Id,
		Name:                     message.Name,
		Medium:                   message.Medium,
		Aliases:                  message.Aliases,
		Channels:                 message.Channels,
		AudienceApplicability:    message.AudienceApplicability,
		DeliveryApplicability:    message.DeliveryApplicability,
		ProducingOperation:       message.ProducingOperation,
		Prerequisites:            message.Prerequisites,
		Priority:                 message.Priority,
		PriorityReason:           message.PriorityReason,
		PriorityScope:            message.PriorityScope,
		DefinitionStatus:         message.DefinitionStatus,
		ImplementationStatus:     message.ImplementationStatus,
		OperationalReadiness:     message.OperationalReadiness,
		OutputQuality:            message.OutputQuality,
		DistributionConnectivity: message.DistributionConnectivity,
		Owner:                    message.Owner,
		SourceRefs:               message.SourceRefs,
		NextAction:               message.NextAction,
	}
}

func qualificationRecord(message *capabilitiesv1.CapabilityQualification) internalcapabilities.CapabilityQualification {
	return internalcapabilities.CapabilityQualification{
		ID:                message.Id,
		CapabilityID:      message.CapabilityId,
		LatestArtifactID:  message.LatestArtifactId,
		LatestRunID:       message.LatestRunId,
		Environment:       message.Environment,
		ValidatedAt:       message.ValidatedAt,
		ObservedAt:        message.ObservedAt,
		FreshnessBasis:    message.FreshnessBasis,
		CandidateIdentity: message.CandidateIdentity,
		MaxAgeSeconds:     message.MaxAgeSeconds,
		Limitation:        message.Limitation,
		NextAction:        message.NextAction,
	}
}

// Module returns the capability catalog module for server.New.
func Module(db *database.RoutedDB) module.Module {
	path, h := capabilitiesconnect.NewCapabilitiesServiceHandler(handler{
		repository:   internalcapabilities.NewRepository(db),
		evidence:     newArtifactOutputResolver(db),
		distribution: newChannelManagerDistributionResolver(),
	})
	return module.Module{Name: "capabilities", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

// Schema returns the capability domain schema for EnsureSchemas.
func Schema() string { return internalcapabilities.Schema() }

// Endpoints describes every capability RPC for the endpoint manifest.
var Endpoints = []module.EndpointDescriptor{
	{ID: "capabilities_list", Path: capabilitiesconnect.CapabilitiesServiceListCapabilitiesProcedure, Method: "POST", Summary: "List marketing capabilities", Category: "capabilities"},
	{ID: "capabilities_get", Path: capabilitiesconnect.CapabilitiesServiceGetCapabilityProcedure, Method: "POST", Summary: "Get a capability by id or alias", Category: "capabilities"},
	{ID: "capabilities_upsert", Path: capabilitiesconnect.CapabilitiesServiceUpsertCapabilityProcedure, Method: "POST", Summary: "Create or update a capability", Category: "capabilities"},
	{ID: "capabilities_record_qualification", Path: capabilitiesconnect.CapabilitiesServiceRecordQualificationProcedure, Method: "POST", Summary: "Record observed capability qualification", Category: "capabilities"},
	{ID: "capabilities_link", Path: capabilitiesconnect.CapabilitiesServiceLinkCapabilityProcedure, Method: "POST", Summary: "Link a capability to an owner record", Category: "capabilities"},
}
