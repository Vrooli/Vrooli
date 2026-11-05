// Package deploymentsvc is the Connect implementation of
// vrooli.scenario_to_cloud.v1.deployments.DeploymentsService. It is a thin
// adapter: identity resolution lives in the identity package, storage in the
// repository, and every failure is the same typed apierrors.Error the REST
// surface writes, attached to the Connect error as a detail.
package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"

	deploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/errors"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/identity"
)

// SchemaVersion is the response schema version clients negotiate on.
const SchemaVersion = "1"

// DefaultPageSize bounds a list response when the caller gives no size.
const DefaultPageSize = 100

// MaxPageSize is the largest page a caller may request.
const MaxPageSize = 500

// Repository is the narrow storage seam the service needs.
type Repository interface {
	identity.Resolver
	GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
	ListDeployments(ctx context.Context, filter domain.ListFilter) ([]*domain.Deployment, error)
}

// Service implements deploymentsv1connect.DeploymentsServiceHandler.
type Service struct {
	repo Repository
}

// New builds the service over a repository.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// Handler returns the Connect mount path and handler.
func (s *Service) Handler(opts ...connect.HandlerOption) (string, http.Handler) {
	return deploymentsv1connect.NewDeploymentsServiceHandler(s, opts...)
}

// ResolveDeployment maps one selector to exactly one DeploymentRef.
func (s *Service) ResolveDeployment(ctx context.Context, req *connect.Request[deploymentsv1.ResolveDeploymentRequest]) (*connect.Response[deploymentsv1.ResolveDeploymentResponse], error) {
	sel := req.Msg.GetSelector()
	ref, err := identity.Resolve(ctx, s.repo, identity.Selector{
		ID:          sel.GetId(),
		ScenarioID:  sel.GetScenarioId(),
		Environment: sel.GetEnvironment(),
		Domain:      sel.GetDomain(),
		Host:        sel.GetHost(),
	})
	if err != nil {
		return nil, ConnectError(err)
	}
	return connect.NewResponse(&deploymentsv1.ResolveDeploymentResponse{SchemaVersion: SchemaVersion, Ref: DeploymentRefProto(ref)}), nil
}

// GetDeployment reads one deployment record by stable id.
func (s *Service) GetDeployment(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentRequest]) (*connect.Response[deploymentsv1.GetDeploymentResponse], error) {
	id := req.Msg.GetId()
	if id == "" {
		return nil, ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "id is required"))
	}
	d, err := s.repo.GetDeployment(ctx, id)
	if err != nil {
		return nil, ConnectError(apierrors.Internal("Failed to get deployment", err))
	}
	if d == nil {
		return nil, ConnectError(apierrors.New(apierrors.CodeDeploymentNotFound, "Deployment not found").WithDetail("id", id))
	}
	msg, err := DeploymentProto(d)
	if err != nil {
		return nil, ConnectError(apierrors.Internal("Failed to encode deployment", err))
	}
	return connect.NewResponse(&deploymentsv1.GetDeploymentResponse{SchemaVersion: SchemaVersion, Deployment: msg}), nil
}

// ListDeployments returns summaries, newest first. The page token is the
// offset of the next record; it is opaque to clients.
func (s *Service) ListDeployments(ctx context.Context, req *connect.Request[deploymentsv1.ListDeploymentsRequest]) (*connect.Response[deploymentsv1.ListDeploymentsResponse], error) {
	filter := domain.ListFilter{}
	if v := req.Msg.GetScenarioId(); v != "" {
		filter.ScenarioID = &v
	}
	if v := req.Msg.GetEnvironment(); v != "" {
		filter.Environment = &v
	}
	if v := req.Msg.GetStatus(); v != "" {
		status := domain.DeploymentStatus(v)
		filter.Status = &status
	}
	pageSize := int(req.Msg.GetPageSize())
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	if token := req.Msg.GetPageToken(); token != "" {
		offset, err := strconv.Atoi(token)
		if err != nil || offset < 0 {
			return nil, ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "page_token is not valid").WithDetail("page_token", token))
		}
		filter.Offset = offset
	}
	// Fetch one extra record to learn whether another page exists.
	filter.Limit = pageSize + 1
	deployments, err := s.repo.ListDeployments(ctx, filter)
	if err != nil {
		return nil, ConnectError(apierrors.Internal("Failed to list deployments", err))
	}
	resp := &deploymentsv1.ListDeploymentsResponse{SchemaVersion: SchemaVersion}
	if len(deployments) > pageSize {
		deployments = deployments[:pageSize]
		resp.NextPageToken = strconv.Itoa(filter.Offset + pageSize)
	}
	for _, d := range deployments {
		resp.Deployments = append(resp.Deployments, DeploymentSummaryProto(d))
	}
	return connect.NewResponse(resp), nil
}

// ConnectError maps a typed error onto a Connect error carrying the same
// stable code both as the message prefix and as an errors.v1.Error detail, so
// generated clients read one code set regardless of transport.
func ConnectError(err error) *connect.Error {
	typed := apierrors.As(err)
	cerr := connect.NewError(connectCode(typed.Status()), errors.New(typed.Error()))
	if detail, detailErr := connect.NewErrorDetail(ErrorProto(typed)); detailErr == nil {
		cerr.AddDetail(detail)
	}
	return cerr
}

func connectCode(status int) connect.Code {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	case http.StatusForbidden:
		return connect.CodePermissionDenied
	case http.StatusNotFound:
		return connect.CodeNotFound
	case http.StatusConflict:
		return connect.CodeAborted
	case http.StatusPreconditionRequired, http.StatusPreconditionFailed, http.StatusFailedDependency:
		return connect.CodeFailedPrecondition
	case http.StatusNotImplemented:
		return connect.CodeUnimplemented
	case http.StatusServiceUnavailable:
		return connect.CodeUnavailable
	default:
		return connect.CodeInternal
	}
}

// ErrorProto projects a typed error onto the wire message.
func ErrorProto(e *apierrors.Error) *errorsv1.Error {
	msg := &errorsv1.Error{Code: e.Code, Message: e.Message, Retryable: e.Retryable}
	if e.NextAction != nil {
		msg.NextAction = &errorsv1.NextAction{Owner: e.NextAction.Owner, Kind: e.NextAction.Kind, Reference: e.NextAction.Reference, Label: e.NextAction.Label}
	}
	if len(e.Details) > 0 {
		if details, err := structpb.NewStruct(jsonRoundTrip(e.Details)); err == nil {
			msg.Details = details
		}
	}
	return msg
}

// jsonRoundTrip coerces arbitrary detail values into the JSON-compatible
// shapes structpb accepts (maps, slices, strings, numbers, bools).
func jsonRoundTrip(in map[string]any) map[string]any {
	raw, err := json.Marshal(in)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

// DeploymentRefProto projects a deployment identity onto the wire message.
func DeploymentRefProto(ref identity.DeploymentRef) *identityv1.DeploymentRef {
	return &identityv1.DeploymentRef{
		Id:          ref.ID,
		ScenarioId:  ref.ScenarioID,
		Environment: ref.Environment,
		Target:      TargetRefProto(ref.Target),
	}
}

// TargetRefProto projects a target binding onto the wire message.
func TargetRefProto(t identity.TargetRef) *identityv1.TargetRef {
	return &identityv1.TargetRef{
		MachineId:            t.MachineID,
		NodeId:               t.NodeID,
		EnrollmentGeneration: t.EnrollmentGeneration,
		Transport:            t.Transport,
		Locator: &identityv1.TargetLocator{
			Host:    t.Locator.Host,
			Port:    int32(t.Locator.Port),
			User:    t.Locator.User,
			Workdir: t.Locator.Workdir,
		},
	}
}

// DeploymentRefFromProto reads a wire identity back into the domain type.
func DeploymentRefFromProto(msg *identityv1.DeploymentRef) identity.DeploymentRef {
	if msg == nil {
		return identity.DeploymentRef{}
	}
	ref := identity.DeploymentRef{ID: msg.GetId(), ScenarioID: msg.GetScenarioId(), Environment: msg.GetEnvironment()}
	if t := msg.GetTarget(); t != nil {
		ref.Target = identity.TargetRef{
			MachineID:            t.GetMachineId(),
			NodeID:               t.GetNodeId(),
			EnrollmentGeneration: t.GetEnrollmentGeneration(),
			Transport:            t.GetTransport(),
		}
		if l := t.GetLocator(); l != nil {
			ref.Target.Locator = identity.TargetLocator{Host: l.GetHost(), Port: int(l.GetPort()), User: l.GetUser(), Workdir: l.GetWorkdir()}
		}
	}
	return ref
}

// DeploymentSummaryProto projects a record onto the list-view message.
func DeploymentSummaryProto(d *domain.Deployment) *deploymentsv1.DeploymentSummary {
	return &deploymentsv1.DeploymentSummary{
		Ref:             DeploymentRefProto(d.Ref()),
		Name:            d.Name,
		Status:          string(d.Status),
		Domain:          manifestDomain(d.Manifest),
		ErrorMessage:    deref(d.ErrorMessage),
		ProgressStep:    deref(d.ProgressStep),
		ProgressPercent: d.ProgressPercent,
		CreatedAt:       timestamppb.New(d.CreatedAt),
		LastDeployedAt:  optionalTimestamp(d.LastDeployedAt),
	}
}

// DeploymentProto projects a record onto the detail message. The manifest is
// carried as a Struct; secrets are never stored in it.
func DeploymentProto(d *domain.Deployment) (*deploymentsv1.Deployment, error) {
	msg := &deploymentsv1.Deployment{
		Ref:             DeploymentRefProto(d.Ref()),
		Name:            d.Name,
		Status:          string(d.Status),
		Domain:          manifestDomain(d.Manifest),
		Fence:           d.Fence,
		BundleSha256:    deref(d.BundleSHA256),
		ErrorMessage:    deref(d.ErrorMessage),
		ErrorStep:       deref(d.ErrorStep),
		ProgressStep:    deref(d.ProgressStep),
		ProgressPercent: d.ProgressPercent,
		CreatedAt:       timestamppb.New(d.CreatedAt),
		UpdatedAt:       timestamppb.New(d.UpdatedAt),
		LastDeployedAt:  optionalTimestamp(d.LastDeployedAt),
		LastInspectedAt: optionalTimestamp(d.LastInspectedAt),
		DesiredState:    string(d.DesiredState.Normalized()),
	}
	if len(d.Manifest) > 0 {
		var raw map[string]any
		if err := json.Unmarshal(d.Manifest, &raw); err != nil {
			return nil, err
		}
		manifest, err := structpb.NewStruct(raw)
		if err != nil {
			return nil, err
		}
		msg.Manifest = manifest
	}
	return msg, nil
}

func manifestDomain(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m domain.CloudManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	return m.Edge.Domain
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func optionalTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
