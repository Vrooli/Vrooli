package packaged

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client interface {
		Do(*http.Request) (*http.Response, error)
	}
	base string
}

func (h handlers) capture(ctx cliapp.RunContext) error {
	runs, err := strconv.Atoi(ctx.Flag("runs"))
	if err != nil {
		return fmt.Errorf("--runs must be an integer: %w", err)
	}
	request := map[string]any{"scenario": ctx.Flag("scenario"), "artifact_path": ctx.Flag("artifact"), "ramp": ctx.Flag("ramp"), "strategy": ctx.Flag("strategy"), "optimization_toggles": map[string]bool{"shared_runtime_readiness": ctx.BoolFlag("shared-runtime-readiness"), "secure_custom_scheme_ui": ctx.BoolFlag("secure-custom-scheme-ui"), "v8_snapshot_code_cache": ctx.BoolFlag("v8-snapshot-code-cache"), "electron_fuses": ctx.BoolFlag("electron-fuses")}, "runs": runs, "condition": ctx.Flag("condition"), "experiment_id": ctx.Flag("experiment"), "journey": ctx.Flag("journey")}
	var response struct {
		ExperimentID string            `json:"experiment_id"`
		Samples      []json.RawMessage `json:"samples"`
		PlannedWork  json.RawMessage   `json:"planned_work,omitempty"`
	}
	path := "/api/v1/packaged/capture"
	if ctx.BoolFlag("plan") {
		path += "/plan"
	}
	if err := h.call(context.Background(), http.MethodPost, path, request, &response); err != nil {
		return err
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(response)
	}
	verb, count := "Captured", len(response.Samples)
	if ctx.BoolFlag("plan") {
		verb, count = "Planned", runs
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("%s %d %s packaged samples for %s", verb, count, ctx.Flag("condition"), ctx.Flag("scenario")), "Experiment: " + response.ExperimentID, "Planned work: " + string(response.PlannedWork)}})
}

func (h handlers) report(ctx cliapp.RunContext) error {
	query := url.Values{}
	for flag, key := range map[string]string{"scenario": "scenario", "ramp": "ramp", "strategy": "strategy", "condition": "condition", "experiment": "experiment_id"} {
		if value := strings.TrimSpace(ctx.Flag(flag)); value != "" {
			query.Set(key, value)
		}
	}
	var response json.RawMessage
	if err := h.call(context.Background(), http.MethodGet, "/api/v1/packaged/report?"+query.Encode(), nil, &response); err != nil {
		return err
	}
	if ctx.JSON() {
		_, err := ctx.Stdout().Write(append(response, '\n'))
		return err
	}
	var decoded struct {
		Cells []struct {
			Scenario     string `json:"scenario"`
			Strategy     string `json:"strategy"`
			Condition    string `json:"condition"`
			ExperimentID string `json:"experiment_id"`
			Samples      int    `json:"samples"`
			Comparable   bool   `json:"comparable"`
			Metrics      map[string]struct {
				P50 *float64 `json:"p50"`
			} `json:"metrics"`
		} `json:"cells"`
	}
	if err := json.Unmarshal(response, &decoded); err != nil {
		return err
	}
	lines := make([]string, 0, len(decoded.Cells))
	comparableCount := 0
	format := func(value *float64, unit string) string {
		if value == nil {
			return "unavailable"
		}
		return fmt.Sprintf("%.1f %s", *value, unit)
	}
	for _, group := range decoded.Cells {
		if group.Comparable {
			comparableCount++
		}
		lines = append(lines, fmt.Sprintf("%s %s (%s): strategy=%s, %d samples, app_interactive p50 %s, steady PSS p50 %s, artifact p50 %s, comparable=%t", group.Scenario, group.Condition, group.ExperimentID, group.Strategy, group.Samples, format(group.Metrics["app_interactive_ms"].P50, "ms"), format(group.Metrics["steady_pss_mb"].P50, "MB"), format(group.Metrics["artifact_bytes"].P50, "bytes"), group.Comparable))
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Packaged launch report: %d comparable of %d cells", comparableCount, len(lines)), "Measurements are not an adoption decision; review sample count, target gaps, parity and resource tradeoffs."}, ResultsHeading: "Packaged cells", Results: lines})
}

func (h handlers) compare(ctx cliapp.RunContext) error {
	request := map[string]string{"a": ctx.Positional("a"), "b": ctx.Positional("b")}
	var response json.RawMessage
	if err := h.call(context.Background(), http.MethodPost, "/api/v1/packaged/compare", request, &response); err != nil {
		return err
	}
	if ctx.JSON() {
		_, err := ctx.Stdout().Write(append(response, '\n'))
		return err
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{"Compared packaged experiments", request["a"] + " versus " + request["b"]}})
}

func (h handlers) call(parent context.Context, method, path string, body any, out any) error {
	ctx, cancel := context.WithTimeout(parent, 65*time.Minute)
	defer cancel()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(h.base, "/")+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("call packaged API: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 64<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("packaged API returned %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode packaged API response: %w", err)
	}
	return nil
}
