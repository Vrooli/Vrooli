# Workspace Sandbox - Axes of Change

This document identifies the **primary axes of change** for the workspace-sandbox scenario and where each axis should land in the codebase. The goal is to make future changes along each axis **localized, cheap, and low-risk**.

## Primary Axes of Change

### 1. Driver Implementation Variants

**Likelihood of Change**: High
**Current Status**: Well-localized via `Driver` interface

| Variation | Description |
|-----------|-------------|
| overlayfs (kernel) | Primary Linux driver, requires privileges |
| fuse-overlayfs | Unprivileged operation, slower |
| bind-mount fallback | Simple copy-based fallback for limited systems |
| Cross-platform | macOS/Windows drivers (future) |

**Extension Point**: `api/internal/driver/driver.go` - `Driver` interface

**To add a new driver**:
1. Create new file in `api/internal/driver/` (e.g., `bindmount.go`)
2. Implement the `Driver` interface
3. Add driver type constant to `driver.go`
4. Wire driver selection in `main.go` based on config

---

### 2. Approval & Review Policies

**Likelihood of Change**: High
**Current Status**: Hardcoded in service - needs extraction

| Variation | Description |
|-----------|-------------|
| Require human approval | Default: changes require explicit approval |
| Auto-approve | Agent changes applied automatically |
| Threshold-based | Auto-approve small changes, require review for large |
| Time-delayed | Auto-approve after N hours if no rejection |

**Desired Extension Point**: `api/internal/policy/approval.go` - `ApprovalPolicy` interface

**Current Pain Point**: Approval logic is embedded in `service.go:Approve()`. Adding a new policy requires modifying service code.

---

### 3. Commit Attribution & Message Policy

**Likelihood of Change**: Medium
**Current Status**: Hardcoded in service

| Variation | Description |
|-----------|-------------|
| Agent attribution | Commit as agent (default) |
| User attribution | Commit as reviewing user |
| Co-authored | Both agent and reviewer credited |
| Custom message format | Configurable commit message templates |

**Desired Extension Point**: `api/internal/policy/attribution.go` - `AttributionPolicy` interface

**Current Pain Point**: Commit message and author are passed directly to patcher in `service.go:Approve()`.

---

### 4. GC & Lifecycle Policies

**Likelihood of Change**: High
**Current Status**: Not yet implemented, no extension point

| Variation | Description |
|-----------|-------------|
| TTL-based | Delete sandboxes older than N hours |
| Size-based | Delete when total size exceeds limit |
| Idle-based | Delete sandboxes unused for N hours |
| Status-based | Auto-cleanup approved/rejected sandboxes |

**Desired Extension Point**: `api/internal/policy/gc.go` - `GCPolicy` interface

**Current Pain Point**: No GC system exists. When added, should be policy-driven.

---

### 5. Pre-Commit Validation Hooks

**Likelihood of Change**: Medium
**Current Status**: Not yet implemented

| Variation | Description |
|-----------|-------------|
| Run tests | Execute test suite before commit |
| Run linter | Execute lint checks before commit |
| Custom scripts | Run arbitrary validation scripts |
| None | No validation (fast path) |

**Desired Extension Point**: `api/internal/policy/validation.go` - `ValidationPolicy` interface

---

### 6. Path & Scope Policies

**Likelihood of Change**: Medium
**Current Status**: Hardcoded path validation in pathutil.go

| Variation | Description |
|-----------|-------------|
| Exclusion patterns | Paths that can never be sandboxed |
| Inclusion patterns | Only allow specific paths |
| Sensitive path protection | Extra restrictions for config files |

**Desired Extension Point**: `api/internal/policy/scope.go` - `ScopePolicy` interface

---

### 7. Observability & Metrics

**Likelihood of Change**: Medium
**Current Status**: Uses standard `log` package, no metrics

| Variation | Description |
|-----------|-------------|
| Prometheus metrics | Standard metrics endpoint |
| StatsD | Push metrics to aggregator |
| OpenTelemetry | Distributed tracing |
| Structured logging | JSON logs for aggregation |

**Desired Extension Point**: `api/internal/observability/` package with interfaces

---

## Current Change Topography

| Axis | Localization | Effort to Change |
|------|--------------|------------------|
| New driver | Excellent | Add file, implement interface |
| Approval policy | Poor | Modify service.go internals |
| Attribution policy | Poor | Modify service.go internals |
| GC policy | N/A | Not implemented |
| Validation hooks | N/A | Not implemented |
| Scope policy | Medium | Modify pathutil.go |
| Observability | Poor | Scattered log calls |

## Stable Core vs Volatile Edges

### Stable Core (rarely changes)
- `types/types.go` - Domain entities
- `types/status.go` - State machine
- `types/errors.go` - Error types
- `driver/driver.go` - Driver interface
- `repository/sandbox_repo.go` - Repository interface

### Volatile Edges (expected to evolve)
- Approval policies
- GC policies
- Validation hooks
- Attribution rules
- Metrics collection
- Configuration values

## Recommendations

### Immediate (this session)
1. Create `api/internal/config/` package to centralize tunable levers
2. Extract policy interfaces to prepare for future variation
3. Make volatile values configurable without code changes

### Future (subsequent sessions)
1. Implement GC system with policy interface
2. Add pre-commit validation hook interface
3. Add structured logging interface
4. Add metrics interface
