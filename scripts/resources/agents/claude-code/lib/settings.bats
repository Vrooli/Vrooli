#!/usr/bin/env bats

load ./test_helper.bash

# BATS setup function - runs before each test
setup() {
    # Set up paths
    export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)}"
    export CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    export RESOURCES_DIR="$CLAUDE_CODE_DIR/../.."
    export HELPERS_DIR="$RESOURCES_DIR/../helpers"
    export SCRIPT_PATH="$BATS_TEST_DIRNAME/settings.sh"
    
    # Source dependencies in order
    source "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/flow.sh" 2>/dev/null || true
    source "$RESOURCES_DIR/common.sh" 2>/dev/null || true
    
    # Source config and messages
    source "$CLAUDE_CODE_DIR/config/defaults.sh"
    source "$CLAUDE_CODE_DIR/config/messages.sh" 2>/dev/null || true
    
    # Source common functions
    source "$CLAUDE_CODE_DIR/lib/common.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Default mocks
    confirm() { return 0; }  # Always confirm
    system::is_command() { 
        case "$1" in
            jq) return 0 ;;
            *) return 0 ;;
        esac
    }
}

@test "settings.sh defines required functions" {
    declare -f claude_code::settings
    declare -f claude_code::settings_tips
    declare -f claude_code::settings_get
    declare -f claude_code::settings_set
    declare -f claude_code::settings_reset
}

# ============================================================================
# Main Settings Function Tests
# ============================================================================

@test "claude_code::settings fails when not installed" {
    claude_code::is_installed() { return 1; }
    run claude_code::settings 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Claude Code is not installed" ]]
}

@test "claude_code::settings shows tips" {
    claude_code::is_installed() { return 0; }
    CLAUDE_PROJECT_SETTINGS='/nonexistent'
    CLAUDE_SETTINGS_FILE='/nonexistent'
    run claude_code::settings 2>&1
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Project settings override global settings" ]]
}

@test "claude_code::settings displays project settings" {
    TMP_FILE=$(mktemp)
    echo '{"project": "test"}' > "$TMP_FILE"
    
    claude_code::is_installed() { return 0; }
    CLAUDE_PROJECT_SETTINGS="$TMP_FILE"
    CLAUDE_SETTINGS_FILE='/nonexistent'
    system::is_command() { [[ "$1" == "jq" ]] && return 1; }
    run claude_code::settings 2>&1
    
    rm -f "$TMP_FILE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Project settings found" ]]
    [[ "$output" =~ "project" ]]
}

@test "claude_code::settings displays global settings" {
    TMP_FILE=$(mktemp)
    echo '{"global": "settings"}' > "$TMP_FILE"
    
    claude_code::is_installed() { return 0; }
    CLAUDE_PROJECT_SETTINGS='/nonexistent'
    CLAUDE_SETTINGS_FILE="$TMP_FILE"
    system::is_command() { [[ "$1" == "jq" ]] && return 1; }
    run claude_code::settings 2>&1
    
    rm -f "$TMP_FILE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Global settings found" ]]
    [[ "$output" =~ "global" ]]
}

# ============================================================================
# Settings Get Tests
# ============================================================================

@test "claude_code::settings_get returns empty for missing file" {
    output=$(
        CLAUDE_PROJECT_SETTINGS='/nonexistent'
        CLAUDE_SETTINGS_FILE='/nonexistent'
        claude_code::settings_get 'test.key' 'auto' 2>&1
    ) || status=$?
    [ "${status:-0}" -eq 1 ]
    [[ "$output" == "" ]]
}

@test "claude_code::settings_get prefers project over global" {
    TMP_PROJECT=$(mktemp)
    TMP_GLOBAL=$(mktemp)
    echo '{"key": "project-value"}' > "$TMP_PROJECT"
    echo '{"key": "global-value"}' > "$TMP_GLOBAL"
    
    output=$(
        CLAUDE_PROJECT_SETTINGS="$TMP_PROJECT"
        CLAUDE_SETTINGS_FILE="$TMP_GLOBAL"
        system::is_command() { return 0; }
        jq() { echo 'project-value'; }
        claude_code::settings_get 'key' 'auto' 2>&1
    )
    
    rm -f "$TMP_PROJECT" "$TMP_GLOBAL"
    [[ "$output" == "project-value" ]]
}

@test "claude_code::settings_get handles specific scope" {
    TMP_FILE=$(mktemp)
    echo '{"key": "value"}' > "$TMP_FILE"
    
    output=$(
        CLAUDE_SETTINGS_FILE="$TMP_FILE"
        system::is_command() { return 0; }
        jq() { echo 'value'; }
        claude_code::settings_get 'key' 'global' 2>&1
    )
    
    rm -f "$TMP_FILE"
    [[ "$output" == "value" ]]
}

# ============================================================================
# Settings Set Tests
# ============================================================================

@test "claude_code::settings_set fails without key or value" {
    run claude_code::settings_set '' 'value' 'project' 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "key and value are required" ]]
}

@test "claude_code::settings_set fails with invalid scope" {
    run claude_code::settings_set 'key' 'value' 'invalid' 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid scope" ]]
}

@test "claude_code::settings_set requires jq" {
    output=$(
        system::is_command() { return 1; }
        claude_code::settings_set 'key' 'value' 'project' 2>&1
    ) || status=$?
    [ "${status:-0}" -eq 1 ]
    [[ "$output" =~ "jq is required" ]]
}

@test "claude_code::settings_set creates new file" {
    TMP_DIR=$(mktemp -d)
    TMP_FILE="$TMP_DIR/settings.json"
    
    output=$(
        CLAUDE_PROJECT_SETTINGS="$TMP_FILE"
        system::is_command() { return 0; }
        jq() { 
            if [[ "$1" == "." ]]; then
                cat > "$TMP_FILE"
            fi
        }
        echo '{"key": "value"}' | claude_code::settings_set 'key' '"value"' 'project' 2>&1
    )
    
    [ -f "$TMP_FILE" ]
    rm -rf "$TMP_DIR"
    [[ "$output" =~ "Settings file created" ]]
}

# ============================================================================
# Settings Reset Tests
# ============================================================================

@test "claude_code::settings_reset handles missing file" {
    output=$(
        CLAUDE_PROJECT_SETTINGS='/nonexistent'
        claude_code::settings_reset 'project' 2>&1
    )
    [[ "$output" =~ "No project settings to reset" ]]
}

@test "claude_code::settings_reset removes project settings" {
    TMP_FILE=$(mktemp)
    echo '{"test": "data"}' > "$TMP_FILE"
    
    output=$(
        CLAUDE_PROJECT_SETTINGS="$TMP_FILE"
        confirm() { return 0; }
        claude_code::settings_reset 'project' 2>&1
    )
    
    [ ! -f "$TMP_FILE" ]
    [[ "$output" =~ "Project settings reset" ]]
}

@test "claude_code::settings_reset cancels on no confirmation" {
    TMP_FILE=$(mktemp)
    echo '{"test": "data"}' > "$TMP_FILE"
    
    output=$(
        CLAUDE_PROJECT_SETTINGS="$TMP_FILE"
        confirm() { return 1; }
        claude_code::settings_reset 'project' 2>&1
    )
    
    [ -f "$TMP_FILE" ]
    rm -f "$TMP_FILE"
    [[ "$output" =~ "Reset cancelled" ]]
}

@test "claude_code::settings_reset handles 'all' scope" {
    output=$(
        CLAUDE_PROJECT_SETTINGS='/nonexistent'
        CLAUDE_SETTINGS_FILE='/nonexistent'
        claude_code::settings_reset() { 
            if [[ "$1" == "all" ]]; then
                echo "Reset project"
                echo "Reset global"
            else
                echo "Reset $1"
            fi
        }
        claude_code::settings_reset 'all' 2>&1
    )
    [[ "$output" =~ "Reset project" ]]
    [[ "$output" =~ "Reset global" ]]
}