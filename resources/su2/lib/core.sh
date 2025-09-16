#!/bin/bash
# SU2 Core Functionality

# Ensure configuration is loaded
[[ -z "$SU2_PORT" ]] && source "${SCRIPT_DIR}/config/defaults.sh"

# Install SU2 and dependencies
install_su2() {
    local force="${1:-}"
    
    echo "Installing SU2 CFD platform..."
    
    # Create data directories
    mkdir -p "${SU2_MESHES_DIR}" "${SU2_CONFIGS_DIR}" "${SU2_RESULTS_DIR}" "${SU2_CACHE_DIR}"
    
    # Build Docker image with MPI support
    create_dockerfile
    
    echo "Building SU2 Docker image..."
    docker build -t "vrooli/su2:latest" "${SCRIPT_DIR}" || {
        echo "Error: Failed to build SU2 Docker image" >&2
        return 1
    }
    
    # Download example cases
    download_examples
    
    echo "SU2 installation complete"
    return 0
}

# Create Dockerfile for SU2
create_dockerfile() {
    cat > "${SCRIPT_DIR}/Dockerfile" << 'EOF'
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    python3 \
    python3-pip \
    openmpi-bin \
    libopenmpi-dev \
    git \
    wget \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
RUN pip3 install flask flask-cors numpy matplotlib pandas

# Download and compile SU2
WORKDIR /opt
RUN git clone --depth 1 --branch v7.5.1 https://github.com/su2code/SU2.git su2

WORKDIR /opt/su2
RUN ./meson.py build -Denable-mpi=true -Denable-autodiff=false \
    && ./ninja -C build install

# Set environment variables
ENV SU2_RUN=/opt/su2/bin
ENV PATH="${SU2_RUN}:${PATH}"
ENV PYTHONPATH="/opt/su2/bin:${PYTHONPATH}"

# Create working directory
WORKDIR /workspace

# Copy API server
COPY lib/api_server.py /app/api_server.py

# Expose API port
EXPOSE 9514

# Run API server
CMD ["python3", "/app/api_server.py"]
EOF
}

# Start SU2 service
start_su2() {
    local wait="${1:-}"
    
    echo "Starting SU2 service..."
    
    # Check if already running
    if docker ps --format '{{.Names}}' | grep -q "^${SU2_CONTAINER_NAME}$"; then
        echo "SU2 is already running"
        return 0
    fi
    
    # Start container
    docker run -d \
        --name "${SU2_CONTAINER_NAME}" \
        -p "${SU2_PORT}:9514" \
        -v "${SU2_MESHES_DIR}:/workspace/meshes:ro" \
        -v "${SU2_CONFIGS_DIR}:/workspace/configs:ro" \
        -v "${SU2_RESULTS_DIR}:/workspace/results:rw" \
        -v "${SU2_CACHE_DIR}:/workspace/cache:rw" \
        -e OMP_NUM_THREADS="${SU2_OMP_NUM_THREADS}" \
        --cpus="${SU2_CPU_LIMIT}" \
        --memory="${SU2_MEMORY_LIMIT}" \
        --restart unless-stopped \
        "vrooli/su2:latest" || {
            echo "Error: Failed to start SU2 container" >&2
            return 1
        }
    
    if [[ "$wait" == "--wait" ]]; then
        echo "Waiting for SU2 to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if timeout 5 curl -sf "http://localhost:${SU2_PORT}/health" > /dev/null 2>&1; then
                echo "SU2 is ready"
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        echo "Error: SU2 failed to start within timeout" >&2
        return 1
    fi
    
    echo "SU2 service started"
    return 0
}

# Stop SU2 service
stop_su2() {
    echo "Stopping SU2 service..."
    
    if ! docker ps --format '{{.Names}}' | grep -q "^${SU2_CONTAINER_NAME}$"; then
        echo "SU2 is not running"
        return 0
    fi
    
    docker stop "${SU2_CONTAINER_NAME}" --time="${SU2_SHUTDOWN_TIMEOUT}" || {
        echo "Warning: Failed to stop SU2 gracefully" >&2
    }
    
    docker rm "${SU2_CONTAINER_NAME}" 2>/dev/null || true
    
    echo "SU2 service stopped"
    return 0
}

# Restart SU2 service
restart_su2() {
    stop_su2
    sleep 2
    start_su2 "$@"
}

# Uninstall SU2
uninstall_su2() {
    echo "Uninstalling SU2..."
    
    # Stop service
    stop_su2
    
    # Remove Docker image
    docker rmi "vrooli/su2:latest" 2>/dev/null || true
    
    # Optional: Remove data directories
    read -p "Remove data directories? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "${SU2_DATA_DIR}"
    fi
    
    echo "SU2 uninstalled"
    return 0
}

# Show service status
show_status() {
    local format="${1:-text}"
    
    if docker ps --format '{{.Names}}' | grep -q "^${SU2_CONTAINER_NAME}$"; then
        if [[ "$format" == "--json" ]]; then
            echo '{"status":"running","port":'${SU2_PORT}',"container":"'${SU2_CONTAINER_NAME}'"}'
        else
            echo "SU2 Status: Running"
            echo "  Port: ${SU2_PORT}"
            echo "  Container: ${SU2_CONTAINER_NAME}"
            echo "  Health: $(check_health)"
        fi
    else
        if [[ "$format" == "--json" ]]; then
            echo '{"status":"stopped"}'
        else
            echo "SU2 Status: Stopped"
        fi
    fi
}

# Check service health
check_health() {
    if timeout "${SU2_HEALTH_TIMEOUT}" curl -sf "http://localhost:${SU2_PORT}/health" > /dev/null 2>&1; then
        echo "Healthy"
        return 0
    else
        echo "Unhealthy"
        return 1
    fi
}

# Show service logs
show_logs() {
    local lines="${1:-50}"
    
    if docker ps -a --format '{{.Names}}' | grep -q "^${SU2_CONTAINER_NAME}$"; then
        docker logs "${SU2_CONTAINER_NAME}" --tail "$lines"
    else
        echo "No SU2 container found" >&2
        return 1
    fi
}

# Content list operation
content_list() {
    local type="${1:-all}"
    
    case "$type" in
        meshes)
            ls -la "${SU2_MESHES_DIR}" 2>/dev/null || echo "No meshes found"
            ;;
        configs)
            ls -la "${SU2_CONFIGS_DIR}" 2>/dev/null || echo "No configs found"
            ;;
        results)
            ls -la "${SU2_RESULTS_DIR}" 2>/dev/null || echo "No results found"
            ;;
        all|*)
            echo "=== Meshes ==="
            ls "${SU2_MESHES_DIR}" 2>/dev/null || echo "None"
            echo -e "\n=== Configs ==="
            ls "${SU2_CONFIGS_DIR}" 2>/dev/null || echo "None"
            echo -e "\n=== Results ==="
            ls "${SU2_RESULTS_DIR}" 2>/dev/null || echo "None"
            ;;
    esac
}

# Content add operation
content_add() {
    local file="${1:-}"
    local type="${2:-auto}"
    
    if [[ -z "$file" ]] || [[ ! -f "$file" ]]; then
        echo "Error: File not found: $file" >&2
        return 1
    fi
    
    local filename=$(basename "$file")
    local dest_dir=""
    
    # Determine destination based on file extension or type
    case "$type" in
        mesh|meshes)
            dest_dir="${SU2_MESHES_DIR}"
            ;;
        config|configs)
            dest_dir="${SU2_CONFIGS_DIR}"
            ;;
        auto|*)
            if [[ "$filename" =~ \.(su2|cgns|mesh)$ ]]; then
                dest_dir="${SU2_MESHES_DIR}"
            elif [[ "$filename" =~ \.cfg$ ]]; then
                dest_dir="${SU2_CONFIGS_DIR}"
            else
                echo "Error: Cannot determine file type for: $filename" >&2
                return 1
            fi
            ;;
    esac
    
    cp "$file" "$dest_dir/" || {
        echo "Error: Failed to copy file" >&2
        return 1
    }
    
    echo "Added $filename to $(basename $dest_dir)"
    return 0
}

# Content get operation
content_get() {
    local id="${1:-}"
    local format="${2:-csv}"
    
    if [[ -z "$id" ]]; then
        echo "Error: Result ID required" >&2
        return 1
    fi
    
    local result_file="${SU2_RESULTS_DIR}/${id}"
    
    if [[ ! -f "$result_file" ]] && [[ ! -d "$result_file" ]]; then
        echo "Error: Result not found: $id" >&2
        return 1
    fi
    
    case "$format" in
        csv)
            find "$result_file" -name "*.csv" -exec cat {} \;
            ;;
        vtk)
            find "$result_file" -name "*.vtk" -exec echo "VTK file: {}" \;
            ;;
        all)
            ls -la "$result_file"
            ;;
        *)
            echo "Error: Unknown format: $format" >&2
            return 1
            ;;
    esac
}

# Content remove operation
content_remove() {
    local item="${1:-}"
    
    if [[ -z "$item" ]]; then
        echo "Error: Item name required" >&2
        return 1
    fi
    
    # Try to find and remove the item
    local removed=false
    
    for dir in "${SU2_MESHES_DIR}" "${SU2_CONFIGS_DIR}" "${SU2_RESULTS_DIR}"; do
        if [[ -e "${dir}/${item}" ]]; then
            rm -rf "${dir}/${item}"
            echo "Removed ${item} from $(basename $dir)"
            removed=true
        fi
    done
    
    if [[ "$removed" == "false" ]]; then
        echo "Error: Item not found: $item" >&2
        return 1
    fi
    
    return 0
}

# Content execute operation - run CFD simulation
content_execute() {
    local mesh="${1:-}"
    local config="${2:-}"
    
    if [[ -z "$mesh" ]] || [[ -z "$config" ]]; then
        echo "Usage: $0 content execute <mesh_file> <config_file>" >&2
        return 1
    fi
    
    # Check if service is running
    if ! docker ps --format '{{.Names}}' | grep -q "^${SU2_CONTAINER_NAME}$"; then
        echo "Error: SU2 service is not running" >&2
        return 1
    fi
    
    # Submit simulation via API
    local response=$(curl -sf -X POST "http://localhost:${SU2_PORT}/api/simulate" \
        -H "Content-Type: application/json" \
        -d "{\"mesh\":\"$mesh\",\"config\":\"$config\"}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to submit simulation" >&2
        return 1
    fi
    
    echo "$response"
    return 0
}

# Download example cases
download_examples() {
    echo "Downloading SU2 example cases..."
    
    # NACA0012 example
    local naca_mesh="https://raw.githubusercontent.com/su2code/SU2/master/meshes/airfoil/mesh_NACA0012_inv.su2"
    local naca_config="https://raw.githubusercontent.com/su2code/SU2/master/config_templates/euler/inv_NACA0012.cfg"
    
    wget -q -O "${SU2_MESHES_DIR}/naca0012.su2" "$naca_mesh" || true
    wget -q -O "${SU2_CONFIGS_DIR}/naca0012.cfg" "$naca_config" || true
    
    # Create simplified test config
    create_test_config
    
    echo "Example cases downloaded"
}

# Create test configuration
create_test_config() {
    cat > "${SU2_CONFIGS_DIR}/test.cfg" << 'EOF'
% Test configuration for NACA0012
SOLVER= EULER
MATH_PROBLEM= DIRECT
RESTART_SOL= NO
MACH_NUMBER= 0.8
AOA= 1.25
FREESTREAM_PRESSURE= 101325.0
FREESTREAM_TEMPERATURE= 288.15
MARKER_EULER= ( airfoil )
MARKER_FAR= ( farfield )
MARKER_PLOTTING= ( airfoil )
MARKER_MONITORING= ( airfoil )
CFL_NUMBER= 5.0
CFL_ADAPT= NO
ITER= 100
LINEAR_SOLVER= FGMRES
LINEAR_SOLVER_ERROR= 1E-6
LINEAR_SOLVER_ITER= 20
CONV_FIELD= RMS_DENSITY
CONV_RESIDUAL_MINVAL= -8
CONV_STARTITER= 10
CONV_CAUCHY_ELEMS= 100
CONV_CAUCHY_EPS= 1E-6
MESH_FILENAME= naca0012.su2
MESH_FORMAT= SU2
OUTPUT_FILES= (RESTART_ASCII, PARAVIEW, SURFACE_PARAVIEW)
OUTPUT_WRT_FREQ= 50
HISTORY_OUTPUT= YES
EOF
}