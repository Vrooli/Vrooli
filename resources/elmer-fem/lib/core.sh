#!/bin/bash
# Elmer FEM Core Functions - v2.0 Contract Implementation

set -euo pipefail

# Runtime configuration
# Note: ELMER_PORT is imported from cli.sh
readonly ELMER_DATA_DIR="${VROOLI_DATA:-${HOME}/.vrooli/data}/elmer-fem"
readonly ELMER_LOG_FILE="${ELMER_DATA_DIR}/elmer.log"
readonly ELMER_PID_FILE="${ELMER_DATA_DIR}/elmer.pid"
readonly ELMER_CONTAINER_NAME="vrooli-elmer-fem"
readonly ELMER_IMAGE="vrooli/elmer-fem:latest"

# Ensure data directory exists
mkdir -p "${ELMER_DATA_DIR}"/{cases,meshes,results,temp}

# ============================================================================
# Info Command
# ============================================================================
core::info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "Elmer FEM Runtime Information:"
        echo "================================"
        jq -r 'to_entries | .[] | "\(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json" 2>/dev/null || {
            echo "ERROR: runtime.json not found"
            return 1
        }
    fi
}

# ============================================================================
# Lifecycle Management
# ============================================================================
core::manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            manage::install "$@"
            ;;
        start)
            manage::start "$@"
            ;;
        stop)
            manage::stop "$@"
            ;;
        restart)
            manage::restart "$@"
            ;;
        uninstall)
            manage::uninstall "$@"
            ;;
        *)
            echo "ERROR: Unknown manage subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

manage::install() {
    echo "Installing Elmer FEM resource..."
    
    # Check Docker availability
    if ! command -v docker &> /dev/null; then
        echo "ERROR: Docker is required but not installed" >&2
        return 1
    fi
    
    # Build Docker image
    echo "Building Elmer FEM Docker image..."
    if [[ -f "${SCRIPT_DIR}/Dockerfile" ]]; then
        docker build -t "${ELMER_IMAGE}" "${SCRIPT_DIR}" || {
            echo "ERROR: Failed to build Docker image" >&2
            return 1
        }
    else
        echo "ERROR: Dockerfile not found at ${SCRIPT_DIR}/Dockerfile" >&2
        return 1
    fi
    
    # Create example cases
    manage::setup_examples
    
    # Initialize configuration
    manage::init_config
    
    echo "Elmer FEM installation complete"
    return 0
}

manage::build_custom_image() {
    # Create Dockerfile for custom build if official fails
    cat > "${ELMER_DATA_DIR}/Dockerfile" << 'EOF'
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    gfortran \
    libopenmpi-dev \
    openmpi-bin \
    liblapack-dev \
    libblas-dev \
    python3 \
    python3-pip \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Build Elmer from source
WORKDIR /opt
RUN git clone https://github.com/ElmerCSC/elmerfem.git && \
    cd elmerfem && \
    mkdir build && cd build && \
    cmake .. -DWITH_MPI=ON && \
    make -j$(nproc) && \
    make install

# Add Python API dependencies
RUN pip3 install flask fastapi uvicorn numpy pandas

WORKDIR /workspace
EXPOSE 8192

CMD ["python3", "/workspace/api_server.py"]
EOF

    docker build -t elmer/elmerfem:custom "${ELMER_DATA_DIR}"
    ELMER_IMAGE="elmer/elmerfem:custom"
}

manage::setup_examples() {
    echo "Setting up canonical examples..."
    
    # Create heat transfer example
    mkdir -p "${ELMER_DATA_DIR}/cases/heat_transfer"
    cat > "${ELMER_DATA_DIR}/cases/heat_transfer/case.sif" << 'EOF'
Header
  Mesh DB "." "mesh"
End

Simulation
  Max Output Level = 5
  Coordinate System = Cartesian 2D
  Simulation Type = Steady State
  Output Intervals = 1
End

Body 1
  Equation = 1
  Material = 1
End

Equation 1
  Active Solvers(1) = 1
End

Solver 1
  Equation = Heat Equation
  Procedure = "HeatSolve" "HeatSolver"
  Variable = Temperature
  Linear System Solver = Iterative
  Linear System Iterative Method = BiCGStab
  Linear System Max Iterations = 500
End

Material 1
  Heat Conductivity = 1.0
  Density = 1.0
  Heat Capacity = 1.0
End

Boundary Condition 1
  Target Boundaries = 1
  Temperature = 100.0
End

Boundary Condition 2
  Target Boundaries = 2
  Temperature = 0.0
End
EOF
    
    echo "Examples configured in ${ELMER_DATA_DIR}/cases/"
}

manage::init_config() {
    # Store runtime configuration
    cat > "${ELMER_DATA_DIR}/config.json" << EOF
{
    "port": ${ELMER_PORT},
    "mpi_processes": 4,
    "max_memory": "4G",
    "data_dir": "${ELMER_DATA_DIR}",
    "version": "9.0"
}
EOF
}

manage::start() {
    local wait_flag="${1:-}"
    
    echo "Starting Elmer FEM service on port ${ELMER_PORT}..."
    
    # Check if already running
    if docker ps --format '{{.Names}}' | grep -q "^${ELMER_CONTAINER_NAME}$"; then
        echo "Elmer FEM is already running"
        return 0
    fi
    
    # Start container with API server
    docker run -d \
        --name "${ELMER_CONTAINER_NAME}" \
        -p "${ELMER_PORT}:8192" \
        -v "${ELMER_DATA_DIR}/cases:/workspace/cases" \
        -v "${ELMER_DATA_DIR}/meshes:/workspace/meshes" \
        -v "${ELMER_DATA_DIR}/results:/workspace/results" \
        -v "${SCRIPT_DIR}/lib/api_server.py:/workspace/api_server.py" \
        -e "ELMER_FEM_PORT=8192" \
        -e "ELMER_FEM_DATA_DIR=/workspace" \
        "${ELMER_IMAGE}"
    
    # Save PID
    docker inspect -f '{{.State.Pid}}' "${ELMER_CONTAINER_NAME}" > "${ELMER_PID_FILE}"
    
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "Waiting for Elmer FEM to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if timeout 5 curl -sf "http://localhost:${ELMER_PORT}/health" &>/dev/null; then
                echo "Elmer FEM is ready"
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        echo "ERROR: Elmer FEM failed to start within timeout" >&2
        return 1
    fi
    
    echo "Elmer FEM started successfully"
    return 0
}

manage::stop() {
    echo "Stopping Elmer FEM service..."
    
    if docker ps --format '{{.Names}}' | grep -q "^${ELMER_CONTAINER_NAME}$"; then
        docker stop "${ELMER_CONTAINER_NAME}" || true
        docker rm "${ELMER_CONTAINER_NAME}" || true
        rm -f "${ELMER_PID_FILE}"
        echo "Elmer FEM stopped"
    else
        echo "Elmer FEM is not running"
    fi
    
    return 0
}

manage::restart() {
    manage::stop
    sleep 2
    manage::start "$@"
}

manage::uninstall() {
    local keep_data="${1:-}"
    
    echo "Uninstalling Elmer FEM resource..."
    
    # Stop service
    manage::stop
    
    # Remove Docker image
    docker rmi "${ELMER_IMAGE}" 2>/dev/null || true
    
    # Remove data unless --keep-data specified
    if [[ "$keep_data" != "--keep-data" ]]; then
        echo "Removing data directory..."
        rm -rf "${ELMER_DATA_DIR}"
    fi
    
    echo "Elmer FEM uninstalled"
    return 0
}

# ============================================================================
# Testing
# ============================================================================
core::test() {
    local phase="${1:-all}"
    
    # Delegate to test.sh
    "${SCRIPT_DIR}/lib/test.sh" "$phase"
}

# ============================================================================
# Content Management
# ============================================================================
core::content() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        add)
            content::add "$@"
            ;;
        list)
            content::list "$@"
            ;;
        get)
            content::get "$@"
            ;;
        remove)
            content::remove "$@"
            ;;
        execute)
            content::execute "$@"
            ;;
        *)
            echo "ERROR: Unknown content subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

content::list() {
    echo "Available Elmer FEM cases:"
    echo "=========================="
    ls -la "${ELMER_DATA_DIR}/cases/" 2>/dev/null || echo "No cases found"
}

content::execute() {
    local case_name="${1:-heat_transfer}"
    
    echo "Executing Elmer simulation: $case_name"
    
    # Call API to execute
    curl -X POST "http://localhost:${ELMER_PORT}/case/solve" \
        -H "Content-Type: application/json" \
        -d "{\"case\": \"$case_name\"}" || {
        echo "ERROR: Failed to execute simulation" >&2
        return 1
    }
}

content::add() {
    local file="${1:-}"
    local name="${2:-}"
    
    if [[ -z "$file" ]]; then
        echo "ERROR: File path required" >&2
        return 1
    fi
    
    if [[ -z "$name" ]]; then
        name=$(basename "$file" .sif)
    fi
    
    cp -r "$file" "${ELMER_DATA_DIR}/cases/$name/"
    echo "Added case: $name"
}

content::get() {
    local case_name="${1:-}"
    
    if [[ -z "$case_name" ]]; then
        echo "ERROR: Case name required" >&2
        return 1
    fi
    
    local result_file="${ELMER_DATA_DIR}/results/${case_name}/results.vtu"
    
    if [[ -f "$result_file" ]]; then
        echo "$result_file"
    else
        echo "ERROR: Results not found for case: $case_name" >&2
        return 1
    fi
}

content::remove() {
    local case_name="${1:-}"
    
    if [[ -z "$case_name" ]]; then
        echo "ERROR: Case name required" >&2
        return 1
    fi
    
    rm -rf "${ELMER_DATA_DIR}/cases/${case_name}"
    rm -rf "${ELMER_DATA_DIR}/results/${case_name}"
    echo "Removed case: $case_name"
}

# ============================================================================
# Status and Monitoring
# ============================================================================
core::status() {
    local format="${1:-text}"
    
    local status="stopped"
    local pid=""
    
    if docker ps --format '{{.Names}}' | grep -q "^${ELMER_CONTAINER_NAME}$"; then
        status="running"
        pid=$(docker inspect -f '{{.State.Pid}}' "${ELMER_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
    fi
    
    if [[ "$format" == "--json" ]]; then
        cat << EOF
{
    "service": "elmer-fem",
    "status": "${status}",
    "port": ${ELMER_PORT},
    "pid": "${pid}",
    "data_dir": "${ELMER_DATA_DIR}",
    "container": "${ELMER_CONTAINER_NAME}"
}
EOF
    else
        echo "Elmer FEM Status"
        echo "================"
        echo "Status: ${status}"
        echo "Port: ${ELMER_PORT}"
        echo "PID: ${pid}"
        echo "Data: ${ELMER_DATA_DIR}"
        echo "Container: ${ELMER_CONTAINER_NAME}"
    fi
}

core::logs() {
    local lines="${1:-50}"
    
    if docker ps --format '{{.Names}}' | grep -q "^${ELMER_CONTAINER_NAME}$"; then
        docker logs --tail "$lines" "${ELMER_CONTAINER_NAME}"
    else
        echo "Elmer FEM is not running"
        return 1
    fi
}