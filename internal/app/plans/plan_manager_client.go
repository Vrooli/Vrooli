package plans

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/api-core/discovery"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

const (
	planManagerClientParameterA = 500
)

const planManagerScenario = "plan-manager"

type PlanManagerClient interface {
	ListPlans(ctx context.Context, workspace WorkspaceScope, includeArchived bool) ([]PlanRecord, error)
	GetPlan(ctx context.Context, workspace WorkspaceScope, ref string) (PlanRecord, error)
	RenderMarkdown(ctx context.Context, workspace WorkspaceScope, ref string) (RenderedPlan, error)
}

type RenderedPlan struct {
	Plan    PlanRecord
	Content string
}

type HTTPPlanManagerClient struct {
	Client  *http.Client
	BaseURL string
}

func NewDefaultPlanManagerClient(ctx context.Context) (PlanManagerClient, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, planManagerScenario)
	if err != nil {
		return nil, fmt.Errorf("%w: discover %s: %v", ErrPlanManagerUnavailable, planManagerScenario, err)
	}
	return HTTPPlanManagerClient{
		Client:  &http.Client{Timeout: tuning.ControlPlaneClientTimeout()},
		BaseURL: strings.TrimRight(url, "/"),
	}, nil
}

func (c HTTPPlanManagerClient) ListPlans(ctx context.Context, workspace WorkspaceScope, includeArchived bool) ([]PlanRecord, error) {
	var resp plansv1.ListPlansResponse
	if err := c.call(ctx, "/vrooli.plan_manager.v1.plans.PlansService/ListPlans", &plansv1.ListPlansRequest{
		IncludeArchived: includeArchived,
		Workspace:       workspaceScopeToProto(workspace),
	}, &resp); err != nil {
		return nil, err
	}
	out := make([]PlanRecord, 0, len(resp.GetPlans()))
	for _, p := range resp.GetPlans() {
		out = append(out, recordFromProto(p))
	}
	return out, nil
}

func (c HTTPPlanManagerClient) GetPlan(ctx context.Context, workspace WorkspaceScope, ref string) (PlanRecord, error) {
	var resp plansv1.GetPlanResponse
	if err := c.call(ctx, "/vrooli.plan_manager.v1.plans.PlansService/GetPlan", &plansv1.GetPlanRequest{Id: ref, Workspace: workspaceScopeToProto(workspace)}, &resp); err != nil {
		return PlanRecord{}, err
	}
	return recordFromProto(resp.GetPlan()), nil
}

func (c HTTPPlanManagerClient) RenderMarkdown(ctx context.Context, workspace WorkspaceScope, ref string) (RenderedPlan, error) {
	var resp plansv1.RenderMarkdownResponse
	if err := c.call(ctx, "/vrooli.plan_manager.v1.plans.PlansService/RenderMarkdown", &plansv1.RenderMarkdownRequest{Id: ref, Workspace: workspaceScopeToProto(workspace)}, &resp); err != nil {
		return RenderedPlan{}, err
	}
	record := recordFromProto(resp.GetPlan())
	if record.ID == "" {
		record = PlanRecord{ID: ref, Slug: ref}
	}
	if record.Path == "" {
		record.Path = resp.GetMirror().GetPath()
	}
	return RenderedPlan{Plan: record, Content: resp.GetMarkdown()}, nil
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

func (c HTTPPlanManagerClient) call(ctx context.Context, path string, req, resp proto.Message) error {
	body, err := protojson.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return classifyPlanManagerTransportError(err)
	}
	defer httpResp.Body.Close()
	raw, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return classifyPlanManagerStatus(c.BaseURL, httpResp.StatusCode, string(raw))
	}
	return protojson.Unmarshal(raw, resp)
}

func classifyPlanManagerTransportError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("%w: %v", ErrPlanManagerTimeout, err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("%w: %v", ErrPlanManagerTimeout, err)
	}
	return fmt.Errorf("%w: %v", ErrPlanManagerUnavailable, err)
}

func ClassifyPlanManagerTransportError(err error) error {
	return classifyPlanManagerTransportError(err)
}

func classifyPlanManagerStatus(baseURL string, status int, body string) error {
	msg := fmt.Sprintf("plan-manager at %s returned HTTP %d: %s", baseURL, status, body)
	if code := connectErrorCode(body); code != "" {
		switch code {
		case "not_found":
			return fmt.Errorf("%w: %w: %s", ErrPlanManagerNotFound, ErrPlanManagerHTTPStatus, msg)
		case "invalid_argument", "failed_precondition", "out_of_range":
			return fmt.Errorf("%w: %w: %s", ErrPlanManagerInvalid, ErrPlanManagerHTTPStatus, msg)
		case "already_exists", "aborted":
			return fmt.Errorf("%w: %w: %s", ErrPlanManagerConflict, ErrPlanManagerHTTPStatus, msg)
		case "unavailable":
			return fmt.Errorf("%w: %w: %s", ErrPlanManagerUnavailable, ErrPlanManagerHTTPStatus, msg)
		case "deadline_exceeded":
			return fmt.Errorf("%w: %w: %s", ErrPlanManagerTimeout, ErrPlanManagerHTTPStatus, msg)
		}
	}
	switch {
	case status == http.StatusNotFound:
		return fmt.Errorf("%w: %w: %s", ErrPlanManagerNotFound, ErrPlanManagerHTTPStatus, msg)
	case status == http.StatusBadRequest || status == http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: %w: %s", ErrPlanManagerInvalid, ErrPlanManagerHTTPStatus, msg)
	case status == http.StatusConflict:
		return fmt.Errorf("%w: %w: %s", ErrPlanManagerConflict, ErrPlanManagerHTTPStatus, msg)
	case status >= planManagerClientParameterA:
		return fmt.Errorf("%w: %w: %s", ErrPlanManagerServer, ErrPlanManagerHTTPStatus, msg)
	default:
		return fmt.Errorf("%w: %s", ErrPlanManagerHTTPStatus, msg)
	}
}

func ClassifyPlanManagerStatus(baseURL string, status int, body string) error {
	return classifyPlanManagerStatus(baseURL, status, body)
}

func connectErrorCode(body string) string {
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Code)
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
