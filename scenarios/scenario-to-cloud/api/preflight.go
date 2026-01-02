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
	ID      string               `json:"id"`
	Title   string               `json:"title"`
	Status  PreflightCheckStatus `json:"status"`
	Details string               `json:"details,omitempty"`
	Hint    string               `json:"hint,omitempty"`
	Data    map[string]string    `json:"data,omitempty"`
}

type PreflightResponse struct {
	OK        bool              `json:"ok"`
	Checks    []PreflightCheck  `json:"checks"`
	Issues    []ValidationIssue `json:"issues,omitempty"`
	Timestamp string            `json:"timestamp"`
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
		// Build helpful DNS provider instructions
		domain := manifest.Edge.Domain
		targetIP := vpsIPs[0] // Use first IP for instructions
		hint := fmt.Sprintf(
			"Update DNS A record for %s to point to %s.\n\n"+
				"Common DNS providers:\n"+
				"- Cloudflare: DNS > Records > Add A record with name '@' or subdomain, IPv4 address %s\n"+
				"- Namecheap: Domain List > Manage > Advanced DNS > Add A Record\n"+
				"- GoDaddy: DNS Management > Add > Type: A, Points to: %s\n"+
				"- DigitalOcean: Networking > Domains > Add A record\n\n"+
				"DNS changes can take up to 48 hours to propagate (usually 5-30 minutes).",
			domain, targetIP, targetIP, targetIP,
		)
		fail(
			"dns_points_to_vps",
			"DNS points to VPS",
			fmt.Sprintf("%s resolves to %s, not your VPS (%s)", domain, strings.Join(edgeIPs, ", "), strings.Join(vpsIPs, ", ")),
			hint,
			map[string]string{"vps_ips": strings.Join(vpsIPs, ","), "edge_ips": strings.Join(edgeIPs, ","), "domain": domain},
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
			"Ensure the VPS is running Ubuntu and that /etc/os-release is readable.",
			map[string]string{"stderr": osRes.Stderr},
		)
	} else {
		id, ver := parseOSRelease(osRes.Stdout)
		if id != "ubuntu" {
			fail(
				"os_release",
				"Ubuntu version",
				fmt.Sprintf("Unsupported OS: %s", id),
				"scenario-to-cloud requires Ubuntu. Debian may work but is untested.",
				map[string]string{"id": id, "version_id": ver},
			)
		} else if ver == "24.04" {
			pass("os_release", "Ubuntu version", "Ubuntu 24.04 detected", map[string]string{"id": id, "version_id": ver})
		} else if ver == "22.04" || ver == "20.04" {
			// Older LTS versions should work but aren't officially tested
			warn(
				"os_release",
				"Ubuntu version",
				fmt.Sprintf("Ubuntu %s detected (24.04 recommended)", ver),
				"Ubuntu 22.04/20.04 should work but 24.04 LTS is recommended for best compatibility.",
				map[string]string{"id": id, "version_id": ver},
			)
		} else {
			warn(
				"os_release",
				"Ubuntu version",
				fmt.Sprintf("Ubuntu %s detected (24.04 recommended)", ver),
				"This Ubuntu version is untested. Consider using Ubuntu 24.04 LTS.",
				map[string]string{"id": id, "version_id": ver},
			)
		}
	}

	portsRes, portsErr := sshRunner.Run(ctx, cfg, `ss -ltnpH '( sport = :80 or sport = :443 )' 2>/dev/null || ss -ltnH '( sport = :80 or sport = :443 )'`)
	if portsErr != nil {
		warn(
			"ports_80_443",
			"Ports 80/443 availability",
			"Unable to check ports 80/443 via ss",
			"Ensure ports 80 and 443 are free for Caddy/Let's Encrypt HTTP-01.",
			map[string]string{"stderr": portsRes.Stderr},
		)
	} else if strings.TrimSpace(portsRes.Stdout) != "" {
		// Parse the ss output to identify what's running
		processes := parsePortProcesses(portsRes.Stdout)
		hint := "Stop the service(s) using these ports before deploying."
		if len(processes) > 0 {
			hint = fmt.Sprintf("Services using ports: %s. Run: sudo systemctl stop <service> or sudo kill <pid>", strings.Join(processes, ", "))
		}
		fail(
			"ports_80_443",
			"Ports 80/443 availability",
			"Port 80 and/or 443 appears to already be in use",
			hint,
			map[string]string{"ss": portsRes.Stdout, "processes": strings.Join(processes, ",")},
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
		warn("disk_free", "Disk free space", "Unable to determine free disk space", "Ensure the VPS has sufficient free disk for builds and resources.", map[string]string{"stderr": diskRes.Stderr})
	} else {
		kb, _ := strconv.ParseInt(strings.TrimSpace(diskRes.Stdout), 10, 64)
		const minKB = 5 * 1024 * 1024 // 5 GiB
		if kb > 0 && kb < minKB {
			fail(
				"disk_free",
				"Disk free space",
				fmt.Sprintf("Low free disk space: %s", formatBytes(kb)),
				"At least 5 GB free space is recommended. Run: sudo apt clean && sudo journalctl --vacuum-size=100M",
				map[string]string{"free_kb": diskRes.Stdout, "free_human": formatBytes(kb)},
			)
		} else {
			pass("disk_free", "Disk free space", fmt.Sprintf("Free space: %s", formatBytes(kb)), map[string]string{"free_kb": diskRes.Stdout, "free_human": formatBytes(kb)})
		}
	}

	ramRes, ramErr := sshRunner.Run(ctx, cfg, `awk '/MemTotal/ {print $2}' /proc/meminfo`)
	if ramErr != nil || ramRes.ExitCode != 0 {
		warn("ram_total", "RAM", "Unable to determine total RAM", "Ensure the VPS has sufficient RAM for the scenario and resources.", map[string]string{"stderr": ramRes.Stderr})
	} else {
		kb, _ := strconv.ParseInt(strings.TrimSpace(ramRes.Stdout), 10, 64)
		const minKB = 1024 * 1024      // 1 GiB
		const warnKB = 2 * 1024 * 1024 // 2 GiB
		if kb > 0 && kb < minKB {
			fail(
				"ram_total",
				"RAM",
				fmt.Sprintf("Low RAM: %s", formatBytes(kb)),
				"At least 1 GB RAM is required. 2+ GB is recommended for most scenarios.",
				map[string]string{"memtotal_kb": ramRes.Stdout, "memtotal_human": formatBytes(kb)},
			)
		} else if kb > 0 && kb < warnKB {
			warn(
				"ram_total",
				"RAM",
				fmt.Sprintf("RAM: %s (2+ GB recommended)", formatBytes(kb)),
				"Your VPS has limited RAM. Consider upgrading for better performance.",
				map[string]string{"memtotal_kb": ramRes.Stdout, "memtotal_human": formatBytes(kb)},
			)
		} else {
			pass("ram_total", "RAM", fmt.Sprintf("RAM: %s", formatBytes(kb)), map[string]string{"memtotal_kb": ramRes.Stdout, "memtotal_human": formatBytes(kb)})
		}
	}

	// Check: required system commands (bootstrap will install if missing)
	requiredCmds := []struct {
		name string
		id   string
	}{
		{"curl", "cmd_curl"},
		{"git", "cmd_git"},
		{"unzip", "cmd_unzip"},
		{"tar", "cmd_tar"},
	}
	for _, cmd := range requiredCmds {
		res, err := sshRunner.Run(ctx, cfg, "which "+cmd.name)
		if err != nil || res.ExitCode != 0 {
			warn(cmd.id, cmd.name+" available",
				cmd.name+" not found on VPS",
				"Bootstrap phase will install this automatically",
				nil)
		} else {
			pass(cmd.id, cmd.name+" available",
				"Found at "+strings.TrimSpace(res.Stdout), nil)
		}
	}

	// Check: jq (nice to have, warn only)
	jqRes, jqErr := sshRunner.Run(ctx, cfg, "which jq")
	if jqErr != nil || jqRes.ExitCode != 0 {
		warn("cmd_jq", "jq available",
			"jq not found on VPS",
			"Bootstrap phase will install this automatically",
			nil)
	} else {
		pass("cmd_jq", "jq available",
			"Found at "+strings.TrimSpace(jqRes.Stdout), nil)
	}

	// Check: apt access (required for bootstrap to work)
	aptRes, aptErr := sshRunner.Run(ctx, cfg, "apt-get update --print-uris 2>&1 | head -1")
	if aptErr != nil {
		fail("apt_access", "apt accessible",
			"Unable to run apt-get",
			"Bootstrap requires apt access. Ensure the user has sudo/root privileges.",
			map[string]string{"error": aptErr.Error()})
	} else if strings.Contains(aptRes.Stderr, "Permission denied") || strings.Contains(aptRes.Stdout, "Permission denied") {
		fail("apt_access", "apt accessible",
			"apt-get permission denied",
			"Bootstrap requires apt access. Ensure the user has sudo/root privileges.",
			nil)
	} else {
		pass("apt_access", "apt accessible", "apt-get is accessible", nil)
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

// parsePortProcesses extracts process names from ss -ltnp output
// Example line: LISTEN 0 4096 *:80 *:* users:(("nginx",pid=1234,fd=6))
func parsePortProcesses(ssOutput string) []string {
	var processes []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(ssOutput, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for users:(("name",pid=N,fd=N)) pattern
		if idx := strings.Index(line, "users:(("); idx != -1 {
			rest := line[idx+8:] // Skip "users:(("
			if endIdx := strings.Index(rest, `"`); endIdx > 0 {
				// Find the process name between quotes
				if startQuote := strings.Index(rest, `"`); startQuote == 0 {
					nameEnd := strings.Index(rest[1:], `"`)
					if nameEnd > 0 {
						name := rest[1 : nameEnd+1]
						if !seen[name] {
							seen[name] = true
							// Also try to get pid
							if pidIdx := strings.Index(rest, "pid="); pidIdx != -1 {
								pidEnd := strings.IndexAny(rest[pidIdx+4:], ",)")
								if pidEnd > 0 {
									pid := rest[pidIdx+4 : pidIdx+4+pidEnd]
									processes = append(processes, fmt.Sprintf("%s (pid %s)", name, pid))
								} else {
									processes = append(processes, name)
								}
							} else {
								processes = append(processes, name)
							}
						}
					}
				}
			}
		} else {
			// No process info available, just note the port
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				localAddr := fields[3]
				if strings.HasSuffix(localAddr, ":80") && !seen["port80"] {
					seen["port80"] = true
					processes = append(processes, "unknown on :80")
				}
				if strings.HasSuffix(localAddr, ":443") && !seen["port443"] {
					seen["port443"] = true
					processes = append(processes, "unknown on :443")
				}
			}
		}
	}

	return processes
}

// formatBytes converts bytes to human-readable format
func formatBytes(kb int64) string {
	if kb >= 1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(kb)/(1024*1024))
	} else if kb >= 1024 {
		return fmt.Sprintf("%.1f MB", float64(kb)/1024)
	}
	return fmt.Sprintf("%d KB", kb)
}
