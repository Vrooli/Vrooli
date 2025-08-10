#!/usr/bin/env bash
# Agent Metareasoning Manager startup script
# Go-only implementation

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Load Vrooli utilities
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Configuration
API_PORT="${API_PORT:-8093}"

startup::check_go_available() {
    log::info "Checking Go implementation..."
    
    # Check Go implementation
    if [[ -f "$SCENARIO_DIR/api/main.go" ]] && command -v go >/dev/null 2>&1; then
        log::success "✅ Go implementation available"
        return 0
    else
        log::error "❌ Go implementation not available"
        if [[ ! -f "$SCENARIO_DIR/api/main.go" ]]; then
            log::info "   Missing: api/main.go"
        fi
        if ! command -v go >/dev/null 2>&1; then
            log::info "   Missing: go runtime"
        fi
        return 1
    fi
}

startup::build_go_api() {
    log::info "Building Go coordination API..."
    
    cd "$SCENARIO_DIR/api" || {
        log::error "Failed to change to api directory"
        return 1
    }
    
    # Download dependencies
    log::info "Downloading Go dependencies..."
    go mod download || {
        log::error "Failed to download Go dependencies"
        return 1
    }
    
    # Build the binary
    log::info "Compiling Go binary..."
    go build -o metareasoning-api main.go || {
        log::error "Failed to build Go binary"
        return 1
    }
    
    # Make executable
    chmod +x metareasoning-api
    
    log::success "Go API built successfully"
    return 0
}

startup::start_go_api() {
    log::info "Starting Go coordination API on port $API_PORT..."
    
    cd "$SCENARIO_DIR/api" || return 1
    
    # Set environment variables
    export PORT="$API_PORT"
    export N8N_BASE_URL="${N8N_BASE_URL:-http://localhost:5678}"
    export WINDMILL_BASE_URL="${WINDMILL_BASE_URL:-http://localhost:8000}"
    export WINDMILL_WORKSPACE="${WINDMILL_WORKSPACE:-demo}"
    
    # Start the API server
    ./metareasoning-api &
    local api_pid=$!
    
    # Wait for server to start
    log::info "Waiting for Go API to start..."
    local attempts=0
    local max_attempts=30
    
    while [[ $attempts -lt $max_attempts ]]; do
        if curl -sf "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
            log::success "Go API started successfully (PID: $api_pid)"
            echo "$api_pid" > "$SCENARIO_DIR/.api_pid"
            return 0
        fi
        
        sleep 1
        ((attempts++))
    done
    
    log::error "Go API failed to start within $max_attempts seconds"
    kill "$api_pid" 2>/dev/null || true
    return 1
}


startup::show_endpoints() {
    log::info ""
    log::success "🚀 Agent Metareasoning Manager is running!"
    log::info ""
    log::info "Go Coordination API endpoints:"
    log::info "  Health:    http://localhost:$API_PORT/health"
    log::info "  Workflows: http://localhost:$API_PORT/workflows"
    log::info "  Analysis:  http://localhost:$API_PORT/analyze/{type}"
    log::info "  Execute:   http://localhost:$API_PORT/execute/{platform}/{workflow}"
    
    log::info ""
    log::info "CLI Commands:"
    log::info "  metareasoning --help"
    log::info "  metareasoning health"
    log::info "  metareasoning analyze pros-cons \"Should we migrate?\""
    log::info ""
    log::info "External Services:"
    log::info "  Windmill Dashboard: http://localhost:${WINDMILL_PORT:-8000}"
    log::info "  n8n Editor:         http://localhost:${N8N_PORT:-5678}"
    log::info ""
}

startup::cleanup() {
    log::info "Cleaning up startup processes..."
    
    # Kill API server if running
    if [[ -f "$SCENARIO_DIR/.api_pid" ]]; then
        local api_pid
        api_pid=$(cat "$SCENARIO_DIR/.api_pid")
        if kill "$api_pid" 2>/dev/null; then
            log::info "Stopped API server (PID: $api_pid)"
        fi
        rm -f "$SCENARIO_DIR/.api_pid"
    fi
}

# Main startup sequence
startup::main() {
    log::header "Starting Agent Metareasoning Manager"
    
    # Set up cleanup on exit
    trap startup::cleanup EXIT INT TERM
    
    # Check Go availability
    startup::check_go_available || return 1
    
    # Build and start Go API
    startup::build_go_api || return 1
    startup::start_go_api || return 1
    
    # Show available endpoints
    startup::show_endpoints
    
    # Keep running
    log::info "Press Ctrl+C to stop..."
    wait
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    startup::main "$@"
fi