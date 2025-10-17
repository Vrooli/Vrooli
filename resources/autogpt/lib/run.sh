#!/bin/bash

# AutoGPT Run Functions
# Execute AutoGPT with specific goals or tasks

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
AUTOGPT_RUN_DIR="${APP_ROOT}/resources/autogpt/lib"

# Source common functions
source "${AUTOGPT_RUN_DIR}/common.sh"

# Main run function
autogpt::run() {
    local goal="${1:-}"
    local mode="${2:-task}"  # task, continuous, benchmark
    
    if [[ -z "${goal}" ]]; then
        log::error "No goal specified"
        echo "Usage: vrooli resource autogpt run \"Your goal here\""
        return 1
    fi
    
    log::header "Running AutoGPT"
    log::info "Goal: ${goal}"
    log::info "Mode: ${mode}"
    
    # Check if AutoGPT is installed
    if ! autogpt::is_installed; then
        log::error "AutoGPT is not installed. Run 'install' first."
        return 1
    fi
    
    # Start AutoGPT if not running
    if ! autogpt::is_running; then
        log::info "Starting AutoGPT service..."
        source "${AUTOGPT_LIB_DIR}/start.sh"
        autogpt::start || {
            log::error "Failed to start AutoGPT"
            return 1
        }
        # Wait for service to be ready
        sleep 5
    fi
    
    # Create a goal file
    local goal_file="/tmp/autogpt_goal_$(date +%s).json"
    cat > "${goal_file}" <<EOF
{
    "name": "User Goal",
    "description": "${goal}",
    "mode": "${mode}",
    "tasks": []
}
EOF
    
    # Inject the goal
    source "${AUTOGPT_LIB_DIR}/inject.sh"
    autogpt::inject_goal "${goal_file}" || {
        log::error "Failed to inject goal"
        rm -f "${goal_file}"
        return 1
    }
    
    # Execute the goal
    log::info "Executing goal in AutoGPT..."
    
    # Register agent for tracking
    local agent_id
    agent_id=$(agents::generate_id)
    local command="autogpt::run \"$goal\" $mode"
    
    # Run AutoGPT with the goal
    local output
    if [[ "${mode}" == "continuous" ]]; then
        # Continuous mode - runs until goal is achieved
        (
            # Setup cleanup trap
            autogpt::setup_agent_cleanup "$agent_id"
            
            docker exec -i "${AUTOGPT_CONTAINER_NAME}" python -m autogpt \
                --continuous \
                --goal "${goal}" \
                2>&1 | tee "${AUTOGPT_LOGS_DIR}/run_$(date +%s).log"
        ) &
    else
        # Task mode - single execution
        (
            # Setup cleanup trap
            autogpt::setup_agent_cleanup "$agent_id"
            
            docker exec -i "${AUTOGPT_CONTAINER_NAME}" python -m autogpt \
                --goal "${goal}" \
                2>&1 | tee "${AUTOGPT_LOGS_DIR}/run_$(date +%s).log"
        ) &
    fi
    
    local agent_pid=$!
    
    # Register the agent
    if agents::register "$agent_id" "$agent_pid" "$command"; then
        log::debug "Agent registered: $agent_id (PID: $agent_pid)"
    else
        log::warn "Failed to register agent in tracking system"
    fi
    
    # Wait for execution to complete and capture output
    wait $agent_pid
    local exit_code=$?
    
    # Clean up agent from registry
    agents::unregister "$agent_id" 2>/dev/null || true
    
    # Read output from log file (since we ran in background)
    local log_file="${AUTOGPT_LOGS_DIR}/run_$(date +%s).log"
    if [[ -f "$log_file" ]]; then
        output=$(cat "$log_file")
    else
        output="AutoGPT execution completed (no output captured)"
    fi
    
    # Display output
    echo "${output}"
    
    # Clean up
    rm -f "${goal_file}"
    
    if [[ $exit_code -eq 0 ]]; then
        log::success "AutoGPT execution complete"
    else
        log::error "AutoGPT execution failed with exit code: $exit_code"
    fi
    log::info "Logs saved to ${AUTOGPT_LOGS_DIR}"
    
    return $exit_code
}

# Run in interactive mode
autogpt::run_interactive() {
    log::header "AutoGPT Interactive Mode"
    
    if ! autogpt::is_running; then
        log::error "AutoGPT must be running for interactive mode"
        log::info "Start it with: vrooli resource autogpt start"
        return 1
    fi
    
    log::info "Entering interactive mode..."
    log::info "Type 'exit' to quit"
    
    # Run interactive session
    docker exec -it "${AUTOGPT_CONTAINER_NAME}" python -m autogpt --interactive
}

# Run with a specific configuration
autogpt::run_with_config() {
    local config_file="${1:-}"
    local goal="${2:-}"
    
    if [[ -z "${config_file}" ]]; then
        log::error "No configuration file specified"
        return 1
    fi
    
    if [[ ! -f "${config_file}" ]]; then
        log::error "Configuration file not found: ${config_file}"
        return 1
    fi
    
    # Inject the configuration
    source "${AUTOGPT_LIB_DIR}/inject.sh"
    autogpt::inject_config "${config_file}" || {
        log::error "Failed to inject configuration"
        return 1
    }
    
    # Run with the goal if provided
    if [[ -n "${goal}" ]]; then
        autogpt::run "${goal}"
    else
        log::info "Configuration injected. Start AutoGPT to use it."
    fi
}

# Run benchmark
autogpt::run_benchmark() {
    local benchmark="${1:-general}"
    
    log::header "Running AutoGPT Benchmark"
    log::info "Benchmark: ${benchmark}"
    
    if ! autogpt::is_running; then
        log::error "AutoGPT must be running for benchmarks"
        return 1
    fi
    
    # Run benchmark
    docker exec -i "${AUTOGPT_CONTAINER_NAME}" python -m autogpt.benchmark \
        --test "${benchmark}" \
        2>&1 | tee "${AUTOGPT_LOGS_DIR}/benchmark_$(date +%s).log"
    
    log::success "Benchmark complete"
}