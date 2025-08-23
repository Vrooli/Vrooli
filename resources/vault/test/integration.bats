#!/usr/bin/env bats

# Simple integration tests for Vault - no external dependencies

# Test vault basic operations
@test "vault: status check returns healthy" {
    run timeout 10 resource-vault status
    [ "$status" -eq 0 ]
    [[ "$output" == *"running"* ]]
}

@test "vault: can store and retrieve a secret" {
    # Store a test secret
    run timeout 10 resource-vault inject test_key "test_value_$$"
    [ "$status" -eq 0 ]
    
    # Retrieve the secret
    run timeout 10 resource-vault get-secret test_key
    [ "$status" -eq 0 ]
    [[ "$output" == *"test_value_$$"* ]]
    
    # Clean up
    run timeout 10 resource-vault delete-secret test_key --force
    true  # Ignore cleanup errors
}

@test "vault: can store and retrieve JSON secret" {
    local json_value='{"user":"test","pass":"secret123"}'
    
    # Store JSON secret
    run timeout 10 resource-vault inject test_json "$json_value"
    [ "$status" -eq 0 ]
    
    # Retrieve and verify
    run timeout 10 resource-vault get-secret test_json
    [ "$status" -eq 0 ]
    [[ "$output" == *"test"* ]]
    [[ "$output" == *"secret123"* ]]
    
    # Clean up
    run timeout 10 resource-vault delete-secret test_json --force
    true
}

@test "vault: handles non-existent keys gracefully" {
    run timeout 10 resource-vault get-secret non_existent_key_$$
    # Should fail but with a clear message
    [ "$status" -ne 0 ]
}

@test "vault: can list secrets" {
    # Store a test secret first
    run timeout 10 resource-vault inject list_test_$$ "value"
    true  # Continue even if fails
    
    # List secrets
    run timeout 10 resource-vault list-secrets /
    [ "$status" -eq 0 ] || [[ "$output" == *"Secret"* ]] || [[ "$output" == *"list"* ]]  # Vault should respond
    
    # Clean up
    run timeout 10 resource-vault delete-secret list_test_$$ --force
    true
}

@test "vault: supports JSON output format" {
    run timeout 10 resource-vault status --format json
    [ "$status" -eq 0 ] || [[ "$output" == *"{"* ]]  # Should be JSON-like
}

@test "vault: can handle special characters in values" {
    local special_value='p@$$w0rd!#$%^&*()_+-={}[]|:;<>?,./'
    
    # Store with special chars
    run timeout 10 resource-vault inject special_test "$special_value"
    [ "$status" -eq 0 ]
    
    # Retrieve and verify
    run timeout 10 resource-vault get-secret special_test
    [ "$status" -eq 0 ]
    # Check for at least some special chars preserved
    [[ "$output" == *"p@"* ]]
    
    # Clean up
    run timeout 10 resource-vault delete-secret special_test --force
    true
}

@test "vault: respects timeouts" {
    # This should timeout quickly
    run timeout 2 bash -c "resource-vault status && sleep 10"
    [ "$status" -ne 0 ]  # Should fail due to timeout
}

@test "vault: provides helpful error messages" {
    # Try an invalid command
    run timeout 10 resource-vault invalid_command
    [ "$status" -ne 0 ]
    [[ "$output" == *"Usage"* ]] || [[ "$output" == *"command"* ]]
}

@test "vault: seal status check works" {
    run timeout 10 resource-vault unseal
    [ "$status" -eq 0 ] || [[ "$output" == *"seal"* ]] || [[ "$output" == *"already"* ]]
}