# Scenario-to-Cloud Architectural Seams

This document describes the responsibility boundaries and seams in the scenario-to-cloud codebase, providing guidance on where to add or modify behavior.

## Domain Overview

Scenario-to-Cloud is a deployment orchestrator that:
1. Validates and normalizes deployment manifests
2. Builds minimal "mini-Vrooli" bundles containing only required components
3. Runs preflight checks on VPS targets
4. Transfers bundles and executes VPS setup
5. Deploys and monitors scenarios on remote VPS targets

## Responsibility Zones

### 1. Entry/Presentation Layer

**Location:** `handlers_*.go` files (main package)

| File | Responsibility |
|------|---------------|
| `handlers_bundle.go` | Bundle operations: build, list, stats, cleanup, manifest validation |
| `handlers_deployment.go` | Deployment CRUD and execution orchestration |
| `handlers_vps_operations.go` | VPS setup/deploy/inspect plan and apply endpoints |
| `handlers_vps_management.go` | VPS management actions (reboot, stop, cleanup levels) |
| `handlers_live_state.go` | Real-time VPS state inspection (files, processes, drift) |
| `handlers_edge.go` | Edge/TLS management (DNS checks, Caddy control) |
| `handlers_history.go` | Deployment history and log retrieval |
| `handlers_terminal.go` | WebSocket terminal access |
| `handlers_investigation.go` | Agent-assisted deployment investigation |
| `handlers_docs.go` | Documentation serving |
| `handlers_progress.go` | SSE-based deployment progress streaming |
| `handlers_tasks.go` | Task management endpoints |
| `handlers_health.go` | Health check endpoints |
| `vps/preflight/handlers.go` | Preflight checks and fix actions (port cleanup, disk management) |
| `secrets/handlers.go` | Secret retrieval for deployments |
| `secrets/handlers_management.go` | Secret management operations |
| `ssh/handlers.go` | SSH key management and connection testing |

**Pattern:** Handlers decode requests, delegate to domain/service logic, encode responses.
All SSH handlers use the factory pattern (`HandleXxx(dep) http.HandlerFunc`) and accept their dependencies explicitly (e.g., `*KeyService`, `KeyCopier`, `Runner`).

### 2. Domain Types

**Location:** `domain/` package

The `domain/` package contains all core domain types, organized by concern:

| File | Responsibility |
|------|---------------|
| `domain/deployment.go` | Deployment entity, status enum, history events, request/response DTOs |
| `domain/investigation.go` | Investigation entity and related types |
| `domain/manifest.go` | CloudManifest and all nested types (target, scenario, bundle, edge, secrets), validation types |
| `domain/vps_state.go` | VPS live state types (processes, ports, system resources, Caddy state) |
| `domain/bundle.go` | Bundle artifact, statistics, and request/response DTOs |
| `domain/preflight.go` | Preflight check types (status, check, response) |
| `domain/vps.go` | VPS operation types (plan steps, setup/deploy/inspect/stop results) |

**Pattern:** Domain types are pure data structures with no business logic. They are imported via type aliases in the main package for backward compatibility:

```go
// In bundle.go
type BundleArtifact = domain.BundleArtifact
```

### 3. Domain Logic / Business Rules

| File | Responsibility |
|------|---------------|
| `manifest/` | Manifest validation and normalization rules (uses `domain.CloudManifest`) |
| `bundle/` | Bundle building rules, file inclusion/exclusion logic |
| `vps/preflight/runner.go` | Preflight check orchestration (`preflight.Run`) |
| `vps/preflight/checks.go` | Individual preflight check definitions |
| `vps/preflight/credentials.go` | Credential validation for preflight |
| `vps/deploy.go` | Deployment execution logic, secret validation |
| `vps/setup.go` | VPS setup step orchestration |
| `vps/inspect.go` | VPS inspection logic |
| `vps/stop.go` | Scenario stopping logic |
| `vps/live_state.go` | Live VPS state collection via SSH (includes output parsing) |
| `vps/step_config.go` | Per-step execution parameters and timeouts |
| `vps/progress.go` | VPS deployment progress tracking |
| `vps/tls.go` | TLS verification steps |
| `vps/commands.go` | VPS command construction helpers |
| `scenarios.go` | Scenario discovery and dependency resolution |

### 4. Orchestration / Coordination

| File | Responsibility |
|------|---------------|
| `main.go` | Server initialization, route wiring, middleware |
| `investigation/service.go` | Agent-manager integration for deployment investigation |
| `deployment/progress.go` | Deployment progress tracking types |
| `deployment/hub.go` | SSE hub for broadcasting progress updates |
| `deployment/orchestrator.go` | Deployment pipeline orchestration |
| `deployment/manifest_refresh.go` | Manifest refresh logic for redeployments |

### 5. Integration / Infrastructure

| File | Responsibility |
|------|---------------|
| `persistence/repository.go` | Database schema initialization |
| `persistence/deployment.go` | Deployment CRUD operations |
| `persistence/investigation.go` | Investigation CRUD operations |
| `agentmanager/client.go` | Agent-manager HTTP client |
| `agentmanager/service.go` | Agent-manager integration service |
| `secrets/client.go` | Secrets-manager integration (`secrets.Fetcher` interface, `secrets.Client` impl) |
| `secrets/generator.go` | Secret value generation (`secrets.GeneratorFunc` interface) |
| `secrets/writer.go` | Secret file writing |
| `secrets/handlers.go` | Secret retrieval HTTP handlers |
| `secrets/handlers_management.go` | Secret management HTTP handlers |
| `internal/shellutil/shell.go` | Shell quoting (`QuoteSingle`), `VrooliCommand`, `SafeRemoteJoin`, `ValidateTildeExpansion` |
| `internal/stringutil/strings.go` | String utility functions |

### 6. SSH Package

**Location:** `ssh/` package

The `ssh/` package provides SSH and SCP execution infrastructure for VPS operations:

| File | Responsibility |
|------|---------------|
| `ssh/doc.go` | Package documentation |
| `ssh/config.go` | SSH connection config, defaults (port 22, user "root"), `Result` type, `nowTimestamp()` |
| `ssh/options.go` | SSH/SCP option structs (`RunOptions`, `SCPOptions`, `HandlerOptions`), argument assembly (`BuildSSHArgs`, `BuildSCPArgs`) |
| `ssh/runner.go` | `Runner` and `SCPRunner` interfaces, `ExecRunner`/`ExecSCPRunner` impls, `runSSH` helper, `boundedBuffer` |
| `ssh/connect.go` | Connection testing (`TestConnection`), `IsIPv6` |
| `ssh/errors.go` | SSH error classification (`ClassifyError`, `newCommandError`), `ErrorInfoFromSSHError` bridge to domain types |
| `ssh/keys.go` | `KeyService` struct + methods: `DiscoverKeys`, `ReadPublicKey`, `DeleteKey`, `parseKeyFile`, `getKeyFingerprint` |
| `ssh/keys_generate.go` | `KeyService.GenerateKey` method via `ssh-keygen` |
| `ssh/keys_copy.go` | `KeyCopier` interface, `ExecKeyCopier` impl via `golang.org/x/crypto/ssh` password auth |
| `ssh/command.go` | `CommandRunner` interface + `ExecCommandRunner` impl for local command abstraction |
| `ssh/handlers.go` | HTTP handler factories — all accept explicit dependencies (`*KeyService`, `KeyCopier`, `Runner`) |
| `ssh/types.go` | Domain types: `KeyType`, `KeyInfo` |
| `ssh/dto.go` | HTTP request/response DTOs |
| `ssh/path.go` | Path utilities: `GetSSHDir`, `ValidateSSHPath`, `ValidateKeyFilename`, `ExpandPath` |
| `ssh/format.go` | Command formatting for display/logging (uses `shellutil.QuoteSingle`) |

### 7. Cross-Cutting Concerns

| File | Responsibility |
|------|---------------|
| `internal/shellutil/shell.go` | Shell quoting (`QuoteSingle`), `VrooliCommand`, `SafeRemoteJoin`, `ValidateTildeExpansion` |
| `internal/stringutil/strings.go` | String utility functions |
| `internal/httputil/decode.go` | Generic JSON request decoding |
| `internal/httputil/response.go` | HTTP response writing helpers |
| `http_helpers.go` | JSON response writing, error responses |

## Testability Seams (Interfaces for Substitution)

The codebase uses interface-based seams to enable testing without requiring live external services.

### SSH/SCP Operations

**Location:** `ssh/runner.go`

| Interface | Implementation | Purpose |
|-----------|---------------|---------|
| `ssh.Runner` | `ssh.ExecRunner` | Execute commands on remote VPS via SSH |
| `ssh.SCPRunner` | `ssh.ExecSCPRunner` | Copy files to remote VPS via SCP |

**Usage in tests:** Use `FakeSSHRunner` / `FakeSCPRunner` from `test_fakes.go`:

```go
sshFake := &FakeSSHRunner{Responses: map[string]ssh.Result{
    "echo ok": {ExitCode: 0, Stdout: "ok"},
}}
```

### Local Command Execution (ssh-keygen)

**Location:** `ssh/command.go`

| Interface | Implementation | Purpose |
|-----------|---------------|---------|
| `ssh.CommandRunner` | `ssh.ExecCommandRunner` | Execute local commands (e.g., `ssh-keygen`) |

**Usage in tests:** Pass a fake `CommandRunner` to `ssh.NewKeyService(fake, tmpDir)`:

```go
keySvc := ssh.NewKeyService(myFake, t.TempDir())
```

### Key Copying

**Location:** `ssh/keys_copy.go`

| Interface | Implementation | Purpose |
|-----------|---------------|---------|
| `ssh.KeyCopier` | `ssh.ExecKeyCopier` | Copy SSH public key to remote server via password auth |

**Usage in tests:** Implement `KeyCopier` to return test responses without network calls.

### DNS Resolution

**Location:** `dns/` package (`dns.Service`, `dns.Resolver`)

The DNS service exposes a resolver seam for DNS lookups and encapsulates
comparisons + hints used by preflight and edge checks.

**Usage in tests:** Use `FakeResolver` from `test_fakes.go` (or a local test fake)
to control DNS responses.

### Secrets Fetching

**Location:** `secrets/client.go`

| Interface | Implementation | Purpose |
|-----------|---------------|---------|
| `secrets.Fetcher` | `secrets.Client` | Fetch secrets from secrets-manager service |

**Methods:**
- `FetchBundleSecrets(ctx, scenario, tier, resources)` - Retrieve secrets manifest
- `HealthCheck(ctx)` - Verify secrets-manager is reachable

**Usage in tests:** Implement `secrets.Fetcher` to return test secrets without HTTP calls.

### Secrets Generation

**Location:** `secrets/generator.go`

| Interface | Implementation | Purpose |
|-----------|---------------|---------|
| `secrets.GeneratorFunc` | `secrets.Generator` | Generate per-install secrets using crypto/rand |

**Methods:**
- `GenerateSecrets(plans)` - Generate secret values for per_install_generated class

**Usage in tests:** Implement `secrets.GeneratorFunc` to return deterministic values.

### Progress Tracking

**Location:** `vps/progress.go` and `deployment/progress.go`

The `ProgressRepo` interface abstracts deployment progress persistence:
```go
type ProgressRepo interface {
    UpdateDeploymentProgress(ctx context.Context, id, step string, percent float64) error
}
```

`ProgressEvent` carries structured error metadata (`ErrorCategory`, `Retryable`, `Hint`) when a step fails with a classifiable SSH error. The `deployment.Event` mirrors these fields and the `progressHubAdapter` copies them through.

### Error Classification Bridge

**Location:** `ssh/errors.go`

`ErrorInfoFromSSHError(*SSHError) *domain.ErrorInfo` converts an `*ssh.SSHError` into a `*domain.ErrorInfo` suitable for JSON API responses. The `failStep` closures in `vps/setup.go` and `vps/deploy.go` use `errors.As` to extract `*ssh.SSHError` from any error and populate `ErrorInfo` on the result types.

## Key Seams (Where to Make Changes)

### Adding a New Domain Type
1. Add types to appropriate file in `domain/` package
2. Add type alias in the main package file that uses them
3. Update this SEAMS.md to document the new types

### Adding a New Deployment Phase
1. Define step types in the relevant `vps/*.go` file
2. Add handler(s) in appropriate `handlers_*.go`
3. Update route registration in `main.go`

### Adding a New VPS Action
1. Add action type to `handlers_vps_management.go`
2. Add confirmation validation
3. Add command builder function

### Adding a New Preflight Check
1. Add check logic to `preflight.Run` in `vps/preflight/runner.go`
2. Use pass/warn/fail closures for consistent output

### Modifying Manifest Schema
1. Update types in `domain/manifest.go`
2. Update validation in `manifest/` (`ValidateAndNormalizeManifest`)
3. Update tests in `manifest_contract_test.go`

### Adding Bundle Contents
1. Update bundling rules in `bundle/`
2. Update tests in `bundling_rules_test.go`

### Adding a Database Table
1. Add migration in `persistence/repository.go` InitSchema
2. Add CRUD methods in new or existing `persistence/*.go` file
3. Add domain types in `domain/*.go`

### Adding a New Integration Seam
When adding code that talks to external services:
1. Define an interface describing the operations needed
2. Create a concrete implementation using real HTTP/SSH/etc.
3. Add `var _ Interface = (*Implementation)(nil)` compile-time check
4. Accept the interface (not concrete type) in consuming functions
5. Document the seam in this file

## Dependency Direction

```
handlers_*.go (main package)
      │
      ├──► domain/* (types, DTOs)
      │
      ├──► manifest/ (validation)
      │
      ├──► vps/ (business logic: deploy, setup, inspect, stop, live_state)
      │    └──► vps/preflight/ (preflight checks)
      │
      ├──► bundle/ (bundling logic)
      │
      ├──► deployment/ (orchestration, progress hub)
      │
      ├──► secrets/ (fetching, generation, writing)
      │
      ├──► ssh/ (SSH/SCP execution, key management)
      │
      ├──► investigation/ (agent investigation)
      │
      └──► persistence/* (data access)
               │
               └──► domain/* (entities)
```

**Import hierarchy:**
- `domain/` has no internal dependencies (leaf package)
- Main package imports `domain/` and re-exports types via aliases
- `persistence/` imports `domain/` for entity types
- `agentmanager/` is self-contained integration code
- `ssh/`, `secrets/`, `vps/`, `deployment/`, `investigation/` import `domain/` and relevant internal packages
- `internal/shellutil/` and `internal/stringutil/` are leaf utility packages

## File Size Indicators

Files over 500 lines that may benefit from further splitting:

| File | Lines | Notes |
|------|-------|-------|
| `bundle/builder.go` | ~1070 | Consider separating stats/cleanup into `bundle_cleanup.go` |
| `handlers_live_state.go` | ~700 | Reduced after type extraction to domain; cohesive |
| `investigation/service.go` | ~847 | Complex orchestration, may be acceptable |

## Anti-Patterns to Avoid

1. **Don't add business logic to handlers** - Handlers should only decode/encode and delegate
2. **Don't add HTTP concerns to domain logic** - vps/*.go should not import net/http
3. **Don't scatter SSH commands** - Use `ssh.Runner` / `ssh.SCPRunner` interfaces via `ssh/runner.go`
4. **Don't duplicate manifest validation** - Always go through ValidateAndNormalizeManifest
5. **Don't define domain types outside domain/** - Use type aliases in main package for backward compatibility
6. **Don't import main package from domain/** - domain/ must remain a leaf package

## Recent Architectural Changes

### Domain Type Consolidation (2025-01)
Moved domain types into the `domain/` package for consistency:
- `domain/manifest.go` - CloudManifest and related types (from manifest.go)
- `domain/vps_state.go` - VPS state types (from vps_live_state.go)
- `domain/bundle.go` - Bundle types (from bundle.go)

Type aliases in the main package preserve backward compatibility.

### Boundary-of-Responsibility Enforcement (2026-01)
Extended domain type consolidation following the screaming-architecture-audit and boundary-of-responsibility-enforcement guides:

**New domain files:**
- `domain/preflight.go` - Preflight check types (PreflightCheckStatus, PreflightCheck, PreflightResponse)
- `domain/vps.go` - VPS operation result types (VPSPlanStep, VPSSetupResult, VPSDeployResult, VPSInspectResult, VPSStopResult, MissingSecretInfo)

**Moved from handlers to domain:**
- `domain/bundle.go` - Added bundle request/response DTOs (BundleCleanupRequest, BundleCleanupResponse, BundleStatsResponse, BundleDeleteResponse, VPSBundleListRequest, VPSBundleInfo, VPSBundleListResponse, VPSBundleDeleteRequest, VPSBundleDeleteResponse)

**Updated main package files:**
- `preflight.go` - Now uses type aliases from domain.PreflightCheckStatus, etc.
- `vps/setup.go` - Now uses type aliases from domain.VPSPlanStep, domain.VPSSetupResult
- `vps/deploy.go` - Now uses type aliases from domain.VPSDeployResult, domain.MissingSecretInfo
- `handlers_bundle.go` - Now uses type aliases from domain for all DTOs

**Pattern established:**
Handler files now use the pattern:
```go
type (
    BundleCleanupRequest = domain.BundleCleanupRequest
    // etc.
)
```

### Testability Interface Extraction (2026-01)
Added explicit interfaces for external integrations to enable testing without live services:

**New interfaces:**
- `secrets.Fetcher` in `secrets/client.go` - Abstracts secrets-manager HTTP calls
- `secrets.GeneratorFunc` in `secrets/generator.go` - Abstracts crypto/rand secret generation

**Pattern:**
```go
// Interface definition
type Fetcher interface {
    FetchBundleSecrets(ctx context.Context, scenario, tier string, resources []string) (*SecretsManagerResponse, error)
    HealthCheck(ctx context.Context) error
}

// Compile-time interface check
var _ Fetcher = (*Client)(nil)
```

**Existing seams preserved:**
- `ssh.Runner` / `ssh.SCPRunner` for VPS command execution (already well-used in tests)
- `dns.Service` / `dns.Resolver` for DNS lookups and comparisons
- `ProgressRepo` for deployment progress persistence

### Server-Level Dependency Injection (2026-01-03)
Promoted seams from inline instantiation to proper dependency injection via the `Server` struct:

**Server struct now holds seams:**
```go
type Server struct {
    // ... existing fields ...

    // Seam: SSH command execution (defaults to ssh.ExecRunner)
    sshRunner ssh.Runner
    // Seam: SCP file transfer (defaults to ssh.ExecSCPRunner)
    scpRunner ssh.SCPRunner
    // Seam: Secrets fetching (defaults to secrets.NewClient())
    secretsFetcher secrets.Fetcher
    // Seam: Secrets generation (defaults to secrets.NewGenerator())
    secretsGenerator secrets.GeneratorFunc
    // Seam: DNS services (defaults to dns.NewService(dns.NetResolver{}, dns.WithTimeout(...)))
    dnsService dns.Service
    // Seam: TLS probe service (defaults to tlsinfo.NewService(...))
    tlsService tlsinfo.Service
    // Seam: TLS ALPN runner (defaults to tlsinfo.DefaultALPNRunner)
    tlsALPNRunner tlsinfo.ALPNRunner
    // Seam: Deployment repository (defaults to persistence.Repository)
    deploymentRepo DeploymentRepository
}
```

**Handler migration complete - all handlers now use Server seams:**
- `handlers_deployment.go`: Uses `s.sshRunner`, `s.scpRunner`, `s.secretsFetcher`, `s.secretsGenerator`
- `vps/preflight/handlers.go`: Uses `s.sshRunner`, `s.dnsService`
- `handlers_bundle.go`: Uses `s.sshRunner`
- `handlers_live_state.go`: Uses `s.sshRunner`
- `handlers_vps_management.go`: Uses `s.sshRunner`
- `handlers_edge.go`: Uses `s.sshRunner`, `s.dnsService`

**Testing pattern:**
```go
// In tests, create a Server with fake seams:
srv := &Server{
    sshRunner:        &FakeSSHRunner{Responses: map[string]ssh.Result{...}},
    scpRunner:        &FakeSCPRunner{},
    secretsFetcher:   &FakeSecretsFetcher{Response: testSecrets},
    secretsGenerator: &FakeSecretsGenerator{Values: map[string]string{"key": "deterministic-value"}},
    dnsService:      dns.NewService(&dns.FakeResolver{Hosts: map[string][]string{"example.com": {"1.2.3.4"}}}),
    // ... other fields ...
}
```

**Seam migration status: COMPLETE**
All handlers that use external integrations (SSH, SCP, DNS, secrets) now obtain these dependencies from the Server struct rather than creating instances inline. This makes the entire API testable without live external services.

### Reusable Test Fakes (2026-01-03)

Added `test_fakes.go` with reusable fake implementations for all seams:

| Fake | Interface | Purpose |
|------|-----------|---------|
| `FakeResolver` | `dns.Resolver` | Control DNS lookup responses |
| `FakeSSHRunner` | `ssh.Runner` | Control SSH command responses, track calls |
| `FakeSCPRunner` | `ssh.SCPRunner` | Control SCP copy results, track calls |
| `FakeSecretsFetcher` | `secrets.Fetcher` | Control secrets-manager responses |
| `FakeSecretsGenerator` | `secrets.GeneratorFunc` | Return deterministic secret values |

**Features of all fakes:**
- Thread-safe call recording via `Calls` field
- Configurable error injection via `Err`/`Errs` fields
- Default behaviors for unspecified cases
- Compile-time interface verification via `var _ Interface = (*Fake)(nil)`

**Example with call verification:**
```go
sshFake := &FakeSSHRunner{Responses: map[string]ssh.Result{
    "echo ok": {ExitCode: 0, Stdout: "ok"},
}}
// ... run handler ...
if len(sshFake.Calls) != 1 || sshFake.Calls[0] != "echo ok" {
    t.Errorf("unexpected SSH calls: %v", sshFake.Calls)
}
```

## Future Improvements

### Local Command Execution Seam

**Location:** `scenarios.go`, `handlers_edge.go`

There are a few remaining inline `exec.Command` usages for local operations:

| File | Function | Purpose | Priority |
|------|----------|---------|----------|
| `scenarios.go:553` | `discoverScenarioPort` | Runs `vrooli scenario port` to discover ports | Low |
| `handlers_edge.go:456` | `runLocalCommand` | Runs local bash commands for edge operations | Low |

**Why low priority:**
- These are local-only operations (not remote VPS commands)
- They're used infrequently and for discovery/utility purposes
- The primary external integration seams (SSH/SCP/DNS/secrets) are fully migrated

**Potential seam design:**
```go
type LocalCommandRunner interface {
    Run(ctx context.Context, cmd string) (stdout string, err error)
}
```

This would allow tests to control local command execution, useful for:
- Testing port discovery without running vrooli CLI
- Testing edge operations without bash execution

**Recommendation:** Extract if tests require controlling local command behavior, or if local command failures become a testing pain point.

### Analyzer Client Seam

**Location:** `scenarios.go:428`

The `GetScenarioRequiredResources` function creates an inline `http.Client` to call the analyzer API.

**Current pattern:**
```go
client := &http.Client{Timeout: 5 * time.Second}
resp, err := client.Do(req)
```

**Potential seam design:**
```go
type AnalyzerClient interface {
    Analyze(ctx context.Context, scenarioID string) (*AnalyzerResponse, error)
}
```

**Recommendation:** Extract if tests need to mock analyzer responses. Currently, analyzer is typically available during testing via the scenario lifecycle.

### Bundle Cleanup Separation
`bundle/builder.go` (~1070 lines) could be split:
- `bundle/builder.go` - Core bundling logic
- `bundle/cleanup.go` - Stats, cleanup, retention logic

### SSH Subsystem Refactoring (2026-02)

**Shell utility extraction:**
- Moved `QuoteSingle`, `VrooliCommand`, `SafeRemoteJoin`, `ValidateTildeExpansion` from `ssh/shell.go` to `internal/shellutil/shell.go`
- 15+ consumer files now import `shellutil` directly instead of `ssh` for string quoting
- Deleted `ssh/shell.go` and `ssh/shell_test.go`

**KeyService struct:**
- Replaced package-level functions + global `defaultCommandRunner` with `KeyService` struct
- `NewKeyService(cmd, sshDir)` accepts explicit `CommandRunner` and SSH directory for testing
- All key operations (`DiscoverKeys`, `GenerateKey`, `ReadPublicKey`, `DeleteKey`) are now methods
- Removed `SetCommandRunner` global state (test-race hazard eliminated)

**KeyCopier interface:**
- Added `KeyCopier` interface and `ExecKeyCopier` implementation in `ssh/keys_copy.go`
- `HandleCopyKey` now accepts `KeyCopier` dependency

**Handler DI consistency:**
- All SSH handlers now accept explicit dependencies: `HandleListKeys(ks)`, `HandleGenerateKey(ks)`, `HandleGetPublicKey(ks)`, `HandleDeleteKey(ks)`, `HandleCopyKey(copier)`, `HandleTestConnection(runner)`

**Error handling consolidation:**
- `ssh/errors.go` contains `ClassifyError` (exported) for SSH error classification and `ErrorInfoFromSSHError` for bridging to domain types
- `ssh/runner.go` contains `Runner`/`SCPRunner` interfaces, `ExecRunner`/`ExecSCPRunner` implementations, and `boundedBuffer`

**New tests:**
- `ValidateSSHPath` table-driven tests in `ssh/path_test.go`
- `boundedBuffer` and `exitCode` tests in `ssh/runner_test.go`
