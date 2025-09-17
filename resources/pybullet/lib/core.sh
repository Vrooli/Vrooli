#!/bin/bash

# PyBullet Core Functions
# Implements v2.0 universal contract requirements

set -euo pipefail

RESOURCE_NAME="pybullet"
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="$RESOURCE_DIR/config"
VENV_DIR="$RESOURCE_DIR/.venv"
API_DIR="$RESOURCE_DIR/api"
EXAMPLES_DIR="$RESOURCE_DIR/examples"

# Source configuration
source "$CONFIG_DIR/defaults.sh"

# Port allocation
VROOLI_ROOT="${VROOLI_ROOT:-/home/matthalloran8/Vrooli}"
if [[ -f "$VROOLI_ROOT/scripts/resources/port_registry.sh" ]]; then
    source "$VROOLI_ROOT/scripts/resources/port_registry.sh"
    PYBULLET_PORT="${PYBULLET_PORT:-$(get_port 'pybullet' 2>/dev/null || echo 11460)}"
else
    PYBULLET_PORT="${PYBULLET_PORT:-11460}"
fi

# Show help information
show_help() {
    cat << EOF
PyBullet Physics Simulation Resource

USAGE:
    resource-pybullet <command> [options]

COMMANDS:
    help                    Show this help message
    info [--json]          Show resource runtime information
    manage <subcommand>    Lifecycle management
        install            Install PyBullet and dependencies
        uninstall         Remove PyBullet
        start [--wait]    Start PyBullet API server
        stop              Stop PyBullet server
        restart           Restart PyBullet server
    test <type>           Run tests
        smoke             Quick health check (<30s)
        integration       Full functionality test
        unit             Test library functions
        all              Run all tests
    content <subcommand>  Manage simulations
        list             List available demos/models
        add <file>       Add simulation/model file
        get <name>       Get simulation details
        remove <name>    Remove simulation
        execute <name>   Run simulation
    status [--json]      Show detailed status
    logs [--tail N]      View resource logs
    credentials          Display API credentials

EXAMPLES:
    # Start PyBullet server
    resource-pybullet manage start --wait

    # Run basic physics demo
    resource-pybullet content execute pendulum

    # Check health status
    resource-pybullet test smoke

DEFAULT CONFIGURATION:
    Port: $PYBULLET_PORT
    API: http://localhost:$PYBULLET_PORT
    Venv: $VENV_DIR

EOF
}

# Show runtime information
show_info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "$CONFIG_DIR/runtime.json"
    else
        echo "PyBullet Resource Information:"
        echo "=============================="
        jq -r 'to_entries[] | "\(.key): \(.value)"' "$CONFIG_DIR/runtime.json"
    fi
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            install_pybullet "$@"
            ;;
        uninstall)
            uninstall_pybullet "$@"
            ;;
        start)
            start_pybullet "$@"
            ;;
        stop)
            stop_pybullet "$@"
            ;;
        restart)
            restart_pybullet "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand '$subcommand'"
            exit 1
            ;;
    esac
}

# Install PyBullet and dependencies
install_pybullet() {
    local force="${1:-false}"
    
    echo "Installing PyBullet physics simulation..."
    
    # Check if already installed
    if [[ -d "$VENV_DIR" ]] && [[ "$force" != "--force" ]]; then
        echo "PyBullet already installed. Use --force to reinstall."
        return 2
    fi
    
    # Clean up any partial installation
    if [[ "$force" == "--force" ]] && [[ -d "$VENV_DIR" ]]; then
        echo "Removing existing installation..."
        rm -rf "$VENV_DIR"
    fi
    
    # Check Python requirements
    if ! command -v python3 &> /dev/null; then
        echo "Error: Python 3 is not installed"
        echo "Please install Python 3.8 or later"
        return 1
    fi
    
    # Check if pip is available
    if ! python3 -m pip --version &> /dev/null; then
        echo "Error: pip is not installed"
        echo "Please install python3-pip package"
        return 1
    fi
    
    # Try to create virtual environment, fallback to custom directory if needed
    echo "Setting up Python environment..."
    if python3 -m venv "$VENV_DIR" 2>/dev/null; then
        echo "Virtual environment created successfully"
        # Activate and install packages
        source "$VENV_DIR/bin/activate"
        python3 -m pip install --upgrade pip
        python3 -m pip install pybullet numpy scipy fastapi uvicorn httpx
    else
        echo "Note: python3-venv not available, using alternative installation"
        # Create custom Python environment structure
        mkdir -p "$VENV_DIR/bin" "$VENV_DIR/lib"
        
        # Install packages to custom location
        echo "Installing packages to custom location..."
        python3 -m pip install --target="$VENV_DIR/lib" --upgrade pip
        python3 -m pip install --target="$VENV_DIR/lib" pybullet numpy scipy fastapi uvicorn httpx
        
        # Create activation script for custom environment
        cat > "$VENV_DIR/bin/activate" << 'ACTIVATE_EOF'
#!/bin/bash
# Custom activation script for PyBullet environment
export PYBULLET_VENV="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"
export PYBULLET_ORIGINAL_PATH="$PATH"
export PYBULLET_ORIGINAL_PYTHONPATH="${PYTHONPATH:-}"
export PATH="$PYBULLET_VENV/bin:$PATH"
export PYTHONPATH="$PYBULLET_VENV/lib:${PYTHONPATH:-}"

# Create deactivate function
deactivate() {
    export PATH="$PYBULLET_ORIGINAL_PATH"
    export PYTHONPATH="$PYBULLET_ORIGINAL_PYTHONPATH"
    unset PYBULLET_VENV
    unset PYBULLET_ORIGINAL_PATH
    unset PYBULLET_ORIGINAL_PYTHONPATH
    unset -f deactivate
}
ACTIVATE_EOF
        chmod +x "$VENV_DIR/bin/activate"
        
        # Create wrapper scripts for custom environment
        cat > "$VENV_DIR/bin/python" << 'EOF'
#!/bin/bash
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
export PYTHONPATH="$SCRIPT_DIR/../lib:${PYTHONPATH:-}"
exec python3 "$@"
EOF
        chmod +x "$VENV_DIR/bin/python"
        
        cat > "$VENV_DIR/bin/uvicorn" << 'EOF'
#!/bin/bash
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
export PYTHONPATH="$SCRIPT_DIR/../lib:${PYTHONPATH:-}"
exec python3 -m uvicorn "$@"
EOF
        chmod +x "$VENV_DIR/bin/uvicorn"
    fi
    
    # Create API server script
    mkdir -p "$API_DIR"
    create_api_server
    
    # Create example simulations
    mkdir -p "$EXAMPLES_DIR"
    create_examples
    
    echo "PyBullet installation complete!"
    return 0
}

# Uninstall PyBullet
uninstall_pybullet() {
    local force="${1:-false}"
    local keep_data="${2:-false}"
    
    if [[ "$force" != "--force" ]]; then
        read -p "Remove PyBullet? This will delete the virtual environment. (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Uninstall cancelled."
            return 1
        fi
    fi
    
    # Stop if running
    stop_pybullet --force
    
    # Remove virtual environment
    if [[ -d "$VENV_DIR" ]]; then
        echo "Removing virtual environment..."
        rm -rf "$VENV_DIR"
    fi
    
    # Optionally keep data
    if [[ "$keep_data" != "--keep-data" ]]; then
        rm -rf "$EXAMPLES_DIR"/*.pkl 2>/dev/null || true
    fi
    
    echo "PyBullet uninstalled successfully."
    return 0
}

# Start PyBullet API server
start_pybullet() {
    local wait_flag="${1:-}"
    
    echo "Starting PyBullet API server on port $PYBULLET_PORT..."
    
    # Check if already running
    if pgrep -f "uvicorn.*$PYBULLET_PORT" > /dev/null; then
        echo "PyBullet server already running."
        return 2
    fi
    
    # Check if API server exists
    if [[ ! -f "$API_DIR/server.py" ]]; then
        echo "API server not found. Creating it..."
        mkdir -p "$API_DIR"
        create_api_server
    fi
    
    # Start API server (handle both venv and custom installation)
    if [[ -f "$VENV_DIR/bin/activate" ]]; then
        source "$VENV_DIR/bin/activate"
    else
        export PYTHONPATH="$VENV_DIR/lib:${PYTHONPATH:-}"
    fi
    
    # Use custom uvicorn wrapper if it exists, otherwise use python -m uvicorn
    if [[ -x "$VENV_DIR/bin/uvicorn" ]]; then
        nohup "$VENV_DIR/bin/uvicorn" api.server:app --host 0.0.0.0 --port "$PYBULLET_PORT" \
            --app-dir "$RESOURCE_DIR" > "$RESOURCE_DIR/pybullet.log" 2>&1 &
    else
        nohup python3 -m uvicorn api.server:app --host 0.0.0.0 --port "$PYBULLET_PORT" \
            --app-dir "$RESOURCE_DIR" > "$RESOURCE_DIR/pybullet.log" 2>&1 &
    fi
    
    local pid=$!
    echo $pid > "$RESOURCE_DIR/pybullet.pid"
    
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "Waiting for server to be ready..."
        for i in {1..30}; do
            if timeout 5 curl -sf "http://localhost:$PYBULLET_PORT/health" > /dev/null 2>&1; then
                echo "PyBullet server is ready!"
                return 0
            fi
            sleep 1
        done
        echo "Warning: Server started but health check failed"
    fi
    
    echo "PyBullet server started (PID: $pid)"
    return 0
}

# Stop PyBullet server
stop_pybullet() {
    local force="${1:-}"
    
    echo "Stopping PyBullet server..."
    
    if [[ -f "$RESOURCE_DIR/pybullet.pid" ]]; then
        local pid=$(cat "$RESOURCE_DIR/pybullet.pid")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm "$RESOURCE_DIR/pybullet.pid"
            echo "PyBullet server stopped."
            return 0
        fi
    fi
    
    # Fallback: find and kill by port
    local pids=$(lsof -ti ":$PYBULLET_PORT" 2>/dev/null || true)
    if [[ -n "$pids" ]]; then
        kill $pids
        echo "PyBullet server stopped (fallback method)."
        return 0
    fi
    
    echo "PyBullet server not running."
    return 2
}

# Restart PyBullet server
restart_pybullet() {
    stop_pybullet --force
    sleep 2
    start_pybullet "$@"
}

# Handle content subcommands
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        list)
            list_content
            ;;
        add)
            add_content "$@"
            ;;
        get)
            get_content "$@"
            ;;
        remove)
            remove_content "$@"
            ;;
        execute)
            execute_content "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand '$subcommand'"
            exit 1
            ;;
    esac
}

# List available simulations
list_content() {
    echo "Available simulations:"
    echo "====================="
    if [[ -d "$EXAMPLES_DIR" ]]; then
        local files=$(ls -1 "$EXAMPLES_DIR"/*.py 2>/dev/null)
        if [[ -n "$files" ]]; then
            echo "$files" | xargs -n1 basename 2>/dev/null | sed 's/\.py$//'
        else
            echo "(No simulations found. Use 'resource-pybullet content add' to add one)"
        fi
    else
        echo "(Examples directory not found)"
    fi
}

# Execute simulation
execute_content() {
    local sim_name="${1:-}"
    
    if [[ -z "$sim_name" ]]; then
        echo "Error: Simulation name required"
        exit 1
    fi
    
    local sim_file="$EXAMPLES_DIR/${sim_name}.py"
    if [[ ! -f "$sim_file" ]]; then
        echo "Error: Simulation '$sim_name' not found"
        exit 1
    fi
    
    source "$VENV_DIR/bin/activate"
    python "$sim_file"
}

# Show status
show_status() {
    local format="${1:-text}"
    
    local status="stopped"
    local pid=""
    local uptime=""
    
    if [[ -f "$RESOURCE_DIR/pybullet.pid" ]]; then
        pid=$(cat "$RESOURCE_DIR/pybullet.pid")
        if kill -0 "$pid" 2>/dev/null; then
            status="running"
            uptime=$(ps -o etime= -p "$pid" | xargs)
        fi
    fi
    
    if [[ "$format" == "--json" ]]; then
        cat << EOF
{
    "status": "$status",
    "pid": "$pid",
    "uptime": "$uptime",
    "port": $PYBULLET_PORT,
    "health_url": "http://localhost:$PYBULLET_PORT/health"
}
EOF
    else
        echo "PyBullet Status:"
        echo "==============="
        echo "Status: $status"
        [[ -n "$pid" ]] && echo "PID: $pid"
        [[ -n "$uptime" ]] && echo "Uptime: $uptime"
        echo "Port: $PYBULLET_PORT"
        echo "API URL: http://localhost:$PYBULLET_PORT"
    fi
}

# Show logs
show_logs() {
    local tail_lines="${1:-50}"
    
    if [[ "$tail_lines" == "--tail" ]]; then
        tail_lines="${2:-50}"
    fi
    
    if [[ -f "$RESOURCE_DIR/pybullet.log" ]]; then
        tail -n "$tail_lines" "$RESOURCE_DIR/pybullet.log"
    else
        echo "No logs available."
    fi
}

# Show credentials
show_credentials() {
    cat << EOF
PyBullet API Credentials:
========================
URL: http://localhost:$PYBULLET_PORT
Health: http://localhost:$PYBULLET_PORT/health
Docs: http://localhost:$PYBULLET_PORT/docs

No authentication required for local development.
EOF
}

# Create API server
create_api_server() {
    cat > "$API_DIR/server.py" << 'EOF'
"""PyBullet API Server"""

import os
import time
import asyncio
from typing import Dict, Any, Optional
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import pybullet as p
import pybullet_data

app = FastAPI(title="PyBullet Physics API", version="1.0.0")

# Global simulation instances
simulations: Dict[str, int] = {}

class SimulationCreate(BaseModel):
    name: str
    gravity: list = [0, 0, -9.81]
    timestep: float = 1/240
    use_gui: bool = False

class SimulationStep(BaseModel):
    steps: int = 1

class SpawnObject(BaseModel):
    shape: str = "box"
    position: list = [0, 0, 1]
    size: list = [1, 1, 1]
    mass: float = 1.0

@app.get("/health")
async def health():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "pybullet",
        "version": "2.4.1",
        "simulations": len(simulations),
        "timestamp": time.time()
    }

@app.post("/simulation/create")
async def create_simulation(config: SimulationCreate):
    """Create new simulation instance"""
    if config.name in simulations:
        raise HTTPException(status_code=400, detail="Simulation already exists")
    
    if config.use_gui:
        client_id = p.connect(p.GUI)
    else:
        client_id = p.connect(p.DIRECT)
    
    p.setGravity(*config.gravity, physicsClientId=client_id)
    p.setTimeStep(config.timestep, physicsClientId=client_id)
    p.setAdditionalSearchPath(pybullet_data.getDataPath())
    
    # Load ground plane
    p.loadURDF("plane.urdf", physicsClientId=client_id)
    
    simulations[config.name] = client_id
    
    return {
        "name": config.name,
        "client_id": client_id,
        "status": "created"
    }

@app.delete("/simulation/{name}")
async def destroy_simulation(name: str):
    """Destroy simulation instance"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    p.disconnect(simulations[name])
    del simulations[name]
    
    return {"status": "destroyed", "name": name}

@app.post("/simulation/{name}/step")
async def step_simulation(name: str, config: SimulationStep):
    """Advance simulation by N steps"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    client_id = simulations[name]
    for _ in range(config.steps):
        p.stepSimulation(physicsClientId=client_id)
    
    return {"status": "stepped", "steps": config.steps}

@app.get("/simulation/{name}/state")
async def get_state(name: str):
    """Get current simulation state"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    client_id = simulations[name]
    num_bodies = p.getNumBodies(physicsClientId=client_id)
    
    bodies = []
    for i in range(num_bodies):
        pos, orn = p.getBasePositionAndOrientation(i, physicsClientId=client_id)
        vel, ang_vel = p.getBaseVelocity(i, physicsClientId=client_id)
        bodies.append({
            "id": i,
            "position": pos,
            "orientation": orn,
            "velocity": vel,
            "angular_velocity": ang_vel
        })
    
    return {
        "name": name,
        "num_bodies": num_bodies,
        "bodies": bodies
    }

@app.post("/simulation/{name}/spawn")
async def spawn_object(name: str, config: SpawnObject):
    """Spawn object in simulation"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    client_id = simulations[name]
    
    # Create collision shape
    if config.shape == "box":
        shape_id = p.createCollisionShape(p.GEOM_BOX, halfExtents=[s/2 for s in config.size])
    elif config.shape == "sphere":
        shape_id = p.createCollisionShape(p.GEOM_SPHERE, radius=config.size[0])
    else:
        raise HTTPException(status_code=400, detail=f"Unknown shape: {config.shape}")
    
    # Create body
    body_id = p.createMultiBody(
        baseMass=config.mass,
        baseCollisionShapeIndex=shape_id,
        basePosition=config.position,
        physicsClientId=client_id
    )
    
    return {"body_id": body_id, "shape": config.shape, "position": config.position}

if __name__ == "__main__":
    import uvicorn
    # Port must come from environment, no fallback
    port_str = os.environ.get("PYBULLET_PORT")
    if not port_str:
        raise ValueError("PYBULLET_PORT environment variable not set")
    port = int(port_str)
    uvicorn.run(app, host="0.0.0.0", port=port)
EOF
}

# Create example simulations
create_examples() {
    # Pendulum example
    cat > "$EXAMPLES_DIR/pendulum.py" << 'EOF'
"""Simple pendulum simulation"""

import pybullet as p
import time
import math

# Connect to physics server
p.connect(p.DIRECT)
p.setGravity(0, 0, -9.81)

# Create pendulum
base = p.createMultiBody(baseMass=0, basePosition=[0, 0, 2])
link = p.createMultiBody(
    baseMass=1.0,
    baseCollisionShapeIndex=p.createCollisionShape(p.GEOM_SPHERE, radius=0.1),
    basePosition=[0, 0, 1]
)

# Create constraint (joint)
joint = p.createConstraint(
    base, -1, link, -1,
    p.JOINT_POINT2POINT,
    [0, 0, 0], [0, 0, 0], [0, 0, 1]
)

# Apply initial velocity
p.resetBaseVelocity(link, linearVelocity=[1, 0, 0])

# Simulate
print("Simulating pendulum for 5 seconds...")
for i in range(240 * 5):  # 5 seconds at 240Hz
    p.stepSimulation()
    if i % 24 == 0:  # Print every 0.1s
        pos, _ = p.getBasePositionAndOrientation(link)
        print(f"t={i/240:.1f}s: position={pos}")

p.disconnect()
print("Simulation complete!")
EOF

    # Cart-pole example
    cat > "$EXAMPLES_DIR/cartpole.py" << 'EOF'
"""Cart-pole balancing simulation"""

import pybullet as p
import pybullet_data
import time

# Connect and setup
p.connect(p.DIRECT)
p.setAdditionalSearchPath(pybullet_data.getDataPath())
p.setGravity(0, 0, -9.81)

# Load ground
plane = p.loadURDF("plane.urdf")

# Create cart
cart = p.loadURDF("cartpole.urdf", [0, 0, 0.5])

# Simulate with random control
print("Simulating cart-pole...")
for i in range(1000):
    # Apply random force to cart
    if i % 50 == 0:
        force = (p.random.uniform() - 0.5) * 10
        p.applyExternalForce(cart, 0, [force, 0, 0], [0, 0, 0], p.WORLD_FRAME)
    
    p.stepSimulation()
    
    if i % 100 == 0:
        pos, _ = p.getBasePositionAndOrientation(cart)
        print(f"Step {i}: Cart position x={pos[0]:.2f}")

p.disconnect()
print("Simulation complete!")
EOF
}

# Add content
add_content() {
    local file="${1:-}"
    
    if [[ -z "$file" || ! -f "$file" ]]; then
        echo "Error: Valid file path required"
        exit 1
    fi
    
    cp "$file" "$EXAMPLES_DIR/"
    echo "Added simulation: $(basename "$file")"
}

# Get content details
get_content() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Simulation name required"
        exit 1
    fi
    
    local file="$EXAMPLES_DIR/${name}.py"
    if [[ -f "$file" ]]; then
        head -20 "$file"
    else
        echo "Error: Simulation '$name' not found"
        exit 1
    fi
}

# Remove content
remove_content() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Simulation name required"
        exit 1
    fi
    
    local file="$EXAMPLES_DIR/${name}.py"
    if [[ -f "$file" ]]; then
        rm "$file"
        echo "Removed simulation: $name"
    else
        echo "Error: Simulation '$name' not found"
        exit 1
    fi
}