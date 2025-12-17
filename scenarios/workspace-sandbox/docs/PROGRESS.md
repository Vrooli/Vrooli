# Progress Log

Track development progress, decisions, and significant changes.

| Date | Author | Status Snapshot | Notes |
|------|--------|-----------------|-------|
| 2025-12-16 | Claude Opus 4.5 | Architectural improvements | Extracted status state machine, consolidated domain errors, added DomainError interface, defined ServiceAPI interface, reduced handler cognitive load |
| 2025-12-17 | Generator Agent | Initialization complete | Scenario scaffold created, PRD drafted with 26 operational targets (9 P0, 9 P1, 8 P2), requirements registry generated (20 requirements covering 19 targets), documentation set initialized |
| 2025-12-16 | Claude Opus 4.5 | Change Axis & Control Surface | Created unified config package, extracted policy interfaces, documented axes of change, added functional options pattern for service configuration |

## Initialization Summary

### What Was Created
- **Scaffold**: react-vite template applied (Go API + React UI + CLI)
- **PRD**: 26 operational targets across P0/P1/P2 priorities
- **Requirements**: 20 module.json files generated via prd-control-tower
- **Documentation**: README.md, RESEARCH.md, PROBLEMS.md, PROGRESS.md

### Key Decisions
1. **Template choice**: react-vite selected for full-stack capability (API + UI + CLI)
2. **Category**: developer_tools (reflects target users: agents, developers, CI/CD)
3. **Driver architecture**: overlayfs + bwrap as primary Linux implementation
4. **Safety vs security**: Explicitly documented as safety-focused, not security-hardened

### Research Findings
- No existing workspace-sandbox capability in Vrooli
- test-genie has process containment but not file-system overlay sandboxing
- overlayfs + bwrap combination is optimal for speed and storage efficiency

### Open Questions
- fuse-overlayfs vs privileged overlayfs support
- PostgreSQL vs SQLite for metadata
- Process tracking via cgroups vs process groups
- Diff format: standard unified vs Git-style

### Next Steps for Improvers
1. Implement P0-001: Sandbox Create/Mount Operations
2. Implement P0-002: overlayfs driver
3. Add unit tests for path normalization and mutual exclusion
4. Set up database schema for sandbox metadata

---

## 2025-12-16 Change Axis & Control Surface Design

### Focus: Evolution Resilience and Tunable Levers

Following the ecosystem-manager architectural guidelines for "Change Axis & Evolution Resilience" and "Control Surface & Tunable Levers", this session focused on:
1. Identifying primary axes of change
2. Localizing change along clear extension points
3. Extracting hardcoded values into configurable levers

### Key Changes

#### 1. AXES.md Documentation (docs/AXES.md)
- **Created**: New file documenting 7 primary axes of change
- **Content**: Change topography analysis, stable core vs volatile edges, recommendations

#### 2. Unified Config Package (api/internal/config/config.go)
- **Created**: Centralized configuration with coherent groupings
- **Groups**: Server, Limits, Lifecycle, Policy, Driver, Database
- **Features**:
  - Environment variable loading with `WORKSPACE_SANDBOX_` prefix
  - Validation with clear error messages
  - Sane defaults that work out of the box
  - Intention-revealing names for all levers

#### 3. Policy Interfaces (api/internal/policy/)
- **Created**: 4 new files defining pluggable behavior
  - `policy.go` - Core interfaces
  - `approval.go` - ApprovalPolicy implementations
  - `attribution.go` - AttributionPolicy implementation
  - `validation.go` - ValidationPolicy implementations
- **Interfaces**:
  - `ApprovalPolicy` - Decides auto-approve vs require-human
  - `AttributionPolicy` - Controls commit author/message format
  - `ValidationPolicy` - Pre-commit validation hooks

#### 4. Service Options Pattern (api/internal/sandbox/service.go)
- **Added**: Functional options for policy injection
  - `WithApprovalPolicy()`
  - `WithAttributionPolicy()`
  - `WithValidationPolicy()`
- **Updated**: `Approve()` method to consult policies

#### 5. Main.go Refactoring (api/main.go)
- **Updated**: Uses unified config package
- **Updated**: HTTP timeouts from config (not hardcoded)
- **Updated**: Shutdown timeout from config
- **Updated**: Driver and service config from unified config

#### 6. Handlers Update (api/internal/handlers/handlers.go)
- **Added**: Config field to Handlers struct
- **Added**: Version constant for health endpoint

### Files Changed
- `docs/AXES.md` (new)
- `api/internal/config/config.go` (new)
- `api/internal/policy/policy.go` (new)
- `api/internal/policy/approval.go` (new)
- `api/internal/policy/attribution.go` (new)
- `api/internal/policy/validation.go` (new)
- `api/internal/sandbox/service.go` (modified)
- `api/internal/handlers/handlers.go` (modified)
- `api/main.go` (modified)
- `api/internal/repository/sandbox_repo.go` (modified)
- `docs/SEAMS.md` (updated with new extension points)

### Configuration Levers Extracted

| Category | Lever | Default | Description |
|----------|-------|---------|-------------|
| Server | ReadTimeout | 30s | HTTP read timeout |
| Server | WriteTimeout | 30s | HTTP write timeout |
| Server | IdleTimeout | 120s | HTTP idle timeout |
| Server | ShutdownTimeout | 10s | Graceful shutdown timeout |
| Limits | MaxSandboxes | 1000 | Maximum active sandboxes |
| Limits | MaxSandboxSizeMB | 10240 | Max size per sandbox (10GB) |
| Limits | MaxTotalSizeMB | 102400 | Total sandbox storage (100GB) |
| Limits | DefaultListLimit | 100 | Default page size |
| Lifecycle | DefaultTTL | 24h | Sandbox time-to-live |
| Lifecycle | IdleTimeout | 4h | Idle before GC eligible |
| Lifecycle | GCInterval | 15m | GC run frequency |
| Policy | RequireHumanApproval | true | Require human review |
| Policy | AutoApproveThresholdFiles | 10 | Max files for auto-approve |
| Policy | CommitAuthorMode | agent | Who is commit author |

### Tests
All existing tests pass. Changes are backwards-compatible.

### Next Steps
- Add unit tests for new config validation
- Add unit tests for policy implementations
- Implement GC system using policy pattern
- Add structured logging interface

---

## 2025-12-16 Architectural Improvements

### Focus: Cognitive Load Reduction & Decision Boundary Extraction

Following the ecosystem-manager architectural guidelines, this session focused on making the codebase easier to understand, reason about, and safely modify.

### Key Changes

#### 1. Status State Machine Extraction (types/status.go)
- **Created**: New file with all status-related logic
- **Added**: Explicit decision functions (`CanStop`, `CanApprove`, `CanReject`, `CanDelete`, `CanGenerateDiff`, `CanGetWorkspacePath`)
- **Added**: `ValidTransitions` matrix documenting the state machine
- **Added**: `InvalidTransitionError` for clear error reporting
- **Benefit**: State transitions are now testable, documented, and easy to find

#### 2. Domain Errors Consolidation (types/errors.go)
- **Created**: New file with all domain error types
- **Added**: `DomainError` interface with `HTTPStatus()` and `IsRetryable()`
- **Moved**: `NotFoundError`, `ScopeConflictError` from service.go
- **Added**: `ValidationError`, `StateError`, `DriverError`
- **Benefit**: Errors are co-located with types and enable consistent HTTP error mapping

#### 3. Handler Error Consolidation (handlers/handlers.go)
- **Added**: `HandleDomainError()` method for single-line error handling
- **Added**: `JSONSuccess()` and `JSONCreated()` response helpers
- **Updated**: All handlers to use new helpers
- **Benefit**: Reduced boilerplate, consistent error responses

#### 4. Service Interface Definition (sandbox/service.go)
- **Added**: `ServiceAPI` interface documenting all service operations
- **Updated**: Handlers to depend on interface, not concrete type
- **Benefit**: Clear contract documentation, enables mock testing

### Files Changed
- `api/internal/types/status.go` (new)
- `api/internal/types/errors.go` (new)
- `api/internal/types/types.go` (removed duplicated Status definitions)
- `api/internal/sandbox/service.go` (added interface, use new state checks)
- `api/internal/handlers/handlers.go` (consolidated error handling, response helpers)
- `docs/SEAMS.md` (comprehensive documentation of improvements)

### Tests
All existing tests pass. The architectural changes are backwards-compatible.

### Next Steps
- Add unit tests for new state machine functions
- Add unit tests for domain error behaviors
- Consider adding handler unit tests using mock ServiceAPI

---

*Future entries should follow the table format above. Include date, author (agent or human), status summary, and notes about what changed.*
