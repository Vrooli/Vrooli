#!/usr/bin/env bats

# =============================================================================
# Tests for the heal-from-sandbox command (two-phase architecture)
#
# This command is called by workspace-sandbox pre-teardown hooks to detect
# scenarios running from a sandbox's merged/ directory, stop them before the
# overlay filesystem is unmounted, and restart them from the canonical repo
# in the background.
#
# The tests verify:
#   - Detection: correctly identifies scenarios whose working_dir is inside
#     the sandbox merged path, and ignores those running from the real repo
#     or different sandboxes
#   - Two-phase ordering: all stops complete before any restarts begin
#   - Failure handling: restart failures for one scenario don't block others
#   - Argument/env var handling: --merged-path, --dry-run, SANDBOX_MERGED_DIR
#
# See also:
#   - cli/commands/scenario/modules/heal.sh (implementation)
#   - scenarios/workspace-sandbox/api/internal/policy/teardown.go (hook caller)
#   - scripts/lib/scenario/runner.sh (sandbox-aware path resolution)
# =============================================================================

# Stub log functions
log::info() { echo "INFO: $*"; }
log::warn() { echo "WARN: $*"; }
log::warning() { echo "WARN: $*"; }
log::error() { echo "ERROR: $*" >&2; }
log::debug() { echo "DEBUG: $*" >&2; }
log::success() { echo "OK: $*"; }

SCRIPT_DIR="$(cd "$(dirname "${BATS_TEST_FILENAME:-${BASH_SOURCE[0]}}")" && pwd)"

# Source var.sh and heal.sh with the real APP_ROOT so the source succeeds.
source "${SCRIPT_DIR}/../utils/var.sh"
export APP_ROOT="$var_ROOT_DIR"
source "${var_ROOT_DIR}/cli/commands/scenario/modules/heal.sh"

setup() {
    # Create temporary process metadata directory
    export HOME="$BATS_TEST_TMPDIR"
    mkdir -p "$BATS_TEST_TMPDIR/.vrooli/processes/scenarios"

    # Point APP_ROOT to a nonexistent path so that when
    # scenario::heal::from_sandbox() runs, its internal
    # `source "${APP_ROOT}/scripts/lib/utils/lifecycle.sh"` fails silently
    # (it has `|| true`), preventing it from overriding our stubs.
    export APP_ROOT="/nonexistent/fake-app-root"

    # File-based call tracking (works across subshells, unlike arrays).
    export _HEAL_TRACK_DIR="$BATS_TEST_TMPDIR/heal-tracking"
    mkdir -p "$_HEAL_TRACK_DIR"
    : > "$_HEAL_TRACK_DIR/calls.log"

    # Redefine lifecycle stubs before each test. These record their calls
    # to a tracking file so tests can verify ordering and behavior.
    eval 'lifecycle::stop_scenario_processes() {
        echo "stop:$1" >> "$_HEAL_TRACK_DIR/calls.log"
        return 0
    }'
    eval 'scenario::lifecycle::start() {
        echo "start:$1" >> "$_HEAL_TRACK_DIR/calls.log"
        return 0
    }'

    # Clear sandbox env vars
    unset SANDBOX_MERGED_DIR
}

# Helper: create mock process metadata for a scenario
create_process_metadata() {
    local scenario_name="$1"
    local working_dir="$2"
    local dir="$BATS_TEST_TMPDIR/.vrooli/processes/scenarios/$scenario_name"
    mkdir -p "$dir"
    cat > "$dir/start-api.json" <<EOF
{
    "pid": 12345,
    "scenario": "$scenario_name",
    "step": "start-api",
    "working_dir": "$working_dir",
    "status": "running"
}
EOF
}

# =============================================================================
# Detection tests — verify that heal correctly identifies affected scenarios
# =============================================================================

@test "heal: exits 0 when no process metadata exists" {
    export SANDBOX_MERGED_DIR="/tmp/sandbox/abc123/merged"
    run scenario::heal::from_sandbox --dry-run
    [ "$status" -eq 0 ]
}

@test "heal: exits 0 when process dir is empty" {
    export SANDBOX_MERGED_DIR="/tmp/sandbox/abc123/merged"
    mkdir -p "$BATS_TEST_TMPDIR/.vrooli/processes/scenarios/empty-scenario"
    run scenario::heal::from_sandbox --dry-run
    [ "$status" -eq 0 ]
}

@test "heal: detects scenario running from sandbox merged path" {
    local merged="/tmp/sandbox/abc123/merged"
    create_process_metadata "my-scenario" "$merged/scenarios/my-scenario"

    export SANDBOX_MERGED_DIR="$merged"
    run scenario::heal::from_sandbox --dry-run
    [ "$status" -eq 0 ]
    [[ "$output" == *"my-scenario"* ]]
    [[ "$output" == *"dry-run"* ]]
}

@test "heal: ignores scenarios running from real repo" {
    create_process_metadata "real-scenario" "/home/user/Vrooli/scenarios/real-scenario"

    export SANDBOX_MERGED_DIR="/tmp/sandbox/abc123/merged"
    run scenario::heal::from_sandbox --dry-run
    [ "$status" -eq 0 ]
    [[ "$output" != *"scenario(s) affected"* ]]
}

@test "heal: ignores scenarios running from a different sandbox" {
    create_process_metadata "other-scenario" "/tmp/sandbox/DIFFERENT/merged/scenarios/other-scenario"

    export SANDBOX_MERGED_DIR="/tmp/sandbox/abc123/merged"
    run scenario::heal::from_sandbox --dry-run
    [ "$status" -eq 0 ]
    [[ "$output" != *"scenario(s) affected"* ]]
}

@test "heal: detects multiple scenarios from same sandbox" {
    local merged="/tmp/sandbox/abc123/merged"
    create_process_metadata "scenario-a" "$merged/scenarios/scenario-a"
    create_process_metadata "scenario-b" "$merged/scenarios/scenario-b"

    export SANDBOX_MERGED_DIR="$merged"
    run scenario::heal::from_sandbox --dry-run
    [ "$status" -eq 0 ]
    [[ "$output" == *"2 scenario(s) affected"* ]]
}

@test "heal: errors when no merged path provided" {
    unset SANDBOX_MERGED_DIR
    run scenario::heal::from_sandbox
    [ "$status" -eq 1 ]
    [[ "$output" == *"no merged path provided"* ]]
}

@test "heal: --merged-path flag overrides env var" {
    local merged="/tmp/sandbox/abc123/merged"
    create_process_metadata "my-scenario" "$merged/scenarios/my-scenario"

    export SANDBOX_MERGED_DIR="/wrong/path"
    run scenario::heal::from_sandbox --merged-path "$merged" --dry-run
    [ "$status" -eq 0 ]
    [[ "$output" == *"my-scenario"* ]]
}

# =============================================================================
# Two-phase ordering test
#
# This verifies that ALL scenarios are stopped before ANY restarts begin.
# If restarts started before all stops completed, a slow restart could push
# us past the hook timeout, leaving some scenarios un-stopped when the
# overlay unmounts — they'd become orphaned with no filesystem.
# =============================================================================

@test "heal: stops all scenarios before restarting any" {
    local merged="/tmp/sandbox/abc123/merged"
    create_process_metadata "scenario-a" "$merged/scenarios/scenario-a"
    create_process_metadata "scenario-b" "$merged/scenarios/scenario-b"

    export SANDBOX_MERGED_DIR="$merged"

    # Run heal (not dry-run) — stubs record calls to tracking file
    scenario::heal::from_sandbox

    # Wait briefly for background restarts to write their tracking entries
    sleep 1

    local log_file="$_HEAL_TRACK_DIR/calls.log"
    [ -f "$log_file" ]

    # Verify both stops happened
    local stop_count start_count
    stop_count=$(grep -c "^stop:" "$log_file")
    start_count=$(grep -c "^start:" "$log_file")
    [ "$stop_count" -eq 2 ]
    [ "$start_count" -eq 2 ]

    # Verify ordering: the last stop must appear before the first start.
    local last_stop_line first_start_line
    last_stop_line=$(grep -n "^stop:" "$log_file" | tail -1 | cut -d: -f1)
    first_start_line=$(grep -n "^start:" "$log_file" | head -1 | cut -d: -f1)
    [ "$last_stop_line" -lt "$first_start_line" ]
}

# =============================================================================
# Restart failure handling
#
# When one scenario fails to restart, the heal command should still attempt
# to restart the remaining scenarios. A single failure must not block the
# entire heal operation.
# =============================================================================

@test "heal: continues when restart fails for one scenario" {
    local merged="/tmp/sandbox/abc123/merged"
    create_process_metadata "fail-scenario" "$merged/scenarios/fail-scenario"
    create_process_metadata "ok-scenario" "$merged/scenarios/ok-scenario"

    # Override start stub to fail for one specific scenario
    eval 'scenario::lifecycle::start() {
        echo "start:$1" >> "$_HEAL_TRACK_DIR/calls.log"
        if [[ "$1" == "fail-scenario" ]]; then
            return 1
        fi
        return 0
    }'

    export SANDBOX_MERGED_DIR="$merged"
    scenario::heal::from_sandbox

    # Wait briefly for background restarts
    sleep 1

    # Both restarts should have been attempted regardless of the failure
    local log_file="$_HEAL_TRACK_DIR/calls.log"
    local start_count
    start_count=$(grep -c "^start:" "$log_file")
    [ "$start_count" -eq 2 ]
}
