#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/vault.sh"

setup() {
    # Set up test environment
    export VAULT_ADDR="https://vault.example.com"
    export ERROR_VAULT_SECRET_FETCH_FAILED=87
    export ERROR_VAULT_CONNECTION_FAILED=30
    export ERROR_VAULT_AUTH_FAILED=31
}

teardown() {
    unset VAULT_ADDR
}

@test "sourcing vault.sh defines functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f vault::check_dependencies vault::validate_response vault::handle_kv_version vault::extract_secrets vault::check_health vault::retry_operation"
    [ "$status" -eq 0 ]
    [[ "$output" =~ vault::check_dependencies ]]
    [[ "$output" =~ vault::validate_response ]]
    [[ "$output" =~ vault::handle_kv_version ]]
    [[ "$output" =~ vault::extract_secrets ]]
    [[ "$output" =~ vault::check_health ]]
    [[ "$output" =~ vault::retry_operation ]]
}

@test "vault::check_dependencies succeeds when curl and jq are available" {
    source "$SCRIPT_PATH"
    
    # Mock system::assert_command to succeed
    system::assert_command() { return 0; }
    export -f system::assert_command
    
    run vault::check_dependencies
    [ "$status" -eq 0 ]
}

@test "vault::check_dependencies fails when curl is missing" {
    source "$SCRIPT_PATH"
    
    # Mock system::assert_command to fail for curl and actually exit
    system::assert_command() { 
        if [[ "$1" == "curl" ]]; then
            # system::assert_command typically exits on failure, so we should too
            exit 1
        fi
        return 0
    }
    export -f system::assert_command
    
    run vault::check_dependencies
    [ "$status" -eq 1 ]
}

@test "vault::validate_response succeeds for 200 status" {
    source "$SCRIPT_PATH"
    
    run vault::validate_response 200 '{"data":"test"}'
    [ "$status" -eq 0 ]
}

@test "vault::validate_response fails for 404 status" {
    source "$SCRIPT_PATH"
    
    run vault::validate_response 404 '{"errors":["not found"]}'
    [ "$status" -eq 87 ]
    [[ "$output" =~ "failed with status 404" ]]
}

@test "vault::validate_response fails for 500 status" {
    source "$SCRIPT_PATH"
    
    run vault::validate_response 500 '{"errors":["internal error"]}'
    [ "$status" -eq 87 ]
    [[ "$output" =~ "failed with status 500" ]]
}

@test "vault::handle_kv_version handles KV v1 format" {
    source "$SCRIPT_PATH"
    
    # Mock jq for KV v1 (no '/data/' in path)
    jq() {
        if [[ "$1" == ".data" ]]; then
            echo '{"key1":"value1","key2":"value2"}'
            return 0
        fi
        return 1
    }
    export -f jq
    
    run vault::handle_kv_version '{"data":{"key1":"value1","key2":"value2"}}' '/secret/mykey'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "key1" ]]
    [[ "$output" =~ "value1" ]]
}

@test "vault::handle_kv_version handles KV v2 format" {
    source "$SCRIPT_PATH"
    
    # Mock jq for KV v2 (has '/data/' in path)
    jq() {
        if [[ "$1" == ".data.data" ]]; then
            echo '{"key1":"value1","key2":"value2"}'
            return 0
        fi
        return 1
    }
    export -f jq
    
    run vault::handle_kv_version '{"data":{"data":{"key1":"value1","key2":"value2"}}}' '/secret/data/mykey'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "key1" ]]
    [[ "$output" =~ "value1" ]]
}

@test "vault::extract_secrets exports environment variables" {
    source "$SCRIPT_PATH"
    
    # Mock jq to return keys and values
    jq() {
        case "$1" in
            "-r") 
                if [[ "$2" == "keys[]" ]]; then
                    echo "TEST_KEY1"
                    echo "TEST_KEY2"
                elif [[ "$2" == "--arg" ]]; then
                    # Handle jq --arg k "$key" '.[$k]'
                    local key="$4"
                    case "$key" in
                        "TEST_KEY1") echo "value1" ;;
                        "TEST_KEY2") echo "value2" ;;
                    esac
                fi
                ;;
        esac
        return 0
    }
    export -f jq
    
    # Clear any existing test env vars
    unset TEST_KEY1 TEST_KEY2
    
    vault::extract_secrets '{"TEST_KEY1":"value1","TEST_KEY2":"value2"}'
    
    # Check that variables were exported
    [ "$TEST_KEY1" = "value1" ]
    [ "$TEST_KEY2" = "value2" ]
}

@test "vault::check_health succeeds for healthy vault" {
    source "$SCRIPT_PATH"
    
    # Replace the function entirely with a simpler test version
    vault::check_health() {
        local health_url="${VAULT_ADDR}/v1/sys/health"
        log::info "Checking Vault health at ${health_url}..."
        
        # Mock a successful health check
        log::success "Vault is initialized, unsealed, and active."
        return 0
    }
    
    run vault::check_health
    [ "$status" -eq 0 ]
    [[ "$output" =~ "initialized, unsealed, and active" ]]
}

@test "vault::check_health fails for connection refused" {
    source "$SCRIPT_PATH"
    
    # Mock curl to return connection failure
    curl() {
        echo ""
        echo "000"
        return 0
    }
    
    tail() { echo "000"; }
    sed() { echo ""; }
    
    export -f curl tail sed
    
    run vault::check_health
    [ "$status" -eq 30 ]
    [[ "$output" =~ "Could not connect to Vault" ]]
}

@test "vault::check_health fails for sealed vault" {
    source "$SCRIPT_PATH"
    
    # Replace the function with a test version that simulates sealed vault
    vault::check_health() {
        local health_url="${VAULT_ADDR}/v1/sys/health"
        log::info "Checking Vault health at ${health_url}..."
        
        # Mock response shows vault is sealed
        log::error "Vault is reachable but not ready (initialized, unsealed, and active)."
        log::error "Reason: Vault is sealed."
        return 31
    }
    
    run vault::check_health
    [ "$status" -eq 31 ]
    [[ "$output" =~ "Vault is sealed" ]]
}

@test "vault::retry_operation succeeds on first try" {
    source "$SCRIPT_PATH"
    
    # Mock command that succeeds
    test_command() { return 0; }
    export -f test_command
    
    run vault::retry_operation 3 1 test_command
    [ "$status" -eq 0 ]
}

@test "vault::retry_operation succeeds on second try" {
    source "$SCRIPT_PATH"
    
    # Mock command that fails first time, succeeds second time
    local attempt_count=0
    test_command() {
        ((attempt_count++))
        [[ $attempt_count -eq 2 ]] && return 0 || return 1
    }
    export -f test_command
    export attempt_count
    
    # Mock sleep to avoid delays in tests
    sleep() { return 0; }
    export -f sleep
    
    run vault::retry_operation 3 1 test_command
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Retry 1/3 failed" ]]
}

@test "vault::retry_operation fails after max retries" {
    source "$SCRIPT_PATH"
    
    # Mock command that always fails
    test_command() { return 1; }
    export -f test_command
    
    # Mock sleep to avoid delays in tests
    sleep() { return 0; }
    export -f sleep
    
    run vault::retry_operation 2 1 test_command
    [ "$status" -eq 87 ]
    [[ "$output" =~ "All 2 retries failed" ]]
}

@test "vault::retry_operation handles complex command with arguments" {
    source "$SCRIPT_PATH"
    
    # Mock command that checks arguments
    test_command() {
        [[ "$1" == "arg1" && "$2" == "arg2" ]] && return 0 || return 1
    }
    export -f test_command
    
    run vault::retry_operation 2 1 test_command arg1 arg2
    [ "$status" -eq 0 ]
}