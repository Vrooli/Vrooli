package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type VPSDeployRequest struct {
	Manifest CloudManifest `json:"manifest"`
}

type VPSDeployResult struct {
	OK        bool          `json:"ok"`
	Steps     []VPSPlanStep `json:"steps"`
	Error     string        `json:"error,omitempty"`
	Timestamp string        `json:"timestamp"`
}

func BuildVPSDeployPlan(manifest CloudManifest) ([]VPSPlanStep, error) {
	cfg := sshConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	uiPort := manifest.Ports.UI

	caddyfile := buildCaddyfile(manifest.Edge.Domain, uiPort)
	caddyfilePath := "/etc/caddy/Caddyfile"

	steps := []VPSPlanStep{
		{
			ID:          "caddy_install",
			Title:       "Install Caddy",
			Description: "Ensure Caddy is installed (apt) and enabled (systemd).",
			Command: localSSHCommand(cfg, strings.Join([]string{
				"command -v caddy >/dev/null || (apt-get update -y && apt-get install -y caddy)",
				"systemctl enable --now caddy",
			}, " && ")),
		},
		{
			ID:          "caddy_config",
			Title:       "Configure Caddy",
			Description: "Write a minimal Caddyfile and reload.",
			Command: localSSHCommand(cfg, strings.Join([]string{
				fmt.Sprintf("printf '%%s' %s > %s", shellQuoteSingle(caddyfile), shellQuoteSingle(caddyfilePath)),
				"systemctl reload caddy",
			}, " && ")),
		},
	}

	for _, res := range stableUniqueStrings(manifest.Dependencies.Resources) {
		steps = append(steps, VPSPlanStep{
			ID:          "resource_start_" + res,
			Title:       "Start resource: " + res,
			Description: "Start required Vrooli resources via the mini install.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("cd %s && vrooli resource start %s", shellQuoteSingle(workdir), shellQuoteSingle(res))),
		})
	}

	for _, scen := range stableUniqueStrings(manifest.Dependencies.Scenarios) {
		if scen == manifest.Scenario.ID {
			continue
		}
		steps = append(steps, VPSPlanStep{
			ID:          "scenario_start_" + scen,
			Title:       "Start scenario: " + scen,
			Description: "Start dependent scenarios (excluding the target).",
			Command:     localSSHCommand(cfg, fmt.Sprintf("cd %s && vrooli scenario start %s", shellQuoteSingle(workdir), shellQuoteSingle(scen))),
		})
	}

	steps = append(steps,
		VPSPlanStep{
			ID:          "scenario_start_target",
			Title:       "Start target scenario with fixed ports",
			Description: "Starts the target scenario with UI_PORT/API_PORT/WS_PORT overrides.",
			Command: localSSHCommand(cfg, fmt.Sprintf(
				"cd %s && UI_PORT=%d API_PORT=%d WS_PORT=%d vrooli scenario start %s",
				shellQuoteSingle(workdir),
				manifest.Ports.UI,
				manifest.Ports.API,
				manifest.Ports.WS,
				shellQuoteSingle(manifest.Scenario.ID),
			)),
		},
		VPSPlanStep{
			ID:          "verify_local",
			Title:       "Verify local health",
			Description: "Checks the UI health endpoint locally on the VPS.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("curl -fsS --max-time 5 http://127.0.0.1:%d/health", uiPort)),
		},
		VPSPlanStep{
			ID:          "verify_https",
			Title:       "Verify HTTPS health",
			Description: "Checks https://<domain>/health via Caddy + Let’s Encrypt.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("curl -fsS --max-time 10 https://%s/health", manifest.Edge.Domain)),
		},
	)

	return steps, nil
}

func RunVPSDeploy(ctx context.Context, manifest CloudManifest, sshRunner SSHRunner) VPSDeployResult {
	steps, err := BuildVPSDeployPlan(manifest)
	if err != nil {
		return VPSDeployResult{OK: false, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	cfg := sshConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir

	run := func(cmd string) error {
		res, err := sshRunner.Run(ctx, cfg, cmd)
		if err != nil {
			if res.Stderr != "" {
				return fmt.Errorf("%w: %s", err, res.Stderr)
			}
			return err
		}
		return nil
	}

	if err := run("command -v caddy >/dev/null || (apt-get update -y && apt-get install -y caddy)"); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run("systemctl enable --now caddy"); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	caddyfilePath := "/etc/caddy/Caddyfile"
	caddyfile := buildCaddyfile(manifest.Edge.Domain, manifest.Ports.UI)
	if err := run(fmt.Sprintf("printf '%%s' %s > %s", shellQuoteSingle(caddyfile), shellQuoteSingle(caddyfilePath))); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run("systemctl reload caddy"); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	for _, res := range stableUniqueStrings(manifest.Dependencies.Resources) {
		if err := run(fmt.Sprintf("cd %s && vrooli resource start %s", shellQuoteSingle(workdir), shellQuoteSingle(res))); err != nil {
			return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
		}
	}

	for _, scen := range stableUniqueStrings(manifest.Dependencies.Scenarios) {
		if scen == manifest.Scenario.ID {
			continue
		}
		if err := run(fmt.Sprintf("cd %s && vrooli scenario start %s", shellQuoteSingle(workdir), shellQuoteSingle(scen))); err != nil {
			return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
		}
	}

	if err := run(fmt.Sprintf("cd %s && UI_PORT=%d API_PORT=%d WS_PORT=%d vrooli scenario start %s", shellQuoteSingle(workdir), manifest.Ports.UI, manifest.Ports.API, manifest.Ports.WS, shellQuoteSingle(manifest.Scenario.ID))); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	if err := run(fmt.Sprintf("curl -fsS --max-time 5 http://127.0.0.1:%d/health", manifest.Ports.UI)); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(fmt.Sprintf("curl -fsS --max-time 10 https://%s/health", manifest.Edge.Domain)); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	return VPSDeployResult{OK: true, Steps: steps, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

func buildCaddyfile(domain string, uiPort int) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = "example.com"
	}
	return fmt.Sprintf(`%s {
  reverse_proxy 127.0.0.1:%d
}`, domain, uiPort)
}
