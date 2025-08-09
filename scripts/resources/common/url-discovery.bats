#!/usr/bin/env bats

# Test suite for url-discovery.sh
# Tests the resource URL discovery infrastructure

# Setup test directory and source var.sh first
COMMON_DIR="$BATS_TEST_DIRNAME"
_HERE="$COMMON_DIR"

# shellcheck disable=SC1091
source "${_HERE}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/test_helper.bash"

# Load mocks for testing
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/docker.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/jq.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/http.sh"

# Setup and teardown
setup() {
    # Reset mock states
    if command -v mock::docker::reset &>/dev/null; then
        mock::docker::reset
    fi
    if command -v mock::jq::reset &>/dev/null; then
        mock::jq::reset
    fi
    if command -v mock::http::reset &>/dev/null; then
        mock::http::reset
    fi
    
    # Clean up cache directory
    rm -rf /tmp/vrooli-url-discovery 2>/dev/null || true
    
    # Export required variables
    export var_SCRIPTS_RESOURCES_DIR="/test/resources"
    export var_LOG_FILE="/test/lib/utils/log.sh"
    export VROOLI_COMMON_SOURCED=""
    
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
    rm -rf /tmp/vrooli-url-discovery 2>/dev/null || true
    rm -rf /test 2>/dev/null || true
}

# ============================================================================
# Loading and basic tests
# ============================================================================

@test "url-discovery.sh prevents multiple sourcing" {
    run bash -c "
        export VROOLI_URL_DISCOVERY_SOURCED=1
        source '$COMMON_DIR/url-discovery.sh'
        echo 'AFTER_SOURCE'
    "
    
    assert_success
    [[ "$output" =~ "AFTER_SOURCE" ]]
}

@test "url-discovery.sh loads successfully" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        
        # Create minimal common.sh mock
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        echo 'SUCCESS'
    "
    
    assert_success
    [[ "$output" =~ "SUCCESS" ]]
}

@test "cache directory constants are defined" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        echo \"CACHE_DIR: \$URL_DISCOVERY_CACHE_DIR\"
        echo \"CACHE_TTL: \$URL_DISCOVERY_CACHE_TTL\"
    "
    
    assert_success
    [[ "$output" =~ "CACHE_DIR: /tmp/vrooli-url-discovery" ]]
    [[ "$output" =~ "CACHE_TTL: 30" ]]
}

# ============================================================================
# Function existence tests
# ============================================================================

@test "url_discovery::init_cache function exists" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        type -t url_discovery::init_cache
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "url_discovery::get_cache function exists" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        type -t url_discovery::get_cache
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "url_discovery::set_cache function exists" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        type -t url_discovery::set_cache
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "url_discovery::parse_config function exists" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        type -t url_discovery::parse_config
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "url_discovery::validate_url function exists" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        type -t url_discovery::validate_url
    "
    
    assert_success
    [ "$output" = "function" ]
}

# ============================================================================
# Cache management tests
# ============================================================================

@test "init_cache creates cache directory" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        url_discovery::init_cache
        [ -d '/tmp/vrooli-url-discovery' ] && echo 'CACHE_DIR_EXISTS'
    "
    
    assert_success
    [[ "$output" =~ "CACHE_DIR_EXISTS" ]]
}

@test "set_cache and get_cache work correctly" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        url_discovery::init_cache
        
        # Set cache
        test_data='{\"url\": \"http://localhost:8080\"}'
        url_discovery::set_cache 'test-resource' \"\$test_data\"
        
        # Get cache
        cached_data=\$(url_discovery::get_cache 'test-resource')
        echo \"\$cached_data\"
    "
    
    assert_success
    [[ "$output" =~ "http://localhost:8080" ]]
}

@test "get_cache returns error for expired cache" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        url_discovery::init_cache
        
        # Create old cache file
        cache_file='/tmp/vrooli-url-discovery/test-resource.json'
        echo '{\"old\": \"data\"}' > \"\$cache_file\"
        touch -t 200001010000 \"\$cache_file\"  # Set old timestamp
        
        # Try to get expired cache
        url_discovery::get_cache 'test-resource' 2>/dev/null
    "
    
    [ "$status" -eq 1 ]
}

@test "clear_cache removes all cache files" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        url_discovery::init_cache
        
        # Create some cache files
        url_discovery::set_cache 'resource1' '{\"test\": 1}'
        url_discovery::set_cache 'resource2' '{\"test\": 2}'
        
        # Clear all cache
        url_discovery::clear_cache
        
        # Check if files are gone
        [ ! -f '/tmp/vrooli-url-discovery/resource1.json' ] && echo 'CACHE_CLEARED'
    "
    
    assert_success
    [[ "$output" =~ "CACHE_CLEARED" ]]
}

# ============================================================================
# Configuration parsing tests
# ============================================================================

@test "parse_config handles known resources" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        
        # Parse config for ollama
        config=\$(url_discovery::parse_config 'ollama')
        echo \"\$config\" | jq -r '.type'
        echo \"\$config\" | jq -r '.default_port'
        echo \"\$config\" | jq -r '.name'
    "
    
    assert_success
    [[ "$output" =~ "http" ]]
    [[ "$output" =~ "11434" ]]
    [[ "$output" =~ "Ollama" ]]
}

@test "parse_config returns error for unknown resource" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        
        url_discovery::parse_config 'unknown-resource'
    "
    
    [ "$status" -eq 1 ]
}

# ============================================================================
# URL validation tests
# ============================================================================

@test "validate_url handles TCP connections" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        
        # Mock timeout command
        timeout() {
            # Simulate successful TCP connection
            return 0
        }
        export -f timeout
        
        url_discovery::validate_url 'localhost:5432' 3
    "
    
    assert_success
}

@test "validate_url handles HTTP URLs" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        
        # Mock curl command
        curl() {
            # Simulate successful HTTP response
            return 0
        }
        export -f curl
        
        url_discovery::validate_url 'http://localhost:8080' 3
    "
    
    assert_success
}

# ============================================================================
# Resource configuration tests
# ============================================================================

@test "resource configurations include all major categories" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        
        # Check for resources from different categories
        url_discovery::parse_config 'postgres' >/dev/null && echo 'STORAGE_OK'
        url_discovery::parse_config 'ollama' >/dev/null && echo 'AI_OK'
        url_discovery::parse_config 'n8n' >/dev/null && echo 'AUTOMATION_OK'
        url_discovery::parse_config 'judge0' >/dev/null && echo 'EXECUTION_OK'
        url_discovery::parse_config 'searxng' >/dev/null && echo 'SEARCH_OK'
    "
    
    assert_success
    [[ "$output" =~ "STORAGE_OK" ]]
    [[ "$output" =~ "AI_OK" ]]
    [[ "$output" =~ "AUTOMATION_OK" ]]
    [[ "$output" =~ "EXECUTION_OK" ]]
    [[ "$output" =~ "SEARCH_OK" ]]
}

@test "CLI resources are handled correctly" {
    run bash -c "
        export var_SCRIPTS_RESOURCES_DIR='/test/resources'
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils /test/resources
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        cat > /test/resources/common.sh << 'EOF'
[[ -n \"\${VROOLI_COMMON_SOURCED:-}\" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/url-discovery.sh'
        
        # Check Claude Code (CLI tool)
        config=\$(url_discovery::parse_config 'claude-code')
        echo \"\$config\" | jq -r '.type'
    "
    
    assert_success
    [[ "$output" =~ "cli" ]]
}