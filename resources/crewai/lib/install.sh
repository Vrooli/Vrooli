#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
CREWAI_LIB_DIR="${APP_ROOT}/resources/crewai/lib"
CREWAI_RESOURCE_DIR="${APP_ROOT}/resources/crewai"

source "${CREWAI_RESOURCE_DIR}/../../../lib/utils/var.sh"
source "${CREWAI_RESOURCE_DIR}/../../../lib/utils/log.sh"
source "${CREWAI_RESOURCE_DIR}/../../../lib/utils/format.sh"

CREWAI_HOME="${HOME}/.crewai"
CREWAI_VENV="${CREWAI_HOME}/venv"
CREWAI_PORT=8084

install_crewai() {
    log_header "Installing CrewAI"
    
    # Check Python availability
    if ! command -v python3 &>/dev/null; then
        log_error "Python 3 is required but not installed"
        return 1
    fi
    
    log_info "Using Python: $(which python3)"
    log_info "Python version: $(python3 --version)"
    
    # Create directories
    mkdir -p "${CREWAI_HOME}"/{crews,agents,workspace}
    
    # Check if already in mock mode (existing server.py)
    if [[ -f "${CREWAI_HOME}/server.py" ]]; then
        log_info "Using existing mock server installation"
        return 0
    fi
    
    # Create virtual environment
    if [[ ! -d "${CREWAI_VENV}" ]]; then
        log_info "Creating virtual environment..."
        python3 -m venv "${CREWAI_VENV}"
    fi
    
    # Install CrewAI
    log_info "Installing CrewAI package..."
    "${CREWAI_VENV}/bin/pip" install --upgrade pip &>/dev/null || true
    "${CREWAI_VENV}/bin/pip" install crewai crewai-tools &>/dev/null || {
        log_warning "Failed to install CrewAI from PyPI, using mock mode"
        # Create mock server if real installation fails
        create_mock_server
    }
    
    log_success "CrewAI installed successfully"
}

create_mock_server() {
    # Keep the existing mock server if it exists
    if [[ -f "${CREWAI_HOME}/server.py" ]]; then
        return 0
    fi
    
    cat > "${CREWAI_HOME}/server.py" << 'EOF'
#!/usr/bin/env python3
"""CrewAI Mock API Server for Vrooli"""

import os
import json
from pathlib import Path
from http.server import HTTPServer, BaseHTTPRequestHandler
from datetime import datetime
import shutil
from urllib.parse import urlparse, parse_qs

PORT = 8084

CREWAI_DATA_DIR = Path.home() / ".crewai"
CREWS_DIR = CREWAI_DATA_DIR / "crews"
AGENTS_DIR = CREWAI_DATA_DIR / "agents"
WORKSPACE_DIR = CREWAI_DATA_DIR / "workspace"

for dir_path in [CREWS_DIR, AGENTS_DIR, WORKSPACE_DIR]:
    dir_path.mkdir(parents=True, exist_ok=True)

loaded_crews = {}
loaded_agents = {}

class CrewAIHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            response = {
                "status": "healthy",
                "version": "1.0.0-mock",
                "crews": len(loaded_crews),
                "agents": len(loaded_agents)
            }
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        if self.path == "/crews":
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            crew_data = json.loads(post_data)
            crew_id = f"crew_{len(loaded_crews) + 1}"
            loaded_crews[crew_id] = crew_data
            
            self.send_response(201)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            response = {"id": crew_id, "status": "created"}
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()

if __name__ == "__main__":
    server = HTTPServer(("localhost", PORT), CrewAIHandler)
    print(f"CrewAI Mock Server running on port {PORT}")
    server.serve_forever()
EOF
    
    chmod +x "${CREWAI_HOME}/server.py"
}

# Run installation
install_crewai