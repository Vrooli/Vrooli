#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Load test helpers
load "../../__test/helpers/bats-support/load"
load "../../__test/helpers/bats-assert/load"

# Load mocks
load "../../__test/fixtures/mocks/logs.sh"

# Script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/firewall.sh"

setup() {
    # Initialize mocks
    mock::cleanup_logs "" || true
    
    # Set up test environment
    export ENVIRONMENT="development"
    export PORT_UI="3000"
    export PORT_SERVER="5329"
    export SUDO_MODE="auto"  # Set a mode that allows sudo by default
    
    # Mock ufw state
    export MOCK_UFW_ACTIVE="false"
    export MOCK_UFW_DEFAULT_INCOMING="allow"
    export MOCK_UFW_DEFAULT_OUTGOING="allow"
    export MOCK_UFW_RULES=""
    export MOCK_UFW_AVAILABLE="true"
    export MOCK_UFW_SUCCESS="true"
    
    # Mock sudo availability for tests
    export MOCK_SUDO_AVAILABLE="true"
    export MOCK_SUDO_PASSWORDLESS="true"
    
    # Source necessary dependencies that the script needs
    source "${BATS_TEST_DIRNAME}/../utils/var.sh"
    source "${var_LOG_FILE}"
    source "${var_EXIT_CODES_FILE}"
    
    # Mock functions need to be defined before sourcing flow.sh
    setup_mocks
    
    # Now source flow.sh with our mocks in place
    source "${var_FLOW_FILE}"
    source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
}

teardown() {
    # Clean up mocks
    mock::cleanup_logs "" || true
    
    # Unset environment variables
    unset ENVIRONMENT PORT_UI PORT_SERVER || true
    unset MOCK_UFW_ACTIVE MOCK_UFW_DEFAULT_INCOMING MOCK_UFW_DEFAULT_OUTGOING || true
    unset MOCK_UFW_RULES MOCK_UFW_AVAILABLE MOCK_UFW_SUCCESS || true
    unset MOCK_SUDO_AVAILABLE MOCK_SUDO_PASSWORDLESS || true
}

# Define all mocks in a function that can be called from setup
setup_mocks() {
    # Mock system::is_command to control ufw availability
    system::is_command() {
        case "$1" in
            ufw)
                [[ "${MOCK_UFW_AVAILABLE:-true}" == "true" ]]
                ;;
            *)
                command -v "$1" >/dev/null 2>&1
                ;;
        esac
    }
    export -f system::is_command

    # Mock command for checking sudo availability
    command() {
        if [[ "$1" == "-v" ]] && [[ "$2" == "sudo" ]]; then
            [[ "${MOCK_SUDO_AVAILABLE:-true}" == "true" ]]
        else
            builtin command "$@"
        fi
    }
    export -f command

    # Mock sudo command
    sudo() {
        # Handle sudo -n true test (for passwordless check)
        if [[ "$1" == "-n" ]] && [[ "$2" == "true" ]]; then
            if [[ "${MOCK_SUDO_PASSWORDLESS:-true}" == "true" ]]; then
                return 0
            else
                return 1
            fi
        fi
        
        # Pass through to the actual command being run (remove 'sudo' prefix)
        "$@"
    }
    export -f sudo

    # Mock ufw command
    ufw() {
        local args="$*"
        
        # Handle status commands
        if [[ "$args" == "status" ]]; then
            if [[ "${MOCK_UFW_ACTIVE}" == "true" ]]; then
                echo "Status: active"
            else
                echo "Status: inactive"
            fi
            echo ""
            echo "To                         Action      From"
            echo "--                         ------      ----"
            if [[ -n "${MOCK_UFW_RULES}" ]]; then
                # Format rules to show outbound rules with (out) suffix
                echo "${MOCK_UFW_RULES}" | sed 's/ALLOW OUT   Anywhere/ALLOW       Anywhere (out)/g'
            fi
            return 0
        fi
        
        if [[ "$args" == "status verbose" ]]; then
            if [[ "${MOCK_UFW_ACTIVE}" == "true" ]]; then
                echo "Status: active"
            else
                echo "Status: inactive"
            fi
            echo "Logging: on (low)"
            echo "Default: ${MOCK_UFW_DEFAULT_INCOMING:-deny} (incoming), ${MOCK_UFW_DEFAULT_OUTGOING:-allow} (outgoing), disabled (routed)"
            echo "New profiles: skip"
            echo ""
            echo "To                         Action      From"
            echo "--                         ------      ----"
            if [[ -n "${MOCK_UFW_RULES}" ]]; then
                echo "${MOCK_UFW_RULES}"
            fi
            return 0
        fi
        
        if [[ "$args" == "status numbered" ]]; then
            if [[ "${MOCK_UFW_ACTIVE}" == "true" ]]; then
                echo "Status: active"
            else
                echo "Status: inactive"
            fi
            echo ""
            echo "     To                         Action      From"
            echo "     --                         ------      ----"
            if [[ -n "${MOCK_UFW_RULES}" ]]; then
                local line_num=1
                while IFS= read -r rule; do
                    echo "[ $line_num] $rule"
                    ((line_num++))
                done <<< "${MOCK_UFW_RULES}"
            fi
            return 0
        fi
        
        # Handle enable/disable commands
        if [[ "$args" == "--force enable" ]]; then
            MOCK_UFW_ACTIVE="true"
            return 0
        fi
        
        if [[ "$args" == "disable" ]]; then
            MOCK_UFW_ACTIVE="false"
            return 0
        fi
        
        # Handle default policy commands
        if [[ "$args" == "default allow outgoing" ]]; then
            MOCK_UFW_DEFAULT_OUTGOING="allow"
            return 0
        fi
        
        if [[ "$args" == "default deny incoming" ]]; then
            MOCK_UFW_DEFAULT_INCOMING="deny"
            return 0
        fi
        
        # Handle allow commands
        if [[ "$args" =~ ^allow ]]; then
            # Extract port/protocol from the command
            local rule_text="${args#allow }"
            
            # Handle outbound rules
            if [[ "$rule_text" =~ ^out ]]; then
                rule_text="${rule_text#out }"
                # Add to rules list with OUT indicator
                if [[ -z "${MOCK_UFW_RULES}" ]]; then
                    MOCK_UFW_RULES="${rule_text}                      ALLOW OUT   Anywhere"
                else
                    MOCK_UFW_RULES="${MOCK_UFW_RULES}
${rule_text}                      ALLOW OUT   Anywhere"
                fi
            else
                # Add to rules list (incoming)
                if [[ -z "${MOCK_UFW_RULES}" ]]; then
                    MOCK_UFW_RULES="${rule_text}                      ALLOW       Anywhere"
                else
                    MOCK_UFW_RULES="${MOCK_UFW_RULES}
${rule_text}                      ALLOW       Anywhere"
                fi
            fi
            return 0
        fi
        
        # Handle reload command
        if [[ "$args" == "reload" ]]; then
            return 0
        fi
        
        # Default success for unhandled commands
        [[ "${MOCK_UFW_SUCCESS:-true}" == "true" ]]
    }
    export -f ufw

    # Mock sysctl command
    sysctl() {
        if [[ "$1" == "-p" ]]; then
            return 0
        fi
        command sysctl "$@"
    }
    export -f sysctl
}

# Tests for firewall::is_development function
@test "firewall::is_development recognizes 'development' environment" {
    source "$SCRIPT_PATH"
    
    run firewall::is_development "development"
    assert_success
}

@test "firewall::is_development recognizes 'dev' environment" {
    source "$SCRIPT_PATH"
    
    run firewall::is_development "dev"
    assert_success
}

@test "firewall::is_development recognizes uppercase DEV environment" {
    source "$SCRIPT_PATH"
    
    run firewall::is_development "DEV"
    assert_success
}

@test "firewall::is_development rejects production environment" {
    source "$SCRIPT_PATH"
    
    run firewall::is_development "production"
    assert_failure
}

@test "firewall::is_development uses environment variable when no argument" {
    source "$SCRIPT_PATH"
    
    export ENVIRONMENT="development"
    run firewall::is_development
    assert_success
    
    export ENVIRONMENT="production"
    run firewall::is_development
    assert_failure
}

# Tests for firewall::setup function
@test "firewall::setup requires environment parameter" {
    unset ENVIRONMENT
    source "$SCRIPT_PATH"
    
    run firewall::setup
    assert_failure
    assert_output --partial "Environment is required to setup firewall"
}

@test "firewall::setup skips when sudo mode is skip" {
    export SUDO_MODE="skip"
    # Re-setup mocks since SUDO_MODE changed
    setup_mocks
    source "$SCRIPT_PATH"
    
    run firewall::setup "development"
    assert_success
    assert_output --partial "Skipping firewall setup due to sudo mode"
}

@test "firewall::setup skips when ufw is not available" {
    export MOCK_UFW_AVAILABLE="false"
    export SUDO_MODE="auto"  # Explicitly allow sudo for this test
    source "$SCRIPT_PATH"
    setup_mocks  # Set up mocks after sourcing to override any functions from the script
    
    run firewall::setup "development"
    assert_success
    assert_output --partial "ufw not found; skipping firewall setup"
}

@test "firewall::setup enables UFW when inactive" {
    export MOCK_UFW_ACTIVE="false"
    export SUDO_MODE="auto"  # Allow sudo
    source "$SCRIPT_PATH"
    setup_mocks  # Set up mocks after sourcing
    
    run firewall::setup "development"
    assert_success
    assert_output --partial "Enabling UFW"
    # Note: State change happens inside the ufw mock function during execution
}

@test "firewall::setup skips enabling when UFW already active" {
    export MOCK_UFW_ACTIVE="true"
    export SUDO_MODE="auto"  # Allow sudo
    setup_mocks
    source "$SCRIPT_PATH"
    
    run firewall::setup "development"
    assert_success
    assert_output --partial "UFW already active"
}

@test "firewall::setup sets default policies when not configured" {
    export MOCK_UFW_DEFAULT_INCOMING="allow"
    export MOCK_UFW_DEFAULT_OUTGOING="deny"
    export SUDO_MODE="auto"  # Allow sudo
    source "$SCRIPT_PATH"
    setup_mocks  # Set up mocks after sourcing
    
    run firewall::setup "development"
    assert_success
    assert_output --partial "Setting default policies"
    # Note: State changes happen inside the ufw mock function during execution
}

@test "firewall::setup skips default policies when already set" {
    export MOCK_UFW_DEFAULT_INCOMING="deny"
    export MOCK_UFW_DEFAULT_OUTGOING="allow"
    export SUDO_MODE="auto"  # Allow sudo
    setup_mocks
    source "$SCRIPT_PATH"
    
    run firewall::setup "development"
    assert_success
    assert_output --partial "Default policies already set"
}

@test "firewall::setup adds critical outbound rules" {
    export SUDO_MODE="auto"  # Allow sudo
    setup_mocks
    source "$SCRIPT_PATH"
    
    run firewall::setup "development"
    assert_success
    assert_output --partial "Adding critical outbound rule"
    assert_output --partial "80/tcp"
    assert_output --partial "443/tcp"
    assert_output --partial "53"
}

@test "firewall::setup adds standard incoming ports" {
    export SUDO_MODE="auto"  # Allow sudo
    export MOCK_UFW_RULES=""  # Start with no rules
    source "$SCRIPT_PATH"
    setup_mocks  # Set up mocks after sourcing to override any functions from the script
    
    run firewall::setup "production"
    assert_success
    assert_output --partial "Allowing incoming 80/tcp"
    assert_output --partial "Allowing incoming 443/tcp"
    assert_output --partial "Allowing incoming 22/tcp"
    assert_output --partial "Allowing incoming 3389/tcp"
}

@test "firewall::setup adds development ports in dev environment" {
    export PORT_UI="3000"
    export PORT_SERVER="5329"
    export SUDO_MODE="auto"  # Allow sudo
    setup_mocks
    source "$SCRIPT_PATH"
    
    run firewall::setup "development"
    assert_success
    assert_output --partial "Allowing incoming 3000/tcp"
    assert_output --partial "Allowing incoming 5329/tcp"
}

@test "firewall::setup skips development ports in production" {
    export PORT_UI="3000"
    export PORT_SERVER="5329"
    export SUDO_MODE="auto"  # Allow sudo
    setup_mocks
    source "$SCRIPT_PATH"
    
    run firewall::setup "production"
    assert_success
    refute_output --partial "Allowing incoming 3000/tcp"
    refute_output --partial "Allowing incoming 5329/tcp"
}

@test "firewall::setup skips existing incoming rules" {
    export SUDO_MODE="auto"  # Allow sudo
    source "$SCRIPT_PATH"
    
    # Set up the mock rules AFTER setting up the functions
    setup_mocks
    export MOCK_UFW_RULES="80/tcp                      ALLOW       Anywhere
443/tcp                     ALLOW       Anywhere"
    
    run firewall::setup "production"
    assert_success
    assert_output --partial "Incoming rule for 80/tcp already exists"
    assert_output --partial "Incoming rule for 443/tcp already exists"
}

@test "firewall::setup reloads when changes are made" {
    export MOCK_UFW_ACTIVE="false"
    export SUDO_MODE="auto"  # Allow sudo
    setup_mocks
    source "$SCRIPT_PATH"
    
    run firewall::setup "development"
    assert_success
    assert_output --partial "Reloading sysctl and UFW"
}

@test "firewall::setup skips reload when no changes" {
    export SUDO_MODE="auto"  # Allow sudo
    source "$SCRIPT_PATH"
    
    # Set up the mock state AFTER setting up the functions
    setup_mocks
    export MOCK_UFW_ACTIVE="true"
    export MOCK_UFW_DEFAULT_INCOMING="deny"
    export MOCK_UFW_DEFAULT_OUTGOING="allow"
    export MOCK_UFW_RULES="80/tcp                      ALLOW OUT   Anywhere
443/tcp                     ALLOW OUT   Anywhere
53                          ALLOW OUT   Anywhere
53/udp                      ALLOW OUT   Anywhere
80/tcp                      ALLOW       Anywhere
443/tcp                     ALLOW       Anywhere
22/tcp                      ALLOW       Anywhere
3389/tcp                    ALLOW       Anywhere"
    
    run firewall::setup "production"
    assert_success
    assert_output --partial "No firewall changes needed; skipping reload"
}

@test "firewall::setup completes successfully" {
    export SUDO_MODE="auto"  # Allow sudo
    setup_mocks
    source "$SCRIPT_PATH"
    
    run firewall::setup "development"
    assert_success
    assert_output --partial "Firewall setup complete"
}

@test "firewall::setup can be run directly with arguments" {
    export SUDO_MODE="auto"  # Allow sudo
    setup_mocks
    run bash "$SCRIPT_PATH" "development"
    assert_success
    assert_output --partial "Firewall setup complete"
}

@test "firewall::setup uses environment variable when run directly" {
    export SUDO_MODE="auto"  # Allow sudo
    setup_mocks
    export ENVIRONMENT="production"
    run bash "$SCRIPT_PATH"
    assert_success
    assert_output --partial "Firewall setup complete"
}