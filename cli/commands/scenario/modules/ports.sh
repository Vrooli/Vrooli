#!/usr/bin/env bash
# Scenario Port Management Module
# Handles port queries and allocation

set -euo pipefail

# Get a specific port for a scenario
# Ultra-fast command to get a specific port for a scenario
# Reads directly from process JSON files, no HTTP calls
scenario::ports::get() {
    local scenario_name="${1:-}"
    local port_name="${2:-}"
    
    if [[ -z "$scenario_name" ]] || [[ -z "$port_name" ]]; then
        log::error "Scenario name and port name required"
        echo "Usage: vrooli scenario port <scenario-name> <port-name>"
        echo ""
        echo "Examples:"
        echo "  vrooli scenario port ecosystem-manager API_PORT"
        echo "  vrooli scenario port ecosystem-manager UI_PORT"
        return 1
    fi
    
    # Map port names to process step names and try multiple variations
    # This is a convention: API_PORT -> start-api, UI_PORT -> start-ui
    local -a possible_steps=()
    case "$port_name" in
        API_PORT)
            possible_steps=("start-api" "start-app" "api" "app")
            ;;
        UI_PORT)
            possible_steps=("start-ui" "ui")
            ;;
        *)
            # For other port names, try lowercase without _PORT suffix
            # e.g., WORKER_PORT -> start-worker
            local base_name="${port_name%_PORT}"
            possible_steps=("start-${base_name,,}" "${base_name,,}")
            ;;
    esac
    
    # Try each possible step name
    for step_name in "${possible_steps[@]}"; do
        local process_file="$HOME/.vrooli/processes/scenarios/${scenario_name}/${step_name}.json"
        
        if [[ -f "$process_file" ]]; then
            # Extract port directly from JSON file (single jq call, very fast)
            local port_value
            port_value=$(jq -r '.port // empty' "$process_file" 2>/dev/null)
            
            if [[ -n "$port_value" ]] && [[ "$port_value" != "null" ]]; then
                # Just output the port number, nothing else (for easy scripting)
                echo "$port_value"
                return 0
            fi
        fi
    done
    
    # Port not found in any process file
    return 1
}

# Map port names to process step names (utility function)
# Returns the first/most likely step name for compatibility
scenario::ports::map_port_to_step() {
    local port_name="$1"
    
    case "$port_name" in
        API_PORT)
            echo "start-api"  # Most common convention
            ;;
        UI_PORT)
            echo "start-ui"
            ;;
        *)
            # For other port names, try lowercase without _PORT suffix
            # e.g., WORKER_PORT -> start-worker
            local base_name="${port_name%_PORT}"
            echo "start-${base_name,,}"
            ;;
    esac
}

# Get port value from process file (utility function)
scenario::ports::get_from_process_file() {
    local scenario_name="$1"
    local step_name="$2"
    
    local process_file="$HOME/.vrooli/processes/scenarios/${scenario_name}/${step_name}.json"
    
    if [[ ! -f "$process_file" ]]; then
        return 1
    fi
    
    # Extract port directly from JSON file
    local port_value
    port_value=$(jq -r '.port // empty' "$process_file" 2>/dev/null)
    
    if [[ -n "$port_value" ]] && [[ "$port_value" != "null" ]]; then
        echo "$port_value"
        return 0
    else
        return 1
    fi
}