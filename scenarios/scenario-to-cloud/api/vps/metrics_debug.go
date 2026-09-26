package vps

import (
	"context"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/vps/systemmetrics"
)

// RunSystemMetricsDebug runs every system observation through the prober and
// returns both the raw answers and the parsed metrics snapshot for
// troubleshooting.
func RunSystemMetricsDebug(ctx context.Context, identity sshidentity.DeploymentSSHIdentity, prober Prober) domain.MetricsDebugResult {
	pubKeyContent := ""
	if identity.AuthMode == sshidentity.AuthModeExplicitKey && identity.KeyPath != "" {
		if content, _, err := sshidentity.ReadPublicKeyAndFingerprint(identity.KeyPath); err == nil {
			pubKeyContent = content
		}
	}

	result := domain.MetricsDebugResult{
		OK:        true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	collector := systemmetrics.CollectorForOS("linux")
	osRelease := observe("os_release", "cat", "/etc/os-release")
	osStart := time.Now()
	osReleaseRes, osReleaseErr := prober.Reach.Exec(ctx, prober.Target, osRelease.cmd)
	osDuration := time.Since(osStart).Milliseconds()
	if osReleaseErr == nil {
		result.OSID, result.OSVersion = systemmetrics.ParseOSRelease(osReleaseRes.Stdout)
		if result.OSID != "" {
			collector = systemmetrics.CollectorForOS(result.OSID)
		}
	}
	result.Collector = collector.Name()

	probes := []probe{
		observe("ssh_ping", "uname", "-s"),
		observe("ssh_key_check", "cat", HomeDir(prober.Target.Locator.User)+"/.ssh/authorized_keys"),
	}
	for _, spec := range collector.SystemCommands() {
		if spec.ID == systemmetrics.CPUUsageProbeID {
			continue
		}
		probes = append(probes, observe(spec.ID, spec.Program, spec.Args...))
	}
	results := runProbes(ctx, prober, probes)
	cpu := sampleCPUUsage(ctx, prober, collector)
	results[cpu.id] = cpu

	if ctx.Err() != nil {
		result.OK = false
		result.Error = "context cancelled: " + ctx.Err().Error()
		return result
	}

	var first error
	for _, p := range probes {
		if res := results[p.id]; res.err != nil && first == nil {
			first = fmt.Errorf("%s: %w", p.id, res.err)
		}
	}

	result.System = parseSystemState(results, identity, pubKeyContent, collector)
	result.OK = result.System.SSH.Connected
	if !result.OK {
		if first != nil {
			result.Error = first.Error()
		} else {
			result.Error = "target reachability probe failed"
		}
	}

	result.Commands = make([]domain.MetricsDebugCommand, 0, len(probes)+2)
	result.Commands = append(result.Commands, domain.MetricsDebugCommand{
		ID:         "os_release",
		Command:    strings.Join(osRelease.cmd.Argv(), " "),
		Stdout:     osReleaseRes.Stdout,
		Stderr:     osReleaseRes.Stderr,
		ExitCode:   osReleaseRes.ExitCode,
		DurationMs: osDuration,
		Error:      errString(osReleaseErr),
	})
	for _, p := range append(probes, probe{id: cpu.id, cmd: observe(cpu.id, "cat", "/proc/stat").cmd}) {
		execRes, ok := results[p.id]
		if !ok {
			result.Commands = append(result.Commands, domain.MetricsDebugCommand{ID: p.id, Command: strings.Join(p.cmd.Argv(), " "), Error: "missing probe result"})
			continue
		}
		result.Commands = append(result.Commands, domain.MetricsDebugCommand{
			ID:         p.id,
			Command:    strings.Join(p.cmd.Argv(), " "),
			Stdout:     execRes.result.Stdout,
			Stderr:     execRes.result.Stderr,
			ExitCode:   execRes.result.ExitCode,
			DurationMs: execRes.durationMs,
			Error:      errString(execRes.err),
		})
	}

	return result
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
