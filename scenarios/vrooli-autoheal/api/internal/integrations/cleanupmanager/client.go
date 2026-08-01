// Package cleanupmanager reports disk pressure to the cleanup-manager
// scenario.
//
// This is vrooli-autoheal's own path to remediation, deliberately independent
// of system-monitor's. Two safeguards that both route through a single
// mediator share a failure mode, and the 2026-07-31 incident was exactly a
// case of every path to action being dead at once. The cost of independence is
// that both safeguards can report the same event; cleanup-manager collapses
// duplicate reports into one execution, so that cost is paid there.
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

const reportPressureProcedure = "/vrooli.cleanup_manager.v1.cleanup.CleanupService/ReportPressure"

// Band names the escalation band. These are the proto enum names;
// cleanup-manager rejects anything it does not recognise rather than
// defaulting, so they must match exactly.
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

// Outcome is what cleanup-manager did about the report.
type Outcome struct {
	Action                 string   `json:"action"`
	PlanID                 string   `json:"planId"`
	EstimatedBytes         int64    `json:"estimatedBytes,string"`
	ReclaimedBytes         int64    `json:"reclaimedBytes,string"`
	ProvidersApplied       []string `json:"providersApplied"`
	ProvidersWithheld      []string `json:"providersWithheld"`
	Reason                 string   `json:"reason"`
	AutonomousApplyEnabled bool     `json:"autonomousApplyEnabled"`
}

// Reporter is the seam the disk check heals through. Tests substitute a fake
// so the heal action can be exercised without a live cleanup-manager.
type Reporter interface {
	ReportPressure(ctx context.Context, report Report) (Outcome, error)
}

// Config controls Client behaviour.
type Config struct {
	BaseURL string
	Timeout time.Duration
}

// Client is the production Reporter.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a Client. The timeout is generous because a critical report
// runs estimate, preview, and apply before responding.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// ReportPressure sends a pressure signal.
func (c *Client) ReportPressure(ctx context.Context, report Report) (Outcome, error) {
	if c == nil {
		return Outcome{}, fmt.Errorf("cleanup-manager client not configured")
	}

	base, err := c.resolveURL(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("resolve cleanup-manager url: %w", err)
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
		return Outcome{}, fmt.Errorf("cleanup-manager unreachable: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Outcome{}, fmt.Errorf("cleanup-manager returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var outcome Outcome
	if err := json.Unmarshal(raw, &outcome); err != nil {
		return Outcome{}, fmt.Errorf("cleanup-manager payload malformed: %w", err)
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
	url, err := discovery.ResolveScenarioURLDefault(ctx, "cleanup-manager")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(url, "/"), nil
}
