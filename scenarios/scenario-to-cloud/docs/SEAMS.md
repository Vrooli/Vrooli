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

**Location:** `handlers_*.go` files

| File | Responsibility |
|------|---------------|
| `handlers_bundle.go` | Bundle operations: build, list, stats, cleanup, manifest validation |
| `handlers_deployment.go` | Deployment CRUD and execution orchestration |
| `handlers_preflight.go` | Preflight checks and fix actions (port cleanup, disk management) |
| `handlers_vps_operations.go` | VPS setup/deploy/inspect plan and apply endpoints |
| `handlers_vps_management.go` | VPS management actions (reboot, stop, cleanup levels) |
| `handlers_live_state.go` | Real-time VPS state inspection (files, processes, drift) |
| `handlers_edge.go` | Edge/TLS management (DNS checks, Caddy control) |
| `handlers_history.go` | Deployment history and log retrieval |
| `handlers_terminal.go` | WebSocket terminal access |
| `handlers_ssh.go` | SSH key management |
| `handlers_investigation.go` | Agent-assisted deployment investigation |
| `handlers_secrets.go` | Secret retrieval for deployments |
| `handlers_docs.go` | Documentation serving |
| `handlers_progress.go` | SSE-based deployment progress streaming |

**Pattern:** Handlers decode requests, delegate to domain/service logic, encode responses.

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
| `manifest.go` | Manifest validation and normalization rules (uses `domain.CloudManifest`) |
| `bundle.go` | Bundle building rules, file inclusion/exclusion logic |
| `preflight.go` | Preflight check definitions and execution |
| `vps_deploy.go` | Deployment execution logic, secret validation |
| `vps_setup.go` | VPS setup step orchestration |
| `vps_inspect.go` | VPS inspection logic |
| `vps_stop.go` | Scenario stopping logic |
| `vps_live_state.go` | Live VPS state collection via SSH |
| `scenarios.go` | Scenario discovery and dependency resolution |

### 4. Orchestration / Coordination

| File | Responsibility |
|------|---------------|
| `main.go` | Server initialization, route wiring, middleware |
| `investigation_service.go` | Agent-manager integration for deployment investigation |
| `progress.go` | Deployment progress tracking types |
| `progress_hub.go` | SSE hub for broadcasting progress updates |

### 5. Integration / Infrastructure

| File | Responsibility |
|------|---------------|
| `persistence/repository.go` | Database schema initialization |
| `persistence/deployment.go` | Deployment CRUD operations |
| `persistence/investigation.go` | Investigation CRUD operations |
| `agentmanager/client.go` | Agent-manager HTTP client |
| `agentmanager/service.go` | Agent-manager integration service |
| `vps_runners.go` | SSH/SCP command execution interfaces |
| `ssh_keys.go` | SSH key generation and management |
| `secrets_client.go` | Secrets-manager integration |
| `secrets_generator.go` | Secret value generation |
| `secrets_writer.go` | Secret file writing |
| `parsers.go` | VPS command output parsing (ps, ss, /proc/stat, Caddyfile) |

### 6. Cross-Cutting Concerns

| File | Responsibility |
|------|---------------|
| `http_helpers.go` | JSON response writing, error responses |
| `vps_utils.go` | SSH config, shell quoting, remote path utilities |

## Testability Seams (Interfaces for Substitution)

The codebase uses interface-based seams to enable testing without requiring live external services.

### SSH/SCP Operations

**Location:** `vps_runners.go`

| Interface | Implementation | Purpose |
|-----------|---------------|---------|
| `SSHRunner` | `ExecSSHRunner` | Execute commands on remote VPS via SSH |
| `SCPRunner` | `ExecSCPRunner` | Copy files to remote VPS via SCP |

**Usage in tests:** Use `fakeSSHRunner` (see `preflight_test.go`) to return predetermined responses.

```go
type fakeSSHRunner struct {
    responses map[string]SSHResult
    errs      map[string]error
}
```

### DNS Resolution

**Location:** Inline interface in `preflight.go`

The preflight function accepts a resolver interface:
```go
resolver interface {
    LookupHost(ctx context.Context, host string) ([]string, error)
}
```

**Usage in tests:** Use `fakeResolver` to control DNS responses.

### Secrets Fetching

**Location:** `secrets_client.go`

| Interface | Implementation | Purpose |
|-----------|---------------|---------|
| `SecretsFetcher` | `SecretsClient` | Fetch secrets from secrets-manager service |

**Methods:**
- `FetchBundleSecrets(ctx, scenario, tier, resources)` - Retrieve secrets manifest
- `HealthCheck(ctx)` - Verify secrets-manager is reachable

**Usage in tests:** Implement `SecretsFetcher` to return test secrets without HTTP calls.

### Secrets Generation

**Location:** `secrets_generator.go`

| Interface | Implementation | Purpose |
|-----------|---------------|---------|
| `SecretsGeneratorFunc` | `SecretsGenerator` | Generate per-install secrets using crypto/rand |

**Methods:**
- `GenerateSecrets(plans)` - Generate secret values for per_install_generated class

**Usage in tests:** Implement `SecretsGeneratorFunc` to return deterministic values.

### Progress Tracking

**Location:** `progress.go`

The `ProgressRepo` interface abstracts deployment progress persistence:
```go
type ProgressRepo interface {
    UpdateDeploymentProgress(ctx context.Context, id, step string, percent float64) error
}
```

## Key Seams (Where to Make Changes)

### Adding a New Domain Type
1. Add types to appropriate file in `domain/` package
2. Add type alias in the main package file that uses them
3. Update this SEAMS.md to document the new types

### Adding a New Deployment Phase
1. Define step types in the relevant `vps_*.go` file
2. Add handler(s) in appropriate `handlers_*.go`
3. Update route registration in `main.go`

### Adding a New VPS Action
1. Add action type to `handlers_vps_management.go`
2. Add confirmation validation
3. Add command builder function

### Adding a New Preflight Check
1. Add check logic to `RunVPSPreflight` in `preflight.go`
2. Use pass/warn/fail closures for consistent output

### Modifying Manifest Schema
1. Update types in `domain/manifest.go`
2. Update validation in `manifest.go` (`ValidateAndNormalizeManifest`)
3. Update tests in `manifest_contract_test.go`

### Adding Bundle Contents
1. Update bundling rules in `bundle.go`
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
handlers_*.go
      │
      ├──► domain/* (types, DTOs)
      │
      ├──► manifest.go (validation, uses domain types)
      │
      ├──► vps_*.go (business logic)
      │
      ├──► bundle.go (bundling logic)
      │
      ├──► preflight.go (preflight logic)
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

## File Size Indicators

Files over 500 lines that may benefit from further splitting:

| File | Lines | Notes |
|------|-------|-------|
| `bundle.go` | ~1070 | Consider separating stats/cleanup into `bundle_cleanup.go` |
| `handlers_live_state.go` | ~700 | Reduced after type extraction to domain; cohesive |
| `ssh_keys.go` | ~888 | Consider separating generation from management |
| `investigation_service.go` | ~847 | Complex orchestration, may be acceptable |

## Anti-Patterns to Avoid

1. **Don't add business logic to handlers** - Handlers should only decode/encode and delegate
2. **Don't add HTTP concerns to domain logic** - vps_*.go should not import net/http
3. **Don't scatter SSH commands** - Use vps_runners.go interfaces
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
- `vps_setup.go` - Now uses type aliases from domain.VPSPlanStep, domain.VPSSetupResult
- `vps_deploy.go` - Now uses type aliases from domain.VPSDeployResult, domain.MissingSecretInfo
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
- `SecretsFetcher` in `secrets_client.go` - Abstracts secrets-manager HTTP calls
- `SecretsGeneratorFunc` in `secrets_generator.go` - Abstracts crypto/rand secret generation

**Pattern:**
```go
// Interface definition
type SecretsFetcher interface {
    FetchBundleSecrets(ctx context.Context, scenario, tier string, resources []string) (*SecretsManagerResponse, error)
    HealthCheck(ctx context.Context) error
}

// Compile-time interface check
var _ SecretsFetcher = (*SecretsClient)(nil)
```

**Existing seams preserved:**
- `SSHRunner` / `SCPRunner` for VPS command execution (already well-used in tests)
- Inline resolver interface for DNS lookups in preflight
- `ProgressRepo` for deployment progress persistence

### Server-Level Dependency Injection (2026-01-03)
Promoted seams from inline instantiation to proper dependency injection via the `Server` struct:

**Server struct now holds seams:**
```go
type Server struct {
    // ... existing fields ...

    // Seam: SSH command execution (defaults to ExecSSHRunner)
    sshRunner SSHRunner
    // Seam: SCP file transfer (defaults to ExecSCPRunner)
    scpRunner SCPRunner
    // Seam: Secrets fetching (defaults to NewSecretsClient())
    secretsFetcher SecretsFetcher
    // Seam: Secrets generation (defaults to NewSecretsGenerator())
    secretsGenerator SecretsGeneratorFunc
    // Seam: DNS resolution (defaults to NetResolver)
    dnsResolver DNSResolver
}
```

**Handler migration complete - all handlers now use Server seams:**
- `handlers_deployment.go`: Uses `s.sshRunner`, `s.scpRunner`, `s.secretsFetcher`, `s.secretsGenerator`
- `handlers_preflight.go`: Uses `s.sshRunner`, `s.dnsResolver`
- `handlers_bundle.go`: Uses `s.sshRunner`
- `handlers_live_state.go`: Uses `s.sshRunner`
- `handlers_vps_management.go`: Uses `s.sshRunner`
- `handlers_edge.go`: Uses `s.sshRunner`, `s.dnsResolver`

**Updated business logic in `vps_deploy.go`:**
- `RunVPSDeploy`: Now accepts `SecretsGeneratorFunc` parameter (defaults to `NewSecretsGenerator()` if nil)
- `RunVPSDeployWithProgress`: Now accepts `SecretsGeneratorFunc` parameter (defaults to `NewSecretsGenerator()` if nil)

**Benefits:**
- All handlers can be fully tested with fake implementations
- No live SSH/SCP/DNS/secrets-manager connections required in tests
- Production code unchanged (defaults to real implementations)

**Testing pattern:**
```go
// In tests, create a Server with fake seams:
srv := &Server{
    sshRunner:        &FakeSSHRunner{Responses: map[string]SSHResult{...}},
    scpRunner:        &FakeSCPRunner{},
    secretsFetcher:   &FakeSecretsFetcher{Response: testSecrets},
    secretsGenerator: &FakeSecretsGenerator{Values: map[string]string{"key": "deterministic-value"}},
    dnsResolver:      &FakeResolver{Hosts: map[string][]string{"example.com": {"1.2.3.4"}}},
    // ... other fields ...
}
```

**Seam migration status: COMPLETE**
All handlers that use external integrations (SSH, SCP, DNS, secrets) now obtain these dependencies from the Server struct rather than creating instances inline. This makes the entire API testable without live external services.

### Reusable Test Fakes (2026-01-03)

Added `test_fakes.go` with reusable fake implementations for all seams:

| Fake | Interface | Purpose |
|------|-----------|---------|
| `FakeResolver` | `DNSResolver` | Control DNS lookup responses |
| `FakeSSHRunner` | `SSHRunner` | Control SSH command responses, track calls |
| `FakeSCPRunner` | `SCPRunner` | Control SCP copy results, track calls |
| `FakeSecretsFetcher` | `SecretsFetcher` | Control secrets-manager responses |
| `FakeSecretsGenerator` | `SecretsGeneratorFunc` | Return deterministic secret values |

**Features of all fakes:**
- Thread-safe call recording via `Calls` field
- Configurable error injection via `Err`/`Errs` fields
- Default behaviors for unspecified cases
- Compile-time interface verification via `var _ Interface = (*Fake)(nil)`

**Example with call verification:**
```go
ssh := &FakeSSHRunner{Responses: map[string]SSHResult{
    "echo ok": {ExitCode: 0, Stdout: "ok"},
}}
// ... run handler ...
if len(ssh.Calls) != 1 || ssh.Calls[0] != "echo ok" {
    t.Errorf("unexpected SSH calls: %v", ssh.Calls)
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

### Orchestrator Extraction
The `runDeploymentPipeline` function in `handlers_deployment.go` is ~180 lines of orchestration logic that coordinates:
1. Secrets fetching
2. Bundle building
3. VPS setup
4. VPS deployment
5. Progress tracking

Consider extracting to a `DeploymentOrchestrator` struct with explicit dependencies:
```go
type DeploymentOrchestrator struct {
    repo        DeploymentRepository
    progressHub *ProgressHub
    logger      func(msg string, fields map[string]interface{})
    sshRunner   SSHRunner
    scpRunner   SCPRunner
}

func (o *DeploymentOrchestrator) Execute(ctx context.Context, id string, manifest CloudManifest, ...) error
```

Benefits:
- Clearer responsibility boundary
- Easier testing through dependency injection
- Separates HTTP concerns from orchestration logic

### Bundle Cleanup Separation
`bundle.go` (~1070 lines) could be split:
- `bundle.go` - Core bundling logic
- `bundle_cleanup.go` - Stats, cleanup, retention logic

### SSH Keys Separation
`ssh_keys.go` (~888 lines) could be split:
- `ssh_keys.go` - Key management
- `ssh_keys_generator.go` - Key generation logic
