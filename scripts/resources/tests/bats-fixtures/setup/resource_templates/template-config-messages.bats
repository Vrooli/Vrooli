#!/usr/bin/env bats
# Tests for RESOURCE_NAME config/messages.sh localization functions
#
# Template Usage:
# 1. Copy this file to RESOURCE_NAME/config/messages.bats
# 2. Replace RESOURCE_NAME with your resource name (e.g., "ollama", "n8n")
# 3. Replace MESSAGE_KEYS with your actual message keys
# 4. Implement resource-specific message tests
# 5. Remove this header comment block

bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export RESOURCE_NAME_LANG="en"  # Default language
    export RESOURCE_NAME_MESSAGES_DIR="/tmp/RESOURCE_NAME-test/messages"
    
    # Get resource directory path
    RESOURCE_NAME_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    
    # Create test message directory
    mkdir -p "$RESOURCE_NAME_MESSAGES_DIR"
    
    # Load configuration and initialize messages
    source "${RESOURCE_NAME_DIR}/config/defaults.sh"
    source "${RESOURCE_NAME_DIR}/config/messages.sh"
    RESOURCE_NAME::export_config
}

teardown() {
    # Clean up test environment
    cleanup_mocks
    rm -rf "/tmp/RESOURCE_NAME-test"
}

# Test message initialization

@test "RESOURCE_NAME::messages::init should initialize message system" {
    run RESOURCE_NAME::messages::init
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::messages::init should set default language" {
    unset RESOURCE_NAME_LANG
    run RESOURCE_NAME::messages::init
    [ "$status" -eq 0 ]
    [ "$RESOURCE_NAME_LANG" = "en" ]
}

@test "RESOURCE_NAME::messages::init should respect language override" {
    export RESOURCE_NAME_LANG="es"
    run RESOURCE_NAME::messages::init
    [ "$status" -eq 0 ]
    [ "$RESOURCE_NAME_LANG" = "es" ]
}

# Test message retrieval

@test "RESOURCE_NAME::msg should return message for valid key" {
    # Mock message data
    declare -A RESOURCE_NAME_MESSAGES
    RESOURCE_NAME_MESSAGES["welcome"]="Welcome to RESOURCE_NAME"
    
    run RESOURCE_NAME::msg "welcome"
    [ "$status" -eq 0 ]
    [ "$output" = "Welcome to RESOURCE_NAME" ]
}

@test "RESOURCE_NAME::msg should return key for missing message" {
    run RESOURCE_NAME::msg "nonexistent_key"
    [ "$status" -eq 0 ]
    [ "$output" = "nonexistent_key" ]
}

@test "RESOURCE_NAME::msg should handle empty key" {
    run RESOURCE_NAME::msg ""
    [ "$status" -ne 0 ]
}

# Test message formatting

@test "RESOURCE_NAME::msg should support parameter substitution" {
    # Mock message with placeholder
    declare -A RESOURCE_NAME_MESSAGES
    RESOURCE_NAME_MESSAGES["user_greeting"]="Hello, %s!"
    
    run RESOURCE_NAME::msg "user_greeting" "John"
    [ "$status" -eq 0 ]
    [ "$output" = "Hello, John!" ]
}

@test "RESOURCE_NAME::msg should handle multiple parameters" {
    # Mock message with multiple placeholders
    declare -A RESOURCE_NAME_MESSAGES
    RESOURCE_NAME_MESSAGES["service_status"]="%s service is %s on port %d"
    
    run RESOURCE_NAME::msg "service_status" "RESOURCE_NAME" "running" "8080"
    [ "$status" -eq 0 ]
    [ "$output" = "RESOURCE_NAME service is running on port 8080" ]
}

@test "RESOURCE_NAME::msg should handle missing parameters gracefully" {
    # Mock message expecting parameters
    declare -A RESOURCE_NAME_MESSAGES
    RESOURCE_NAME_MESSAGES["needs_param"]="Value: %s"
    
    run RESOURCE_NAME::msg "needs_param"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Value:" ]]  # Should not crash
}

# Test language switching

@test "RESOURCE_NAME::messages::set_language should change active language" {
    run RESOURCE_NAME::messages::set_language "fr"
    [ "$status" -eq 0 ]
    [ "$RESOURCE_NAME_LANG" = "fr" ]
}

@test "RESOURCE_NAME::messages::set_language should reload messages" {
    # Create test message files
    mkdir -p "$RESOURCE_NAME_MESSAGES_DIR/en"
    mkdir -p "$RESOURCE_NAME_MESSAGES_DIR/es"
    echo 'RESOURCE_NAME_MESSAGES["hello"]="Hello"' > "$RESOURCE_NAME_MESSAGES_DIR/en/common.sh"
    echo 'RESOURCE_NAME_MESSAGES["hello"]="Hola"' > "$RESOURCE_NAME_MESSAGES_DIR/es/common.sh"
    
    # Load English
    RESOURCE_NAME::messages::set_language "en"
    run RESOURCE_NAME::msg "hello"
    [ "$output" = "Hello" ]
    
    # Switch to Spanish
    RESOURCE_NAME::messages::set_language "es"
    run RESOURCE_NAME::msg "hello"
    [ "$output" = "Hola" ]
}

@test "RESOURCE_NAME::messages::set_language should fallback to English for unsupported language" {
    run RESOURCE_NAME::messages::set_language "unsupported"
    [ "$status" -eq 0 ]
    [ "$RESOURCE_NAME_LANG" = "en" ]
}

# Test message loading

@test "RESOURCE_NAME::messages::load_language_pack should load message file" {
    # Create test message file
    mkdir -p "$RESOURCE_NAME_MESSAGES_DIR/en"
    cat > "$RESOURCE_NAME_MESSAGES_DIR/en/test.sh" << 'EOF'
RESOURCE_NAME_MESSAGES["test_msg"]="Test message"
RESOURCE_NAME_MESSAGES["another_msg"]="Another message"
EOF
    
    run RESOURCE_NAME::messages::load_language_pack "en" "test"
    [ "$status" -eq 0 ]
    
    # Verify messages were loaded
    run RESOURCE_NAME::msg "test_msg"
    [ "$output" = "Test message" ]
}

@test "RESOURCE_NAME::messages::load_language_pack should handle missing file" {
    run RESOURCE_NAME::messages::load_language_pack "en" "nonexistent"
    [ "$status" -ne 0 ]
}

@test "RESOURCE_NAME::messages::load_all should load all message files" {
    # Create multiple test message files
    mkdir -p "$RESOURCE_NAME_MESSAGES_DIR/en"
    echo 'RESOURCE_NAME_MESSAGES["msg1"]="Message 1"' > "$RESOURCE_NAME_MESSAGES_DIR/en/pack1.sh"
    echo 'RESOURCE_NAME_MESSAGES["msg2"]="Message 2"' > "$RESOURCE_NAME_MESSAGES_DIR/en/pack2.sh"
    
    run RESOURCE_NAME::messages::load_all "en"
    [ "$status" -eq 0 ]
    
    # Verify all messages were loaded
    run RESOURCE_NAME::msg "msg1"
    [ "$output" = "Message 1" ]
    run RESOURCE_NAME::msg "msg2"
    [ "$output" = "Message 2" ]
}

# Test message validation

@test "RESOURCE_NAME::messages::validate should check message completeness" {
    # Create incomplete message file
    mkdir -p "$RESOURCE_NAME_MESSAGES_DIR/es"
    cat > "$RESOURCE_NAME_MESSAGES_DIR/en/complete.sh" << 'EOF'
RESOURCE_NAME_MESSAGES["msg1"]="English message 1"
RESOURCE_NAME_MESSAGES["msg2"]="English message 2"
EOF
    cat > "$RESOURCE_NAME_MESSAGES_DIR/es/incomplete.sh" << 'EOF'
RESOURCE_NAME_MESSAGES["msg1"]="Spanish message 1"
# msg2 is missing
EOF
    
    run RESOURCE_NAME::messages::validate "es"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "missing" || "$output" =~ "incomplete" ]]
}

@test "RESOURCE_NAME::messages::validate should pass for complete translations" {
    # Create complete message files
    mkdir -p "$RESOURCE_NAME_MESSAGES_DIR/en" "$RESOURCE_NAME_MESSAGES_DIR/fr"
    cat > "$RESOURCE_NAME_MESSAGES_DIR/en/complete.sh" << 'EOF'
RESOURCE_NAME_MESSAGES["msg1"]="English message 1"
RESOURCE_NAME_MESSAGES["msg2"]="English message 2"
EOF
    cat > "$RESOURCE_NAME_MESSAGES_DIR/fr/complete.sh" << 'EOF'
RESOURCE_NAME_MESSAGES["msg1"]="French message 1"
RESOURCE_NAME_MESSAGES["msg2"]="French message 2"
EOF
    
    run RESOURCE_NAME::messages::validate "fr"
    [ "$status" -eq 0 ]
}

# Test standard message keys (customize based on your resource)

@test "RESOURCE_NAME messages should include standard keys" {
    RESOURCE_NAME::messages::init
    
    # Test that standard message keys exist (customize these)
    run RESOURCE_NAME::msg "service_starting"
    [ "$status" -eq 0 ]
    [[ "$output" != "service_starting" ]]  # Should not return the key itself
    
    run RESOURCE_NAME::msg "service_stopped"
    [ "$status" -eq 0 ]
    [[ "$output" != "service_stopped" ]]
    
    run RESOURCE_NAME::msg "installation_complete"
    [ "$status" -eq 0 ]
    [[ "$output" != "installation_complete" ]]
}

# Test message categories

@test "RESOURCE_NAME should organize messages by category" {
    # Test that messages are properly categorized (customize based on your structure)
    run RESOURCE_NAME::messages::list_categories
    [ "$status" -eq 0 ]
    [[ "$output" =~ "installation" ]]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "errors" ]]
}

@test "RESOURCE_NAME should load category-specific messages" {
    run RESOURCE_NAME::messages::load_category "installation"
    [ "$status" -eq 0 ]
    
    # Verify installation messages are available
    run RESOURCE_NAME::msg "installation_starting"
    [[ "$output" != "installation_starting" ]]
}

# Test error messages

@test "RESOURCE_NAME error messages should be descriptive" {
    # Load error messages
    RESOURCE_NAME::messages::load_category "errors"
    
    run RESOURCE_NAME::msg "error_connection_failed"
    [ "$status" -eq 0 ]
    [[ ${#output} -gt 10 ]]  # Should be descriptive, not just a key
    [[ "$output" =~ "connection" ]]
}

@test "RESOURCE_NAME error messages should support error codes" {
    run RESOURCE_NAME::msg "error_code" "E001" "Connection timeout"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "E001" ]]
    [[ "$output" =~ "Connection timeout" ]]
}

# Test message caching (if implemented)

@test "RESOURCE_NAME::messages should cache loaded messages" {
    # Load messages
    RESOURCE_NAME::messages::load_all "en"
    
    # Modify message file
    echo 'RESOURCE_NAME_MESSAGES["cached_test"]="Modified"' >> "$RESOURCE_NAME_MESSAGES_DIR/en/test.sh"
    
    # Should still return cached version
    run RESOURCE_NAME::msg "cached_test"
    [ "$output" != "Modified" ]
}

@test "RESOURCE_NAME::messages::clear_cache should reload messages" {
    # Load and cache messages
    RESOURCE_NAME::messages::load_all "en"
    
    # Clear cache and reload
    run RESOURCE_NAME::messages::clear_cache
    [ "$status" -eq 0 ]
    
    # Should reload fresh messages
    RESOURCE_NAME::messages::load_all "en"
}

# Test pluralization (if supported)

@test "RESOURCE_NAME::msg should handle pluralization" {
    # Mock pluralized messages
    declare -A RESOURCE_NAME_MESSAGES
    RESOURCE_NAME_MESSAGES["item_count_one"]="%d item"
    RESOURCE_NAME_MESSAGES["item_count_other"]="%d items"
    
    run RESOURCE_NAME::msg "item_count" 1
    [ "$output" = "1 item" ]
    
    run RESOURCE_NAME::msg "item_count" 5
    [ "$output" = "5 items" ]
}

# Add resource-specific message tests here:

# Example templates for different resource types:

# For AI resources:
# @test "RESOURCE_NAME should have model management messages" { ... }
# @test "RESOURCE_NAME should have training/inference messages" { ... }

# For automation resources:
# @test "RESOURCE_NAME should have workflow execution messages" { ... }
# @test "RESOURCE_NAME should have integration messages" { ... }

# For storage resources:
# @test "RESOURCE_NAME should have backup/restore messages" { ... }
# @test "RESOURCE_NAME should have storage quota messages" { ... }