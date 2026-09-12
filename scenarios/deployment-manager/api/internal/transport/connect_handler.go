// Package transport provides the temporary typed Connect boundary for domains
// whose business handlers are already correct but whose request models are
// still being moved into domain-specific protos. The transport is generated;
// the adapter only translates the existing handler response at the edge.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"deployment-manager/dependencies"
	"deployment-manager/deployments"
	"deployment-manager/fitness"
	"deployment-manager/releases"
	"deployment-manager/swaps"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	approvalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/approvals/approvalsv1connect"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/dependencies/dependenciesv1connect"
	deploymentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/deployments/deploymentsv1connect"
	fitnessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/fitness/fitnessv1connect"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/lpbs/lpbsv1connect"
	migrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/migration/migrationv1connect"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	releasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	swapsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/swaps/swapsv1connect"
	telemetryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/telemetry/telemetryv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Route struct {
	Path    string
	Handler http.Handler
}

type Handler struct {
	dependencies         *dependencies.Handler
	fitness              *fitness.Handler
	deployments          *deployments.Handler
	orchestrator         *deployments.Orchestrator
	swaps                *swaps.Handler
	telemetry            http.HandlerFunc
	telemetryUpload      http.HandlerFunc
	migrationReport      http.HandlerFunc
	migrationStatus      http.HandlerFunc
	approvals            *deployments.ApprovalsHandler
	lpbsGet              http.HandlerFunc
	lpbsSave             http.HandlerFunc
	releaseList          http.HandlerFunc
	releaseGet           http.HandlerFunc
	releaseOperation     http.HandlerFunc
	releaseDossier       http.HandlerFunc
	releaseVerify        http.HandlerFunc
	releaseReconcile     http.HandlerFunc
	releaseRecover       http.HandlerFunc
	releaseStart         http.HandlerFunc
	releaseIdentity      releases.IdentityRepository
	authorize            func(context.Context) error
	readAuthorize        func(context.Context) error
	receiptAuthorize     func(context.Context) error
	preparationAuthorize func(context.Context) error
}

// WithAuthorization applies the mutation boundary to the legacy adapter
// while its domain handlers migrate to typed services. Read operations remain
// available to the appropriate authenticated readers; every routed write must
// carry a verified operator identity.
func (h *Handler) WithAuthorization(authorize func(context.Context) error) *Handler {
	h.authorize = authorize
	return h
}

// WithReadAuthorization applies the authenticated reviewer boundary to all
// generated read operations served by this compatibility adapter.
func (h *Handler) WithReadAuthorization(authorize func(context.Context) error) *Handler {
	h.readAuthorize = authorize
	return h
}

// WithClientUpdateReceiptAuthorization sets the service-to-service boundary
// for owner-produced update receipts. It is separate from WithAuthorization,
// which is intentionally restricted to human operator mutations.
func (h *Handler) WithClientUpdateReceiptAuthorization(authorize func(context.Context) error) *Handler {
	h.receiptAuthorize = authorize
	return h
}

// WithReleasePreparationAuthorization sets the service-to-service boundary
// for immutable candidate and destination registration.
func (h *Handler) WithReleasePreparationAuthorization(authorize func(context.Context) error) *Handler {
	h.preparationAuthorize = authorize
	return h
}

// WithReleaseIdentityRepository lets the typed release adapter include the
// canonical objects behind release IDs. Historical rows without an identity
// remain readable; current rows fail closed if their durable identity cannot
// be retrieved.
func (h *Handler) WithReleaseIdentityRepository(repository releases.IdentityRepository) *Handler {
	h.releaseIdentity = repository
	return h
}

// WithReleaseDossierHandler supplies the read-only reviewer dossier to the
// generated release service without widening the legacy constructor.
func (h *Handler) WithReleaseDossierHandler(handler http.HandlerFunc) *Handler {
	h.releaseDossier = handler
	return h
}

// WithReleaseOperationHandler supplies the durable operation standing to the
// typed release service. It is separate from release lookup so the operation
// contract cannot depend on release ID naming conventions.
func (h *Handler) WithReleaseOperationHandler(handler http.HandlerFunc) *Handler {
	h.releaseOperation = handler
	return h
}

// WithReleaseReconcileHandler supplies the owner-observation compatibility
// adapter to the typed release service.
func (h *Handler) WithReleaseReconcileHandler(handler http.HandlerFunc) *Handler {
	h.releaseReconcile = handler
	return h
}

// WithReleaseRecoverHandler supplies the exact owner-recovery adapter to the
// typed release service. The domain handler remains responsible for its
// stronger destructive authorization and receipt checks.
func (h *Handler) WithReleaseRecoverHandler(handler http.HandlerFunc) *Handler {
	h.releaseRecover = handler
	return h
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

func (h *Handler) GetReleaseDossier(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "release_id")
	return h.invoke(ctx, http.MethodGet, "/api/v1/releases/"+url.PathEscape(id)+"/dossier", nil, h.releaseDossier, map[string]string{"release_id": id}, nil)
}

func (h *Handler) ReverifyRelease(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "release_id")
	query := url.Values{"deep": {strconv.FormatBool(boolField(req.Msg, "deep", false))}}
	return h.invoke(ctx, http.MethodPost, "/api/v1/releases/"+url.PathEscape(id)+"/verify", req.Msg, h.releaseVerify, map[string]string{"release_id": id}, query)
}

func (h *Handler) ReconcileRelease(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "release_id")
	query := url.Values{"deep": {strconv.FormatBool(boolField(req.Msg, "deep", false))}}
	return h.invoke(ctx, http.MethodPost, "/api/v1/releases/"+url.PathEscape(id)+"/reconcile", req.Msg, h.releaseReconcile, map[string]string{"release_id": id}, query)
}

func (h *Handler) RecoverRelease(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "release_id")
	return h.invoke(ctx, http.MethodPost, "/api/v1/releases/"+url.PathEscape(id)+"/recover", req.Msg, h.releaseRecover, map[string]string{"release_id": id}, nil)
}

func (h *Handler) StartRelease(ctx context.Context, req *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	id := stringField(req.Msg, "profile_id")
	return h.invoke(ctx, http.MethodPost, "/api/v1/profiles/"+url.PathEscape(id)+"/releases/start", req.Msg, h.releaseStart, map[string]string{"id": id}, nil)
}

func (h *Handler) invoke(ctx context.Context, method, path string, payload *structpb.Value, fn http.HandlerFunc, vars map[string]string, query url.Values) (*connect.Response[structpb.Value], error) {
	if fn == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("operation %s is not configured", path))
	}
	if method == http.MethodGet || method == http.MethodHead {
		if h.readAuthorize == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("read authorization is not configured"))
		}
		if err := h.readAuthorize(ctx); err != nil {
			code := connect.CodeUnauthenticated
			message := "verified deployment-manager reader identity required"
			if principal, ok := identity.PrincipalFromContext(ctx); ok && principal.Verified {
				code = connect.CodePermissionDenied
				message = "deployment-manager read capability required"
			}
			return nil, connect.NewError(code, errors.New(message))
		}
	} else {
		if h.authorize == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("mutation authorization is not configured"))
		}
		if err := h.authorize(ctx); err != nil {
			code := connect.CodeUnauthenticated
			message := "verified deployment operator identity required"
			if principal, ok := identity.PrincipalFromContext(ctx); ok && principal.Verified {
				code = connect.CodePermissionDenied
				message = "deployment-manager mutation capability required"
			}
			return nil, connect.NewError(code, errors.New(message))
		}
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

func boolField(value *structpb.Value, key string, fallback bool) bool {
	if value == nil || value.GetStructValue() == nil {
		return fallback
	}
	field, ok := value.GetStructValue().Fields[key]
	if !ok || field == nil {
		return fallback
	}
	return field.GetBoolValue()
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

func (s releasesService) List(ctx context.Context, req *connect.Request[releasesv1.ListReleasesRequest]) (*connect.Response[releasesv1.ListReleasesResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response, err := s.ListReleases(ctx, typedValueRequest(map[string]interface{}{"profile_id": req.Msg.GetProfileId(), "limit": strconv.FormatUint(uint64(req.Msg.GetLimit()), 10)}))
	if err != nil {
		return nil, err
	}
	values, err := s.releaseEnvelope(ctx, response.Msg, "releases")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&releasesv1.ListReleasesResponse{Releases: values}), nil
}

func (s releasesService) Get(ctx context.Context, req *connect.Request[releasesv1.GetReleaseRequest]) (*connect.Response[releasesv1.GetReleaseResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response, err := s.GetRelease(ctx, typedValueRequest(map[string]interface{}{"release_id": req.Msg.GetReleaseId()}))
	if err != nil {
		return nil, err
	}
	release, err := s.releaseEnvelopeOne(ctx, response.Msg, "release")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&releasesv1.GetReleaseResponse{Release: release}), nil
}

func (s releasesService) GetOperation(ctx context.Context, req *connect.Request[releasesv1.GetReleaseOperationRequest]) (*connect.Response[releasesv1.GetReleaseOperationResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response, err := s.invoke(ctx, http.MethodGet, "/api/v1/release-operations/"+url.PathEscape(req.Msg.GetOperationId()), nil, s.releaseOperation, map[string]string{"operation_id": req.Msg.GetOperationId()}, nil)
	if err != nil {
		return nil, err
	}
	operation, err := releaseOperation(response.Msg.AsInterface())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&releasesv1.GetReleaseOperationResponse{Operation: operation}), nil
}

func (s releasesService) Dossier(ctx context.Context, req *connect.Request[releasesv1.GetReleaseDossierRequest]) (*connect.Response[releasesv1.GetReleaseDossierResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response, err := s.GetReleaseDossier(ctx, typedValueRequest(map[string]interface{}{"release_id": req.Msg.GetReleaseId()}))
	if err != nil {
		return nil, err
	}
	dossier, err := releaseDossier(response.Msg.AsInterface())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&releasesv1.GetReleaseDossierResponse{Dossier: dossier}), nil
}

func (s releasesService) Reverify(ctx context.Context, req *connect.Request[releasesv1.ReverifyReleaseRequest]) (*connect.Response[releasesv1.ReverifyReleaseResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response, err := s.ReverifyRelease(ctx, typedValueRequest(map[string]interface{}{"release_id": req.Msg.GetReleaseId(), "deep": req.Msg.GetDeep()}))
	if err != nil {
		return nil, err
	}
	release, err := s.releaseEnvelopeOne(ctx, response.Msg, "release")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&releasesv1.ReverifyReleaseResponse{Release: release}), nil
}

func (s releasesService) Reconcile(ctx context.Context, req *connect.Request[releasesv1.ReconcileReleaseRequest]) (*connect.Response[releasesv1.ReconcileReleaseResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response, err := s.ReconcileRelease(ctx, typedValueRequest(map[string]interface{}{"release_id": req.Msg.GetReleaseId(), "deep": req.Msg.GetDeep()}))
	if err != nil {
		return nil, err
	}
	release, err := s.releaseEnvelopeOne(ctx, response.Msg, "release")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&releasesv1.ReconcileReleaseResponse{Release: release}), nil
}

func (s releasesService) Recover(ctx context.Context, req *connect.Request[releasesv1.RecoverReleaseRequest]) (*connect.Response[releasesv1.RecoverReleaseResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	artifacts := make(map[string]interface{}, len(req.Msg.GetRepairArtifactIds()))
	for platform, id := range req.Msg.GetRepairArtifactIds() {
		artifacts[platform] = float64(id)
	}
	response, err := s.RecoverRelease(ctx, typedValueRequest(map[string]interface{}{
		"release_id": req.Msg.GetReleaseId(), "review_key": req.Msg.GetReviewKey(), "candidate_id": req.Msg.GetCandidateId(),
		"destination_revision_id": req.Msg.GetDestinationRevisionId(), "action": req.Msg.GetAction(),
		"expected_predecessor_revision": float64(req.Msg.GetExpectedPredecessorRevision()), "data_compatibility": req.Msg.GetDataCompatibility(),
		"repair_artifact_ids": artifacts, "repair_bundle_sha256": req.Msg.GetRepairBundleSha256(), "idempotency_key": req.Msg.GetIdempotencyKey(),
		"confirmation": req.Msg.GetConfirmation(), "dry_run": req.Msg.GetDryRun(),
	}))
	if err != nil {
		return nil, err
	}
	raw, ok := response.Msg.AsInterface().(map[string]interface{})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("release recovery response is not an object"))
	}
	var receipt releases.RecoveryReceipt
	if err := decodeJSONValue(raw["receipt"], &receipt); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode recovery receipt: %w", err))
	}
	return connect.NewResponse(&releasesv1.RecoverReleaseResponse{ReleaseId: stringValue(raw["release_id"]), DryRun: boolValue(raw["dry_run"]), Receipt: receipt.Proto()}), nil
}

func (s releasesService) Start(ctx context.Context, req *connect.Request[releasesv1.StartReleaseRequest]) (*connect.Response[releasesv1.StartReleaseResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	message := req.Msg
	platforms := make([]interface{}, 0, len(message.GetPlatforms()))
	for _, platform := range message.GetPlatforms() {
		platforms = append(platforms, platform)
	}
	fields := map[string]interface{}{
		"profile_id": message.GetProfileId(), "channel": message.GetChannel(), "git_commit_hash": message.GetGitCommitHash(),
		"artifact_digest": message.GetArtifactDigest(), "candidate_id": message.GetCandidateId(),
		"destination_revision_id": message.GetDestinationRevisionId(), "authorization_epoch": float64(message.GetAuthorizationEpoch()),
		"idempotency_key": message.GetIdempotencyKey(), "readiness_review_key": message.GetReadinessReviewKey(),
		"release_version": message.GetReleaseVersion(), "release_notes": message.GetReleaseNotes(), "platforms": platforms,
		"cloud_deployment_name": message.GetCloudDeploymentName(),
		"cloud_bundle_path":     message.GetCloudBundlePath(), "cloud_bundle_sha256": message.GetCloudBundleSha256(),
		"cloud_bundle_size_bytes": float64(message.GetCloudBundleSizeBytes()), "cloud_run_preflight": message.GetCloudRunPreflight(),
	}
	if raw := strings.TrimSpace(message.GetCloudManifestJson()); raw != "" {
		var manifest interface{}
		if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cloud manifest is invalid JSON: %w", err))
		}
		fields["cloud_manifest"] = manifest
	}
	response, err := s.StartRelease(ctx, typedValueRequest(fields))
	if err != nil {
		return nil, err
	}
	raw, ok := response.Msg.AsInterface().(map[string]interface{})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("release start response is not an object"))
	}
	result := &releasesv1.StartReleaseResponse{OperationId: stringValue(raw["operation_id"]), ReleaseId: stringValue(raw["release_id"]), Status: stringValue(raw["status"])}
	if value, exists := raw["release"]; exists {
		release, err := releaseView(value)
		if err != nil {
			return nil, err
		}
		if err := s.enrichReleaseView(ctx, release); err != nil {
			return nil, err
		}
		result.Release = release
	}
	return connect.NewResponse(result), nil
}

func (s releasesService) RegisterCandidate(ctx context.Context, req *connect.Request[releasesv1.RegisterCandidateRequest]) (*connect.Response[releasesv1.RegisterCandidateResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.GetCandidate() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("candidate is required"))
	}
	if s.preparationAuthorize == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("release preparation authorization is not configured"))
	}
	if err := s.preparationAuthorize(ctx); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	repository, ok := s.releaseIdentity.(releases.IdentityRepository)
	if !ok {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("release identity repository is unavailable"))
	}
	candidate, err := releases.CandidateFromProto(req.Msg.GetCandidate())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	record, err := repository.RegisterCandidate(ctx, candidate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	value, err := record.Candidate.Proto()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&releasesv1.RegisterCandidateResponse{Candidate: &releasesv1.CandidateRecord{
		CandidateId: record.ID, Candidate: value, ArtifactManifestDigest: record.ArtifactManifestDigest,
		CreatedAt: timestamppb.New(record.CreatedAt),
	}}), nil
}

func (s releasesService) RegisterDestinationRevision(ctx context.Context, req *connect.Request[releasesv1.RegisterDestinationRevisionRequest]) (*connect.Response[releasesv1.RegisterDestinationRevisionResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.GetRevision() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("destination revision is required"))
	}
	if s.preparationAuthorize == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("release preparation authorization is not configured"))
	}
	if err := s.preparationAuthorize(ctx); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	repository, ok := s.releaseIdentity.(releases.IdentityRepository)
	if !ok {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("release identity repository is unavailable"))
	}
	revision, err := releases.DestinationRevisionFromProto(req.Msg.GetRevision())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	record, err := repository.RegisterDestinationRevision(ctx, revision)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	value, err := record.Revision.Proto()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&releasesv1.RegisterDestinationRevisionResponse{Destination: &releasesv1.DestinationRevisionRecord{
		DestinationRevisionId: record.ID, Revision: value, CreatedAt: timestamppb.New(record.CreatedAt),
	}}), nil
}

func (s releasesService) RecordClientUpdateReceipt(ctx context.Context, req *connect.Request[releasesv1.RecordClientUpdateReceiptRequest]) (*connect.Response[releasesv1.RecordClientUpdateReceiptResponse], error) {
	if req == nil || req.Msg == nil || req.Msg.GetReceipt() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("release id and receipt are required"))
	}
	if s.receiptAuthorize == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("client update receipt authorization is not configured"))
	}
	if err := s.receiptAuthorize(ctx); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	repository, ok := s.releaseIdentity.(releases.ClientUpdateReceiptRepository)
	if !ok {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("client update receipt repository is unavailable"))
	}
	releaseID := strings.TrimSpace(req.Msg.GetReleaseId())
	if releaseID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("release id is required"))
	}
	if err := repository.RecordClientUpdateReceipt(ctx, releaseID, releases.ClientUpdateReceiptFromProto(req.Msg.GetReceipt())); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&releasesv1.RecordClientUpdateReceiptResponse{ReleaseId: releaseID, Accepted: true}), nil
}

func typedValueRequest(fields map[string]interface{}) *connect.Request[structpb.Value] {
	value, err := structpb.NewValue(fields)
	if err != nil {
		return connect.NewRequest(structpb.NewNullValue())
	}
	return connect.NewRequest(value)
}

func (s releasesService) releaseEnvelope(ctx context.Context, value *structpb.Value, key string) ([]*releasesv1.ReleaseView, error) {
	raw, ok := value.AsInterface().(map[string]interface{})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("release response is not an object"))
	}
	items, ok := raw[key].([]interface{})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("release response list is malformed"))
	}
	result := make([]*releasesv1.ReleaseView, 0, len(items))
	for _, item := range items {
		view, err := releaseView(item)
		if err != nil {
			return nil, err
		}
		if err := s.enrichReleaseView(ctx, view); err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

func (s releasesService) releaseEnvelopeOne(ctx context.Context, value *structpb.Value, key string) (*releasesv1.ReleaseView, error) {
	raw, ok := value.AsInterface().(map[string]interface{})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("release response is not an object"))
	}
	item, ok := raw[key].(map[string]interface{})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("release response is missing a release"))
	}
	view, err := releaseView(item)
	if err != nil {
		return nil, err
	}
	if err := s.enrichReleaseView(ctx, view); err != nil {
		return nil, err
	}
	return view, nil
}

func (s releasesService) enrichReleaseView(ctx context.Context, view *releasesv1.ReleaseView) error {
	if s.releaseIdentity == nil || view == nil {
		return nil
	}
	if view.GetCandidateId() != "" {
		record, err := s.releaseIdentity.GetCandidate(ctx, view.GetCandidateId())
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("load candidate %q: %w", view.GetCandidateId(), err))
		}
		candidate, err := record.Candidate.Proto()
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("encode candidate %q: %w", view.GetCandidateId(), err))
		}
		view.Candidate = candidate
	}
	if view.GetDestinationRevisionId() != "" {
		record, err := s.releaseIdentity.GetDestinationRevision(ctx, view.GetDestinationRevisionId())
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("load destination revision %q: %w", view.GetDestinationRevisionId(), err))
		}
		destination, err := record.Revision.Proto()
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("encode destination revision %q: %w", view.GetDestinationRevisionId(), err))
		}
		view.DestinationRevision = destination
	}
	if view.GetReadinessReviewKey() != "" {
		record, err := s.releaseIdentity.GetReview(ctx, view.GetReadinessReviewKey())
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("load review binding %q: %w", view.GetReadinessReviewKey(), err))
		}
		binding, err := record.Binding.Proto()
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("encode review binding %q: %w", view.GetReadinessReviewKey(), err))
		}
		view.ReviewBinding = binding
	}
	if receipts, ok := s.releaseIdentity.(releases.PublicationReceiptRepository); ok && view.GetReleaseId() != "" {
		stored, err := receipts.ListPublicationReceipts(ctx, view.GetReleaseId())
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("load publication receipts for %q: %w", view.GetReleaseId(), err))
		}
		for _, receipt := range stored {
			value := receipt.Proto()
			view.PublicationReceipts = append(view.PublicationReceipts, value)
		}
	}
	if receipts, ok := s.releaseIdentity.(releases.RecoveryReceiptRepository); ok && view.GetReleaseId() != "" {
		stored, err := receipts.ListRecoveryReceipts(ctx, view.GetReleaseId())
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("load recovery receipts for %q: %w", view.GetReleaseId(), err))
		}
		for _, receipt := range stored {
			view.RecoveryReceipts = append(view.RecoveryReceipts, receipt.Proto())
		}
	}
	if receipts, ok := s.releaseIdentity.(releases.ClientUpdateReceiptRepository); ok && view.GetReleaseId() != "" {
		stored, err := receipts.ListClientUpdateReceipts(ctx, view.GetReleaseId())
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("load client update receipts for %q: %w", view.GetReleaseId(), err))
		}
		for _, receipt := range stored {
			view.ClientUpdateReceipts = append(view.ClientUpdateReceipts, receipt.Proto())
		}
	}
	return nil
}

func releaseView(value interface{}) (*releasesv1.ReleaseView, error) {
	raw, ok := value.(map[string]interface{})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("release record is malformed"))
	}
	view := &releasesv1.ReleaseView{
		ReleaseId: stringValue(raw["id"]), ProfileId: stringValue(raw["profile_id"]), DeploymentId: stringValue(raw["deployment_id"]), GitCommitHash: stringValue(raw["git_commit_hash"]),
		ArtifactDigest: stringValue(raw["artifact_digest"]), CandidateId: stringValue(raw["candidate_id"]), DestinationRevisionId: stringValue(raw["destination_revision_id"]),
		AuthorizationEpoch: uint64Value(raw["authorization_epoch"]), ReadinessReviewKey: stringValue(raw["readiness_review_key"]), ReleaseVersion: stringValue(raw["release_version"]),
		Channel: stringValue(raw["channel"]), Status: stringValue(raw["status"]), ReleaseNotes: stringValue(raw["release_notes"]), ReleasedBy: stringValue(raw["released_by"]),
		CreatedAt: timestampValue(raw["created_at"]), PublishedAt: timestampValue(raw["published_at"]), UpdatedAt: timestampValue(raw["updated_at"]),
	}
	if platforms, ok := raw["platforms"].([]interface{}); ok {
		for _, item := range platforms {
			platform, ok := item.(map[string]interface{})
			if !ok {
				return nil, connect.NewError(connect.CodeInternal, errors.New("release platform record is malformed"))
			}
			view.Platforms = append(view.Platforms, &releasesv1.ReleasePlatformView{Platform: stringValue(platform["platform"]), Status: stringValue(platform["status"]), Error: stringValue(platform["error"])})
		}
	}
	if receipts, ok := raw["recovery_receipts"].([]interface{}); ok {
		for _, item := range receipts {
			receipt, ok := item.(map[string]interface{})
			if !ok {
				return nil, connect.NewError(connect.CodeInternal, errors.New("recovery receipt record is malformed"))
			}
			view.RecoveryReceipts = append(view.RecoveryReceipts, &releasesv1.RecoveryReceipt{
				ReleaseId: stringValue(receipt["release_id"]), CandidateId: stringValue(receipt["candidate_id"]),
				DestinationRevisionId: stringValue(receipt["destination_revision_id"]), DeploymentId: stringValue(receipt["deployment_id"]),
				Action: stringValue(receipt["action"]), Outcome: stringValue(receipt["outcome"]), Health: stringValue(receipt["health"]),
				BundleSha256: stringValue(receipt["bundle_sha256"]), ExternalReceipt: stringValue(receipt["external_receipt"]), ObservedAt: timestampValue(receipt["observed_at"]),
				DryRun: boolValue(receipt["dry_run"]),
			})
		}
	}
	if receipts, ok := raw["publication_receipts"].([]interface{}); ok {
		for _, item := range receipts {
			receipt, ok := item.(map[string]interface{})
			if !ok {
				return nil, connect.NewError(connect.CodeInternal, errors.New("publication receipt record is malformed"))
			}
			view.PublicationReceipts = append(view.PublicationReceipts, &releasesv1.PublicationReceipt{
				CandidateId: stringValue(receipt["candidate_id"]), DestinationRevisionId: stringValue(receipt["destination_revision_id"]),
				TargetId: stringValue(receipt["target_id"]), ArtifactDigest: stringValue(receipt["artifact_digest"]),
				DestinationObject: stringValue(receipt["destination_object"]), Producer: stringValue(receipt["producer"]),
				ExternalReceipt: stringValue(receipt["external_receipt"]), Outcome: releaseReceiptOutcomeProto(releases.ReceiptOutcome(stringValue(receipt["outcome"]))),
				ObservedAt: timestampValue(receipt["observed_at"]),
			})
		}
	}
	if receipts, ok := raw["client_update_receipts"].([]interface{}); ok {
		for _, item := range receipts {
			receipt, ok := item.(map[string]interface{})
			if !ok {
				return nil, connect.NewError(connect.CodeInternal, errors.New("client update receipt record is malformed"))
			}
			view.ClientUpdateReceipts = append(view.ClientUpdateReceipts, &releasesv1.ClientUpdateReceipt{
				CandidateId: stringValue(receipt["candidate_id"]), PredecessorRef: stringValue(receipt["predecessor_ref"]),
				SuccessorDigest: stringValue(receipt["successor_digest"]), TargetId: stringValue(receipt["target_id"]),
				VerifiedVersion: stringValue(receipt["verified_version"]), Outcome: releaseReceiptOutcomeProto(releases.ReceiptOutcome(stringValue(receipt["outcome"]))),
				Producer: stringValue(receipt["producer"]), ExternalReceipt: stringValue(receipt["external_receipt"]), ObservedAt: timestampValue(receipt["observed_at"]),
			})
		}
	}
	return view, nil
}

func releaseDossier(value interface{}) (*releasesv1.ReleaseDossier, error) {
	raw, ok := value.(map[string]interface{})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("release dossier is malformed"))
	}
	releaseRaw, ok := raw["release"].(map[string]interface{})
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("release dossier is missing a release"))
	}
	release, err := releaseView(releaseRaw)
	if err != nil {
		return nil, err
	}
	dossier := &releasesv1.ReleaseDossier{
		SchemaVersion: uint32Value(raw["schema_version"]), GeneratedAt: timestampValue(raw["generated_at"]), Release: release,
	}
	if healthRaw, ok := raw["health"].(map[string]interface{}); ok {
		var health releases.ReleaseHealth
		if err := decodeJSONValue(healthRaw, &health); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode release health: %w", err))
		}
		dossier.Health = releaseHealthProto(health)
	} else {
		return nil, connect.NewError(connect.CodeInternal, errors.New("release dossier is missing health"))
	}
	if candidateRaw, ok := raw["candidate"].(map[string]interface{}); ok {
		var record releases.CandidateRecord
		if err := decodeJSONValue(candidateRaw, &record); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode candidate record: %w", err))
		}
		candidate, err := record.Candidate.Proto()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode candidate record: %w", err))
		}
		dossier.Candidate = &releasesv1.CandidateRecord{CandidateId: record.ID, Candidate: candidate, ArtifactManifestDigest: record.ArtifactManifestDigest, CreatedAt: timestampValue(record.CreatedAt.Format(time.RFC3339Nano))}
	}
	if destinationRaw, ok := raw["destination"].(map[string]interface{}); ok {
		var record releases.DestinationRevisionRecord
		if err := decodeJSONValue(destinationRaw, &record); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode destination record: %w", err))
		}
		destination, err := record.Revision.Proto()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode destination record: %w", err))
		}
		dossier.Destination = &releasesv1.DestinationRevisionRecord{DestinationRevisionId: record.ID, Revision: destination, CreatedAt: timestampValue(record.CreatedAt.Format(time.RFC3339Nano))}
	}
	if reviewRaw, ok := raw["review"].(map[string]interface{}); ok {
		var record releases.ReviewRecord
		if err := decodeJSONValue(reviewRaw, &record); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode review record: %w", err))
		}
		binding, err := record.Binding.Proto()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode review record: %w", err))
		}
		dossier.Review = &releasesv1.ReviewRecord{ReviewId: record.ID, Binding: binding, Status: record.Status, ApprovedAt: releaseTimeProto(record.ApprovedAt), RevokedAt: releaseTimeProto(record.RevokedAt), CreatedAt: releaseTimeProto(&record.CreatedAt)}
	}
	if missing, ok := raw["missing_proof"].([]interface{}); ok {
		for _, item := range missing {
			dossier.MissingProof = append(dossier.MissingProof, stringValue(item))
		}
	}
	return dossier, nil
}

func releaseOperation(value interface{}) (*releasesv1.ReleaseOperation, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode release operation: %w", err))
	}
	var operation releases.Operation
	if err := json.Unmarshal(data, &operation); err != nil || operation.ID == "" {
		if err == nil {
			err = errors.New("operation is missing an operation_id")
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode release operation: %w", err))
	}
	return operation.Proto(), nil
}

func decodeJSONValue(value interface{}, target interface{}) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}

func releaseHealthProto(health releases.ReleaseHealth) *releasesv1.ReleaseHealth {
	value := &releasesv1.ReleaseHealth{ReleaseId: health.ReleaseID, Status: health.Status, ObservedAt: timestamppb.New(health.ObservedAt), PublicationVerified: health.PublicationVerified, ClientUpdatesHealthy: health.ClientUpdatesHealthy, RecoveryStanding: health.RecoveryStanding, KnownDurationsMillis: health.KnownDurationsMillis, SupportedControls: health.SupportedControls, UnsupportedControls: health.UnsupportedControls}
	for _, alert := range health.Alerts {
		value.Alerts = append(value.Alerts, &releasesv1.ReleaseAlert{Code: alert.Code, Severity: alert.Severity, Target: alert.Target, Message: alert.Message, NextAction: alert.NextAction})
	}
	return value
}

func releaseReceiptOutcomeProto(outcome releases.ReceiptOutcome) releasesv1.ReceiptOutcome {
	values := map[releases.ReceiptOutcome]releasesv1.ReceiptOutcome{
		releases.ReceiptPrepared:    releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_PREPARED,
		releases.ReceiptStaged:      releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_STAGED,
		releases.ReceiptPublished:   releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_PUBLISHED,
		releases.ReceiptVerified:    releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_VERIFIED,
		releases.ReceiptFailed:      releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_FAILED,
		releases.ReceiptAmbiguous:   releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_AMBIGUOUS,
		releases.ReceiptUnavailable: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_UNAVAILABLE,
	}
	return values[outcome]
}

func releaseTimeProto(value *time.Time) *timestamppb.Timestamp {
	if value == nil || value.IsZero() {
		return nil
	}
	return timestamppb.New(*value)
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func uint64Value(value interface{}) uint64 {
	if number, ok := value.(float64); ok && number >= 0 {
		return uint64(number)
	}
	return 0
}

func uint32Value(value interface{}) uint32 {
	if number, ok := value.(float64); ok && number >= 0 {
		return uint32(number)
	}
	return 0
}

func boolValue(value interface{}) bool {
	boolean, _ := value.(bool)
	return boolean
}

func timestampValue(value interface{}) *timestamppb.Timestamp {
	parsed, err := time.Parse(time.RFC3339Nano, stringValue(value))
	if err != nil || parsed.IsZero() {
		return nil
	}
	return timestamppb.New(parsed)
}
