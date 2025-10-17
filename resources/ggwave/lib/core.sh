#!/bin/bash
# GGWave Core Library Functions

set -euo pipefail

# Install GGWave
ggwave::install() {
    local force=false
    
    for arg in "$@"; do
        case "$arg" in
            --force) force=true ;;
        esac
    done
    
    echo "Installing GGWave resource..."
    
    # Check if already installed
    if docker ps -a --format '{{.Names}}' | grep -q "^${GGWAVE_CONTAINER_NAME}$"; then
        if [[ "$force" != "true" ]]; then
            echo "GGWave is already installed. Use --force to reinstall."
            return 2
        fi
        echo "Force reinstalling GGWave..."
        ggwave::uninstall --force
    fi
    
    # Ensure network exists
    if ! docker network ls --format '{{.Name}}' | grep -q "^${GGWAVE_NETWORK}$"; then
        echo "Creating Docker network: ${GGWAVE_NETWORK}"
        docker network create "${GGWAVE_NETWORK}" || true
    fi
    
    # Build Docker image
    echo "Building GGWave Docker image..."
    
    # Build the image from the Dockerfile
    if ! docker build -t "vrooli-ggwave:latest" "${GGWAVE_DIR}/docker"; then
        echo "Error: Failed to build Docker image"
        return 1
    fi
    
    # For backwards compatibility, still create the server.py file
    cat > "${GGWAVE_DATA_DIR}/server.py" << 'EOF'
#!/usr/bin/env python3
import http.server
import json
import base64
from urllib.parse import urlparse

class GGWaveHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                "status": "healthy",
                "version": "0.4.0",
                "modes_available": ["normal", "fast", "dt", "ultrasonic"]
            }
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        try:
            content_length = int(self.headers.get('Content-Length', 0))
            post_data = self.rfile.read(content_length) if content_length > 0 else b'{}'
            
            if self.path == '/api/encode':
                data = json.loads(post_data.decode('utf-8'))
                input_text = data.get('data', '')
                mode = data.get('mode', 'normal')
                
                # Simulate FSK encoding with actual data transformation
                # In production, this would use real GGWave library
                encoded_data = base64.b64encode(input_text.encode()).decode()
                
                response = {
                    "audio": encoded_data,
                    "duration_ms": len(input_text) * 100,  # Roughly 100ms per byte
                    "mode": mode,
                    "bytes": len(input_text)
                }
                
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(response).encode())
                
            elif self.path == '/api/decode':
                data = json.loads(post_data.decode('utf-8'))
                audio_data = data.get('audio', '')
                mode = data.get('mode', 'auto')
                
                # Simulate FSK decoding
                try:
                    decoded_text = base64.b64decode(audio_data).decode('utf-8')
                except:
                    decoded_text = "Error decoding"
                
                response = {
                    "data": decoded_text,
                    "confidence": 0.95,
                    "mode_detected": mode if mode != 'auto' else 'normal',
                    "errors_corrected": 0
                }
                
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(response).encode())
                
            else:
                self.send_response(404)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps({"error": "Not found"}).encode())
                
        except Exception as e:
            self.send_response(500)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"error": str(e)}).encode())
    
    def log_message(self, format, *args):
        # Log to stdout for Docker logs
        import sys
        sys.stdout.write("%s - - [%s] %s\n" %
                         (self.address_string(),
                          self.log_date_time_string(),
                          format % args))
        sys.stdout.flush()

if __name__ == '__main__':
    server = http.server.HTTPServer(('0.0.0.0', 8196), GGWaveHandler)
    print('GGWave API server running on port 8196')
    server.serve_forever()
EOF
    
    # Run container with built image
    docker run -d \
        --name "${GGWAVE_CONTAINER_NAME}" \
        --network "${GGWAVE_NETWORK}" \
        -p "${GGWAVE_PORT}:8196" \
        -v "${GGWAVE_DATA_DIR}:/data" \
        --restart unless-stopped \
        vrooli-ggwave:latest
    
    echo "GGWave installed successfully"
    return 0
}

# Uninstall GGWave
ggwave::uninstall() {
    local force=false
    local keep_data=false
    
    for arg in "$@"; do
        case "$arg" in
            --force) force=true ;;
            --keep-data) keep_data=true ;;
        esac
    done
    
    echo "Uninstalling GGWave resource..."
    
    # Stop container if running
    if docker ps --format '{{.Names}}' | grep -q "^${GGWAVE_CONTAINER_NAME}$"; then
        echo "Stopping GGWave container..."
        docker stop "${GGWAVE_CONTAINER_NAME}" &>/dev/null || true
    fi
    
    # Remove container
    if docker ps -a --format '{{.Names}}' | grep -q "^${GGWAVE_CONTAINER_NAME}$"; then
        echo "Removing GGWave container..."
        docker rm "${GGWAVE_CONTAINER_NAME}" &>/dev/null || true
    fi
    
    # Remove data if not keeping
    if [[ "$keep_data" != "true" ]] && [[ -d "${GGWAVE_DATA_DIR}" ]]; then
        if [[ "$force" == "true" ]] || ggwave::confirm "Remove GGWave data directory?"; then
            echo "Removing data directory..."
            rm -rf "${GGWAVE_DATA_DIR}"
        fi
    fi
    
    echo "GGWave uninstalled successfully"
    return 0
}

# Start GGWave service
ggwave::start() {
    local wait=false
    
    for arg in "$@"; do
        case "$arg" in
            --wait) wait=true ;;
        esac
    done
    
    echo "Starting GGWave service..."
    
    # Check if container exists
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${GGWAVE_CONTAINER_NAME}$"; then
        echo "Error: GGWave is not installed. Run 'manage install' first."
        return 1
    fi
    
    # Start container
    docker start "${GGWAVE_CONTAINER_NAME}"
    
    if [[ "$wait" == "true" ]]; then
        echo "Waiting for GGWave to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if ggwave::health_check; then
                echo "GGWave is ready"
                return 0
            fi
            sleep 1
            ((attempt++))
        done
        
        echo "Error: GGWave failed to start within ${max_attempts} seconds"
        return 1
    fi
    
    echo "GGWave started"
    return 0
}

# Stop GGWave service
ggwave::stop() {
    local force=false
    
    for arg in "$@"; do
        case "$arg" in
            --force) force=true ;;
        esac
    done
    
    echo "Stopping GGWave service..."
    
    if docker ps --format '{{.Names}}' | grep -q "^${GGWAVE_CONTAINER_NAME}$"; then
        if [[ "$force" == "true" ]]; then
            docker kill "${GGWAVE_CONTAINER_NAME}" &>/dev/null || true
        else
            docker stop "${GGWAVE_CONTAINER_NAME}" &>/dev/null || true
        fi
        echo "GGWave stopped"
    else
        echo "GGWave is not running"
    fi
    
    return 0
}

# Restart GGWave service
ggwave::restart() {
    echo "Restarting GGWave service..."
    ggwave::stop
    ggwave::start "$@"
}

# Check health status
ggwave::health_check() {
    timeout 5 curl -sf "http://localhost:${GGWAVE_PORT}/health" &>/dev/null
}

# Show service status
ggwave::status() {
    local verbose=false
    local json_output=false
    
    for arg in "$@"; do
        case "$arg" in
            --verbose) verbose=true ;;
            --json) json_output=true ;;
        esac
    done
    
    # Check container status
    local container_status="not_installed"
    local health_status="unknown"
    
    if docker ps -a --format '{{.Names}}' | grep -q "^${GGWAVE_CONTAINER_NAME}$"; then
        if docker ps --format '{{.Names}}' | grep -q "^${GGWAVE_CONTAINER_NAME}$"; then
            container_status="running"
            if ggwave::health_check; then
                health_status="healthy"
            else
                health_status="unhealthy"
            fi
        else
            container_status="stopped"
        fi
    fi
    
    if [[ "$json_output" == "true" ]]; then
        cat << EOF
{
  "service": "ggwave",
  "version": "${GGWAVE_VERSION}",
  "container_status": "${container_status}",
  "health_status": "${health_status}",
  "port": ${GGWAVE_PORT},
  "mode": "${GGWAVE_MODE}"
}
EOF
    else
        echo "=== GGWave Service Status ==="
        echo "Container: ${container_status}"
        echo "Health: ${health_status}"
        echo "Port: ${GGWAVE_PORT}"
        echo "Mode: ${GGWAVE_MODE}"
        
        if [[ "$verbose" == "true" ]] && [[ "$container_status" == "running" ]]; then
            echo ""
            echo "Container Details:"
            docker ps --filter "name=${GGWAVE_CONTAINER_NAME}" --format "table {{.Status}}\t{{.Ports}}"
        fi
    fi
}

# Show service logs
ggwave::logs() {
    local follow=false
    local lines=50
    
    for arg in "$@"; do
        case "$arg" in
            --follow|-f) follow=true ;;
            --lines=*) lines="${arg#*=}" ;;
        esac
    done
    
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${GGWAVE_CONTAINER_NAME}$"; then
        echo "Error: GGWave container not found"
        return 1
    fi
    
    if [[ "$follow" == "true" ]]; then
        docker logs -f "${GGWAVE_CONTAINER_NAME}"
    else
        docker logs --tail "${lines}" "${GGWAVE_CONTAINER_NAME}"
    fi
}

# Content management functions
ggwave::content::add() {
    local data=""
    local mode="${GGWAVE_MODE}"
    
    for arg in "$@"; do
        case "$arg" in
            --data=*) data="${arg#*=}" ;;
            --mode=*) mode="${arg#*=}" ;;
        esac
    done
    
    if [[ -z "$data" ]]; then
        echo "Error: --data parameter required"
        return 1
    fi
    
    echo "Adding data for transmission: $data"
    # In production, this would store data for later transmission
    echo "$data" > "${GGWAVE_DATA_DIR}/pending_transmission.txt"
    echo "Data queued for transmission"
}

ggwave::content::list() {
    echo "=== Pending Transmissions ==="
    if [[ -f "${GGWAVE_DATA_DIR}/pending_transmission.txt" ]]; then
        cat "${GGWAVE_DATA_DIR}/pending_transmission.txt"
    else
        echo "No pending transmissions"
    fi
}

ggwave::content::get() {
    echo "=== Received Data ==="
    if [[ -f "${GGWAVE_DATA_DIR}/received_data.txt" ]]; then
        cat "${GGWAVE_DATA_DIR}/received_data.txt"
    else
        echo "No received data"
    fi
}

ggwave::content::remove() {
    echo "Clearing transmission queues..."
    rm -f "${GGWAVE_DATA_DIR}/pending_transmission.txt"
    rm -f "${GGWAVE_DATA_DIR}/received_data.txt"
    echo "Queues cleared"
}

ggwave::content::execute() {
    local data=""
    local mode="${GGWAVE_MODE}"
    
    for arg in "$@"; do
        case "$arg" in
            --data=*) data="${arg#*=}" ;;
            --mode=*) mode="${arg#*=}" ;;
        esac
    done
    
    if [[ -z "$data" ]]; then
        echo "Error: --data parameter required"
        return 1
    fi
    
    echo "Transmitting data via $mode mode: $data"
    echo "[Simulated] Encoding data to audio signal..."
    echo "[Simulated] Playing audio through speaker..."
    echo "[Simulated] Transmission complete"
    
    # In production, this would actually encode and play the audio
    return 0
}

# Utility function for confirmation
ggwave::confirm() {
    local prompt="${1:-Continue?}"
    read -p "${prompt} [y/N] " -n 1 -r
    echo
    [[ $REPLY =~ ^[Yy]$ ]]
}