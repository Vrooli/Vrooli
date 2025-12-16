package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type PreflightCheckStatus string

const (
	PreflightPass PreflightCheckStatus = "pass"
	PreflightWarn PreflightCheckStatus = "warn"
	PreflightFail PreflightCheckStatus = "fail"
)

type PreflightCheck struct {
	ID      string             `json:"id"`
	Title   string             `json:"title"`
	Status  PreflightCheckStatus `json:"status"`
	Details string             `json:"details,omitempty"`
	Hint    string             `json:"hint,omitempty"`
	Data    map[string]string  `json:"data,omitempty"`
}

type PreflightResponse struct {
	OK        bool             `json:"ok"`
	Checks    []PreflightCheck `json:"checks"`
	Issues    []ValidationIssue `json:"issues,omitempty"`
	Timestamp string           `json:"timestamp"`
}

func RunVPSPreflight(
	ctx context.Context,
	manifest CloudManifest,
	resolver interface {
		LookupHost(ctx context.Context, host string) ([]string, error)
	},
	sshRunner SSHRunner,
) PreflightResponse {
	cfg := sshConfigFromManifest(manifest)
	checks := make([]PreflightCheck, 0, 8)
	fail := func(id, title, details, hint string, data map[string]string) {
		checks = append(checks, PreflightCheck{
			ID:      id,
			Title:   title,
			Status:  PreflightFail,
			Details: details,
			Hint:    hint,
			Data:    data,
		})
	}
	pass := func(id, title, details string, data map[string]string) {
		checks = append(checks, PreflightCheck{
			ID:      id,
			Title:   title,
			Status:  PreflightPass,
			Details: details,
			Data:    data,
		})
	}
	warn := func(id, title, details, hint string, data map[string]string) {
		checks = append(checks, PreflightCheck{
			ID:      id,
			Title:   title,
			Status:  PreflightWarn,
			Details: details,
			Hint:    hint,
			Data:    data,
		})
	}

	vpsIPs, vpsErr := resolveHostIPs(ctx, resolver, cfg.Host)
	if vpsErr != nil {
		fail(
			"dns_vps_host",
			"Resolve VPS host",
			fmt.Sprintf("Unable to resolve VPS host %q", cfg.Host),
			vpsErr.Error(),
			nil,
		)
	} else {
		pass("dns_vps_host", "Resolve VPS host", "Resolved VPS host", map[string]string{"ips": strings.Join(vpsIPs, ",")})
	}

	edgeIPs, edgeErr := resolveHostIPs(ctx, resolver, manifest.Edge.Domain)
	if edgeErr != nil {
		fail(
			"dns_edge_domain",
			"Resolve edge domain",
			fmt.Sprintf("Unable to resolve edge.domain %q", manifest.Edge.Domain),
			edgeErr.Error(),
			nil,
		)
	} else {
		pass("dns_edge_domain", "Resolve edge domain", "Resolved edge.domain", map[string]string{"ips": strings.Join(edgeIPs, ",")})
	}

	if vpsErr == nil && edgeErr == nil && !intersects(vpsIPs, edgeIPs) {
		fail(
			"dns_points_to_vps",
			"DNS points to VPS",
			"edge.domain does not resolve to the same IP(s) as target.vps.host",
			fmt.Sprintf("Update DNS for %s to point to one of: %s", manifest.Edge.Domain, strings.Join(vpsIPs, ", ")),
			map[string]string{"vps_ips": strings.Join(vpsIPs, ","), "edge_ips": strings.Join(edgeIPs, ",")},
		)
	} else if vpsErr == nil && edgeErr == nil {
		pass("dns_points_to_vps", "DNS points to VPS", "edge.domain resolves to the VPS", nil)
	}

	if _, err := sshRunner.Run(ctx, cfg, "echo ok"); err != nil {
		fail(
			"ssh_connect",
			"SSH connectivity",
			"Unable to run a remote command over SSH",
			"Confirm SSH key auth works (root login for P0) and that port 22 is reachable.",
			map[string]string{"host": cfg.Host, "user": cfg.User, "port": strconv.Itoa(cfg.Port)},
		)
	} else {
		pass("ssh_connect", "SSH connectivity", "SSH command executed successfully", map[string]string{"host": cfg.Host, "user": cfg.User})
	}

	osRes, osErr := sshRunner.Run(ctx, cfg, "cat /etc/os-release")
	if osErr != nil || osRes.ExitCode != 0 {
		fail(
			"os_release",
			"Ubuntu version",
			"Unable to read /etc/os-release",
			"Ensure the VPS is Ubuntu 24.04 and that /etc/os-release is readable.",
			map[string]string{"stderr": osRes.Stderr},
		)
	} else {
		id, ver := parseOSRelease(osRes.Stdout)
		if id != "ubuntu" {
			fail(
				"os_release",
				"Ubuntu version",
				fmt.Sprintf("Unsupported OS: %s", id),
				"P0 supports Ubuntu 24.04 only.",
				map[string]string{"id": id, "version_id": ver},
			)
		} else if ver != "24.04" {
			fail(
				"os_release",
				"Ubuntu version",
				fmt.Sprintf("Unsupported Ubuntu version: %s", ver),
				"P0 supports Ubuntu 24.04 only.",
				map[string]string{"id": id, "version_id": ver},
			)
		} else {
			pass("os_release", "Ubuntu version", "Ubuntu 24.04 detected", map[string]string{"id": id, "version_id": ver})
		}
	}

	portsRes, portsErr := sshRunner.Run(ctx, cfg, `ss -ltnH '( sport = :80 or sport = :443 )'`)
	if portsErr != nil {
		warn(
			"ports_80_443",
			"Ports 80/443 availability",
			"Unable to check ports 80/443 via ss",
			"Ensure ports 80 and 443 are free for Caddy/Let’s Encrypt HTTP-01.",
			map[string]string{"stderr": portsRes.Stderr},
		)
	} else if strings.TrimSpace(portsRes.Stdout) != "" {
		fail(
			"ports_80_443",
			"Ports 80/443 availability",
			"Port 80 and/or 443 appears to already be in use",
			"Stop whatever is bound to 80/443 before deploying Caddy.",
			map[string]string{"ss": portsRes.Stdout},
		)
	} else {
		pass("ports_80_443", "Ports 80/443 availability", "Ports 80/443 appear free", nil)
	}

	netRes, netErr := sshRunner.Run(ctx, cfg, `curl -fsS --max-time 5 https://example.com >/dev/null`)
	if netErr != nil || netRes.ExitCode != 0 {
		warn(
			"outbound_network",
			"Outbound network",
			"Unable to confirm outbound HTTPS access with curl",
			"Ensure outbound network access is allowed (apt/pnpm downloads, Let’s Encrypt).",
			map[string]string{"stderr": netRes.Stderr},
		)
	} else {
		pass("outbound_network", "Outbound network", "Outbound HTTPS access looks OK", nil)
	}

	diskRes, diskErr := sshRunner.Run(ctx, cfg, `df -Pk / | tail -n 1 | awk '{print $4}'`)
	if diskErr != nil || diskRes.ExitCode != 0 {
		warn("disk_free", "Disk free space", "Unable to determine free disk space", "Ensure the VPS has sufficient free disk for builds and resources.", nil)
	} else {
		kb, _ := strconv.ParseInt(strings.TrimSpace(diskRes.Stdout), 10, 64)
		const minKB = 5 * 1024 * 1024 // 5 GiB
		if kb > 0 && kb < minKB {
			fail("disk_free", "Disk free space", fmt.Sprintf("Low free disk space: %d KB", kb), "Increase disk size or free space before deploying.", map[string]string{"free_kb": diskRes.Stdout})
		} else {
			pass("disk_free", "Disk free space", "Disk free space looks OK", map[string]string{"free_kb": diskRes.Stdout})
		}
	}

	ramRes, ramErr := sshRunner.Run(ctx, cfg, `awk '/MemTotal/ {print $2}' /proc/meminfo`)
	if ramErr != nil || ramRes.ExitCode != 0 {
		warn("ram_total", "RAM", "Unable to determine total RAM", "Ensure the VPS has sufficient RAM for the scenario and resources.", nil)
	} else {
		kb, _ := strconv.ParseInt(strings.TrimSpace(ramRes.Stdout), 10, 64)
		const minKB = 1024 * 1024 // 1 GiB
		if kb > 0 && kb < minKB {
			fail("ram_total", "RAM", fmt.Sprintf("Low RAM: %d KB", kb), "Increase RAM before deploying.", map[string]string{"memtotal_kb": ramRes.Stdout})
		} else {
			pass("ram_total", "RAM", "RAM looks OK", map[string]string{"memtotal_kb": ramRes.Stdout})
		}
	}

	ok := true
	for _, c := range checks {
		if c.Status == PreflightFail {
			ok = false
			break
		}
	}

	return PreflightResponse{
		OK:        ok,
		Checks:    checks,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func parseOSRelease(contents string) (id, versionID string) {
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			versionID = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
		}
	}
	return strings.ToLower(id), versionID
}
