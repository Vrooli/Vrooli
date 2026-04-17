package vps

import (
	"context"
	"fmt"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/vps/systemmetrics"
	"sync"
	"time"
)

// RunSystemMetricsDebug executes all system metrics commands and returns both
// raw command output and the parsed metrics snapshot for troubleshooting.
func RunSystemMetricsDebug(ctx context.Context, manifest domain.CloudManifest, sshRunner ssh.Runner) domain.MetricsDebugResult {
	resolver := sshidentity.DefaultResolver{}
	identity, _ := resolver.Resolve(manifest, nil)
	cfg := sshidentity.EffectiveSSHConfig(manifest, identity)
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
	osReleaseCmd := "cat /etc/os-release 2>/dev/null || true"
	osReleaseRes, osReleaseErr := sshRunner.Run(ctx, cfg, osReleaseCmd, ssh.DefaultRunOptions())
	if osReleaseErr == nil {
		result.OSID, result.OSVersion = systemmetrics.ParseOSRelease(osReleaseRes.Stdout)
		if result.OSID != "" {
			collector = systemmetrics.CollectorForOS(result.OSID)
		}
	}
	result.Collector = collector.Name()

	commands := []sshCommand{
		{id: "ssh_ping", command: "echo ok"},
		{id: "ssh_key_check", command: "cat ~/.ssh/authorized_keys 2>/dev/null || echo ''"},
	}
	for _, spec := range collector.SystemCommands() {
		// In debug mode we prefer smoothed CPU over multiple samples.
		if spec.ID == "cpuusage" {
			spec.Command = "for i in 1 2 3 4; do cat /proc/stat | head -1; if [ \"$i\" -lt 4 ]; then sleep 1; fi; done"
		}
		commands = append(commands, sshCommand{id: spec.ID, command: spec.Command})
	}

	results := make(map[string]sshCommandResult, len(commands))
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		first error
	)

	for _, cmd := range commands {
		wg.Add(1)
		go func(c sshCommand) {
			defer wg.Done()
			cmdStart := time.Now()
			res, err := sshRunner.Run(ctx, cfg, c.command, ssh.DefaultRunOptions())

			mu.Lock()
			results[c.id] = sshCommandResult{
				id:         c.id,
				result:     res,
				err:        err,
				durationMs: time.Since(cmdStart).Milliseconds(),
			}
			if err != nil && first == nil {
				first = fmt.Errorf("%s: %w", c.id, err)
			}
			mu.Unlock()
		}(cmd)
	}
	wg.Wait()

	if ctx.Err() != nil {
		result.OK = false
		result.Error = "context cancelled: " + ctx.Err().Error()
		return result
	}

	result.System = parseSystemState(results, identity, pubKeyContent, collector)
	result.OK = result.System.SSH.Connected
	if !result.OK {
		if first != nil {
			result.Error = first.Error()
		} else {
			result.Error = "ssh connectivity check failed"
		}
	}

	result.Commands = make([]domain.MetricsDebugCommand, 0, len(commands)+1)
	result.Commands = append(result.Commands, domain.MetricsDebugCommand{
		ID:         "os_release",
		Command:    osReleaseCmd,
		Stdout:     osReleaseRes.Stdout,
		Stderr:     osReleaseRes.Stderr,
		ExitCode:   osReleaseRes.ExitCode,
		DurationMs: 0,
		Error:      errString(osReleaseErr),
	})

	for _, cmd := range commands {
		execRes, ok := results[cmd.id]
		if !ok {
			result.Commands = append(result.Commands, domain.MetricsDebugCommand{
				ID:      cmd.id,
				Command: cmd.command,
				Error:   "missing command result",
			})
			continue
		}
		result.Commands = append(result.Commands, domain.MetricsDebugCommand{
			ID:         cmd.id,
			Command:    cmd.command,
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
