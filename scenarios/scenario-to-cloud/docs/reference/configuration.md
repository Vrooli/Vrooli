# SSH Configuration Reference

> [CODE: api/ssh/options.go] — RunOptions, SCPOptions, HandlerOptions structs and defaults
> [CODE: api/ssh/errors.go] — Error category sentinels and ClassifyError
> [CODE: api/vps/step_config.go] — StepConfig and per-step defaults

All SSH operations in scenario-to-cloud are configured through typed options structs. This document describes every tunable lever, its default value, and when to adjust it.

## RunOptions

Controls SSH command execution (`Runner.Run`).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ConnectTimeout` | `time.Duration` | `5s` | TCP connection timeout. Defaults to 5s when zero/unset. |
| `ServerAliveInterval` | `time.Duration` | `5s` | Keepalive probe interval. Set to 0 to disable keepalives. |
| `ServerAliveCountMax` | `int` | `1` | Failed probes before disconnect. With 5s interval and max 1, dead connections detected in ~10s. |
| `StrictHostKey` | `bool` | `true` | When true, uses `StrictHostKeyChecking=accept-new` (TOFU model). When false, the SSH option is omitted entirely, falling through to the system default. |
| `IdentitiesOnly` | `bool` | `false` | When true, only uses the specified key file (ignores ssh-agent). Used for connection testing. |
| `MaxOutputBytes` | `int` | `524288` (512KB) | Maximum stdout/stderr captured per command. Output beyond this is truncated with a warning. |
| `ErrorContextLines` | `int` | `50` | Number of trailing output lines included in error messages. |
| `CommandTimeout` | `time.Duration` | `0` (inherit from ctx) | Per-command timeout. When 0, defers to the context deadline. |

### Presets

- **`DefaultRunOptions()`** -- Standard production settings. Used by all VPS operations (deploy, setup, inspect, stop).
- **`TestConnectionOptions()`** -- 10s connect timeout, `IdentitiesOnly=true`. Used by the SSH connection test endpoint.

## SCPOptions

Controls SCP file transfers (`SCPRunner.Copy`).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ConnectTimeout` | `time.Duration` | `5s` | TCP connection timeout for SCP. |
| `StrictHostKey` | `bool` | `true` | Host key checking (same behavior as RunOptions). |
| `TransferTimeout` | `time.Duration` | `10m` | Overall transfer deadline. Bundles can be 50-200MB. |
| `MaxOutputBytes` | `int` | `524288` (512KB) | Max stderr captured on SCP failure. |

### Preset

- **`DefaultSCPOptions()`** -- Standard production settings.

## HandlerOptions

Controls HTTP handler-level timeouts for SSH endpoints.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `TestConnectionTimeout` | `time.Duration` | `30s` | Context deadline for `POST /ssh/test`. |
| `CopyKeyTimeout` | `time.Duration` | `30s` | Context deadline for `POST /ssh/copy-key`. |

### Preset

- **`DefaultHandlerOptions()`** -- Standard production settings.

## Intentionally Not Configurable

| Setting | Value | Rationale |
|---------|-------|-----------|
| `BatchMode` | Always `yes` | All operations are non-interactive. Prompts would hang. |
| Shell wrapper | Always `bash -lc` | Remote PATH setup requires login shell for `vrooli` CLI discovery. |
| `StrictHostKeyChecking` mode | Always `accept-new` (when enabled) | TOFU model: accept on first connect, reject on change. |
| Output format | Bounded buffer with truncation marker | Prevents unbounded memory from chatty commands. |

## Error Categories

SSH errors are classified into sentinel categories that determine retryability:

| Category | Retryable | Status Constant | Typical Cause |
|----------|-----------|-----------------|---------------|
| `ErrAuth` | No | `auth_failed` | Wrong key, permission denied |
| `ErrHostKey` | No | `host_key_changed` | Host key mismatch (MITM or reinstall) |
| `ErrTimeout` | Yes | `timeout` | Network latency, firewall dropping packets |
| `ErrUnreachable` | Yes | `host_unreachable` | Connection refused, no route (IPv4) |
| `ErrIPv6` | Yes | `ipv6_unavailable` | IPv6 connectivity issues |
| `ErrCommand` | No | `error` | Remote command failed (non-zero exit) |
| `ErrDiskSpace` | No | `disk_full` | Server ran out of disk space |
| `ErrDNS` | No | `dns_failed` | DNS resolution failed for hostname |
| `ErrKeyFormat` | No | `key_error` | SSH key file corrupted or wrong permissions |

Use `errors.Is(err, ssh.ErrTimeout)` to branch on category. Use `sshErr.Retryable` to decide whether to retry.

## Step Configuration

Each VPS deployment step can have per-step execution parameters via `StepConfig`:

| Field | Type | Description |
|-------|------|-------------|
| `CommandTimeout` | `time.Duration` | Per-step SSH command timeout. Overrides RunOptions default. |
| `MaxRetries` | `int` | Number of retry attempts on retryable errors (0 = no retry). |
| `RetryDelay` | `time.Duration` | Delay between retry attempts. |

### Default Step Configurations

| Step ID | Timeout | Max Retries | Retry Delay |
|---------|---------|-------------|-------------|
| `mkdir` | 15s | 0 | - |
| `bootstrap` | 2m | 0 | - |
| `extract` | 1m | 0 | - |
| `setup` | 5m | 0 | - |
| `autoheal` | 15s | 0 | - |
| `verify_setup` | 10s | 0 | - |
| `scenario_stop` | 30s | 0 | - |
| `caddy_install` | 1m | 0 | - |
| `caddy_config` | 15s | 0 | - |
| `firewall_inbound` | 15s | 0 | - |
| `secrets_provision` | 30s | 0 | - |
| `resource_start` | 2m | 1 | 5s |
| `scenario_deps` | 2m | 0 | - |
| `scenario_target` | 2m | 0 | - |
| `wait_for_ui` | 45s | 0 | - |
| `verify_local` | 15s | 0 | - |
| `verify_https` | 20s | 0 | - |
| `verify_origin` | 20s | 0 | - |
| `verify_public` | 20s | 0 | - |

Steps not listed inherit `DefaultRunOptions()` (context-based timeout). Use `RunOptionsForStep(stepID)` to get the merged options.

## ErrorInfo in API Responses

When a deployment step fails, the result types (`VPSSetupResult`, `VPSDeployResult`) include an optional `ErrorInfo` field with structured error metadata:

```json
{
  "ok": false,
  "error": "No space left on device",
  "error_info": {
    "message": "No space left on device",
    "category": "disk_full",
    "hint": "The server has run out of disk space. SSH in and run `df -h` to investigate.",
    "retryable": false,
    "exit_code": 1
  },
  "failed_step": "extract",
  "duration_ms": 45230
}
```

Progress events (SSE) also carry structured error fields when available: `error_category`, `retryable`, and `hint`.

## Canonical SSH Identity Model

> [CODE: api/sshidentity/model.go]
> [CODE: api/persistence/deployment.go]

Per-deployment SSH identity is stored in `deployments.ssh_identity` as JSON:

```json
{
  "key_path": "~/.ssh/id_ed25519",
  "public_key_fingerprint": "SHA256:...",
  "auth_mode": "explicit_key",
  "verification_state": "authorized",
  "last_verified_at": "2026-02-08T18:30:00Z"
}
```

Field semantics:
- `auth_mode`: `explicit_key` | `agent` | `default_ssh` | `unknown`
- `verification_state`: `authorized` | `unauthorized` | `unknown`
  - only meaningful as `authorized`/`unauthorized` for `explicit_key`
  - `agent`/`default_ssh` remain `unknown` because identity is ambient, not pinned

Resolver precedence:
1. `target.vps.key_path` from manifest
2. persisted explicit key path (if present and local key exists)
3. ambient SSH transport classification (`agent` or `default_ssh`)
