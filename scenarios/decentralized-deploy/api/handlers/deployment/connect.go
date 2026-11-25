package deployment

import (
	"context"
	"errors"
	"regexp"
	"strings"

	core "decentralized-deploy/internal/deployment"

	"connectrpc.com/connect"
	"decentralized-deploy/internal/module"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	deploymentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/decentralized-deploy/v1/deployment"
	deploymentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/decentralized-deploy/v1/deployment/deployment_v1connect"
)

var deploymentIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

type Records interface {
	Get(context.Context, string) (core.Record, error)
}

func ConnectModule(service *core.Service, records Records, artifactRoot string) module.Module {
	connectPath, connectHandler := deploymentconnect.NewDeploymentServiceHandler(NewConnectHandler(service, records, artifactRoot))
	return module.Module{
		Name: "deployment",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: ConnectEndpoints,
	}
}

type ConnectHandler struct {
	deploymentconnect.UnimplementedDeploymentServiceHandler
	service      *core.Service
	records      Records
	artifactRoot string
}

func NewConnectHandler(service *core.Service, records Records, artifactRoot string) *ConnectHandler {
	return &ConnectHandler{service: service, records: records, artifactRoot: artifactRoot}
}

func (h *ConnectHandler) Deploy(ctx context.Context, req *connect.Request[deploymentv1.DeployRequest]) (*connect.Response[deploymentv1.DeployResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("deployment storage router is unavailable"))
	}
	if req.Msg == nil || !deploymentIDPattern.MatchString(req.Msg.Id) || req.Msg.Name == "" || strings.TrimSpace(req.Msg.ComputeReference) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("deployment id, name, and Compute Manager reference are required"))
	}
	record, err := h.service.DeployWithComputeReference(ctx, req.Msg.Id, core.Artifact{
		Path: h.artifactRoot, Name: "reference", StorageProvider: req.Msg.StorageProvider, EntryPath: "site/index.html",
	}, req.Msg.Name, req.Msg.ComputeReference)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&deploymentv1.DeployResponse{Deployment: toProto(record)}), nil
}

func (h *ConnectHandler) Get(ctx context.Context, req *connect.Request[deploymentv1.GetRequest]) (*connect.Response[deploymentv1.GetResponse], error) {
	if h.records == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("deployment record store is unavailable"))
	}
	if req.Msg == nil || !deploymentIDPattern.MatchString(req.Msg.Id) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid deployment id is required"))
	}
	record, err := h.records.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&deploymentv1.GetResponse{Deployment: toProto(record)}), nil
}

func (h *ConnectHandler) Resolve(ctx context.Context, req *connect.Request[deploymentv1.ResolveRequest]) (*connect.Response[deploymentv1.ResolveResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("deployment lifecycle is unavailable"))
	}
	if req.Msg == nil || !deploymentIDPattern.MatchString(req.Msg.Id) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid deployment id is required"))
	}
	record, body, err := h.service.Resolve(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&deploymentv1.ResolveResponse{Deployment: toProto(record), Content: body}), nil
}

func (h *ConnectHandler) Renew(ctx context.Context, req *connect.Request[deploymentv1.RenewRequest]) (*connect.Response[deploymentv1.RenewResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("deployment lifecycle is unavailable"))
	}
	if req.Msg == nil || !deploymentIDPattern.MatchString(req.Msg.Id) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid deployment id is required"))
	}
	record, err := h.service.Renew(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&deploymentv1.RenewResponse{Deployment: toProto(record)}), nil
}

func (h *ConnectHandler) Teardown(ctx context.Context, req *connect.Request[deploymentv1.TeardownRequest]) (*connect.Response[deploymentv1.TeardownResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("deployment lifecycle is unavailable"))
	}
	if req.Msg == nil || !deploymentIDPattern.MatchString(req.Msg.Id) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid deployment id is required"))
	}
	record, err := h.service.Teardown(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&deploymentv1.TeardownResponse{Deployment: toProto(record)}), nil
}

func toProto(record core.Record) *deploymentv1.DeploymentReceipt {
	result := &deploymentv1.DeploymentReceipt{
		Id: record.ID, StorageProvider: record.Storage.Provider, ContentId: record.Storage.CID,
		ContentUri: record.Storage.URI, State: record.State, ComputeReference: record.ComputeReference,
		Names: make([]*deploymentv1.NamedContent, 0, len(record.Names)),
	}
	for _, name := range record.Names {
		result.Names = append(result.Names, &deploymentv1.NamedContent{Provider: name.Provider, Name: name.Name, Target: name.Target})
	}
	return result
}
