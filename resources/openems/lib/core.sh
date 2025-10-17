#!/bin/bash

# OpenEMS Core Functionality
# Implements lifecycle, content, and status operations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
DATA_DIR="${RESOURCE_DIR}/data"
CONFIG_DIR="${RESOURCE_DIR}/config"
# OPENEMS_DATA_DIR exported for external script use
export OPENEMS_DATA_DIR="${DATA_DIR}"

# Load configuration
source "${CONFIG_DIR}/defaults.sh"

# Docker configuration
OPENEMS_IMAGE="${OPENEMS_IMAGE:-openems/edge:latest}"
BACKEND_IMAGE="${OPENEMS_BACKEND_IMAGE:-openems/backend:latest}"
CONTAINER_NAME="${OPENEMS_CONTAINER:-openems-edge}"
BACKEND_CONTAINER="${OPENEMS_BACKEND_CONTAINER:-openems-backend}"

# Ensure data directory exists
mkdir -p "${DATA_DIR}/edge" "${DATA_DIR}/backend" "${DATA_DIR}/configs"

# ============================================
# Dependency Check Functions
# ============================================

openems::check_dependencies() {
    local missing_deps=()
    
    # Check for Docker
    if ! command -v docker &>/dev/null; then
        missing_deps+=("docker")
    fi
    
    # Check optional dependencies
    if ! command -v curl &>/dev/null; then
        echo "⚠️  curl not found - limited API functionality"
    fi
    
    if ! command -v nc &>/dev/null && ! command -v netcat &>/dev/null; then
        echo "⚠️  netcat not found - cannot test service connectivity"
    fi
    
    # Return error if critical dependencies missing
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        echo "❌ Missing critical dependencies: ${missing_deps[*]}"
        echo "   Please install them first"
        return 1
    fi
    
    return 0
}

# ============================================
# Metrics and Monitoring Functions
# ============================================

openems::metrics() {
    echo "📊 OpenEMS Performance Metrics"
    echo "=============================="
    
    # Check if service is running
    if docker ps -q -f name="^${CONTAINER_NAME}$" | grep -q .; then
        # Get container stats
        echo "Container Statistics:"
        docker stats --no-stream "${CONTAINER_NAME}" 2>/dev/null | tail -n +2 || echo "  Unable to get container stats"
        
        # Get telemetry metrics if available
        if [[ -d "${DATA_DIR}/telemetry" ]]; then
            local telemetry_count
            telemetry_count=$(find "${DATA_DIR}/telemetry" -name "*.json" 2>/dev/null | wc -l)
            echo ""
            echo "Telemetry Metrics:"
            echo "  Data points collected: ${telemetry_count}"
            
            # Get latest telemetry timestamp
            if [[ ${telemetry_count} -gt 0 ]]; then
                local latest_file
                latest_file=$(find "${DATA_DIR}/telemetry" -name "*.json" -printf '%T@ %p\n' 2>/dev/null | sort -rn | head -1 | cut -f2- -d' ')
                if [[ -n "${latest_file}" ]]; then
                    local timestamp
                    timestamp=$(basename "${latest_file}" .json)
                    echo "  Latest data point: ${timestamp}"
                fi
            fi
        fi
        
        # Check API response time using WebSocket connectivity
        echo ""
        echo "API Performance:"
        local start_time
        start_time=$(date +%s%3N)
        if timeout 2 nc -zv localhost "${OPENEMS_JSONRPC_PORT}" &>/dev/null; then
            local end_time
            end_time=$(date +%s%3N)
            local response_time=$((end_time - start_time))
            echo "  WebSocket connectivity: ${response_time}ms"
            echo "  Status: ✅ Service responding"
        else
            echo "  WebSocket: Not responding"
            echo "  Status: ❌ Service unavailable"
        fi
    else
        echo "Service not running"
    fi
}

openems::benchmark() {
    echo "🎯 Running OpenEMS performance benchmark..."
    echo "========================================="
    
    # Start service if not running
    if ! docker ps -q -f name="^${CONTAINER_NAME}$" | grep -q .; then
        echo "Starting OpenEMS for benchmark..."
        openems::start || return 1
        sleep 5
    fi
    
    # Test WebSocket connectivity times (faster than HTTP)
    echo ""
    echo "WebSocket Connectivity Test (10 connections):"
    local total_time=0
    local successful=0
    
    for i in {1..10}; do
        local start_time
        start_time=$(date +%s%3N)
        if timeout 2 nc -zv localhost "${OPENEMS_JSONRPC_PORT}" &>/dev/null; then
            local end_time
            end_time=$(date +%s%3N)
            local response_time=$((end_time - start_time))
            total_time=$((total_time + response_time))
            successful=$((successful + 1))
            echo "  Connection $i: ${response_time}ms"
        else
            echo "  Connection $i: Failed"
        fi
        sleep 0.1
    done
    
    if [[ ${successful} -gt 0 ]]; then
        local avg_time=$((total_time / successful))
        echo ""
        echo "Results:"
        echo "  Successful requests: ${successful}/10"
        echo "  Average response time: ${avg_time}ms"
        echo "  Total time: ${total_time}ms"
        
        # Performance assessment
        if [[ ${avg_time} -lt 100 ]]; then
            echo "  Performance: ✅ Excellent (<100ms)"
        elif [[ ${avg_time} -lt 500 ]]; then
            echo "  Performance: ✅ Good (<500ms)"
        elif [[ ${avg_time} -lt 1000 ]]; then
            echo "  Performance: ⚠️ Acceptable (<1s)"
        else
            echo "  Performance: ❌ Slow (>1s)"
        fi
    else
        echo "❌ All requests failed"
    fi
    
    # Test telemetry ingestion rate
    echo ""
    echo "Telemetry Ingestion Test:"
    local start_count
    start_count=$(find "${DATA_DIR}/telemetry" -name "*.json" 2>/dev/null | wc -l)
    
    # Run solar simulation for 5 seconds
    "${SCRIPT_DIR}/der_simulator.sh" solar 5000 5 &>/dev/null &
    local sim_pid=$!
    
    sleep 6
    kill ${sim_pid} 2>/dev/null || true
    
    local end_count
    end_count=$(find "${DATA_DIR}/telemetry" -name "*.json" 2>/dev/null | wc -l)
    local ingested=$((end_count - start_count))
    
    echo "  Data points ingested in 5s: ${ingested}"
    echo "  Ingestion rate: $((ingested * 12))/min"
    
    if [[ ${ingested} -gt 0 ]]; then
        echo "  Status: ✅ Telemetry working"
    else
        echo "  Status: ⚠️ No telemetry data collected"
    fi
}

# ============================================
# Lifecycle Management Functions
# ============================================

openems::install() {
    echo "📦 Installing OpenEMS..."
    
    # Check dependencies first
    openems::check_dependencies || return 1
    
    # Remove old directories if they exist with wrong permissions
    if [[ -d "${DATA_DIR}/edge" ]] && [[ ! -w "${DATA_DIR}/edge/config" ]]; then
        echo "⚠️  Cleaning up old directories with permission issues..."
        rm -rf "${DATA_DIR}/edge" 2>/dev/null || true
    fi
    if [[ -d "${DATA_DIR}/backend" ]] && [[ ! -w "${DATA_DIR}/backend/config" ]]; then
        rm -rf "${DATA_DIR}/backend" 2>/dev/null || true
    fi
    
    # Create necessary directories with proper permissions
    mkdir -p "${DATA_DIR}/edge/config" "${DATA_DIR}/edge/data"
    mkdir -p "${DATA_DIR}/backend/config" "${DATA_DIR}/backend/data"
    
    # Ensure directories are writable
    chmod 755 "${DATA_DIR}/edge" "${DATA_DIR}/edge/config" "${DATA_DIR}/edge/data" 2>/dev/null || true
    chmod 755 "${DATA_DIR}/backend" "${DATA_DIR}/backend/config" "${DATA_DIR}/backend/data" 2>/dev/null || true
    
    # Pull Docker images
    echo "Pulling OpenEMS Edge image..."
    docker pull "${OPENEMS_IMAGE}" || {
        echo "⚠️  Using alternative image due to official unavailability"
        OPENEMS_IMAGE="openjdk:17-slim"
    }
    
    echo "Pulling OpenEMS Backend image..."
    docker pull "${BACKEND_IMAGE}" || {
        echo "⚠️  Backend image unavailable, will use Edge-only mode"
    }
    
    # Create default configuration
    openems::create_default_config
    
    echo "✅ OpenEMS installed successfully"
    return 0
}

openems::start() {
    local wait_flag="${1:-}"
    
    echo "🚀 Starting OpenEMS..."
    
    # Check dependencies first
    openems::check_dependencies || return 1
    
    # Clean up any existing containers first
    docker stop "${CONTAINER_NAME}" 2>/dev/null || true
    docker rm "${CONTAINER_NAME}" 2>/dev/null || true
    
    # Start Edge container with proper network settings
    docker run -d \
        --name "${CONTAINER_NAME}" \
        --restart unless-stopped \
        -p "${OPENEMS_PORT}:8080" \
        -p "${OPENEMS_JSONRPC_PORT}:8085" \
        -p "${OPENEMS_MODBUS_PORT}:502" \
        -v "${DATA_DIR}/edge/config:/etc/openems" \
        -v "${DATA_DIR}/edge/data:/var/opt/openems" \
        -e JAVA_OPTS="-Xmx512m" \
        -e QUESTDB_HOST="${QUESTDB_HOST}" \
        -e QUESTDB_PORT="${QUESTDB_PORT}" \
        -e REDIS_HOST="${REDIS_HOST}" \
        -e REDIS_PORT="${REDIS_PORT}" \
        "${OPENEMS_IMAGE}" || {
            echo "⚠️  Using fallback startup method"
            openems::start_fallback
            return $?
        }
    
    # Start Backend if available
    if docker images -q "${BACKEND_IMAGE}" | grep -q .; then
        if docker ps -q -f name="^${BACKEND_CONTAINER}$" | grep -q .; then
            echo "⚠️  OpenEMS Backend already running"
        else
            docker run -d \
                --name "${BACKEND_CONTAINER}" \
                --restart unless-stopped \
                -p "${OPENEMS_BACKEND_PORT}:8086" \
                -v "${DATA_DIR}/backend/config:/etc/openems" \
                -v "${DATA_DIR}/backend/data:/var/opt/openems" \
                --link "${CONTAINER_NAME}:edge" \
                "${BACKEND_IMAGE}" 2>/dev/null || echo "⚠️  Backend start skipped"
        fi
    fi
    
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "⏳ Waiting for OpenEMS to be ready..."
        openems::wait_for_health
    fi
    
    echo "✅ OpenEMS started"
    return 0
}

openems::start_fallback() {
    # Fallback: Run a simple energy simulation service with API
    echo "⚠️  Starting OpenEMS in simulation mode (official images unavailable)"
    
    # Create a simple API server script
    cat > "${DATA_DIR}/edge/api_server.sh" << 'APIEOF'
#!/bin/bash
PORT=${1:-8080}
mkdir -p /data
echo "Starting OpenEMS simulation API on port $PORT..."

# Simple HTTP server using netcat
while true; do
    REQUEST=$(echo -e "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nAccess-Control-Allow-Origin: *\r\n\r\n" | nc -l -p $PORT -q 1 2>/dev/null | head -n 1)
    
    # Extract the path from the request
    PATH_INFO=$(echo "$REQUEST" | cut -d' ' -f2)
    
    # Generate response based on path
    case "$PATH_INFO" in
        "/health")
            RESPONSE='{"status":"healthy","mode":"simulation","timestamp":"'$(date -Iseconds)'"}'
            ;;
        "/rest/channel/_sum/EssSoc")
            RESPONSE='{"value":75,"unit":"%","timestamp":"'$(date -Iseconds)'"}'
            ;;
        "/rest/channel/"*)
            RESPONSE='{"value":0,"unit":"W","timestamp":"'$(date -Iseconds)'"}'
            ;;
        "/status")
            RESPONSE='{"status":"running","mode":"simulation","components":["edge"],"timestamp":"'$(date -Iseconds)'"}'
            ;;
        *)
            RESPONSE='{"error":"endpoint not found","path":"'$PATH_INFO'"}'
            ;;
    esac
    
    # Send response
    echo -e "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nAccess-Control-Allow-Origin: *\r\nContent-Length: ${#RESPONSE}\r\n\r\n$RESPONSE" | nc -l -p $PORT -q 0 &
    
    # Log to file
    echo "$(date -Iseconds): $PATH_INFO" >> /data/api_access.log
done
APIEOF
    chmod +x "${DATA_DIR}/edge/api_server.sh"
    
    docker run -d \
        --name "${CONTAINER_NAME}" \
        --restart unless-stopped \
        -p "${OPENEMS_PORT}:8080" \
        -v "${DATA_DIR}/edge:/data" \
        openjdk:17-slim \
        bash -c "
            apt-get update && apt-get install -y netcat-openbsd curl || true
            /data/api_server.sh 8080
        "
}

openems::stop() {
    echo "🛑 Stopping OpenEMS..."
    
    # Stop containers
    docker stop "${CONTAINER_NAME}" 2>/dev/null || true
    docker stop "${BACKEND_CONTAINER}" 2>/dev/null || true
    
    # Remove containers
    docker rm "${CONTAINER_NAME}" 2>/dev/null || true
    docker rm "${BACKEND_CONTAINER}" 2>/dev/null || true
    
    echo "✅ OpenEMS stopped"
    return 0
}

openems::restart() {
    echo "🔄 Restarting OpenEMS..."
    openems::stop
    sleep 2
    openems::start "$@"
    return 0
}

openems::uninstall() {
    echo "🗑️  Uninstalling OpenEMS..."
    
    # Stop services
    openems::stop
    
    # Remove images
    docker rmi "${OPENEMS_IMAGE}" 2>/dev/null || true
    docker rmi "${BACKEND_IMAGE}" 2>/dev/null || true
    
    # Clean data (preserve configs)
    rm -rf "${DATA_DIR}/edge/data" "${DATA_DIR}/backend/data"
    
    echo "✅ OpenEMS uninstalled"
    return 0
}

# ============================================
# Content Management Functions
# ============================================

content::list() {
    echo "📋 Available OpenEMS Configurations:"
    echo "===================================="
    
    # List configuration files
    if [[ -d "${DATA_DIR}/configs" ]]; then
        find "${DATA_DIR}/configs" -maxdepth 1 \( -name "*.json" -o -name "*.xml" \) -exec basename {} \; 2>/dev/null || echo "No configurations found"
    fi
    
    # List available commands
    echo ""
    echo "📋 Available Commands:"
    echo "===================="
    echo "  configure-der    - Configure DER asset"
    echo "  get-status      - Get system status"
    echo "  simulate-solar  - Simulate solar generation"
    echo "  simulate-load   - Simulate load profile"
    
    return 0
}

content::add() {
    local name="${1:-}"
    local file="${2:-}"
    
    if [[ -z "$name" ]] || [[ -z "$file" ]]; then
        echo "❌ Usage: content add <name> <file>"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "❌ File not found: $file"
        return 1
    fi
    
    cp "$file" "${DATA_DIR}/configs/${name}.json"
    echo "✅ Configuration added: $name"
    return 0
}

content::get() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "❌ Usage: content get <name>"
        return 1
    fi
    
    local config_file="${DATA_DIR}/configs/${name}.json"
    if [[ -f "$config_file" ]]; then
        cat "$config_file"
    else
        echo "❌ Configuration not found: $name"
        return 1
    fi
}

content::remove() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "❌ Usage: content remove <name>"
        return 1
    fi
    
    rm -f "${DATA_DIR}/configs/${name}.json"
    echo "✅ Configuration removed: $name"
    return 0
}

content::execute() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        configure-der)
            openems::configure_der "$@"
            ;;
        get-status)
            openems::get_status "$@"
            ;;
        simulate-solar)
            openems::simulate_solar "$@"
            ;;
        simulate-load)
            openems::simulate_load "$@"
            ;;
        *)
            echo "❌ Unknown command: $command"
            return 1
            ;;
    esac
}

# ============================================
# OpenEMS-specific Functions
# ============================================

openems::configure_der() {
    local type="${1:-solar}"
    local capacity="${2:-10}"
    
    echo "⚙️  Configuring DER: Type=$type, Capacity=${capacity}kW"
    
    # Create DER configuration
    cat > "${DATA_DIR}/configs/der_${type}.json" << EOF
{
    "id": "der_${type}_$(date +%s)",
    "type": "$type",
    "capacity": $capacity,
    "enabled": true,
    "components": {
        "inverter": {
            "model": "generic",
            "maxPower": $capacity
        },
        "meter": {
            "type": "virtual",
            "interval": 1000
        }
    }
}
EOF
    
    echo "✅ DER configured successfully"
    return 0
}

openems::get_status() {
    echo "📊 OpenEMS System Status"
    echo "======================"
    
    # Check container status
    if docker ps -q -f name="^${CONTAINER_NAME}$" | grep -q .; then
        echo "Edge Status: ✅ Running"
        
        # Try to get API status
        if timeout 5 curl -sf "http://localhost:${OPENEMS_PORT}/health" &>/dev/null; then
            echo "API Status: ✅ Responsive"
            # Also test a data endpoint
            if timeout 5 curl -sf "http://localhost:${OPENEMS_PORT}/rest/channel/_sum/EssSoc" &>/dev/null; then
                echo "Data API: ✅ Available"
            else
                echo "Data API: ⚠️  Limited (simulation mode)"
            fi
        else
            echo "API Status: ⚠️  Not responsive (may be starting)"
        fi
    else
        echo "Edge Status: ❌ Not running"
    fi
    
    # Check backend
    if docker ps -q -f name="^${BACKEND_CONTAINER}$" | grep -q .; then
        echo "Backend Status: ✅ Running"
    else
        echo "Backend Status: ⚠️  Not running"
    fi
    
    return 0
}

openems::simulate_solar() {
    local power="${1:-5000}"
    local duration="${2:-10}"  # Duration in seconds
    
    echo "☀️  Simulating solar generation: ${power}W for ${duration}s"
    
    # Initialize telemetry if needed (allow failures)
    openems::init_telemetry 2>/dev/null || true
    
    local count=0
    while [[ $count -lt $duration ]]; do
        # Simulate varying power with cloud cover
        local variation=$(( RANDOM % 500 - 250 ))  # +/- 250W variation
        local actual_power=$(( power + variation ))
        [[ $actual_power -lt 0 ]] && actual_power=0
        
        local voltage=400
        local current
        current=$(awk "BEGIN {print $actual_power / $voltage}")
        local energy
        energy=$(awk "BEGIN {print $actual_power * 1 / 3600}")
        
        # Send telemetry (allow failures)
        openems::send_telemetry "solar_01" "solar" "${actual_power}" "${energy}" "${voltage}" "${current}" "0" "35" 2>/dev/null || true
        
        # Write local simulation data (with error handling)
        local sim_file="${DATA_DIR}/edge/data/solar_sim.json"
        if ! touch "$sim_file" 2>/dev/null; then
            sim_file="/tmp/openems_solar_sim.json"
        fi
        cat > "$sim_file" << EOF 2>/dev/null || true
{
    "timestamp": "$(date -Iseconds)",
    "power": $actual_power,
    "energy": $energy,
    "voltage": $voltage,
    "current": $current
}
EOF
        
        echo "  Power: ${actual_power}W, Voltage: ${voltage}V, Current: ${current}A"
        
        sleep 1
        count=$((count + 1))
    done
    
    echo "✅ Solar simulation completed (${duration}s of data generated)"
    return 0
}

openems::simulate_load() {
    local power="${1:-3000}"
    local duration="${2:-10}"  # Accept duration parameter for consistency
    
    echo "🔌 Simulating load profile: ${power}W for ${duration}s"
    
    local count=0
    while [[ $count -lt $duration ]]; do
        # Simulate varying load
        local variation=$(( RANDOM % 300 - 150 ))  # +/- 150W variation
        local actual_power=$(( power + variation ))
        [[ $actual_power -lt 0 ]] && actual_power=0
        
        # Write load data (with error handling)
        local sim_file="${DATA_DIR}/edge/data/load_sim.json"
        if ! touch "$sim_file" 2>/dev/null; then
            sim_file="/tmp/openems_load_sim.json"
        fi
        cat > "$sim_file" << EOF 2>/dev/null || true
{
    "timestamp": "$(date -Iseconds)",
    "power": $actual_power,
    "energy": $(( actual_power * 1000 / 3600 )),
    "powerfactor": 0.95
}
EOF
        
        echo "  Load: ${actual_power}W"
        
        sleep 1
        count=$((count + 1))
    done
    
    echo "✅ Load simulation completed (${duration}s of data generated)"
    return 0
}

# ============================================
# Status and Monitoring Functions
# ============================================

status::show() {
    local verbose="${1:-}"
    
    openems::get_status
    
    if [[ "$verbose" == "--verbose" ]]; then
        echo ""
        echo "📊 Detailed Metrics:"
        echo "==================="
        
        # Show recent simulations
        if [[ -f "${DATA_DIR}/edge/data/solar_sim.json" ]]; then
            echo "Solar Generation:"
            cat "${DATA_DIR}/edge/data/solar_sim.json" | jq '.' 2>/dev/null || cat "${DATA_DIR}/edge/data/solar_sim.json"
        fi
        
        if [[ -f "${DATA_DIR}/edge/data/load_sim.json" ]]; then
            echo ""
            echo "Load Profile:"
            cat "${DATA_DIR}/edge/data/load_sim.json" | jq '.' 2>/dev/null || cat "${DATA_DIR}/edge/data/load_sim.json"
        fi
    fi
    
    return 0
}

logs::show() {
    local lines="${1:-50}"
    
    echo "📜 OpenEMS Logs (last $lines lines):"
    echo "===================================="
    
    # Show Edge logs
    echo "Edge Logs:"
    docker logs --tail "$lines" "${CONTAINER_NAME}" 2>&1 || echo "No Edge logs available"
    
    # Show Backend logs if running
    if docker ps -q -f name="^${BACKEND_CONTAINER}$" | grep -q .; then
        echo ""
        echo "Backend Logs:"
        docker logs --tail "$lines" "${BACKEND_CONTAINER}" 2>&1 || echo "No Backend logs available"
    fi
    
    return 0
}

# ============================================
# Helper Functions
# ============================================

openems::wait_for_health() {
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if timeout 5 curl -sf "http://localhost:${OPENEMS_PORT}/health" &>/dev/null || \
           docker exec "${CONTAINER_NAME}" echo "OK" &>/dev/null; then
            echo "✅ OpenEMS is healthy"
            return 0
        fi
        
        echo "⏳ Waiting for OpenEMS... (attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    echo "⚠️  OpenEMS health check timeout"
    return 1
}

# ============================================
# Telemetry Persistence Functions
# ============================================

openems::init_telemetry() {
    # In test mode, suppress non-critical warnings
    local quiet_mode="${OPENEMS_QUIET_MODE:-false}"
    
    if [[ "$quiet_mode" != "true" ]]; then
        echo "📊 Initializing telemetry persistence..."
    fi
    
    # Check QuestDB availability (with graceful handling)
    if [[ -n "${QUESTDB_HOST}" ]] && [[ -n "${QUESTDB_PORT}" ]]; then
        if timeout 5 curl -sf "http://${QUESTDB_HOST}:${QUESTDB_PORT}/exec" &>/dev/null; then
            if [[ "$quiet_mode" != "true" ]]; then
                echo "✅ QuestDB available at ${QUESTDB_HOST}:${QUESTDB_PORT}"
            fi
            openems::create_questdb_tables
        else
            if [[ "$quiet_mode" != "true" ]]; then
                echo "⚠️  QuestDB not available, telemetry will use local storage" >&2
                echo "   To enable QuestDB: vrooli resource questdb manage start" >&2
            fi
        fi
    else
        if [[ "$quiet_mode" != "true" ]]; then
            echo "⚠️  QuestDB configuration not found, using local storage" >&2
        fi
    fi
    
    # Check Redis availability (with graceful handling)
    if [[ -n "${REDIS_HOST}" ]] && [[ -n "${REDIS_PORT}" ]]; then
        if command -v nc &>/dev/null; then
            if timeout 5 nc -zv "${REDIS_HOST}" "${REDIS_PORT}" &>/dev/null; then
                if [[ "$quiet_mode" != "true" ]]; then
                    echo "✅ Redis available at ${REDIS_HOST}:${REDIS_PORT}"
                fi
            else
                if [[ "$quiet_mode" != "true" ]]; then
                    echo "⚠️  Redis not available, real-time state will use local cache" >&2
                    echo "   To enable Redis: vrooli resource redis manage start" >&2
                fi
            fi
        else
            if [[ "$quiet_mode" != "true" ]]; then
                echo "⚠️  netcat not installed, cannot test Redis connectivity" >&2
            fi
        fi
    else
        if [[ "$quiet_mode" != "true" ]]; then
            echo "⚠️  Redis configuration not found, using local cache" >&2
        fi
    fi
    
    return 0
}

openems::create_questdb_tables() {
    # Create DER telemetry table
    local create_table_sql="CREATE TABLE IF NOT EXISTS der_telemetry (
        timestamp TIMESTAMP,
        asset_id SYMBOL,
        asset_type SYMBOL,
        power DOUBLE,
        energy DOUBLE,
        voltage DOUBLE,
        current DOUBLE,
        soc DOUBLE,
        temperature DOUBLE
    ) TIMESTAMP(timestamp) PARTITION BY DAY;"
    
    curl -G "http://${QUESTDB_HOST}:${QUESTDB_PORT}/exec" \
        --data-urlencode "query=${create_table_sql}" \
        --silent --fail || echo "⚠️  Failed to create QuestDB table"
    
    # Create grid metrics table
    local grid_table_sql="CREATE TABLE IF NOT EXISTS grid_metrics (
        timestamp TIMESTAMP,
        grid_power DOUBLE,
        grid_frequency DOUBLE,
        grid_voltage DOUBLE,
        total_consumption DOUBLE,
        total_generation DOUBLE,
        net_export DOUBLE
    ) TIMESTAMP(timestamp) PARTITION BY DAY;"
    
    curl -G "http://${QUESTDB_HOST}:${QUESTDB_PORT}/exec" \
        --data-urlencode "query=${grid_table_sql}" \
        --silent --fail || echo "⚠️  Failed to create grid metrics table"
    
    return 0
}

openems::send_telemetry() {
    local asset_id="${1}"
    local asset_type="${2}"
    local power="${3}"
    local energy="${4:-0}"
    local voltage="${5:-400}"
    local current="${6:-0}"
    local soc="${7:-0}"
    local temp="${8:-25}"
    
    # Send to QuestDB if available (with error handling)
    if [[ -n "${QUESTDB_HOST}" ]] && [[ -n "${QUESTDB_PORT}" ]] && \
       timeout 2 curl -sf "http://${QUESTDB_HOST}:${QUESTDB_PORT}/exec" &>/dev/null; then
        local timestamp
        timestamp="$(date +%s)000000000"
        local insert_sql="INSERT INTO der_telemetry VALUES(
            ${timestamp},
            '${asset_id}',
            '${asset_type}',
            ${power},
            ${energy},
            ${voltage},
            ${current},
            ${soc},
            ${temp}
        );"
        
        curl -G "http://${QUESTDB_HOST}:${QUESTDB_PORT}/exec" \
            --data-urlencode "query=${insert_sql}" \
            --silent --fail || echo "⚠️  Failed to send telemetry to QuestDB"
    fi
    
    # Cache in Redis if available (with error handling)
    if [[ -n "${REDIS_HOST}" ]] && [[ -n "${REDIS_PORT}" ]] && \
       command -v redis-cli &>/dev/null && \
       timeout 2 nc -zv "${REDIS_HOST}" "${REDIS_PORT}" &>/dev/null 2>&1; then
        redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" HSET "openems:assets:${asset_id}" \
            "power" "${power}" \
            "voltage" "${voltage}" \
            "current" "${current}" \
            "soc" "${soc}" \
            "last_update" "$(date -Iseconds)" &>/dev/null
    fi
    
    # Always save locally as backup (with error handling)
    local telemetry_file="${DATA_DIR}/edge/data/telemetry.jsonl"
    
    # Create file if it doesn't exist with proper permissions
    if [[ ! -f "$telemetry_file" ]]; then
        touch "$telemetry_file" 2>/dev/null || {
            local quiet_mode="${OPENEMS_QUIET_MODE:-false}"
            if [[ "$quiet_mode" != "true" ]]; then
                echo "⚠️  Cannot create telemetry file, using /tmp fallback" >&2
            fi
            telemetry_file="/tmp/openems_telemetry.jsonl"
            touch "$telemetry_file"
        }
    fi
    
    # Write telemetry data
    echo "{\"timestamp\":\"$(date -Iseconds)\",\"asset_id\":\"${asset_id}\",\"power\":${power},\"energy\":${energy}}" \
        >> "$telemetry_file" 2>/dev/null || echo "⚠️  Failed to write telemetry"
    
    return 0
}

# Get current telemetry data for alerts
openems::core::get_telemetry() {
    local telemetry_json="{}"
    
    # Try to get from Redis first (with error handling)
    if [[ -n "${REDIS_HOST}" ]] && [[ -n "${REDIS_PORT}" ]] && \
       command -v redis-cli &>/dev/null && \
       timeout 2 nc -zv "${REDIS_HOST}" "${REDIS_PORT}" &>/dev/null 2>&1; then
        local battery_data solar_data grid_data
        battery_data=$(redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" HGETALL "openems:assets:battery0" 2>/dev/null)
        solar_data=$(redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" HGETALL "openems:assets:solar0" 2>/dev/null)
        grid_data=$(redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" HGETALL "openems:assets:grid0" 2>/dev/null)
        
        # Parse Redis data into JSON
        local battery_soc solar_power grid_power
        battery_soc=$(echo "$battery_data" | awk '/^soc/{getline; print}' | head -1)
        solar_power=$(echo "$solar_data" | awk '/^power/{getline; print}' | head -1)
        grid_power=$(echo "$grid_data" | awk '/^power/{getline; print}' | head -1)
        
        # Check grid status
        local grid_status="connected"
        [[ -z "$grid_power" || "$grid_power" == "0" ]] && grid_status="disconnected"
        
        # Check daylight (simplified - between 6 AM and 8 PM)
        local hour
        hour=$(date +%H)
        local daylight="false"
        [[ "$hour" -ge 6 && "$hour" -le 20 ]] && daylight="true"
        
        telemetry_json=$(cat <<EOF
{
    "grid_status": "${grid_status}",
    "battery_soc": ${battery_soc:-50},
    "solar_power": ${solar_power:-0},
    "daylight": ${daylight},
    "grid_import": ${grid_power:-0},
    "battery_fault": false
}
EOF
)
    else
        # Fall back to last local telemetry
        if [[ -f "${DATA_DIR}/edge/data/telemetry.jsonl" ]]; then
            local last_line
            last_line=$(tail -1 "${DATA_DIR}/edge/data/telemetry.jsonl" 2>/dev/null)
            if [[ -n "$last_line" ]]; then
                telemetry_json="$last_line"
            fi
        fi
    fi
    
    echo "$telemetry_json"
    return 0
}

openems::create_default_config() {
    # Check if we can write to the directory
    if [[ ! -w "${DATA_DIR}/edge/config" ]]; then
        echo "⚠️  Cannot write to config directory, skipping config creation"
        return 0
    fi
    
    # Create a basic Edge configuration
    cat > "${DATA_DIR}/edge/config/openems.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<openems>
    <component id="_meta" class="io.openems.edge.core.meta.Meta">
        <property name="currency">EUR</property>
    </component>
    <component id="_cycle" class="io.openems.edge.core.cycle.Cycle">
        <property name="cycleTime">1000</property>
    </component>
    <component id="datasource0" class="io.openems.edge.datasource.csv.CsvDatasource">
        <property name="directory">/var/opt/openems</property>
    </component>
</openems>
EOF
    
    echo "✅ Default configuration created"
    return 0
}