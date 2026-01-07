# Agent Spawning Security Model

This document describes the security architecture for spawned AI agents in the test-genie Generate tab, including current limitations and safeguards.

## Overview

The Generate tab allows spawning multiple AI agents in parallel to generate tests for scenarios. Agent execution is managed by the **agent-manager** service, which handles tasks, runs, profiles, and coordination.

## Security Architecture

### Defense-in-Depth Layers

```
┌─────────────────────────────────────────────────────────────────┐
│ Layer 1: UI Validation                                          │
│ - Immutable safety preamble (locked in editor)                  │
│ - Real-time run status via agent-manager WebSocket              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 2: Test-Genie API (spawn orchestration)                   │
│ - Server-generated preamble (appended to prompts)               │
│ - Batch spawning via agent-manager Tasks + Runs                 │
│ - Model and concurrency validation                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 3: Agent-Manager (execution management)                   │
│ - Profile-based configuration (tools, timeouts, permissions)    │
│ - Task and Run lifecycle management                             │
│ - Runner coordination (Claude Code execution)                   │
│ - WebSocket events for real-time status                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 4: Claude Code Runner                                     │
│ - Tool-level restrictions                                       │
│ - Working directory scope                                       │
│ - Permission prompts for operations                             │
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

### 1. Safety Preamble (Context Attachment)

Each agent spawn includes a safety preamble that is appended to the prompt. This preamble specifies:
- Working directory (scenario path)
- Allowed scope paths
- Maximum files that can be modified
- Maximum bytes that can be written
- Network access status
- Allowed bash commands

The preamble is generated server-side by test-genie and cannot be modified by clients.

### 2. Agent Profile Configuration

Agent-manager profiles define the security constraints for execution:
- **Allowed Tools**: Read, Write, Edit, Glob, Grep, Bash
- **Skip Permissions**: false (agents must request permission for operations)
- **Timeout**: 15 minutes maximum execution time
- **Max Turns**: 50 conversation turns limit

### 3. Bash Command Allowlist

The safety preamble specifies which bash commands are allowed:
- **Test runners**: `pnpm test`, `go test`, `vitest`, `jest`, `bats`, `pytest`
- **Build commands**: `pnpm build`, `go build`, `make`
- **Linters**: `eslint`, `prettier`, `gofmt`, `golangci-lint`
- **Safe inspection**: `ls`, `pwd`, `which`, `wc`, `diff`
- **Git read-only**: `git status`, `git diff`, `git log`, `git show`, `git branch`

All other bash commands are blocked by the preamble instructions.

### 4. Batch Spawning with Tags

Test-genie uses batch spawning with unique tags:
- Each batch gets a UUID (e.g., `test-genie-{batchId}-{index}`)
- Tags enable tracking and stopping specific runs
- Batch status queries use tag prefix filtering

## Operational Security

### Monitoring

- All agent operations are logged
- WebSocket broadcasts agent status changes in real-time (via agent-manager)
- Active agents panel shows running agents and their status
- Run events available for detailed execution tracking

### Stopping Agents

- Individual agents can be stopped via UI or API
- "Stop All" button terminates all running test-genie agents
- Agent-manager handles graceful shutdown of runners

### Cleanup

- Agent-manager manages run retention and cleanup
- Completed runs remain queryable for debugging

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
| `AGENT_MANAGER_ENABLED` | `true` | Enable agent-manager integration |
| `AGENT_MANAGER_PROFILE_KEY` | `test-genie` | Profile key for test-genie agents |
| `VROOLI_ROOT` | `$HOME/Vrooli` | Repository root for path validation |

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/agents/spawn` | POST | Spawn new agents (batch) |
| `/api/v1/agents/active` | GET | List active agents |
| `/api/v1/agents/{id}` | GET | Get specific agent details |
| `/api/v1/agents/{id}/stop` | POST | Stop specific agent |
| `/api/v1/agents/stop-all` | POST | Stop all test-genie agents |
| `/api/v1/agents/blocked-commands` | GET | Get security info |
| `/api/v1/agents/containment-status` | GET | Get containment status |
| `/api/v1/agents/status` | GET | Get agent-manager connection status |
| `/api/v1/agents/ws-url` | GET | Get agent-manager WebSocket URL |

## References

- [Agent-Manager Service](https://github.com/vrooli/vrooli/tree/master/scenarios/agent-manager)
- [Test-Genie API Handlers](../../api/internal/app/httpserver/agent_handlers.go)
- [Agent-Manager Client](../../api/agentmanager/client.go)
