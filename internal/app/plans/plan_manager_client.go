package plans

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/demand"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/vrooli/internal/tuning"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

const planManagerScenario = "plan-manager"

var planManagerDemandSequence uint64

type PlanManagerClient interface {
	ListPlans(ctx context.Context, workspace WorkspaceScope, includeArchived bool) ([]PlanRecord, error)
	GetPlan(ctx context.Context, workspace WorkspaceScope, ref string) (PlanRecord, error)
	RenderMarkdown(ctx context.Context, workspace WorkspaceScope, ref string) (RenderedPlan, error)
}

type RenderedPlan struct {
	Plan    PlanRecord
	Content string
}

// ConnectPlanManagerClient is the control plane's typed client for Plan
// Manager. Client uses the default transport when nil, preserving the shared
// provenance/identity forwarding behavior used by other platform Connect
// clients.
type ConnectPlanManagerClient struct {
	Client           *http.Client
	BaseURL          string
	Demand           demand.LeaseClient
	DemandConsumerID string
	service          plansconnect.PlansServiceClient
	resolveURL       func(context.Context) (string, error)
}

func NewDefaultPlanManagerClient(ctx context.Context) (PlanManagerClient, error) {
	return newDiscoveredPlanManagerClient(ctx, discovery.ResolveScenarioURLDefault)
}

func newDiscoveredPlanManagerClient(ctx context.Context, resolveURL func(context.Context, string) (string, error)) (PlanManagerClient, error) {
	url, err := resolveURL(ctx, planManagerScenario)
	if err != nil {
		return nil, fmt.Errorf("%w: discover %s: %w", ErrPlanManagerUnavailable, planManagerScenario, err)
	}
	url = strings.TrimRight(url, "/")
	client := &http.Client{Timeout: tuning.ControlPlaneClientTimeout()}
	return ConnectPlanManagerClient{
		Client:           client,
		BaseURL:          url,
		Demand:           demand.Client{},
		DemandConsumerID: "control-plane:plan-manager-client",
		resolveURL: func(ctx context.Context) (string, error) {
			return resolveURL(ctx, planManagerScenario)
		},
	}, nil
}

func (c ConnectPlanManagerClient) plansService(ctx context.Context) (plansconnect.PlansServiceClient, error) {
	if c.resolveURL != nil {
		url, err := c.resolveURL(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: resolve %s: %w", ErrPlanManagerUnavailable, planManagerScenario, err)
		}
		client := c.Client
		if client == nil {
			client = http.DefaultClient
		}
		return plansconnect.NewPlansServiceClient(client, strings.TrimRight(url, "/")), nil
	}
	if c.service != nil {
		return c.service, nil
	}
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	return plansconnect.NewPlansServiceClient(client, c.BaseURL), nil
}

func (c ConnectPlanManagerClient) ListPlans(ctx context.Context, workspace WorkspaceScope, includeArchived bool) ([]PlanRecord, error) {
	release, err := c.acquireDemand(ctx, "list-plans")
	if err != nil {
		return nil, err
	}
	if release != nil {
		defer release()
	}
	service, err := c.plansService(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := service.ListPlans(ctx, connect.NewRequest(&plansv1.ListPlansRequest{
		IncludeArchived: includeArchived,
		Workspace:       workspaceScopeToProto(workspace),
	}))
	if err != nil {
		return nil, classifyPlanManagerConnectError(err)
	}
	out := make([]PlanRecord, 0, len(resp.Msg.GetPlans()))
	for _, p := range resp.Msg.GetPlans() {
		out = append(out, recordFromProto(p))
	}
	return out, nil
}

func (c ConnectPlanManagerClient) GetPlan(ctx context.Context, workspace WorkspaceScope, ref string) (PlanRecord, error) {
	release, err := c.acquireDemand(ctx, "get-plan:"+ref)
	if err != nil {
		return PlanRecord{}, err
	}
	if release != nil {
		defer release()
	}
	service, err := c.plansService(ctx)
	if err != nil {
		return PlanRecord{}, err
	}
	resp, err := service.GetPlan(ctx, connect.NewRequest(&plansv1.GetPlanRequest{
		Id:        ref,
		Workspace: workspaceScopeToProto(workspace),
	}))
	if err != nil {
		return PlanRecord{}, classifyPlanManagerConnectError(err)
	}
	return recordFromProto(resp.Msg.GetPlan()), nil
}

func (c ConnectPlanManagerClient) RenderMarkdown(ctx context.Context, workspace WorkspaceScope, ref string) (RenderedPlan, error) {
	release, err := c.acquireDemand(ctx, "render-markdown:"+ref)
	if err != nil {
		return RenderedPlan{}, err
	}
	if release != nil {
		defer release()
	}
	service, err := c.plansService(ctx)
	if err != nil {
		return RenderedPlan{}, err
	}
	resp, err := service.RenderMarkdown(ctx, connect.NewRequest(&plansv1.RenderMarkdownRequest{
		Id:        ref,
		Workspace: workspaceScopeToProto(workspace),
	}))
	if err != nil {
		return RenderedPlan{}, classifyPlanManagerConnectError(err)
	}
	msg := resp.Msg
	record := recordFromProto(msg.GetPlan())
	if record.ID == "" {
		record = PlanRecord{ID: ref, Slug: ref}
	}
	if record.Path == "" {
		record.Path = msg.GetMirror().GetPath()
	}
	return RenderedPlan{Plan: record, Content: msg.GetMarkdown()}, nil
}

func workspaceScopeToProto(scope WorkspaceScope) *plansv1.WorkspaceScope {
	if strings.TrimSpace(scope.ID) == "" && strings.TrimSpace(scope.Root) == "" {
		return nil
	}
	return &plansv1.WorkspaceScope{
		Id:   strings.TrimSpace(scope.ID),
		Root: strings.TrimSpace(scope.Root),
	}
}

func (c ConnectPlanManagerClient) acquireDemand(ctx context.Context, requestID string) (func(), error) {
	if c.Demand == nil {
		return nil, nil
	}
	consumerID := strings.TrimSpace(c.DemandConsumerID)
	if consumerID == "" {
		consumerID = "control-plane:plan-manager-client"
	}
	requestID = fmt.Sprintf("%s:%d", requestID, atomic.AddUint64(&planManagerDemandSequence, 1))
	_, release, err := demand.AcquireScoped(ctx, c.Demand, demand.AcquireRequest{
		LeaseID:    demand.StableLeaseID(consumerID, planManagerScenario, requestID),
		Scenario:   planManagerScenario,
		ConsumerID: consumerID,
		Kind:       demand.KindDependency,
		RequestID:  requestID,
		Metadata:   "owner=plan-manager-client",
	}, "plan-manager request complete")
	if err != nil {
		return nil, fmt.Errorf("acquire plan-manager demand lease: %w", err)
	}
	return release, nil
}

func classifyPlanManagerConnectError(err error) error {
	if err == nil {
		return nil
	}
	if connect.CodeOf(err) == connect.CodeCanceled {
		return fmt.Errorf("%w: %w", ErrPlanManagerConnectError, err)
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("%w: %w", ErrPlanManagerTimeout, err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("%w: %w", ErrPlanManagerTimeout, err)
	}
	switch connect.CodeOf(err) {
	case connect.CodeNotFound:
		return fmt.Errorf("%w: %w", ErrPlanManagerNotFound, err)
	case connect.CodeInvalidArgument, connect.CodeFailedPrecondition, connect.CodeOutOfRange:
		return fmt.Errorf("%w: %w", ErrPlanManagerInvalid, err)
	case connect.CodeAlreadyExists, connect.CodeAborted:
		return fmt.Errorf("%w: %w", ErrPlanManagerConflict, err)
	case connect.CodeUnavailable:
		return fmt.Errorf("%w: %w", ErrPlanManagerUnavailable, err)
	case connect.CodeDeadlineExceeded:
		return fmt.Errorf("%w: %w", ErrPlanManagerTimeout, err)
	case connect.CodeInternal, connect.CodeUnknown:
		return fmt.Errorf("%w: %w", ErrPlanManagerServer, err)
	default:
		return fmt.Errorf("%w: %w", ErrPlanManagerConnectError, err)
	}
}

func ClassifyPlanManagerConnectError(err error) error {
	return classifyPlanManagerConnectError(err)
}

func recordFromProto(p *sharedv1.Plan) PlanRecord {
	if p == nil {
		return PlanRecord{}
	}
	mirror := p.GetMirror()
	return PlanRecord{
		ID:            p.GetId(),
		Title:         p.GetTitle(),
		Slug:          p.GetSlug(),
		Path:          mirror.GetPath(),
		CreatedAt:     parsePlansTime(p.GetCreatedAt()),
		UpdatedAt:     parsePlansTime(p.GetUpdatedAt()),
		Archived:      p.GetStatus() == sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED,
		ContentHash:   p.GetContentHash(),
		SourcePath:    sourcePathFromProto(p),
		WorkspaceID:   p.GetWorkspaceId(),
		WorkspaceRoot: p.GetWorkspaceRoot(),
	}
}

func sourcePathFromProto(p *sharedv1.Plan) string {
	if p.GetImportProvenance() == nil {
		return ""
	}
	return p.GetImportProvenance().GetSourcePath()
}

func parsePlansTime(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339Nano, value)
	return t
}
