# Tunable Levers: Agents, Containment & Security Configuration

This document describes the tunable configuration levers for the test-genie agent orchestration, containment, and security systems. These levers allow operators to adjust behavior without code changes, via environment variables.

## Design Principles

1. **Safe defaults**: All defaults are production-ready. You don't need to configure anything to get started.
2. **Bounded ranges**: All numeric values are clamped to safe ranges to prevent misconfiguration.
3. **Clear tradeoffs**: Each lever documents what happens when you increase or decrease it.
4. **No silent failures**: Invalid values are clamped, not rejected, to prevent startup failures.
5. **Validation reporting**: Configs provide warnings when non-default security settings are detected.

## Agent Configuration

These levers control agent lifecycle, coordination, and execution limits.

### Environment Variables

| Variable | Range | Default | Description |
|----------|-------|---------|-------------|
| `AGENT_LOCK_TIMEOUT_MINUTES` | 5-120 | 20 | How long scope locks remain valid before expiring |
| `AGENT_HEARTBEAT_INTERVAL_MINUTES` | 1-LockTimeout/2 | LockTimeout/4 | How often agents renew their locks |
| `AGENT_MAX_HEARTBEAT_FAILURES` | 1-10 | 3 | Consecutive failures before agent is stopped |
| `AGENT_RETENTION_DAYS` | 1-365 | 7 | Days to keep completed agent records |
| `AGENT_CLEANUP_INTERVAL_MINUTES` | 15-1440 | 60 | How often background cleanup runs |
| `AGENT_MAX_PROMPTS` | 1-100 | 20 | Max prompts per spawn request |
| `AGENT_MAX_CONCURRENT` | 1-20 | 10 | Max parallel agents per spawn |
| `AGENT_DEFAULT_CONCURRENCY` | 1-MaxConcurrent | 3 | Default concurrency if not specified |
| `AGENT_SPAWN_SESSION_TTL_MINUTES` | 5-480 | 30 | How long spawn sessions last |
| `AGENT_IDEMPOTENCY_TTL_MINUTES` | 5-480 | 30 | How long idempotency keys are cached |

### Tradeoff Guide

#### Lock Timeout (`AGENT_LOCK_TIMEOUT_MINUTES`)
- **Higher (e.g., 60)**: Agents can run longer tasks without heartbeat pressure. Use for long-running test suites.
- **Lower (e.g., 10)**: Faster recovery when agents crash/hang. Use when quick iteration matters more than long tasks.

#### Retention Days (`AGENT_RETENTION_DAYS`)
- **Higher (e.g., 30)**: More history for debugging and auditing.
- **Lower (e.g., 3)**: Smaller database, faster queries.

#### Max Concurrent Agents (`AGENT_MAX_CONCURRENT`)
- **Higher (e.g., 15)**: Faster batch completion but more CPU/memory pressure.
- **Lower (e.g., 5)**: Gentler on resources, better for constrained environments.

## Containment Configuration

These levers control OS-level isolation for agent execution.

### Environment Variables

| Variable | Range | Default | Description |
|----------|-------|---------|-------------|
| `CONTAINMENT_DOCKER_IMAGE` | any | ubuntu:22.04 | Docker image for containerized execution |
| `CONTAINMENT_MAX_MEMORY_MB` | 256-16384 | 2048 | Memory limit per agent container |
| `CONTAINMENT_MAX_CPU_PERCENT` | 50-800 | 200 | CPU limit (100 = 1 core, 200 = 2 cores) |
| `CONTAINMENT_AVAILABILITY_TIMEOUT_SECONDS` | 1-30 | 5 | Timeout for Docker availability check |
| `CONTAINMENT_PREFER_DOCKER` | true/false | true | Prefer Docker over other providers |
| `CONTAINMENT_ALLOW_FALLBACK` | true/false | true | Allow uncontained execution if no sandbox available |
| `CONTAINMENT_DROP_ALL_CAPABILITIES` | true/false | true | Drop all Linux capabilities in containers |
| `CONTAINMENT_NO_NEW_PRIVILEGES` | true/false | true | Prevent privilege escalation |
| `CONTAINMENT_READ_ONLY_ROOT_FS` | true/false | false | Make container root filesystem read-only |

### Tradeoff Guide

#### Docker Image (`CONTAINMENT_DOCKER_IMAGE`)
- Use a specific tag (not `latest`) for reproducibility.
- Larger images (e.g., with dev tools) enable more agent capabilities.
- Smaller images start faster but may lack tools agents need.

#### Memory Limit (`CONTAINMENT_MAX_MEMORY_MB`)
- **Higher (e.g., 4096)**: Agents can handle larger codebases/operations.
- **Lower (e.g., 1024)**: Better isolation, prevents runaway memory usage.

#### CPU Limit (`CONTAINMENT_MAX_CPU_PERCENT`)
- **Higher (e.g., 400)**: Faster execution, especially for parallel tests.
- **Lower (e.g., 100)**: Gentler on host system, better for shared environments.

#### Allow Fallback (`CONTAINMENT_ALLOW_FALLBACK`)
- **true**: Test-genie works even without Docker, but with reduced security.
- **false**: Fail hard if no containment available. Use in production.

## Security Configuration

These levers control bash command validation, prompt scanning, and extensibility.

### Environment Variables

| Variable | Range | Default | Description |
|----------|-------|---------|-------------|
| `SECURITY_EXTRA_ALLOWED_BASH_COMMANDS` | comma-separated | (empty) | Additional bash command prefixes to allow |
| `SECURITY_PROMPT_VALIDATION_STRICT` | true/false | true | Enable stricter prompt pattern scanning |
| `SECURITY_MAX_PROMPT_LENGTH` | 1000-1000000 | 100000 | Maximum prompt length in characters |
| `SECURITY_ALLOW_GLOB_PATTERNS` | true/false | true | Allow glob patterns in allowed bash commands |

### Tradeoff Guide

#### Extra Allowed Bash Commands (`SECURITY_EXTRA_ALLOWED_BASH_COMMANDS`)
- **More commands**: Agents have more flexibility to run scenario-specific operations.
- **Fewer commands**: Tighter security, but agents may be blocked from necessary operations.
- **Format**: Comma-separated command prefixes (e.g., `cargo test,cargo build,make deploy`)
- **SECURITY NOTE**: These commands are ADDED to the built-in allowlist, not replacements. Be careful what you add.

#### Prompt Validation Strict (`SECURITY_PROMPT_VALIDATION_STRICT`)
- **true (default)**: More patterns are flagged as potentially dangerous.
- **false**: Only the most obvious dangerous patterns are flagged. Use for testing or when false positives are problematic.

#### Max Prompt Length (`SECURITY_MAX_PROMPT_LENGTH`)
- **Higher (e.g., 500000)**: Agents can receive more context, useful for large codebases.
- **Lower (e.g., 50000)**: Limits prompt injection attack surface.

#### Allow Glob Patterns (`SECURITY_ALLOW_GLOB_PATTERNS`)
- **true (default)**: Glob patterns are allowed only for safe commands (test runners, `ls`).
- **false**: All glob patterns are rejected. Use for maximum security.

### Built-in Bash Command Allowlist

The following bash commands are always allowed (prefixes):
- **Test runners**: `pnpm test`, `npm test`, `go test`, `vitest`, `jest`, `bats`, `make test`, `pytest`
- **Build commands**: `pnpm build`, `npm run build`, `go build`, `make`
- **Linters/formatters**: `eslint`, `prettier`, `gofmt`, `gofumpt`, `golangci-lint`
- **Type checking**: `tsc`, `pnpm typecheck`
- **Safe inspection**: `ls`, `pwd`, `which`, `wc`, `diff`
- **Git read-only**: `git status`, `git diff`, `git log`, `git show`, `git branch`, `git remote`

### Blocked Patterns (Defense-in-Depth)

These patterns are blocked in prompts regardless of command allowlist:
- Git destructive: `git checkout`, `git reset --hard`, `git push`, `git commit`
- Filesystem: `rm`, `mv`, `cp`, `mkdir`, `rmdir`, `chmod`, `chown`
- System: `sudo`, `su`, `systemctl`, `kill`, `pkill`
- Database: `DROP`, `TRUNCATE`, `DELETE FROM`, `UPDATE SET`, `INSERT INTO`
- Package managers: `npm install`, `pnpm add`, `go get`, `pip install`
- Remote execution: `curl ... | sh`, `wget ... | sh`
- Shell execution: `eval`, `exec`, `source`

## Programmatic Configuration

All three configs can be set programmatically for testing:

```go
// Agent config
cfg := agents.DefaultConfig()
cfg.LockTimeoutMinutes = 30
cfg.MaxConcurrentAgents = 5

svc := agents.NewAgentService(repo, agents.WithConfig(cfg))

// Containment config
ccfg := containment.DefaultConfig()
ccfg.DockerImage = "custom:image"
ccfg.MaxMemoryMB = 4096

provider := containment.NewDockerProvider(containment.WithContainmentConfig(ccfg))

// Security config
scfg := security.LoadSecurityConfigFromEnv()
scfg.ExtraAllowedBashCommands = []security.AllowedBashCommand{
    {Prefix: "cargo test", Description: "Rust test runner"},
}

validator := security.NewBashCommandValidator(security.WithSecurityConfig(scfg))
```

## Validation Reporting

All configs provide validation reports that warn about non-default or potentially insecure settings:

```go
// Agent config validation
agentResult := cfg.ValidateWithReport()
for _, warning := range agentResult.Warnings {
    log.Printf("Agent config warning: %s", warning)
}

// Containment config validation
containmentResult := ccfg.ValidateWithReport()
for _, warning := range containmentResult.Warnings {
    log.Printf("Containment config warning: %s", warning)
}

// Security config validation
securityWarnings, err := scfg.Validate()
for _, warning := range securityWarnings {
    log.Printf("Security config warning: %s", warning)
}
```

## Monitoring

### Agent Metrics
- Active agents: `GET /api/v1/agents`
- Active locks: included in agent list response
- Configuration: `GET /api/v1/agents/blocked-commands` includes safe defaults

### Containment Status
- Current provider: `GET /api/v1/agents/containment-status`
- Security level: 0 (none) to 7 (Docker)
- Available providers: lists what's detected on the system

## Examples

### Development Environment
```bash
# More permissive for faster iteration
export AGENT_LOCK_TIMEOUT_MINUTES=10
export AGENT_RETENTION_DAYS=1
export CONTAINMENT_ALLOW_FALLBACK=true

# Allow additional commands for Rust development
export SECURITY_EXTRA_ALLOWED_BASH_COMMANDS="cargo test,cargo build,cargo clippy"
```

### Production Environment
```bash
# Stricter security, longer retention
export AGENT_LOCK_TIMEOUT_MINUTES=30
export AGENT_RETENTION_DAYS=30
export CONTAINMENT_ALLOW_FALLBACK=false
export CONTAINMENT_MAX_MEMORY_MB=4096

# Tighter security controls
export SECURITY_PROMPT_VALIDATION_STRICT=true
export SECURITY_MAX_PROMPT_LENGTH=50000
```

### CI/CD Environment
```bash
# Fast startup, minimal resources
export AGENT_MAX_CONCURRENT=2
export AGENT_LOCK_TIMEOUT_MINUTES=15
export CONTAINMENT_AVAILABILITY_TIMEOUT_SECONDS=2

# Disable glob patterns in CI for predictability
export SECURITY_ALLOW_GLOB_PATTERNS=false
```

### High-Security Environment
```bash
# Maximum isolation
export CONTAINMENT_ALLOW_FALLBACK=false
export CONTAINMENT_DROP_ALL_CAPABILITIES=true
export CONTAINMENT_NO_NEW_PRIVILEGES=true
export CONTAINMENT_READ_ONLY_ROOT_FS=true

# Strict agent limits
export AGENT_DEFAULT_MAX_FILES=20
export AGENT_DEFAULT_MAX_BYTES=524288  # 512KB
export AGENT_DEFAULT_NETWORK_ENABLED=false

# Strict security controls
export SECURITY_PROMPT_VALIDATION_STRICT=true
export SECURITY_MAX_PROMPT_LENGTH=25000
export SECURITY_ALLOW_GLOB_PATTERNS=false
```
