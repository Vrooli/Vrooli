package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// The containment verbs talk to the operator-diagnostic REST routes served by
// handlers/strays (scan, adopt, reclaim and the pin/floor report).

type strayFinding struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	Kind        string   `json:"kind"`
	Library     string   `json:"library"`
	Owner       string   `json:"owner"`
	CatalogIDs  []string `json:"catalog_ids"`
	Bytes       int64    `json:"bytes"`
	Files       int      `json:"files"`
	Action      string   `json:"action"`
	Destination string   `json:"destination"`
	Reason      string   `json:"reason"`
}

type strayReport struct {
	Root    string `json:"managed_root"`
	Scanned []struct {
		Path   string `json:"path"`
		Exists bool   `json:"exists"`
		Source string `json:"source"`
	} `json:"scanned"`
	Findings []strayFinding `json:"findings"`
	Totals   struct {
		Stray      int64 `json:"stray_bytes"`
		Adoptable  int64 `json:"adoptable_bytes"`
		Reclaim    int64 `json:"reclaimable_bytes"`
		ReportOnly int64 `json:"report_only_bytes"`
		Cache      int64 `json:"managed_cache_bytes"`
	} `json:"totals"`
}

func (h *handlers) restDo(method, path string, body any, out any) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, strings.TrimRight(baseURL, "/")+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &apiErr) == nil {
			if apiErr.Error.Message != "" {
				return fmt.Errorf("%s", apiErr.Error.Message)
			}
			if apiErr.Message != "" {
				return fmt.Errorf("%s", apiErr.Message)
			}
		}
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return json.Unmarshal(data, out)
}

func (h *handlers) strays(ctx cliapp.RunContext) error {
	var report strayReport
	if err := h.restDo(http.MethodGet, "/api/v1/models/strays", nil, &report); err != nil {
		return fmt.Errorf("scan for stray model bytes: %w", err)
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(report)
	}
	return ctx.RenderList(renderStrayReport(report))
}

func renderStrayReport(report strayReport) cliapp.ListReport {
	rows := make([]string, 0, len(report.Findings))
	for _, f := range report.Findings {
		owner := f.Owner
		if len(f.CatalogIDs) > 0 {
			owner += " " + strings.Join(f.CatalogIDs, ",")
		}
		row := fmt.Sprintf("%s  %s  %s  [%s] %s: %s", f.ID, humanBytes(f.Bytes), f.Path, strings.ToUpper(f.Action), owner, f.Reason)
		if f.Destination != "" && f.Action == "adopt" {
			row += " -> " + f.Destination
		}
		rows = append(rows, row)
	}
	var scanned []string
	for _, s := range report.Scanned {
		state := "absent"
		if s.Exists {
			state = "present"
		}
		scanned = append(scanned, fmt.Sprintf("%s (%s, %s)", s.Path, s.Source, state))
	}
	summary := []string{
		fmt.Sprintf("Stray model bytes outside receipts: %s (adoptable %s, reclaimable %s, other software's or unknown %s); managed library caches %s.", humanBytes(report.Totals.Stray), humanBytes(report.Totals.Adoptable), humanBytes(report.Totals.Reclaim), humanBytes(report.Totals.ReportOnly), humanBytes(report.Totals.Cache)),
		"Scanned: " + strings.Join(scanned, "; "),
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Findings (adopt: `models adopt-stray <id>`; reclaim: `models reclaim-stray <id>`)", Results: rows}
}

func (h *handlers) strayAction(action string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		id := ctx.Positional("id")
		var receipt map[string]any
		if err := h.restDo(http.MethodPost, "/api/v1/models/strays/"+action, map[string]string{"id": id}, &receipt); err != nil {
			return fmt.Errorf("%s stray %s: %w", action, id, err)
		}
		if ctx.JSON() {
			return json.NewEncoder(ctx.Stdout()).Encode(receipt)
		}
		line := fmt.Sprintf("%s %v (%.0f bytes)", strings.ToUpper(action[:1])+action[1:], receipt["path"], receipt["bytes"])
		if dest, _ := receipt["destination"].(string); dest != "" {
			line += " -> " + dest
		}
		out := []string{line}
		if note, _ := receipt["note"].(string); note != "" {
			out = append(out, note)
		}
		return ctx.RenderList(cliapp.ListReport{Summary: out})
	}
}

type containmentReport struct {
	Env struct {
		ManagedRoot string            `json:"managed_root"`
		CacheRoot   string            `json:"cache_root"`
		Pins        map[string]string `json:"pins"`
		Inference   []string          `json:"inference_flags"`
		Acquisition []string          `json:"acquisition_flags"`
	} `json:"env"`
	Disk struct {
		FloorBytes          int64    `json:"floor_bytes"`
		FloorReasons        []string `json:"floor_reasons"`
		DiskTotalBytes      int64    `json:"disk_total_bytes"`
		FreeBytes           int64    `json:"free_bytes"`
		HeadroomBytes       int64    `json:"headroom_bytes"`
		FloorPercentOfDisk  float64  `json:"floor_percent_of_disk"`
		SmallWriteFloor     string   `json:"small_write_floor"`
		InstallPeakFormula  string   `json:"install_peak_formula"`
		JobAdmissionSummary string   `json:"job_admission_summary"`
	} `json:"disk"`
}

func (h *handlers) containment(ctx cliapp.RunContext) error {
	var report containmentReport
	if err := h.restDo(http.MethodGet, "/api/v1/models/containment", nil, &report); err != nil {
		return fmt.Errorf("read containment: %w", err)
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(report)
	}
	pins := make([]string, 0, len(report.Env.Pins))
	for name, dir := range report.Env.Pins {
		pins = append(pins, name+"="+dir)
	}
	sort.Strings(pins)
	rows := append([]string{}, pins...)
	rows = append(rows, "inference flags: "+strings.Join(report.Env.Inference, " "), "acquisition flags: "+strings.Join(report.Env.Acquisition, " "))
	summary := []string{
		fmt.Sprintf("Every model-loading process runs with %d cache/home variables pinned under %s.", len(report.Env.Pins), report.Env.CacheRoot),
		fmt.Sprintf("Disk floor %s (%.1f%% of a %s disk); %s free, %s above the floor.", humanBytes(report.Disk.FloorBytes), report.Disk.FloorPercentOfDisk, humanBytes(report.Disk.DiskTotalBytes), humanBytes(report.Disk.FreeBytes), humanBytes(report.Disk.HeadroomBytes)),
	}
	for _, reason := range report.Disk.FloorReasons {
		summary = append(summary, "  floor reason: "+reason)
	}
	summary = append(summary, "Small writes: "+report.Disk.SmallWriteFloor, "Install peak: "+report.Disk.InstallPeakFormula, "Admission: "+report.Disk.JobAdmissionSummary)
	return ctx.RenderList(cliapp.ListReport{Summary: summary, ResultsHeading: "Pinned variables and flags", Results: rows})
}
