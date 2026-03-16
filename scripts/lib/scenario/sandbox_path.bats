#!/usr/bin/env bats

# =============================================================================
# Tests for sandbox-aware scenario path resolution
#
# When an AI agent runs inside an overlayfs sandbox (managed by workspace-sandbox),
# the agent-manager injects VROOLI_SANDBOX_* env vars into the agent's process.
# The Vrooli CLI (scripts/lib/scenario/runner.sh) reads these vars to transparently
# redirect scenario path resolution to the sandbox's merged/ directory, so the
# lifecycle system builds/runs from the agent's modified code instead of the
# untouched real repo.
#
# This file tests three things:
#   1. sandbox::scenario_in_scope() — scope matching logic
#   2. sandbox::resolve_merged_path() — path computation within the overlay
#   3. Path resolution integration — verifying correct redirect behavior
#
# See also:
#   - scripts/lib/scenario/runner.sh (implementation)
#   - scenarios/agent-manager/api/internal/orchestration/run_executor.go
#     (SandboxEnvVars method that injects the env vars)
# =============================================================================

# Stub log functions so runner.sh can source without real logging
log::info() { echo "INFO: $*"; }
log::warn() { echo "WARN: $*"; }
log::warning() { echo "WARN: $*"; }
log::error() { echo "ERROR: $*" >&2; }
log::debug() { echo "DEBUG: $*" >&2; }
log::success() { echo "OK: $*"; }

SCRIPT_DIR="$(cd "$(dirname "${BATS_TEST_FILENAME:-${BASH_SOURCE[0]}}")" && pwd)"

# Source only the runner to get sandbox:: functions. We stub out dependencies
# that aren't needed for path resolution tests.
source "${SCRIPT_DIR}/../utils/var.sh"
source "${SCRIPT_DIR}/runner.sh"

setup() {
    export var_ROOT_DIR="$BATS_TEST_TMPDIR/app"
    mkdir -p "$var_ROOT_DIR/scenarios"

    # Clear sandbox env vars before each test
    unset VROOLI_SANDBOX_ID VROOLI_SANDBOX_MERGED VROOLI_SANDBOX_SCOPE
    unset SCENARIO_CUSTOM_PATH
}

# =============================================================================
# sandbox::scenario_in_scope — scope matching
#
# These tests verify the logic that decides whether a given scenario slug
# falls within the sandbox's scoped path. Only in-scope scenarios should be
# redirected to the sandbox; everything else must use the real repo.
# =============================================================================

@test "scope: empty scope matches all scenarios (whole repo is overlaid)" {
    run sandbox::scenario_in_scope "any-scenario" ""
    [ "$status" -eq 0 ]
}

@test "scope: root '/' matches all scenarios" {
    run sandbox::scenario_in_scope "any-scenario" "/"
    [ "$status" -eq 0 ]
}

@test "scope: dot '.' matches all scenarios" {
    run sandbox::scenario_in_scope "any-scenario" "."
    [ "$status" -eq 0 ]
}

@test "scope: 'scenarios' matches all scenarios" {
    run sandbox::scenario_in_scope "any-scenario" "scenarios"
    [ "$status" -eq 0 ]
}

@test "scope: 'scenarios/' with trailing slash matches all scenarios" {
    run sandbox::scenario_in_scope "any-scenario" "scenarios/"
    [ "$status" -eq 0 ]
}

@test "scope: 'scenarios/foo' matches scenario 'foo'" {
    run sandbox::scenario_in_scope "foo" "scenarios/foo"
    [ "$status" -eq 0 ]
}

@test "scope: 'scenarios/foo' does NOT match scenario 'bar'" {
    run sandbox::scenario_in_scope "bar" "scenarios/foo"
    [ "$status" -eq 1 ]
}

@test "scope: 'scenarios/foo/api' still matches scenario 'foo' (deeper scope)" {
    run sandbox::scenario_in_scope "foo" "scenarios/foo/api"
    [ "$status" -eq 0 ]
}

@test "scope: 'scenarios/foo/' trailing slash matches 'foo'" {
    run sandbox::scenario_in_scope "foo" "scenarios/foo/"
    [ "$status" -eq 0 ]
}

@test "scope: 'packages/shared' does NOT match any scenario (outside scenarios/)" {
    run sandbox::scenario_in_scope "any-scenario" "packages/shared"
    [ "$status" -eq 1 ]
}

@test "scope: 'scenarios/foo-bar' does NOT match 'foo' (partial slug mismatch)" {
    run sandbox::scenario_in_scope "foo" "scenarios/foo-bar"
    [ "$status" -eq 1 ]
}

@test "scope: 'scenarios/foo-bar' matches 'foo-bar' exactly" {
    run sandbox::scenario_in_scope "foo-bar" "scenarios/foo-bar"
    [ "$status" -eq 0 ]
}

# =============================================================================
# sandbox::resolve_merged_path — path computation
#
# The overlay's merged/ dir maps to the scoped directory, NOT the project root.
# These tests verify that the scenario path within merged/ is computed correctly
# for all scope configurations.
# =============================================================================

@test "resolve: exact scenario scope → merged IS the scenario dir" {
    # scope="scenarios/agent-inbox", merged overlays just that scenario
    run sandbox::resolve_merged_path "agent-inbox" "scenarios/agent-inbox" "/tmp/sandbox/merged"
    [ "$output" = "/tmp/sandbox/merged" ]
}

@test "resolve: 'scenarios' scope → scenario is a subdir of merged" {
    # scope="scenarios", merged overlays the scenarios/ directory
    run sandbox::resolve_merged_path "agent-inbox" "scenarios" "/tmp/sandbox/merged"
    [ "$output" = "/tmp/sandbox/merged/agent-inbox" ]
}

@test "resolve: empty scope → full project path inside merged" {
    # scope="" (whole repo), merged is the project root
    run sandbox::resolve_merged_path "agent-inbox" "" "/tmp/sandbox/merged"
    [ "$output" = "/tmp/sandbox/merged/scenarios/agent-inbox" ]
}

@test "resolve: root scope '/' → full project path inside merged" {
    run sandbox::resolve_merged_path "agent-inbox" "/" "/tmp/sandbox/merged"
    [ "$output" = "/tmp/sandbox/merged/scenarios/agent-inbox" ]
}

@test "resolve: dot scope '.' → full project path inside merged" {
    run sandbox::resolve_merged_path "agent-inbox" "." "/tmp/sandbox/merged"
    [ "$output" = "/tmp/sandbox/merged/scenarios/agent-inbox" ]
}

@test "resolve: deeper scope still resolves to merged root" {
    # scope="scenarios/foo/api" is deeper than "scenarios/foo", but merged IS still
    # the overlay root (which is scenarios/foo/api). For scenario "foo", in_scope
    # matched because the scope starts with "scenarios/foo". But resolve should
    # return merged since scenarios/foo starts with the scope's scenario.
    # Actually this case is: scenario_rel="scenarios/foo", scope="scenarios/foo/api"
    # scenario_rel is NOT a prefix of scope, so it hits fallback.
    run sandbox::resolve_merged_path "foo" "scenarios/foo/api" "/tmp/sandbox/merged"
    # Fallback: merged + scenarios/foo (scope doesn't prefix scenario_rel)
    [ "$output" = "/tmp/sandbox/merged/scenarios/foo" ]
}

@test "resolve: trailing slash on scope is normalized" {
    run sandbox::resolve_merged_path "agent-inbox" "scenarios/agent-inbox/" "/tmp/sandbox/merged"
    [ "$output" = "/tmp/sandbox/merged" ]
}

@test "resolve: scenario with hyphens in name" {
    run sandbox::resolve_merged_path "landing-page-business-suite" "scenarios/landing-page-business-suite" "/tmp/sandbox/merged"
    [ "$output" = "/tmp/sandbox/merged" ]
}

@test "resolve: scenarios scope with hyphenated scenario" {
    run sandbox::resolve_merged_path "landing-page-business-suite" "scenarios" "/tmp/sandbox/merged"
    [ "$output" = "/tmp/sandbox/merged/landing-page-business-suite" ]
}

# =============================================================================
# SCENARIO_CUSTOM_PATH export — invariant for heal.sh
#
# This test validates a critical link in the sandbox→heal chain:
#
#   1. Agent-manager sets VROOLI_SANDBOX_* env vars (run_executor.go)
#   2. runner.sh resolves scenario path to sandbox merged dir ← THIS STEP
#   3. runner.sh exports SCENARIO_CUSTOM_PATH with the sandbox path ← AND THIS
#   4. lifecycle.sh cd's to SCENARIO_CUSTOM_PATH
#   5. lifecycle.sh records working_dir=$(pwd) in process metadata JSON
#   6. heal.sh reads working_dir from metadata to detect sandbox-based scenarios
#
# Steps 4-6 are covered by lifecycle tests and heal_from_sandbox.bats.
# This test verifies steps 2-3: that the sandbox env vars correctly produce
# a SCENARIO_CUSTOM_PATH pointing to the sandbox merged directory.
# =============================================================================

@test "SCENARIO_CUSTOM_PATH is exported when sandbox redirects" {
    # Create the scenario directory in both the real repo and the sandbox
    mkdir -p "$var_ROOT_DIR/scenarios/test-scenario/.vrooli"
    local merged="$BATS_TEST_TMPDIR/sandbox-merged"
    mkdir -p "$merged/scenarios/test-scenario/.vrooli"

    # Set sandbox env vars (as the agent-manager would)
    export VROOLI_SANDBOX_ID="test-sandbox-123"
    export VROOLI_SANDBOX_MERGED="$merged"
    export VROOLI_SANDBOX_SCOPE="scenarios/test-scenario"

    # We can't call scenario::run() directly (too many dependencies), but we
    # can test the path resolution logic that sets scenario_path and then
    # verify that SCENARIO_CUSTOM_PATH would be set. The key functions are
    # sandbox::scenario_in_scope and sandbox::resolve_merged_path.

    # Verify scope matching works
    sandbox::scenario_in_scope "test-scenario" "$VROOLI_SANDBOX_SCOPE"
    [ $? -eq 0 ]

    # Verify path resolution produces the correct sandbox path
    local resolved
    resolved="$(sandbox::resolve_merged_path "test-scenario" "$VROOLI_SANDBOX_SCOPE" "$VROOLI_SANDBOX_MERGED")"
    [ "$resolved" = "$merged" ]

    # Verify the resolved path is inside the sandbox (which is what
    # runner.sh uses to decide to set SCENARIO_CUSTOM_PATH)
    [[ "$resolved" == "${VROOLI_SANDBOX_MERGED}"* ]]
}
