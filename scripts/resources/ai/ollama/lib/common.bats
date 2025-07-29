#!/usr/bin/env bats

# Tests for Ollama common utilities

setup() {
    # Mock the resources directory structure
    export SCRIPT_DIR="$(dirname "$BATS_TEST_FILENAME")"
    export RESOURCES_DIR="$SCRIPT_DIR/../.."
    export DESCRIPTION="Test Ollama management"
    
    # Mock functions that would normally be sourced
    args::reset() { return 0; }
    args::register_help() { return 0; }
    args::register_yes() { return 0; }
    args::register() { return 0; }
    args::is_asking_for_help() { return 1; }
    args::parse() { return 0; }
    args::get() { 
        case "$1" in
            "action") echo "status" ;;
            "force") echo "no" ;;
            "yes") echo "no" ;;
            "models") echo "" ;;
            "skip-models") echo "no" ;;
            "text") echo "" ;;
            "model") echo "" ;;
            "type") echo "general" ;;
            "format") echo "text" ;;
            "temperature") echo "0.8" ;;
            "max-tokens") echo "" ;;
            "top-p") echo "0.9" ;;
            "top-k") echo "40" ;;
            "seed") echo "" ;;
            "system") echo "" ;;
        esac
    }
    args::usage() { echo "Usage: $1"; }
    
    resources::update_config() { return 0; }
    
    # Source the common utilities
    source "$(dirname "$BATS_TEST_FILENAME")/common.sh"
}

@test "ollama::parse_arguments sets environment variables correctly" {
    ollama::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$FORCE" = "no" ]
    [ "$YES" = "no" ]
    [ "$MODELS_INPUT" = "" ]
    [ "$SKIP_MODELS" = "no" ]
    [ "$PROMPT_TEXT" = "" ]
    [ "$PROMPT_MODEL" = "" ]
    [ "$PROMPT_TYPE" = "general" ]
    [ "$OUTPUT_FORMAT" = "text" ]
    [ "$TEMPERATURE" = "0.8" ]
    [ "$MAX_TOKENS" = "" ]
    [ "$TOP_P" = "0.9" ]
    [ "$TOP_K" = "40" ]
    [ "$SEED" = "" ]
    [ "$SYSTEM_PROMPT" = "" ]
}

@test "ollama::usage displays usage information" {
    run ollama::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Examples:" ]]
    [[ "$output" =~ "--action install" ]]
    [[ "$output" =~ "--action prompt" ]]
    [[ "$output" =~ "--action status" ]]
}

@test "ollama::update_config calls resources::update_config" {
    # Mock the function to capture call
    resources::update_config() {
        echo "Called with: $*"
    }
    
    run ollama::update_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Called with: ai ollama" ]]
}

@test "ollama::validate_temperature accepts valid temperatures" {
    # Valid temperatures
    ollama::validate_temperature "0.0"
    [ "$?" -eq 0 ]
    
    ollama::validate_temperature "0.8"
    [ "$?" -eq 0 ]
    
    ollama::validate_temperature "1.0"
    [ "$?" -eq 0 ]
    
    ollama::validate_temperature "2.0"
    [ "$?" -eq 0 ]
    
    ollama::validate_temperature "0.5"
    [ "$?" -eq 0 ]
}

@test "ollama::validate_temperature rejects invalid temperatures" {
    # Invalid temperatures
    ollama::validate_temperature "-0.1"
    [ "$?" -eq 1 ]
    
    ollama::validate_temperature "2.1"
    [ "$?" -eq 1 ]
    
    ollama::validate_temperature "abc"
    [ "$?" -eq 1 ]
    
    ollama::validate_temperature ""
    [ "$?" -eq 1 ]
    
    ollama::validate_temperature "3.0"
    [ "$?" -eq 1 ]
}

@test "ollama::validate_top_p accepts valid top-p values" {
    # Valid top-p values
    ollama::validate_top_p "0.0"
    [ "$?" -eq 0 ]
    
    ollama::validate_top_p "0.5"
    [ "$?" -eq 0 ]
    
    ollama::validate_top_p "0.9"
    [ "$?" -eq 0 ]
    
    ollama::validate_top_p "1.0"
    [ "$?" -eq 0 ]
}

@test "ollama::validate_top_p rejects invalid top-p values" {
    # Invalid top-p values
    ollama::validate_top_p "-0.1"
    [ "$?" -eq 1 ]
    
    ollama::validate_top_p "1.1"
    [ "$?" -eq 1 ]
    
    ollama::validate_top_p "abc"
    [ "$?" -eq 1 ]
    
    ollama::validate_top_p ""
    [ "$?" -eq 1 ]
}

@test "ollama::validate_top_k accepts valid top-k values" {
    # Valid top-k values
    ollama::validate_top_k "1"
    [ "$?" -eq 0 ]
    
    ollama::validate_top_k "40"
    [ "$?" -eq 0 ]
    
    ollama::validate_top_k "100"
    [ "$?" -eq 0 ]
}

@test "ollama::validate_top_k rejects invalid top-k values" {
    # Invalid top-k values
    ollama::validate_top_k "0"
    [ "$?" -eq 1 ]
    
    ollama::validate_top_k "-1"
    [ "$?" -eq 1 ]
    
    ollama::validate_top_k "abc"
    [ "$?" -eq 1 ]
    
    ollama::validate_top_k ""
    [ "$?" -eq 1 ]
    
    ollama::validate_top_k "1.5"
    [ "$?" -eq 1 ]
}

@test "ollama::validate_max_tokens accepts valid max-tokens values" {
    # Valid max-tokens values
    ollama::validate_max_tokens ""      # Empty is valid
    [ "$?" -eq 0 ]
    
    ollama::validate_max_tokens "1"
    [ "$?" -eq 0 ]
    
    ollama::validate_max_tokens "100"
    [ "$?" -eq 0 ]
    
    ollama::validate_max_tokens "4096"
    [ "$?" -eq 0 ]
}

@test "ollama::validate_max_tokens rejects invalid max-tokens values" {
    # Invalid max-tokens values
    ollama::validate_max_tokens "0"
    [ "$?" -eq 1 ]
    
    ollama::validate_max_tokens "-1"
    [ "$?" -eq 1 ]
    
    ollama::validate_max_tokens "abc"
    [ "$?" -eq 1 ]
    
    ollama::validate_max_tokens "1.5"
    [ "$?" -eq 1 ]
}

@test "all validation functions exist and are callable" {
    # Test that all validation functions are defined
    type ollama::validate_temperature >/dev/null
    type ollama::validate_top_p >/dev/null
    type ollama::validate_top_k >/dev/null
    type ollama::validate_max_tokens >/dev/null
}