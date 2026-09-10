# SSH Configuration Reference

> [CODE: api/reach/sshadapter/runner.go] — RunOptions, SCPOptions, ConnectionConfig and defaults
> [CODE: api/reach/sshadapter/adapter.go] — exit-code classification into reach kinds
> [CODE: api/vps/step_config.go] — StepConfig and per-step defaults

The bounded SSH reach adapter (`api/reach/sshadapter`) is the only code that runs `ssh`/`scp`; it is selected by an explicit `ssh` transport binding and configured through typed options structs. This document describes every tunable lever, its default value, and when to adjust it.

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
| `CommandTimeout` | `time.Duration` | `0` (inherit from ctx) | Per-command timeout. When 0, defers to the context deadline. The adapter sets it from `reach.Command.Timeout`. |
| `Stdin` | `[]byte` | empty | Bytes fed to the remote command. A value that must reach the target travels here, never in the command string. |

### Presets

- **`DefaultRunOptions()`** -- Standard production settings. Used for every reach command over the ssh transport.

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

## Connection resolution

`ConnectionConfig` (host, port, user, key path, known_hosts file) is built per target by the server's resolver: the locator comes from the deployment's target binding and the key path from the credential binding `vrooli/scenario-to-cloud:ssh-key`. A target without a binding gets an empty key path, which means the operator's ambient identity (agent or default identity files) authenticates. No manifest field names a key.

## Error classification

The adapter classifies by exit code only and never reads message text: `255` (ssh's own failure: unreachable, refused, rejected key, changed host key) is `target_offline` with the first stderr line as detail; `127` (program absent on the target) is `reach_protocol_unsupported`; a context deadline is `transport_failure`; any other non-zero exit is the target's own typed answer and is returned to the caller, not raised.

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

| Action ID | Timeout | Max Retries | Retry Delay |
|---------|---------|-------------|-------------|
| `host.prepare` | 2m | 0 | - |
| `edge.firewall.allow` | 15s | 0 | - |
| `data.inventory` | 30s | 0 | - |
| `release.verify` | 1m | 0 | - |
| `release.stage` | 2m | 0 | - |
| `release.activate` | 2m | 0 | - |
| `install_vrooli` (sub-step of `release.activate`) | 2m | 0 | - |
| `config.apply` | 5m | 0 | - |
| `workload.stop` | 30s | 0 | - |
| `edge.route.apply` | 1m | 0 | - |
| `credentials.provision` | 30s | 0 | - |
| `runtime.start_dependencies` | 2m | 1 | 5s |
| `workload.start` | 2m | 0 | - |
| `verify.readiness` | 20s | 0 | - |
| `release.retain_predecessor` | 15s | 0 | - |

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
1. the key file the credential binding `vrooli/scenario-to-cloud:ssh-key` names
2. persisted explicit key path (if present and local key exists)
3. ambient SSH transport classification (`agent` or `default_ssh`)

## Authentication

> [CODE: api/authz/config.go] — `FromEnvironment`, the only place these variables are read
> [CODE: api/authz/routes.go] — the route table; rendered as [Authorization Matrix](authorization-matrix.md)
> [CODE: api/authz/middleware.go] — the default-deny enforcement boundary

Every management route passes through one boundary. There is no anonymous mode: the server refuses to start when the resolved provider chain is empty.

| Variable | Default | Description |
|----------|---------|-------------|
| `VROOLI_AUTH_MODE` | `personal_local` | `personal_local` proves the current OS user for loopback requests (no token) and appends any shared providers; `shared` uses only `VROOLI_AUTH_PROVIDERS` and fails startup when that list is empty. |
| `VROOLI_AUTH_PROVIDERS` | empty | Comma list of shared providers handled by `api-core/authn` (`scenario_authenticator`, `cloudflare_access`) with their own `VROOLI_AUTH_*` settings. |
| `VROOLI_AUTH_RECOVERY_URL` | provider default | Sign-in URL returned as the `next_action` of an `unauthenticated` refusal. |
| `API_BIND_ADDRESS` | `127.0.0.1` | Read by `api-core/server`. A non-loopback bind requires `VROOLI_ALLOWED_HOSTS`. |
| `VROOLI_ALLOWED_HOSTS` | loopback only | Host header values the API answers for; anything else is `forbidden_host`. |
| `VROOLI_ALLOWED_ORIGINS` | none | Extra browser origins (`scheme://host[:port]`) allowed to change state. Same-origin requests and, on a loopback bind, loopback origins (the scenario UI dev server) are always allowed. |
| `SCENARIO_TO_CLOUD_AUTHZ_POLICY` | `~/.vrooli/scenario-to-cloud/authz-policy.json` | Operator policy file binding principals to targets (below). |
| `SCENARIO_TO_CLOUD_MAX_BODY_BYTES` | `2097152` | Request body ceiling; larger bodies are `request_too_large`. |

### Scopes

Scopes are the shared coarse vocabulary declared in `.vrooli/service.json` (`authentication.capabilities`): `scenario-to-cloud:read`, `scenario-to-cloud:write`, `scenario-to-cloud:destructive`. The personal-local owner holds all three. Wildcards `*`, `scenario-to-cloud:*` and `*:<effect>` resolve through `scopecatalog.Resolve`.

### Target policy

```json
{
  "principals": {
    "osuser:1000":      { "environments": ["*"] },
    "ops@example.net":  { "environments": ["staging"], "machines": ["machine:m-2"] }
  },
  "revoked": ["former-operator"]
}
```

- A `personal_local` owner that the file does not name is unrestricted; every other unnamed principal has no target grant.
- A named principal may act on a deployment when the grant lists its environment or its target key (`machine:<id>` / `host:<host>`); `*` grants all.
- Legacy routes that name the target inside the request body (`/preflight`, `/preflight/fix/*`, `/preflight/disk/*`, `/vps/*`) are closed to restricted principals because the boundary cannot resolve that target ahead of the handler.
- `revoked` subjects are refused at admission and at the effect recheck; the file is re-read when its modification time changes.
- Refusals are `forbidden_target` and never disclose whether the id exists or where it lives.

### Principal kinds

Human sessions may use every row their scopes allow. Service principals and verified agents may use only rows marked `service_allowed` in the matrix (reads and health), never secret, host or interactive routes. Paired-node credentials arrive through the Bridge relay as forwarded `Authorization` headers and are verified by the same chain.

### Browser boundary

State-changing methods and the terminal upgrade check `Origin` (and `Sec-Fetch-Site: cross-site`); mismatches are `forbidden_origin`. The terminal WebSocket requires an `Origin`, an `interactive`-class grant (`scenario-to-cloud:destructive`) and is closed when the credential expires. Credential and identity responses carry `Cache-Control: no-store`. Each principal may have at most 4 effectful requests in flight (`too_many_requests`).

### Effect recheck and audit

Effectful handlers call `RequireEffect` immediately before their first side effect; it re-verifies the credential, the policy file and the target grant (`forbidden_revoked` / `forbidden_target`). Every admission, denial and effect writes an `authz.*` log line with actor, source, scope, route, deployment id, target key and outcome, never headers or bodies.

### CLI

The CLI sends `SCENARIO_TO_CLOUD_API_TOKEN` (or `VROOLI_API_TOKEN`, or the configured `token`) as a bearer credential. In `personal_local` mode no token is needed from the API host. A typed `unauthenticated` or `forbidden_*` refusal exits 2 and prints the server's `next_action`.
