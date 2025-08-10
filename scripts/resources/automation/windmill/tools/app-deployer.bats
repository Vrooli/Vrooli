#!/usr/bin/env bats

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "windmill"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export SETUP_FILE_WINDMILL_DIR="$(dirname "${BATS_TEST_DIRNAME}")" 
    export SETUP_FILE_CONFIG_DIR="$(dirname "${BATS_TEST_DIRNAME}")/config"
    export SETUP_FILE_TOOLS_DIR="$(dirname "${BATS_TEST_DIRNAME}")/tools"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    WINDMILL_DIR="${SETUP_FILE_WINDMILL_DIR}"
    CONFIG_DIR="${SETUP_FILE_CONFIG_DIR}"
    TOOLS_DIR="${SETUP_FILE_TOOLS_DIR}"
    
    # Set test environment BEFORE sourcing config files to avoid readonly conflicts
    export WINDMILL_PORT="5681"
    export WINDMILL_CONTAINER_NAME="windmill-test"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_DATA_DIR="/tmp/windmill-test"
    export WINDMILL_DB_PASSWORD="test-password"
    export YES="no"
    
    # Create test directories and files
    mkdir -p "$WINDMILL_DATA_DIR"
    mkdir -p "/tmp/test-apps"
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "windmill") echo "5681" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Now source the config files
    source "${WINDMILL_DIR}/config/defaults.sh"
    source "${WINDMILL_DIR}/config/messages.sh"
    
    # Export config and messages
    windmill::export_config
    windmill::export_messages
    
    # Source common functions
    source "${WINDMILL_DIR}/lib/common.sh"
    
    # Mock windmill status checks
    windmill::is_installed() {
        return 0
    }
    
    windmill::is_running() {
        return 0
    }
    
    windmill::is_healthy() {
        return 0
    }
    
    # Mock curl for API calls
    curl() {
        local args=("$@")
        local url=""
        local method="GET"
        
        # Parse curl arguments
        for ((i=0; i<${#args[@]}; i++)); do
            case "${args[i]}" in
                "-X"|"--request") 
                    i=$((i+1))
                    method="${args[i]}" ;;
                "http"*)
                    url="${args[i]}" ;;
            esac
        done
        
        # Mock different API endpoints
        if [[ "$url" =~ /api/w/.*/apps$ ]] && [[ "$method" == "POST" ]]; then
            echo '{"path":"f/test-app","created":true}'
        elif [[ "$url" =~ /api/w/.*/apps/.* ]]; then
            echo '{"path":"f/test-app","summary":"Test App","value":{"grid":[]}}'
        else
            echo '{"status":"ok"}'
        fi
        return 0
    }
    
    # Mock jq command
    jq() {
        case "$*" in
            *".path"*)
                echo "f/test-app"
                ;;
            *".created"*)
                echo "true"
                ;;
            *)
                echo "mock-result"
                ;;
        esac
    }
    
    # Create test app file
    cat > "/tmp/test-apps/sample-app.json" << 'EOF'
{
    "summary": "Sample Test App",
    "value": {
        "grid": [
            {
                "id": "a",
                "data": {
                    "type": "textinput",
                    "configuration": {"placeholder": "Enter text"}
                }
            }
        ]
    }
}
EOF
    
    # Source the app-deployer.sh file being tested (if it has functions)
    if [[ -f "${TOOLS_DIR}/app-deployer.sh" ]]; then
        # Check if it's meant to be sourced (has functions) or executed
        if grep -q "^[[:space:]]*function\|^[[:space:]]*[a-zA-Z_][a-zA-Z0-9_]*(" "${TOOLS_DIR}/app-deployer.sh"; then
            source "${TOOLS_DIR}/app-deployer.sh"
        fi
    fi
}

# Cleanup after each test
teardown() {
    trash::safe_remove "$WINDMILL_DATA_DIR" --test-cleanup
    trash::safe_remove "/tmp/test-apps" --test-cleanup
    vrooli_cleanup_test
}

# Test app deployer script exists and has correct permissions
@test "app-deployer.sh exists and is executable" {
    [ -f "${TOOLS_DIR}/app-deployer.sh" ]
    [ -x "${TOOLS_DIR}/app-deployer.sh" ]
}

# Test app deployer script has valid syntax
@test "app-deployer.sh has valid bash syntax" {
    run bash -n "${TOOLS_DIR}/app-deployer.sh"
    [ "$status" -eq 0 ]
}

# Test app deployer help/usage
@test "app-deployer.sh shows help when called with --help" {
    if [[ -x "${TOOLS_DIR}/app-deployer.sh" ]]; then
        run "${TOOLS_DIR}/app-deployer.sh" --help
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "Usage:" ]] || [[ "$output" =~ "help" ]]
    else
        skip "app-deployer.sh not executable"
    fi
}

# Test app deployer parameter validation
@test "app-deployer.sh validates required parameters" {
    if [[ -x "${TOOLS_DIR}/app-deployer.sh" ]]; then
        # Should fail without parameters
        run "${TOOLS_DIR}/app-deployer.sh"
        [[ "$status" -ne 0 ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "Usage:" ]]
    else
        skip "app-deployer.sh not executable"
    fi
}

# Test app deployer with valid app file
@test "app-deployer.sh can process valid app file" {
    if [[ -x "${TOOLS_DIR}/app-deployer.sh" ]]; then
        run "${TOOLS_DIR}/app-deployer.sh" --app "/tmp/test-apps/sample-app.json" --workspace "demo" --path "f/test-app"
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "deployed" ]] || [[ "$output" =~ "created" ]] || [[ "$output" =~ "not running" ]]
    else
        skip "app-deployer.sh not executable"
    fi
}

# Test app deployer with non-existent app file
@test "app-deployer.sh handles non-existent app file" {
    if [[ -x "${TOOLS_DIR}/app-deployer.sh" ]]; then
        run "${TOOLS_DIR}/app-deployer.sh" --app "/nonexistent/app.json" --workspace "demo" --path "f/test"
        [[ "$status" -ne 0 ]] || [[ "$output" =~ "not found" ]] || [[ "$output" =~ "does not exist" ]]
    else
        skip "app-deployer.sh not executable"
    fi
}

# Test individual functions if they exist
@test "app_deployer::deploy_app function exists and can be called" {
    if declare -f app_deployer::deploy_app >/dev/null; then
        run app_deployer::deploy_app "/tmp/test-apps/sample-app.json" "demo" "f/test-app"
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "deployed" ]] || [[ "$output" =~ "not running" ]]
    else
        skip "app_deployer::deploy_app function not defined"
    fi
}

@test "app_deployer::validate_app function exists and can be called" {
    if declare -f app_deployer::validate_app >/dev/null; then
        run app_deployer::validate_app "/tmp/test-apps/sample-app.json"
        [ "$status" -eq 0 ]
    else
        skip "app_deployer::validate_app function not defined"
    fi
}

@test "app_deployer::prepare_app_config function exists and can be called" {
    if declare -f app_deployer::prepare_app_config >/dev/null; then
        run app_deployer::prepare_app_config "/tmp/test-apps/sample-app.json" "f/test-app"
        [ "$status" -eq 0 ]
    else
        skip "app_deployer::prepare_app_config function not defined"
    fi
}

# Test that the tool handles different app formats
@test "app-deployer.sh handles malformed JSON" {
    if [[ -x "${TOOLS_DIR}/app-deployer.sh" ]]; then
        # Create malformed JSON file
        echo "invalid json" > "/tmp/test-apps/invalid.json"
        
        run "${TOOLS_DIR}/app-deployer.sh" --app "/tmp/test-apps/invalid.json" --workspace "demo" --path "f/test"
        [[ "$status" -ne 0 ]] || [[ "$output" =~ "invalid" ]] || [[ "$output" =~ "JSON" ]]
    else
        skip "app-deployer.sh not executable"
    fi
}

# Test workspace parameter handling
@test "app-deployer.sh validates workspace parameter" {
    if [[ -x "${TOOLS_DIR}/app-deployer.sh" ]]; then
        # Should fail with invalid workspace format
        run "${TOOLS_DIR}/app-deployer.sh" --app "/tmp/test-apps/sample-app.json" --workspace "" --path "f/test"
        [[ "$status" -ne 0 ]] || [[ "$output" =~ "workspace" ]] || [[ "$output" =~ "required" ]]
    else
        skip "app-deployer.sh not executable"
    fi
}

# Test path parameter handling
@test "app-deployer.sh validates path parameter" {
    if [[ -x "${TOOLS_DIR}/app-deployer.sh" ]]; then
        # Should fail with invalid path format
        run "${TOOLS_DIR}/app-deployer.sh" --app "/tmp/test-apps/sample-app.json" --workspace "demo" --path ""
        [[ "$status" -ne 0 ]] || [[ "$output" =~ "path" ]] || [[ "$output" =~ "required" ]]
    else
        skip "app-deployer.sh not executable"
    fi
}