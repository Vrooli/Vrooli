#!/usr/bin/env bash
# Core functionality for Pandas-AI resource

# Get the script directory
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCRIPT_DIR}/../.." && builtin pwd)}"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/network.sh"
source "${APP_ROOT}/scripts/resources/port_registry.sh"

# Resource configuration
readonly RESOURCE_NAME="pandas-ai"
readonly PANDAS_AI_PORT="${PANDAS_AI_PORT:-$(port_registry::get_port "$RESOURCE_NAME")}"
readonly PANDAS_AI_URL="http://localhost:${PANDAS_AI_PORT}"
readonly PANDAS_AI_DATA_DIR="${APP_ROOT}/data/resources/pandas-ai"

# Core functions
pandas_ai::get_status() {
    local verbose="${1:-false}"
    
    # Check if service is running
    if pgrep -f "pandas-ai.*server.py" > /dev/null; then
        # Check health endpoint
        if timeout 5 curl -sf "${PANDAS_AI_URL}/health" > /dev/null 2>&1; then
            echo "healthy"
        else
            echo "degraded"
        fi
    else
        echo "stopped"
    fi
}

pandas_ai::wait_for_ready() {
    local timeout="${1:-30}"
    local elapsed=0
    
    log::info "Waiting for Pandas-AI to be ready..."
    
    while [[ $elapsed -lt $timeout ]]; do
        if timeout 5 curl -sf "${PANDAS_AI_URL}/health" > /dev/null 2>&1; then
            log::success "Pandas-AI is ready"
            return 0
        fi
        sleep 1
        ((elapsed++))
    done
    
    log::error "Pandas-AI did not become ready within ${timeout} seconds"
    return 1
}

pandas_ai::analyze() {
    local query="$1"
    local data="$2"
    
    if [[ -z "$query" ]]; then
        log::error "Query is required"
        return 1
    fi
    
    # Register agent if agent management is available
    local agent_id=""
    if type -t agents::register &>/dev/null; then
        agent_id=$(agents::generate_id)
        local command_string="pandas_ai::analyze $*"
        if agents::register "$agent_id" $$ "$command_string"; then
            log::debug "Registered agent: $agent_id"
            
            # Set up signal handler for cleanup
            pandas_ai::setup_agent_cleanup "$agent_id"
        fi
    fi
    
    local payload
    if [[ -n "$data" ]]; then
        payload=$(jq -n --arg q "$query" --argjson d "$data" '{query: $q, data: $d}')
    else
        payload=$(jq -n --arg q "$query" '{query: $q}')
    fi
    
    local response
    response=$(timeout 10 curl -sf -X POST "${PANDAS_AI_URL}/analyze" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.result // .error // "No result"'
        
        # Unregister agent on successful completion
        if [[ -n "$agent_id" ]] && type -t agents::unregister &>/dev/null; then
            agents::unregister "$agent_id" >/dev/null 2>&1
        fi
        return 0
    else
        log::error "Analysis request failed: $response"
        return 1
    fi
}

pandas_ai::get_info() {
    cat <<EOF
{
  "name": "${RESOURCE_NAME}",
  "version": "1.0.0",
  "port": ${PANDAS_AI_PORT},
  "url": "${PANDAS_AI_URL}",
  "status": "$(pandas_ai::get_status)",
  "data_dir": "${PANDAS_AI_DATA_DIR}"
}
EOF
}

pandas_ai::get_port() {
    echo "${PANDAS_AI_PORT}"
}

pandas_ai::get_url() {
    echo "${PANDAS_AI_URL}"
}