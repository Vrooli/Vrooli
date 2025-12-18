# Execution Modes Guide

> **Last Updated**: 2025-12-18
> **Status**: All Phases Complete (Resource Limits, Log Capture, UI, Interactive Sessions)

This guide explains how to execute commands and run processes in workspace-sandbox, including isolation levels, resource limits, and best practices.

## Quick Reference

| Mode | Command | Use Case | Output |
|------|---------|----------|--------|
| **Quick Command** | `exec` | Single commands, builds, tests | Synchronous, returns after completion |
| **Background Process** | `run` | Long-running agents, daemons | Async, returns PID immediately |
| **Interactive** | `shell`/`attach` | REPLs, interactive tools | Real-time bidirectional I/O |

| Isolation | Flag | Access | Network |
|-----------|------|--------|---------|
| **Full** | (default) | Only `/workspace` | Blocked |
| **Vrooli-Aware** | `--vrooli-aware` | CLIs, configs, localhost | Localhost only |

## Execution Modes

### Quick Command (`exec`)

Run a single command and wait for it to complete. Best for:
- Build commands (`npm run build`, `go build`)
- Test suites (`pytest`, `npm test`)
- One-off scripts
- File operations

```bash
# Basic usage
workspace-sandbox exec <sandbox-id> -- <command> [args...]

# Examples
workspace-sandbox exec abc123 -- ls -la /workspace
workspace-sandbox exec abc123 -- npm run build
workspace-sandbox exec abc123 -- python script.py --input data.json
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
workspace-sandbox run <sandbox-id> [OPTIONS] -- <command> [args...]

# Examples
workspace-sandbox run abc123 --name=my-agent -- python agent.py
workspace-sandbox run abc123 -- npm run dev
workspace-sandbox run abc123 --memory=1024 -- ./train.sh
```

**Characteristics:**
- Returns immediately with PID
- Output captured to log files
- Monitor with `logs --follow`
- Kill with `kill --pid=<PID>`

### Interactive Session (`shell`/`attach`)

Real-time bidirectional I/O with full PTY support. Best for:
- Interactive shells
- Python/Node REPLs
- Interactive tools (vim, htop)
- Debugging sessions

```bash
# Open a shell in the sandbox
workspace-sandbox shell <sandbox-id> [OPTIONS]

# Run a specific command interactively
workspace-sandbox attach <sandbox-id> [OPTIONS] -- <command> [args...]

# Examples
workspace-sandbox shell abc123                              # Opens /bin/sh (or $SHELL)
workspace-sandbox shell abc123 --vrooli-aware               # With Vrooli CLI access
workspace-sandbox attach abc123 -- python                   # Python REPL
workspace-sandbox attach abc123 -- node                     # Node.js REPL
workspace-sandbox attach abc123 --memory=512 -- vim file.py # Edit with vim
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
workspace-sandbox exec abc123 --vrooli-aware -- visited-tracker list
workspace-sandbox run abc123 --vrooli-aware -- scenario-manager start
```

**Additional access:**
- `~/.local/bin/` (Vrooli CLIs, read-only)
- `~/.config/vrooli/` (CLI configurations, read-only)
- Localhost network (can reach local APIs)
- Vrooli environment variables

**Best for:**
- Agents that orchestrate other scenarios
- Commands that need to call other Vrooli CLIs
- Processes that communicate with local services

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
workspace-sandbox exec abc123 --memory=512 --timeout=120 -- npm run build

# Long-running agent with CPU limit
workspace-sandbox run abc123 --memory=1024 --cpu-time=3600 -- python agent.py

# Resource-intensive processing
workspace-sandbox exec abc123 --memory=2048 --max-procs=50 -- ./process-data.sh
```

## CLI Reference

### exec

Execute a command synchronously and return output.

```
workspace-sandbox exec <sandbox-id> [OPTIONS] -- <command> [args...]

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
workspace-sandbox run <sandbox-id> [OPTIONS] -- <command> [args...]

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
workspace-sandbox processes <sandbox-id> [OPTIONS]

Options:
  --running            Only show running processes
  --json               Output as JSON
```

### kill

Terminate processes in a sandbox.

```
workspace-sandbox kill <sandbox-id> (--pid=<PID> | --all)

Options:
  --pid=<PID>          Kill specific process by PID
  --all                Kill all processes in sandbox
```

### logs

View or stream process logs.

```
workspace-sandbox logs <sandbox-id> [OPTIONS]

Options:
  --pid=<PID>          View logs for specific process (omit to list all)
  --follow, -f         Stream logs in real-time
  --tail=<N>           Show last N lines (default: all)
  --json               Output as JSON
```

**Examples:**

```bash
# List all logs for a sandbox
workspace-sandbox logs abc123

# View specific process logs
workspace-sandbox logs abc123 --pid=12345

# Follow logs in real-time
workspace-sandbox logs abc123 --pid=12345 --follow

# Show last 50 lines
workspace-sandbox logs abc123 --pid=12345 --tail=50
```

### shell

Open an interactive shell in the sandbox.

```
workspace-sandbox shell <sandbox-id> [OPTIONS]

Options:
  --memory=<MB>        Memory limit in MB (0 = unlimited)
  --vrooli-aware       Use Vrooli-aware isolation
  --network            Allow network access (default: blocked)
```

**Examples:**

```bash
# Open default shell
workspace-sandbox shell abc123

# Shell with Vrooli CLI access
workspace-sandbox shell abc123 --vrooli-aware

# Memory-limited shell
workspace-sandbox shell abc123 --memory=512
```

### attach

Run a command interactively with full PTY support.

```
workspace-sandbox attach <sandbox-id> [OPTIONS] -- <command> [args...]

Options:
  --memory=<MB>        Memory limit in MB (0 = unlimited)
  --vrooli-aware       Use Vrooli-aware isolation
  --network            Allow network access (default: blocked)
```

**Examples:**

```bash
# Python REPL
workspace-sandbox attach abc123 -- python

# Node.js REPL
workspace-sandbox attach abc123 -- node

# Edit file with vim
workspace-sandbox attach abc123 -- vim myfile.py

# Interactive tool with Vrooli access
workspace-sandbox attach abc123 --vrooli-aware -- htop
```

## API Reference

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
workspace-sandbox exec $sandbox_id --memory=1024 --timeout=300 -- npm run build

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
workspace-sandbox run $sandbox_id --name=code-agent --memory=2048 -- \
  python agent.py --task "refactor authentication"

# Monitor progress
workspace-sandbox logs $sandbox_id --follow

# Check agent's changes
workspace-sandbox diff $sandbox_id

# Stop agent when done
workspace-sandbox kill $sandbox_id --all

# Review and approve changes
workspace-sandbox approve $sandbox_id -m "Agent refactoring"
```

### Orchestrating Other Scenarios

```bash
# Create sandbox with Vrooli-aware isolation
sandbox_id=$(workspace-sandbox create --scope=/home/user/myproject --json | jq -r '.id')

# Run command that uses other Vrooli CLIs
workspace-sandbox exec $sandbox_id --vrooli-aware --network -- \
  visited-tracker add "localhost:8080"

# Check status of another scenario
workspace-sandbox exec $sandbox_id --vrooli-aware -- \
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
2. Use `--vrooli-aware` for localhost-only access
3. Verify the network access is actually needed

### Logs Empty or Missing

**Symptoms:** Log file exists but is empty

**Solutions:**
1. Ensure process is writing to stdout/stderr (not a file directly)
2. Check process is actually running: `workspace-sandbox processes <sandbox-id> --running`
3. Some processes buffer output - try adding flush or unbuffered mode

### Process Not Found After Start

**Symptoms:** `run` returns PID but `processes` shows nothing

**Solutions:**
1. Process may have exited immediately - check `processes` without `--running`
2. Check logs for startup errors: `workspace-sandbox logs <sandbox-id> --pid=<PID>`
3. Verify command path is correct and executable exists in sandbox

## See Also

- [Architecture Documentation](ARCHITECTURE.md) - System design and internals
- [Execution Modes Plan](EXECUTION_MODES_PLAN.md) - Implementation details and future phases
