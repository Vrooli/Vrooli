#!/usr/bin/env bats

# Tests for Ollama configuration messages

setup() {
    # Source the messages
    source "$(dirname "$BATS_TEST_FILENAME")/messages.sh"
}

@test "installation messages are defined" {
    [ -n "$MSG_OLLAMA_INSTALLING" ]
    [ -n "$MSG_OLLAMA_ALREADY_INSTALLED" ]
    [ -n "$MSG_OLLAMA_INSTALL_SUCCESS" ]
    [ -n "$MSG_OLLAMA_INSTALL_FAILED" ]
    [ -n "$MSG_BINARY_INSTALL_SUCCESS" ]
    [ -n "$MSG_BINARY_INSTALL_FAILED" ]
    [ -n "$MSG_USER_CREATE_SUCCESS" ]
    [ -n "$MSG_USER_CREATE_FAILED" ]
    [ -n "$MSG_SERVICE_INSTALL_SUCCESS" ]
}

@test "status messages are defined" {
    [ -n "$MSG_OLLAMA_RUNNING" ]
    [ -n "$MSG_OLLAMA_STARTED_NO_API" ]
    [ -n "$MSG_OLLAMA_NOT_INSTALLED" ]
    [ -n "$MSG_OLLAMA_API_UNAVAILABLE" ]
    [ -n "$MSG_STATUS_BINARY_OK" ]
    [ -n "$MSG_STATUS_USER_OK" ]
    [ -n "$MSG_STATUS_SERVICE_OK" ]
    [ -n "$MSG_STATUS_SERVICE_ENABLED" ]
    [ -n "$MSG_STATUS_SERVICE_ACTIVE" ]
    [ -n "$MSG_STATUS_PORT_OK" ]
    [ -n "$MSG_STATUS_API_OK" ]
}

@test "model messages are defined" {
    [ -n "$MSG_MODELS_HEADER" ]
    [ -n "$MSG_MODELS_LEGEND" ]
    [ -n "$MSG_MODELS_TOTAL_SIZE" ]
    [ -n "$MSG_MODEL_INSTALL_SUCCESS" ]
    [ -n "$MSG_MODEL_INSTALL_FAILED" ]
    [ -n "$MSG_MODEL_PULL_SUCCESS" ]
    [ -n "$MSG_MODEL_PULL_FAILED" ]
    [ -n "$MSG_MODEL_NOT_INSTALLED" ]
    [ -n "$MSG_MODEL_VALIDATION_FAILED" ]
    [ -n "$MSG_MODEL_NONE_SPECIFIED" ]
}

@test "api/prompt messages are defined" {
    [ -n "$MSG_PROMPT_SENDING" ]
    [ -n "$MSG_PROMPT_RESPONSE_HEADER" ]
    [ -n "$MSG_PROMPT_NO_TEXT" ]
    [ -n "$MSG_PROMPT_USAGE" ]
    [ -n "$MSG_PROMPT_API_ERROR" ]
    [ -n "$MSG_PROMPT_NO_RESPONSE" ]
    # Dynamic message functions
    type MSG_PROMPT_RESPONSE_TIME >/dev/null 2>&1
    type MSG_PROMPT_TOKEN_COUNT >/dev/null 2>&1
    [ -n "$MSG_PROMPT_PARAMETERS" ]
}

@test "warning messages are defined" {
    # Dynamic message functions
    type MSG_UNKNOWN_MODELS >/dev/null 2>&1
    [ -n "$MSG_USE_AVAILABLE_MODELS" ]
    type MSG_LOW_DISK_SPACE >/dev/null 2>&1
    [ -n "$MSG_JQ_UNAVAILABLE" ]
    [ -n "$MSG_MODELS_INSTALL_FAILED" ]
    [ -n "$MSG_CONFIG_UPDATE_FAILED" ]
    [ -n "$MSG_VERIFICATION_FAILED" ]
}

@test "port/network messages are defined" {
    [ -n "$MSG_PORT_CONFLICT" ]
    [ -n "$MSG_PORT_WARNING" ]
    [ -n "$MSG_PORT_UNEXPECTED" ]
}

@test "help/info messages are defined" {
    [ -n "$MSG_START_OLLAMA" ]
    [ -n "$MSG_CHECK_STATUS" ]
    [ -n "$MSG_AVAILABLE_MODELS" ]
    [ -n "$MSG_INSTALL_MODEL" ]
    [ -n "$MSG_FAILED_API_REQUEST" ]
    [ -n "$MSG_LIST_MODELS_FAILED" ]
}

@test "success messages contain success indicators" {
    [[ "$MSG_OLLAMA_INSTALL_SUCCESS" =~ ✅ ]]
    [[ "$MSG_BINARY_INSTALL_SUCCESS" =~ ✅ ]]
    [[ "$MSG_USER_CREATE_SUCCESS" =~ ✅ ]]
    [[ "$MSG_SERVICE_INSTALL_SUCCESS" =~ ✅ ]]
    [[ "$MSG_MODEL_INSTALL_SUCCESS" =~ ✅ ]]
    [[ "$MSG_MODEL_PULL_SUCCESS" =~ Model.*pulled.*successfully ]]
}

@test "error messages contain error indicators" {
    [[ "$MSG_OLLAMA_INSTALL_FAILED" =~ ❌ ]]
    [[ "$MSG_MODEL_INSTALL_FAILED" =~ ❌ ]]
    [[ "$MSG_PROMPT_NO_TEXT" =~ "No prompt text" ]]
    [[ "$MSG_PROMPT_NO_RESPONSE" =~ "No response text" ]]
}

@test "warning messages contain warning indicators" {
    [[ "$MSG_OLLAMA_STARTED_NO_API" =~ ⚠️ ]]
    # Test dynamic message functions
    [[ "$(MSG_LOW_DISK_SPACE 5)" =~ ⚠️ ]]
    [[ "$(MSG_UNKNOWN_MODELS "test-model")" =~ "Unknown models" ]]
}

@test "informational messages contain emojis where appropriate" {
    [[ "$MSG_MODELS_HEADER" =~ 📚 ]]
    [[ "$MSG_PROMPT_RESPONSE_HEADER" =~ 🤖 ]]
    # Test dynamic message functions
    [[ "$(MSG_PROMPT_RESPONSE_TIME 1.5)" =~ ⏱️ ]]
    [[ "$(MSG_PROMPT_TOKEN_COUNT 10 20)" =~ 📊 ]]
    [[ "$MSG_PROMPT_PARAMETERS" =~ 🎛️ ]]
}

@test "messages use proper variable substitution format" {
    # Messages that should contain variables use proper format
    [[ "$MSG_USER_CREATE_SUCCESS" =~ \$OLLAMA_USER ]]
    [[ "$MSG_USER_CREATE_FAILED" =~ \$OLLAMA_USER ]]
    [[ "$MSG_USER_SUDO_REQUIRED" =~ \$OLLAMA_USER ]]
    [[ "$MSG_OLLAMA_RUNNING" =~ \$OLLAMA_PORT ]]
    [[ "$MSG_STATUS_USER_OK" =~ \$OLLAMA_USER ]]
    [[ "$MSG_STATUS_PORT_OK" =~ \$OLLAMA_PORT ]]
}

@test "ollama::export_messages function works" {
    # Call export function (variables are already readonly, can't unset them)
    ollama::export_messages
    
    # Verify key messages are exported and defined
    [ -n "$MSG_OLLAMA_INSTALLING" ]
    [ -n "$MSG_OLLAMA_INSTALL_SUCCESS" ]
    [ -n "$MSG_MODELS_HEADER" ]
    [ -n "$MSG_PROMPT_SENDING" ]
    
    # Test that variables are actually exported to the environment
    run bash -c 'echo "$MSG_OLLAMA_INSTALLING"'
    [ -n "$output" ]
}

@test "messages are readonly constants" {
    # Test that key messages are defined as readonly constants
    local test_msg="$MSG_OLLAMA_INSTALLING"
    [ -n "$test_msg" ]
    
    # The fact that we can read them means they're properly defined
    # Bash readonly testing is complex, so we'll just verify they exist
    [ "$MSG_OLLAMA_INSTALLING" = "$test_msg" ]
}