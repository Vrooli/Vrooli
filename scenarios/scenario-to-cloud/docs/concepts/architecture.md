# Architecture

System design and components of Scenario-to-Cloud.

## Overview

Scenario-to-Cloud is a deployment orchestrator that transfers Vrooli scenarios from a local development environment to a production VPS.

```
┌─────────────────────────────────────────────────────────────┐
│                    Local Machine                             │
│  ┌──────────────────┐     ┌──────────────────┐              │
│  │   Scenario-to-   │────▶│   Bundle         │              │
│  │   Cloud UI       │     │   Creator        │              │
│  └────────┬─────────┘     └────────┬─────────┘              │
│           │                        │                         │
│           ▼                        ▼                         │
│  ┌──────────────────┐     ┌──────────────────┐              │
│  │   Go API         │────▶│   Local Vrooli   │              │
│  │   Server         │     │   Scenarios      │              │
│  └────────┬─────────┘     └──────────────────┘              │
└───────────┼─────────────────────────────────────────────────┘
            │ SSH/SCP
            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Remote VPS                                │
│  ┌──────────────────┐     ┌──────────────────┐              │
│  │   Mini-Vrooli    │────▶│   Running        │              │
│  │   Installation   │     │   Scenario       │              │
│  └──────────────────┘     └────────┬─────────┘              │
│                                    │                         │
│  ┌──────────────────┐              ▼                         │
│  │   Caddy          │◀────────────────────────               │
│  │   (HTTPS)        │     Public Traffic                     │
│  └──────────────────┘                                        │
└─────────────────────────────────────────────────────────────┘
```

## Components

### UI (React + TypeScript)

Single-page application providing:
- Deployment wizard with step-by-step guidance
- Deployment management (list, inspect, stop, delete)
- Real-time status updates via Server-Sent Events (SSE)
- Documentation browser

### API (Go)

RESTful API server handling:
- Manifest validation and normalization
- Bundle creation and management
- VPS operations via SSH
- Deployment persistence (PostgreSQL)
- Documentation serving

### Bundle System

Creates minimal, self-contained packages:
- Core Vrooli scripts
- Scenario files
- Resource configurations
- No unnecessary files

### SSH/SCP Integration

Secure file transfer and remote execution:
- Primary: SSH key-based authentication
- Password authentication used only for initial key copying (via `ExecKeyCopier`)
- Idempotent operations
- Error recovery with structured classification

## Data Flow

### Deployment Flow

1. **Manifest Creation**: User configures deployment via UI
2. **Validation**: API validates manifest against schema
3. **Planning**: API generates execution plan
4. **Bundle**: Creates tarball of minimal Vrooli + scenario
5. **Transfer**: SCP sends bundle to VPS
6. **Setup**: SSH runs setup scripts on VPS
7. **Deploy**: SSH starts scenario services
8. **Verify**: Health checks confirm deployment

### Inspection Flow

1. **Request**: UI triggers inspect via API
2. **SSH**: API connects to VPS
3. **Check**: Runs `vrooli scenario status`
4. **Logs**: Retrieves recent log output
5. **Response**: Returns status to UI

## Database Schema

```sql
CREATE TABLE deployments (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    scenario_id TEXT NOT NULL,
    status TEXT NOT NULL,
    manifest JSONB NOT NULL,
    bundle_path TEXT,
    bundle_sha256 TEXT,
    setup_result JSONB,
    deploy_result JSONB,
    last_inspect_result JSONB,
    error_message TEXT,
    error_step TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    last_deployed_at TIMESTAMPTZ,
    last_inspected_at TIMESTAMPTZ
);
```

## Security Considerations

### SSH Keys

- Primary: SSH key-based authentication for all VPS operations
- Supports key path configuration
- Password authentication used only for initial key copying (via `ExecKeyCopier`)
- Key lifecycle: Discover -> Generate -> Copy -> Test -> Delete

### Input Validation

- All manifests validated before use
- Path traversal protection in docs API
- JSON size limits on requests

### Network

- HTTPS enforced via Caddy
- Automatic certificate management
- No sensitive data in logs

## Target Reach and the SSH Adapter

> [CODE: api/reach/reach.go]
> [CODE: api/reach/sshadapter/adapter.go]
> [CODE: api/reach/bridge/adapter.go]

The cloud touches a target only through `reach.Reach`: a typed command (`{verb, argv[]}` or a read-only observation program) dispatched on the transport the deployment's target binding names. There is no shell-string seam and no implicit transport; a revoked enrollment or an unconfigured transport is a typed refusal (`reach_unavailable`, `enrollment_revoked`), never a fallback.

```
┌──────────────────────────────────────────────────────────────┐
│                        reach.Router                           │
│   target_binding.transport ──▶ "bridge" │ "ssh"               │
│                                   │        │                  │
│              ┌────────────────────┘        └──────────────┐   │
│              ▼                                            ▼   │
│   reach/bridge.Adapter                       reach/sshadapter │
│   nodereach Dispatch/Wait/Call               .Adapter          │
│   (durable runs, signed relay)               Runner / SCPRunner│
│                                              (ExecRunner,      │
│                                               ExecSCPRunner;   │
│                                               FakeSSHRunner in │
│                                               tests)           │
└──────────────────────────────────────────────────────────────┘
```

### The bounded SSH adapter

`reach/sshadapter` is a policy-equivalent adapter, not a private connection plane:

- it accepts only validated argv (`reach.ValidateCommand`: no shell syntax, no empty or oversized arguments) and joins it with one quoting rule (`RemoteCommand`, `ObservationCommand`);
- a verb runs the bound workdir's `vrooli` binary; an observation program (`reach.ObservationPrograms`: `cat`, `df`, `du`, `find`, `grep`, `journalctl`, `ls`, `pgrep`, `ps`, `ss`, `stat`, `uname`, `head`) runs outside it and can never change host state;
- stdin never enters the command string (a credential value travels on `Command.Stdin`);
- failures are classified by exit code only: `255` is `target_offline`, `127` is `reach_protocol_unsupported`, a deadline is `transport_failure`, and every other non-zero exit is the target's own typed answer;
- artifacts are placed by `Deliver` (scp into the parent directory, remote sha256 verified against the local bytes, mode applied);
- the interactive terminal is `OpenSession` (a PTY over `golang.org/x/crypto/ssh`) behind the `reach.SessionOpener` seam.

### Connection resolution and key custody

`ConnectionConfig` is resolved per target by the server (`sshConfigForTarget`): the locator from the target binding, the key file from the credential binding `vrooli/scenario-to-cloud:ssh-key` (class `machine_enrollment_credential`, file target = operator-held path, no bytes). A target without a binding is reached with the operator's ambient identity. Host keys are trusted on first use into the scenario's `known_hosts` through `packages/ssh-core`. The manifest carries no key: a legacy row's `target.vps.key_path` is converted into the binding at API start (`persistence.convertLegacyKeyPathBindings`).

### Preflight on a bare host

Preflight runs before the target owner exists, so every host fact is an observation program through reach (`vps/preflight/observe.go`): OS release and firewall state from their files, sockets from `ss`, disk and memory from `df`/`grep`, tools from one `find`, stale processes from `pgrep`. Outbound reachability is not observable through the read-only set and is reported as `warn`; `host.prepare` surfaces a blocked egress at apply time.

### Live State Parallel Inspection

The live state collector runs 15 SSH commands in parallel to gather comprehensive VPS state:

```
                    GetLiveState(cfg, manifest)
                              │
                 ┌────────────┼────────────────────┐
                 │            │                     │
    ┌────────────┼────────────┼─────────────┐       │
    │            │            │             │       │
    ▼            ▼            ▼             ▼       ▼
┌───────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ ┌───────────┐
│ ps    │ │ ss/      │ │ df -h    │ │ free -m │ │ loadavg   │
│ aux   │ │ netstat  │ │          │ │         │ │           │
└───────┘ └──────────┘ └──────────┘ └─────────┘ └───────────┘

┌───────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ ┌───────────┐
│uptime │ │ cpuinfo  │ │cpumodel  │ │cpuusage │ │ scenario  │
│       │ │ (cores)  │ │          │ │ (sample)│ │ status    │
└───────┘ └──────────┘ └──────────┘ └─────────┘ └───────────┘

┌──────────┐ ┌──────────┐ ┌────────────┐ ┌──────────┐ ┌───────────┐
│ resource │ │ caddy    │ │caddy       │ │ssh key   │ │ dir       │
│ status   │ │ config   │ │running?    │ │check     │ │ check     │
└──────────┘ └──────────┘ └────────────┘ └──────────┘ └───────────┘
                              │
                              ▼
                    sync.WaitGroup.Wait()
                              │
                              ▼
                    Parse + assemble VPSLiveState
```

Every command is a `reach.Command` through the bound transport (`vps.Prober`), so tests script answers by argv with `reachtest.Scripted` and never open a connection.

## Edge/TLS Management

> [CODE: api/handlers_edge.go]
> [CODE: api/dns/service.go]
> [CODE: api/tlsinfo/service.go]

The Edge subsystem manages the public-facing layer of a deployment: DNS verification, Caddy control, and TLS certificate lifecycle.

- **DNS Check** (`/edge/dns-check`): Resolves all deployment domains (apex, www, origin) and verifies they point to the VPS IP. Reports per-domain status with hints for misconfiguration.
- **DNS Records** (`/edge/dns-records`): Raw DNS record lookup (A, AAAA, CNAME, MX, TXT) for debugging.
- **Caddy Control** (`/edge/caddy`): Start/stop/restart/reload the Caddy web server on the VPS via SSH.
- **TLS Info** (`/edge/tls`): Probes the deployment domain for certificate details (issuer, validity, SANs, ALPN protocols) using the `tlsinfo.Service` seam.
- **TLS Renewal** (`/edge/tls/renew`): Forces Caddy to renew certificates on the VPS.

## VPS Secrets Management

> [CODE: api/secrets/handlers_management.go]

Post-deployment CRUD for secrets stored on the VPS `.env` file. Supports listing (masked by default), creating, updating, and deleting individual secrets. Optionally restarts the scenario after mutation to pick up new values.

A separate **Expected Secrets** endpoint (`/expected-secrets`) returns the secrets defined in the scenario's `service.json`, enabling the UI to show which secrets are required vs. present.

## Terminal

> [CODE: api/handlers_deployment.go]

The Terminal endpoint (`GET /deployments/{id}/terminal`) upgrades to a WebSocket connection, then opens an interactive SSH session to the deployment's VPS. The server proxies stdin/stdout/stderr bidirectionally, enabling a browser-based terminal.

## Investigation & Tasks

> [CODE: api/investigation/service.go]
> [CODE: api/tasks/service.go]
> [CODE: api/handlers_tasks.go]

The Investigation subsystem integrates with the **agent-manager** scenario to run autonomous debugging sessions against deployments. Legacy investigation endpoints are preserved for backward compatibility, while the new unified **Tasks** API (`/deployments/{id}/tasks`) provides a single interface for both investigate and fix task types.

Tasks support configurable focus (harness/subject), effort levels (logs/inspect/code), and permission scopes (immediate/permanent/prevention).

## SSE Progress Streaming

> [CODE: api/deployment/progress.go]

Long-running operations (execute, start, stop) report real-time progress via **Server-Sent Events** on `GET /deployments/{id}/progress`. The `deployment.Hub` fans out progress events to all connected SSE clients for a given deployment.

Each SSE event carries:
- `step` — Current pipeline step ID
- `status` — Step status (`running`, `complete`, `failed`)
- `percent` — Overall progress percentage
- `message` — Human-readable status message
- `error_category`, `retryable`, `hint` — Structured error metadata (when applicable)

## Canonical SSH Identity Lifecycle

> [CODE: api/sshidentity/model.go]
> [CODE: api/sshidentity/resolve.go]
> [CODE: api/sshidentity/keys.go]
> [CODE: api/deployment/orchestrator.go]
> [CODE: api/handlers_health.go]

Scenario-to-cloud now uses one canonical identity model persisted per deployment (`deployments.ssh_identity`) and shared across setup/deploy/live-state/health.

1. Resolver determines identity precedence: manifest explicit key, persisted explicit key, ambient transport (`agent`/`default_ssh`).
2. Orchestrator persists resolved identity before remote steps execute.
3. Live-state and health use the same identity when building SSH config and key authorization checks.
4. Post-deploy verification updates `verification_state` and `last_verified_at` on the same model.

This removes split logic where deploy transport could succeed while health had no coherent identity context.
