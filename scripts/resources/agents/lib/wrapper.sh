#!/usr/bin/env bash
################################################################################
# Agent Wrapper Library
# 
# Standard wrapper functions for resource agent registration
# Provides consistent agent management across all resources
################################################################################

#######################################
# Execute a function with agent registration and cleanup
# This is the main wrapper that all resources should use for operations
# that need agent tracking (interactive, execution, long-running)
#
# Arguments:
#   $1 - Operation name (for logging/metrics)
#   $2 - Function to execute
#   $@ - Arguments to pass to the function
#
# Returns:
#   Exit code from wrapped function
#
# Usage:
#   agents::with_agent "content-chat" "ollama_chat" "llama3.2:3b"
#   agents::with_agent "batch-process" "claude_code_batch" "simple" "prompt.txt"
#######################################
agents::with_agent() {
    local operation_name="$1"
    local operation_func="$2"
    shift 2
    
    # Check if agent management is available
    if ! type -t agent_manager::register &>/dev/null; then
        log::debug "Agent management not available, executing function directly"
        "$operation_func" "$@"
        return $?
    fi
    
    # Generate agent ID and register
    local agent_id=""
    agent_id=$(agent_manager::generate_id)
    local command_string="resource-${RESOURCE_NAME} ${operation_name} $*"
    
    # Register agent
    local registration_success=false
    if agent_manager::register "$agent_id" $$ "$command_string"; then
        log::debug "Registered agent: $agent_id for operation: $operation_name"
        registration_success=true
        
        # Track metrics if available
        if type -t agents::metrics::increment &>/dev/null; then
            agents::metrics::increment "$REGISTRY_FILE" "$agent_id" "requests" 1
        fi
    else
        log::warn "Failed to register agent for operation: $operation_name"
        # Continue execution even if registration fails
    fi
    
    # Setup cleanup trap if registration succeeded
    if [[ "$registration_success" == "true" ]]; then
        # Export agent ID for cleanup function
        export CURRENT_AGENT_ID="$agent_id"
        
        # Define cleanup function
        agents::cleanup_current_agent() {
            if [[ -n "${CURRENT_AGENT_ID:-}" ]] && type -t agent_manager::unregister &>/dev/null; then
                agent_manager::unregister "${CURRENT_AGENT_ID}" >/dev/null 2>&1
                unset CURRENT_AGENT_ID
            fi
        }
        
        # Register cleanup for common signals
        trap 'agents::cleanup_current_agent' EXIT SIGTERM SIGINT
    fi
    
    # Execute the actual operation
    local result=0
    "$operation_func" "$@"
    result=$?
    
    # Track completion metrics
    if [[ "$registration_success" == "true" ]] && type -t agents::metrics::increment &>/dev/null; then
        if [[ $result -eq 0 ]]; then
            agents::metrics::increment "$REGISTRY_FILE" "$agent_id" "completions" 1
        else
            agents::metrics::increment "$REGISTRY_FILE" "$agent_id" "errors" 1
        fi
    fi
    
    # Manual cleanup (trap will also run, but this ensures immediate cleanup)
    if [[ "$registration_success" == "true" ]] && [[ -n "$agent_id" ]]; then
        if type -t agent_manager::unregister &>/dev/null; then
            agent_manager::unregister "$agent_id" >/dev/null 2>&1
        fi
        unset CURRENT_AGENT_ID
        trap - EXIT SIGTERM SIGINT  # Remove trap
    fi
    
    return $result
}

#######################################
# Simple wrapper for operations that should register agents
# Less flexible than agents::with_agent but easier to use for simple cases
#
# Arguments:
#   $1 - Operation name
#   $@ - Command and arguments to execute
#
# Returns:
#   Exit code from command
#
# Usage:
#   agents::track_operation "model-pull" ollama pull "llama3.2:3b"
#   agents::track_operation "ai-chat" python chat_bot.py --model gpt-4
#######################################
agents::track_operation() {
    local operation_name="$1"
    shift
    
    # Create a temporary function to wrap the command
    local temp_func_name="temp_operation_$$"
    eval "${temp_func_name}() { \"\$@\"; }"
    
    # Use the main wrapper
    agents::with_agent "$operation_name" "$temp_func_name" "$@"
    local result=$?
    
    # Clean up temporary function
    unset -f "$temp_func_name"
    
    return $result
}

#######################################
# Check if an operation should register an agent
# Based on standard patterns for operation types
#
# Arguments:
#   $1 - Operation name/type
#
# Returns:
#   0 if should register agent, 1 if not
#
# Usage:
#   if agents::should_track "chat"; then
#       agents::with_agent "content-chat" "my_chat_func" "$@"
#   else
#       my_chat_func "$@"
#   fi
#######################################
agents::should_track() {
    local operation="$1"
    
    # Operations that SHOULD register agents
    case "$operation" in
        # Interactive operations
        chat|interactive|session|resume|repl)
            return 0
            ;;
        # Execution operations  
        execute|generate|run|process|inference|completion)
            return 0
            ;;
        # Batch operations
        batch|bulk|multi|parallel|pipeline)
            return 0
            ;;
        # Long-running operations
        pull|download|install|train|fine-tune|upload)
            return 0
            ;;
        # Template/content execution
        template-run|content-execute)
            return 0
            ;;
        *)
            # Operations that should NOT register agents
            case "$operation" in
                # Quick info commands
                list|status|info|show|logs|help)
                    return 1
                    ;;
                # Configuration commands
                get|set|config|settings|configure)
                    return 1
                    ;;
                # Service management (not agent operations)
                start|stop|restart|reload)
                    return 1
                    ;;
                *)
                    # Default to tracking for unknown operations (safer)
                    return 0
                    ;;
            esac
            ;;
    esac
}

#######################################
# Convenience function to conditionally wrap operations
# Automatically determines if operation should be tracked
#
# Arguments:
#   $1 - Operation name/type
#   $2 - Function to execute
#   $@ - Arguments to pass to function
#
# Returns:
#   Exit code from function
#
# Usage:
#   agents::auto_track "chat" "ollama_chat" "llama3.2:3b"
#   agents::auto_track "list" "ollama_list_models"  # Won't track
#######################################
agents::auto_track() {
    local operation="$1"
    local operation_func="$2"
    shift 2
    
    if agents::should_track "$operation"; then
        agents::with_agent "$operation" "$operation_func" "$@"
    else
        "$operation_func" "$@"
    fi
}

# Provide a default cleanup implementation so exports succeed even when
# agents::with_agent hasn't registered a cleanup handler yet. The function is
# redefined on demand within agents::with_agent when registration succeeds.
agents::cleanup_current_agent() {
    return 0
}

# Export functions for use by resources
export -f agents::with_agent
export -f agents::track_operation
export -f agents::should_track
export -f agents::auto_track
export -f agents::cleanup_current_agent
