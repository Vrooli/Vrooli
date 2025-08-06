#!/usr/bin/env bats
# Vault Mock Test Suite
#
# Comprehensive tests for the Vault mock implementation
# Tests all Vault commands, state management, error injection,
# and BATS compatibility features

# Test environment setup
setup() {
    # Set up test directory
    export TEST_DIR="$BATS_TEST_TMPDIR/vault-mock-test"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Set up mock directory
    export MOCK_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
    
    # Configure Vault mock state directory
    export VAULT_MOCK_STATE_DIR="$TEST_DIR/vault-state"
    mkdir -p "$VAULT_MOCK_STATE_DIR"
    
    # Source the Vault mock
    source "$MOCK_DIR/vault.sh"
    
    # Reset Vault mock to clean state
    mock::vault::reset
}

teardown() {
    # Clean up test directory
    rm -rf "$TEST_DIR"
}

# Helper functions for assertions
assert_success() {
    if [[ "$status" -ne 0 ]]; then
        echo "Expected success but got status $status" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_failure() {
    if [[ "$status" -eq 0 ]]; then
        echo "Expected failure but got success" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_output() {
    local expected="$1"
    if [[ "$1" == "--partial" ]]; then
        expected="$2"
        if [[ ! "$output" =~ "$expected" ]]; then
            echo "Expected output to contain: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    elif [[ "$1" == "--regexp" ]]; then
        expected="$2"
        if [[ ! "$output" =~ $expected ]]; then
            echo "Expected output to match: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    else
        if [[ "$output" != "$expected" ]]; then
            echo "Expected: $expected" >&2
            echo "Actual: $output" >&2
            return 1
        fi
    fi
}

refute_output() {
    local pattern="$2"
    if [[ "$1" == "--partial" ]] || [[ "$1" == "--regexp" ]]; then
        if [[ "$output" =~ "$pattern" ]]; then
            echo "Expected output NOT to contain: $pattern" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    fi
}

assert_line() {
    local index expected
    if [[ "$1" == "--index" ]]; then
        index="$2"
        expected="$3"
        local lines=()
        while IFS= read -r line; do
            lines+=("$line")
        done <<< "$output"
        
        if [[ "${lines[$index]}" != "$expected" ]]; then
            echo "Line $index mismatch" >&2
            echo "Expected: $expected" >&2
            echo "Actual: ${lines[$index]}" >&2
            return 1
        fi
    fi
}

# Basic connectivity and version tests
@test "vault: version command" {
    run vault version
    assert_success
    assert_output --partial "Vault v1.15.4"
    assert_output --partial "mock implementation"
}

@test "vault: help command shows usage" {
    run vault help
    assert_success
    assert_output --partial "Usage: vault <command> [args]"
    assert_output --partial "Common commands:"
    assert_output --partial "read"
    assert_output --partial "write"
}

@test "vault: status command shows unsealed vault" {
    run vault status
    assert_success
    assert_output --partial "Sealed          false"
    assert_output --partial "Initialized     true"
    assert_output --partial "Version         1.15.4"
}

@test "vault: status command fails when sealed" {
    mock::vault::set_sealed "true"
    
    run vault status
    [[ "$status" -eq 2 ]]
    assert_output --partial "Sealed          true"
}

@test "vault: status command fails when not initialized" {
    mock::vault::set_initialized "false"
    
    run vault status
    [[ "$status" -eq 2 ]]
    assert_output "Vault is not yet initialized"
}

# Error injection tests
@test "vault: error injection - sealed vault" {
    mock::vault::set_sealed "true"
    
    run vault kv get test
    [[ "$status" -eq 2 ]]
    assert_output "Error: Vault is sealed"
}

@test "vault: error injection - connection failed" {
    mock::vault::set_error "connection_failed"
    
    run vault status
    assert_failure
    assert_output "Error: connection failed: connection refused"
}

@test "vault: error injection - token invalid" {
    mock::vault::set_error "token_invalid"
    
    run vault kv get test
    [[ "$status" -eq 2 ]]
    assert_output "Error: permission denied"
}

@test "vault: error injection - server error" {
    mock::vault::set_error "server_error"
    
    run vault status
    [[ "$status" -eq 2 ]]
    assert_output "Error: internal server error"
}

# KV v2 operations tests
@test "vault: kv get existing secret" {
    run vault kv get test
    assert_success
    assert_output --partial "====== Metadata ======"
    assert_output --partial "created_time"
    assert_output --partial "version"
    assert_output --partial "====== Data ======"
    assert_output --partial "username"
    assert_output --partial "testuser"
    assert_output --partial "password"
    assert_output --partial "testpass123"
}

@test "vault: kv get non-existent secret" {
    run vault kv get nonexistent
    assert_failure
    assert_output "No value found at nonexistent"
}

@test "vault: kv put new secret" {
    run vault kv put myapp username=admin password=secret123 api_key=abc123
    assert_success
    assert_output "Success! Data written to: secret/data/myapp"
    
    # Verify it was stored
    run vault kv get myapp
    assert_success
    assert_output --partial "username"
    assert_output --partial "admin"
    assert_output --partial "password"
    assert_output --partial "secret123"
}

@test "vault: kv put requires key=value pairs" {
    run vault kv put myapp badformat
    assert_failure
    assert_output "Error: must provide data as key=value pairs"
}

@test "vault: kv list secrets" {
    vault kv put app1 key=value1
    vault kv put app2 key=value2
    
    run vault kv list
    assert_success
    assert_output --partial "Keys"
    assert_output --partial "----"
    assert_output --partial "test"
    assert_output --partial "app1"
    assert_output --partial "app2"
}

@test "vault: kv delete secret" {
    vault kv put temp key=value
    
    run vault kv delete temp
    assert_success
    assert_output "Success! Data deleted (if it existed) at: secret/data/temp"
    
    # Verify it was deleted
    run vault kv get temp
    assert_failure
}

@test "vault: kv metadata get" {
    run vault kv metadata get test
    assert_success
    assert_output --partial "version"
    assert_output --partial "created_time"
}

@test "vault: kv metadata get non-existent" {
    run vault kv metadata get nonexistent
    assert_failure
    assert_output "No metadata found at nonexistent"
}

# Generic read/write operations
@test "vault: read secret using generic command" {
    run vault read secret/data/test
    assert_success
    assert_output --partial "username"
    assert_output --partial "testuser"
}

@test "vault: write secret using generic command" {
    run vault write secret/data/myservice user=service pass=service123
    assert_success
    assert_output "Success! Data written to: secret/data/myservice"
    
    run vault read secret/data/myservice
    assert_success
    assert_output --partial "user"
    assert_output --partial "service"
}

@test "vault: write generic secret (non-KV)" {
    run vault write generic/config timeout=30 retries=3
    assert_success
    assert_output "Success! Data written to: generic/config"
    
    run vault read generic/config
    assert_success
    assert_output --partial "timeout"
    assert_output --partial "30"
}

@test "vault: list all secrets" {
    vault write generic/app1 key=value1
    vault write generic/app2 key=value2
    
    run vault list
    assert_success
    assert_output --partial "Keys"
    assert_output --partial "secret/data/test"
    assert_output --partial "generic/app1"
    assert_output --partial "generic/app2"
}

@test "vault: delete generic secret" {
    vault write generic/temp key=value
    
    run vault delete generic/temp
    assert_success
    assert_output "Success! Deleted generic/temp"
}

# Auth method operations
@test "vault: auth list shows default methods" {
    run vault auth list
    assert_success
    assert_output --partial "Path         Type        Accessor"
    assert_output --partial "token/"
    assert_output --partial "userpass/"
}

@test "vault: auth enable new method" {
    run vault auth enable ldap
    assert_success
    assert_output "Success! Enabled ldap auth method at: ldap/"
    
    run vault auth list
    assert_success
    assert_output --partial "ldap/"
}

@test "vault: auth enable with custom path" {
    run vault auth enable github github-corp/
    assert_success
    assert_output "Success! Enabled github auth method at: github-corp/"
}

@test "vault: auth disable method" {
    vault auth enable ldap
    
    run vault auth disable ldap
    assert_success
    assert_output "Success! Disabled auth method at: ldap/"
    
    run vault auth list
    assert_success
    refute_output --partial "ldap/"
}

@test "vault: auth enable requires method type" {
    run vault auth enable
    assert_failure
    assert_output "Error: auth method type is required"
}

@test "vault: auth disable requires path" {
    run vault auth disable
    assert_failure
    assert_output "Error: path is required"
}

# Policy operations
@test "vault: policy list shows default policies" {
    run vault policy list
    assert_success
    assert_output --partial "Policies"
    assert_output --partial "root"
    assert_output --partial "default"
}

@test "vault: policy read existing policy" {
    run vault policy read root
    assert_success
    assert_output --partial 'path "*"'
    assert_output --partial "capabilities"
}

@test "vault: policy read non-existent policy" {
    run vault policy read nonexistent
    assert_failure
    assert_output "No policy named: nonexistent"
}

@test "vault: policy write new policy" {
    run vault policy write myapp 'path "secret/data/myapp/*" { capabilities = ["read"] }'
    assert_success
    assert_output "Success! Uploaded policy: myapp"
    
    run vault policy read myapp
    assert_success
    assert_output --partial "secret/data/myapp"
    assert_output --partial "read"
}

@test "vault: policy delete" {
    vault policy write temp 'path "temp/*" { capabilities = ["read"] }'
    
    run vault policy delete temp
    assert_success
    assert_output "Success! Deleted policy: temp"
    
    run vault policy read temp
    assert_failure
}

@test "vault: policy write requires name" {
    run vault policy write
    assert_failure
    assert_output "Error: policy name is required"
}

# Token operations
@test "vault: token create" {
    run vault token create
    assert_success
    assert_output --partial "Key                  Value"
    assert_output --partial "token"
    assert_output --partial "token_accessor"
    assert_output --partial "token_duration"
    assert_output --partial "token_policies"
}

@test "vault: token create with custom policies" {
    run vault token create myapp
    assert_success
    assert_output --partial "token_policies       [\"myapp\"]"
}

@test "vault: token lookup self (root token)" {
    run vault token lookup
    assert_success
    assert_output --partial "Key                 Value"
    assert_output --partial "accessor"
    assert_output --partial "policies"
    assert_output --partial "root"
    assert_output --partial "type"
}

@test "vault: token lookup specific token" {
    # Create a token first
    local token_output
    token_output=$(vault token create 2>/dev/null)
    local token=$(echo "$token_output" | grep "^token" | head -1 | awk '{print $2}')
    
    run vault token lookup "$token"
    assert_success
    assert_output --partial "accessor"
    assert_output --partial "policies"
}

@test "vault: token lookup non-existent token" {
    run vault token lookup nonexistent-token
    assert_failure
    assert_output "Error: token not found"
}

@test "vault: token revoke" {
    # Create a token first
    local token_output
    token_output=$(vault token create 2>/dev/null)
    local token=$(echo "$token_output" | grep "^token" | awk '{print $2}')
    
    run vault token revoke "$token"
    assert_success
    assert_output "Success! Revoked token"
    
    # Verify it's gone
    run vault token lookup "$token"
    assert_failure
}

@test "vault: token revoke requires token" {
    run vault token revoke
    assert_failure
    assert_output "Error: token is required"
}

# Secret engines operations
@test "vault: secrets list shows default engines" {
    run vault secrets list
    assert_success
    assert_output --partial "Path         Type         Accessor"
    assert_output --partial "secret/"
    assert_output --partial "kv"
    assert_output --partial "database/"
}

@test "vault: secrets enable new engine" {
    run vault secrets enable pki
    assert_success
    assert_output "Success! Enabled the pki secrets engine at: pki/"
    
    run vault secrets list
    assert_success
    assert_output --partial "pki/"
}

@test "vault: secrets enable with custom path" {
    run vault secrets enable database mydb/
    assert_success
    assert_output "Success! Enabled the database secrets engine at: mydb/"
}

@test "vault: secrets disable engine" {
    vault secrets enable pki
    
    run vault secrets disable pki
    assert_success
    assert_output "Success! Disabled the secrets engine at: pki/"
    
    run vault secrets list
    assert_success
    refute_output --partial "pki/"
}

@test "vault: secrets enable requires engine type" {
    run vault secrets enable
    assert_failure
    assert_output "Error: engine type is required"
}

# Operator operations
@test "vault: operator init" {
    mock::vault::set_initialized "false"
    
    run vault operator init
    assert_success
    assert_output --partial "Unseal Key 1:"
    assert_output --partial "Initial Root Token:"
    assert_output --partial "Vault initialized"
}

@test "vault: operator unseal" {
    mock::vault::set_sealed "true"
    
    run vault operator unseal mock-key
    assert_success
    assert_output --partial "Sealed             false"
    assert_output --partial "Initialized        true"
}

@test "vault: operator unseal requires initialization" {
    mock::vault::set_initialized "false"
    
    run vault operator unseal mock-key
    assert_failure
    assert_output "Error: Vault is not initialized"
}

@test "vault: operator seal" {
    run vault operator seal
    assert_success
    assert_output "Success! Vault is sealed."
}

# Server operations
@test "vault: server dev mode" {
    run vault server dev
    assert_success
    assert_output --partial "==> Vault server configuration:"
    assert_output --partial "Api Address: http://127.0.0.1:8200"
    assert_output --partial "Development mode should NOT be used in production"
    assert_output --partial "Root Token: root-token"
}

# State persistence tests
@test "vault: state persistence across subshells" {
    # Set data in parent shell
    vault kv put persistent key=persistvalue
    mock::vault::save_state
    
    # Verify in subshell
    output=$(
        source "$MOCK_DIR/vault.sh"
        vault kv get persistent 2>/dev/null | grep persistvalue || echo "NOT_FOUND"
    )
    [[ "$output" =~ "persistvalue" ]]
}

@test "vault: state file creation and loading" {
    vault kv put statetest key=statevalue
    vault policy write statepolicy 'path "test/*" { capabilities = ["read"] }'
    mock::vault::save_state
    
    # Check state file exists
    [[ -f "$VAULT_MOCK_STATE_DIR/vault-state.sh" ]]
    
    # Reset without saving state, then reload
    mock::vault::reset false
    mock::vault::load_state
    
    run vault kv get statetest
    assert_success
    assert_output --partial "statevalue"
    
    run vault policy read statepolicy
    assert_success
    assert_output --partial 'path "test/*"'
}

# Test helper functions
@test "mock::vault::reset clears all data" {
    vault kv put temp key=value
    vault policy write temp 'path "*" { capabilities = ["read"] }'
    vault auth enable ldap
    mock::vault::set_error "connection_failed"
    
    mock::vault::reset
    
    # Data should be cleared
    run vault kv get temp
    assert_failure
    
    # Policies should be reset to defaults
    run vault policy read temp
    assert_failure
    
    # Auth methods should be reset to defaults
    run vault auth list
    refute_output --partial "ldap/"
    
    # Error mode should be cleared
    run vault status
    assert_success
}

@test "mock::vault::assert_secret_exists" {
    vault kv put mytest key=value
    
    run mock::vault::assert_secret_exists "mytest"
    assert_success
    
    run mock::vault::assert_secret_exists "nonexistent"
    assert_failure
    assert_output --partial "Secret 'nonexistent' does not exist"
}

@test "mock::vault::assert_secret_value" {
    vault kv put mytest username=admin password=secret
    
    run mock::vault::assert_secret_value "mytest" "username" "admin"
    assert_success
    
    run mock::vault::assert_secret_value "mytest" "username" "wrong"
    assert_failure
    assert_output --partial "value mismatch"
    assert_output --partial "Expected: 'wrong'"
    assert_output --partial "Actual: 'admin'"
}

@test "mock::vault::assert_policy_exists" {
    vault policy write mypolicy 'path "*" { capabilities = ["read"] }'
    
    run mock::vault::assert_policy_exists "mypolicy"
    assert_success
    
    run mock::vault::assert_policy_exists "nonexistent"
    assert_failure
    assert_output --partial "Policy 'nonexistent' does not exist"
}

@test "mock::vault::assert_auth_method_enabled" {
    vault auth enable ldap
    
    run mock::vault::assert_auth_method_enabled "ldap/"
    assert_success
    
    run mock::vault::assert_auth_method_enabled "ldap"
    assert_success
    
    run mock::vault::assert_auth_method_enabled "nonexistent"
    assert_failure
    assert_output --partial "Auth method at 'nonexistent/' is not enabled"
}

@test "mock::vault::assert_secret_engine_enabled" {
    vault secrets enable pki
    
    run mock::vault::assert_secret_engine_enabled "pki/"
    assert_success
    
    run mock::vault::assert_secret_engine_enabled "pki"
    assert_success
    
    run mock::vault::assert_secret_engine_enabled "nonexistent"
    assert_failure
    assert_output --partial "Secret engine at 'nonexistent/' is not enabled"
}

@test "mock::vault::assert_sealed" {
    run mock::vault::assert_sealed "false"
    assert_success
    
    mock::vault::set_sealed "true"
    
    run mock::vault::assert_sealed "true"
    assert_success
    
    run mock::vault::assert_sealed "false"
    assert_failure
    assert_output --partial "sealed state mismatch"
}

@test "mock::vault::assert_initialized" {
    run mock::vault::assert_initialized "true"
    assert_success
    
    mock::vault::set_initialized "false"
    
    run mock::vault::assert_initialized "false"
    assert_success
    
    run mock::vault::assert_initialized "true"
    assert_failure
    assert_output --partial "initialized state mismatch"
}

@test "mock::vault::dump_state shows current state" {
    vault kv put myapp key=value
    vault policy write mypolicy 'path "*" { capabilities = ["read"] }'
    vault auth enable ldap
    
    run mock::vault::dump_state
    assert_success
    assert_output --partial "=== Vault Mock State ==="
    assert_output --partial "Configuration:"
    assert_output --partial "version: 1.15.4"
    assert_output --partial "Secrets:"
    assert_output --partial "secret/data/myapp"
    assert_output --partial "Policies:"
    assert_output --partial "mypolicy"
    assert_output --partial "Auth Methods:"
    assert_output --partial "ldap/"
}

# Complex scenario tests
@test "vault: complete KV workflow" {
    # Create multiple secrets
    vault kv put myapp/dev username=dev_user password=dev_pass
    vault kv put myapp/prod username=prod_user password=prod_pass
    vault kv put myapp/config api_key=abc123 timeout=30
    
    # List all secrets
    run vault kv list
    assert_success
    assert_output --partial "myapp/dev"
    assert_output --partial "myapp/prod"
    assert_output --partial "myapp/config"
    
    # Read secrets
    run vault kv get myapp/dev
    assert_success
    assert_output --partial "dev_user"
    assert_output --partial "dev_pass"
    
    # Update a secret
    vault kv put myapp/dev username=new_dev_user password=new_dev_pass
    
    run vault kv get myapp/dev
    assert_success
    assert_output --partial "new_dev_user"
    assert_output --partial "new_dev_pass"
    
    # Delete a secret
    vault kv delete myapp/config
    
    run vault kv get myapp/config
    assert_failure
}

@test "vault: auth and policy management workflow" {
    # Create policies
    vault policy write readonly 'path "secret/data/readonly/*" { capabilities = ["read", "list"] }'
    vault policy write readwrite 'path "secret/data/readwrite/*" { capabilities = ["create", "read", "update", "delete", "list"] }'
    
    # Enable auth methods
    vault auth enable userpass
    vault auth enable ldap ldap-corp/
    
    # Create tokens with different policies
    local readonly_token_output
    readonly_token_output=$(vault token create readonly 2>/dev/null)
    local readonly_token=$(echo "$readonly_token_output" | grep "^token" | head -1 | awk '{print $2}')
    
    local readwrite_token_output
    readwrite_token_output=$(vault token create readwrite 2>/dev/null)
    local readwrite_token=$(echo "$readwrite_token_output" | grep "^token" | head -1 | awk '{print $2}')
    
    # Verify tokens have correct policies
    run vault token lookup "$readonly_token"
    assert_success
    assert_output --partial "readonly"
    
    run vault token lookup "$readwrite_token"
    assert_success
    assert_output --partial "readwrite"
    
    # Clean up
    vault token revoke "$readonly_token"
    vault token revoke "$readwrite_token"
    vault auth disable userpass
    vault auth disable ldap-corp
    vault policy delete readonly
    vault policy delete readwrite
}

@test "vault: secrets engine management workflow" {
    # Enable different types of secret engines
    vault secrets enable database
    vault secrets enable pki certificates/
    vault secrets enable transit encryption/
    
    # Verify they're enabled
    run vault secrets list
    assert_success
    assert_output --partial "database/"
    assert_output --partial "certificates/"
    assert_output --partial "encryption/"
    
    # Use KV engine with different paths
    vault kv put app1/config db_host=localhost db_port=5432
    vault kv put app2/config db_host=remote db_port=3306
    
    run vault kv get app1/config
    assert_success
    assert_output --partial "localhost"
    assert_output --partial "5432"
    
    # Disable engines
    vault secrets disable database
    vault secrets disable certificates
    vault secrets disable encryption
    
    run vault secrets list
    assert_success
    refute_output --partial "database/"
    refute_output --partial "certificates/"
    refute_output --partial "encryption/"
}

@test "vault: initialization and unsealing workflow" {
    # Start with uninitialized vault
    mock::vault::set_initialized "false"
    mock::vault::set_sealed "true"
    
    run vault status
    [[ "$status" -eq 2 ]]
    assert_output "Vault is not yet initialized"
    
    # Initialize
    run vault operator init
    assert_success
    assert_output --partial "Unseal Key 1:"
    assert_output --partial "Initial Root Token:"
    
    # Should now be initialized and unsealed
    run vault status
    assert_success
    assert_output --partial "Initialized     true"
    assert_output --partial "Sealed          false"
    
    # Seal it
    vault operator seal
    
    run vault status
    [[ "$status" -eq 2 ]]
    assert_output --partial "Sealed          true"
    
    # Unseal it
    vault operator unseal mock-key
    
    run vault status
    assert_success
    assert_output --partial "Sealed          false"
}

# Error handling tests
@test "vault: commands with missing arguments" {
    run vault kv get
    assert_failure
    assert_output "Error: path is required"
    
    run vault kv put
    assert_failure
    assert_output "Error: path is required"
    
    run vault read
    assert_failure
    assert_output "Error: path is required"
    
    run vault write
    assert_failure
    assert_output "Error: path is required"
}

@test "vault: unknown command" {
    run vault unknowncommand arg1 arg2
    assert_failure
    assert_output --partial "Error: unknown command: unknowncommand"
    assert_output --partial "Run 'vault -help' for usage."
}

@test "vault: kv subcommand help" {
    run vault kv
    assert_failure
    assert_output --partial "Usage: vault kv <subcommand>"
    assert_output --partial "Subcommands:"
    assert_output --partial "get"
    assert_output --partial "put"
    assert_output --partial "list"
}

# Edge cases
@test "vault: empty secret values" {
    run vault kv put emptytest key=""
    assert_success
    
    run vault kv get emptytest
    assert_success
    assert_output --partial "key"
}

@test "vault: secrets with special characters" {
    run vault kv put "app/with-dashes" "key-with-dashes=value-with-dashes"
    assert_success
    
    run vault kv get "app/with-dashes"
    assert_success
    assert_output --partial "key-with-dashes"
    assert_output --partial "value-with-dashes"
    
    run vault kv put "app/with spaces" "key with spaces=value with spaces"
    assert_success
    
    run vault kv get "app/with spaces"
    assert_success
    assert_output --partial "key with spaces"
    assert_output --partial "value with spaces"
}

@test "vault: policy names with special characters" {
    run vault policy write "policy-with-dashes" 'path "*" { capabilities = ["read"] }'
    assert_success
    
    run vault policy read "policy-with-dashes"
    assert_success
    assert_output --partial 'path "*"'
}

@test "vault: multiple key=value pairs in single command" {
    run vault kv put multikey user=admin pass=secret api_key=abc123 timeout=30 enabled=true
    assert_success
    
    run vault kv get multikey
    assert_success
    assert_output --partial "user"
    assert_output --partial "admin"
    assert_output --partial "pass"
    assert_output --partial "secret"
    assert_output --partial "api_key"
    assert_output --partial "abc123"
    assert_output --partial "timeout"
    assert_output --partial "30"
    assert_output --partial "enabled"
    assert_output --partial "true"
}

# JSON handling fallback tests (when jq not available)
@test "vault: fallback behavior when jq not available" {
    # Temporarily hide jq if it exists
    local jq_path=""
    if command -v jq >/dev/null 2>&1; then
        jq_path=$(which jq)
        # Create a temporary directory that doesn't have jq
        export PATH="/usr/bin:/bin"
    fi
    
    # This should still work with fallback parsing
    run vault kv get test
    assert_success
    assert_output --partial "username"
    assert_output --partial "password"
    
    # Restore PATH if we modified it
    if [[ -n "$jq_path" ]]; then
        export PATH="$(dirname "$jq_path"):$PATH"
    fi
}

# Performance and stress tests
@test "vault: handle many secrets efficiently" {
    # Create many secrets
    for i in {1..20}; do
        vault kv put "app$i/config" "key$i=value$i" >/dev/null 2>&1
    done
    
    # List should show all
    run vault kv list
    assert_success
    assert_output --partial "app1/config"
    assert_output --partial "app10/config"
    assert_output --partial "app20/config"
    
    # Individual gets should work
    run vault kv get "app15/config"
    assert_success
    assert_output --partial "key15"
    assert_output --partial "value15"
}

@test "vault: concurrent state modifications" {
    # Simulate concurrent access by rapid sequential changes
    # Note: True concurrent access has race conditions with file-based state
    vault kv put concurrent1 key=value1
    vault kv put concurrent2 key=value2
    vault kv put concurrent3 key=value3
    
    # All should be accessible
    run vault kv get concurrent1
    assert_success
    assert_output --partial "value1"
    
    run vault kv get concurrent2
    assert_success
    assert_output --partial "value2"
    
    run vault kv get concurrent3
    assert_success
    assert_output --partial "value3"
}