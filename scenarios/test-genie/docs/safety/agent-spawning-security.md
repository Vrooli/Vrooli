# Agent Spawning Security Model

This document describes the security architecture for spawned AI agents in the test-genie Generate tab, including current limitations and safeguards.

## Overview

The Generate tab allows spawning multiple AI agents in parallel to generate tests for scenarios. These agents run via the `resource-opencode` CLI tool and interact with the filesystem through OpenCode's tool system.

## Security Architecture

### Defense-in-Depth Layers

```
┌─────────────────────────────────────────────────────────────────┐
│ Layer 1: UI Validation                                          │
│ - Immutable safety preamble (locked in editor)                  │
│ - Session-level spawn conflict tracking                         │
│ - Scope conflict preview before spawning                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 2: API Validation (server-side enforcement)               │
│ - skipPermissions flag rejected (always)                        │
│ - Tool allowlist validation (not blocklist)                     │
│ - Scope path traversal prevention                               │
│ - Prompt scanning for dangerous patterns                        │
│ - Server-generated preamble (replaces client preamble)          │
│ - Idempotency keys (prevent duplicate spawns)                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 3: OpenCode Tool-Level Restrictions                       │
│ - --directory flag sets working directory                       │
│ - Permission system for file operations                         │
│ - Tool-specific access controls                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 4: Scope Locking (coordination)                           │
│ - Database-backed path locks                                    │
│ - 20-minute lock timeout with heartbeat renewal                 │
│ - Prevents overlapping agent work                               │
└─────────────────────────────────────────────────────────────────┘
```

## Known Limitations

### Directory Scoping Is NOT a Security Boundary

**Critical**: The `--directory` flag passed to OpenCode sets the working directory but does NOT enforce filesystem isolation at the OS level.

As acknowledged in [OpenCode issue #2242](https://github.com/sst/opencode/issues/2242):
> "we try to restrict to cwd but agent can use bash to get around it"

**Implication**: An agent could theoretically use bash commands to access files outside its designated directory. The real security comes from the **tool allowlist**, not directory scoping.

### Mitigation

The tool allowlist ensures agents can only execute pre-approved bash commands:

```go
// Only these bash command prefixes are allowed
allowedCommands := []AllowedBashCommand{
    {Prefix: "pnpm test", Description: "pnpm test runner"},
    {Prefix: "go test", Description: "Go test runner"},
    {Prefix: "git status", Description: "Git status"},
    {Prefix: "git diff", Description: "Git diff"},
    {Prefix: "ls", Description: "List directory"},
    // ... other safe commands
}
```

Unrestricted bash (`bash`, `bash(*)`, `*`) is explicitly rejected.

## Security Controls

### 1. Tool Allowlist (Primary Defense)

The bash command validator uses an **allowlist** approach - only explicitly permitted commands are allowed.

**Allowed categories:**
- Test runners: `pnpm test`, `go test`, `vitest`, `jest`, `bats`, `pytest`
- Build commands: `pnpm build`, `go build`, `make`
- Linters: `eslint`, `prettier`, `gofmt`, `golangci-lint`
- Safe inspection: `ls`, `pwd`, `cat`, `head`, `tail`, `diff`, `find`
- Git read-only: `git status`, `git diff`, `git log`, `git show`, `git branch`

**Blocked categories:**
- Unrestricted bash access
- Git write operations: `git push`, `git checkout`, `git commit`, `git add`
- File modifications: `rm`, `mv`, `cp`, `mkdir`, `chmod`, `chown`
- System commands: `sudo`, `su`, `systemctl`, `kill`
- Package managers: `npm install`, `pip install`, `go get`
- Network mutations: `curl -X POST`, remote script execution

**Glob Pattern Restrictions:**

Glob patterns (`*`, `?`) are only allowed with specific safe commands to prevent file enumeration attacks:

```go
// Safe glob prefixes - only these commands can use wildcards
safeGlobPrefixes := []string{
    "bats", "go test", "pytest", "vitest", "jest",
    "pnpm test", "npm test", "make test", "ls", "find",
}
```

**Examples:**
- ✅ `bats *.bats` - allowed (test runner)
- ✅ `ls *.go` - allowed (directory listing)
- ❌ `cat *.log` - blocked (could read sensitive files)
- ❌ `echo *` - blocked (could leak directory contents)

### 2. skipPermissions Rejection

The `skipPermissions` flag is **always rejected** for spawned agents:

```go
if payload.SkipPermissions {
    s.writeError(w, http.StatusBadRequest,
        "skipPermissions is not allowed for spawned agents")
    return
}
```

This ensures agents cannot auto-approve all tool operations.

### 3. Path Traversal Prevention

Scope paths are validated to prevent directory escape:

```go
// Blocked patterns:
// - ../../../etc/passwd
// - /etc/passwd (absolute paths)
// - ~/. ssh/id_rsa (home directory)
// - foo/../../bar (traversal in middle)
```

### 4. Prompt Scanning

Prompts are scanned for dangerous patterns before execution:

- `rm -rf`, `sudo`, `chmod 777`
- `git push --force`, `git checkout --force`
- `DROP TABLE`, `DELETE FROM`
- `curl | bash`, `wget | sh`

### 5. Server-Generated Preamble

The safety preamble is generated server-side and cannot be modified by clients:

```go
// Server always generates its own preamble
serverPreamble := generateSafetyPreamble(scenario, scope, repoRoot)

// Client-provided preamble is logged but replaced
if payload.Preamble != "" {
    s.log("preamble mismatch detected - using server-generated preamble", ...)
}
```

### 6. Scope Locking

Database-backed locks prevent agents from working on overlapping paths:

- Locks acquired when agent is registered
- 20-minute timeout (configurable via `AGENT_LOCK_TIMEOUT_MINUTES`)
- Heartbeat renewal every 5 minutes while agent runs
- Automatic release on agent completion/failure/stop

## Operational Security

### Monitoring

- All agent operations are logged
- WebSocket broadcasts agent status changes in real-time
- Active agents panel shows running agents and their scopes
- Scope locks are visible in the UI

### Stopping Agents

- Individual agents can be stopped via UI or API
- "Stop All" button terminates all running agents
- Process kill signals are sent to agent subprocesses
- Locks are released on stop

### Cleanup

- Background cleanup removes completed agents older than 7 days
- Manual cleanup available via API endpoint
- Orphan detection planned for future releases

## Future Improvements

### Planned Enhancements

1. **OS-Level Sandboxing**: Consider `bubblewrap` or Docker for true filesystem isolation
2. **Network Isolation**: Block outbound network access for agents
3. **Resource Limits**: CPU/memory limits on agent processes
4. **File Operation Auditing**: Track all Read/Write/Edit operations for review
5. **Orphan Process Detection**: Kill orphaned agents on server restart

### Not Currently Implemented

- Process-level sandboxing (agents run as same user)
- Network isolation (agents can make outbound requests)
- CPU/memory limits (agents can consume unlimited resources)
- Automatic rollback (no snapshot/restore capability)

## Risk Assessment

| Risk | Severity | Mitigation | Residual Risk |
|------|----------|------------|---------------|
| Arbitrary file read | Medium | Tool allowlist blocks most file access | Low - would need model to intentionally bypass |
| Arbitrary file write | Medium | Write tool restricted to scenario dir | Low - preamble instructs against |
| Arbitrary code execution | High | Bash allowlist only permits safe commands | Low - extensive allowlist validation |
| Resource exhaustion | Low | Timeout limits, manual stop capability | Medium - no CPU/memory limits |
| Agent interference | Low | Scope locking prevents overlap | Very Low - well-tested |

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENT_LOCK_TIMEOUT_MINUTES` | 20 | Lock expiration time |
| `AGENT_RETENTION_DAYS` | 7 | Days to keep completed agents |
| `VROOLI_ROOT` | `$HOME/Vrooli` | Repository root for path validation |

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/agents/spawn` | POST | Spawn new agents |
| `/api/v1/agents/active` | GET | List active agents |
| `/api/v1/agents/{id}/stop` | POST | Stop specific agent |
| `/api/v1/agents/stop-all` | POST | Stop all agents |
| `/api/v1/agents/blocked-commands` | GET | Get allowlist/blocklist info |

## References

- [OpenCode Documentation](https://opencode.ai/docs/)
- [OpenCode Sandboxing Discussion](https://github.com/sst/opencode/issues/2242)
- [Test-Genie API Handlers](../../api/internal/app/httpserver/agent_handlers.go)
- [Bash Command Validator](../../api/internal/app/httpserver/agent_registry.go)
