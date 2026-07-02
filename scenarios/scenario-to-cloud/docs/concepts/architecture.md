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

## SSH Subsystem

The SSH subsystem (`ssh/` package) is the primary integration layer between the local API and remote VPS hosts. It provides four core interfaces, a structured error pipeline, key management, and parallel command execution.

### SSH Interface Seams

The SSH package exposes four interfaces that serve as testability seams:

```
┌─────────────────────────────────────────────────────────┐
│                     ssh/ package                         │
│                                                          │
│  ┌─────────────────┐    ┌──────────────────┐            │
│  │   Runner         │    │   SCPRunner       │            │
│  │   (runner.go)    │    │   (runner.go)     │            │
│  │                  │    │                   │            │
│  │  Run(ctx, cfg,   │    │  Copy(ctx, cfg,   │            │
│  │    cmd, opts)    │    │    local, remote,  │            │
│  │  -> Result, err  │    │    opts) -> err    │            │
│  └────────┬─────────┘    └────────┬──────────┘            │
│           │                       │                       │
│  Impls:   │ ExecRunner            │ ExecSCPRunner         │
│  Fakes:   │ FakeSSHRunner         │ FakeSCPRunner         │
│           │                       │                       │
│  ┌────────┴─────────┐    ┌───────┴──────────┐            │
│  │  CommandRunner    │    │   KeyCopier       │            │
│  │  (command.go)     │    │   (keys_copy.go)  │            │
│  │                   │    │                   │            │
│  │  Run(ctx, name,   │    │  CopyKey(ctx,     │            │
│  │    args...)       │    │    req) -> resp    │            │
│  │  -> out, err      │    │                   │            │
│  └───────────────────┘    └───────────────────┘            │
│                                                          │
│  Impls: ExecCommandRunner     ExecKeyCopier              │
│  Used by: KeyService          HandleCopyKey handler      │
└─────────────────────────────────────────────────────────┘
```

- **`Runner`** -- Executes SSH commands on remote hosts. Used by VPS setup, deploy, inspect, stop, and live state operations.
- **`SCPRunner`** -- Transfers files to remote hosts via SCP. Used for bundle uploads.
- **`CommandRunner`** -- Executes local commands (e.g., `ssh-keygen`). Used by `KeyService` for key generation and fingerprinting.
- **`KeyCopier`** -- Copies SSH public keys to remote servers via password authentication. Used by the `HandleCopyKey` HTTP handler.

### Deployment Pipeline Flow

The deployment pipeline executes a series of SSH operations in sequence, with per-step timeouts and error classification:

```
Manifest ──▶ Preflight ──▶ Bundle ──▶ Transfer ──▶ Setup ──▶ Deploy ──▶ Verify
                │                       │            │          │          │
                │ SSH: connectivity,     │ SCP:       │ SSH:     │ SSH:     │ SSH:
                │ disk, DNS, ports       │ bundle     │ extract, │ caddy,   │ health
                │                       │ upload     │ scripts  │ start    │ checks
                ▼                       ▼            ▼          ▼          ▼
           preflight.Run          SCPRunner.Copy   Runner.Run  Runner.Run  Runner.Run
                                                     │
                                                     ▼
                                              StepConfig per step:
                                              - CommandTimeout
                                              - MaxRetries
                                              - RetryDelay
```

Each step reports progress via SSE through the `deployment.Hub`. On failure, errors flow through `ClassifyError` to produce structured `ErrorInfo` with category, hint, and retryability.

### Error Classification Pipeline

SSH errors are classified into sentinel categories that determine retryability and provide actionable hints:

```
SSH command fails
       │
       ▼
  exitCode(err) ──▶ exit code extracted from exec.ExitError
       │
       ▼
  ClassifyError(stderr, host, defaultHint)
       │
       ├─── matches "Permission denied"?     ──▶ SSHError{Category: ErrAuth}
       ├─── matches "host key"?               ──▶ SSHError{Category: ErrHostKey}
       ├─── matches "timed out"?              ──▶ SSHError{Category: ErrTimeout, Retryable: true}
       ├─── matches "Connection refused"?     ──▶ SSHError{Category: ErrUnreachable, Retryable: true}
       ├─── matches "No space left"?          ──▶ SSHError{Category: ErrDiskSpace}
       ├─── matches "resolve hostname"?       ──▶ SSHError{Category: ErrDNS}
       ├─── matches "invalid format"?         ──▶ SSHError{Category: ErrKeyFormat}
       ├─── IsIPv6(host) + network error?     ──▶ SSHError{Category: ErrIPv6, Retryable: true}
       └─── fallback                          ──▶ SSHError{Category: ErrCommand}
       │
       ▼
  ErrorInfoFromSSHError(*SSHError)
       │
       ▼
  domain.ErrorInfo{Category, Hint, Retryable, ExitCode}
       │
       ├──▶ VPSSetupResult.ErrorInfo / VPSDeployResult.ErrorInfo  (JSON API)
       └──▶ deployment.Event.ErrorCategory / .Retryable / .Hint   (SSE)
```

Callers use `errors.Is(err, ssh.ErrTimeout)` for category matching and `sshErr.Retryable` for retry decisions.

### Key Management Lifecycle

The `KeyService` manages SSH keys through a complete lifecycle:

```
┌──────────────┐     ┌───────────────┐     ┌──────────────┐
│   Discover   │────▶│   Generate    │────▶│     Copy     │
│              │     │               │     │              │
│ DiscoverKeys │     │ GenerateKey   │     │ ExecKeyCopier│
│ (keys.go)    │     │ (keys_gen.go) │     │ (keys_copy)  │
│              │     │               │     │              │
│ Scans ~/.ssh │     │ ssh-keygen    │     │ golang.org/x │
│ for key files│     │ via           │     │ /crypto/ssh  │
│              │     │ CommandRunner │     │ password auth│
└──────────────┘     └───────────────┘     └──────┬───────┘
                                                   │
                     ┌───────────────┐     ┌───────┴──────┐
                     │    Delete     │     │     Test     │
                     │               │◀────│              │
                     │ DeleteKey     │     │ TestConnection│
                     │ (keys.go)     │     │ (connect.go) │
                     │               │     │              │
                     │ Removes key   │     │ Runner.Run   │
                     │ pair from disk│     │ "echo ok"    │
                     └───────────────┘     └──────────────┘
```

1. **Discover**: Scans `~/.ssh/` for existing key pairs, parses type and fingerprint
2. **Generate**: Creates new key via `ssh-keygen` (uses `CommandRunner` seam)
3. **Copy**: Transfers public key to remote `authorized_keys` via password auth (uses `KeyCopier` seam)
4. **Test**: Verifies key-based SSH connectivity with `echo ok` command
5. **Delete**: Removes key pair files from disk

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

All 15 commands execute through the `ssh.Runner` seam, enabling complete test control without SSH connections.

## Edge/TLS Management

> [CODE: api/handlers_edge.go, api/dns/, api/tlsinfo/]

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

> [CODE: api/handlers_deployment.go (WebSocket handler)]

The Terminal endpoint (`GET /deployments/{id}/terminal`) upgrades to a WebSocket connection, then opens an interactive SSH session to the deployment's VPS. The server proxies stdin/stdout/stderr bidirectionally, enabling a browser-based terminal.

## Investigation & Tasks

> [CODE: api/investigation/, api/tasks/, api/handlers_tasks.go]

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
