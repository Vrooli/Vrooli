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
	}
	if err := h.call(context.Background(), http.MethodPost, "/api/v1/packaged/capture", request, &response); err != nil {
		return err
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(response)
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Captured %d %s packaged launches for %s", len(response.Samples), ctx.Flag("condition"), ctx.Flag("scenario")), "Experiment: " + response.ExperimentID}})
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
			Condition    string `json:"condition"`
			ExperimentID string `json:"experiment_id"`
			Samples      int    `json:"samples"`
			Comparable   bool   `json:"comparable"`
			Metrics      map[string]struct {
				P50       *float64 `json:"p50"`
				Available int      `json:"available_samples"`
			} `json:"metrics"`
		} `json:"cells"`
	}
	if err := json.Unmarshal(response, &decoded); err != nil {
		return err
	}
	lines := make([]string, 0, len(decoded.Cells))
	comparableCount := 0
	for _, group := range decoded.Cells {
		if group.Comparable {
			comparableCount++
		}
		interactive := group.Metrics["app_interactive_ms"].P50
		value := "unavailable"
		if interactive != nil {
			value = fmt.Sprintf("%.1f ms", *interactive)
		}
		lines = append(lines, fmt.Sprintf("%s %s (%s): %d samples, app_interactive p50 %s, comparable=%t", group.Scenario, group.Condition, group.ExperimentID, group.Samples, value, group.Comparable))
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Packaged launch report: %d comparable of %d cells", comparableCount, len(lines))}, ResultsHeading: "Packaged cells", Results: lines})
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
