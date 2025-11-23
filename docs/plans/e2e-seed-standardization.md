# E2E Test Seeding/Cleanup Standardization Plan

**Status**: Draft
**Created**: 2025-11-23
**Owner**: Testing Infrastructure
**Target**: All scenarios with UI/integration tests

## Executive Summary

Browser-automation-studio has pioneered a robust seeding/cleanup pattern for its e2e tests. This plan standardizes and extracts these patterns into reusable utilities that any scenario can adopt, ensuring consistent test isolation, deterministic fixtures, and reliable cleanup across the Vrooli ecosystem.

**Key Goals**:
1. Extract BAS-specific seeding patterns into generic, reusable utilities
2. Reduce boilerplate for scenarios adopting e2e testing
3. Ensure test isolation through proper state management
4. Enable self-documenting seed implementations
5. Maintain backward compatibility with existing BAS tests

## Current State Analysis

### What Works Well

**Browser-automation-studio** has established these patterns:

```
test/playbooks/
├── __seeds/
│   ├── apply.sh          # Creates seed data, writes seed-state.json
│   └── cleanup.sh        # Deletes seed data, removes seed-state.json
├── __subflows/
│   └── load-seed-state.json  # Loads seed state into workflow context
└── capabilities/
    └── [test playbooks reference @seed/key tokens]
```

**Data Flow**:
```
1. Phase starts → apply.sh runs → seed-state.json created
2. Workflows reference @seed/projectId → resolve-workflow.py injects actual value
3. Tests run, may mutate state
4. Test with reset=full → cleanup.sh + apply.sh (fresh state)
5. Phase ends → cleanup.sh (guaranteed via trap)
```

**Strengths**:
- ✅ **Deterministic**: Unique IDs per run prevent collisions
- ✅ **Automatic cleanup**: Even on test failure (via trap)
- ✅ **Declarative**: Fixtures use `@seed/key` tokens instead of hardcoded values
- ✅ **Centralized orchestration**: `phase-helpers.sh` manages lifecycle
- ✅ **Test isolation**: Each run gets fresh state
- ✅ **Idempotent**: Can detect/reuse existing seed state

### Current Limitations

1. **No reusable utilities**: Each scenario must implement `apply.sh`/`cleanup.sh` from scratch
2. **No validation**: `seed-state.json` has no schema, errors caught late
3. **No scaffolding**: High barrier to entry for new scenarios
4. **Inconsistent patterns**: No guidance on API error handling, retries, timeouts
5. **Hidden dependencies**: `jq`, `curl`, `openssl` required but not documented
6. **No testing helpers**: Can't easily test seed scripts in isolation
7. **Manual state file management**: Every scenario reimplements JSON validation

### Real-World Usage

**Currently using seeds**:
- `browser-automation-studio`: Full implementation (projects, workflows)
- `landing-manager`: Stub only (`.gitkeep` placeholder)
- `deployment-manager`: Stub only
- `react-component-library`: Stub only
- `tidiness-manager`: Stub only

**Key insight**: Only BAS has fully implemented seeding. Other scenarios need these patterns but haven't adopted due to complexity.

## Problems & Opportunities

### Problem 1: High Implementation Barrier

**Current**: A scenario wanting e2e tests must:
1. Study BAS's `apply.sh` (177 lines) and `cleanup.sh` (46 lines)
2. Understand phase orchestration (phase-helpers.sh:625-765)
3. Implement API calls, error handling, state file management
4. Create subflow fixtures for loading seed state
5. Learn `@seed/` token resolution

**Impact**: Only 1 of 5 scenarios has implemented seeding despite all needing it.

**Opportunity**: Provide reusable utilities + scaffolding → 80% reduction in implementation effort

---

### Problem 2: Scenario-Specific Assumptions

**Current**: BAS's seed scripts are tightly coupled to BAS API:
- Hardcoded `/api/v1/projects`, `/api/v1/workflows/create`
- BAS-specific cleanup (orphaned workflows, disk-based storage)
- Implicit knowledge of BAS data model

**Impact**: Can't copy-paste for other scenarios (each has different APIs)

**Opportunity**: Extract **generic** utilities that work with any HTTP API + JSON responses

---

### Problem 3: No Failure Recovery

**Current**: If seeding fails midway:
- Partial state left in database
- No rollback mechanism
- Next test run may behave unpredictably
- Manual cleanup required

**Impact**: Flaky tests, wasted developer time debugging state pollution

**Opportunity**: Add transaction-like semantics with automatic rollback

---

### Problem 4: Poor Debuggability

**Current**: When seeds fail:
- No structured logging
- No dry-run mode
- Can't verify cleanup worked
- Hard to test seed scripts in isolation

**Impact**: Long debug cycles, uncertain test reliability

**Opportunity**: Add testing/validation utilities specifically for seed scripts

## Proposed Solution

### Architecture Overview

```
scripts/scenarios/testing/shell/
├── seed-helpers.sh              # NEW: Generic seed utilities
└── seed-testing.sh              # NEW: Seed script testing tools

scripts/scenarios/testing/playbooks/
├── seed-state.schema.json       # NEW: JSON schema for validation
└── scaffold-seeds.sh            # NEW: Generate seed boilerplate

scenarios/<name>/test/playbooks/__seeds/
├── apply.sh                     # Uses seed-helpers.sh
├── cleanup.sh                   # Uses seed-helpers.sh
├── verify-cleanup.sh            # OPTIONAL: Validates cleanup
└── README.md                    # Auto-generated documentation
```

### Core Utilities: `seed-helpers.sh`

**Design Principles**:
1. **Generic**: No scenario-specific assumptions
2. **Composable**: Small functions that combine for complex workflows
3. **Defensive**: Validate inputs, check prerequisites, fail fast
4. **Observable**: Structured logging, clear error messages

**Proposed API**:

```bash
#!/usr/bin/env bash
# Generic utilities for e2e test seeding/cleanup

# ============================================================================
# Environment Resolution
# ============================================================================

seed::resolve_api_port() {
  # Usage: port=$(seed::resolve_api_port "scenario-name" "API_PORT")
  # Returns: Port number or exits with error
  local scenario_name="$1"
  local port_var="${2:-API_PORT}"
  # Implementation uses: vrooli scenario port "$scenario_name" "$port_var"
}

seed::resolve_api_url() {
  # Usage: url=$(seed::resolve_api_url "scenario-name" "/api/v1")
  # Returns: http://localhost:PORT/api/v1
}

# ============================================================================
# Prerequisite Validation
# ============================================================================

seed::require_tools() {
  # Usage: seed::require_tools jq curl openssl
  # Validates all required CLI tools are available
  # Exits with clear error message if any are missing
}

seed::require_scenario_running() {
  # Usage: seed::require_scenario_running "scenario-name"
  # Validates scenario is running and healthy
  # Provides actionable error message with startup commands
}

# ============================================================================
# HTTP Utilities
# ============================================================================

seed::curl_json() {
  # Usage: response=$(seed::curl_json GET "$api_url/resource")
  # Usage: response=$(seed::curl_json POST "$api_url/resource" "$payload")
  # Usage: response=$(seed::curl_json DELETE "$api_url/resource/$id")
  #
  # Features:
  # - Automatic Content-Type: application/json
  # - Timeout handling (default 30s, configurable via SEED_CURL_TIMEOUT)
  # - Structured error messages with HTTP status codes
  # - Optional retry logic (via SEED_CURL_RETRIES)
}

seed::extract_json_field() {
  # Usage: id=$(seed::extract_json_field "$response" ".resource.id // .id // empty")
  # Wrapper around jq with better error messages
  # Returns extracted value or exits with JSONPath details
}

# ============================================================================
# Unique ID Generation
# ============================================================================

seed::generate_run_id() {
  # Usage: run_id=$(seed::generate_run_id)
  # Returns: timestamp-hex (e.g., "1763874256-dd0107")
  # Ensures uniqueness across concurrent test runs
}

seed::generate_unique_name() {
  # Usage: name=$(seed::generate_unique_name "Demo Project")
  # Returns: "Demo Project 1763874256-dd0107"
  # Combines human-readable prefix with unique run ID
}

# ============================================================================
# State File Management
# ============================================================================

seed::state_file_path() {
  # Usage: path=$(seed::state_file_path "$scenario_dir")
  # Returns: $scenario_dir/test/artifacts/runtime/seed-state.json
  # Creates parent directory if needed
}

seed::write_state() {
  # Usage: seed::write_state "$scenario_dir" "$state_json"
  #
  # Features:
  # - Validates JSON structure via jq
  # - Ensures required keys present (configurable)
  # - Creates parent directories
  # - Atomic write (tmp file + mv)
  # - Validates against schema if available
}

seed::read_state() {
  # Usage: state=$(seed::read_state "$scenario_dir")
  # Returns: Full JSON object or empty object if not exists
  # Validates JSON is well-formed before returning
}

seed::validate_state() {
  # Usage: seed::validate_state "$state_json" key1 key2 key3
  # Validates all required keys are present and non-null
  # Exits with descriptive error showing which keys are missing
}

seed::cleanup_state() {
  # Usage: seed::cleanup_state "$scenario_dir"
  # Safely removes seed-state.json
  # Silent if file doesn't exist (idempotent)
}

# ============================================================================
# Resource Tracking (for cleanup)
# ============================================================================

seed::track_resource() {
  # Usage: seed::track_resource "project" "$project_id"
  # Appends to tracked resources list for cleanup
  # Stored in: test/artifacts/runtime/seed-resources.json
  # Format: {"projects": ["id1", "id2"], "workflows": ["id3"]}
}

seed::get_tracked_resources() {
  # Usage: ids=$(seed::get_tracked_resources "project")
  # Returns: JSON array of tracked resource IDs
  # Returns: [] if no resources tracked
}

seed::clear_tracked_resources() {
  # Usage: seed::clear_tracked_resources
  # Removes seed-resources.json
  # Called by cleanup scripts after deletion
}

# ============================================================================
# Logging & Error Handling
# ============================================================================

seed::log_info() {
  # Usage: seed::log_info "Creating demo project..."
  # Outputs: 🌱 Creating demo project...
}

seed::log_success() {
  # Usage: seed::log_success "Seed data applied"
  # Outputs: ✅ Seed data applied
}

seed::log_error() {
  # Usage: seed::log_error "Failed to create project"
  # Outputs to stderr: ❌ Failed to create project
}

seed::log_warning() {
  # Usage: seed::log_warning "Project already exists, reusing"
  # Outputs: ⚠️  Project already exists, reusing
}

# ============================================================================
# Common Patterns
# ============================================================================

seed::cleanup_all_resources() {
  # Usage: seed::cleanup_all_resources "$api_url" "projects"
  # Generic cleanup: GET list → DELETE each item
  #
  # Args:
  #   $1: API base URL
  #   $2: Resource type (e.g., "projects", "workflows")
  #   $3: (optional) JSON path to ID field (default: ".id")
  #   $4: (optional) JSON path to array (default: ".${resource_type}[]")
  #
  # Example:
  #   seed::cleanup_all_resources "$api_url" "projects" ".project.id" ".projects[]"
}

seed::idempotent_create() {
  # Usage: id=$(seed::idempotent_create "$api_url" "projects" "name" "Demo Project" "$create_payload")
  #
  # Features:
  # - Checks if resource exists by search field
  # - Returns existing ID if found
  # - Creates new resource if not found
  # - Returns ID either way
  #
  # Args:
  #   $1: API base URL
  #   $2: Resource type (e.g., "projects")
  #   $3: Search field name (e.g., "name")
  #   $4: Search value
  #   $5: JSON payload for creation
}

# ============================================================================
# Standard Error Codes
# ============================================================================

readonly SEED_ERR_API_UNAVAILABLE=201
readonly SEED_ERR_INVALID_STATE=202
readonly SEED_ERR_CLEANUP_FAILED=203
readonly SEED_ERR_MISSING_TOOLS=204
readonly SEED_ERR_VALIDATION_FAILED=205
```

### Schema Validation: `seed-state.schema.json`

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "E2E Test Seed State",
  "description": "Persistent state from apply.sh used by test fixtures",
  "type": "object",
  "required": ["_metadata"],
  "properties": {
    "_metadata": {
      "type": "object",
      "required": ["scenario", "createdAt", "schemaVersion"],
      "properties": {
        "scenario": {
          "type": "string",
          "description": "Scenario name that created this seed state"
        },
        "createdAt": {
          "type": "string",
          "format": "date-time",
          "description": "ISO 8601 timestamp when seeds were applied"
        },
        "schemaVersion": {
          "type": "string",
          "enum": ["1.0"],
          "description": "Seed state schema version for future migrations"
        },
        "runId": {
          "type": "string",
          "description": "Unique identifier for this test run"
        }
      }
    }
  },
  "additionalProperties": true
}
```

**Usage in apply.sh**:
```bash
seed_payload=$(jq -n \
  --arg scenario "browser-automation-studio" \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg version "1.0" \
  --arg runId "$run_id" \
  --arg projectId "$project_id" \
  --arg projectName "$project_name" \
  '{
    _metadata: {
      scenario: $scenario,
      createdAt: $timestamp,
      schemaVersion: $version,
      runId: $runId
    },
    projectId: $projectId,
    projectName: $projectName
  }')

seed::write_state "$SCENARIO_DIR" "$seed_payload"
```

### Testing Utilities: `seed-testing.sh`

```bash
#!/usr/bin/env bash
# Testing utilities for seed scripts

seed::test::dry_run() {
  # Usage: seed::test::dry_run "$scenario_dir"
  # Validates seed scripts without side effects:
  # - Checks apply.sh/cleanup.sh exist and are executable
  # - Validates required tools available
  # - Checks API connectivity
  # - Validates seed-state.json structure (if exists)
  # Returns 0 if all checks pass
}

seed::test::round_trip() {
  # Usage: seed::test::round_trip "$scenario_dir"
  # Full lifecycle test:
  # 1. Runs apply.sh
  # 2. Validates seed-state.json created
  # 3. Optionally runs verify-cleanup.sh --pre
  # 4. Runs cleanup.sh
  # 5. Validates seed-state.json removed
  # 6. Optionally runs verify-cleanup.sh --post
  # Returns 0 if full cycle succeeds
}

seed::test::isolated() {
  # Usage: seed::test::isolated "$scenario_dir" [apply|cleanup]
  # Runs just apply.sh or cleanup.sh in isolation
  # Useful for debugging specific scripts
}

seed::test::verify_cleanup() {
  # Usage: seed::test::verify_cleanup "$scenario_dir"
  # Checks that cleanup fully removed all seed data:
  # - seed-state.json removed
  # - seed-resources.json removed
  # - Optional verify-cleanup.sh passes
}
```

### Scaffolding: `scaffold-seeds.sh`

```bash
#!/usr/bin/env bash
# Generates seed script boilerplate for a scenario

Usage: scripts/scenarios/testing/playbooks/scaffold-seeds.sh --scenario <path>

Creates:
  test/playbooks/__seeds/apply.sh         # Template with TODO markers
  test/playbooks/__seeds/cleanup.sh       # Template with TODO markers
  test/playbooks/__seeds/README.md        # Documentation template
  test/playbooks/__subflows/load-seed-state.json  # Fixture loader

Example apply.sh template:
```

Generated `apply.sh` template:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Load seed helpers
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "$SCRIPT_DIR/../../../../.." && pwd)"
source "${APP_ROOT}/scripts/scenarios/testing/shell/seed-helpers.sh"

# Resolve scenario details
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
SCENARIO_NAME="${SCENARIO_NAME:-$(basename "$SCENARIO_DIR")}"

# Validate prerequisites
seed::require_tools jq curl
seed::require_scenario_running "$SCENARIO_NAME"

# Get API connection details
API_PORT=$(seed::resolve_api_port "$SCENARIO_NAME" "API_PORT")
API_URL=$(seed::resolve_api_url "$SCENARIO_NAME" "/api/v1")

seed::log_info "Applying seeds for $SCENARIO_NAME..."

# TODO: Implement your seeding logic here
# Example pattern:

# 1. Generate unique identifiers
run_id=$(seed::generate_run_id)
resource_name=$(seed::generate_unique_name "Demo Resource")

# 2. Create resources via API
create_payload=$(jq -n \
  --arg name "$resource_name" \
  '{name: $name, description: "Seeded for e2e testing"}')

response=$(seed::curl_json POST "$API_URL/resources" "$create_payload")
resource_id=$(seed::extract_json_field "$response" ".resource.id // .id")

# 3. Track for cleanup
seed::track_resource "resource" "$resource_id"

# 4. Build seed state
seed_payload=$(jq -n \
  --arg scenario "$SCENARIO_NAME" \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg version "1.0" \
  --arg runId "$run_id" \
  --arg resourceId "$resource_id" \
  --arg resourceName "$resource_name" \
  '{
    _metadata: {
      scenario: $scenario,
      createdAt: $timestamp,
      schemaVersion: $version,
      runId: $runId
    },
    resourceId: $resourceId,
    resourceName: $resourceName
  }')

# 5. Validate and write
seed::validate_state "$seed_payload" resourceId resourceName
seed::write_state "$SCENARIO_DIR" "$seed_payload"

seed::log_success "Seeds applied (resource: $resource_id)"
```

Generated `cleanup.sh` template:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Load seed helpers
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "$SCRIPT_DIR/../../../../.." && pwd)"
source "${APP_ROOT}/scripts/scenarios/testing/shell/seed-helpers.sh"

# Resolve scenario details
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
SCENARIO_NAME="${SCENARIO_NAME:-$(basename "$SCENARIO_DIR")}"

# Check prerequisites (fail silently if scenario unavailable)
if ! seed::require_tools curl jq 2>/dev/null; then
  seed::cleanup_state "$SCENARIO_DIR"
  seed::clear_tracked_resources
  exit 0
fi

# Try to get API connection
if ! API_PORT=$(seed::resolve_api_port "$SCENARIO_NAME" "API_PORT" 2>/dev/null); then
  seed::cleanup_state "$SCENARIO_DIR"
  seed::clear_tracked_resources
  exit 0
fi

API_URL=$(seed::resolve_api_url "$SCENARIO_NAME" "/api/v1")

seed::log_info "Cleaning up seeds for $SCENARIO_NAME..."

# TODO: Implement your cleanup logic here
# Example patterns:

# Option 1: Clean up tracked resources
resource_ids=$(seed::get_tracked_resources "resource")
if [ -n "$resource_ids" ]; then
  printf '%s' "$resource_ids" | jq -r '.[]' | while read -r rid; do
    seed::curl_json DELETE "$API_URL/resources/$rid" >/dev/null 2>&1 || true
  done
fi

# Option 2: Generic cleanup all
seed::cleanup_all_resources "$API_URL" "resources" ".id" ".resources[]"

# Remove state files
seed::cleanup_state "$SCENARIO_DIR"
seed::clear_tracked_resources

seed::log_success "Seeds cleaned up"
```

Generated `README.md` template:

```markdown
# Seed Data for [Scenario Name]

Auto-generated by scaffold-seeds.sh. Edit this file to document your seed strategy.

## What Gets Seeded

<!-- TODO: Document what data is created -->
- **Resource Type**: Description of what gets created and why

## Seed State Keys

<!-- TODO: Document the keys in seed-state.json -->
The `apply.sh` script writes the following keys to `seed-state.json`:

- `resourceId`: Unique identifier for the seeded resource
- `resourceName`: Human-readable name of the resource

## Manual Testing

Test seeds in isolation:

```bash
# From scenario root
cd test/playbooks/__seeds

# Apply seeds
./apply.sh

# Verify state created
cat ../../../test/artifacts/runtime/seed-state.json

# Clean up
./cleanup.sh

# Verify cleanup
[ ! -f ../../../test/artifacts/runtime/seed-state.json ] && echo "✅ Cleaned"
```

## Required API Endpoints

<!-- TODO: Document API dependencies -->
- `POST /api/v1/resources` - Create resource
- `GET /api/v1/resources` - List resources
- `DELETE /api/v1/resources/:id` - Delete resource

## Fixtures Using This Seed

<!-- TODO: List subflows that depend on this seed -->
- `load-seed-state.json` - Loads seed state into workflow context
```

## Implementation Phases

### Phase 1: Foundation (Week 1)

**Goal**: Create reusable utilities without breaking existing BAS tests

**Deliverables**:
1. `scripts/scenarios/testing/shell/seed-helpers.sh`
   - Core utilities (API resolution, curl wrappers, state management)
   - ~200 lines, heavily documented
   - Zero breaking changes

2. `scripts/scenarios/testing/shell/seed-testing.sh`
   - dry_run, round_trip, isolated helpers
   - ~100 lines

3. `scripts/scenarios/testing/playbooks/seed-state.schema.json`
   - Minimal schema with `_metadata`
   - Validation optional (opt-in)

4. Documentation
   - `docs/testing/guides/seed-standardization.md`
   - API reference for all helpers
   - Migration guide

**Testing**:
- Verify helpers work in isolation
- No integration with existing tests yet
- Unit tests for each helper function

**Success Criteria**:
- All helpers executable and documented
- No impact on existing BAS tests
- Ready for Phase 2 integration

---

### Phase 2: BAS Migration (Week 2)

**Goal**: Migrate browser-automation-studio to use new helpers

**Deliverables**:
1. Refactor `browser-automation-studio/test/playbooks/__seeds/apply.sh`
   - Replace custom logic with `seed::*` helpers
   - Reduce from 177 lines → ~80 lines
   - Add `_metadata` to seed-state.json
   - Track resources via `seed::track_resource`

2. Refactor `browser-automation-studio/test/playbooks/__seeds/cleanup.sh`
   - Use `seed::cleanup_all_resources`
   - Reduce from 46 lines → ~30 lines

3. Add `verify-cleanup.sh` (optional)
   - Validates no projects/workflows remain

4. Update `load-seed-state.json`
   - Handle new `_metadata` field gracefully

**Testing**:
- Full integration test suite must pass
- Seed round-trip testing
- Verify cleanup completeness

**Success Criteria**:
- All BAS tests green
- Reduced code duplication
- Better error messages
- Seed state includes metadata

---

### Phase 3: Scaffolding & Documentation (Week 3)

**Goal**: Make it easy for other scenarios to adopt

**Deliverables**:
1. `scripts/scenarios/testing/playbooks/scaffold-seeds.sh`
   - Generate apply.sh, cleanup.sh, README.md templates
   - Include TODO markers for customization

2. Enhanced documentation
   - Step-by-step tutorial: "Adding E2E Tests to Your Scenario"
   - Common patterns cookbook
   - Troubleshooting guide

3. BAS CLI integration (optional)
   - `browser-automation-studio seeds scaffold`
   - `browser-automation-studio seeds test`
   - `browser-automation-studio seeds validate`

**Testing**:
- Generate scaffolding for test scenario
- Verify templates are functional out-of-box
- Documentation review

**Success Criteria**:
- Any developer can scaffold seeds in <5 minutes
- Documentation answers common questions
- Clear path from scaffold → working tests

---

### Phase 4: Pilot Adoption (Week 4)

**Goal**: Validate with 1-2 other scenarios

**Pilot Candidates**:
1. `landing-manager` (simple: just templates/variants)
2. `deployment-manager` (medium complexity: deployments/profiles)

**Deliverables**:
1. Implement seeds for pilot scenarios using scaffolding
2. Document pain points and edge cases
3. Iterate on helpers based on feedback

**Testing**:
- Pilot scenarios' integration tests pass
- Verify isolation between test runs
- Stress test cleanup reliability

**Success Criteria**:
- 2 additional scenarios using seed helpers
- No critical issues found
- Helpers proven generic (not BAS-specific)

---

### Phase 5: Ecosystem Rollout (Ongoing)

**Goal**: Enable remaining scenarios as needed

**Activities**:
1. Announce standardization in team docs
2. Update scenario templates to include seed scaffolding
3. Office hours for teams implementing seeds
4. Collect feedback and iterate

**Success Metrics**:
- 50%+ of scenarios with UI use seed helpers within 3 months
- <10% of seed-related test failures
- Positive developer feedback

## Detailed Examples

### Example 1: Simple Seed (landing-manager)

**Scenario**: landing-manager needs to seed templates and variants for A/B testing.

**apply.sh** (using helpers):

```bash
#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)/scripts/scenarios/testing/shell/seed-helpers.sh"

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SCENARIO_NAME="landing-manager"

seed::require_tools jq curl
seed::require_scenario_running "$SCENARIO_NAME"

API_URL=$(seed::resolve_api_url "$SCENARIO_NAME" "/api/v1")
run_id=$(seed::generate_run_id)

# Create test template
template_payload=$(jq -n \
  --arg name "Test Template $run_id" \
  '{name: $name, content: "Hello {name}"}')

template_resp=$(seed::curl_json POST "$API_URL/templates" "$template_payload")
template_id=$(seed::extract_json_field "$template_resp" ".template.id")
seed::track_resource "template" "$template_id"

# Create variant A
variant_a_payload=$(jq -n \
  --arg tid "$template_id" \
  --arg name "Variant A" \
  '{template_id: $tid, name: $name, headline: "Welcome!"}')

variant_a_resp=$(seed::curl_json POST "$API_URL/variants" "$variant_a_payload")
variant_a_id=$(seed::extract_json_field "$variant_a_resp" ".variant.id")
seed::track_resource "variant" "$variant_a_id"

# Write state
seed_state=$(jq -n \
  --arg scenario "$SCENARIO_NAME" \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg version "1.0" \
  --arg runId "$run_id" \
  --arg templateId "$template_id" \
  --arg variantAId "$variant_a_id" \
  '{
    _metadata: {scenario: $scenario, createdAt: $timestamp, schemaVersion: $version, runId: $runId},
    templateId: $templateId,
    variantAId: $variantAId
  }')

seed::write_state "$SCENARIO_DIR" "$seed_state"
seed::log_success "Landing manager seeds applied"
```

**cleanup.sh** (using helpers):

```bash
#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)/scripts/scenarios/testing/shell/seed-helpers.sh"

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SCENARIO_NAME="landing-manager"

if ! API_URL=$(seed::resolve_api_url "$SCENARIO_NAME" "/api/v1" 2>/dev/null); then
  seed::cleanup_state "$SCENARIO_DIR"
  seed::clear_tracked_resources
  exit 0
fi

# Clean tracked variants
seed::get_tracked_resources "variant" | jq -r '.[]' | while read -r vid; do
  seed::curl_json DELETE "$API_URL/variants/$vid" >/dev/null 2>&1 || true
done

# Clean tracked templates
seed::get_tracked_resources "template" | jq -r '.[]' | while read -r tid; do
  seed::curl_json DELETE "$API_URL/templates/$tid" >/dev/null 2>&1 || true
done

seed::cleanup_state "$SCENARIO_DIR"
seed::clear_tracked_resources
seed::log_success "Landing manager seeds cleaned"
```

**Complexity**: ~40 lines total vs. ~200 if implemented from scratch

---

### Example 2: Complex Seed (deployment-manager)

**Scenario**: deployment-manager needs profiles, secrets, and multi-tier deployments.

**apply.sh** (abbreviated):

```bash
# ... standard boilerplate ...

# Create test profile
profile_id=$(seed::idempotent_create \
  "$API_URL" "profiles" "name" "Test Profile $run_id" \
  "$(jq -n --arg name "Test Profile $run_id" '{name: $name, tier: "local"}')")
seed::track_resource "profile" "$profile_id"

# Create secret vault
vault_payload=$(jq -n --arg pid "$profile_id" '{profile_id: $pid, type: "vault"}')
vault_resp=$(seed::curl_json POST "$API_URL/secrets" "$vault_payload")
vault_id=$(seed::extract_json_field "$vault_resp" ".secret.id")
seed::track_resource "secret" "$vault_id"

# Create deployment config
deploy_payload=$(jq -n \
  --arg pid "$profile_id" \
  --arg scenario "test-service" \
  '{profile_id: $pid, scenario: $scenario, config: {replicas: 1}}')
deploy_resp=$(seed::curl_json POST "$API_URL/deployments" "$deploy_payload")
deploy_id=$(seed::extract_json_field "$deploy_resp" ".deployment.id")
seed::track_resource "deployment" "$deploy_id"

# Write state with multiple IDs
seed_state=$(jq -n \
  --arg scenario "$SCENARIO_NAME" \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg version "1.0" \
  --arg runId "$run_id" \
  --arg profileId "$profile_id" \
  --arg vaultId "$vault_id" \
  --arg deploymentId "$deploy_id" \
  '{
    _metadata: {scenario: $scenario, createdAt: $timestamp, schemaVersion: $version, runId: $runId},
    profileId: $profileId,
    vaultId: $vaultId,
    deploymentId: $deploymentId
  }')

seed::write_state "$SCENARIO_DIR" "$seed_state"
```

**Key features**:
- Uses `idempotent_create` for profile (can retry if network flakes)
- Tracks resources in creation order
- Cleanup happens in reverse order (deployments → secrets → profiles)

---

### Example 3: Using Seed State in Tests

**Workflow**: `test/playbooks/capabilities/templates/variant-switching.json`

```json
{
  "metadata": {
    "description": "Verify A/B variant switching",
    "version": 1,
    "reset": "none"
  },
  "nodes": [
    {
      "id": "load-seeds",
      "type": "workflowCall",
      "data": {
        "label": "Load seed state",
        "workflowId": "@fixture/load-seed-state"
      }
    },
    {
      "id": "navigate-variant-a",
      "type": "navigate",
      "data": {
        "label": "Navigate to variant A",
        "destinationType": "url",
        "url": "${BASE_URL}/variants/${fixture.variantAId}",
        "waitUntil": "networkidle0"
      }
    },
    {
      "id": "assert-headline",
      "type": "assert",
      "data": {
        "label": "Verify variant A headline",
        "selector": "h1",
        "assertMode": "text",
        "expectedText": "Welcome!",
        "failureMessage": "Variant A should show 'Welcome!' headline"
      }
    }
  ],
  "edges": [
    {"id": "e1", "source": "load-seeds", "target": "navigate-variant-a"},
    {"id": "e2", "source": "navigate-variant-a", "target": "assert-headline"}
  ]
}
```

**Resolution flow**:
1. `@fixture/load-seed-state` calls → reads `seed-state.json`
2. Injects `${fixture.variantAId}` → actual UUID from seeds
3. Workflow executes with real data
4. No hardcoded IDs anywhere

## Migration Path for Existing Scenarios

### For Scenarios Without Seeds

**Current State**: Using `.gitkeep` placeholder

**Migration Steps**:
1. Run scaffolding: `scripts/scenarios/testing/playbooks/scaffold-seeds.sh --scenario scenarios/my-scenario`
2. Edit generated `apply.sh` to create your test data
3. Edit generated `cleanup.sh` if custom cleanup needed
4. Update phase test script to enable BAS workflow execution
5. Write first test playbook referencing `@seed/` tokens

**Estimated Effort**: 2-4 hours for simple scenarios

---

### For Browser-automation-studio (Already Has Seeds)

**Current State**: Custom implementation in apply.sh/cleanup.sh

**Migration Steps**:
1. Update `apply.sh`:
   ```bash
   # Add at top
   source "${APP_ROOT}/scripts/scenarios/testing/shell/seed-helpers.sh"

   # Replace custom functions
   - curl_json() { ... }              → seed::curl_json
   - require_tool() { ... }           → seed::require_tools
   - Manual state writing             → seed::write_state
   - Manual JSON extraction           → seed::extract_json_field
   ```

2. Update `cleanup.sh`:
   ```bash
   # Replace list + delete loops
   - Manual curl + jq parsing         → seed::cleanup_all_resources
   - Manual state file removal        → seed::cleanup_state
   ```

3. Add metadata to seed-state.json:
   ```json
   {
     "_metadata": {
       "scenario": "browser-automation-studio",
       "createdAt": "2025-11-23T10:30:00Z",
       "schemaVersion": "1.0",
       "runId": "1763874256-dd0107"
     },
     "projectId": "...",
     "projectName": "...",
     "workflowId": "...",
     "workflowName": "..."
   }
   ```

4. Test full integration suite
5. Deploy alongside documentation

**Estimated Effort**: 4-6 hours (includes testing)

**Breaking Changes**: None (backward compatible)

## Success Metrics

### Quantitative

1. **Adoption Rate**:
   - Target: 50% of scenarios with UI tests using seed helpers within 3 months
   - Measure: Count scenarios with `test/playbooks/__seeds/apply.sh` using `seed::`

2. **Code Reduction**:
   - Target: 60% reduction in seed script LOC per scenario
   - Baseline: BAS apply.sh (177 lines) + cleanup.sh (46 lines) = 223 lines
   - Target: <90 lines using helpers

3. **Time to Implement**:
   - Target: <4 hours from scaffold to working tests
   - Measure: Track time for pilot scenarios

4. **Test Reliability**:
   - Target: <5% seed-related failures
   - Measure: Count failures with "seed" in error message / total test runs

### Qualitative

1. **Developer Experience**:
   - Survey: "Seed helpers made e2e testing easier" (5-point scale)
   - Target: >4.0 average

2. **Documentation Quality**:
   - Survey: "I could implement seeds without asking for help"
   - Target: >80% yes

3. **Maintenance Burden**:
   - Question: "How often do seed scripts break?"
   - Target: "Rarely" or "Never" >90%

## Risks & Mitigations

### Risk 1: Breaking Changes to BAS

**Probability**: Low
**Impact**: High

**Mitigation**:
- Phase 2 is fully backward-compatible (helpers don't change behavior)
- Extensive integration testing before merge
- Feature flag for new helpers (fall back to old logic if issues)

---

### Risk 2: Helpers Too Generic (Don't Fit Edge Cases)

**Probability**: Medium
**Impact**: Medium

**Mitigation**:
- Pilot with diverse scenarios (simple + complex)
- Escape hatch: scenarios can always implement custom logic
- Helpers are opt-in, not required

---

### Risk 3: Poor Documentation Adoption

**Probability**: Medium
**Impact**: Medium

**Mitigation**:
- Scaffolding generates 80% of needed code
- Inline TODO comments guide customization
- Office hours for first few adopters
- Video walkthrough for documentation

---

### Risk 4: Schema Versioning Issues

**Probability**: Low
**Impact**: Low

**Mitigation**:
- Start with minimal schema (just `_metadata`)
- Additional fields are optional (additionalProperties: true)
- No validation enforced initially (opt-in)
- Migration path defined if schema changes

## Open Questions

1. **Should seed helpers support transactions/rollback?**
   - Complexity: High
   - Value: Medium (handles partial failures)
   - Decision: Defer to Phase 5+ (add if needed)

2. **Should we track seed performance metrics?**
   - Example: Time to apply seeds, cleanup duration
   - Value: Nice-to-have for optimization
   - Decision: Add basic timing in Phase 3

3. **Should seed state be version-controlled for snapshots?**
   - Current: Runtime artifact only
   - Alternative: Optional `seed-state.example.json` for docs
   - Decision: Add example file in Phase 3

4. **Should cleanup verify database state?**
   - Current: DELETE requests assumed to work
   - Alternative: Optional `verify-cleanup.sh` checks DB
   - Decision: Support optional verification in Phase 2

## Related Documentation

**Prerequisites**:
- [E2E Testing Architecture](../testing/architecture/PHASED_TESTING.md)
- [UI Automation with BAS](../testing/guides/ui-automation-with-bas.md)
- [Requirement Tracking](../testing/guides/requirement-tracking.md)

**Will Create**:
- `docs/testing/guides/seed-standardization.md` (API reference)
- `docs/testing/guides/implementing-e2e-seeds.md` (tutorial)
- `docs/testing/reference/seed-helpers-api.md` (function reference)

**Will Update**:
- `docs/testing/guides/scenario-testing.md` (add seeding section)
- `docs/testing/guides/troubleshooting.md` (add seed debugging)

## Appendix: Full API Reference

See implementation in Phase 1 for complete function signatures, examples, and error handling.

**Categories**:
- Environment Resolution (2 functions)
- Prerequisite Validation (2 functions)
- HTTP Utilities (2 functions)
- Unique ID Generation (2 functions)
- State File Management (5 functions)
- Resource Tracking (3 functions)
- Logging & Error Handling (4 functions)
- Common Patterns (2 functions)
- Error Codes (5 constants)

**Total**: ~20 exported functions, ~400 lines of code with comprehensive error handling and documentation.
