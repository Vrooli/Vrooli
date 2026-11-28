package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// BootRecoveryStatus is the autoheal API's boot-recovery readiness projection
// (the system-boot-recovery-readiness check), as `vrooli setup status`
// prints it. It is read, never computed here: the check owns the verdict.
type BootRecoveryStatus struct {
	Status        string `json:"status"`
	Message       string `json:"message"`
	Remediation   string `json:"remediation"`
	EvaluatedAt   string `json:"evaluatedAt"`
	Preconditions []struct {
		Name   string `json:"name"`
		State  string `json:"state"`
		Reason string `json:"reason"`
	} `json:"preconditions"`
}

const (
	autohealScenario         = "vrooli-autoheal"
	bootRecoveryReadinessRPC = "/vrooli.vrooli_autoheal.v1.healing.HealingService/GetReadiness"
	bootRecoveryFetchTimeout = 5 * time.Second
)

// fetchBootRecovery asks the running autoheal API for its typed readiness and
// returns the boot_recovery projection. Any failure to reach or parse the API
// is an error; the caller prints "unknown", never a guess.
func fetchBootRecovery(ctx context.Context) (BootRecoveryStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, bootRecoveryFetchTimeout)
	defer cancel()
	base, err := discovery.ResolveScenarioURLDefault(ctx, autohealScenario)
	if err != nil {
		return BootRecoveryStatus{}, fmt.Errorf("resolve %s: %w", autohealScenario, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+bootRecoveryReadinessRPC, bytes.NewReader([]byte("{}")))
	if err != nil {
		return BootRecoveryStatus{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return BootRecoveryStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return BootRecoveryStatus{}, fmt.Errorf("%s returned %s: %s", bootRecoveryReadinessRPC, resp.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		BootRecovery *BootRecoveryStatus `json:"bootRecovery"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return BootRecoveryStatus{}, fmt.Errorf("decode readiness: %w", err)
	}
	if payload.BootRecovery == nil {
		return BootRecoveryStatus{}, fmt.Errorf("readiness carries no boot_recovery projection")
	}
	return *payload.BootRecovery, nil
}

// renderBootRecovery prints the "Boot recovery" block. An unreachable API is
// printed as unknown with the reason: the absence of an answer is itself the
// operator's signal that boot protection is unverified.
func renderBootRecovery(stdout io.Writer, status BootRecoveryStatus, err error) {
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "[INFO]    Boot recovery")
		_, _ = fmt.Fprintf(stdout, "[INFO]      boot recovery: unknown (autoheal API not reachable: %v)\n", err)
		return
	}
	_, _ = fmt.Fprintln(stdout, "[INFO]    Boot recovery")
	line := "boot recovery: " + status.Status
	if status.Message != "" {
		line += " — " + status.Message
	}
	_, _ = fmt.Fprintf(stdout, "[INFO]      %s\n", line)
	for _, p := range status.Preconditions {
		_, _ = fmt.Fprintf(stdout, "[INFO]        %-14s %-12s %s\n", p.Name, p.State, p.Reason)
	}
	if status.Status != "ok" && status.Remediation != "" {
		_, _ = fmt.Fprintf(stdout, "[INFO]      remediation: %s\n", status.Remediation)
	}
}
