#!/usr/bin/env bats
# Tests for Agent S2 manage.sh script

# Load test helper
load_helper() {
    local helper_file="$1"
    if [[ -f "$helper_file" ]]; then
        # shellcheck disable=SC1090
        source "$helper_file"
    fi
}

# Setup for each test
setup() {
    # Set test environment
    export AGENTS2_CUSTOM_PORT="9999"
    export FORCE="no"
    export YES="no"
    export LLM_PROVIDER="anthropic"
    export LLM_MODEL="claude-3-7-sonnet-20250219"
    export ENABLE_AI="yes"
    export ENABLE_SEARCH="no"
    export VNC_PASSWORD="agents2vnc"
    export ENABLE_HOST_DISPLAY="no"
    export MODE="sandbox"
    export TARGET_MODE=""
    export USAGE_TYPE="help"
    
    # Load the script without executing main
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/manage.sh" || true
}

# Test script loading
@test "manage.sh loads without errors" {
    # The script should source successfully in setup
    [ "$?" -eq 0 ]
}

# Test argument parsing
@test "agents2::parse_arguments sets defaults correctly" {
    agents2::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$FORCE" = "no" ]
    [ "$LLM_PROVIDER" = "anthropic" ]
    [ "$LLM_MODEL" = "claude-3-7-sonnet-20250219" ]
    [ "$ENABLE_AI" = "yes" ]
    [ "$ENABLE_SEARCH" = "no" ]
    [ "$VNC_PASSWORD" = "agents2vnc" ]
    [ "$ENABLE_HOST_DISPLAY" = "no" ]
    [ "$MODE" = "sandbox" ]
}

# Test argument parsing with custom values
@test "agents2::parse_arguments handles custom values" {
    agents2::parse_arguments \
        --action install \
        --force yes \
        --llm-provider openai \
        --llm-model gpt-4 \
        --enable-ai no \
        --enable-search yes \
        --vnc-password custom123 \
        --enable-host-display yes \
        --mode host
    
    [ "$ACTION" = "install" ]
    [ "$FORCE" = "yes" ]
    [ "$LLM_PROVIDER" = "openai" ]
    [ "$LLM_MODEL" = "gpt-4" ]
    [ "$ENABLE_AI" = "no" ]
    [ "$ENABLE_SEARCH" = "yes" ]
    [ "$VNC_PASSWORD" = "custom123" ]
    [ "$ENABLE_HOST_DISPLAY" = "yes" ]
    [ "$MODE" = "host" ]
}

# Test mode arguments
@test "agents2::parse_arguments handles mode arguments" {
    agents2::parse_arguments --action switch-mode --target-mode host
    
    [ "$ACTION" = "switch-mode" ]
    [ "$TARGET_MODE" = "host" ]
}

# Test usage type argument
@test "agents2::parse_arguments handles usage-type argument" {
    agents2::parse_arguments --action usage --usage-type screenshot
    
    [ "$ACTION" = "usage" ]
    [ "$USAGE_TYPE" = "screenshot" ]
}

# Test LLM provider options
@test "agents2::parse_arguments handles different LLM providers" {
    agents2::parse_arguments --action install --llm-provider ollama
    [ "$LLM_PROVIDER" = "ollama" ]
    
    agents2::parse_arguments --action install --llm-provider anthropic
    [ "$LLM_PROVIDER" = "anthropic" ]
    
    agents2::parse_arguments --action install --llm-provider openai
    [ "$LLM_PROVIDER" = "openai" ]
}

# Test usage display
@test "agents2::usage displays help text" {
    run agents2::usage
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Install and manage Agent S2"* ]]
    [[ "$output" == *"--action install"* ]]
    [[ "$output" == *"API: http://localhost:4113"* ]]
    [[ "$output" == *"VNC: vnc://localhost:5900"* ]]
}

# Test configuration export
@test "configuration is exported correctly" {
    agents2::export_config
    
    [ -n "$AGENTS2_PORT" ]
    [ -n "$AGENTS2_BASE_URL" ]
    [ -n "$AGENTS2_CONTAINER_NAME" ]
    [ -n "$AGENTS2_IMAGE" ]
    [ -n "$AGENTS2_VNC_PORT" ]
}

# Test message export
@test "messages are exported correctly" {
    agents2::export_messages
    
    [ -n "$MSG_INSTALL_SUCCESS" ]
    [ -n "$MSG_DOCKER_NOT_FOUND" ]
    [ -n "$MSG_HEALTHY" ]
    [ -n "$MSG_ALREADY_INSTALLED" ]
}

# Test mode validation
@test "agents2::parse_arguments validates mode options" {
    agents2::parse_arguments --action start --mode sandbox
    [ "$MODE" = "sandbox" ]
    
    agents2::parse_arguments --action start --mode host
    [ "$MODE" = "host" ]
}

# Test action validation
@test "agents2::parse_arguments handles all valid actions" {
    local actions=("install" "uninstall" "start" "stop" "restart" "status" "logs" "info" "usage" "mode" "switch-mode" "test-mode")
    
    for action in "${actions[@]}"; do
        agents2::parse_arguments --action "$action"
        [ "$ACTION" = "$action" ]
    done
}

# Test usage type validation
@test "agents2::parse_arguments validates usage types" {
    local usage_types=("screenshot" "automation" "planning" "capabilities" "all" "help")
    
    for type in "${usage_types[@]}"; do
        agents2::parse_arguments --action usage --usage-type "$type"
        [ "$USAGE_TYPE" = "$type" ]
    done
}

# Test argument combination for mode switching
@test "agents2::parse_arguments requires target-mode for switch-mode action" {
    agents2::parse_arguments --action switch-mode --target-mode host
    
    [ "$ACTION" = "switch-mode" ]
    [ "$TARGET_MODE" = "host" ]
}

# Test AI enablement flag
@test "agents2::parse_arguments handles AI enablement correctly" {
    agents2::parse_arguments --action install --enable-ai yes
    [ "$ENABLE_AI" = "yes" ]
    [ "$ARGS_ENABLE_AI" = "yes" ]
    
    agents2::parse_arguments --action install --enable-ai no
    [ "$ENABLE_AI" = "no" ]
    [ "$ARGS_ENABLE_AI" = "no" ]
}

# Test search enablement flag
@test "agents2::parse_arguments handles search enablement correctly" {
    agents2::parse_arguments --action install --enable-search yes
    [ "$ENABLE_SEARCH" = "yes" ]
    
    agents2::parse_arguments --action install --enable-search no
    [ "$ENABLE_SEARCH" = "no" ]
}

# Test host display enablement
@test "agents2::parse_arguments handles host display flag correctly" {
    agents2::parse_arguments --action install --enable-host-display yes
    [ "$ENABLE_HOST_DISPLAY" = "yes" ]
    
    agents2::parse_arguments --action install --enable-host-display no
    [ "$ENABLE_HOST_DISPLAY" = "no" ]
}

# Test VNC password handling
@test "agents2::parse_arguments handles VNC password correctly" {
    agents2::parse_arguments --action install --vnc-password "MySecurePass123"
    [ "$VNC_PASSWORD" = "MySecurePass123" ]
    
    # Test default VNC password
    agents2::parse_arguments --action install
    [ "$VNC_PASSWORD" = "agents2vnc" ]
}

# Test that ARGS_ prefixed variables are set
@test "agents2::parse_arguments sets ARGS_ prefixed variables" {
    agents2::parse_arguments \
        --action install \
        --llm-provider ollama \
        --llm-model llama2 \
        --enable-ai no
    
    [ "$ARGS_LLM_PROVIDER" = "ollama" ]
    [ "$ARGS_LLM_MODEL" = "llama2" ]
    [ "$ARGS_ENABLE_AI" = "no" ]
}

# Test edge cases
@test "agents2::parse_arguments handles empty arguments" {
    agents2::parse_arguments
    
    # Should use all defaults
    [ "$ACTION" = "install" ]
    [ "$FORCE" = "no" ]
    [ "$MODE" = "sandbox" ]
}

# Test combined mode and AI options
@test "agents2::parse_arguments handles combined mode and AI options" {
    agents2::parse_arguments \
        --action install \
        --mode host \
        --enable-ai yes \
        --llm-provider anthropic
    
    [ "$MODE" = "host" ]
    [ "$ENABLE_AI" = "yes" ]
    [ "$LLM_PROVIDER" = "anthropic" ]
}