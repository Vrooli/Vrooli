package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// DiskPressure is the typed observation published by system-monitor.
type DiskPressure struct {
	Observed       bool
	ObservedAt     time.Time
	Band           string
	MountPath      string
	UsedPercent    float64
	UsedBytes      int64
	AvailableBytes int64
	TotalBytes     int64
	LastError      string
}

// DiskPressureReader is the ownership boundary between observation and
// remediation. Autoheal consumes the band; it does not reclassify usage.
type DiskPressureReader interface {
	ReadDiskPressure(ctx context.Context) (DiskPressure, error)
}

type systemMonitorDiskPressureReader struct {
	client  *http.Client
	baseURL string
}

// NewSystemMonitorDiskPressureReader creates the production reader. The URL
// may be supplied for tests or explicit deployments; otherwise discovery
// resolves the running system-monitor scenario.
func NewSystemMonitorDiskPressureReader(client *http.Client, baseURL string) DiskPressureReader {
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	return &systemMonitorDiskPressureReader{client: client, baseURL: strings.TrimRight(baseURL, "/")}
}

func (r *systemMonitorDiskPressureReader) ReadDiskPressure(ctx context.Context) (DiskPressure, error) {
	baseURL, err := r.resolveURL(ctx)
	if err != nil {
		return DiskPressure{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/disk-pressure", nil)
	if err != nil {
		return DiskPressure{}, fmt.Errorf("create system-monitor disk-pressure request: %w", err)
	}
	response, err := r.client.Do(request)
	if err != nil {
		return DiskPressure{}, fmt.Errorf("read system-monitor disk pressure: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return DiskPressure{}, fmt.Errorf("system-monitor disk pressure returned status %d", response.StatusCode)
	}

	var payload struct {
		Observed       bool      `json:"observed"`
		ObservedAt     time.Time `json:"observed_at"`
		Band           string    `json:"band"`
		MountPath      string    `json:"mount_path"`
		UsedPercent    float64   `json:"used_percent"`
		UsedBytes      int64     `json:"used_bytes"`
		AvailableBytes int64     `json:"available_bytes"`
		TotalBytes     int64     `json:"total_bytes"`
		LastError      string    `json:"last_error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return DiskPressure{}, fmt.Errorf("decode system-monitor disk pressure: %w", err)
	}
	return DiskPressure{
		Observed: payload.Observed, ObservedAt: payload.ObservedAt,
		Band: strings.ToLower(strings.TrimSpace(payload.Band)), MountPath: payload.MountPath,
		UsedPercent: payload.UsedPercent, UsedBytes: payload.UsedBytes,
		AvailableBytes: payload.AvailableBytes, TotalBytes: payload.TotalBytes,
		LastError: payload.LastError,
	}, nil
}

func (r *systemMonitorDiskPressureReader) resolveURL(ctx context.Context) (string, error) {
	if r.baseURL != "" {
		return r.baseURL, nil
	}
	url, err := discovery.ResolveScenarioURLDefault(ctx, "system-monitor")
	if err != nil {
		return "", fmt.Errorf("resolve system-monitor: %w", err)
	}
	return strings.TrimRight(url, "/"), nil
}
