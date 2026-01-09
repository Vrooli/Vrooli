# CLI Commands

Command-line reference for vrooli-autoheal.

## Overview

The CLI provides direct access to autoheal functionality without needing the web dashboard.

```bash
vrooli-autoheal <command> [options]
```

## Commands

### tick

Run a single health check cycle.

```bash
vrooli-autoheal tick [--force] [--json]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--force` | Ignore interval restrictions, run all checks |
| `--json` | Output results as JSON |

**Examples:**
```bash
# Normal tick (respects intervals)
vrooli-autoheal tick

# Force all checks to run
vrooli-autoheal tick --force

# JSON output for scripting
vrooli-autoheal tick --json | jq '.summary'
```

**Output (default):**
```
Running health checks...
✓ infra-network: Network connectivity OK (15ms)
✓ infra-dns: DNS resolution OK (23ms)
✓ infra-docker: Docker daemon healthy (45ms)
⚠ infra-rdp: xrdp service not active (5ms)
✓ resource-postgres: PostgreSQL healthy (12ms)

Summary: 4 OK, 1 Warning, 0 Critical
```

**Output (JSON):**
```json
{
  "success": true,
  "summary": {
    "total": 5,
    "ok": 4,
    "warning": 1,
    "critical": 0
  },
  "results": [...]
}
```

---

### loop

Run continuous health monitoring.

```bash
vrooli-autoheal loop [--interval-seconds=N]
```

**Options:**
| Option | Default | Description |
|--------|---------|-------------|
| `--interval-seconds` | 60 | Seconds between tick cycles |

**Examples:**
```bash
# Default 60-second interval
vrooli-autoheal loop

# Custom 30-second interval
vrooli-autoheal loop --interval-seconds=30

# Run in background (Linux)
nohup vrooli-autoheal loop > /var/log/autoheal.log 2>&1 &
```

**Output:**
```
Starting autoheal loop (interval: 60s)
Press Ctrl+C to stop

[10:30:00] Tick completed: 5 OK, 0 Warning, 0 Critical
[10:31:00] Tick completed: 5 OK, 0 Warning, 0 Critical
[10:32:00] Tick completed: 4 OK, 1 Warning, 0 Critical
^C
Shutting down gracefully...
```

---

### status

Show current health status summary.

```bash
vrooli-autoheal status [--json]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--json` | Output as JSON |

**Examples:**
```bash
# Human-readable status
vrooli-autoheal status

# JSON for scripting
vrooli-autoheal status --json
```

**Output (default):**
```
Vrooli Autoheal Status
======================

Overall: OK ✓

Summary:
  Total Checks:  7
  Healthy:       6
  Warnings:      1
  Critical:      0

Platform: linux (systemd, docker)

Last Update: 2024-01-15 10:30:00
```

**Exit Codes:**
| Code | Meaning |
|------|---------|
| 0 | All checks OK |
| 1 | At least one warning |
| 2 | At least one critical |

---

### checks

List registered health checks.

```bash
vrooli-autoheal checks [--json]
```

**Output:**
```
Registered Health Checks
========================

infra-network
  Description: TCP connectivity to 8.8.8.8:53
  Interval: 60s
  Category: infrastructure

infra-dns
  Description: DNS resolution check
  Interval: 60s
  Category: infrastructure

infra-docker
  Description: Docker daemon health
  Interval: 120s
  Category: infrastructure
  Platforms: linux, windows, macos

...

Total: 7 checks registered
```

---

### platform

Show detected platform capabilities.

```bash
vrooli-autoheal platform [--json]
```

**Output:**
```
Platform Detection
==================

Platform: linux

Capabilities:
  ✓ systemd       Linux service manager
  ✓ docker        Container runtime
  ✓ cloudflared   Tunnel service
  ✗ rdp           Remote desktop (xrdp not found)
  ✗ launchd       macOS service manager
  ✗ windows-svc   Windows Service Control Manager

Environment:
  ✗ WSL           Windows Subsystem for Linux
  ✓ Headless      No display manager detected
```

---

### watchdog

Manage OS-level watchdog.

#### watchdog status

```bash
vrooli-autoheal watchdog status
```

**Output:**
```
Watchdog Status
===============

Type: systemd
Service: vrooli-autoheal
Status: active (running)
Since: 2024-01-14 08:00:00

Protection: ENABLED ✓
```

#### watchdog install

```bash
vrooli-autoheal watchdog install [--system]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--system` | Install system-wide (requires sudo) |

#### watchdog uninstall

```bash
vrooli-autoheal watchdog uninstall
```

---

### help

Show help information.

```bash
vrooli-autoheal help [command]
```

**Examples:**
```bash
# General help
vrooli-autoheal help

# Command-specific help
vrooli-autoheal help tick
vrooli-autoheal help watchdog
```

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `AUTOHEAL_API_PORT` | API port to connect to | Auto-detect |
| `AUTOHEAL_API_HOST` | API host | localhost |
| `VROOLI_LIFECYCLE_MANAGED` | Set by lifecycle system | - |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success / All OK |
| 1 | Warning status / Minor error |
| 2 | Critical status / Major error |
| 3 | Command error (invalid args, etc.) |

## Scripting Examples

### Check health in CI/CD

```bash
#!/bin/bash
vrooli-autoheal status --json | jq -e '.status == "ok"'
if [ $? -ne 0 ]; then
    echo "Health check failed!"
    exit 1
fi
```

### Monitor specific check

```bash
#!/bin/bash
while true; do
    status=$(vrooli-autoheal status --json | jq -r '.checks[] | select(.checkId == "infra-docker") | .status')
    if [ "$status" != "ok" ]; then
        echo "Docker health degraded: $status"
        # Send alert...
    fi
    sleep 60
done
```

### Force tick and wait for OK

```bash
#!/bin/bash
max_attempts=5
attempt=0

while [ $attempt -lt $max_attempts ]; do
    vrooli-autoheal tick --force --json > /tmp/tick.json

    critical=$(jq '.summary.critical' /tmp/tick.json)
    if [ "$critical" -eq 0 ]; then
        echo "All checks passing"
        exit 0
    fi

    echo "Attempt $attempt: $critical critical checks"
    ((attempt++))
    sleep 30
done

echo "Health checks still failing after $max_attempts attempts"
exit 1
```
