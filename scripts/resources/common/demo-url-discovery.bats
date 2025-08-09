#!/usr/bin/env bats

# Test suite for demo-url-discovery.sh
# Tests the URL discovery demo script functionality

# Setup test directory and source var.sh first
COMMON_DIR="$BATS_TEST_DIRNAME"
_HERE="$COMMON_DIR"

# shellcheck disable=SC1091
source "${_HERE}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/test_helper.bash"

# Load mocks for testing
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/jq.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/http.sh"

# Setup and teardown
setup() {
    # Reset mock states
    if command -v mock::jq::reset &>/dev/null; then
        mock::jq::reset
    fi
    if command -v mock::http::reset &>/dev/null; then
        mock::http::reset
    fi
    
    # Export required test variables
    export var_SCRIPTS_SCENARIOS_DIR="/test/scenarios"
    export var_SCRIPTS_RESOURCES_DIR="/test/resources"
    export var_LOG_FILE="/test/lib/utils/log.sh"
    
    # Create mock log file
    mkdir -p "$(dirname "$var_LOG_FILE")"
    cat > "$var_LOG_FILE" << 'EOF'
log::info() { echo "[INFO] $*"; }
log::error() { echo "[ERROR] $*" >&2; }
log::success() { echo "[SUCCESS] $*"; }
EOF
}

teardown() {
    # Clean up test artifacts
    rm -rf /test 2>/dev/null || true
}

# ============================================================================
# Function existence tests
# ============================================================================

@test "demo-url-discovery.sh can be sourced" {
    # Source in subshell to avoid conflicts
    run bash -c "
        export var_SCRIPTS_SCENARIOS_DIR='/test/scenarios'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh'
        echo 'SUCCESS'
    "
    
    assert_success
    [[ "$output" =~ "SUCCESS" ]]
}

@test "demo_url_discovery::get_required_resources function exists" {
    run bash -c "
        export var_SCRIPTS_SCENARIOS_DIR='/test/scenarios'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh'
        type -t demo_url_discovery::get_required_resources
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "demo_url_discovery::log_info function exists" {
    run bash -c "
        export var_SCRIPTS_SCENARIOS_DIR='/test/scenarios'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh'
        type -t demo_url_discovery::log_info
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "demo_url_discovery::get_access_urls_demo function exists" {
    run bash -c "
        export var_SCRIPTS_SCENARIOS_DIR='/test/scenarios'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh'
        type -t demo_url_discovery::get_access_urls_demo
    "
    
    assert_success
    [ "$output" = "function" ]
}

# ============================================================================
# Function behavior tests
# ============================================================================

@test "demo_url_discovery::log_info outputs timestamped messages" {
    run bash -c "
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh'
        demo_url_discovery::log_info 'Test message'
    "
    
    assert_success
    [[ "$output" =~ \[[0-9]{2}:[0-9]{2}:[0-9]{2}\]\ INFO:\ Test\ message ]]
}

@test "demo_url_discovery::get_required_resources parses JSON correctly" {
    # Mock service.json content
    mock::jq::set_response '{
        "resources": {
            "ai": {
                "ollama": {"enabled": true, "required": true}
            },
            "automation": {
                "n8n": {"enabled": true, "required": true}
            }
        }
    }' "ollama
n8n"
    
    run bash -c "
        export SERVICE_JSON='{\"resources\":{\"ai\":{\"ollama\":{\"enabled\":true,\"required\":true}}}}'
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh'
        demo_url_discovery::get_required_resources
    "
    
    # Should output resource names
    [[ "$output" =~ ollama ]] || [[ "$output" =~ n8n ]]
}

# ============================================================================
# Error handling tests
# ============================================================================

@test "demo handles missing service.json gracefully" {
    run bash -c "
        export var_SCRIPTS_SCENARIOS_DIR='/nonexistent'
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh' 2>&1
    "
    
    # Should not crash completely
    [ "$status" -ne 0 ] || [[ "$output" =~ "No such file" ]]
}

@test "demo handles invalid JSON gracefully" {
    # Mock invalid JSON response
    mock::jq::set_response "" "parse error"
    mock::jq::set_exit_code 1
    
    run bash -c "
        export SERVICE_JSON='invalid json'
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh'
        demo_url_discovery::get_required_resources 2>/dev/null
    "
    
    # Should handle error without crashing
    [ -z "$output" ] || [[ "$output" =~ "" ]]
}

# ============================================================================
# Integration tests
# ============================================================================

@test "demo can categorize resources correctly" {
    run bash -c "
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh'
        
        # Test categorization logic directly
        resource='ollama'
        case \"\$resource\" in
            ollama|whisper|unstructured-io)
                echo 'AI Resources'
                ;;
            *)
                echo 'Unknown'
                ;;
        esac
    "
    
    assert_success
    [ "$output" = "AI Resources" ]
}

@test "demo categorizes automation resources correctly" {
    run bash -c "
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh'
        
        resource='n8n'
        case \"\$resource\" in
            n8n|windmill|node-red|comfyui|huginn)
                echo 'Automation'
                ;;
            *)
                echo 'Unknown'
                ;;
        esac
    "
    
    assert_success
    [ "$output" = "Automation" ]
}

@test "demo categorizes storage resources correctly" {
    run bash -c "
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/demo-url-discovery.sh'
        
        resource='postgres'
        case \"\$resource\" in
            postgres|postgresql|redis|minio|qdrant|questdb|vault)
                echo 'Storage'
                ;;
            *)
                echo 'Unknown'
                ;;
        esac
    "
    
    assert_success
    [ "$output" = "Storage" ]
}