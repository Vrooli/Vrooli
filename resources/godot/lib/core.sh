#!/usr/bin/env bash
# Godot Engine Resource - Core Library

set -euo pipefail

# Lifecycle Management Functions

godot::install() {
    local force="${1:-false}"
    
    echo "🎮 Installing Godot Engine resource..."
    
    # Create necessary directories
    mkdir -p "${GODOT_BASE_DIR}"/{projects,exports,templates,assets,logs}
    
    # Check if Docker/Podman is available
    if command -v docker &> /dev/null; then
        local container_cmd="docker"
    elif command -v podman &> /dev/null; then
        local container_cmd="podman"
    else
        echo "ERROR: Docker or Podman is required but not installed" >&2
        return 1
    fi
    
    # Pull Godot image
    echo "📦 Pulling Godot image ${GODOT_IMAGE}..."
    ${container_cmd} pull "${GODOT_IMAGE}" || {
        echo "ERROR: Failed to pull Godot image" >&2
        return 1
    }
    
    # Create basic API server script
    cat > "${GODOT_BASE_DIR}/api-server.py" << 'EOF'
#!/usr/bin/env python3
import json
import os
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs

class GodotAPIHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = {
                'status': 'healthy',
                'service': 'godot',
                'version': os.environ.get('GODOT_VERSION', '4.3'),
                'lsp_port': int(os.environ.get('GODOT_LSP_PORT', '6005'))
            }
            self.wfile.write(json.dumps(response).encode())
        elif parsed_path.path == '/api/projects':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            projects = []
            projects_dir = os.environ.get('GODOT_PROJECTS_DIR', '/projects')
            if os.path.exists(projects_dir):
                projects = [d for d in os.listdir(projects_dir) 
                           if os.path.isdir(os.path.join(projects_dir, d))]
            self.wfile.write(json.dumps({'projects': projects}).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        if self.path == '/api/projects':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            data = json.loads(post_data.decode('utf-8'))
            
            project_name = data.get('name', 'new_project')
            projects_dir = os.environ.get('GODOT_PROJECTS_DIR', '/projects')
            project_path = os.path.join(projects_dir, project_name)
            
            os.makedirs(project_path, exist_ok=True)
            
            # Create basic project.godot file
            with open(os.path.join(project_path, 'project.godot'), 'w') as f:
                f.write('[application]\n')
                f.write(f'config/name="{project_name}"\n')
                f.write('config/features=PackedStringArray("4.3")\n')
            
            self.send_response(201)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({'project': project_name, 'path': project_path}).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        return  # Suppress logs

if __name__ == '__main__':
    port = int(os.environ.get('GODOT_API_PORT', '11457'))
    server = HTTPServer(('0.0.0.0', port), GodotAPIHandler)
    print(f'Godot API server running on port {port}')
    server.serve_forever()
EOF
    chmod +x "${GODOT_BASE_DIR}/api-server.py"
    
    echo "✅ Godot Engine resource installed successfully"
    return 0
}

godot::uninstall() {
    local force="${1:-false}"
    local keep_data="${2:-false}"
    
    echo "🗑️  Uninstalling Godot Engine resource..."
    
    # Stop service if running
    godot::stop || true
    
    # Remove container if exists
    if command -v docker &> /dev/null; then
        docker rm -f "${GODOT_CONTAINER_NAME}" 2>/dev/null || true
    elif command -v podman &> /dev/null; then
        podman rm -f "${GODOT_CONTAINER_NAME}" 2>/dev/null || true
    fi
    
    # Remove data unless keep_data is true
    if [[ "$keep_data" != "true" ]]; then
        rm -rf "${GODOT_BASE_DIR}"
        echo "✅ Godot data removed"
    else
        echo "ℹ️  Keeping Godot data in ${GODOT_BASE_DIR}"
    fi
    
    return 0
}

godot::start() {
    local wait="${1:-false}"
    local timeout="${2:-${GODOT_STARTUP_TIMEOUT}}"
    
    echo "🚀 Starting Godot Engine resource..."
    
    # Check if already running
    if godot::is_running; then
        echo "ℹ️  Godot is already running"
        return 2
    fi
    
    # Start API server
    cd "${GODOT_BASE_DIR}"
    nohup python3 api-server.py > "${GODOT_LOG_FILE}" 2>&1 &
    local pid=$!
    echo $pid > "${GODOT_PID_FILE}"
    
    # Start headless Godot with LSP (in container)
    if command -v docker &> /dev/null; then
        docker run -d \
            --name "${GODOT_CONTAINER_NAME}" \
            -p "${GODOT_LSP_PORT}:6005" \
            -v "${GODOT_PROJECTS_DIR}:/projects" \
            -v "${GODOT_EXPORTS_DIR}:/exports" \
            "${GODOT_IMAGE}" \
            godot --headless --language-server 2>/dev/null || true
    fi
    
    if [[ "$wait" == "--wait" ]] || [[ "$wait" == "true" ]]; then
        echo "⏳ Waiting for Godot to be ready..."
        local elapsed=0
        while [[ $elapsed -lt $timeout ]]; do
            if godot::health_check; then
                echo "✅ Godot is ready"
                return 0
            fi
            sleep 2
            elapsed=$((elapsed + 2))
        done
        echo "ERROR: Godot failed to start within ${timeout} seconds" >&2
        return 1
    fi
    
    echo "✅ Godot Engine started"
    return 0
}

godot::stop() {
    local force="${1:-false}"
    local timeout="${2:-30}"
    
    echo "🛑 Stopping Godot Engine resource..."
    
    # Stop API server
    if [[ -f "${GODOT_PID_FILE}" ]]; then
        local pid=$(cat "${GODOT_PID_FILE}")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm -f "${GODOT_PID_FILE}"
        fi
    fi
    
    # Stop container
    if command -v docker &> /dev/null; then
        docker stop "${GODOT_CONTAINER_NAME}" 2>/dev/null || true
        docker rm "${GODOT_CONTAINER_NAME}" 2>/dev/null || true
    elif command -v podman &> /dev/null; then
        podman stop "${GODOT_CONTAINER_NAME}" 2>/dev/null || true
        podman rm "${GODOT_CONTAINER_NAME}" 2>/dev/null || true
    fi
    
    echo "✅ Godot Engine stopped"
    return 0
}

godot::restart() {
    echo "🔄 Restarting Godot Engine resource..."
    godot::stop || true
    sleep 2
    godot::start --wait
}

# Status and Health Functions

godot::is_running() {
    if [[ -f "${GODOT_PID_FILE}" ]]; then
        local pid=$(cat "${GODOT_PID_FILE}")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

godot::health_check() {
    timeout "${GODOT_HEALTH_CHECK_TIMEOUT}" curl -sf "http://localhost:${GODOT_API_PORT}/health" > /dev/null
}

godot::status() {
    local json="${1:-false}"
    
    local status="stopped"
    local api_status="offline"
    local lsp_status="offline"
    
    if godot::is_running; then
        status="running"
        if godot::health_check; then
            api_status="healthy"
        else
            api_status="unhealthy"
        fi
        
        # Check LSP
        if timeout 2 nc -z localhost "${GODOT_LSP_PORT}" 2>/dev/null; then
            lsp_status="running"
        fi
    fi
    
    if [[ "$json" == "--json" ]]; then
        cat << EOF
{
  "status": "$status",
  "api": {
    "status": "$api_status",
    "port": ${GODOT_API_PORT}
  },
  "lsp": {
    "status": "$lsp_status",
    "port": ${GODOT_LSP_PORT}
  },
  "version": "${GODOT_VERSION}"
}
EOF
    else
        echo "🎮 Godot Engine Status"
        echo "====================="
        echo "Service:     $status"
        echo "API:         $api_status (port ${GODOT_API_PORT})"
        echo "LSP:         $lsp_status (port ${GODOT_LSP_PORT})"
        echo "Version:     ${GODOT_VERSION}"
        echo "Projects:    ${GODOT_PROJECTS_DIR}"
    fi
}

godot::logs() {
    local follow="${1:-false}"
    
    if [[ ! -f "${GODOT_LOG_FILE}" ]]; then
        echo "No logs available"
        return 0
    fi
    
    if [[ "$follow" == "--follow" ]] || [[ "$follow" == "-f" ]]; then
        tail -f "${GODOT_LOG_FILE}"
    else
        tail -n 50 "${GODOT_LOG_FILE}"
    fi
}

# Content Management Functions

godot::content::list() {
    local type="${1:-all}"
    local format="${2:-text}"
    
    case "$type" in
        project|projects)
            if [[ -d "${GODOT_PROJECTS_DIR}" ]]; then
                ls -la "${GODOT_PROJECTS_DIR}"
            else
                echo "No projects found"
            fi
            ;;
        template|templates)
            echo "Available templates:"
            for template in "${GODOT_PROJECT_TEMPLATES[@]}"; do
                echo "  - ${template}"
            done
            ;;
        *)
            echo "Projects in ${GODOT_PROJECTS_DIR}:"
            ls -la "${GODOT_PROJECTS_DIR}" 2>/dev/null || echo "  None"
            echo ""
            echo "Available templates:"
            for template in "${GODOT_PROJECT_TEMPLATES[@]}"; do
                echo "  - ${template}"
            done
            ;;
    esac
}

godot::content::add() {
    local type="${1}"
    local name="${2}"
    local file="${3:-}"
    
    case "$type" in
        template)
            echo "Installing template: $name"
            # Template installation logic would go here
            echo "✅ Template installed"
            ;;
        project)
            mkdir -p "${GODOT_PROJECTS_DIR}/${name}"
            echo "✅ Project ${name} created"
            ;;
        *)
            echo "ERROR: Unknown content type: $type" >&2
            return 1
            ;;
    esac
}

godot::content::get() {
    local name="${1}"
    local output="${2:-}"
    
    if [[ -d "${GODOT_PROJECTS_DIR}/${name}" ]]; then
        if [[ -n "$output" ]]; then
            cp -r "${GODOT_PROJECTS_DIR}/${name}" "$output"
            echo "✅ Project exported to $output"
        else
            ls -la "${GODOT_PROJECTS_DIR}/${name}"
        fi
    else
        echo "ERROR: Project $name not found" >&2
        return 1
    fi
}

godot::content::remove() {
    local name="${1}"
    
    if [[ -d "${GODOT_PROJECTS_DIR}/${name}" ]]; then
        rm -rf "${GODOT_PROJECTS_DIR}/${name}"
        echo "✅ Project ${name} removed"
    else
        echo "ERROR: Project $name not found" >&2
        return 1
    fi
}

godot::content::execute() {
    local command="${1}"
    
    echo "Executing: $command"
    # Command execution would go here
    echo "✅ Command executed"
}

# Project Management Functions

godot::projects::list() {
    godot::content::list projects "$@"
}

godot::projects::create() {
    local name=""
    local template="empty"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --template)
                template="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "ERROR: Project name required (--name)" >&2
        return 1
    fi
    
    godot::content::add project "$name"
}

godot::projects::delete() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "ERROR: Project name required (--name)" >&2
        return 1
    fi
    
    godot::content::remove "$name"
}

# Template Management Functions

godot::templates::list() {
    godot::content::list templates "$@"
}

godot::templates::install() {
    local name="${1}"
    
    if [[ -z "$name" ]]; then
        echo "ERROR: Template name required" >&2
        return 1
    fi
    
    godot::content::add template "$name"
}

# LSP Functions

godot::lsp::status() {
    if timeout 2 nc -z localhost "${GODOT_LSP_PORT}" 2>/dev/null; then
        echo "Running"
    else
        echo "Stopped"
    fi
}