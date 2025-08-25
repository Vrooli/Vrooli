#!/bin/bash

# SimPy Resource - Common Functions
set -euo pipefail

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SIMPY_LIB_DIR="${APP_ROOT}/resources/simpy/lib"
SIMPY_DIR="${APP_ROOT}/resources/simpy"
SIMPY_CONFIG_DIR="${APP_ROOT}/resources/simpy/config"
SIMPY_EXAMPLES_DIR="${APP_ROOT}/resources/simpy/examples"

# Source shared utilities
source "$SIMPY_LIB_DIR/../../../../lib/utils/var.sh"
source "$SIMPY_LIB_DIR/../../../../lib/utils/format.sh"
source "$SIMPY_LIB_DIR/../../../../lib/utils/log.sh"

# Configuration
SIMPY_RESOURCE_NAME="simpy"
SIMPY_CATEGORY="execution"
SIMPY_DESCRIPTION="Discrete event simulation framework for modeling complex systems"

# Docker configuration
SIMPY_CONTAINER_NAME="simpy-main"
SIMPY_IMAGE_NAME="vrooli/simpy:latest"

# Data directories
SIMPY_SIMULATIONS_DIR="$var_DATA_DIR/simpy/simulations"
SIMPY_OUTPUTS_DIR="$var_DATA_DIR/simpy/outputs"
SIMPY_LOGS_DIR="$var_DATA_DIR/simpy/logs"

# Get SimPy version
simpy::get_version() {
    if docker image inspect "$SIMPY_IMAGE_NAME" &>/dev/null; then
        docker run --rm "$SIMPY_IMAGE_NAME" python -c "import simpy; print(simpy.__version__)" 2>/dev/null || echo "unknown"
    else
        echo "not_installed"
    fi
}

# Check if SimPy is installed
simpy::is_installed() {
    docker image inspect "$SIMPY_IMAGE_NAME" &>/dev/null
}

# Check if SimPy service is healthy
simpy::is_healthy() {
    if ! simpy::is_installed; then
        return 1
    fi
    
    # Test basic SimPy functionality
    docker run --rm "$SIMPY_IMAGE_NAME" python -c "
import simpy
env = simpy.Environment()
env.run(until=1)
print('healthy')
" &>/dev/null
}

# Get simulation count
simpy::get_simulation_count() {
    if [[ -d "$SIMPY_SIMULATIONS_DIR" ]]; then
        find "$SIMPY_SIMULATIONS_DIR" -name "*.py" -type f 2>/dev/null | wc -l
    else
        echo "0"
    fi
}

# Get output count
simpy::get_output_count() {
    if [[ -d "$SIMPY_OUTPUTS_DIR" ]]; then
        find "$SIMPY_OUTPUTS_DIR" -type f 2>/dev/null | wc -l
    else
        echo "0"
    fi
}