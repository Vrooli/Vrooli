#!/usr/bin/env bats

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/windmill"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "windmill"
    
    # Export paths for use in setup()
    export SETUP_FILE_INJECT_SCRIPT="${BATS_TEST_DIRNAME}/inject.sh"
    export SETUP_FILE_WINDMILL_DIR="${BATS_TEST_DIRNAME}"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    INJECT_SCRIPT="${SETUP_FILE_INJECT_SCRIPT}"
    WINDMILL_DIR="${SETUP_FILE_WINDMILL_DIR}"
    
    # Set test environment
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_API_BASE="http://localhost:5681/api"
    export DEFAULT_WORKSPACE="f/vrooli"
    export VROOLI_PROJECT_ROOT="/tmp/test-project"
    
    # Create test project directory structure
    mkdir -p "$VROOLI_PROJECT_ROOT/test-scripts"
    mkdir -p "$VROOLI_PROJECT_ROOT/test-apps"
    mkdir -p "$VROOLI_PROJECT_ROOT/test-schemas"
    
    # Create test files
    echo 'console.log("Hello from TypeScript");' > "$VROOLI_PROJECT_ROOT/test-scripts/hello.ts"
    echo '{"type": "object", "properties": {"name": {"type": "string"}}}' > "$VROOLI_PROJECT_ROOT/test-schemas/hello.json"
    echo '{"value": {"grid": []}, "summary": "Test app"}' > "$VROOLI_PROJECT_ROOT/test-apps/test-app.json"
    
    # Mock system commands
    system::is_command() {
        case "$1" in
            "curl") return 0 ;;
            "jq") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    # Mock curl command to simulate Windmill API responses
    curl() {
        local args=("$@")
        local url=""
        local method="GET"
        local is_silent=false
        
        # Parse curl arguments
        for ((i=0; i<${#args[@]}; i++)); do
            case "${args[i]}" in
                "-s"|"--silent") is_silent=true ;;
                "-X"|"--request") 
                    i=$((i+1))
                    method="${args[i]}" ;;
                "http"*)
                    url="${args[i]}" ;;
            esac
        done
        
        # Mock different API endpoints
        if [[ "$url" =~ /api/version$ ]]; then
            echo '"v1.290.0"'
            return 0
        elif [[ "$url" =~ /api/w/.*/scripts/create$ ]] && [[ "$method" == "POST" ]]; then
            echo '{"path":"test-script","created":true}'
            return 0
        elif [[ "$url" =~ /api/w/.*/apps/create$ ]] && [[ "$method" == "POST" ]]; then
            echo '{"path":"test-app","created":true}'
            return 0
        elif [[ "$url" =~ /api/w/.*/scripts/get/ ]]; then
            echo '{"path":"test-script","exists":true}'
            return 0
        elif [[ "$url" =~ /api/w/.*/apps/get/ ]]; then
            echo '{"path":"test-app","exists":true}'
            return 0
        else
            # Default to connection refused for unhandled endpoints
            if [[ "$is_silent" != "true" ]]; then
                echo "curl: (7) Failed to connect to localhost" >&2
            fi
            return 7
        fi
    }
    
    # Mock jq command
    jq() {
        if [[ "$1" == "." ]]; then
            # Just validate JSON - use real jq if available, otherwise simple check
            if command -v jq >/dev/null 2>&1; then
                command jq "$@"
            else
                # Simple JSON validation - check if it starts with { or [
                local input
                input=$(cat)
                if [[ "$input" =~ ^[[:space:]]*[\{\[] ]]; then
                    echo "$input"
                    return 0
                else
                    echo "parse error: Invalid JSON" >&2
                    return 1
                fi
            fi
        else
            # Mock specific jq operations used in the script
            case "$*" in
                *"type"*)
                    echo "array"
                    ;;
                *"length"*)
                    echo "1"
                    ;;
                *".path"*)
                    echo "f/vrooli/test-script"
                    ;;
                *".file"*)
                    echo "test-scripts/hello.ts"
                    ;;
                *".language"*)
                    echo "typescript"
                    ;;
                *".summary"*)
                    echo "Test script"
                    ;;
                "-c"|"--compact-output")
                    echo '{"path":"f/vrooli/test-script","file":"test-scripts/hello.ts"}'
                    ;;
                *)
                    # Default response for unknown jq operations
                    echo "mock-result"
                    ;;
            esac
        fi
    }
    
    # Mock log functions
    log::info() { echo "INFO: $*"; }
    log::error() { echo "ERROR: $*"; }
    log::success() { echo "SUCCESS: $*"; }
    log::warn() { echo "WARN: $*"; }
    log::debug() { echo "DEBUG: $*"; }
    log::header() { echo "HEADER: $*"; }
    
    # Source the inject script functions without executing main
    source "${INJECT_SCRIPT}"
}

# Cleanup after each test
teardown() {
    # Clean up test files
    trash::safe_remove "$VROOLI_PROJECT_ROOT" --test-cleanup
    vrooli_cleanup_test
}

# Test windmill accessibility check
@test "windmill_inject::check_accessibility returns success when Windmill is running" {
    # Mock successful API response
    curl() {
        if [[ "$*" =~ /api/version ]]; then
            echo '"v1.290.0"'
            return 0
        fi
        return 1
    }
    
    run windmill_inject::check_accessibility
    [ "$status" -eq 0 ]
}

@test "windmill_inject::check_accessibility returns failure when Windmill is not running" {
    # Mock failed API response
    curl() {
        echo "curl: (7) Failed to connect to localhost" >&2
        return 7
    }
    
    run windmill_inject::check_accessibility
    [ "$status" -eq 1 ]
}

# Test language detection
@test "windmill_inject::detect_language detects TypeScript files" {
    result=$(windmill_inject::detect_language "script.ts")
    [ "$result" = "typescript" ]
}

@test "windmill_inject::detect_language detects JavaScript files" {
    result=$(windmill_inject::detect_language "script.js")
    [ "$result" = "javascript" ]
}

@test "windmill_inject::detect_language detects Python files" {
    result=$(windmill_inject::detect_language "script.py")
    [ "$result" = "python3" ]
}

@test "windmill_inject::detect_language defaults to typescript for unknown extensions" {
    result=$(windmill_inject::detect_language "script.unknown")
    [ "$result" = "typescript" ]
}

# Test script validation
@test "windmill_inject::validate_scripts validates correct script configuration" {
    local scripts='[{"path":"f/test","file":"test-scripts/hello.ts","language":"typescript"}]'
    
    run windmill_inject::validate_scripts "$scripts"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "valid" ]]
}

@test "windmill_inject::validate_scripts fails with missing path field" {
    local scripts='[{"file":"test-scripts/hello.ts","language":"typescript"}]'
    
    run windmill_inject::validate_scripts "$scripts"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "missing required 'path' field" ]]
}

@test "windmill_inject::validate_scripts fails with missing file field" {
    local scripts='[{"path":"f/test","language":"typescript"}]'
    
    run windmill_inject::validate_scripts "$scripts"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "missing required 'file' field" ]]
}

@test "windmill_inject::validate_scripts fails with non-existent file" {
    local scripts='[{"path":"f/test","file":"non-existent.ts","language":"typescript"}]'
    
    run windmill_inject::validate_scripts "$scripts"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "file not found" ]]
}

@test "windmill_inject::validate_scripts fails with non-array configuration" {
    local scripts='{"path":"f/test","file":"test-scripts/hello.ts"}'
    
    run windmill_inject::validate_scripts "$scripts"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "must be an array" ]]
}

# Test app validation
@test "windmill_inject::validate_apps validates correct app configuration" {
    local apps='[{"path":"f/test","file":"test-apps/test-app.json","summary":"Test app"}]'
    
    run windmill_inject::validate_apps "$apps"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "valid" ]]
}

@test "windmill_inject::validate_apps fails with missing path field" {
    local apps='[{"file":"test-apps/test-app.json","summary":"Test app"}]'
    
    run windmill_inject::validate_apps "$apps"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "missing required 'path' field" ]]
}

@test "windmill_inject::validate_apps fails with invalid JSON file" {
    # Create invalid JSON file
    echo 'invalid json content' > "$VROOLI_PROJECT_ROOT/test-apps/invalid.json"
    local apps='[{"path":"f/test","file":"test-apps/invalid.json","summary":"Test app"}]'
    
    # Override jq to simulate JSON validation failure
    jq() {
        if [[ "$1" == "." ]]; then
            echo "parse error: Invalid JSON" >&2
            return 1
        fi
        # Fall back to original jq mock for other operations
        case "$*" in
            *"type"*) echo "array" ;;
            *"length"*) echo "1" ;;
            *".path"*) echo "f/test" ;;
            *".file"*) echo "test-apps/invalid.json" ;;
            *) echo "mock-result" ;;
        esac
    }
    
    run windmill_inject::validate_apps "$apps"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "invalid JSON" ]]
}

# Test overall configuration validation
@test "windmill_inject::validate_config validates complete configuration" {
    local config='{"scripts":[{"path":"f/test","file":"test-scripts/hello.ts"}],"apps":[{"path":"f/app","file":"test-apps/test-app.json"}]}'
    
    run windmill_inject::validate_config "$config"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "valid" ]]
}

@test "windmill_inject::validate_config fails with invalid JSON" {
    local config='invalid json'
    
    run windmill_inject::validate_config "$config"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid JSON" ]]
}

@test "windmill_inject::validate_config fails with no injection types" {
    local config='{}'
    
    run windmill_inject::validate_config "$config"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "must have" ]]
}

# Test script import
@test "windmill_inject::import_script successfully imports script" {
    local script_config='{"path":"f/test","file":"test-scripts/hello.ts","summary":"Test script","language":"typescript"}'
    
    run windmill_inject::import_script "$script_config"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Imported script: f/test" ]]
}

# Test app import
@test "windmill_inject::import_app successfully imports app" {
    local app_config='{"path":"f/test-app","file":"test-apps/test-app.json","summary":"Test app"}'
    
    run windmill_inject::import_app "$app_config"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Imported app: f/test-app" ]]
}

# Test rollback functionality
@test "windmill_inject::add_rollback_action adds rollback action" {
    WINDMILL_ROLLBACK_ACTIONS=()
    
    windmill_inject::add_rollback_action "Test action" "echo 'rollback command'"
    
    [ ${#WINDMILL_ROLLBACK_ACTIONS[@]} -eq 1 ]
    [[ "${WINDMILL_ROLLBACK_ACTIONS[0]}" =~ "Test action" ]]
}

@test "windmill_inject::execute_rollback executes rollback actions in reverse order" {
    WINDMILL_ROLLBACK_ACTIONS=(
        "Action 1|echo 'first'"
        "Action 2|echo 'second'"
    )
    
    run windmill_inject::execute_rollback
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Action 2" ]]
    [[ "$output" =~ "Action 1" ]]
}

# Test status checking
@test "windmill_inject::check_status checks script and app status" {
    local config='{"scripts":[{"path":"f/test","file":"test-scripts/hello.ts"}],"apps":[{"path":"f/app","file":"test-apps/test-app.json"}]}'
    
    run windmill_inject::check_status "$config"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "found" ]]
}

# Test main function argument handling
@test "windmill_inject::main handles --validate action" {
    local config='{"scripts":[{"path":"f/test","file":"test-scripts/hello.ts"}]}'
    
    run windmill_inject::main "--validate" "$config"
    [ "$status" -eq 0 ]
}

@test "windmill_inject::main handles --status action" {
    local config='{"scripts":[{"path":"f/test","file":"test-scripts/hello.ts"}]}'
    
    run windmill_inject::main "--status" "$config"
    [ "$status" -eq 0 ]
}

@test "windmill_inject::main handles --help action" {
    run windmill_inject::main "--help"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USAGE:" ]]
}

@test "windmill_inject::main fails with unknown action" {
    run windmill_inject::main "--unknown" "config"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown action" ]]
}

@test "windmill_inject::main fails without configuration" {
    run windmill_inject::main "--validate"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Configuration JSON required" ]]
}

# Test injection script has correct permissions
@test "windmill inject script has correct permissions" {
    [ -x "${WINDMILL_DIR}/inject.sh" ]
}

# Test script structure and syntax
@test "windmill inject script has valid bash syntax" {
    run bash -n "${WINDMILL_DIR}/inject.sh"
    [ "$status" -eq 0 ]
}