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

# Configuration
CREWAI_NAME="crewai"
CREWAI_DATA_DIR="${HOME}/.crewai"
CREWAI_WORKSPACE_DIR="${CREWAI_DATA_DIR}/workspace"
CREWAI_CREWS_DIR="${CREWAI_DATA_DIR}/crews"
CREWAI_AGENTS_DIR="${CREWAI_DATA_DIR}/agents"
CREWAI_PID_FILE="${CREWAI_DATA_DIR}/crewai.pid"
CREWAI_LOG_FILE="${CREWAI_DATA_DIR}/crewai.log"
CREWAI_PORT=8084
CREWAI_SERVER_FILE="${CREWAI_DATA_DIR}/server.py"
CREWAI_MOCK_MODE="true"  # Run in mock mode until venv issue resolved

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
from pathlib import Path
from http.server import HTTPServer, BaseHTTPRequestHandler
from datetime import datetime
import shutil
from urllib.parse import urlparse, parse_qs

# Configuration
PORT = 8084

# Paths
CREWAI_DATA_DIR = Path.home() / ".crewai"
CREWS_DIR = CREWAI_DATA_DIR / "crews"
AGENTS_DIR = CREWAI_DATA_DIR / "agents"
WORKSPACE_DIR = CREWAI_DATA_DIR / "workspace"

# Ensure directories exist
for dir_path in [CREWS_DIR, AGENTS_DIR, WORKSPACE_DIR]:
    dir_path.mkdir(parents=True, exist_ok=True)

# Store loaded crews and agents
loaded_crews = {}
loaded_agents = {}

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
                "workspace": str(WORKSPACE_DIR)
            })
        elif path == "/health":
            self.send_json_response(200, {
                "status": "healthy",
                "timestamp": datetime.utcnow().isoformat(),
                "crews_loaded": len(loaded_crews),
                "agents_loaded": len(loaded_agents)
            })
        elif path == "/crews":
            crews = []
            for crew_file in CREWS_DIR.glob("*.py"):
                crew_name = crew_file.stem
                crews.append({
                    "name": crew_name,
                    "loaded": crew_name in loaded_crews,
                    "path": str(crew_file)
                })
            self.send_json_response(200, {"crews": crews})
        elif path == "/agents":
            agents = []
            for agent_file in AGENTS_DIR.glob("*.py"):
                agent_name = agent_file.stem
                agents.append({
                    "name": agent_name,
                    "loaded": agent_name in loaded_agents,
                    "path": str(agent_file)
                })
            self.send_json_response(200, {"agents": agents})
        else:
            self.send_json_response(404, {"error": "Not found"})
    
    def do_POST(self):
        """Handle POST requests"""
        parsed_path = urlparse(self.path)
        path = parsed_path.path
        
        if path == "/inject":
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            data = json.loads(post_data)
            
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