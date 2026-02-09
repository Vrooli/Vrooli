package inspect

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Client provides API access for inspection operations.
type Client struct {
	api *cliutil.APIClient
}

// NewClient creates a new inspect client.
func NewClient(api *cliutil.APIClient) *Client {
	return &Client{api: api}
}

// Plan generates an inspection plan.
func (c *Client) Plan(manifest map[string]interface{}, opts Options) ([]byte, PlanResponse, error) {
	req := map[string]interface{}{
		"manifest": manifest,
		"options":  opts,
	}
	body, err := c.api.Request("POST", "/api/v1/vps/inspect/plan", nil, req)
	if err != nil {
		return nil, PlanResponse{}, err
	}
	var resp PlanResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, PlanResponse{}, err
	}
	return body, resp, nil
}

// Apply executes the inspection.
func (c *Client) Apply(manifest map[string]interface{}, opts Options) ([]byte, ApplyResponse, error) {
	req := map[string]interface{}{
		"manifest": manifest,
		"options":  opts,
	}
	body, err := c.api.Request("POST", "/api/v1/vps/inspect/apply", nil, req)
	if err != nil {
		return nil, ApplyResponse{}, err
	}
	var resp ApplyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, ApplyResponse{}, err
	}
	return body, resp, nil
}

// LiveState retrieves the live state of a deployment.
func (c *Client) LiveState(id string) ([]byte, LiveStateResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/live-state", id), nil)
	if err != nil {
		return nil, LiveStateResponse{}, err
	}

	// First try legacy flat shape for backward compatibility.
	var resp LiveStateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, LiveStateResponse{}, err
	}
	if hasLegacyLiveStatePayload(resp) {
		return body, resp, nil
	}

	// Then support the current envelope shape: {"result": {...}, "timestamp": "..."}.
	var envelope liveStateEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, LiveStateResponse{}, err
	}
	adapted, ok := adaptLiveStateEnvelope(id, envelope)
	if !ok {
		return body, resp, nil
	}
	return body, adapted, nil
}

type liveStateEnvelope struct {
	Result    liveStateV2 `json:"result"`
	Timestamp string      `json:"timestamp"`
}

type liveStateV2 struct {
	OK        bool            `json:"ok"`
	Timestamp string          `json:"timestamp"`
	Processes *processStateV2 `json:"processes,omitempty"`
	System    *SystemState    `json:"system,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type processStateV2 struct {
	Scenarios []scenarioProcessV2 `json:"scenarios"`
	Resources []resourceProcessV2 `json:"resources"`
}

type scenarioProcessV2 struct {
	ID        string             `json:"id"`
	Status    string             `json:"status"`
	PID       int                `json:"pid"`
	Resources processResourcesV2 `json:"resources"`
}

type processResourcesV2 struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemoryMB   int     `json:"memory_mb"`
}

type resourceProcessV2 struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	PID    int    `json:"pid"`
}

func hasLegacyLiveStatePayload(resp LiveStateResponse) bool {
	return strings.TrimSpace(resp.DeploymentID) != "" ||
		resp.State.Running ||
		resp.State.Healthy ||
		len(resp.Processes) > 0 ||
		len(resp.Resources) > 0
}

func adaptLiveStateEnvelope(deploymentID string, env liveStateEnvelope) (LiveStateResponse, bool) {
	if !env.Result.OK && env.Result.System == nil && env.Result.Processes == nil && strings.TrimSpace(env.Result.Error) == "" {
		return LiveStateResponse{}, false
	}

	resp := LiveStateResponse{
		DeploymentID: deploymentID,
		Timestamp:    strings.TrimSpace(env.Timestamp),
	}
	if resp.Timestamp == "" {
		resp.Timestamp = strings.TrimSpace(env.Result.Timestamp)
	}

	resp.State.Healthy = env.Result.OK && strings.TrimSpace(env.Result.Error) == ""
	resp.State.ErrorMessage = strings.TrimSpace(env.Result.Error)

	if env.Result.Processes != nil {
		for _, sc := range env.Result.Processes.Scenarios {
			p := ProcessInfo{
				PID:        sc.PID,
				Name:       sc.ID,
				Status:     sc.Status,
				CPUPercent: fmt.Sprintf("%.1f%%", sc.Resources.CPUPercent),
			}
			if sc.Resources.MemoryMB > 0 {
				p.MemoryMB = fmt.Sprintf("%dMB", sc.Resources.MemoryMB)
			}
			resp.Processes = append(resp.Processes, p)
			if strings.EqualFold(sc.Status, "running") {
				resp.State.Running = true
			}
		}
		for _, rs := range env.Result.Processes.Resources {
			resp.Resources = append(resp.Resources, ResourceStatus{
				Name:    rs.ID,
				Type:    "resource",
				Status:  rs.Status,
				Healthy: strings.EqualFold(rs.Status, "running"),
			})
		}
	}

	if env.Result.System != nil {
		sys := env.Result.System
		resp.State.CPUPercent = fmt.Sprintf("%.1f%%", sys.CPU.UsagePercent)
		resp.State.MemoryPercent = fmt.Sprintf("%.1f%%", sys.Memory.UsagePercent)
		resp.State.DiskUsedGB = fmt.Sprintf("%d/%d GB", sys.Disk.UsedGB, sys.Disk.TotalGB)
		if sys.UptimeSeconds > 0 {
			resp.State.Uptime = fmt.Sprintf("%ds", sys.UptimeSeconds)
		}
		resp.State.SSHFingerprint = sys.SSH.PublicKeyFingerprint
	}

	return resp, true
}

// Drift retrieves drift detection results for a deployment.
func (c *Client) Drift(id string) ([]byte, DriftResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/drift", id), nil)
	if err != nil {
		return nil, DriftResponse{}, err
	}
	var resp DriftResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, DriftResponse{}, err
	}
	return body, resp, nil
}

// MetricsDebug retrieves raw metrics command output and parsed system metrics.
func (c *Client) MetricsDebug(id string) ([]byte, MetricsDebugResponse, error) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/metrics-debug", id), nil)
	if err != nil {
		return nil, MetricsDebugResponse{}, err
	}
	var resp MetricsDebugResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, MetricsDebugResponse{}, err
	}
	return body, resp, nil
}

// Logs retrieves aggregated logs for a deployment.
func (c *Client) Logs(id string, opts LogsOptions) ([]byte, LogsResponse, error) {
	query := url.Values{}
	if opts.Source != "" {
		query.Set("source", opts.Source)
	}
	if opts.Level != "" {
		query.Set("level", opts.Level)
	}
	if opts.Search != "" {
		query.Set("search", opts.Search)
	}
	if opts.Tail > 0 {
		query.Set("tail", strconv.Itoa(opts.Tail))
	}
	if opts.Since != "" {
		query.Set("since", opts.Since)
	}

	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/logs", id), query)
	if err != nil {
		return nil, LogsResponse{}, err
	}
	var resp LogsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, LogsResponse{}, err
	}
	return body, resp, nil
}

// Files lists files or reads file content from a deployment.
func (c *Client) Files(id string, opts FilesOptions) ([]byte, FilesResponse, error) {
	query := url.Values{}
	if opts.Path != "" {
		query.Set("path", opts.Path)
	}
	if opts.Content {
		query.Set("content", "true")
	}

	body, err := c.api.Get(fmt.Sprintf("/api/v1/deployments/%s/files", id), query)
	if err != nil {
		return nil, FilesResponse{}, err
	}
	var resp FilesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, FilesResponse{}, err
	}
	return body, resp, nil
}
