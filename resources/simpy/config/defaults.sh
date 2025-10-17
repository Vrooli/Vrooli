#!/usr/bin/env bash
# SimPy configuration defaults

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SIMPY_CONFIG_DIR="${APP_ROOT}/resources/simpy/config"

# Source variable utilities FIRST
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# SimPy Configuration
export SIMPY_NAME="simpy"
export SIMPY_DATA_DIR="${var_DATA_DIR}/resources/simpy"
export SIMPY_VENV_DIR="${SIMPY_DATA_DIR}/venv"
export SIMPY_EXAMPLES_DIR="${SIMPY_DATA_DIR}/examples"
export SIMPY_MODELS_DIR="${SIMPY_DATA_DIR}/models"
export SIMPY_RESULTS_DIR="${SIMPY_DATA_DIR}/results"
export SIMPY_LOG_FILE="${SIMPY_DATA_DIR}/simpy.log"

# Python configuration
export SIMPY_PYTHON_VERSION="3.12"
export SIMPY_PACKAGES="simpy numpy pandas matplotlib scipy networkx"
export SIMPY_EXTRA_PACKAGES="plotly seaborn jupyter ipython"

# Service configuration
export SIMPY_HOST="localhost"
export SIMPY_PORT="9510"  # Unique port for SimPy service
export SIMPY_BASE_URL="http://${SIMPY_HOST}:${SIMPY_PORT}"

# Simulation defaults
export SIMPY_DEFAULT_TIMEOUT="3600"  # 1 hour max simulation time
export SIMPY_DEFAULT_SEED="42"
export SIMPY_ENABLE_VISUALIZATION="true"
export SIMPY_ENABLE_LOGGING="true"

#######################################
# Export SimPy configuration
#######################################
simpy::export_config() {
    # Already exported above, but this function ensures consistency
    export SIMPY_NAME SIMPY_DATA_DIR SIMPY_VENV_DIR SIMPY_EXAMPLES_DIR
    export SIMPY_MODELS_DIR SIMPY_RESULTS_DIR SIMPY_LOG_FILE
    export SIMPY_PYTHON_VERSION SIMPY_PACKAGES SIMPY_EXTRA_PACKAGES
    export SIMPY_HOST SIMPY_PORT SIMPY_BASE_URL
    export SIMPY_DEFAULT_TIMEOUT SIMPY_DEFAULT_SEED
    export SIMPY_ENABLE_VISUALIZATION SIMPY_ENABLE_LOGGING
}