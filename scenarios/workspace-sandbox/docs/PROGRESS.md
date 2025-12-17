# Progress Log

Track development progress, decisions, and significant changes.

| Date | Author | Status Snapshot | Notes |
|------|--------|-----------------|-------|
| 2025-12-16 | Claude Opus 4.5 | Intent Clarification & Assumption Hardening | Renamed ConflictType constants for clarity, added state machine ASCII diagram, documented assumptions explicitly in code, added validation guards, created assumption tests as executable documentation |
| 2025-12-17 | Claude Opus 4.5 | Progress & Signal Surface Design | Created PostgreSQL schema (sandboxes, audit_log, check_scope_overlap), added stats endpoint, implemented structured JSON logging, improved error signals with actionable hints. Backend API now fully functional. |
| 2025-12-17 | Claude Opus 4.5 | Comprehensive UI implementation | Replaced template UI with full-featured diff viewer, sandbox list, status header, create dialog. Complete UX for sandbox lifecycle management with approve/reject workflow. |
| 2025-12-17 | Claude Opus 4.5 | Comprehensive handler tests + requirement updates | Added 18 new handler tests covering create/list/get/delete/stop/diff/approve/reject/workspace/driver endpoints, fixed invalid test refs, updated 8 requirement modules with passing test references |
| 2025-12-17 | Claude Opus 4.5 | Test infrastructure + CLI fixes | Fixed CLI build errors (cliapp API), created test directory with run-tests.sh, added handlers_test.go for health endpoint, updated REQ-P0-010 with passing tests |
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

## 2025-12-17 Test Infrastructure + CLI Fixes

### Focus: Fixing Build Issues and Establishing Test Framework

This session focused on resolving critical build issues preventing the scenario from starting and establishing proper test infrastructure.

### Key Changes

#### 1. CLI Build Fixes (cli/app.go)
- **Removed**: Non-existent `Flags` field from command definitions
- **Removed**: Non-existent `cliapp.Flag` type references
- **Replaced**: `a.core.CLI.String()` / `a.core.CLI.Bool()` with direct argument parsing
- **Replaced**: `a.core.APIClient.Post/Delete` with `a.core.APIClient.Request()`
- **Fixed**: `map[string]string` to `url.Values` for query parameters
- **Added**: `net/url` import for proper URL value handling

#### 2. Test Directory Structure (test/)
- **Created**: `test/run-tests.sh` with phased test execution
- **Phases**: Structure validation, unit tests, integration tests, business tests, performance tests
- **Tests**: Health endpoint status codes, JSON schema validation, response time measurement

#### 3. Handler Unit Tests (api/internal/handlers/handlers_test.go)
- **Created**: New test file with mock implementations
- **Tests**:
  - `TestHealthHandler` - Tests status codes for healthy/unhealthy scenarios
  - `TestHealthResponseSchema` - Validates required JSON fields
  - `TestHealthContentType` - Validates Content-Type header
- **Annotations**: `[REQ:REQ-P0-010]` tags for requirement tracking

#### 4. Requirement Updates (requirements/09-health-check-api/module.json)
- **Updated**: REQ-P0-010 status from "draft" to "implemented"
- **Added**: Test references for unit, integration, and performance phases
- **Status**: 4/5 validations now passing

### Files Changed
- `cli/app.go` (modified - CLI API fixes)
- `test/run-tests.sh` (new)
- `api/internal/handlers/handlers_test.go` (new)
- `requirements/09-health-check-api/module.json` (updated)
- `docs/PROGRESS.md` (this file)

### Test Results
- Go unit tests: 20+ tests passing
- Integration tests: Health endpoint validation passing
- Performance tests: Health responds in <10ms (target <100ms)

### Next Steps
- Add unit tests for remaining packages (config, types, policy, repository)
- Implement OT-P0-001: Sandbox Create/Mount Operations tests
- Add integration tests for sandbox CRUD endpoints
- Implement business workflow tests

---

## 2025-12-17 Comprehensive UI Implementation

### Focus: Complete User Experience for Sandbox Lifecycle Management

This session replaced the placeholder template UI with a full-featured interface following the experience architecture guidelines. The goal was to make the sandbox workflow "scream its purpose" to users.

### User Experience Analysis

#### Core Personas Identified
1. **Agent Developer** - Creates sandboxes for agents, monitors them, reviews changes
2. **Code Reviewer** - Reviews sandbox changes, approves/rejects diffs
3. **System Operator** - Monitors all sandboxes, manages cleanup/GC

#### Primary User Flows
1. **Create Sandbox Flow**: Click "Create Sandbox" button -> Fill form -> Create
2. **Review Changes Flow**: Click sandbox in list -> See diff -> Approve/Reject
3. **Monitor Sandboxes Flow**: View dashboard -> Filter by status -> Track health

### Components Created

#### 1. StatusHeader (`ui/src/components/StatusHeader.tsx`)
- Health indicator with live status
- Sandbox statistics (total, active, stopped, error counts)
- Total disk usage display
- Create Sandbox button
- Refresh button

#### 2. SandboxList (`ui/src/components/SandboxList.tsx`)
- Sandboxes grouped by status (Active, Stopped, Error, Approved, Rejected)
- Expandable/collapsible groups
- Each sandbox shows: scope path, owner, size, created time
- Click to select and view details

#### 3. SandboxDetail (`ui/src/components/SandboxDetail.tsx`)
- Full sandbox metadata display
- Status badge with icon
- Copyable workspace path
- Action buttons: Stop, Approve, Reject, Delete
- Confirmation dialogs for destructive actions
- Embedded diff viewer

#### 4. DiffViewer (`ui/src/components/DiffViewer.tsx`)
- GitHub-style unified diff display
- File-level grouping with expand/collapse
- Change type badges (added/modified/deleted)
- Line-level coloring (+green/-red)
- Diff statistics (additions/deletions/file count)
- Empty and loading states

#### 5. CreateSandboxDialog (`ui/src/components/CreateSandboxDialog.tsx`)
- Scope path input (required)
- Project root input (optional)
- Owner name and type selection
- Form validation
- Loading state during creation

### UI Components Added

Added shadcn/ui style components:
- `Card`, `CardHeader`, `CardTitle`, `CardContent`, `CardFooter`
- `Badge` with sandbox status variants
- `Button` with success/destructive variants
- `Dialog` with header, footer, close button
- `Input`, `Label`, `Textarea`
- `Select` dropdown
- `ScrollArea` for scrollable content

### API Client (`ui/src/lib/api.ts`)

Complete TypeScript API client with:
- All sandbox CRUD operations
- Diff retrieval
- Approve/Reject workflow
- Health check
- Driver info
- Type definitions matching Go API

### React Query Hooks (`ui/src/lib/hooks.ts`)

Query and mutation hooks:
- `useHealth`, `useSandboxes`, `useSandbox`, `useDiff`
- `useCreateSandbox`, `useDeleteSandbox`, `useStopSandbox`
- `useApproveSandbox`, `useRejectSandbox`
- Automatic cache invalidation

### Test Selectors (`ui/src/consts/selectors.ts`)

Comprehensive test ID coverage:
- Literal selectors for all major UI elements
- Dynamic selectors for status groups, sandbox by ID
- `SELECTORS` export for component usage

### Files Changed
- `ui/src/App.tsx` (complete rewrite)
- `ui/src/lib/api.ts` (complete rewrite)
- `ui/src/lib/hooks.ts` (new)
- `ui/src/consts/selectors.ts` (updated with comprehensive selectors)
- `ui/src/components/StatusHeader.tsx` (new)
- `ui/src/components/SandboxList.tsx` (new)
- `ui/src/components/SandboxDetail.tsx` (new)
- `ui/src/components/DiffViewer.tsx` (new)
- `ui/src/components/CreateSandboxDialog.tsx` (new)
- `ui/src/components/ui/card.tsx` (new)
- `ui/src/components/ui/badge.tsx` (new)
- `ui/src/components/ui/scroll-area.tsx` (new)
- `ui/src/components/ui/dialog.tsx` (new)
- `ui/src/components/ui/input.tsx` (new)
- `ui/src/components/ui/select.tsx` (new)
- `ui/src/components/ui/button.tsx` (enhanced variants)

### UX Principles Applied

1. **Clarity & Understanding**
   - Clear status indicators with icons and color
   - Help text on form fields
   - Confirmations before destructive actions

2. **Reduce Friction**
   - One-click sandbox selection
   - Inline approve/reject without navigation
   - Auto-refresh of sandbox list

3. **Professional Interaction Design**
   - Lucide icons (not emojis)
   - Consistent spacing and alignment
   - Proper loading and error states

4. **Information Hierarchy**
   - Most important info (status, scope) at top
   - Actions prominently placed
   - Secondary info (metadata) lower

### Build Status
- TypeScript type-check: Pass
- Vite build: Success (295KB JS, 19KB CSS)

### Next Steps
- Add e2e tests in bas/cases/ for UI flows
- Implement hunk-level approval UI (P1 feature)
- Add responsive styles for mobile
- Consider keyboard shortcuts for common actions

---

---

## 2025-12-17 Progress & Signal Surface Design

### Focus: Unblocking API Functionality and Improving Observability

This session focused on two key areas from the scenario improvement methodology:
1. **Progress** - Fixing the database schema to enable full API functionality
2. **Signal & Feedback Surface Design** - Making the system self-explanatory at runtime

### Critical Blocker Resolved

The API was returning 500 errors because the PostgreSQL schema did not exist. This was the highest priority blocker preventing end-to-end testing.

### Database Schema (initialization/postgres/seed.sql)

Created complete PostgreSQL schema with:

#### Tables
1. **sandboxes** - Core table with all metadata columns:
   - Identity: id (UUID primary key)
   - Scope: scope_path, project_root
   - Ownership: owner, owner_type (agent/user/task/system)
   - Status: status (creating/active/stopped/approved/rejected/deleted/error), error_message
   - Timestamps: created_at, last_used_at, stopped_at, approved_at, deleted_at
   - Driver: driver, driver_version, lower_dir, upper_dir, work_dir, merged_dir
   - Accounting: size_bytes, file_count, active_pids, session_count
   - Metadata: tags[], metadata (JSONB)

2. **sandbox_audit_log** - Immutable audit trail:
   - Links to sandbox, event_type, event_time, actor, details

#### Functions
1. **check_scope_overlap** - Checks for path conflicts with active sandboxes
2. **get_sandbox_stats** - Returns aggregate statistics for dashboard
3. **update_last_used_at** - Trigger function for automatic timestamp updates

#### Indexes
- Status filtering (most common query)
- Scope path lookups (for overlap checking)
- Project root lookups
- Owner filtering (partial index)
- Time-based queries (for GC, listing)
- Composite status+created (for common filter combinations)

### Stats Endpoint (api/internal/handlers/handlers.go)

Added `/api/v1/stats` endpoint returning:
```json
{
  "stats": {
    "totalCount": 0,
    "activeCount": 0,
    "stoppedCount": 0,
    "errorCount": 0,
    "approvedCount": 0,
    "rejectedCount": 0,
    "deletedCount": 0,
    "totalSizeBytes": 0,
    "avgSizeBytes": 0
  },
  "timestamp": "2025-12-17T02:46:28Z"
}
```

### Structured Logging (api/internal/logging/logging.go)

Created new logging package with:

#### Event Categories
- `server.*` - Server lifecycle events
- `sandbox.*` - Sandbox operations (created, mounted, stopped, approved, rejected, deleted, error)
- `api.*` - Request/response events
- `driver.*` - Driver operations
- `policy.*` - Policy validation events

#### Features
- JSON-structured output for log aggregation
- Leveled logging (debug, info, warn, error)
- Context-aware logger injection
- Convenience methods for common events
- Duration tracking in milliseconds

#### Sample Output
```json
{"time":"2025-12-17T02:44:24Z","level":"info","event":"server.initialized","message":"Server initialized successfully","service":"workspace-sandbox-api","fields":{"driver":"overlayfs","maxSandboxes":1000,"port":"15426"}}
{"time":"2025-12-17T02:44:30Z","level":"info","event":"api.request","message":"API request completed","service":"workspace-sandbox-api","durationMs":3,"fields":{"method":"GET","path":"/api/v1/sandboxes","statusCode":200}}
```

### Error Signal Improvements (api/internal/types/errors.go)

Enhanced all domain errors with:

1. **Actionable messages** - Error strings now include guidance:
   - Before: `sandbox not found: xyz`
   - After: `sandbox not found: xyz. Verify the ID is correct and the sandbox hasn't been deleted.`

2. **Resolution hints** - Each error type has a `Hint()` method:
   - `NotFoundError.Hint()` → "Use GET /api/v1/sandboxes to list available sandboxes"
   - `ScopeConflictError.Hint()` → "Either delete the conflicting sandbox or choose a non-overlapping path"
   - `StateError.Hint()` → Context-specific guidance based on current status
   - `DriverError.Hint()` → Operation-specific troubleshooting guidance

3. **API response format** - Error responses now include:
   ```json
   {
     "error": "descriptive message with context",
     "code": 404,
     "success": false,
     "hint": "actionable guidance for resolution",
     "retryable": false
   }
   ```

### Files Changed
- `initialization/postgres/seed.sql` (complete rewrite)
- `api/internal/config/config.go` (added Schema field)
- `api/main.go` (schema config, structured logging)
- `api/internal/logging/logging.go` (new)
- `api/internal/handlers/handlers.go` (stats endpoint, hints in errors)
- `api/internal/repository/sandbox_repo.go` (GetStats method)
- `api/internal/types/types.go` (SandboxStats type)
- `api/internal/types/errors.go` (hints for all error types)

### Validation Results

All endpoints now working:
- ✅ `GET /health` - Returns healthy status
- ✅ `GET /api/v1/sandboxes` - Returns empty list (totalCount: 0)
- ✅ `GET /api/v1/stats` - Returns aggregate statistics
- ✅ `GET /api/v1/driver/info` - Returns driver status
- ✅ `GET /api/v1/sandboxes/{id}` - Returns 404 with hint for missing ID
- ✅ Structured JSON logs appear in scenario logs

### Next Steps
- Test full sandbox create → diff → approve workflow
- Add e2e tests for sandbox lifecycle
- Implement GC/prune endpoints
- Add metrics endpoint for system-monitor integration

---

---

## 2025-12-16 Intent Clarification & Assumption Hardening

### Focus: Making Intent Explicit and Hardening Hidden Assumptions

Following the ecosystem-manager phases for Intent Clarification and Assumption Mapping & Hardening, this session focused on:
1. Clarifying naming to communicate intent
2. Adding documentation that explains the "why"
3. Discovering and guarding implicit assumptions
4. Creating tests that serve as executable documentation of assumptions

### Key Changes

#### 1. ConflictType Naming Clarification (types/types.go)

**Problem**: The old names `ConflictTypeNewIsAncestor` and `ConflictTypeExistingIsAncestor` were confusing about which path was the ancestor of which.

**Solution**: Renamed to intent-revealing names:
- `ConflictTypeNewIsAncestor` → `ConflictTypeNewContainsExisting`
- `ConflictTypeExistingIsAncestor` → `ConflictTypeExistingContainsNew`

Added comprehensive documentation:
```go
// ConflictTypeNewContainsExisting means the new scope is a parent of the existing scope.
// If we allow this, the new sandbox could modify files that the existing sandbox
// is also working on.
// Example: new="/project" contains existing="/project/src"
```

#### 2. Package Documentation (types/types.go)

Added comprehensive package-level documentation explaining:
- Domain overview (what a sandbox is, why it exists)
- Key concepts (Scope Path, Status, Overlay Layers)
- Mutual exclusion rule with examples
- Safety model (safety from accidents, not security from adversaries)

#### 3. State Machine ASCII Diagram (types/status.go)

Added visual ASCII art state machine diagram:
```
                              ┌─────────┐
                              │ CREATING │
                              └────┬────┘
                        success │    │ error
                    ┌───────────┘    └──────────┐
                    ▼                           ▼
               ┌────────┐                  ┌───────┐
     ┌────────►│ ACTIVE │◄────┐            │ ERROR │
     │ resume  └───┬────┘     │ resume     └───┬───┘
     ...
```

Documented status categories and key invariants:
1. Only Active sandboxes have a mounted overlay
2. Terminal statuses cannot transition to other states
3. All statuses can transition to Deleted
4. Changes can only be approved/rejected from Active or Stopped

#### 4. Assumption Guards in Service Layer (sandbox/service.go)

Added explicit guards for implicit assumptions in `Create()`:
```go
// ASSUMPTION: Either request or config provides project root
// GUARD: Check this explicitly and provide helpful error message
if projectRoot == "" {
    return nil, types.NewValidationErrorWithHint(
        "projectRoot",
        "project root is required but not provided",
        "Set projectRoot in the request body, or configure PROJECT_ROOT environment variable",
    )
}
```

Added guards in `GetDiff()`:
- Check status allows diff generation
- Verify UpperDir is set
- Verify LowerDir is set
- Handle empty changes gracefully

#### 5. Defensive Checks in Diff Generation (diff/diff.go)

Added filesystem existence checks:
```go
// ASSUMPTION: Directories actually exist
// GUARD: Check filesystem to catch cleanup races or configuration errors
if _, err := os.Stat(s.UpperDir); os.IsNotExist(err) {
    return nil, fmt.Errorf("sandbox upper directory does not exist: %s (sandbox may have been cleaned up)", s.UpperDir)
}
```

#### 6. External Command Documentation (diff/diff.go)

Added package documentation listing all external dependencies:
- `diff` - GNU diffutils for modified file comparison
- `git` - For patch application in git repos
- `patch` - Fallback for non-git directories

Documented assumptions:
- Commands available in PATH
- Exit code behaviors
- File path restrictions

#### 7. Assumption Tests (sandbox/assumptions_test.go)

Created new test file with tests as executable documentation:
- `TestAssumption_OnlyActiveSandboxesHaveMountedOverlays`
- `TestAssumption_TerminalStatusesCannotTransition`
- `TestAssumption_DeleteIsAlwaysAllowed`
- `TestAssumption_MutualExclusionPreventsOverlap`
- `TestAssumption_PathNormalizationPreventsAliases`
- `TestAssumption_ApprovalRequiresStoppedOrActive`
- `TestAssumption_ErrorTypesHaveHints`
- `TestAssumption_SandboxIDsAreStable`

Each test documents:
- The assumption being tested
- Why the assumption exists
- Failure mode if violated

### Files Changed
- `api/internal/types/types.go` (ConflictType rename, package docs)
- `api/internal/types/status.go` (ASCII diagram, invariant docs)
- `api/internal/sandbox/service.go` (assumption guards, method docs)
- `api/internal/sandbox/pathutil_test.go` (updated for new names)
- `api/internal/sandbox/assumptions_test.go` (new - executable assumption docs)
- `api/internal/handlers/handlers_test.go` (updated for new names)
- `api/internal/diff/diff.go` (external deps docs, filesystem checks)

### Tests
All tests pass including new assumption tests:
- 10 assumption tests documenting key invariants
- 6 path overlap tests with semantic names
- All handler tests updated for renamed constants

### Next Steps
- Add assumption tests for driver layer
- Document timing assumptions (mount persistence, file system consistency)
- Add property-based tests for path normalization
- Consider adding runtime assertion mode for development

---

*Future entries should follow the table format above. Include date, author (agent or human), status summary, and notes about what changed.*
