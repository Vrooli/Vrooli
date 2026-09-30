# Secrets Manager - Seams Documentation

This internal document captures:
1. **Responsibility Zones** - where each concern lives (from Boundary Enforcement phase)
2. **Interface Seams** - abstraction boundaries for testability and substitution (from Seam Discovery phase)

The goal is to make code easier to test, safer to change, and more modular by maintaining clear seams.

## Responsibility Zones

### 1. Entry/Presentation (HTTP Handlers)

**Location:** `*_handlers.go` files

| File | Responsibility |
|------|----------------|
| `credential_handlers.go` | HTTP handlers for credential coverage status and provisioning |
| `security_handlers.go` | HTTP handlers for security scanning and compliance |
| `deployment_handlers.go` | HTTP handlers for deployment manifest generation |
| `resource_handlers.go` | HTTP handlers for resource management |
| `health_handlers.go` | HTTP handlers for health checks |
| `orientation.go` | HTTP handlers for orientation/onboarding |

**Guidelines:**
- Handlers should parse requests, validate inputs, delegate to domain/services, and format responses
- Do not perform domain logic directly in handlers
- Cross-cutting concerns (logging, error handling) are acceptable

### 2. Coordination/Orchestration

**Location:** `server.go`, `deployment_manifest_builder.go`

| File | Responsibility |
|------|----------------|
| `server.go` | APIServer wires together all handlers and routes |
| `deployment_manifest_builder.go` | ManifestBuilder orchestrates manifest generation |

**Guidelines:**
- Orchestrators should wire together domain components
- Keep focused on ordering and wiring, not implementing rules

### 3. Domain Rules

**Location:** Domain files (`*_domain.go`, `validator.go`, `scanner.go`)

| File | Responsibility |
|------|----------------|
| `secret_classification.go` | **NEW** - Consolidated secret type classification rules |
| `compliance_domain.go` | **NEW** - Compliance metrics calculation |
| `validator.go` | SecretValidator for secret validation pipeline |
| `scanner.go` | SecretScanner for secret discovery from filesystem |
| `security_scanner.go` | Vulnerability detection patterns and rules |

**Guidelines:**
- Domain functions should be pure where possible (no I/O dependencies)
- Domain logic should not depend on HTTP or transport details
- Classification and calculation functions belong here

### 4. Integrations/Infrastructure

**Location:** Integration files

| File | Responsibility |
|------|----------------|
| `credential_coverage.go` | Credential-authority coverage and status parsing |
| `deployment_manifest_fetcher.go` | Scenario-dependency-analyzer client |
| `deployment_manifest_resolver.go` | Resource resolution from service.json |
| `resource_queries.go` | Database queries for resource data |
| `scan_records.go` | Database persistence for scan records |

**Guidelines:**
- Keep integration details isolated from domain logic
- Use adapters/clients to translate between domain and external systems

### 5. Cross-cutting Concerns

**Location:** Utility files

| File | Responsibility |
|------|----------------|
| `logger.go` | Structured logging |
| `http_utils.go` | HTTP response utilities |
| `types.go` | Shared data structures |
| `string_utils.go` | String manipulation utilities |
| `test_helpers.go` | Test utilities |

## Changes Made (Boundary Enforcement Phase)

### Consolidated Domain Logic

1. **Secret Classification** (`secret_classification.go`)
   - Extracted `ClassifySecretType()`, `IsLikelySecret()`, `IsLikelyRequired()`
   - Added `GetSecretKeywords()` to consolidate keyword lists
   - Previously duplicated in `scanner.go` and `credential_coverage.go`
   - Original functions now delegate to consolidated versions

2. **Compliance Metrics** (`compliance_domain.go`)
   - Extracted `calculateComplianceMetrics()`, `calculatePercentage()`, `tallyVulnerabilitySeverities()`, `buildComplianceResponse()`
   - Previously embedded in `security_handlers.go`
   - Handlers now call domain functions

### Separated Persistence from Handlers (December 2025)

3. **storeDiscoveredSecret** (`resource_queries.go`)
   - Moved from `resource_handlers.go` to `resource_queries.go`
   - Separates database persistence from HTTP handler concerns
   - Added null check for db to prevent panics

### Separated Integration Logic from Handlers (December 2025)

4. **Credential Authority Functions** (`credential_coverage.go`)
   - Moved canonical credential identity mapping from `credential_handlers.go`
   - Keeps credential-authority client integration logic together with other credential coverage functions
   - HTTP handlers now call these integration functions

### Consolidated Secret Keywords (December 2025)

5. **Scanner Keywords** (`scanner.go`)
   - `getSecretKeywords()` now delegates to `GetSecretKeywords()` in `secret_classification.go`
   - Eliminates duplicate keyword lists between scanner and classification modules

### Documented Technical Debt

1. **No plaintext fallback** (`credential_coverage.go`)
   - Authority status failures are returned explicitly
   - The production path never scans files, environment variables, or remote secret services for substitute values

## Known Responsibility Leaks (Future Work)

### High Priority

1. **Global State** (`main.go`, `security_scan.go`)
   - Package-level `logger` and `db` variables
   - Global caching state in `security_scan.go`
   - Consider dependency injection pattern

2. **Authority availability** (`credential_coverage.go`)
   - Credential-authority failures remain explicit and actionable
   - Tests substitute the authority client instead of depending on host state

### Medium Priority

3. **Dynamic SQL Building** (`resource_handlers.go:62-104`)
   - `ResourceSecretUpdate` builds SQL inline
   - Consider moving to repository pattern

4. **Mixed Concerns in security_scan.go**
   - Contains caching, scanning, and persistence logic
   - Could be split into separate files

## Where to Add New Functionality

| Need to... | Add to... |
|------------|-----------|
| Add new API endpoint | Create handler in appropriate `*_handlers.go`, register in `server.go` |
| Add new domain rule | Add function to relevant `*_domain.go` file |
| Add new secret type classification | Update `secret_classification.go` |
| Add new compliance metric | Update `compliance_domain.go` |
| Add new credential-authority integration | Update `credential_coverage.go` or `credential_coverage.go` |
| Add new database query | Add to `resource_queries.go` |
| Add cross-cutting utility | Add to `http_utils.go` or create specific utility file |
| Add secret keyword detection | Update `secret_classification.go` with new keywords |

---

## Interface Seams (Seam Discovery Phase - December 2025)

Interface seams are boundaries where behavior can be substituted without invasive changes.
These enable testing, swapping implementations, and isolating side effects.

### Existing Well-Defined Seams

These interfaces already exist and enable dependency injection:

| Seam | Production implementation | Test substitute | Primary consumer |
|------|---------------------------|-----------------|------------------|
| `envx.Reader` (`api/internal/envx/env.go`) | `envx.OS` | `mocks.FakeEnv` | `SecretValidator` |
| `ManifestClock` (`deployment_manifest_builder.go`) | `systemManifestClock` | `fixedManifestClock` | `ManifestBuilder` |
| `CredentialAuthorityClient` (`credential_coverage.go`) | `DefaultCredentialAuthorityClient` | `credentialAuthorityStub` | Credential coverage and validation |
| `ScenarioCLI` (`scenario_list.go`) | `DefaultScenarioCLI` | scenario-list test fakes | `ScenarioHandlers` |

`envx.Reader` was introduced on 2026-07-23 and is constructor-injected through
`NewSecretValidatorWithEnv`. It is intentionally narrow: the validator needs a
single named value, and tests can now validate environment-backed secrets
without changing global process state. The remaining direct environment reads
are tracked in `PROBLEMS.md`; new domain code must use an injected reader.

#### 0. ManifestClock (`deployment_manifest_builder.go`)

```go
type ManifestClock interface {
    Now() time.Time
}
```

**Implementation:** `systemManifestClock`, constructed by `APIServer`
**Used by:** `ManifestBuilder` for generated manifest timestamps
**Testability:** `fixedManifestClock` gives deployment tests an exact timestamp without wall-clock coupling

#### 1. SecretStore (`deployment_manifest_fetcher.go:32`)

```go
type SecretStore interface {
    FetchSecrets(ctx context.Context, tier string, resources []string, includeOptional bool) ([]DeploymentSecretEntry, error)
    PersistManifest(ctx context.Context, scenario, tier string, manifest *DeploymentManifest) error
}
```

**Implementation:** `PostgresSecretStore`
**Used by:** `ManifestBuilder` for database operations
**Testability:** Fully mockable for unit tests

#### 2. AnalyzerClient (`deployment_manifest_fetcher.go:41`)

```go
type AnalyzerClient interface {
    FetchDeploymentReport(ctx context.Context, scenario string) (*analyzerDeploymentReport, error)
}
```

**Implementation:** `HTTPAnalyzerClient`
**Used by:** `ManifestBuilder`, `ResourceResolver` for analyzer integration
**Testability:** Fully mockable; isolates HTTP calls to external service

#### 2a. HTTPDoer (`deployment_manifest_fetcher.go`)

**Status:** Implemented (2026-07-23)

`HTTPAnalyzerClient` owns the narrow outbound `HTTPDoer` seam and receives both
the HTTP transport and analyzer URL resolver through
`NewHTTPAnalyzerClientWithDeps()`. Production wires a standard `http.Client`;
the test fake records the request and returns a deterministic response. This
keeps the analyzer boundary testable without network access or a live
scenario-dependency-analyzer service.

#### 3. ResourceResolver (`deployment_manifest_resolver.go:20`)

```go
type ResourceResolver interface {
    ResolveResources(ctx context.Context, scenario string, requestedResources []string) ResolvedResources
}
```

**Implementation:** `DefaultResourceResolver`
**Used by:** `ManifestBuilder` for resource discovery
**Testability:** Fully mockable; isolates filesystem access to service.json

#### 4. ManifestBuilder Dependency Injection (`deployment_manifest_builder.go:64-78`)

The `NewManifestBuilderWithDeps()` constructor enables full dependency injection:

```go
func NewManifestBuilderWithDeps(
    secretStore SecretStore,
    analyzerClient AnalyzerClient,
    resourceResolver ResourceResolver,
    logger *Logger,
) *ManifestBuilder
```

**Pattern:** Constructor injection for all external dependencies
**Testability:** Can test all builder logic with mocks, no real I/O needed

#### 5. CredentialAuthorityClient Interface (`credential_coverage.go:27-36`)

**Status:** Implemented (December 2025)

```go
type CredentialAuthorityClient interface {
    GetSecretsStatus(ctx context.Context, resourceFilter string) (*CredentialCoverageStatus, error)
}
```

**Implementation:** `DefaultCredentialAuthorityClient`
**Injection Points:**
- `SetCredentialAuthorityClient()` - Package-level setter for tests
- `NewDefaultCredentialAuthorityClient()` - Production constructor
**Testability:** Fully mockable; isolates all `exec.Command` calls to vrooli credentials CLI

**Benefits:**
- Tests can mock authority responses without host-service installation
- Keeps fallback behavior explicit instead of introducing a second secret store
- Consolidates exec.Command calls to single implementation

#### 6. ScenarioCLI Interface (`scenario_list.go:22-25`)

**Status:** Implemented (December 2025 - Second Pass)

```go
type ScenarioCLI interface {
    ListScenarios(ctx context.Context) ([]scenarioSummary, error)
}
```

**Implementation:** `DefaultScenarioCLI`
**Injection Points:**
- `SetScenarioCLI()` - Package-level setter for tests
- `NewScenarioHandlersWithCLI()` - Handler constructor for dependency injection
**Used by:** `ScenarioHandlers` for the `/scenarios` endpoint
**Testability:** Fully mockable; isolates vrooli CLI calls

**Benefits:**
- Tests can mock scenario lists without vrooli CLI installation
- Enables testing scenario handlers in isolation
- Follows the established CredentialAuthorityClient pattern

### Seams That Could Be Strengthened (Lower Priority)

These areas have some structure but could benefit from further abstraction:

#### 1. Filesystem Operations

**Current State:** Direct `os.ReadFile`, `filepath.WalkDir` calls in:
- `scanner.go:findResourceFiles()`, `scanFile()`
- `credential_coverage.go:scanResourcesDirectly()`, `loadResourceSecrets()`
- `deployment_manifest_resolver.go:findServiceJSON()`

**Problem:** Tests require actual filesystem fixtures

**Recommendation:** For complex scenarios, consider `afero.Fs` interface or similar abstraction

#### 3. Clock Dependency

**Current State:** `time.Now()` called directly throughout:
- `deployment_manifest_builder.go:147` - manifest generation timestamp
- `credential_coverage.go:107` - last updated timestamp
- Various scan and validation timestamps

**Problem:** Time-dependent tests are flaky or require sleep()

**Recommendation:** Inject clock interface for time-sensitive tests. Example pattern:

```go
type Clock interface {
    Now() time.Time
}

type RealClock struct{}
func (RealClock) Now() time.Time { return time.Now() }
```

#### 4. Port Discovery (`deployment_manifest_fetcher.go:400-413`)

**Current State:** `discoverAnalyzerPort()` calls `vrooli scenario port` directly

**Problem:** Hard to test HTTP client logic without vrooli CLI

**Recommendation:** Could be extracted to an interface, but lower priority since it's an internal helper and the AnalyzerClient interface already enables mocking the overall fetch behavior.

### Seam Violation Patterns to Avoid

1. **Bypassing existing seams:**
   - DON'T call `exec.Command("vrooli credentials", ...)` directly in new code
   - DO use `CredentialAuthorityClient` interface methods or `SetCredentialAuthorityClient()` for testing
   - DON'T call `exec.Command("vrooli", "scenario", "list", ...)` directly
   - DO use `ScenarioCLI` interface methods or `SetScenarioCLI()` for testing

2. **Mixing concerns at seams:**
   - DON'T add business logic to integration functions
   - DO keep integration functions focused on data translation

3. **Testing through side effects:**
   - DON'T write tests that depend on authority/filesystem state
   - DO use interface mocks where seams exist

### Test Patterns for Each Seam Type

| Seam | Test Pattern | Example Location |
|------|--------------|------------------|
| `SecretStore` | Mock interface | `deployment_manifest_test.go` |
| `AnalyzerClient` | Mock interface | `deployment_manifest_test.go` |
| `HTTPDoer` | Recording fake transport | `deployment_manifest_test.go` |
| `ResourceResolver` | Mock interface | Use `NewManifestBuilderWithDeps()` |
| `CredentialAuthorityClient` | Mock interface | Use `SetCredentialAuthorityClient()` |
| `ScenarioCLI` | Mock interface | Use `SetScenarioCLI()` or `NewScenarioHandlersWithCLI()` |
| Filesystem | Temp directories | `scanner_test.go` |

### Seam Health Checklist

When modifying secrets-manager code:

- [ ] Does the change add new I/O? → Consider if an interface seam is needed
- [ ] Does the change call an external CLI? → Use existing integration functions or add to interface
- [ ] Is the change testable with mocks? → If not, the seam may need strengthening
- [ ] Does the change bypass an existing interface? → Refactor to use the seam
