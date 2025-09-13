#!/bin/bash
set -euo pipefail

# CrewAI Core Functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CREWAI_LIB_DIR="${APP_ROOT}/resources/crewai/lib"
CREWAI_ROOT_DIR="${APP_ROOT}/resources/crewai"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Configuration - Get port from registry or environment
if [[ -z "${CREWAI_PORT:-}" ]]; then
    CREWAI_PORT=$("${APP_ROOT}/scripts/resources/port_registry.sh" crewai | grep -E "crewai\s+:" | awk '{print $3}')
    CREWAI_PORT="${CREWAI_PORT:-8084}"
fi

CREWAI_NAME="${CREWAI_NAME:-crewai}"
CREWAI_DATA_DIR="${CREWAI_DATA_DIR:-${HOME}/.crewai}"
CREWAI_WORKSPACE_DIR="${CREWAI_WORKSPACE_DIR:-${CREWAI_DATA_DIR}/workspace}"
CREWAI_CREWS_DIR="${CREWAI_CREWS_DIR:-${CREWAI_DATA_DIR}/crews}"
CREWAI_AGENTS_DIR="${CREWAI_AGENTS_DIR:-${CREWAI_DATA_DIR}/agents}"
CREWAI_PID_FILE="${CREWAI_PID_FILE:-${CREWAI_DATA_DIR}/crewai.pid}"
CREWAI_LOG_FILE="${CREWAI_LOG_FILE:-${CREWAI_DATA_DIR}/crewai.log}"
CREWAI_SERVER_FILE="${CREWAI_SERVER_FILE:-${CREWAI_DATA_DIR}/server.py}"
CREWAI_MOCK_MODE="${CREWAI_MOCK_MODE:-true}"  # Run in mock mode until venv issue resolved

# Check if Python is available
check_python() {
    if command -v python3 &>/dev/null; then
        echo "python3"
    else
        log::error "Python 3 is required but not found"
        return 1
    fi
}

# Initialize directories
init_directories() {
    mkdir -p "${CREWAI_DATA_DIR}"
    mkdir -p "${CREWAI_WORKSPACE_DIR}"
    mkdir -p "${CREWAI_CREWS_DIR}"
    mkdir -p "${CREWAI_AGENTS_DIR}"
}

# Install CrewAI
install_crewai() {
    init_directories
    
    log::info "Installing CrewAI in mock mode..."
    
    # Create server file
    create_server_file
    
    log::success "CrewAI installed successfully (mock mode)"
}

# Create server file
create_server_file() {
    cat > "${CREWAI_SERVER_FILE}" << 'EOF'
#!/usr/bin/env python3
"""CrewAI Mock API Server for Vrooli"""

import os
import json
import uuid
from pathlib import Path
from http.server import HTTPServer, BaseHTTPRequestHandler
from datetime import datetime
import shutil
from urllib.parse import urlparse, parse_qs
import threading
import time

# Configuration
PORT = ${CREWAI_PORT:-8084}

# Paths
CREWAI_DATA_DIR = Path.home() / ".crewai"
CREWS_DIR = CREWAI_DATA_DIR / "crews"
AGENTS_DIR = CREWAI_DATA_DIR / "agents"
WORKSPACE_DIR = CREWAI_DATA_DIR / "workspace"
TASKS_DIR = CREWAI_DATA_DIR / "tasks"

# Ensure directories exist
for dir_path in [CREWS_DIR, AGENTS_DIR, WORKSPACE_DIR, TASKS_DIR]:
    dir_path.mkdir(parents=True, exist_ok=True)

# Store loaded crews and agents
loaded_crews = {}
loaded_agents = {}
task_executions = {}

class CrewAIHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        """Handle GET requests"""
        parsed_path = urlparse(self.path)
        path = parsed_path.path
        
        if path == "/":
            self.send_json_response(200, {
                "name": "CrewAI Server",
                "version": "1.0.0-mock",
                "status": "running",
                "crews": len(loaded_crews),
                "agents": len(loaded_agents),
                "workspace": str(WORKSPACE_DIR),
                "capabilities": ["crews", "agents", "tasks", "inject", "execute"]
            })
        elif path == "/health":
            self.send_json_response(200, {
                "status": "healthy",
                "timestamp": datetime.utcnow().isoformat(),
                "crews_loaded": len(loaded_crews),
                "agents_loaded": len(loaded_agents),
                "active_tasks": len([t for t in task_executions.values() if t["status"] == "running"])
            })
        elif path == "/crews":
            crews = []
            for crew_file in CREWS_DIR.glob("*.py"):
                crew_name = crew_file.stem
                crew_data = loaded_crews.get(crew_name, {})
                crews.append({
                    "name": crew_name,
                    "loaded": crew_name in loaded_crews,
                    "path": str(crew_file),
                    "agents": crew_data.get("agents", []),
                    "tasks": crew_data.get("tasks", [])
                })
            # Also include JSON crews
            for crew_file in CREWS_DIR.glob("*.json"):
                crew_name = crew_file.stem
                try:
                    with open(crew_file) as f:
                        crew_data = json.load(f)
                    crews.append({
                        "name": crew_name,
                        "loaded": True,
                        "path": str(crew_file),
                        "agents": crew_data.get("agents", []),
                        "tasks": crew_data.get("tasks", [])
                    })
                except:
                    pass
            self.send_json_response(200, {"crews": crews})
        elif path == "/agents":
            agents = []
            for agent_file in AGENTS_DIR.glob("*.py"):
                agent_name = agent_file.stem
                agent_data = loaded_agents.get(agent_name, {})
                agents.append({
                    "name": agent_name,
                    "loaded": agent_name in loaded_agents,
                    "path": str(agent_file),
                    "role": agent_data.get("role", ""),
                    "goal": agent_data.get("goal", "")
                })
            # Also include JSON agents
            for agent_file in AGENTS_DIR.glob("*.json"):
                agent_name = agent_file.stem
                try:
                    with open(agent_file) as f:
                        agent_data = json.load(f)
                    agents.append({
                        "name": agent_name,
                        "loaded": True,
                        "path": str(agent_file),
                        "role": agent_data.get("role", ""),
                        "goal": agent_data.get("goal", "")
                    })
                except:
                    pass
            self.send_json_response(200, {"agents": agents})
        elif path == "/tasks":
            tasks = list(task_executions.values())
            self.send_json_response(200, {"tasks": tasks})
        elif path.startswith("/tasks/"):
            task_id = path.split("/")[-1]
            if task_id in task_executions:
                self.send_json_response(200, task_executions[task_id])
            else:
                self.send_json_response(404, {"error": "Task not found"})
        elif path.startswith("/crews/"):
            crew_name = path.split("/")[-1]
            crew_file = CREWS_DIR / f"{crew_name}.json"
            if crew_file.exists():
                with open(crew_file) as f:
                    crew_data = json.load(f)
                self.send_json_response(200, crew_data)
            else:
                self.send_json_response(404, {"error": "Crew not found"})
        elif path.startswith("/agents/"):
            agent_name = path.split("/")[-1]
            agent_file = AGENTS_DIR / f"{agent_name}.json"
            if agent_file.exists():
                with open(agent_file) as f:
                    agent_data = json.load(f)
                self.send_json_response(200, agent_data)
            else:
                self.send_json_response(404, {"error": "Agent not found"})
        else:
            self.send_json_response(404, {"error": "Not found"})
    
    def do_POST(self):
        """Handle POST requests"""
        parsed_path = urlparse(self.path)
        path = parsed_path.path
        
        content_length = int(self.headers.get('Content-Length', 0))
        if content_length > 0:
            post_data = self.rfile.read(content_length)
            try:
                data = json.loads(post_data)
            except:
                data = {}
        else:
            data = {}
        
        if path == "/inject":
            file_path = data.get('file_path')
            file_type = data.get('file_type')
            
            if not file_path or not file_type:
                self.send_json_response(400, {"error": "Missing file_path or file_type"})
                return
            
            source_path = Path(file_path)
            if not source_path.exists():
                self.send_json_response(404, {"error": f"File not found: {file_path}"})
                return
            
            if file_type == "crew":
                dest_dir = CREWS_DIR
            elif file_type == "agent":
                dest_dir = AGENTS_DIR
            else:
                self.send_json_response(400, {"error": "file_type must be 'crew' or 'agent'"})
                return
            
            dest_path = dest_dir / source_path.name
            
            try:
                shutil.copy2(source_path, dest_path)
                self.send_json_response(200, {
                    "status": "injected",
                    "type": file_type,
                    "name": source_path.stem,
                    "destination": str(dest_path)
                })
            except Exception as e:
                self.send_json_response(500, {"error": str(e)})
        
        elif path == "/crews":
            # Create a new crew
            crew_name = data.get("name", f"crew_{int(time.time())}")
            agents = data.get("agents", [])
            tasks = data.get("tasks", [])
            
            crew_data = {
                "name": crew_name,
                "agents": agents,
                "tasks": tasks,
                "created": datetime.utcnow().isoformat()
            }
            
            crew_file = CREWS_DIR / f"{crew_name}.json"
            with open(crew_file, "w") as f:
                json.dump(crew_data, f, indent=2)
            
            loaded_crews[crew_name] = crew_data
            
            self.send_json_response(201, {
                "status": "created",
                "crew": crew_data,
                "path": str(crew_file)
            })
        
        elif path == "/agents":
            # Create a new agent
            agent_name = data.get("name", f"agent_{int(time.time())}")
            role = data.get("role", "assistant")
            goal = data.get("goal", "help with tasks")
            backstory = data.get("backstory", "")
            
            agent_data = {
                "name": agent_name,
                "role": role,
                "goal": goal,
                "backstory": backstory,
                "created": datetime.utcnow().isoformat()
            }
            
            agent_file = AGENTS_DIR / f"{agent_name}.json"
            with open(agent_file, "w") as f:
                json.dump(agent_data, f, indent=2)
            
            loaded_agents[agent_name] = agent_data
            
            self.send_json_response(201, {
                "status": "created",
                "agent": agent_data,
                "path": str(agent_file)
            })
        
        elif path == "/execute":
            # Execute a crew (mock execution)
            crew_name = data.get("crew")
            input_data = data.get("input", {})
            
            if not crew_name:
                self.send_json_response(400, {"error": "Missing crew name"})
                return
            
            crew_file = CREWS_DIR / f"{crew_name}.json"
            if not crew_file.exists():
                self.send_json_response(404, {"error": "Crew not found"})
                return
            
            # Create mock task execution
            task_id = str(uuid.uuid4())
            task_data = {
                "id": task_id,
                "crew": crew_name,
                "input": input_data,
                "status": "running",
                "started": datetime.utcnow().isoformat(),
                "progress": 0,
                "result": None
            }
            
            task_executions[task_id] = task_data
            
            # Simulate async execution
            def mock_execute():
                for i in range(1, 11):
                    time.sleep(0.5)
                    task_executions[task_id]["progress"] = i * 10
                
                task_executions[task_id]["status"] = "completed"
                task_executions[task_id]["completed"] = datetime.utcnow().isoformat()
                task_executions[task_id]["result"] = {
                    "output": f"Mock execution of {crew_name} completed",
                    "data": input_data
                }
            
            thread = threading.Thread(target=mock_execute)
            thread.daemon = True
            thread.start()
            
            self.send_json_response(202, {
                "status": "started",
                "task_id": task_id,
                "message": f"Crew {crew_name} execution started"
            })
        
        elif path.startswith("/crews/") and path.endswith("/delete"):
            crew_name = path.split("/")[-2]
            crew_file = CREWS_DIR / f"{crew_name}.json"
            if crew_file.exists():
                crew_file.unlink()
                if crew_name in loaded_crews:
                    del loaded_crews[crew_name]
                self.send_json_response(200, {"status": "deleted", "crew": crew_name})
            else:
                self.send_json_response(404, {"error": "Crew not found"})
        
        elif path.startswith("/agents/") and path.endswith("/delete"):
            agent_name = path.split("/")[-2]
            agent_file = AGENTS_DIR / f"{agent_name}.json"
            if agent_file.exists():
                agent_file.unlink()
                if agent_name in loaded_agents:
                    del loaded_agents[agent_name]
                self.send_json_response(200, {"status": "deleted", "agent": agent_name})
            else:
                self.send_json_response(404, {"error": "Agent not found"})
        
        else:
            self.send_json_response(404, {"error": "Not found"})
    
    def do_DELETE(self):
        """Handle DELETE requests"""
        parsed_path = urlparse(self.path)
        path = parsed_path.path
        
        if path.startswith("/crews/"):
            crew_name = path.split("/")[-1]
            crew_file = CREWS_DIR / f"{crew_name}.json"
            if crew_file.exists():
                crew_file.unlink()
                if crew_name in loaded_crews:
                    del loaded_crews[crew_name]
                self.send_json_response(200, {"status": "deleted", "crew": crew_name})
            else:
                self.send_json_response(404, {"error": "Crew not found"})
        
        elif path.startswith("/agents/"):
            agent_name = path.split("/")[-1]
            agent_file = AGENTS_DIR / f"{agent_name}.json"
            if agent_file.exists():
                agent_file.unlink()
                if agent_name in loaded_agents:
                    del loaded_agents[agent_name]
                self.send_json_response(200, {"status": "deleted", "agent": agent_name})
            else:
                self.send_json_response(404, {"error": "Agent not found"})
        
        else:
            self.send_json_response(404, {"error": "Not found"})
    
    def send_json_response(self, status_code, data):
        """Send JSON response"""
        self.send_response(status_code)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())
    
    def log_message(self, format, *args):
        """Override to suppress default logging"""
        pass

if __name__ == "__main__":
    # Create sample data if none exists
    if not list(AGENTS_DIR.glob("*.json")):
        sample_agent = {
            "name": "researcher",
            "role": "Senior Research Analyst",
            "goal": "Gather and analyze information",
            "backstory": "Expert at finding and synthesizing information"
        }
        with open(AGENTS_DIR / "researcher.json", "w") as f:
            json.dump(sample_agent, f, indent=2)
    
    if not list(CREWS_DIR.glob("*.json")):
        sample_crew = {
            "name": "research_crew",
            "agents": ["researcher"],
            "tasks": ["gather_info", "analyze_data"],
            "description": "Crew for research and analysis tasks"
        }
        with open(CREWS_DIR / "research_crew.json", "w") as f:
            json.dump(sample_crew, f, indent=2)
    
    server = HTTPServer(('0.0.0.0', PORT), CrewAIHandler)
    print(f"CrewAI Mock Server running on port {PORT}")
    server.serve_forever()
EOF
    chmod +x "${CREWAI_SERVER_FILE}"
}

# Start CrewAI service
start_crewai() {
    if is_running; then
        log::info "CrewAI is already running"
        return 0
    fi
    
    init_directories
    
    if [[ ! -f "${CREWAI_SERVER_FILE}" ]]; then
        create_server_file
    fi
    
    log::info "Starting CrewAI service (mock mode)..."
    
    # Start server directly with python3
    nohup python3 "${CREWAI_SERVER_FILE}" > "${CREWAI_LOG_FILE}" 2>&1 &
    local pid=$!
    echo $pid > "${CREWAI_PID_FILE}"
    
    sleep 2
    
    if is_running; then
        log::success "CrewAI started on port ${CREWAI_PORT}"
    else
        log::error "Failed to start CrewAI"
        return 1
    fi
}

# Stop CrewAI service
stop_crewai() {
    if [[ -f "${CREWAI_PID_FILE}" ]]; then
        local pid=$(cat "${CREWAI_PID_FILE}")
        if kill -0 "$pid" 2>/dev/null; then
            log::info "Stopping CrewAI..."
            kill "$pid"
            rm -f "${CREWAI_PID_FILE}"
            log::success "CrewAI stopped"
        else
            rm -f "${CREWAI_PID_FILE}"
            log::info "CrewAI was not running"
        fi
    else
        log::info "CrewAI is not running"
    fi
}

# Check if running
is_running() {
    if [[ -f "${CREWAI_PID_FILE}" ]]; then
        local pid=$(cat "${CREWAI_PID_FILE}")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        else
            rm -f "${CREWAI_PID_FILE}"
        fi
    fi
    
    # Check if process is listening on port
    if ss -tlnp 2>/dev/null | grep -q ":${CREWAI_PORT}"; then
        return 0
    fi
    
    return 1
}

# List crews
list_crews() {
    if [[ -d "${CREWAI_CREWS_DIR}" ]]; then
        log::header "Available Crews"
        local count=0
        for crew_file in "${CREWAI_CREWS_DIR}"/*.py; do
            if [[ -f "$crew_file" ]]; then
                basename "$crew_file" .py
                ((count++))
            fi
        done
        if [[ $count -eq 0 ]]; then
            log::info "No crews found"
        fi
    else
        log::info "Crews directory not initialized"
    fi
}

# List agents
list_agents() {
    if [[ -d "${CREWAI_AGENTS_DIR}" ]]; then
        log::header "Available Agents"
        local count=0
        for agent_file in "${CREWAI_AGENTS_DIR}"/*.py; do
            if [[ -f "$agent_file" ]]; then
                basename "$agent_file" .py
                ((count++))
            fi
        done
        if [[ $count -eq 0 ]]; then
            log::info "No agents found"
        fi
    else
        log::info "Agents directory not initialized"
    fi
}

# Get health status
get_health() {
    if is_running; then
        curl -s "http://localhost:${CREWAI_PORT}/health" 2>/dev/null || echo "{}"
    else
        echo "{}"
    fi
}