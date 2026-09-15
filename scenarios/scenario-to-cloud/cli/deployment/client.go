package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"connectrpc.com/connect"
	deploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/evidence"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/evidence/evidencev1connect"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health/healthv1connect"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/identity"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/plans/plansv1connect"

	"scenario-to-cloud/cli/internal/selector"
	"scenario-to-cloud/cli/internal/transport"
	"scenario-to-cloud/cli/operation"
)

// Client is the deployment command group's access to the API: the generated
// Deployments, Plans, Health and Evidence services, the operations client
// for durable waits, and the REST client for the surfaces that have no
// Connect service yet (create, delete, stop, history, recovery points,
// cloud recovery). It carries no policy: every decision is the server's.
type Client struct {
	tr          transport.Transport
	Deployments deploymentsv1connect.DeploymentsServiceClient
	Plans       plansv1connect.PlansServiceClient
	Health      healthv1connect.HealthServiceClient
	Evidence    evidencev1connect.EvidenceServiceClient
	Operations  *operation.Client
}

// NewClient builds the client over the shared transport.
func NewClient(tr transport.Transport) *Client {
	return &Client{
		tr:          tr,
		Deployments: deploymentsv1connect.NewDeploymentsServiceClient(tr.HTTP, tr.BaseURL),
		Plans:       plansv1connect.NewPlansServiceClient(tr.HTTP, tr.BaseURL),
		Health:      healthv1connect.NewHealthServiceClient(tr.HTTP, tr.BaseURL),
		Evidence:    evidencev1connect.NewEvidenceServiceClient(tr.HTTP, tr.BaseURL),
		Operations:  operation.NewClient(tr),
	}
}

// Resolve maps a selector to exactly one deployment reference.
func (c *Client) Resolve(ctx context.Context, sel selector.Selector) (*identityv1.DeploymentRef, error) {
	return selector.Resolve(ctx, c.Deployments, sel)
}

// Get reads one deployment record.
func (c *Client) Get(ctx context.Context, id string) (*deploymentsv1.GetDeploymentResponse, error) {
	resp, err := c.Deployments.GetDeployment(ctx, connect.NewRequest(&deploymentsv1.GetDeploymentRequest{Id: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// List returns deployment summaries.
func (c *Client) List(ctx context.Context, req *deploymentsv1.ListDeploymentsRequest) (*deploymentsv1.ListDeploymentsResponse, error) {
	resp, err := c.Deployments.ListDeployments(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// CompileOptions are the compile-time inputs: the scope and whether the
// release bundle must be rebuilt before the plan pins its digest.
type CompileOptions struct {
	Scope            string
	ForceBundleBuild bool
}

// ApplyOptions are the admission inputs.
type ApplyOptions struct {
	Scope        string
	RequestKey   string
	RunPreflight bool
}

// CompilePlan previews the executable plan. The server ensures the release
// bundle exists (or rebuilds it with ForceBundleBuild) so the digest is real.
func (c *Client) CompilePlan(ctx context.Context, id string, opts CompileOptions) (*plansv1.CompilePlanResponse, error) {
	resp, err := c.Plans.CompilePlan(ctx, connect.NewRequest(&plansv1.CompilePlanRequest{DeploymentId: id, Scope: opts.Scope, ForceBundleBuild: opts.ForceBundleBuild}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// ApplyPlan admits the reviewed digest as a durable operation.
func (c *Client) ApplyPlan(ctx context.Context, id, digest string, opts ApplyOptions) (*plansv1.ApplyPlanResponse, error) {
	resp, err := c.Plans.ApplyPlan(ctx, connect.NewRequest(&plansv1.ApplyPlanRequest{DeploymentId: id, PlanDigest: digest, RequestKey: opts.RequestKey, Scope: opts.Scope, RunPreflight: opts.RunPreflight}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// HealthObservation returns the typed health observation.
func (c *Client) HealthObservation(ctx context.Context, id string) (*healthv1.GetHealthObservationResponse, error) {
	resp, err := c.Health.GetHealthObservation(ctx, connect.NewRequest(&healthv1.GetHealthObservationRequest{DeploymentId: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// RequestPublication opens a governed publication for the deployment.
func (c *Client) RequestPublication(ctx context.Context, req *evidencev1.PublicationRequest) (*evidencev1.Publication, error) {
	resp, err := c.Evidence.RequestPublication(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// ApplyPublication activates an approved publication by its reviewed digest.
func (c *Client) ApplyPublication(ctx context.Context, req *evidencev1.ApplyPublicationRequest) (*evidencev1.Publication, error) {
	resp, err := c.Evidence.ApplyPublication(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// GetPublication reads the publication standing.
func (c *Client) GetPublication(ctx context.Context, id, requestKey string) (*evidencev1.Publication, error) {
	resp, err := c.Evidence.GetPublication(ctx, connect.NewRequest(&evidencev1.GetPublicationRequest{DeploymentId: id, RequestKey: requestKey}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// --- REST surfaces (no generated service yet) ---

func (c *Client) rest(method, path string, query url.Values, body any, out any) ([]byte, error) {
	raw, err := c.tr.API.Request(method, path, query, body)
	if err != nil {
		return nil, err
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return raw, fmt.Errorf("decode %s %s: %w", method, path, err)
		}
	}
	return raw, nil
}

// Create creates or updates a deployment record from a manifest.
func (c *Client) Create(req CreateRequest) ([]byte, CreateResponse, error) {
	var resp CreateResponse
	raw, err := c.rest("POST", "/api/v1/deployments", nil, req, &resp)
	return raw, resp, err
}

// Delete removes a deployment record.
func (c *Client) Delete(id string, opts DeleteOptions) ([]byte, DeleteResponse, error) {
	query := url.Values{}
	if opts.Stop {
		query.Set("stop", "true")
	}
	if opts.Cleanup {
		query.Set("cleanup", "true")
	}
	var resp DeleteResponse
	raw, err := c.rest("DELETE", "/api/v1/deployments/"+url.PathEscape(id), query, nil, &resp)
	return raw, resp, err
}

// Stop stops the workload synchronously (no plan scope exists for stop).
func (c *Client) Stop(id string) ([]byte, StopResponse, error) {
	var resp StopResponse
	raw, err := c.rest("POST", "/api/v1/deployments/"+url.PathEscape(id)+"/stop", nil, nil, &resp)
	return raw, resp, err
}

// History returns the deployment history events.
func (c *Client) History(id string) ([]byte, HistoryResponse, error) {
	var resp HistoryResponse
	raw, err := c.rest("GET", "/api/v1/deployments/"+url.PathEscape(id)+"/history", nil, nil, &resp)
	return raw, resp, err
}

// ListRecoveryPoints lists the deployment's recovery points.
func (c *Client) ListRecoveryPoints(id string) ([]byte, RecoveryPointsResponse, error) {
	var resp RecoveryPointsResponse
	raw, err := c.rest("GET", "/api/v1/deployments/"+url.PathEscape(id)+"/recovery-points", nil, nil, &resp)
	return raw, resp, err
}

// CaptureRecoveryPoint captures a consistent recovery point.
func (c *Client) CaptureRecoveryPoint(id string, req RecoveryPointCaptureRequest) ([]byte, RecoveryPointResponse, error) {
	var resp RecoveryPointResponse
	raw, err := c.rest("POST", "/api/v1/deployments/"+url.PathEscape(id)+"/recovery-points", nil, req, &resp)
	return raw, resp, err
}

// VerifyRecoveryPoint verifies a recovery point (open=true also proves the
// recovery key resolves).
func (c *Client) VerifyRecoveryPoint(id, rp string, open bool) ([]byte, RecoveryPointVerifyResponse, error) {
	query := url.Values{}
	if open {
		query.Set("open", "true")
	}
	var resp RecoveryPointVerifyResponse
	raw, err := c.rest("GET", "/api/v1/deployments/"+url.PathEscape(id)+"/recovery-points/"+url.PathEscape(rp)+"/verify", query, nil, &resp)
	return raw, resp, err
}

// RestoreRecoveryPoint restores a recovery point onto a target.
func (c *Client) RestoreRecoveryPoint(id, rp string, req RecoveryPointRestoreRequest) ([]byte, RecoveryPointRestoreResponse, error) {
	var resp RecoveryPointRestoreResponse
	raw, err := c.rest("POST", "/api/v1/deployments/"+url.PathEscape(id)+"/recovery-points/"+url.PathEscape(rp)+"/restore", nil, req, &resp)
	return raw, resp, err
}

// Recovery runs the governed cloud recovery (rollback / forward repair):
// dry_run previews and issues the preview_ref an execution must present.
func (c *Client) Recovery(id string, req RecoveryRequest) ([]byte, RecoveryResponse, error) {
	var resp RecoveryResponse
	raw, err := c.rest("POST", "/api/v1/deployments/"+url.PathEscape(id)+"/recovery", nil, req, &resp)
	return raw, resp, err
}
