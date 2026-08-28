package soak

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

type runDocument struct {
	SchemaVersion int    `json:"schemaVersion"`
	RunID         string `json:"runId"`
	Lane          string `json:"lane"`
	Assertions    []struct {
		Name    string `json:"name"`
		Outcome string `json:"outcome"`
		Detail  string `json:"detail"`
	} `json:"assertions"`
}

type soakHealthDocument struct {
	Service string `json:"service"`
}

type soakDriverHealthDocument struct {
	Ready            bool `json:"ready"`
	Sessions         int  `json:"sessions"`
	ActiveRecordings int  `json:"active_recordings"`
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	driverURL := firstNonEmpty(ctx.Flag("driver-url"), h.getenv("PLAYWRIGHT_DRIVER_URL"))
	uiURL := firstNonEmpty(ctx.Flag("ui-url"), h.getenv("AUDIO_TOOLS_UI_URL"), h.getenv("UI_BASE_URL"))
	fixture := firstNonEmpty(ctx.Flag("fixture"), h.getenv("AUDIO_TOOLS_SOAK_FIXTURE"))
	if fixture == "" {
		fixture = filepath.Join("bas", "fixtures", "dictation-reference.wav")
	}
	if !filepath.IsAbs(fixture) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		fixture = filepath.Join(wd, fixture)
	}
	if driverURL == "" || uiURL == "" {
		return errors.New("soak run requires --driver-url/PLAYWRIGHT_DRIVER_URL and --ui-url/AUDIO_TOOLS_UI_URL")
	}
	engineID := firstNonEmpty(ctx.Flag("engine-id"))
	modelID := firstNonEmpty(ctx.Flag("model-id"))
	if engineID == "" || modelID == "" {
		return errors.New("soak run requires --engine-id and --model-id so evidence is tied to an exact provider cell")
	}
	if _, err := os.Stat(fixture); err != nil {
		return fmt.Errorf("soak fixture %q: %w", fixture, err)
	}
	if err := h.verifyAudioToolsAPI(); err != nil {
		return err
	}
	if err := verifySoakDriver(driverURL); err != nil {
		return err
	}

	turns, err := positiveInt(ctx.Flag("turns"), 3)
	if err != nil {
		return fmt.Errorf("--turns: %w", err)
	}
	feedMS, err := positiveInt(ctx.Flag("feed-ms"), 4000)
	if err != nil {
		return fmt.Errorf("--feed-ms: %w", err)
	}
	lane := firstNonEmpty(ctx.Flag("lane"), "realtime")
	simulatedMinutes, err := positiveInt(ctx.Flag("simulated-minutes"), 0)
	if err != nil {
		return fmt.Errorf("--simulated-minutes: %w", err)
	}
	if lane == "accelerated" && simulatedMinutes == 0 {
		simulatedMinutes = 60
	}
	profile := firstNonEmpty(ctx.Flag("profile"), "realistic")
	request := map[string]any{
		"driver_url": driverURL, "ui_url": strings.TrimRight(uiURL, "/"), "fixture": fixture,
		"surface": firstNonEmpty(ctx.Flag("surface"), "audio-tools"),
		"lane":    lane, "profile": profile, "turns": turns, "feed_ms": feedMS,
		"fault": ctx.Flag("fault"), "reference_text": ctx.Flag("reference-text"),
		"engine_id":         engineID,
		"model_id":          modelID,
		"strategy":          firstNonEmpty(ctx.Flag("strategy"), "product"),
		"policy":            firstNonEmpty(ctx.Flag("policy"), "default"),
		"shape":             firstNonEmpty(ctx.Flag("shape"), "burst"),
		"simulated_minutes": simulatedMinutes,
	}
	requestTimeout := 10 * time.Minute
	if lane == "realtime" {
		captureBudget := time.Duration(feedMS) * time.Duration(turns) * time.Millisecond
		if candidate := captureBudget + 5*time.Minute; candidate > requestTimeout {
			requestTimeout = candidate
		}
	}
	response, err := h.core.APIClient.WithTimeout(requestTimeout).Request("POST", h.core.APIPath("/validation/soak"), nil, request)
	if err != nil {
		return cliapp.WrapAPIError("soak run", err, nil)
	}
	var doc runDocument
	if err := json.Unmarshal(response, &doc); err != nil {
		return fmt.Errorf("decode conformance run: %w", err)
	}
	evidencePath := strings.TrimSpace(ctx.Flag("evidence-path"))
	if evidencePath != "" {
		if err := os.MkdirAll(filepath.Dir(evidencePath), 0o750); err != nil {
			return fmt.Errorf("create evidence directory: %w", err)
		}
		if err := os.WriteFile(evidencePath, append(response, '\n'), 0o600); err != nil {
			return fmt.Errorf("write evidence %q: %w", evidencePath, err)
		}
	}
	if ctx.JSON() {
		_, err = ctx.Stdout().Write(response)
		if err == nil && !strings.HasSuffix(string(response), "\n") {
			_, err = io.WriteString(ctx.Stdout(), "\n")
		}
		if err != nil {
			return err
		}
		failed := 0
		for _, assertion := range doc.Assertions {
			if assertion.Outcome != "passed" {
				failed++
			}
		}
		if failed > 0 {
			return fmt.Errorf("soak run %s did not qualify: %d assertion(s) failed or were not measured", doc.RunID, failed)
		}
		return nil
	}
	failed := 0
	for _, assertion := range doc.Assertions {
		if assertion.Outcome != "passed" {
			failed++
			fmt.Fprintf(ctx.Stdout(), "%s: %s (%s)\n", assertion.Name, assertion.Outcome, assertion.Detail)
		}
	}
	if failed > 0 {
		return fmt.Errorf("soak run %s did not qualify: %d assertion(s) failed or were not measured", doc.RunID, failed)
	}
	message := fmt.Sprintf("audio-tools soak qualified: %s (%s); evidence retained by the API", doc.RunID, doc.Lane)
	if evidencePath != "" {
		message += "; exported=" + evidencePath
	}
	fmt.Fprintln(ctx.Stdout(), message)
	return nil
}

// verifySoakDriver prevents admission while another browser qualification (or
// an orphaned session from one) still occupies the shared BAS driver. This is
// deliberately a cheap capacity check: a long run must not discover a
// non-isolated browser state only after it has consumed its capture window.
func verifySoakDriver(driverURL string) error {
	base := strings.TrimRight(driverURL, "/")
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Get(base + "/health")
	if err != nil {
		return fmt.Errorf("soak preflight against BAS %s: %w", base, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("soak preflight: BAS %s returned HTTP %d", base, response.StatusCode)
	}
	var health soakDriverHealthDocument
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil {
		return fmt.Errorf("soak preflight: decode BAS health response from %s: %w", base, err)
	}
	if !health.Ready {
		return fmt.Errorf("soak preflight: BAS %s is not ready; wait for the driver to become ready", base)
	}
	if health.Sessions != 0 || health.ActiveRecordings != 0 {
		return fmt.Errorf("soak preflight: BAS %s has %d active session(s) and %d active recording(s); close or reconcile them before starting a qualification", base, health.Sessions, health.ActiveRecordings)
	}
	return nil
}

// verifyAudioToolsAPI prevents a long qualification from silently targeting
// the ambient scenario's API when the CLI is launched from another scenario's
// shell. A soak is expensive enough that a cheap identity check belongs before
// request admission, not after the first failed hour-long attempt.
func (h *handlers) verifyAudioToolsAPI() error {
	base := h.core.APIClient.BaseURL()
	response, err := h.core.APIClient.WithTimeout(10*time.Second).Get(h.core.HealthPath(), nil)
	if err != nil {
		return fmt.Errorf("soak preflight against %s: %w", base, err)
	}
	var health soakHealthDocument
	if err := json.Unmarshal(response, &health); err != nil {
		return fmt.Errorf("soak preflight: decode health response from %s: %w", base, err)
	}
	if strings.TrimSpace(health.Service) != "audio-tools-api" {
		return fmt.Errorf("soak preflight: API %s identifies as %q, want %q; set --api-base to the Audio Tools API", base, health.Service, "audio-tools-api")
	}
	return nil
}

func positiveInt(raw string, def int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, errors.New("must be a positive integer")
	}
	return n, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
