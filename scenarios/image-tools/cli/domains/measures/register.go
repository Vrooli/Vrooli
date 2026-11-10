// Package measures exposes query surfaces for owner-recorded runtime evidence.
package measures

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "measures"

type modelProfile struct {
	Operation      string `json:"operation"`
	ModelID        string `json:"model_id"`
	Profile        string `json:"profile"`
	Status         string `json:"status"`
	Samples        int    `json:"samples"`
	LatencyP50MS   int64  `json:"latency_p50_ms"`
	LatencyP95MS   int64  `json:"latency_p95_ms"`
	PeakMemoryB    int64  `json:"peak_memory_bytes"`
	Device         string `json:"device"`
	LastEvidence   string `json:"last_evidence"`
	LastRecordedAt string `json:"last_recorded_at"`
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"model_profiles": func(ctx cliapp.RunContext) error {
			client, baseURL := cliapp.NewConnectHTTPClient(core)
			profiles, err := queryProfiles(context.Background(), client, baseURL, ctx.Flag("model"), ctx.Flag("operation"))
			if err != nil {
				return err
			}
			if ctx.JSON() {
				return json.NewEncoder(ctx.Stdout()).Encode(profiles)
			}
			rows := make([]string, 0, len(profiles))
			for _, p := range profiles {
				rows = append(rows, fmt.Sprintf("%s / %s / %s: %s (%d samples, p50 %d ms, p95 %d ms, peak %d bytes, %s)", p.Operation, p.ModelID, p.Profile, p.Status, p.Samples, p.LatencyP50MS, p.LatencyP95MS, p.PeakMemoryB, p.Device))
			}
			if len(rows) == 0 {
				rows = []string{"No model profiles have been recorded yet."}
			}
			return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("%d model profile(s)", len(profiles))}, ResultsHeading: "Model profiles", Results: rows})
		},
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("measures: load from manifest: %w", err)
	}
	return group, nil
}

func queryProfiles(ctx context.Context, client httpDoer, baseURL, modelID, operation string) ([]modelProfile, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/api/v1/measures/models"
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("measures: build model profile query: %w", err)
	}
	query := parsed.Query()
	if modelID != "" {
		query.Set("model_id", modelID)
	}
	if operation != "" {
		query.Set("operation", operation)
	}
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("measures: build model profile query: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("measures: query model profiles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20+1))
	if err != nil {
		return nil, fmt.Errorf("measures: read model profile response: %w", err)
	}
	if len(body) > 4<<20 {
		return nil, fmt.Errorf("measures: model profile response exceeds 4 MiB")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("measures: API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var profiles []modelProfile
	if err := json.Unmarshal(body, &profiles); err != nil {
		return nil, fmt.Errorf("measures: decode model profiles: %w", err)
	}
	return profiles, nil
}
