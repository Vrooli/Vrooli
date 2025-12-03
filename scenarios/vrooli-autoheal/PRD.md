# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Self-healing supervisor that bootstraps the Vrooli environment, installs OS-level watchdogs, and continuously monitors/repairs critical infrastructure across platforms
- **Primary users/verticals**: DevOps engineers, system administrators, Vrooli operators, automated agents
- **Deployment surfaces**: CLI (`vrooli autoheal tick/loop/status`), API (health registry), UI (dashboard showing health status and history)
- **Value promise**: Ensures Vrooli infrastructure survives reboots, crashes, and failures without manual intervention. Centralizes all health logic in one place with a clean registry of checks.

### Why It Matters

1. **Survival across reboots & crashes**: Brings Vrooli back to a known-good state after OS reboots, Docker restarts, or process crashes without manual intervention
2. **Keeps ops surface reachable**: Ensures remote access (RDP/xrdp), core monitors, and management scenarios stay online so operators can always reach and control the system
3. **Centralizes health logic**: All "is the system healthy?" logic lives in one place with a clean registry of checks instead of scattered scripts and cron jobs
4. **Cross-platform with platform-specific smarts**: Same high-level behavior on Linux/Windows/macOS while doing the right OS-specific things (systemd, Windows services, launchd, RDP variants)
5. **Foundation for higher-level automation**: Other scenarios and agents can rely on vrooli-autoheal as a trusted service that keeps critical infrastructure running

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | CLI tick command | Single-shot bootstrap + health cycle via `vrooli autoheal tick`
- [ ] OT-P0-002 | CLI loop command | Long-running mode with configurable interval via `vrooli autoheal loop`
- [ ] OT-P0-003 | Platform detection | Detect platform (linux/windows/macos/other) and capabilities (supportsRdp, supportsSystemd, etc.)
- [ ] OT-P0-004 | Health check registry | Extensible registry pattern for registering/running health checks with intervals and platform filters
- [ ] OT-P0-005 | Core bootstrap | Bootstrap DB, core resources, and critical scenarios from cold state
- [ ] OT-P0-006 | Resource health checks | Monitor configured resources (postgres, redis, qdrant, ollama) with auto-restart on failure
- [ ] OT-P0-007 | Scenario health checks | Monitor configured scenarios with auto-restart on failure
- [ ] OT-P0-008 | OS watchdog installer | Idempotently install/verify systemd/launchd/Windows service that keeps autoheal loop running
- [ ] OT-P0-009 | Health result persistence | Store health check results with timestamps for status queries and UI display
- [ ] OT-P0-010 | CLI status command | Show last-known health summary via `vrooli autoheal status`

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Infrastructure checks | Network connectivity, DNS resolution, time synchronization checks
- [ ] OT-P1-002 | System resource checks | Disk space, swap usage, zombie processes, port exhaustion monitoring
- [ ] OT-P1-003 | RDP/remote access health | Platform-specific RDP/xrdp/TermService monitoring with auto-restart
- [ ] OT-P1-004 | Docker daemon health | Monitor Docker service and restart if unresponsive
- [ ] OT-P1-005 | Cloudflared tunnel health | Monitor cloudflared service and tunnel connectivity
- [ ] OT-P1-006 | Health history window | Store recent health check history (24h) for dashboards and trend analysis
- [ ] OT-P1-007 | Web UI dashboard | React dashboard showing current health status, recent events, and auto-heal actions
- [ ] OT-P1-008 | Configurable check intervals | Per-check interval configuration with smart scheduling (only run when interval elapsed)
- [ ] OT-P1-009 | Graceful shutdown | Handle SIGINT/SIGTERM cleanly in loop mode

### 🟢 P2 – Future / expansion ideas

- [ ] OT-P2-001 | Certificate expiration monitoring | Check SSL certificates and warn before expiration
- [ ] OT-P2-002 | Display manager health | GDM/lightdm/sddm monitoring for Linux desktops
- [ ] OT-P2-003 | Webhook notifications | Send alerts to Slack/Discord/email on critical failures
- [ ] OT-P2-004 | Custom check plugins | Allow external check definitions via config files
- [ ] OT-P2-005 | AI-powered root cause analysis | Use Ollama to analyze failure patterns and suggest fixes
- [ ] OT-P2-006 | Mobile status app | Push notifications and status monitoring on mobile devices

## 🧱 Tech Direction Snapshot

- **Preferred stacks/frameworks**: Go API (cross-platform, low overhead), React + Vite UI (TypeScript), Bash CLI wrapper calling Go binary
- **Data + storage expectations**: PostgreSQL for health history and configuration, in-memory cache for current state
- **Integration strategy**:
  - Uses `vrooli resource status/start/stop` CLI for resource management
  - Uses `vrooli scenario status/start/stop` CLI for scenario management
  - Direct API calls only where CLI is insufficient (e.g., health endpoint checks)
- **Non-goals / guardrails**:
  - Will NOT replace existing monitoring tools (system-monitor handles metrics/anomalies)
  - Will NOT implement alerting/paging (use dedicated alerting scenarios)
  - Will NOT manage non-Vrooli services beyond OS-level watchdog

## 🤝 Dependencies & Launch Plan

- **Required resources**:
  - `postgres` - Health check history and configuration storage
- **Optional resources**:
  - `redis` - Real-time status pub/sub (fallback: polling)
  - `ollama` - AI analysis for P2 features
- **Scenario dependencies**:
  - None (autoheal is foundational, other scenarios depend on it)
- **Operational risks**:
  - OS-level watchdog installation requires elevated privileges on some platforms
  - Platform detection edge cases on WSL/Docker containers
  - Circular dependency risk (autoheal monitors scenarios that might monitor autoheal)
- **Launch sequencing**:
  1. Core CLI (tick/loop/status) with platform detection
  2. Health check registry with resource/scenario checks
  3. OS watchdog installers (Linux first, then Windows/macOS)
  4. Web UI dashboard
  5. P1 infrastructure/system checks

## 🎨 UX & Branding

- **Look & feel**: Dark theme with status-driven colors (green=healthy, amber=warning, red=critical), clean dashboard layout
- **Accessibility**: WCAG AA compliance, high contrast mode, screen reader support for health status
- **Voice & messaging**: Technical but reassuring - "Your infrastructure is protected"
- **Branding hooks**: Shield/heartbeat iconography, Vrooli brand colors as accents

## 📎 Appendix

### CLI Command Reference

```bash
# Single health cycle (bootstrap + all checks + watchdog verify)
vrooli autoheal tick

# Long-running mode (default 60s interval)
vrooli autoheal loop [--interval-seconds=60]

# Show last-known health summary
vrooli autoheal status [--json]
```

### Platform Capabilities Model

```typescript
type Platform = "linux" | "windows" | "macos" | "other";

interface PlatformCapabilities {
  supportsRdp: boolean;
  supportsSystemd: boolean;
  supportsLaunchd: boolean;
  supportsWindowsServices: boolean;
  isHeadlessServer: boolean;
  hasDocker: boolean;
}
```

### Health Check Interface

```typescript
interface HealthCheck {
  id: string;
  description: string;
  intervalSeconds: number;
  platforms?: Platform[];  // omit = all platforms
  run(ctx: HealthCheckContext): Promise<HealthResult>;
}

interface HealthResult {
  status: "ok" | "warning" | "critical";
  message: string;
  details?: Record<string, unknown>;
}
```

### Related Documentation

- Existing autoheal scripts: `scripts/maintenance/autoheal/`
- Maintenance orchestrator: `scenarios/maintenance-orchestrator/PRD.md`
- System monitor: `scenarios/system-monitor/PRD.md`
