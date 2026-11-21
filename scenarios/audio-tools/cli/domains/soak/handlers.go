package soak

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func (h *handlers) run(ctx cliapp.RunContext) error {
	identityToken := strings.TrimSpace(h.getenv("VROOLI_AGENT_IDENTITY_TOKEN"))
	grantReceipt := strings.TrimSpace(h.getenv("VROOLI_BROWSER_GRANT_RECEIPT"))
	testLease := strings.TrimSpace(h.getenv("TEST_GENIE_DEMAND_LEASE_RECEIPT_ID"))
	if identityToken == "" || grantReceipt == "" || testLease == "" {
		return errors.New("soak run requires trusted Agent Manager identity and browser grant plus the active Test Genie demand lease receipt")
	}
	engineID := firstNonEmpty(ctx.Flag("engine-id"))
	modelID := firstNonEmpty(ctx.Flag("model-id"))
	if engineID == "" || modelID == "" {
		return errors.New("soak run requires --engine-id and --model-id so evidence is tied to an exact provider cell")
	}
	if err := h.verifyAudioToolsAPI(); err != nil {
		return err
	}

	lane := firstNonEmpty(ctx.Flag("lane"), "realtime")
	turns, err := positiveInt(ctx.Flag("turns"), 3)
	if err != nil {
		return fmt.Errorf("--turns: %w", err)
	}
	feedMS, err := positiveInt(ctx.Flag("feed-ms"), defaultFeedMS(lane))
	if err != nil {
		return fmt.Errorf("--feed-ms: %w", err)
	}
	simulatedMinutes, err := positiveInt(ctx.Flag("simulated-minutes"), 0)
	if err != nil {
		return fmt.Errorf("--simulated-minutes: %w", err)
	}
	referenceCycleMS, err := positiveInt(ctx.Flag("reference-cycle-ms"), 0)
	if err != nil {
		return fmt.Errorf("--reference-cycle-ms: %w", err)
	}
	if lane == "accelerated" && simulatedMinutes == 0 {
		simulatedMinutes = 60
	}
	profile := firstNonEmpty(ctx.Flag("profile"), "realistic")
	request := map[string]any{
		"lane": lane, "profile": profile, "turns": turns, "feed_ms": feedMS,
		"fault": ctx.Flag("fault"), "reference_text": ctx.Flag("reference-text"),
		"reference_cycle_ms": referenceCycleMS,
		"engine_id":          engineID,
		"model_id":           modelID,
		"strategy":           firstNonEmpty(ctx.Flag("strategy"), "product"),
		"policy":             firstNonEmpty(ctx.Flag("policy"), "default"),
		"shape":              firstNonEmpty(ctx.Flag("shape"), "burst"),
		"simulated_minutes":  simulatedMinutes,
		"mobile_recovery":    strings.EqualFold(strings.TrimSpace(ctx.Flag("mobile-recovery")), "true"),
	}
	headers := map[string]string{
		"X-Agent-Identity-Token":         identityToken,
		"X-Vrooli-Browser-Grant-Receipt": grantReceipt,
		"X-Test-Genie-Lease-Receipt":     testLease,
	}
	requestTimeout := 10 * time.Minute
	if lane == "realtime" {
		captureBudget := time.Duration(feedMS) * time.Duration(turns) * time.Millisecond
		if candidate := captureBudget + 10*time.Minute; candidate > requestTimeout {
			requestTimeout = candidate
		}
	}
	response, err := h.core.APIClient.WithTimeout(requestTimeout).RequestWithHeaders("POST", h.core.APIPath("/validation/soak"), nil, request, headers)
	if err != nil {
		return cliapp.WrapAPIError("soak run", err, nil)
	}
	var doc runDocument
	if err := json.Unmarshal(response, &doc); err != nil {
		return fmt.Errorf("decode conformance run: %w", err)
	}
	evidencePath, err := h.resolveEvidencePath(ctx.Flag("evidence-path"))
	if err != nil {
		return err
	}
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

func defaultFeedMS(lane string) int {
	if strings.EqualFold(strings.TrimSpace(lane), "accelerated") {
		// Virtual capture advances the qualification corpus independently of
		// wall-clock feed time. Keep the browser lifecycle exercised, but do not
		// spend four real seconds per virtual turn and then fail the declared
		// accelerated <=60s budget.
		return 50
	}
	return 4000
}

func (h *handlers) resolveEvidencePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw), nil
	}

	root, err := h.scenarioRoot()
	if err != nil {
		return "", err
	}
	candidate := filepath.Clean(filepath.Join(root, raw))
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return "", fmt.Errorf("validate evidence path: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("relative evidence path %q escapes the audio-tools scenario; use an absolute path for an external export", raw)
	}
	return candidate, nil
}

func (h *handlers) scenarioRoot() (string, error) {
	if configured := strings.TrimSpace(h.getenv("AUDIO_TOOLS_SCENARIO_ROOT")); configured != "" {
		return filepath.Abs(configured)
	}
	directory, err := h.getwd()
	if err != nil {
		return "", fmt.Errorf("resolve audio-tools scenario root: %w", err)
	}
	for {
		candidate := filepath.Join(directory, "scenarios", "audio-tools")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, nil
		}
		if filepath.Base(directory) == "audio-tools" && filepath.Base(filepath.Dir(directory)) == "scenarios" {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", fmt.Errorf("audio-tools scenario root not found from %q", directory)
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
