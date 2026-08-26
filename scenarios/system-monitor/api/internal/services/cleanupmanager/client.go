// Package cleanupmanager provides a thin Connect client for reporting disk
// pressure to the storage-manager scenario.
//
// Contract: reporting pressure never takes the monitor down. Timeouts, refused
// connections, and non-2xx responses are returned as errors for logging, but
// the threshold loop treats a failed report as a logged warning rather than a
// reason to stop evaluating. A monitor that dies because its remediation peer
// is offline is worse than a monitor that keeps observing.
package cleanupmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// reportPressureProcedure is the Connect procedure path. Connect accepts plain
// JSON POSTs to the procedure URL, so this client needs no generated stub and
// no dependency on the storage-manager module.
const reportPressureProcedure = "/vrooli.cleanup_manager.v1.cleanup.CleanupService/ReportPressure"

// Band names the escalation band being reported. These strings are the proto
// enum names; an unrecognised value is rejected by storage-manager rather than
// defaulted, so they must match exactly.
type Band string

const (
	BandWarning  Band = "PRESSURE_BAND_WARNING"
	BandHigh     Band = "PRESSURE_BAND_HIGH"
	BandCritical Band = "PRESSURE_BAND_CRITICAL"
)

// Report is an outbound pressure signal.
type Report struct {
	SourceScenario string  `json:"sourceScenario"`
	Partition      string  `json:"partition"`
	UsedPercent    float64 `json:"usedPercent"`
	Band           Band    `json:"band"`
	AvailableBytes int64   `json:"availableBytes"`
}

// Outcome is what storage-manager did about the report.
type Outcome struct {
	Band                   string   `json:"band"`
	Action                 string   `json:"action"`
	PlanID                 string   `json:"planId"`
	EstimatedBytes         int64    `json:"estimatedBytes,string"`
	ReclaimedBytes         int64    `json:"reclaimedBytes,string"`
	ProvidersApplied       []string `json:"providersApplied"`
	ProvidersWithheld      []string `json:"providersWithheld"`
	Reason                 string   `json:"reason"`
	BugReference           string   `json:"bugReference"`
	AutonomousApplyEnabled bool     `json:"autonomousApplyEnabled"`
}

// Config controls Client behaviour.
type Config struct {
	BaseURL string        // optional; falls back to env / discovery
	Timeout time.Duration // default 10s
}

// Client reports disk pressure to storage-manager.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a Client.
//
// The default timeout is generous because a critical-band report runs an
// estimate, a preview, and an apply on the far side before responding.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// ReportPressure sends a pressure signal and returns what storage-manager did.
func (c *Client) ReportPressure(ctx context.Context, report Report) (Outcome, error) {
	if c == nil {
		return Outcome{}, fmt.Errorf("storage-manager client not configured")
	}

	base, err := c.resolveURL(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("resolve storage-manager url: %w", err)
	}

	body, err := json.Marshal(report)
	if err != nil {
		return Outcome{}, fmt.Errorf("encode pressure report: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+reportPressureProcedure, bytes.NewReader(body))
	if err != nil {
		return Outcome{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Outcome{}, fmt.Errorf("storage-manager unreachable: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Outcome{}, fmt.Errorf("storage-manager returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var outcome Outcome
	if err := json.Unmarshal(raw, &outcome); err != nil {
		return Outcome{}, fmt.Errorf("storage-manager payload malformed: %w", err)
	}
	return outcome, nil
}

func (c *Client) resolveURL(ctx context.Context) (string, error) {
	if c.baseURL != "" {
		return strings.TrimRight(c.baseURL, "/"), nil
	}
	if env := os.Getenv("VROOLI_CLEANUP_MANAGER_API_URL"); env != "" {
		return strings.TrimRight(env, "/"), nil
	}
	url, err := discovery.ResolveScenarioURLDefault(ctx, "storage-manager")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(url, "/"), nil
}
