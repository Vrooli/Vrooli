#!/usr/bin/env bash
# SimPy core functionality

# Get script directory
SIMPY_CORE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
# shellcheck disable=SC1091
source "${SIMPY_CORE_DIR}/../config/defaults.sh"
# shellcheck disable=SC1091
source "${SIMPY_CORE_DIR}/../../../../lib/utils/log.sh"

# Export configuration
simpy::export_config

#######################################
# Initialize SimPy environment
#######################################
simpy::init() {
    # Create necessary directories
    mkdir -p "$SIMPY_DATA_DIR" \
             "$SIMPY_EXAMPLES_DIR" \
             "$SIMPY_MODELS_DIR" \
             "$SIMPY_RESULTS_DIR"
    
    # Initialize log file
    touch "$SIMPY_LOG_FILE"
    
    return 0
}

#######################################
# Check if SimPy is installed
#######################################
simpy::is_installed() {
    # Check if service file exists and SimPy module is available
    [[ -f "$SIMPY_DATA_DIR/simpy-service.py" ]] && \
    python3 -c "import simpy" 2>/dev/null
}

#######################################
# Check if SimPy service is running
#######################################
simpy::is_running() {
    pgrep -f "simpy-service.py" >/dev/null 2>&1
}

#######################################
# Get SimPy service PID
#######################################
simpy::get_pid() {
    pgrep -f "simpy-service.py" | head -1
}

#######################################
# Test SimPy connection
#######################################
simpy::test_connection() {
    curl -s -f "http://${SIMPY_HOST}:${SIMPY_PORT}/health" >/dev/null 2>&1
}

#######################################
# Run a SimPy simulation
#######################################
simpy::run_simulation() {
    local script_path="$1"
    local output_path="${2:-${SIMPY_RESULTS_DIR}/output.json}"
    
    if [[ ! -f "$script_path" ]]; then
        log::error "Simulation script not found: $script_path"
        return 1
    fi
    
    # Run simulation in virtual environment
    "${SIMPY_VENV_DIR}/bin/python" "$script_path" > "$output_path" 2>&1
}

#######################################
# List available examples
#######################################
simpy::list_examples() {
    if [[ -d "$SIMPY_EXAMPLES_DIR" ]]; then
        find "$SIMPY_EXAMPLES_DIR" -name "*.py" -type f | while read -r file; do
            basename "$file" .py
        done
    fi
}

#######################################
# Get SimPy version
#######################################
simpy::get_version() {
    # Try virtual env first, fall back to system Python
    if [[ -f "$SIMPY_VENV_DIR/bin/python" ]]; then
        "${SIMPY_VENV_DIR}/bin/python" -c "import simpy; print(simpy.__version__)" 2>/dev/null || \
        python3 -c "import simpy; print(simpy.__version__)" 2>/dev/null || \
        echo "not_installed"
    else
        python3 -c "import simpy; print(simpy.__version__)" 2>/dev/null || \
        echo "not_installed"
    fi
}
