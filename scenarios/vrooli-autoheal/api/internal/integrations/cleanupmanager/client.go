// Package cleanupmanager reports disk pressure to the storage-manager
// scenario.
//
// This is vrooli-autoheal's own path to remediation, deliberately independent
// of system-monitor's. Two safeguards that both route through a single
// mediator share a failure mode, and the 2026-07-31 incident was exactly a
// case of every path to action being dead at once. The cost of independence is
// that both safeguards can report the same event; storage-manager collapses
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
// storage-manager rejects anything it does not recognise rather than
// defaulting, so they must match exactly.
type Band string

const (
	BandWarning  Band = "PRESSURE_BAND_WARNING"
	BandHigh     Band = "PRESSURE_BAND_HIGH"
	BandCritical Band = "PRESSURE_BAND_CRITICAL"
)

// Report is an outbound pressure signal.
type Report struct {
	SourceScenario       string      `json:"sourceScenario"`
	Partition            string      `json:"partition"`
	UsedPercent          float64     `json:"usedPercent"`
	Band                 Band        `json:"band"`
	AvailableBytes       int64       `json:"availableBytes"`
	FillRateBytesPerHour int64       `json:"fillRateBytesPerHour,omitempty"`
	HotWriters           []HotWriter `json:"hotWriters,omitempty"`
	Trigger              string      `json:"trigger,omitempty"`
}

// HotWriter identifies a governed root whose growth rate exceeded policy.
type HotWriter struct {
	Root          string `json:"root"`
	CurrentBytes  int64  `json:"currentBytes"`
	BytesPerHour  int64  `json:"bytesPerHour"`
	WindowSeconds int64  `json:"windowSeconds"`
}

// Outcome is what storage-manager did about the report.
type Outcome struct {
	Action                 string   `json:"action"`
	PlanID                 string   `json:"planId"`
	EstimatedBytes         int64    `json:"estimatedBytes,string"`
	ReclaimedBytes         int64    `json:"reclaimedBytes,string"`
	ProvidersApplied       []string `json:"providersApplied"`
	ProvidersWithheld      []string `json:"providersWithheld"`
	Reason                 string   `json:"reason"`
	AutonomousApplyEnabled bool     `json:"autonomousApplyEnabled"`
	BugReference           string   `json:"bugReference"`
	RunID                  string   `json:"runId"`
}

// Reporter is the seam the disk check heals through. Tests substitute a fake
// so the heal action can be exercised without a live storage-manager.
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
