// Package transport provides the temporary typed Connect boundary for domains
// whose business handlers are already correct but whose request models are
// still being moved into domain-specific protos. The transport is generated;
// the adapter only translates the existing handler response at the edge.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"deployment-manager/dependencies"
	"deployment-manager/deployments"
	"deployment-manager/fitness"
	"deployment-manager/swaps"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	approvalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/approvals/approvalsv1connect"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/dependencies/dependenciesv1connect"
	deploymentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/deployments/deploymentsv1connect"
	fitnessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/fitness/fitnessv1connect"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/lpbs/lpbsv1connect"
	migrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/migration/migrationv1connect"
	releasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	swapsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/swaps/swapsv1connect"
	telemetryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/telemetry/telemetryv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type Route struct {
	Path    string
	Handler http.Handler
}

type Handler struct {
	dependencies    *dependencies.Handler
	fitness         *fitness.Handler
	deployments     *deployments.Handler
	orchestrator    *deployments.Orchestrator
	swaps           *swaps.Handler
	telemetry       http.HandlerFunc
	telemetryUpload http.HandlerFunc
	migrationReport http.HandlerFunc
	migrationStatus http.HandlerFunc
	approvals       *deployments.ApprovalsHandler
	lpbsGet         http.HandlerFunc
	lpbsSave        http.HandlerFunc
	releaseList     http.HandlerFunc
	releaseGet      http.HandlerFunc
	releaseVerify   http.HandlerFunc
	releaseStart    http.HandlerFunc
}

// NewHandler keeps the existing domain handlers as the source of business
// behavior while making their client boundary generated Connect-RPC.
func NewHandler(deps *dependencies.Handler, fit *fitness.Handler, deploy *deployments.Handler, orchestrator *deployments.Orchestrator, swapsHandler *swaps.Handler, telemetryHandler http.HandlerFunc, telemetryUploadHandler http.HandlerFunc, migrationReport http.HandlerFunc, migrationStatus http.HandlerFunc, approvalsHandler *deployments.ApprovalsHandler, lpbsGet http.HandlerFunc, lpbsSave http.HandlerFunc, releaseList http.HandlerFunc, releaseGet http.HandlerFunc, releaseVerify http.HandlerFunc, releaseStart http.HandlerFunc) *Handler {
	return &Handler{dependencies: deps, fitness: fit, deployments: deploy, orchestrator: orchestrator, swaps: swapsHandler, telemetry: telemetryHandler, telemetryUpload: telemetryUploadHandler, migrationReport: migrationReport, migrationStatus: migrationStatus, approvals: approvalsHandler, lpbsGet: lpbsGet, lpbsSave: lpbsSave, releaseList: releaseList, releaseGet: releaseGet, releaseVerify: releaseVerify, releaseStart: releaseStart}
}

func (h *Handler) Analyze(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	scenario := stringField(req.Msg, "scenario")
	return h.invoke(ctx, http.MethodGet, "/api/v1/dependencies/analyze/"+url.PathEscape(scenario), nil, h.dependencies.AnalyzeDependencies, nil, nil)
}

func (h *Handler) Score(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return h.invoke(ctx, http.MethodPost, "/api/v1/fitness/score", req.Msg, h.fitness.ScoreFitness, nil, nil)
}

func (h *Handler) Deploy(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	return h.invoke(ctx, http.MethodPost, "/api/v1/deploy/"+url.PathEscape(id), req.Msg, h.deployments.Deploy, map[string]string{"profile_id": id}, nil)
}

func (h *Handler) DeployDesktop(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return h.invoke(ctx, http.MethodPost, "/api/v1/deploy-desktop", req.Msg, h.orchestrator.DeployDesktop, nil, nil)
}

func (h *Handler) Status(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "deployment_id")
	return h.invoke(ctx, http.MethodGet, "/api/v1/deployments/"+url.PathEscape(id), nil, h.deployments.Status, map[string]string{"deployment_id": id}, nil)
}

func (h *Handler) ListSwaps(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	scenario := stringField(req.Msg, "scenario")
	return h.invoke(ctx, http.MethodGet, "/api/v1/swaps/list/"+url.PathEscape(scenario), nil, h.swaps.List, map[string]string{"scenario": scenario}, nil)
}

func (h *Handler) AnalyzeSwaps(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	from, to := stringField(req.Msg, "from"), stringField(req.Msg, "to")
	return h.invoke(ctx, http.MethodGet, "/api/v1/swaps/analyze/"+url.PathEscape(from)+"/"+url.PathEscape(to), nil, h.swaps.Analyze, map[string]string{"from": from, "to": to}, nil)
}

func (h *Handler) CascadeSwaps(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	from, to := stringField(req.Msg, "from"), stringField(req.Msg, "to")
	return h.invoke(ctx, http.MethodGet, "/api/v1/swaps/cascade/"+url.PathEscape(from)+"/"+url.PathEscape(to), nil, h.swaps.Cascade, map[string]string{"from": from, "to": to}, nil)
}

func (h *Handler) ApplySwaps(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return h.invoke(ctx, http.MethodPost, "/api/v1/swaps/apply", req.Msg, h.swaps.Apply, nil, nil)
}

func (h *Handler) ApplySwapsToProfile(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	return h.invoke(ctx, http.MethodPost, "/api/v1/profiles/"+url.PathEscape(id)+"/swaps", req.Msg, h.swaps.ApplyToProfile, map[string]string{"id": id}, nil)
}

func (h *Handler) ListTelemetry(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return h.invoke(ctx, http.MethodGet, "/api/v1/telemetry", nil, h.telemetry, nil, nil)
}

func (h *Handler) UploadTelemetry(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return h.invoke(ctx, http.MethodPost, "/api/v1/telemetry/upload", req.Msg, h.telemetryUpload, nil, nil)
}

func (h *Handler) ReportMigration(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return h.invoke(ctx, http.MethodPost, "/api/v1/migration-tasks", req.Msg, h.migrationReport, nil, nil)
}

func (h *Handler) StatusMigration(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	query := url.Values{"name": {stringField(req.Msg, "name")}, "kind": {stringFieldDefault(req.Msg, "kind", "fix")}}
	return h.invoke(ctx, http.MethodGet, "/api/v1/migration-tasks/status?"+query.Encode(), nil, h.migrationStatus, nil, query)
}

func (h *Handler) ListApprovals(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	query := url.Values{"commit": {stringField(req.Msg, "git_commit_hash")}}
	return h.invoke(ctx, http.MethodGet, "/api/v1/profiles/"+url.PathEscape(id)+"/approvals", nil, h.approvals.ListByProfile, map[string]string{"id": id}, query)
}

func (h *Handler) GetApproval(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "id")
	return h.invoke(ctx, http.MethodGet, "/api/v1/approvals/"+url.PathEscape(id), nil, h.approvals.Get, map[string]string{"id": id}, nil)
}

func (h *Handler) CreateApproval(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	return h.invoke(ctx, http.MethodPost, "/api/v1/profiles/"+url.PathEscape(id)+"/approvals", req.Msg, h.approvals.Create, map[string]string{"id": id}, nil)
}

func (h *Handler) DecideApproval(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "id")
	return h.invoke(ctx, http.MethodPost, "/api/v1/approvals/"+url.PathEscape(id)+"/decide", req.Msg, h.approvals.Decide, map[string]string{"id": id}, nil)
}

func (h *Handler) CheckReleaseGate(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	query := url.Values{"commit": {stringField(req.Msg, "git_commit_hash")}}
	return h.invoke(ctx, http.MethodGet, "/api/v1/profiles/"+url.PathEscape(id)+"/release-gate", nil, h.approvals.CheckReleaseGate, map[string]string{"id": id}, query)
}

func (h *Handler) SetRequiredPlatforms(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	return h.invoke(ctx, http.MethodPut, "/api/v1/profiles/"+url.PathEscape(id)+"/required-platforms", req.Msg, h.approvals.SetRequiredPlatforms, map[string]string{"id": id}, nil)
}

func (h *Handler) GetRequiredPlatforms(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	return h.invoke(ctx, http.MethodGet, "/api/v1/profiles/"+url.PathEscape(id)+"/required-platforms", nil, h.approvals.GetRequiredPlatforms, map[string]string{"id": id}, nil)
}

func (h *Handler) GetLPBSConfig(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	return h.invoke(ctx, http.MethodGet, "/api/v1/profiles/"+url.PathEscape(id)+"/lpbs-config", nil, h.lpbsGet, map[string]string{"id": id}, nil)
}

func (h *Handler) SaveLPBSConfig(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	return h.invoke(ctx, http.MethodPut, "/api/v1/profiles/"+url.PathEscape(id)+"/lpbs-config", req.Msg, h.lpbsSave, map[string]string{"id": id}, nil)
}

func (h *Handler) ListReleases(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	query := url.Values{"limit": {stringFieldDefault(req.Msg, "limit", "10")}}
	return h.invoke(ctx, http.MethodGet, "/api/v1/profiles/"+url.PathEscape(id)+"/releases", nil, h.releaseList, map[string]string{"id": id}, query)
}

func (h *Handler) GetRelease(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "release_id")
	return h.invoke(ctx, http.MethodGet, "/api/v1/releases/"+url.PathEscape(id), nil, h.releaseGet, map[string]string{"release_id": id}, nil)
}

func (h *Handler) ReverifyRelease(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "release_id")
	query := url.Values{"deep": {stringFieldDefault(req.Msg, "deep", "false")}}
	return h.invoke(ctx, http.MethodPost, "/api/v1/releases/"+url.PathEscape(id)+"/verify", req.Msg, h.releaseVerify, map[string]string{"release_id": id}, query)
}

func (h *Handler) StartRelease(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	return h.invoke(ctx, http.MethodPost, "/api/v1/profiles/"+url.PathEscape(id)+"/releases/start", req.Msg, h.releaseStart, map[string]string{"id": id}, nil)
}

func (h *Handler) invoke(ctx context.Context, method, path string, payload *structpb.Value, fn http.HandlerFunc, vars map[string]string, query url.Values) (*connect.Response[structpb.Value], error) {
	if fn == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("operation %s is not configured", path))
	}
	var body []byte
	if payload != nil {
		var err error
		body, err = protojson.Marshal(payload)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	requestPath := path
	if query != nil && strings.Contains(path, "?") == false {
		requestPath += "?" + query.Encode()
	}
	req := httptest.NewRequestWithContext(ctx, method, requestPath, bytes.NewReader(body))
	if vars != nil {
		req = mux.SetURLVars(req, vars)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	fn(response, req)
	if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
		return nil, connect.NewError(codeForStatus(response.Code), fmt.Errorf("%s: %s", path, strings.TrimSpace(response.Body.String())))
	}
	var raw interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &raw); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode %s response: %w", path, err))
	}
	value, err := structpb.NewValue(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode %s response: %w", path, err))
	}
	return connect.NewResponse(value), nil
}

func stringField(value *structpb.Value, key string) string { return stringFieldDefault(value, key, "") }
func stringFieldDefault(value *structpb.Value, key, fallback string) string {
	if value == nil || value.GetStructValue() == nil {
		return fallback
	}
	if field, ok := value.GetStructValue().Fields[key]; ok && field.GetStringValue() != "" {
		return field.GetStringValue()
	}
	return fallback
}

func codeForStatus(status int) connect.Code {
	switch status {
	case http.StatusBadRequest:
		return connect.CodeInvalidArgument
	case http.StatusNotFound:
		return connect.CodeNotFound
	case http.StatusConflict:
		return connect.CodeAlreadyExists
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	case http.StatusForbidden:
		return connect.CodePermissionDenied
	default:
		return connect.CodeInternal
	}
}

func Routes(h *Handler) []Route {
	dependenciesPath, dependenciesHandler := dependenciesconnect.NewDependenciesServiceHandler(h)
	fitnessPath, fitnessHandler := fitnessconnect.NewFitnessServiceHandler(h)
	deploymentsPath, deploymentsHandler := deploymentsconnect.NewDeploymentsServiceHandler(h)
	swapsPath, swapsHandler := swapsconnect.NewSwapsServiceHandler(swapsService{h})
	telemetryPath, telemetryHandler := telemetryconnect.NewTelemetryServiceHandler(telemetryService{h})
	migrationPath, migrationHandler := migrationconnect.NewMigrationServiceHandler(migrationService{h})
	approvalsPath, approvalsHandler := approvalsconnect.NewApprovalsServiceHandler(approvalsService{h})
	lpbsPath, lpbsHandler := lpbsconnect.NewLPBSServiceHandler(lpbsService{h})
	releasesPath, releasesHandler := releasesconnect.NewReleasesServiceHandler(releasesService{h})
	return []Route{
		{Path: dependenciesPath, Handler: dependenciesHandler},
		{Path: fitnessPath, Handler: fitnessHandler},
		{Path: deploymentsPath, Handler: deploymentsHandler},
		{Path: swapsPath, Handler: swapsHandler},
		{Path: telemetryPath, Handler: telemetryHandler},
		{Path: migrationPath, Handler: migrationHandler},
		{Path: approvalsPath, Handler: approvalsHandler},
		{Path: lpbsPath, Handler: lpbsHandler},
		{Path: releasesPath, Handler: releasesHandler},
	}
}

type swapsService struct{ *Handler }

func (s swapsService) List(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.ListSwaps(ctx, req)
}

func (s swapsService) Analyze(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.AnalyzeSwaps(ctx, req)
}

func (s swapsService) Cascade(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.CascadeSwaps(ctx, req)
}

func (s swapsService) Apply(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.ApplySwaps(ctx, req)
}

func (s swapsService) ApplyToProfile(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.ApplySwapsToProfile(ctx, req)
}

type telemetryService struct{ *Handler }

func (s telemetryService) List(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.ListTelemetry(ctx, req)
}

func (s telemetryService) Upload(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.UploadTelemetry(ctx, req)
}

type migrationService struct{ *Handler }

func (s migrationService) Report(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.ReportMigration(ctx, req)
}

func (s migrationService) Status(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.StatusMigration(ctx, req)
}

type approvalsService struct{ *Handler }

func (s approvalsService) List(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.ListApprovals(ctx, req)
}

func (s approvalsService) Get(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.GetApproval(ctx, req)
}

func (s approvalsService) Create(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.CreateApproval(ctx, req)
}

func (s approvalsService) Decide(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.DecideApproval(ctx, req)
}

func (s approvalsService) CheckReleaseGate(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.Handler.CheckReleaseGate(ctx, req)
}

func (s approvalsService) SetRequiredPlatforms(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.Handler.SetRequiredPlatforms(ctx, req)
}

func (s approvalsService) GetRequiredPlatforms(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.Handler.GetRequiredPlatforms(ctx, req)
}

type lpbsService struct{ *Handler }

func (s lpbsService) GetConfig(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.GetLPBSConfig(ctx, req)
}

func (s lpbsService) SaveConfig(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.SaveLPBSConfig(ctx, req)
}

type releasesService struct{ *Handler }

func (s releasesService) List(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.ListReleases(ctx, req)
}

func (s releasesService) Get(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.GetRelease(ctx, req)
}

func (s releasesService) Reverify(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.ReverifyRelease(ctx, req)
}

func (s releasesService) Start(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return s.StartRelease(ctx, req)
}
