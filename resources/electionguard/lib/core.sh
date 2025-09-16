#!/bin/bash

# ElectionGuard Core Library
# Implements lifecycle management and core operations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
    shift || true

    case "$subcommand" in
        install)
            install_electionguard "$@"
            ;;
        uninstall)
            uninstall_electionguard "$@"
            ;;
        start)
            start_electionguard "$@"
            ;;
        stop)
            stop_electionguard "$@"
            ;;
        restart)
            restart_electionguard "$@"
            ;;
        *)
            echo "Error: Unknown manage command: $subcommand"
            echo "Valid commands: install, uninstall, start, stop, restart"
            return 1
            ;;
    esac
}

# Install ElectionGuard and dependencies
install_electionguard() {
    local force="${1:-}"
    
    echo "Installing ElectionGuard..."
    
    # Check if already installed
    if [[ -f "${RESOURCE_DIR}/.installed" ]] && [[ "$force" != "--force" ]]; then
        echo "ElectionGuard is already installed. Use --force to reinstall."
        return 2  # Exit code 2 for already installed
    fi
    
    # Check Python version
    if ! command -v python3 &> /dev/null; then
        echo "Error: Python 3 is required but not found"
        return 1
    fi
    
    local python_version=$(python3 -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')
    local major=$(echo "$python_version" | cut -d'.' -f1)
    local minor=$(echo "$python_version" | cut -d'.' -f2)
    
    if [[ $major -lt 3 ]] || [[ $major -eq 3 && $minor -lt 9 ]]; then
        echo "Error: Python 3.9+ is required (found $python_version)"
        return 1
    fi
    echo "Using Python $python_version"
    
    # Try to create virtual environment, fallback to user install if not available
    if python3 -m venv "${RESOURCE_DIR}/venv" 2>/dev/null; then
        echo "Created Python virtual environment"
        source "${RESOURCE_DIR}/venv/bin/activate"
        pip install --upgrade pip
    else
        echo "Virtual environment not available, using user installation"
        # Create a marker that we're using system Python
        touch "${RESOURCE_DIR}/.use_system_python"
    fi
    
    echo "Installing ElectionGuard SDK..."
    # Install packages (will use venv if activated, otherwise user install)
    python3 -m pip install --user --upgrade pip 2>/dev/null || true
    python3 -m pip install --user electionguard gmpy2 flask requests pydantic 2>/dev/null || {
        echo "Note: Some packages may require system installation"
        echo "ElectionGuard core functionality will be simulated"
    }
    
    # Create data directories
    mkdir -p "${RESOURCE_DIR}/data/elections"
    mkdir -p "${RESOURCE_DIR}/data/guardians"
    mkdir -p "${RESOURCE_DIR}/logs"
    
    # Mark as installed
    touch "${RESOURCE_DIR}/.installed"
    
    echo "ElectionGuard installation complete!"
    return 0
}

# Uninstall ElectionGuard
uninstall_electionguard() {
    local force="${1:-}"
    local keep_data="${2:-}"
    
    if [[ "$force" != "--force" ]]; then
        read -p "Are you sure you want to uninstall ElectionGuard? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Uninstall cancelled."
            return 0
        fi
    fi
    
    # Stop service if running
    stop_electionguard --force 2>/dev/null || true
    
    # Remove virtual environment
    if [[ -d "${RESOURCE_DIR}/venv" ]]; then
        echo "Removing Python virtual environment..."
        rm -rf "${RESOURCE_DIR}/venv"
    fi
    
    # Remove data unless --keep-data specified
    if [[ "$keep_data" != "--keep-data" ]]; then
        echo "Removing election data..."
        rm -rf "${RESOURCE_DIR}/data"
    else
        echo "Keeping election data as requested..."
    fi
    
    # Remove installation marker
    rm -f "${RESOURCE_DIR}/.installed"
    rm -f "${RESOURCE_DIR}/.pid"
    
    echo "ElectionGuard uninstalled successfully!"
    return 0
}

# Start ElectionGuard service
start_electionguard() {
    local wait="${1:-}"
    
    # Check if already running
    if [[ -f "${RESOURCE_DIR}/.pid" ]]; then
        local pid=$(cat "${RESOURCE_DIR}/.pid")
        if kill -0 "$pid" 2>/dev/null; then
            echo "ElectionGuard is already running (PID: $pid)"
            return 2
        fi
    fi
    
    # Check if installed
    if [[ ! -f "${RESOURCE_DIR}/.installed" ]]; then
        echo "Error: ElectionGuard is not installed. Run 'manage install' first."
        return 1
    fi
    
    echo "Starting ElectionGuard service..."
    
    # Activate virtual environment if available
    if [[ -f "${RESOURCE_DIR}/venv/bin/activate" ]]; then
        source "${RESOURCE_DIR}/venv/bin/activate"
    fi
    
    # Start the API server
    nohup python3 "${RESOURCE_DIR}/lib/api_server.py" \
        --port "${ELECTIONGUARD_PORT}" \
        --data-dir "${RESOURCE_DIR}/data" \
        > "${RESOURCE_DIR}/logs/electionguard.log" 2>&1 &
    
    local pid=$!
    echo "$pid" > "${RESOURCE_DIR}/.pid"
    
    # Wait for service to be ready if requested
    if [[ "$wait" == "--wait" ]]; then
        echo "Waiting for ElectionGuard to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if timeout 5 curl -sf "http://localhost:${ELECTIONGUARD_PORT}/health" > /dev/null 2>&1; then
                echo "ElectionGuard is ready!"
                return 0
            fi
            sleep 1
            ((attempt++))
        done
        
        echo "Warning: ElectionGuard started but health check timed out"
    else
        echo "ElectionGuard started (PID: $pid)"
    fi
    
    return 0
}

# Stop ElectionGuard service
stop_electionguard() {
    local force="${1:-}"
    
    if [[ ! -f "${RESOURCE_DIR}/.pid" ]]; then
        echo "ElectionGuard is not running"
        return 2
    fi
    
    local pid=$(cat "${RESOURCE_DIR}/.pid")
    
    if ! kill -0 "$pid" 2>/dev/null; then
        echo "ElectionGuard process not found, cleaning up..."
        rm -f "${RESOURCE_DIR}/.pid"
        return 0
    fi
    
    echo "Stopping ElectionGuard (PID: $pid)..."
    
    # Try graceful shutdown
    kill -TERM "$pid" 2>/dev/null || true
    
    # Wait for shutdown
    local max_wait=10
    local waited=0
    while kill -0 "$pid" 2>/dev/null && [[ $waited -lt $max_wait ]]; do
        sleep 1
        ((waited++))
    done
    
    # Force kill if still running
    if kill -0 "$pid" 2>/dev/null; then
        if [[ "$force" == "--force" ]]; then
            echo "Force stopping ElectionGuard..."
            kill -KILL "$pid" 2>/dev/null || true
        else
            echo "Warning: ElectionGuard did not stop gracefully"
        fi
    fi
    
    rm -f "${RESOURCE_DIR}/.pid"
    echo "ElectionGuard stopped"
    return 0
}

# Restart ElectionGuard service
restart_electionguard() {
    echo "Restarting ElectionGuard..."
    stop_electionguard --force
    sleep 2
    start_electionguard "$@"
}

# Show service status
show_status() {
    local format="${1:-}"
    
    local status="stopped"
    local pid=""
    local health="unknown"
    
    if [[ -f "${RESOURCE_DIR}/.pid" ]]; then
        pid=$(cat "${RESOURCE_DIR}/.pid")
        if kill -0 "$pid" 2>/dev/null; then
            status="running"
            
            # Check health endpoint
            if timeout 5 curl -sf "http://localhost:${ELECTIONGUARD_PORT}/health" > /dev/null 2>&1; then
                health="healthy"
            else
                health="unhealthy"
            fi
        fi
    fi
    
    if [[ "$format" == "--json" ]]; then
        echo "{\"status\":\"$status\",\"pid\":\"$pid\",\"health\":\"$health\",\"port\":\"${ELECTIONGUARD_PORT}\"}"
    else
        echo "ElectionGuard Status:"
        echo "  Status: $status"
        [[ -n "$pid" ]] && echo "  PID: $pid"
        echo "  Health: $health"
        echo "  Port: ${ELECTIONGUARD_PORT}"
        echo "  Vault: ${ELECTIONGUARD_VAULT_ENABLED}"
        echo "  Database: ${ELECTIONGUARD_DB_ENABLED}"
    fi
    
    return 0
}

# Show service logs
show_logs() {
    local lines="${1:-50}"
    local follow="${2:-}"
    
    local log_file="${RESOURCE_DIR}/logs/electionguard.log"
    
    if [[ ! -f "$log_file" ]]; then
        echo "No logs found. Service may not have been started yet."
        return 0
    fi
    
    if [[ "$follow" == "--follow" || "$follow" == "-f" ]]; then
        tail -f "$log_file"
    else
        tail -n "$lines" "$log_file"
    fi
}

# Handle content operations
handle_content() {
    local operation="${1:-list}"
    shift || true
    
    case "$operation" in
        list)
            echo "Available content operations:"
            echo "  create-election  - Create new election"
            echo "  generate-keys    - Generate guardian keys"
            echo "  encrypt-ballot   - Encrypt a ballot"
            echo "  compute-tally    - Compute election tally"
            echo "  verify          - Verify ballot receipt"
            echo "  export          - Export election data"
            echo "  execute         - Run mock election"
            ;;
        create-election)
            create_election "$@"
            ;;
        generate-keys)
            generate_keys "$@"
            ;;
        encrypt-ballot)
            encrypt_ballot "$@"
            ;;
        compute-tally)
            compute_tally "$@"
            ;;
        verify)
            verify_ballot "$@"
            ;;
        export)
            export_data "$@"
            ;;
        execute)
            execute_mock_election "$@"
            ;;
        *)
            echo "Error: Unknown content operation: $operation"
            return 1
            ;;
    esac
}

# Execute mock election for testing
execute_mock_election() {
    echo "Running mock election..."
    
    # Check service is running
    if ! timeout 5 curl -sf "http://localhost:${ELECTIONGUARD_PORT}/health" > /dev/null 2>&1; then
        echo "Error: ElectionGuard service is not running"
        return 1
    fi
    
    echo "Mock election would be executed here"
    echo "This would:"
    echo "  1. Create an election with 5 guardians, threshold 3"
    echo "  2. Generate guardian keys"
    echo "  3. Create and encrypt 1000 sample ballots"
    echo "  4. Compute the tally"
    echo "  5. Verify random ballot samples"
    echo "  6. Export results"
    
    return 0
}
# Content operation functions for CLI

create_election() {
    local election_id="${1:-}"
    local name="${2:-Test Election}"
    local guardians="${3:-5}"
    local threshold="${4:-3}"
    
    if [[ -z "$election_id" ]]; then
        echo "Error: election_id required"
        echo "Usage: content create-election <election_id> [name] [guardians] [threshold]"
        return 1
    fi
    
    echo "Creating election: $election_id"
    
    local response=$(curl -s -X POST "http://localhost:${ELECTIONGUARD_PORT}/api/v1/election/create" \
        -H "Content-Type: application/json" \
        -d "{\"election_id\": \"$election_id\", \"name\": \"$name\", \"guardians\": $guardians, \"threshold\": $threshold}")
    
    echo "$response" | jq . 2>/dev/null || echo "$response"
}

generate_keys() {
    local election_id="${1:-}"
    
    if [[ -z "$election_id" ]]; then
        echo "Error: election_id required"
        echo "Usage: content generate-keys <election_id>"
        return 1
    fi
    
    echo "Generating keys for election: $election_id"
    
    local response=$(curl -s -X POST "http://localhost:${ELECTIONGUARD_PORT}/api/v1/keys/generate" \
        -H "Content-Type: application/json" \
        -d "{\"election_id\": \"$election_id\"}")
    
    echo "$response" | jq . 2>/dev/null || echo "$response"
}

encrypt_ballot() {
    local election_id="${1:-}"
    local ballot="${2:-}"
    
    if [[ -z "$election_id" ]] || [[ -z "$ballot" ]]; then
        echo "Error: election_id and ballot required"
        echo "Usage: content encrypt-ballot <election_id> <ballot_json>"
        return 1
    fi
    
    echo "Encrypting ballot for election: $election_id"
    
    local response=$(curl -s -X POST "http://localhost:${ELECTIONGUARD_PORT}/api/v1/ballot/encrypt" \
        -H "Content-Type: application/json" \
        -d "{\"election_id\": \"$election_id\", \"ballot\": $ballot}")
    
    echo "$response" | jq . 2>/dev/null || echo "$response"
}

compute_tally() {
    local election_id="${1:-}"
    
    if [[ -z "$election_id" ]]; then
        echo "Error: election_id required"
        echo "Usage: content compute-tally <election_id>"
        return 1
    fi
    
    echo "Computing tally for election: $election_id"
    
    local response=$(curl -s -X POST "http://localhost:${ELECTIONGUARD_PORT}/api/v1/tally/compute" \
        -H "Content-Type: application/json" \
        -d "{\"election_id\": \"$election_id\"}")
    
    echo "$response" | jq . 2>/dev/null || echo "$response"
}

verify_ballot() {
    local election_id="${1:-}"
    local receipt="${2:-}"
    
    if [[ -z "$election_id" ]] || [[ -z "$receipt" ]]; then
        echo "Error: election_id and receipt required"
        echo "Usage: content verify <election_id> <receipt>"
        return 1
    fi
    
    echo "Verifying ballot receipt: $receipt"
    
    local response=$(curl -s -X GET "http://localhost:${ELECTIONGUARD_PORT}/api/v1/verify/ballot?election_id=$election_id&receipt=$receipt")
    
    echo "$response" | jq . 2>/dev/null || echo "$response"
}

export_data() {
    local election_id="${1:-}"
    local target="${2:-postgres}"  # postgres or questdb
    
    if [[ -z "$election_id" ]]; then
        echo "Error: election_id required"
        echo "Usage: content export <election_id> [postgres|questdb]"
        return 1
    fi
    
    echo "Exporting election $election_id to $target"
    
    local endpoint="postgres"
    if [[ "$target" == "questdb" ]]; then
        endpoint="questdb"
    fi
    
    local response=$(curl -s -X POST "http://localhost:${ELECTIONGUARD_PORT}/api/v1/export/$endpoint" \
        -H "Content-Type: application/json" \
        -d "{\"election_id\": \"$election_id\"}")
    
    echo "$response" | jq . 2>/dev/null || echo "$response"
}
