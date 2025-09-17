#!/usr/bin/env bash
# MEEP Core Library Functions

set -euo pipefail

# Ensure defaults are loaded
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fi
[[ -f "${SCRIPT_DIR}/config/defaults.sh" ]] && source "${SCRIPT_DIR}/config/defaults.sh"

#######################################
# Manage lifecycle commands
#######################################
meep::manage() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        install)
            meep::install "$@"
            ;;
        start)
            meep::start "$@"
            ;;
        stop)
            meep::stop "$@"
            ;;
        restart)
            meep::restart "$@"
            ;;
        uninstall)
            meep::uninstall "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand" >&2
            return 1
            ;;
    esac
}

#######################################
# Install MEEP and dependencies
#######################################
meep::install() {
    echo "Installing MEEP..."
    
    # Create data directories
    mkdir -p "${MEEP_DATA_DIR}"
    mkdir -p "${MEEP_RESULTS_DIR}"
    mkdir -p "${MEEP_TEMPLATES_DIR}"
    
    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        echo "Error: Docker is required but not installed" >&2
        return 1
    fi
    
    # Build Docker image
    echo "Building MEEP Docker image..."
    local dockerfile="${SCRIPT_DIR}/Dockerfile"
    
    if [[ ! -f "$dockerfile" ]]; then
        echo "Creating Dockerfile..."
        meep::create_dockerfile
    fi
    
    docker build -t "${MEEP_IMAGE_NAME}:${MEEP_IMAGE_TAG}" "${SCRIPT_DIR}" || {
        echo "Error: Failed to build Docker image" >&2
        return 1
    }
    
    # Install Python API server
    echo "Setting up Python API server..."
    meep::create_api_server
    
    # Install example templates
    echo "Installing simulation templates..."
    meep::install_templates
    
    echo "MEEP installation complete"
    return 0
}

#######################################
# Start MEEP service
#######################################
meep::start() {
    local wait_flag=false
    [[ "${1:-}" == "--wait" ]] && wait_flag=true
    
    echo "Starting MEEP service..."
    
    # Check if already running
    if meep::is_running; then
        echo "MEEP is already running"
        return 0
    fi
    
    # Start Docker container
    docker run -d \
        --name "${MEEP_CONTAINER_NAME}" \
        --network host \
        -v "${MEEP_DATA_DIR}:/data" \
        -e "MEEP_PORT=${MEEP_PORT}" \
        -e "OMP_NUM_THREADS=${MEEP_OMP_NUM_THREADS}" \
        -e "POSTGRES_HOST=localhost" \
        -e "POSTGRES_PORT=5433" \
        -e "MINIO_ENDPOINT=localhost:9000" \
        --memory="${MEEP_MEMORY_LIMIT}" \
        --cpus="${MEEP_CPU_LIMIT}" \
        "${MEEP_IMAGE_NAME}:${MEEP_IMAGE_TAG}" || {
        echo "Error: Failed to start MEEP container" >&2
        return 1
    }
    
    if [[ "$wait_flag" == "true" ]]; then
        echo "Waiting for MEEP to be ready..."
        meep::wait_for_health
    fi
    
    echo "MEEP service started on port ${MEEP_PORT}"
    return 0
}

#######################################
# Stop MEEP service
#######################################
meep::stop() {
    echo "Stopping MEEP service..."
    
    if ! meep::is_running; then
        echo "MEEP is not running"
        return 0
    fi
    
    docker stop "${MEEP_CONTAINER_NAME}" || {
        echo "Error: Failed to stop MEEP container" >&2
        return 1
    }
    
    docker rm "${MEEP_CONTAINER_NAME}" || {
        echo "Error: Failed to remove MEEP container" >&2
        return 1
    }
    
    echo "MEEP service stopped"
    return 0
}

#######################################
# Restart MEEP service
#######################################
meep::restart() {
    echo "Restarting MEEP service..."
    meep::stop
    sleep 2
    meep::start "$@"
}

#######################################
# Uninstall MEEP
#######################################
meep::uninstall() {
    echo "Uninstalling MEEP..."
    
    # Stop service if running
    meep::stop 2>/dev/null || true
    
    # Remove Docker image
    if docker images | grep -q "${MEEP_IMAGE_NAME}"; then
        docker rmi "${MEEP_IMAGE_NAME}:${MEEP_IMAGE_TAG}" || {
            echo "Warning: Failed to remove Docker image" >&2
        }
    fi
    
    # Optional: Remove data directory
    read -p "Remove data directory ${MEEP_DATA_DIR}? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "${MEEP_DATA_DIR}"
        echo "Data directory removed"
    fi
    
    echo "MEEP uninstalled"
    return 0
}

#######################################
# Show service status
#######################################
meep::status() {
    local json_flag=false
    [[ "${1:-}" == "--json" ]] && json_flag=true
    
    local status="stopped"
    local health="unhealthy"
    local container_id=""
    local uptime=""
    
    if meep::is_running; then
        status="running"
        container_id=$(docker ps -qf "name=${MEEP_CONTAINER_NAME}")
        uptime=$(docker ps -f "name=${MEEP_CONTAINER_NAME}" --format "{{.Status}}")
        
        # Check health
        if meep::check_health; then
            health="healthy"
        fi
    fi
    
    if [[ "$json_flag" == "true" ]]; then
        cat << EOF
{
    "status": "${status}",
    "health": "${health}",
    "container_id": "${container_id}",
    "uptime": "${uptime}",
    "port": ${MEEP_PORT},
    "mpi_processes": ${MEEP_MPI_PROCESSES}
}
EOF
    else
        echo "MEEP Status"
        echo "==========="
        echo "Status: ${status}"
        echo "Health: ${health}"
        [[ -n "$container_id" ]] && echo "Container: ${container_id}"
        [[ -n "$uptime" ]] && echo "Uptime: ${uptime}"
        echo "Port: ${MEEP_PORT}"
        echo "MPI Processes: ${MEEP_MPI_PROCESSES}"
    fi
}

#######################################
# View service logs
#######################################
meep::logs() {
    local tail_lines="${1:-100}"
    
    if ! meep::is_running; then
        echo "MEEP is not running" >&2
        return 1
    fi
    
    docker logs "${MEEP_CONTAINER_NAME}" --tail "$tail_lines"
}

#######################################
# Content management
#######################################
meep::content() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        list)
            meep::content_list
            ;;
        add)
            meep::content_add "$@"
            ;;
        get)
            meep::content_get "$@"
            ;;
        remove)
            meep::content_remove "$@"
            ;;
        execute)
            meep::content_execute "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand: $subcommand" >&2
            return 1
            ;;
    esac
}

#######################################
# List simulation templates
#######################################
meep::content_list() {
    echo "Available MEEP simulation templates:"
    echo "===================================="
    
    if [[ -d "${MEEP_TEMPLATES_DIR}" ]]; then
        for template in "${MEEP_TEMPLATES_DIR}"/*.py; do
            if [[ -f "$template" ]]; then
                basename "$template" .py
            fi
        done
    else
        echo "No templates found"
    fi
}

#######################################
# Add simulation template
#######################################
meep::content_add() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        echo "Error: Please provide a file to add" >&2
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "Error: File not found: $file" >&2
        return 1
    fi
    
    local filename=$(basename "$file")
    cp "$file" "${MEEP_TEMPLATES_DIR}/${filename}"
    echo "Template added: ${filename}"
}

#######################################
# Get simulation template
#######################################
meep::content_get() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Please provide a template name" >&2
        return 1
    fi
    
    local template="${MEEP_TEMPLATES_DIR}/${name}.py"
    
    if [[ ! -f "$template" ]]; then
        echo "Error: Template not found: $name" >&2
        return 1
    fi
    
    cat "$template"
}

#######################################
# Remove simulation template
#######################################
meep::content_remove() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Please provide a template name" >&2
        return 1
    fi
    
    local template="${MEEP_TEMPLATES_DIR}/${name}.py"
    
    if [[ ! -f "$template" ]]; then
        echo "Error: Template not found: $name" >&2
        return 1
    fi
    
    rm "$template"
    echo "Template removed: $name"
}

#######################################
# Execute simulation template
#######################################
meep::content_execute() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Please provide a template name" >&2
        return 1
    fi
    
    if ! meep::is_running; then
        echo "Error: MEEP service is not running" >&2
        return 1
    fi
    
    echo "Executing simulation: $name"
    
    # Submit simulation via API
    local response
    response=$(curl -sf -X POST \
        "http://localhost:${MEEP_PORT}/simulation/create" \
        -H "Content-Type: application/json" \
        -d "{\"template\": \"${name}\"}" 2>/dev/null) || {
        echo "Error: Failed to submit simulation" >&2
        return 1
    }
    
    local sim_id=$(echo "$response" | jq -r '.simulation_id')
    echo "Simulation ID: ${sim_id}"
    
    # Start simulation
    curl -sf -X POST \
        "http://localhost:${MEEP_PORT}/simulation/${sim_id}/run" || {
        echo "Error: Failed to start simulation" >&2
        return 1
    }
    
    echo "Simulation started. Check status with: resource-meep status"
}

#######################################
# Helper: Check if service is running
#######################################
meep::is_running() {
    docker ps -q -f "name=${MEEP_CONTAINER_NAME}" | grep -q .
}

#######################################
# Helper: Check service health
#######################################
meep::check_health() {
    timeout 5 curl -sf "http://localhost:${MEEP_PORT}/health" &> /dev/null
}

#######################################
# Helper: Wait for service to be healthy
#######################################
meep::wait_for_health() {
    local max_attempts=30
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if meep::check_health; then
            echo "MEEP is ready"
            return 0
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    echo
    echo "Error: MEEP failed to become ready" >&2
    return 1
}

#######################################
# Helper: Create Dockerfile
#######################################
meep::create_dockerfile() {
    cat > "${SCRIPT_DIR}/Dockerfile" << 'EOF'
FROM ubuntu:22.04

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    python3 \
    python3-pip \
    python3-dev \
    python3-numpy \
    python3-scipy \
    python3-matplotlib \
    libhdf5-dev \
    libhdf5-openmpi-dev \
    libfftw3-dev \
    libgsl-dev \
    liblapack-dev \
    libblas-dev \
    openmpi-bin \
    libopenmpi-dev \
    wget \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install MEEP via conda (most reliable method)
RUN wget https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh && \
    bash Miniconda3-latest-Linux-x86_64.sh -b -p /opt/conda && \
    rm Miniconda3-latest-Linux-x86_64.sh

ENV PATH="/opt/conda/bin:${PATH}"

# Install MEEP with MPI support
RUN conda install -y -c conda-forge pymeep=*=mpi_mpich_* && \
    conda install -y -c conda-forge h5py matplotlib flask && \
    conda clean -a -y

# Install additional Python packages
RUN pip3 install --no-cache-dir \
    fastapi \
    uvicorn \
    pydantic \
    aiofiles

# Create working directory
WORKDIR /app

# Copy API server
COPY lib/api_server.py /app/api_server.py
COPY examples /app/templates

# Create data directories
RUN mkdir -p /data/results /data/templates

# Expose API port
EXPOSE 8193

# Run API server
CMD ["python3", "/app/api_server.py"]
EOF
}

#######################################
# Helper: Create API server
#######################################
meep::create_api_server() {
    cat > "${SCRIPT_DIR}/lib/api_server.py" << 'EOF'
#!/usr/bin/env python3
"""
MEEP API Server - Provides REST interface for FDTD simulations
"""

import os
import json
import asyncio
import hashlib
from datetime import datetime
from pathlib import Path
from typing import Optional, Dict, Any, List

from fastapi import FastAPI, HTTPException, UploadFile, File
from fastapi.responses import FileResponse, JSONResponse
from pydantic import BaseModel
import uvicorn
import numpy as np

# Try to import MEEP
try:
    import meep as mp
    MEEP_AVAILABLE = True
except ImportError:
    MEEP_AVAILABLE = False
    print("Warning: MEEP not available, running in mock mode")

app = FastAPI(title="MEEP API Server", version="1.0.0")

# Configuration
API_PORT = int(os.environ.get("MEEP_PORT", 8193))
DATA_DIR = Path("/data")
RESULTS_DIR = DATA_DIR / "results"
TEMPLATES_DIR = DATA_DIR / "templates"

# Ensure directories exist
RESULTS_DIR.mkdir(parents=True, exist_ok=True)
TEMPLATES_DIR.mkdir(parents=True, exist_ok=True)

# Store active simulations
active_simulations = {}

class SimulationRequest(BaseModel):
    template: Optional[str] = None
    resolution: Optional[float] = 50
    runtime: Optional[float] = 100
    geometry: Optional[Dict[str, Any]] = None
    sources: Optional[List[Dict[str, Any]]] = None
    monitors: Optional[List[Dict[str, Any]]] = None

class SimulationResponse(BaseModel):
    simulation_id: str
    status: str
    created_at: str

class HealthResponse(BaseModel):
    status: str
    meep_available: bool
    version: str
    uptime_seconds: float

# Track server start time
server_start_time = datetime.now()

@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    uptime = (datetime.now() - server_start_time).total_seconds()
    
    return HealthResponse(
        status="healthy",
        meep_available=MEEP_AVAILABLE,
        version="1.0.0",
        uptime_seconds=uptime
    )

@app.post("/simulation/create", response_model=SimulationResponse)
async def create_simulation(request: SimulationRequest):
    """Create a new simulation"""
    sim_id = hashlib.md5(
        json.dumps(request.dict(), sort_keys=True).encode()
    ).hexdigest()[:12]
    
    active_simulations[sim_id] = {
        "id": sim_id,
        "status": "created",
        "created_at": datetime.now().isoformat(),
        "config": request.dict()
    }
    
    return SimulationResponse(
        simulation_id=sim_id,
        status="created",
        created_at=active_simulations[sim_id]["created_at"]
    )

@app.post("/simulation/{sim_id}/run")
async def run_simulation(sim_id: str):
    """Execute a simulation"""
    if sim_id not in active_simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    sim = active_simulations[sim_id]
    sim["status"] = "running"
    
    # In production, this would launch actual MEEP simulation
    # For scaffolding, we'll simulate completion
    if MEEP_AVAILABLE:
        # Run basic waveguide simulation
        asyncio.create_task(run_meep_simulation(sim_id))
    else:
        # Mock simulation
        await asyncio.sleep(1)
        sim["status"] = "completed"
    
    return {"message": f"Simulation {sim_id} started"}

@app.get("/simulation/{sim_id}/status")
async def get_simulation_status(sim_id: str):
    """Get simulation status"""
    if sim_id not in active_simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    sim = active_simulations[sim_id]
    return {
        "simulation_id": sim_id,
        "status": sim["status"],
        "created_at": sim["created_at"]
    }

@app.get("/simulation/{sim_id}/fields")
async def get_field_data(sim_id: str):
    """Download field data in HDF5 format"""
    if sim_id not in active_simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    # Check if results exist
    result_file = RESULTS_DIR / f"{sim_id}_fields.h5"
    
    if not result_file.exists():
        # Create mock HDF5 file for testing
        import h5py
        with h5py.File(result_file, 'w') as f:
            f.create_dataset('ez', data=np.random.rand(100, 100))
    
    return FileResponse(
        path=result_file,
        media_type="application/x-hdf5",
        filename=f"{sim_id}_fields.h5"
    )

@app.get("/simulation/{sim_id}/spectra")
async def get_spectra_data(sim_id: str):
    """Get transmission/reflection spectra"""
    if sim_id not in active_simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    # Mock spectral data
    frequencies = np.linspace(0.1, 0.5, 100)
    transmission = np.exp(-(frequencies - 0.3)**2 / 0.01)
    
    return {
        "frequencies": frequencies.tolist(),
        "transmission": transmission.tolist(),
        "reflection": (1 - transmission).tolist()
    }

@app.post("/batch/sweep")
async def parameter_sweep(
    template: str,
    parameter: str,
    values: List[float]
):
    """Run parameter sweep"""
    sweep_id = hashlib.md5(
        f"{template}{parameter}{values}".encode()
    ).hexdigest()[:12]
    
    simulations = []
    for value in values:
        sim_id = f"{sweep_id}_{value}"
        active_simulations[sim_id] = {
            "id": sim_id,
            "status": "queued",
            "created_at": datetime.now().isoformat(),
            "sweep_id": sweep_id,
            "parameter": parameter,
            "value": value
        }
        simulations.append(sim_id)
    
    return {
        "sweep_id": sweep_id,
        "simulations": simulations,
        "parameter": parameter,
        "values": values
    }

@app.get("/templates")
async def list_templates():
    """List available simulation templates"""
    templates = []
    
    # Add built-in templates
    templates.extend([
        {"name": "waveguide", "description": "2D waveguide with bends"},
        {"name": "resonator", "description": "Ring resonator"},
        {"name": "photonic_crystal", "description": "2D photonic crystal"}
    ])
    
    # Add user templates
    for template_file in TEMPLATES_DIR.glob("*.py"):
        templates.append({
            "name": template_file.stem,
            "description": "User template"
        })
    
    return {"templates": templates}

async def run_meep_simulation(sim_id: str):
    """Run actual MEEP simulation (async)"""
    sim = active_simulations[sim_id]
    
    try:
        # Simple waveguide simulation
        cell_size = mp.Vector3(16, 8, 0)
        
        geometry = [
            mp.Block(
                mp.Vector3(mp.inf, 1, mp.inf),
                center=mp.Vector3(),
                material=mp.Medium(epsilon=12)
            )
        ]
        
        sources = [
            mp.Source(
                mp.ContinuousSource(frequency=0.15),
                component=mp.Ez,
                center=mp.Vector3(-7, 0)
            )
        ]
        
        pml_layers = [mp.PML(1.0)]
        
        sim_obj = mp.Simulation(
            cell_size=cell_size,
            boundary_layers=pml_layers,
            geometry=geometry,
            sources=sources,
            resolution=sim["config"].get("resolution", 50)
        )
        
        sim_obj.run(until=sim["config"].get("runtime", 100))
        
        # Save results
        result_file = RESULTS_DIR / f"{sim_id}_fields.h5"
        sim_obj.output_epsilon(output_file=str(result_file))
        
        sim["status"] = "completed"
    except Exception as e:
        sim["status"] = "failed"
        sim["error"] = str(e)

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=API_PORT)
EOF
    chmod +x "${SCRIPT_DIR}/lib/api_server.py"
}

#######################################
# Helper: Install simulation templates
#######################################
meep::install_templates() {
    # Create waveguide template
    cat > "${MEEP_TEMPLATES_DIR}/waveguide.py" << 'EOF'
import meep as mp
import numpy as np

# Waveguide with 90-degree bend
cell_size = mp.Vector3(16, 16, 0)

geometry = [
    mp.Block(
        mp.Vector3(12, 1, mp.inf),
        center=mp.Vector3(-1.5, 0),
        material=mp.Medium(epsilon=12)
    ),
    mp.Block(
        mp.Vector3(1, 12, mp.inf),
        center=mp.Vector3(4, 2.5),
        material=mp.Medium(epsilon=12)
    )
]

sources = [
    mp.Source(
        mp.GaussianSource(0.15, fwidth=0.1),
        component=mp.Ez,
        center=mp.Vector3(-7, 0)
    )
]

pml_layers = [mp.PML(1.0)]

sim = mp.Simulation(
    cell_size=cell_size,
    boundary_layers=pml_layers,
    geometry=geometry,
    sources=sources,
    resolution=50
)

sim.run(until=200)
EOF

    # Create ring resonator template
    cat > "${MEEP_TEMPLATES_DIR}/resonator.py" << 'EOF'
import meep as mp
import numpy as np

# Ring resonator
n = 3.4  # Silicon index
w = 1    # Waveguide width
r = 5    # Ring radius
gap = 0.2  # Coupling gap

pad = 4
dpml = 2

sxy = 2 * (r + w + gap + pad + dpml)
cell_size = mp.Vector3(sxy, sxy, 0)

geometry = [
    mp.Cylinder(r + w, height=mp.inf, material=mp.Medium(index=n)),
    mp.Cylinder(r, height=mp.inf, material=mp.air),
    mp.Block(
        mp.Vector3(mp.inf, w, mp.inf),
        center=mp.Vector3(0, -(r + gap + w/2)),
        material=mp.Medium(index=n)
    )
]

sources = [
    mp.Source(
        mp.GaussianSource(0.15, fwidth=0.1),
        component=mp.Ez,
        center=mp.Vector3(-(r + pad), -(r + gap + w/2))
    )
]

sim = mp.Simulation(
    cell_size=cell_size,
    boundary_layers=[mp.PML(dpml)],
    geometry=geometry,
    sources=sources,
    resolution=50
)

sim.run(until=300)
EOF

    # Create photonic crystal template
    cat > "${MEEP_TEMPLATES_DIR}/photonic_crystal.py" << 'EOF'
import meep as mp
import numpy as np

# 2D photonic crystal
sx = 16
sy = 16

geometry = []
for i in range(-8, 9):
    for j in range(-8, 9):
        geometry.append(
            mp.Cylinder(
                0.2, 
                height=mp.inf,
                center=mp.Vector3(i, j),
                material=mp.Medium(epsilon=12)
            )
        )

sources = [
    mp.Source(
        mp.GaussianSource(0.25, fwidth=0.2),
        component=mp.Ez,
        center=mp.Vector3()
    )
]

sim = mp.Simulation(
    cell_size=mp.Vector3(sx, sy, 0),
    boundary_layers=[mp.PML(1.0)],
    geometry=geometry,
    sources=sources,
    resolution=30
)

sim.run(until=200)
EOF

    echo "Simulation templates installed"
}

# Export functions for use in other scripts
export -f meep::manage
export -f meep::install
export -f meep::start
export -f meep::stop
export -f meep::restart
export -f meep::uninstall
export -f meep::status
export -f meep::logs
export -f meep::content
export -f meep::is_running
export -f meep::check_health