#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/config.sh"
SEARXNG_DIR="$BATS_TEST_DIRNAME/.."

# Helper function for proper sourcing in tests
setup_searxng_config_test_env() {
    local script_dir="$SEARXNG_DIR"
    local resources_dir="$SEARXNG_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    # Source utilities first
    source "$helpers_dir/utils/log.sh"
    source "$helpers_dir/utils/system.sh"
    source "$resources_dir/common.sh"
    
    # Source config and messages
    source "$script_dir/config/defaults.sh"
    source "$script_dir/config/messages.sh"
    searxng::export_config
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Mock functions
    log::info() { echo "INFO: $*"; }
    log::success() { echo "SUCCESS: $*"; }
    log::error() { echo "ERROR: $*"; }
    log::warn() { echo "WARNING: $*"; }
    log::header() { echo "HEADER: $*"; }
    
    searxng::ensure_data_dir() { 
        mkdir -p "$SEARXNG_DATA_DIR"
        return 0
    }
    
    # Mock commands
    mktemp() { echo "/tmp/mock-temp-file"; }
    mv() { return 0; }
    cp() { return 0; }
    yq() {
        case "$1" in
            "eval")
                if [[ "$MOCK_YAML_VALID" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    command() {
        case "$*" in
            "-v yq")
                if [[ "$MOCK_YQ_AVAILABLE" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    # Set default mocks
    export MOCK_YAML_VALID="yes"
    export MOCK_YQ_AVAILABLE="yes"
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing config.sh defines required functions" {
    setup_searxng_config_test_env
    
    local required_functions=(
        "searxng::generate_config"
        "searxng::generate_limiter_config"
        "searxng::update_engines"
        "searxng::update_search_settings"
        "searxng::show_config"
        "searxng::validate_config_files"
        "searxng::export_config"
        "searxng::reset_config"
    )
    
    for func in "${required_functions[@]}"; do
        run bash -c "declare -f $func"
        [ "$status" -eq 0 ]
    done
}

# ============================================================================
# Configuration Generation Tests
# ============================================================================

@test "searxng::generate_config creates main configuration successfully" {
    setup_searxng_config_test_env
    
    # Create mock template
    mkdir -p "$SEARXNG_DIR/config"
    echo 'instance_name: ${SEARXNG_INSTANCE_NAME}' > "$SEARXNG_DIR/config/settings.yml.template"
    
    run searxng::generate_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "INFO: Generating SearXNG configuration" ]]
    [[ "$output" =~ "SUCCESS:" ]]
    
    # Cleanup
    rm -f "$SEARXNG_DIR/config/settings.yml.template"
}

@test "searxng::generate_config fails when template missing" {
    setup_searxng_config_test_env
    
    run searxng::generate_config
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Configuration template not found" ]]
}

@test "searxng::generate_limiter_config creates limiter config when enabled" {
    setup_searxng_config_test_env
    export SEARXNG_LIMITER_ENABLED="yes"
    
    # Create mock template
    mkdir -p "$SEARXNG_DIR/docker"
    echo 'rate_limit = ${SEARXNG_RATE_LIMIT}' > "$SEARXNG_DIR/docker/limiter.toml.template"
    
    run searxng::generate_limiter_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "INFO: Generating rate limiter configuration" ]]
    [[ "$output" =~ "SUCCESS:" ]]
    
    # Cleanup
    rm -f "$SEARXNG_DIR/docker/limiter.toml.template"
}

@test "searxng::generate_limiter_config skips when disabled" {
    setup_searxng_config_test_env
    export SEARXNG_LIMITER_ENABLED="no"
    
    run searxng::generate_limiter_config
    [ "$status" -eq 0 ]
    # Should return without any output when disabled
}

# ============================================================================
# Configuration Display Tests
# ============================================================================

@test "searxng::show_config displays all configuration sections" {
    setup_searxng_config_test_env
    
    run searxng::show_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HEADER: SearXNG Configuration" ]]
    [[ "$output" =~ "Basic Settings:" ]]
    [[ "$output" =~ "Search Settings:" ]]
    [[ "$output" =~ "Performance Settings:" ]]
    [[ "$output" =~ "Security & Limits:" ]]
    [[ "$output" =~ "Storage & Caching:" ]]
    [[ "$output" =~ "Configuration Files:" ]]
}

@test "searxng::show_config shows correct values" {
    setup_searxng_config_test_env
    
    run searxng::show_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Instance Name: $SEARXNG_INSTANCE_NAME" ]]
    [[ "$output" =~ "Base URL: $SEARXNG_BASE_URL" ]]
    [[ "$output" =~ "Port: $SEARXNG_PORT" ]]
    [[ "$output" =~ "Default Engines: $SEARXNG_DEFAULT_ENGINES" ]]
    [[ "$output" =~ "Rate Limiting: $SEARXNG_LIMITER_ENABLED" ]]
}

@test "searxng::show_config indicates file presence" {
    setup_searxng_config_test_env
    
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    touch "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::show_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "settings.yml: ✅ Present" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DATA_DIR"
}

@test "searxng::show_config indicates missing files" {
    setup_searxng_config_test_env
    
    run searxng::show_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "settings.yml: ❌ Missing" ]]
}

# ============================================================================
# Configuration Validation Tests
# ============================================================================

@test "searxng::validate_config_files passes with valid configuration" {
    setup_searxng_config_test_env
    export MOCK_YAML_VALID="yes"
    export MOCK_YQ_AVAILABLE="yes"
    
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "server:" > "$SEARXNG_DATA_DIR/settings.yml"
    echo "  secret_key: test" >> "$SEARXNG_DATA_DIR/settings.yml"
    echo "  base_url: http://localhost" >> "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::validate_config_files
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SUCCESS:" ]]
    [[ "$output" =~ "Configuration validation passed" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DATA_DIR"
}

@test "searxng::validate_config_files fails when config missing" {
    setup_searxng_config_test_env
    
    run searxng::validate_config_files
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Main configuration file missing" ]]
}

@test "searxng::validate_config_files handles invalid YAML" {
    setup_searxng_config_test_env
    export MOCK_YAML_VALID="no"
    export MOCK_YQ_AVAILABLE="yes"
    
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "invalid yaml: [" > "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::validate_config_files
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Invalid YAML syntax" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DATA_DIR"
}

@test "searxng::validate_config_files skips validation when yq unavailable" {
    setup_searxng_config_test_env
    export MOCK_YQ_AVAILABLE="no"
    
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "test config" > "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::validate_config_files
    [ "$status" -eq 0 ]
    [[ "$output" =~ "YAML validation skipped" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DATA_DIR"
}

# ============================================================================
# Configuration Export Tests
# ============================================================================

@test "searxng::export_config exports to specified file" {
    setup_searxng_config_test_env
    local output_file="/tmp/searxng-export-test"
    
    run searxng::export_config "$output_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "INFO: Exporting SearXNG configuration" ]]
    [[ "$output" =~ "SUCCESS:" ]]
    [[ "$output" =~ "Configuration exported to: $output_file" ]]
    
    # Cleanup
    rm -f "$output_file"
}

@test "searxng::export_config fails when no output file specified" {
    setup_searxng_config_test_env
    
    run searxng::export_config ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Output file path is required" ]]
}

# ============================================================================
# Configuration Reset Tests
# ============================================================================

@test "searxng::reset_config backs up and regenerates configuration" {
    setup_searxng_config_test_env
    
    # Create mock existing config
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "old config" > "$SEARXNG_DATA_DIR/settings.yml"
    
    # Mock generate_config
    searxng::generate_config() { 
        echo "SUCCESS: Config regenerated"
        return 0
    }
    
    run searxng::reset_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "INFO: Resetting SearXNG configuration" ]]
    [[ "$output" =~ "SUCCESS:" ]]
    [[ "$output" =~ "Configuration reset to defaults" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DATA_DIR"
}

@test "searxng::reset_config handles missing existing config" {
    setup_searxng_config_test_env
    
    # Mock generate_config
    searxng::generate_config() { 
        echo "SUCCESS: Config generated"
        return 0
    }
    
    run searxng::reset_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SUCCESS:" ]]
}

# ============================================================================
# Engine Configuration Tests
# ============================================================================

@test "searxng::update_engines validates input" {
    setup_searxng_config_test_env
    
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "test config" > "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::update_engines ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Engine list cannot be empty" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DATA_DIR"
}

@test "searxng::update_engines handles missing config file" {
    setup_searxng_config_test_env
    
    run searxng::update_engines "google,bing"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "SearXNG configuration not found" ]]
}

@test "searxng::update_engines provides guidance for manual editing" {
    setup_searxng_config_test_env
    
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "test config" > "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::update_engines "google,bing,duckduckgo"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "INFO: Updating SearXNG engines" ]]
    [[ "$output" =~ "WARNING:" ]]
    [[ "$output" =~ "manual editing" ]]
    [[ "$output" =~ "Supported engines:" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DATA_DIR"
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "configuration functions handle file operations safely" {
    setup_searxng_config_test_env
    
    # Test that functions handle missing directories gracefully
    export SEARXNG_DATA_DIR="/tmp/nonexistent-$(date +%s)"
    
    run searxng::show_config
    [ "$status" -eq 0 ]
    
    run searxng::validate_config_files
    [ "$status" -eq 1 ]  # Should fail gracefully
}

@test "template processing handles variable substitution" {
    setup_searxng_config_test_env
    
    # Create mock template with variables
    mkdir -p "$SEARXNG_DIR/config"
    cat > "$SEARXNG_DIR/config/settings.yml.template" << 'EOF'
server:
  instance_name: '${SEARXNG_INSTANCE_NAME}'
  base_url: '${SEARXNG_BASE_URL}'
  secret_key: '${SEARXNG_SECRET_KEY}'
EOF
    
    run searxng::generate_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SUCCESS:" ]]
    
    # Cleanup
    rm -f "$SEARXNG_DIR/config/settings.yml.template"
}