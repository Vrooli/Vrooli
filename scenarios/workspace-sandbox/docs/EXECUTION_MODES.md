# Execution Modes Guide

> **Last Updated**: 2026-04-28
> **Status**: All Phases Complete (Resource Limits, Log Capture, UI, Interactive Sessions, Structured Exit / Split Streams / Native Stdin / SSE)

This guide explains how to execute commands and run processes in workspace-sandbox, including isolation levels, resource limits, and best practices.

## /processes capability matrix (background mode)

| Capability | Wire surface | Notes |
|---|---|---|
| Start process | `POST /sandboxes/{id}/processes` | `withStdin: true` requests a real stdin pipe; default false. |
| Stream stdin | `POST /sandboxes/{id}/processes/{pid}/stdin?close=true` | Body is appended to the process stdin; `close=true` sends EOF. |
| Read stdout/stderr (single pull) | `GET /sandboxes/{id}/processes/{pid}/logs?stream=stdout\|stderr` | `stream` query parameter is required. |
| Stream stdout/stderr (push) | `GET /sandboxes/{id}/processes/{pid}/logs/stream?stream=stdout\|stderr` | Server-Sent Events. The server emits one `event: exit` (JSON ExitInfo) when the process terminates, then `event: end`. |
| Structured exit info | tracker JSON `exitCode`, `signal`, `oomKilled` | Driver wait reaper records these via `Tracker.RecordExit`; survives process disappearance. |
| Kill | `DELETE /sandboxes/{id}/processes/{pid}` | Best-effort SIGTERM→SIGKILL escalation. |
| List | `GET /sandboxes/{id}/processes` | Includes terminated processes with their exit info; pass `?running=true` to filter. |

The `agent-manager` `SandboxLauncher` is the canonical client and exercises every cell of this matrix in protected mode.

## Quick Reference

| Mode | Command | Use Case | Output |
|------|---------|----------|--------|
| **Quick Command** | `exec` | Single commands, builds, tests | Synchronous, returns after completion |
| **Background Process** | `run` | Long-running agents, daemons | Async, returns PID immediately |
| **Interactive** | `shell`/`attach` | REPLs, interactive tools | Real-time bidirectional I/O |

| Isolation | Flag | Access | Network |
|-----------|------|--------|---------|
| **Full** | (default) | Only `/workspace` | Blocked |
| **Vrooli-Aware** | `--vrooli-aware` | CLIs, configs, localhost | Unrestricted (intended: localhost-only — no backend enforces `network-loopback-only` yet; tracked as `knw-1784006975589682125`) |

## Execution Modes

### Quick Command (`exec`)

Run a single command and wait for it to complete. Best for:
- Build commands (`npm run build`, `go build`)
- Test suites (`pytest`, `npm test`)
- One-off scripts
- File operations

```bash
# Basic usage
workspace-sandbox process exec <sandbox-id> -- <command> [args...]

# Examples
workspace-sandbox process exec abc123 -- ls -la /workspace
workspace-sandbox process exec abc123 -- npm run build
workspace-sandbox process exec abc123 -- python script.py --input data.json
```

**Characteristics:**
- Blocks until the command completes
- Returns stdout/stderr after completion
- Exit code propagated to CLI
- Supports timeout for long-running commands

### Background Process (`run`)

Start a detached process that runs in the background. Best for:
- Long-running agents
- Development servers
- Background tasks
- Processes you want to monitor via logs

```bash
# Basic usage
workspace-sandbox process run <sandbox-id> [OPTIONS] -- <command> [args...]

# Examples
workspace-sandbox process run abc123 --name=my-agent -- python agent.py
workspace-sandbox process run abc123 -- npm run dev
workspace-sandbox process run abc123 --memory=1024 -- ./train.sh
```

**Characteristics:**
- Returns immediately with PID
- Output captured to log files
- Monitor with `workspace-sandbox process logs <sandbox-id> --pid=<PID> --follow`
- Kill with `workspace-sandbox process kill <sandbox-id> --pid=<PID>`

### Interactive Session (`shell`/`attach`)

Real-time bidirectional I/O with full PTY support. Best for:
- Interactive shells
- Python/Node REPLs
- Interactive tools (vim, htop)
- Debugging sessions

```bash
# Open a shell in the sandbox
workspace-sandbox process shell <sandbox-id> [OPTIONS]

# Run a specific command interactively
workspace-sandbox process attach <sandbox-id> [OPTIONS] -- <command> [args...]

# Examples
workspace-sandbox process shell abc123                              # Opens /bin/sh (or $SHELL)
workspace-sandbox process shell abc123 --vrooli-aware               # With Vrooli CLI access
workspace-sandbox process attach abc123 -- python                   # Python REPL
workspace-sandbox process attach abc123 -- node                     # Node.js REPL
workspace-sandbox process attach abc123 --memory=512 -- vim file.py # Edit with vim
```

**Characteristics:**
- Full PTY support (colors, cursor positioning, line editing)
- Terminal resize support (automatically detects window changes)
- Real-time stdin/stdout streaming via WebSocket
- Exit with Ctrl+D or by typing `exit`

## Isolation Levels

### Full Isolation (Default)

Maximum security - the process only sees the sandbox workspace:

```
/
├── bin/                 (read-only system binaries)
├── lib/, lib64/         (read-only libraries)
├── usr/                 (read-only)
├── proc/                (process info)
├── dev/                 (minimal devices)
├── tmp/                 (writable tmpfs, ephemeral)
├── etc/                 (only resolv.conf, hosts, passwd, group)
├── workspace/           (sandbox merged - READ/WRITE)
└── workspace-readonly/  (original repo - READ-ONLY)
```

**Blocked:**
- Network access
- Home directory
- Other sandboxes
- System configuration

**Best for:**
- Untrusted code execution
- Maximum isolation requirements
- Single-task processing

### Vrooli-Aware Isolation

Extended access for orchestration tasks:

```bash
workspace-sandbox process exec abc123 --vrooli-aware -- visited-tracker list
workspace-sandbox process run abc123 --vrooli-aware -- scenario-manager start
```

**Additional access:**
- `$HOME` through a per-sandbox writable overlay for ordinary CLI config/cache reads and writes.
- `~/.vrooli` as a live read-only host mount. This keeps runtime databases, port state, and logs current inside long-lived sandboxes without allowing agents to mutate host runtime state directly.
- Localhost network (can reach local APIs)
- Vrooli environment variables
- `vrooli scenario start|restart|stop <name>` and `vrooli scenario port <name> <port-name>` through the workspace-sandbox host lifecycle proxy. The proxy accepts only those allowlisted scenario actions and strips sandbox identity before executing the host `vrooli` command. Port lookup remains dynamic and host-authoritative; sandboxed callers never infer target ports.

**Best for:**
- Agents that orchestrate other scenarios
- Commands that need to call other Vrooli CLIs
- Processes that communicate with local services

### Per-OS Containment

Isolation is carried out by a per-OS *containment backend* selected behind a
platform-neutral seam. Each backend reports the guarantees it enforces using a
platform-neutral vocabulary (served by `GET /api/v1/driver/containment`), so
clients reason about isolation strength without knowing the OS mechanism. The
backends are honest about what they cannot enforce: a missing guarantee is
reported as absent, not silently downgraded.

| Enforcement | Linux (`bwrap`) | macOS (`seatbelt`) | Other (`none`) |
|-------------|:---------------:|:------------------:|:--------------:|
| `filesystem-write-containment` (writes confined to the sandbox writable set) | ✅ | ✅ | ❌ |
| `network-deny` (network blocked unless the profile allows it) | ✅ | ✅ | ❌ |
| `pid-namespace` (process table isolated) | ✅ | ❌ | ❌ |
| `path-illusion` (workspace visible at `/workspace`) | ✅ | ❌ | ❌ |

- **Linux** uses [bubblewrap](https://github.com/containers/bubblewrap): the
  merged dir is bind-mounted at `/workspace` (path illusion), namespaces are
  unshared (pid, network when denied), and resource limits are applied via
  `prlimit`.
- **macOS** uses [Seatbelt](https://developer.apple.com/) via the system
  `sandbox-exec` binary with a generated profile. It is **partial by design**:
  writes are denied outside the sandbox writable set and network is denied when
  the profile disallows it, but there is **no path illusion** (the workspace
  stays at its host `mergedDir`, so `workspacePath == mergedDir` and
  `pathIllusion == false`) and **no pid namespace**. Resource limits are applied
  by an in-binary `rlimit-exec` self-exec shim (`setrlimit`) that replaces
  Linux-only `prlimit`.
- **Other platforms** have no native containment backend: tracking execution
  uses the direct path and reports no containment guarantees; protected
  execution is rejected as unavailable rather than silently downgraded.

Because Seatbelt still enforces the two guarantees `agent-manager` protected
mode depends on (`filesystem-write-containment` + `network-deny`), protected
mode runs on macOS without a degradation warning; the absent path-illusion and
pid-namespace guarantees are reported but not required by protected mode.

> **Cross-platform status.** The per-OS compile matrix, driver availability,
> and shipped-vs-deferred items are tracked in
> [`docs/internal/PORTABILITY_AUDIT.md`](internal/PORTABILITY_AUDIT.md). macOS
> support is verified at the compile/unit level on the Linux dev host; the
> operator checklist for the real-Mac field shakeout is
> [`docs/guides/macos-shakeout.md`](guides/macos-shakeout.md).

## Resource Limits

Resource limits help prevent runaway processes and enable fair resource sharing.

### Available Limits

| Flag | API Field | Description | Default |
|------|-----------|-------------|---------|
| `--memory=<MB>` | `memoryLimitMB` | Max address space in MB | Unlimited |
| `--cpu-time=<SEC>` | `cpuTimeSec` | Max CPU seconds consumed | Unlimited |
| `--timeout=<SEC>` | `timeoutSec` | Wall-clock timeout (exec only) | Unlimited |
| `--max-procs=<N>` | `maxProcesses` | Max child processes | Unlimited |
| `--max-files=<N>` | `maxOpenFiles` | Max open file descriptors | System default |

### Recommended Defaults

| Use Case | Memory | CPU Time | Timeout | Max Procs |
|----------|--------|----------|---------|-----------|
| Quick build/test | 512 MB | 60s | 120s | 50 |
| Agent session | 1024 MB | 600s | 3600s | 100 |
| Data processing | 2048 MB | 1800s | 7200s | 200 |

### Examples with Limits

```bash
# Memory-constrained build
workspace-sandbox process exec abc123 --memory=512 --timeout=120 -- npm run build

# Long-running agent with CPU limit
workspace-sandbox process run abc123 --memory=1024 --cpu-time=3600 -- python agent.py

# Resource-intensive processing
workspace-sandbox process exec abc123 --memory=2048 --max-procs=50 -- ./process-data.sh
```

## CLI Reference

### exec

Execute a command synchronously and return output.

```
workspace-sandbox process exec <sandbox-id> [OPTIONS] -- <command> [args...]

Options:
  --memory=<MB>        Memory limit in MB (0 = unlimited)
  --cpu-time=<SEC>     CPU time limit in seconds (0 = unlimited)
  --timeout=<SEC>      Wall-clock timeout in seconds (0 = unlimited)
  --max-procs=<N>      Max child processes (0 = unlimited)
  --max-files=<N>      Max open file descriptors (0 = unlimited)
  --network            Allow network access (default: blocked)
  --vrooli-aware       Use Vrooli-aware isolation (can access CLIs, localhost)
  --workdir=<PATH>     Working directory inside sandbox (default: /workspace)
  --env=KEY=VALUE      Set environment variable (repeatable)
  --json               Output result as JSON
```

### run

Start a background process and return PID.

```
workspace-sandbox process run <sandbox-id> [OPTIONS] -- <command> [args...]

Options:
  --name=<NAME>        Friendly name for the process
  --memory=<MB>        Memory limit in MB (0 = unlimited)
  --cpu-time=<SEC>     CPU time limit in seconds (0 = unlimited)
  --max-procs=<N>      Max child processes (0 = unlimited)
  --max-files=<N>      Max open file descriptors (0 = unlimited)
  --network            Allow network access (default: blocked)
  --vrooli-aware       Use Vrooli-aware isolation
  --workdir=<PATH>     Working directory inside sandbox (default: /workspace)
  --env=KEY=VALUE      Set environment variable (repeatable)
  --json               Output result as JSON
```

### processes

List processes in a sandbox.

```
workspace-sandbox process list <sandbox-id> [OPTIONS]

Options:
  --running            Only show running processes
  --json               Output as JSON
```

### kill

Terminate processes in a sandbox.

```
workspace-sandbox process kill <sandbox-id> (--pid=<PID> | --all)

Options:
  --pid=<PID>          Kill specific process by PID
  --all                Kill all processes in sandbox
```

### logs

View or stream process logs.

```
workspace-sandbox process logs <sandbox-id> [OPTIONS]

Options:
  --pid=<PID>          View logs for specific process (omit to list all)
  --follow, -f         Stream logs in real-time
  --tail=<N>           Show last N lines (default: all)
  --stream=<STREAM>    Read stdout (default) or stderr
  --json               Output as JSON
```

**Examples:**

```bash
# List all logs for a sandbox
workspace-sandbox process logs abc123

# View specific process logs
workspace-sandbox process logs abc123 --pid=12345 --stream=stdout

# Follow logs in real-time
workspace-sandbox process logs abc123 --pid=12345 --stream=stderr --follow

# Show last 50 lines
workspace-sandbox process logs abc123 --pid=12345 --stream=stdout --tail=50
```

### shell

Open an interactive shell in the sandbox.

```
workspace-sandbox process shell <sandbox-id> [OPTIONS]

Options:
  --memory=<MB>        Memory limit in MB (0 = unlimited)
  --vrooli-aware       Use Vrooli-aware isolation
  --network            Allow network access (default: blocked)
```

**Examples:**

```bash
# Open default shell
workspace-sandbox process shell abc123

# Shell with Vrooli CLI access
workspace-sandbox process shell abc123 --vrooli-aware

# Memory-limited shell
workspace-sandbox process shell abc123 --memory=512
```

### attach

Run a command interactively with full PTY support.

```
workspace-sandbox process attach <sandbox-id> [OPTIONS] -- <command> [args...]

Options:
  --memory=<MB>        Memory limit in MB (0 = unlimited)
  --vrooli-aware       Use Vrooli-aware isolation
  --network            Allow network access (default: blocked)
```

**Examples:**

```bash
# Python REPL
workspace-sandbox process attach abc123 -- python

# Node.js REPL
workspace-sandbox process attach abc123 -- node

# Edit file with vim
workspace-sandbox process attach abc123 -- vim myfile.py

# Interactive tool with Vrooli access
workspace-sandbox process attach abc123 --vrooli-aware -- htop
```

## API Reference

For command, background-process and interactive launches, omitted or `null`
`allowNetwork` inherits the
isolation profile. Explicit `false` disables network access even when the profile
allows it; a tracking-only backend refuses that request before launching.
Explicit `true` enables network access. The response's containment enforcements
describe the effective launch, not merely the selected profile.

### Execute Command

```http
POST /api/v1/sandboxes/{id}/exec
Content-Type: application/json

{
  "command": "npm",
  "args": ["run", "build"],
  "isolationLevel": "full",
  "memoryLimitMB": 512,
  "cpuTimeSec": 60,
  "timeoutSec": 120,
  "allowNetwork": false,
  "env": {"NODE_ENV": "production"},
  "workingDir": "/workspace"
}
```

**Response:**

```json
{
  "exitCode": 0,
  "stdout": "Build complete!",
  "stderr": "",
  "pid": 12345,
  "timedOut": false
}
```

### Start Background Process

```http
POST /api/v1/sandboxes/{id}/processes
Content-Type: application/json

{
  "command": "python",
  "args": ["agent.py"],
  "name": "my-agent",
  "isolationLevel": "vrooli-aware",
  "memoryLimitMB": 1024,
  "cpuTimeSec": 3600,
  "allowNetwork": true
}
```

**Response:**

```json
{
  "pid": 12345,
  "sandboxId": "abc123-...",
  "command": "python",
  "name": "my-agent",
  "startedAt": "2025-12-18T10:30:00Z",
  "logPath": "/var/lib/workspace-sandbox/abc123/logs/12345.log"
}
```

### Get Process Logs

```http
GET /api/v1/sandboxes/{id}/processes/{pid}/logs?tail=50&offset=0
```

**Response:**

```json
{
  "pid": 12345,
  "sandboxId": "abc123-...",
  "path": "/var/lib/workspace-sandbox/abc123/logs/12345.log",
  "sizeBytes": 1024,
  "isActive": true,
  "content": "Agent started...\nProcessing task 1..."
}
```

### Stream Process Logs (SSE)

```http
GET /api/v1/sandboxes/{id}/processes/{pid}/logs/stream
Accept: text/event-stream
```

Events:
- `data: <log content>` - New log content
- `event: error, data: <message>` - Error occurred
- `event: end, data: stream closed` - Process ended

### Interactive Session (WebSocket)

```http
GET /api/v1/sandboxes/{id}/exec-interactive
Upgrade: websocket
Connection: Upgrade
```

**Protocol:**

1. Client connects via WebSocket
2. Client sends start request as first message:

```json
{
  "command": "/bin/bash",
  "args": [],
  "isolationLevel": "full",
  "allowNetwork": false,
  "memoryLimitMB": 512,
  "cols": 80,
  "rows": 24
}
```

3. Server allocates PTY and starts process
4. Messages flow bidirectionally:

```json
// Client → Server: stdin
{"type": "stdin", "data": "ls -la\n"}

// Server → Client: stdout
{"type": "stdout", "data": "total 12\ndrwxr-xr-x ..."}

// Client → Server: terminal resize
{"type": "resize", "cols": 120, "rows": 40}

// Server → Client: process exit
{"type": "exit", "code": 0}

// Server → Client: error
{"type": "error", "data": "failed to start process"}

// Client → Server: ping (keepalive)
{"type": "ping"}

// Server → Client: pong
{"type": "pong"}
```

### List All Logs

```http
GET /api/v1/sandboxes/{id}/logs
```

## Examples and Use Cases

### Running a Build

```bash
# Create sandbox for a project
sandbox_id=$(workspace-sandbox create --scope=/home/user/myproject --json | jq -r '.id')

# Run build with resource limits
workspace-sandbox process exec $sandbox_id --memory=1024 --timeout=300 -- npm run build

# Check the diff
workspace-sandbox diff $sandbox_id

# Approve changes if build artifacts should be kept
workspace-sandbox approve $sandbox_id -m "Build artifacts"
```

### Running an Agent

```bash
# Create sandbox
sandbox_id=$(workspace-sandbox create --scope=/home/user/myproject --json | jq -r '.id')

# Start agent in background
workspace-sandbox process run $sandbox_id --name=code-agent --memory=2048 -- \
  python agent.py --task "refactor authentication"

# Monitor progress
workspace-sandbox process logs $sandbox_id --follow

# Check agent's changes
workspace-sandbox diff $sandbox_id

# Stop agent when done
workspace-sandbox process kill $sandbox_id --all

# Review and approve changes
workspace-sandbox approve $sandbox_id -m "Agent refactoring"
```

### Orchestrating Other Scenarios

```bash
# Create sandbox with Vrooli-aware isolation
sandbox_id=$(workspace-sandbox create --scope=/home/user/myproject --json | jq -r '.id')

# Run command that uses other Vrooli CLIs
workspace-sandbox process exec $sandbox_id --vrooli-aware --network -- \
  visited-tracker add "localhost:8080"

# Check status of another scenario
workspace-sandbox process exec $sandbox_id --vrooli-aware -- \
  scenario-manager status api-server
```

## Troubleshooting

### Process Times Out

**Symptoms:** Process exits with code 124, `timedOut: true`

**Solutions:**
1. Increase `--timeout` value
2. For background processes, use `run` instead of `exec` (no timeout enforced)
3. Check if the process is stuck (logs)

### Memory Limit Exceeded

**Symptoms:** Process killed unexpectedly, exit code 137

**Solutions:**
1. Increase `--memory` value
2. Profile your process to understand memory usage
3. Process large data in chunks

### Network Access Denied

**Symptoms:** Connection refused, network errors

**Solutions:**
1. Add `--network` flag for full network access
2. Use `--vrooli-aware` for Vrooli CLI/config + network access (note: grants unrestricted network today, not loopback-only — see the isolation table above)
3. Verify the network access is actually needed

### Logs Empty or Missing

**Symptoms:** Log file exists but is empty

**Solutions:**
1. Ensure process is writing to stdout/stderr (not a file directly)
2. Check process is actually running: `workspace-sandbox process list <sandbox-id> --running`
3. Some processes buffer output - try adding flush or unbuffered mode

### Process Not Found After Start

**Symptoms:** `run` returns PID but `processes` shows nothing

**Solutions:**
1. Process may have exited immediately - check `processes` without `--running`
2. Check logs for startup errors: `workspace-sandbox process logs <sandbox-id> --pid=<PID>`
3. Verify command path is correct and executable exists in sandbox

## Companion services (F7)

Sandbox companion services are declared per sandbox. The current API stores a complete, versioned declaration in sandbox metadata and records a declaration audit event:

Client Rehearsal obtains its task-owned Sandbox UUID through the existing creation API; there is no separate rehearsal-only sandbox type or hidden ID allocation path. Create with the workspace scope/root it is authorized to isolate, a task-specific `owner`, and a stable `idempotencyKey`, then use the returned `id` for the companion routes below:

```http
POST /api/v1/sandboxes
Content-Type: application/json

{"name":"f7-rehearsal-task-42","scopePath":"/synthetic-rehearsal","projectRoot":"/workspace/client-rehearsal","owner":"client-rehearsal:task-42","tags":["f7","task-owned"],"idempotencyKey":"client-rehearsal:task-42"}
```

The caller must supply paths accepted by the normal Sandbox creation contract. Profile declaration, start, stop, and delete all address the returned UUID. Reusing the idempotency key allows safe create retries to resolve to the same sandbox. The `owner` label and UUID provide task association, but this API does not enforce authorization by owner label; the engagement service must retain and use only the UUID returned for that task.

Create retries bind the key to the original owner, owner type, canonical project root and canonical scope. A retry with a different owner or workspace identity receives a conflict without the existing sandbox ID. Matching retries return the existing record, including companion profile metadata. Callers must use a new idempotency key for a distinct task; knowing the key is not a substitute for the caller-side task/UUID association.

```http
PUT /api/v1/sandboxes/{id}/companion-services
Content-Type: application/json

{"services":[{"name":"workflow-apis","kind":"http-mock","version":"1.0.0","httpProfile":{"schemaVersion":1,"operations":[{"selector":{"nodeType":"n8n-nodes-base.gmail","resource":"message","operation":"send"},"route":{"host":"gmail.synthetic.invalid","method":"POST","path":"/v1/messages","status":202,"contentType":"application/json","body":"{\"messageId\":\"message_synthetic_1\",\"status\":\"queued\"}"}},{"selector":{"nodeType":"n8n-nodes-base.googleSheets","resource":"sheet","operation":"append"},"route":{"host":"sheets.synthetic.invalid","method":"POST","path":"/v1/rows","query":"spreadsheetId=synthetic-sheet","status":201,"contentType":"application/json","body":"{\"rowId\":\"row_synthetic_1\"}"}}]}}]}
```

Read the declaration with `GET /api/v1/sandboxes/{id}/companion-services`. The declaration accepts at most 16 named services. Supported kinds are `postgres`, `redis`, and `http-mock`; versions must be pinned numeric versions. Redis remains fail-closed. HTTP mock declarations may use either legacy `httpFixtures` or the generic `httpProfile`, but never both. A profile has schema version 1 and 1–128 operations. Each operation maps a reusable `(nodeType, resource, operation)` selector to one exact synthetic route and fixed response. Selectors and routes must each be unique; selectors contain no workflow ID or node-instance name. Workspace Sandbox validates the bounded selector tuple and fixture fields, sorts normalized operations deterministically, and exposes the resulting SHA-256 profile digest.

Each route requires an exact `*.synthetic.invalid` host, allowlisted method, exact path and fixed status/content type/body. Optional query strings are parsed, bounded to 32 unique keys / 2 KiB total, and canonicalized by key order; duplicate values, malformed encodings, sensitive/credential-shaped values and real email addresses are rejected. Responses support JSON, plain text and XML, with a 64 KiB per-response and 1 MiB aggregate response cap. Redirect statuses, sensitive response fields and duplicate routes are rejected. The profile adapter in client-rehearsal interprets OpenAPI or recorded exchanges and sends normalized static operations; Workspace Sandbox does not parse source descriptions, open their URLs or accept upstream forwarding behavior. Unknown routes return a local not-found response.

Start and stop a declared service with `POST /api/v1/sandboxes/{id}/companion-services/{name}/start` and `/stop`. Workspace Sandbox delegates to fixed n8n resource-owner operations; it never calls Docker itself. HTTP mock starts pass only the bounded static routes on owner stdin, require an owner receipt for the exact fixture hosts, fixed port 18080, the matching Sandbox UUID, attached status and `noEgressVerified: true`. The start response includes host base addresses, normalized selector-to-operation URLs/methods, and the normalized profile digest. A declaration is not proof that a service is running. Sandbox stop/delete stops persisted running companions first, records each successful stop, and refuses to finish if cleanup fails. Owner fake tests verify the contract; actual container DNS/route response, no-egress enforcement and cleanup still require runtime evidence.

## See Also

- [Architecture Documentation](ARCHITECTURE.md) - System design and internals
- [PRD.md](../PRD.md) - Requirements and planned phases
