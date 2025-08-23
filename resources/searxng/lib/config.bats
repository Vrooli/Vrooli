#!/usr/bin/env bats

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Setup for each test
setup() {
    # Source var.sh to get path variables
    source "${BATS_TEST_DIRNAME}/../../../../lib/utils/var.sh"
    
    # Load Vrooli test infrastructure
    source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"
    
    # Path to the script under test
    SCRIPT_PATH="$BATS_TEST_DIRNAME/config.sh"
    SEARXNG_DIR="$BATS_TEST_DIRNAME/.."
    
    # Source dependencies using var_ variables
    # shellcheck disable=SC1091
    source "${var_LOG_FILE}"
    # shellcheck disable=SC1091
    source "${var_SYSTEM_COMMANDS_FILE}"
    # shellcheck disable=SC1091
    source "${var_RESOURCES_COMMON_FILE}"
    
    # Source config and messages
    source "$SEARXNG_DIR/config/defaults.sh"
    source "$SEARXNG_DIR/config/messages.sh"
    searxng::export_config
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # SearXNG-specific mock functions
    searxng::ensure_data_dir() { 
        mkdir -p "$SEARXNG_DATA_DIR"
        return 0
    }
    
    # Mock commands specific to SearXNG config
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
    
    # Setup standard mocks AFTER sourcing real utilities (to override them)
    vrooli_auto_setup
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing config.sh defines required functions" {
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
    # Create mock template
    mkdir -p "$SEARXNG_DIR/config"
    echo 'instance_name: ${SEARXNG_INSTANCE_NAME}' > "$SEARXNG_DIR/config/settings.yml.template"
    
    run searxng::generate_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[INFO]" ]]
    [[ "$output" =~ "[SUCCESS]" ]]
    
    # Cleanup
    trash::safe_remove "$SEARXNG_DIR/config/settings.yml.template" --test-cleanup
}

@test "searxng::generate_config fails when template missing" {
    run searxng::generate_config
    [ "$status" -eq 1 ]
    [[ "$output" =~ "[ERROR]" ]]
    [[ "$output" =~ "Configuration template not found" ]]
}

@test "searxng::generate_limiter_config creates limiter config when enabled" {
    export SEARXNG_LIMITER_ENABLED="yes"
    
    # Create mock template
    mkdir -p "$SEARXNG_DIR/docker"
    echo 'rate_limit = ${SEARXNG_RATE_LIMIT}' > "$SEARXNG_DIR/docker/limiter.toml.template"
    
    run searxng::generate_limiter_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[INFO]" ]]
    [[ "$output" =~ "[SUCCESS]" ]]
    
    # Cleanup
    trash::safe_remove "$SEARXNG_DIR/docker/limiter.toml.template" --test-cleanup
}

@test "searxng::generate_limiter_config skips when disabled" {
    export SEARXNG_LIMITER_ENABLED="no"
    
    run searxng::generate_limiter_config
    [ "$status" -eq 0 ]
    # Should return without any output when disabled
}

# ============================================================================
# Configuration Display Tests
# ============================================================================

@test "searxng::show_config displays all configuration sections" {
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
    run searxng::show_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Instance Name: $SEARXNG_INSTANCE_NAME" ]]
    [[ "$output" =~ "Base URL: $SEARXNG_BASE_URL" ]]
    [[ "$output" =~ "Port: $SEARXNG_PORT" ]]
    [[ "$output" =~ "Default Engines: $SEARXNG_DEFAULT_ENGINES" ]]
    [[ "$output" =~ "Rate Limiting: $SEARXNG_LIMITER_ENABLED" ]]
}

@test "searxng::show_config indicates file presence" {
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    touch "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::show_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "settings.yml: ✅ Present" ]]
    
    # Cleanup
    trash::safe_remove "$SEARXNG_DATA_DIR" --test-cleanup
}

@test "searxng::show_config indicates missing files" {
    run searxng::show_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "settings.yml: ❌ Missing" ]]
}

# ============================================================================
# Configuration Validation Tests
# ============================================================================

@test "searxng::validate_config_files passes with valid configuration" {
    export MOCK_YAML_VALID="yes"
    export MOCK_YQ_AVAILABLE="yes"
    
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "server:" > "$SEARXNG_DATA_DIR/settings.yml"
    echo "  secret_key: test" >> "$SEARXNG_DATA_DIR/settings.yml"
    echo "  base_url: http://localhost" >> "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::validate_config_files
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[SUCCESS]" ]]
    [[ "$output" =~ "Configuration validation passed" ]]
    
    # Cleanup
    trash::safe_remove "$SEARXNG_DATA_DIR" --test-cleanup
}

@test "searxng::validate_config_files fails when config missing" {
    run searxng::validate_config_files
    [ "$status" -eq 1 ]
    [[ "$output" =~ "[ERROR]" ]]
    [[ "$output" =~ "Main configuration file missing" ]]
}

@test "searxng::validate_config_files handles invalid YAML" {
    export MOCK_YAML_VALID="no"
    export MOCK_YQ_AVAILABLE="yes"
    
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "invalid yaml: [" > "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::validate_config_files
    [ "$status" -eq 1 ]
    [[ "$output" =~ "[ERROR]" ]]
    [[ "$output" =~ "Invalid YAML syntax" ]]
    
    # Cleanup
    trash::safe_remove "$SEARXNG_DATA_DIR" --test-cleanup
}

@test "searxng::validate_config_files skips validation when yq unavailable" {
    export MOCK_YQ_AVAILABLE="no"
    
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "test config" > "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::validate_config_files
    [ "$status" -eq 0 ]
    [[ "$output" =~ "YAML validation skipped" ]]
    
    # Cleanup
    trash::safe_remove "$SEARXNG_DATA_DIR" --test-cleanup
}

# ============================================================================
# Configuration Export Tests
# ============================================================================

@test "searxng::export_config exports to specified file" {
    local output_file="/tmp/searxng-export-test"
    
    run searxng::export_config "$output_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[INFO]" ]]
    [[ "$output" =~ "[SUCCESS]" ]]
    [[ "$output" =~ "Configuration exported to: $output_file" ]]
    
    # Cleanup
    trash::safe_remove "$output_file" --test-cleanup
}

@test "searxng::export_config fails when no output file specified" {
    run searxng::export_config ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "[ERROR]" ]]
    [[ "$output" =~ "Output file path is required" ]]
}

# ============================================================================
# Configuration Reset Tests
# ============================================================================

@test "searxng::reset_config backs up and regenerates configuration" {
    # Create mock existing config
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "old config" > "$SEARXNG_DATA_DIR/settings.yml"
    
    # Mock generate_config
    searxng::generate_config() { 
        echo "[SUCCESS] Config regenerated"
        return 0
    }
    
    run searxng::reset_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[INFO]" ]]
    [[ "$output" =~ "[SUCCESS]" ]]
    [[ "$output" =~ "Configuration reset to defaults" ]]
    
    # Cleanup
    trash::safe_remove "$SEARXNG_DATA_DIR" --test-cleanup
}

@test "searxng::reset_config handles missing existing config" {
    # Mock generate_config
    searxng::generate_config() { 
        echo "[SUCCESS] Config generated"
        return 0
    }
    
    run searxng::reset_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[SUCCESS]" ]]
}

# ============================================================================
# Engine Configuration Tests
# ============================================================================

@test "searxng::update_engines validates input" {
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "test config" > "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::update_engines ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "[ERROR]" ]]
    [[ "$output" =~ "Engine list cannot be empty" ]]
    
    # Cleanup
    trash::safe_remove "$SEARXNG_DATA_DIR" --test-cleanup
}

@test "searxng::update_engines handles missing config file" {
    run searxng::update_engines "google,bing"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "[ERROR]" ]]
    [[ "$output" =~ "SearXNG configuration not found" ]]
}

@test "searxng::update_engines provides guidance for manual editing" {
    # Create mock config file
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "test config" > "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::update_engines "google,bing,duckduckgo"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[INFO]" ]]
    [[ "$output" =~ "[WARNING]" ]]
    [[ "$output" =~ "manual editing" ]]
    [[ "$output" =~ "Supported engines:" ]]
    
    # Cleanup
    trash::safe_remove "$SEARXNG_DATA_DIR" --test-cleanup
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "configuration functions handle file operations safely" {
    # Test that functions handle missing directories gracefully
    export SEARXNG_DATA_DIR="/tmp/nonexistent-$(date +%s)"
    
    run searxng::show_config
    [ "$status" -eq 0 ]
    
    run searxng::validate_config_files
    [ "$status" -eq 1 ]  # Should fail gracefully
}

@test "template processing handles variable substitution" {
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
    [[ "$output" =~ "[SUCCESS]" ]]
    
    # Cleanup
    trash::safe_remove "$SEARXNG_DIR/config/settings.yml.template" --test-cleanup
}