#!/usr/bin/env bats

# Self-contained BATS tests for hooks.sh.
# Stubs out external dependencies so no test fixture infrastructure is needed.

# BATS setup function - runs before each test
setup() {
    CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    APP_ROOT="$(builtin cd "$CLAUDE_CODE_DIR/../.." && builtin pwd)"
    export APP_ROOT

    # Create temp directories for test settings files
    TEST_TMP_DIR=$(mktemp -d)
    TEST_PROJECT_DIR="$TEST_TMP_DIR/project/.claude"
    TEST_GLOBAL_DIR="$TEST_TMP_DIR/global/.claude"
    mkdir -p "$TEST_PROJECT_DIR" "$TEST_GLOBAL_DIR"

    export CLAUDE_PROJECT_SETTINGS="$TEST_PROJECT_DIR/settings.json"
    export CLAUDE_SETTINGS_FILE="$TEST_GLOBAL_DIR/settings.json"

    # Stub log functions (hooks.sh uses log::error)
    log::error() { echo "ERROR: $*" >&2; }
    log::warn() { echo "WARN: $*" >&2; }
    log::info() { echo "INFO: $*" >&2; }
    log::debug() { :; }
    export -f log::error log::warn log::info log::debug

    # Stub system::is_command — jq is available
    system::is_command() {
        case "$1" in
            jq) command -v jq &>/dev/null ;;
            *) return 0 ;;
        esac
    }
    export -f system::is_command

    # Disable the source guard so we can re-source
    unset _CLAUDE_CODE_HOOKS_SOURCED

    # Source hooks.sh directly (skip its var.sh/trash.sh sourcing by pre-defining)
    # We need to make source of var.sh and trash.sh succeed harmlessly
    _hooks_source_dir="$BATS_TEST_DIRNAME"

    # Source the script under test in a way that tolerates missing var.sh
    (
        # Provide stubs for things hooks.sh tries to source
        source "$_hooks_source_dir/hooks.sh" 2>/dev/null
    ) || true

    # Now source it for real with our stubs active
    # Override APP_ROOT to avoid var.sh sourcing issues
    _saved_app_root="$APP_ROOT"
    # Temporarily make var.sh and trash sourcing succeed
    var_TRASH_FILE="/dev/null"
    export var_TRASH_FILE
    # Source hooks.sh - it will try to source var.sh which may fail, that's ok
    source "$_hooks_source_dir/hooks.sh" 2>/dev/null || {
        # If sourcing failed due to var.sh, just define the functions directly
        # by sourcing with stubs
        unset _CLAUDE_CODE_HOOKS_SOURCED
        eval "$(sed -n '/^claude_code::hooks_add/,/^}/p' "$_hooks_source_dir/hooks.sh")"
        eval "$(sed -n '/^claude_code::hooks_remove/,/^}/p' "$_hooks_source_dir/hooks.sh")"
    }
    APP_ROOT="$_saved_app_root"
}

# BATS teardown function - runs after each test
teardown() {
    if [[ -d "${TEST_TMP_DIR:-}" ]]; then
        rm -rf "$TEST_TMP_DIR"
    fi
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "hooks.sh defines required functions" {
    declare -f claude_code::hooks_add >/dev/null
    declare -f claude_code::hooks_remove >/dev/null
}

# ============================================================================
# hooks_add Tests
# ============================================================================

@test "hooks_add creates hook in empty/missing settings file (project scope)" {
    [ ! -f "$CLAUDE_PROJECT_SETTINGS" ]

    run claude_code::hooks_add "Stop" "test-hook" '{"type":"http","url":"http://localhost:3000/stop"}' "project"
    [ "$status" -eq 0 ]

    [ -f "$CLAUDE_PROJECT_SETTINGS" ]
    local result
    result=$(jq '.hooks.Stop[0].hooks[0]._id' "$CLAUDE_PROJECT_SETTINGS")
    [ "$result" = '"test-hook"' ]
    result=$(jq '.hooks.Stop[0].hooks[0].type' "$CLAUDE_PROJECT_SETTINGS")
    [ "$result" = '"http"' ]
    result=$(jq '.hooks.Stop[0].matcher' "$CLAUDE_PROJECT_SETTINGS")
    [ "$result" = '"*"' ]
}

@test "hooks_add creates hook in empty/missing settings file (global scope)" {
    [ ! -f "$CLAUDE_SETTINGS_FILE" ]

    run claude_code::hooks_add "Stop" "global-hook" '{"type":"http","url":"http://localhost:4000/stop"}' "global"
    [ "$status" -eq 0 ]

    [ -f "$CLAUDE_SETTINGS_FILE" ]
    local result
    result=$(jq '.hooks.Stop[0].hooks[0]._id' "$CLAUDE_SETTINGS_FILE")
    [ "$result" = '"global-hook"' ]
}

@test "hooks_add deduplicates - same identifier updates existing hook" {
    run claude_code::hooks_add "Stop" "my-hook" '{"type":"http","url":"http://old-url"}' "project"
    [ "$status" -eq 0 ]

    run claude_code::hooks_add "Stop" "my-hook" '{"type":"http","url":"http://new-url","timeout":30}' "project"
    [ "$status" -eq 0 ]

    local count
    count=$(jq '.hooks.Stop | length' "$CLAUDE_PROJECT_SETTINGS")
    [ "$count" -eq 1 ]

    local url
    url=$(jq -r '.hooks.Stop[0].hooks[0].url' "$CLAUDE_PROJECT_SETTINGS")
    [ "$url" = "http://new-url" ]

    local timeout
    timeout=$(jq '.hooks.Stop[0].hooks[0].timeout' "$CLAUDE_PROJECT_SETTINGS")
    [ "$timeout" = "30" ]
}

@test "hooks_add preserves unrelated hooks in same event" {
    run claude_code::hooks_add "Stop" "hook-a" '{"type":"http","url":"http://a"}' "project"
    [ "$status" -eq 0 ]
    run claude_code::hooks_add "Stop" "hook-b" '{"type":"http","url":"http://b"}' "project"
    [ "$status" -eq 0 ]

    local count
    count=$(jq '.hooks.Stop | length' "$CLAUDE_PROJECT_SETTINGS")
    [ "$count" -eq 2 ]

    local id_a id_b
    id_a=$(jq -r '.hooks.Stop[0].hooks[0]._id' "$CLAUDE_PROJECT_SETTINGS")
    id_b=$(jq -r '.hooks.Stop[1].hooks[0]._id' "$CLAUDE_PROJECT_SETTINGS")
    [ "$id_a" = "hook-a" ]
    [ "$id_b" = "hook-b" ]
}

@test "hooks_add preserves hooks in other events" {
    run claude_code::hooks_add "Stop" "stop-hook" '{"type":"http","url":"http://stop"}' "project"
    [ "$status" -eq 0 ]
    run claude_code::hooks_add "PreToolUse" "pretool-hook" '{"type":"http","url":"http://pretool"}' "project"
    [ "$status" -eq 0 ]

    local stop_count pretool_count
    stop_count=$(jq '.hooks.Stop | length' "$CLAUDE_PROJECT_SETTINGS")
    pretool_count=$(jq '.hooks.PreToolUse | length' "$CLAUDE_PROJECT_SETTINGS")
    [ "$stop_count" -eq 1 ]
    [ "$pretool_count" -eq 1 ]
}

@test "hooks_add preserves existing non-hook settings" {
    echo '{"permissions":{"allow":["Read"]},"env":{"FOO":"bar"}}' | jq '.' > "$CLAUDE_PROJECT_SETTINGS"

    run claude_code::hooks_add "Stop" "my-hook" '{"type":"http","url":"http://test"}' "project"
    [ "$status" -eq 0 ]

    local perm
    perm=$(jq -r '.permissions.allow[0]' "$CLAUDE_PROJECT_SETTINGS")
    [ "$perm" = "Read" ]
    local env_val
    env_val=$(jq -r '.env.FOO' "$CLAUDE_PROJECT_SETTINGS")
    [ "$env_val" = "bar" ]
    local hook_id
    hook_id=$(jq -r '.hooks.Stop[0].hooks[0]._id' "$CLAUDE_PROJECT_SETTINGS")
    [ "$hook_id" = "my-hook" ]
}

@test "hooks_add fails with missing arguments" {
    run claude_code::hooks_add "" "id" '{}' "project"
    [ "$status" -eq 1 ]
    run claude_code::hooks_add "Stop" "" '{}' "project"
    [ "$status" -eq 1 ]
    run claude_code::hooks_add "Stop" "id" "" "project"
    [ "$status" -eq 1 ]
}

@test "hooks_add fails with invalid scope" {
    run claude_code::hooks_add "Stop" "id" '{"type":"http"}' "invalid"
    [ "$status" -eq 1 ]
}

@test "hooks_add handles malformed settings file gracefully" {
    echo "not valid json" > "$CLAUDE_PROJECT_SETTINGS"

    run claude_code::hooks_add "Stop" "my-hook" '{"type":"http","url":"http://test"}' "project"
    [ "$status" -eq 0 ]

    local hook_id
    hook_id=$(jq -r '.hooks.Stop[0].hooks[0]._id' "$CLAUDE_PROJECT_SETTINGS")
    [ "$hook_id" = "my-hook" ]
}

# ============================================================================
# hooks_remove Tests
# ============================================================================

@test "hooks_remove removes existing hook" {
    run claude_code::hooks_add "Stop" "to-remove" '{"type":"http","url":"http://remove-me"}' "project"
    [ "$status" -eq 0 ]

    run claude_code::hooks_remove "Stop" "to-remove" "project"
    [ "$status" -eq 0 ]

    local has_hooks
    has_hooks=$(jq 'has("hooks")' "$CLAUDE_PROJECT_SETTINGS")
    [ "$has_hooks" = "false" ]
}

@test "hooks_remove preserves other hooks in same event" {
    run claude_code::hooks_add "Stop" "keep-me" '{"type":"http","url":"http://keep"}' "project"
    [ "$status" -eq 0 ]
    run claude_code::hooks_add "Stop" "remove-me" '{"type":"http","url":"http://remove"}' "project"
    [ "$status" -eq 0 ]

    run claude_code::hooks_remove "Stop" "remove-me" "project"
    [ "$status" -eq 0 ]

    local count
    count=$(jq '.hooks.Stop | length' "$CLAUDE_PROJECT_SETTINGS")
    [ "$count" -eq 1 ]
    local remaining_id
    remaining_id=$(jq -r '.hooks.Stop[0].hooks[0]._id' "$CLAUDE_PROJECT_SETTINGS")
    [ "$remaining_id" = "keep-me" ]
}

@test "hooks_remove preserves hooks in other events" {
    run claude_code::hooks_add "Stop" "stop-hook" '{"type":"http","url":"http://stop"}' "project"
    [ "$status" -eq 0 ]
    run claude_code::hooks_add "PreToolUse" "pretool-hook" '{"type":"http","url":"http://pretool"}' "project"
    [ "$status" -eq 0 ]

    run claude_code::hooks_remove "Stop" "stop-hook" "project"
    [ "$status" -eq 0 ]

    local has_stop
    has_stop=$(jq '.hooks | has("Stop")' "$CLAUDE_PROJECT_SETTINGS")
    [ "$has_stop" = "false" ]
    local pretool_count
    pretool_count=$(jq '.hooks.PreToolUse | length' "$CLAUDE_PROJECT_SETTINGS")
    [ "$pretool_count" -eq 1 ]
}

@test "hooks_remove is idempotent for non-existent event" {
    echo '{}' > "$CLAUDE_PROJECT_SETTINGS"
    run claude_code::hooks_remove "NonExistent" "no-such-hook" "project"
    [ "$status" -eq 0 ]
}

@test "hooks_remove is idempotent for non-existent identifier" {
    run claude_code::hooks_add "Stop" "existing" '{"type":"http","url":"http://test"}' "project"
    [ "$status" -eq 0 ]
    run claude_code::hooks_remove "Stop" "non-existent" "project"
    [ "$status" -eq 0 ]

    local count
    count=$(jq '.hooks.Stop | length' "$CLAUDE_PROJECT_SETTINGS")
    [ "$count" -eq 1 ]
}

@test "hooks_remove is idempotent when settings file missing" {
    [ ! -f "$CLAUDE_PROJECT_SETTINGS" ]
    run claude_code::hooks_remove "Stop" "no-such-hook" "project"
    [ "$status" -eq 0 ]
}

@test "hooks_remove preserves non-hook settings" {
    echo '{"permissions":{"allow":["Read"]},"hooks":{"Stop":[{"matcher":"*","hooks":[{"type":"http","_id":"rm-me"}]}]}}' | jq '.' > "$CLAUDE_PROJECT_SETTINGS"

    run claude_code::hooks_remove "Stop" "rm-me" "project"
    [ "$status" -eq 0 ]

    local perm
    perm=$(jq -r '.permissions.allow[0]' "$CLAUDE_PROJECT_SETTINGS")
    [ "$perm" = "Read" ]
    local has_hooks
    has_hooks=$(jq 'has("hooks")' "$CLAUDE_PROJECT_SETTINGS")
    [ "$has_hooks" = "false" ]
}

@test "hooks_remove works with global scope" {
    run claude_code::hooks_add "Stop" "global-rm" '{"type":"http","url":"http://global"}' "global"
    [ "$status" -eq 0 ]
    run claude_code::hooks_remove "Stop" "global-rm" "global"
    [ "$status" -eq 0 ]

    local has_hooks
    has_hooks=$(jq 'has("hooks")' "$CLAUDE_SETTINGS_FILE")
    [ "$has_hooks" = "false" ]
}

@test "hooks_remove fails with missing arguments" {
    run claude_code::hooks_remove "" "id" "project"
    [ "$status" -eq 1 ]
    run claude_code::hooks_remove "Stop" "" "project"
    [ "$status" -eq 1 ]
}

@test "hooks_remove cleans up empty event but keeps other events" {
    run claude_code::hooks_add "Stop" "only-hook" '{"type":"http","url":"http://test"}' "project"
    [ "$status" -eq 0 ]
    run claude_code::hooks_add "PreToolUse" "other" '{"type":"http","url":"http://other"}' "project"
    [ "$status" -eq 0 ]

    run claude_code::hooks_remove "Stop" "only-hook" "project"
    [ "$status" -eq 0 ]

    local has_stop
    has_stop=$(jq '.hooks | has("Stop")' "$CLAUDE_PROJECT_SETTINGS")
    [ "$has_stop" = "false" ]
    local has_hooks
    has_hooks=$(jq 'has("hooks")' "$CLAUDE_PROJECT_SETTINGS")
    [ "$has_hooks" = "true" ]
}
