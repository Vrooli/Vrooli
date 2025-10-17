#!/usr/bin/env bash
# Open MCT Core Functions

set -euo pipefail

# Load configuration
CORE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${CORE_DIR}/../config/defaults.sh"

# Show resource information
show_info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${CORE_DIR}/../config/runtime.json"
    else
        echo "Open MCT Resource Information"
        echo "============================="
        echo "Version: 2.0.0"
        echo "Port: ${OPENMCT_PORT}"
        echo "WebSocket Port: ${OPENMCT_WS_PORT}"
        echo "Data Directory: ${OPENMCT_DATA_DIR}"
        echo "Container: ${OPENMCT_CONTAINER_NAME}"
        echo ""
        echo "Configuration:"
        echo "  Max Streams: ${OPENMCT_MAX_STREAMS}"
        echo "  History Days: ${OPENMCT_HISTORY_DAYS}"
        echo "  Demo Mode: ${OPENMCT_ENABLE_DEMO}"
        echo "  Auth Enabled: ${OPENMCT_AUTH_ENABLED}"
    fi
}

# Handle lifecycle management
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            install_openmct "$@"
            ;;
        uninstall)
            uninstall_openmct "$@"
            ;;
        start)
            start_openmct "$@"
            ;;
        stop)
            stop_openmct "$@"
            ;;
        restart)
            restart_openmct "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand"
            echo "Valid subcommands: install, uninstall, start, stop, restart"
            exit 1
            ;;
    esac
}

# Install Open MCT
install_openmct() {
    echo "Installing Open MCT..."
    
    # Create required directories
    mkdir -p "$OPENMCT_DATA_DIR" "$OPENMCT_CONFIG_DIR" "$OPENMCT_PLUGINS_DIR"
    
    # Build Docker image
    echo "Building Open MCT Docker image..."
    
    # Create Dockerfile if it doesn't exist
    if [[ ! -f "${CORE_DIR}/../docker/Dockerfile" ]]; then
        create_dockerfile
    fi
    
    # Build the image
    docker build -t "$OPENMCT_IMAGE" "${CORE_DIR}/../docker" || {
        echo "Error: Failed to build Docker image"
        exit 1
    }
    
    # Initialize SQLite database
    echo "Initializing telemetry database..."
    sqlite3 "${OPENMCT_DATA_DIR}/telemetry.db" << EOF
CREATE TABLE IF NOT EXISTS telemetry (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    stream TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    value REAL,
    data TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_telemetry_stream_timestamp ON telemetry(stream, timestamp);
CREATE INDEX IF NOT EXISTS idx_telemetry_timestamp ON telemetry(timestamp);
EOF
    
    echo "Open MCT installed successfully"
}

# Create Dockerfile for Open MCT
create_dockerfile() {
    mkdir -p "${CORE_DIR}/../docker"
    
    cat > "${CORE_DIR}/../docker/Dockerfile" << 'EOF'
FROM node:18-alpine

# Install dependencies
RUN apk add --no-cache git python3 make g++ sqlite

# Create app directory
WORKDIR /app

# Clone Open MCT
RUN git clone https://github.com/nasa/openmct.git . && \
    git checkout v2.2.2

# Install npm dependencies
RUN npm install && npm install sqlite3 ws express

# Copy custom configuration
COPY server.js /app/server.js
COPY telemetry-server.js /app/telemetry-server.js
COPY index.html /app/index.html

# Create data directory
RUN mkdir -p /data

# Expose ports
EXPOSE 8099 8198

# Start server
CMD ["node", "server.js"]
EOF

    # Create server.js
    cat > "${CORE_DIR}/../docker/server.js" << 'EOF'
const express = require('express');
const path = require('path');
const sqlite3 = require('sqlite3').verbose();
const WebSocket = require('ws');
const http = require('http');

const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ port: process.env.OPENMCT_WS_PORT || 8198 });

const PORT = process.env.OPENMCT_PORT || 8099;
const DATA_DIR = process.env.OPENMCT_DATA_DIR || '/data';

// Initialize database
const db = new sqlite3.Database(path.join(DATA_DIR, 'telemetry.db'));

// Middleware
app.use(express.json());

// Serve Open MCT dist files
app.use('/openmct.js', express.static(path.join(__dirname, 'dist/openmct.js')));
app.use('/openmct.js.map', express.static(path.join(__dirname, 'dist/openmct.js.map')));
app.use('/espresso-theme.css', express.static(path.join(__dirname, 'dist/espresso-theme.css')));
app.use('/snow-theme.css', express.static(path.join(__dirname, 'dist/snow-theme.css')));

// Serve custom index
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Serve static files from dist
app.use(express.static(path.join(__dirname, 'dist')));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', timestamp: Date.now() });
});

// Telemetry API endpoints
app.get('/api/telemetry/:stream', (req, res) => {
    const { stream } = req.params;
    res.json({
        stream,
        name: stream.replace(/_/g, ' '),
        type: 'telemetry.point',
        values: ['timestamp', 'value']
    });
});

app.post('/api/telemetry/:stream/data', (req, res) => {
    const { stream } = req.params;
    const { timestamp, value, data } = req.body;
    
    db.run(
        'INSERT INTO telemetry (stream, timestamp, value, data) VALUES (?, ?, ?, ?)',
        [stream, timestamp || Date.now(), value, JSON.stringify(data || {})],
        (err) => {
            if (err) {
                res.status(500).json({ error: err.message });
            } else {
                res.json({ success: true });
                
                // Broadcast to WebSocket clients
                wss.clients.forEach(client => {
                    if (client.readyState === WebSocket.OPEN) {
                        client.send(JSON.stringify({
                            stream,
                            timestamp: timestamp || Date.now(),
                            value,
                            data
                        }));
                    }
                });
            }
        }
    );
});

app.get('/api/telemetry/history', (req, res) => {
    const { stream, start, end } = req.query;
    
    let query = 'SELECT * FROM telemetry WHERE 1=1';
    const params = [];
    
    if (stream) {
        query += ' AND stream = ?';
        params.push(stream);
    }
    if (start) {
        query += ' AND timestamp >= ?';
        params.push(parseInt(start));
    }
    if (end) {
        query += ' AND timestamp <= ?';
        params.push(parseInt(end));
    }
    
    query += ' ORDER BY timestamp DESC LIMIT 1000';
    
    db.all(query, params, (err, rows) => {
        if (err) {
            res.status(500).json({ error: err.message });
        } else {
            res.json(rows);
        }
    });
});

// WebSocket connection handling
wss.on('connection', (ws) => {
    console.log('New WebSocket connection');
    
    ws.on('message', (message) => {
        try {
            const data = JSON.parse(message);
            
            // Store in database
            db.run(
                'INSERT INTO telemetry (stream, timestamp, value, data) VALUES (?, ?, ?, ?)',
                [data.stream, data.timestamp || Date.now(), data.value, JSON.stringify(data.data || {})]
            );
            
            // Broadcast to other clients
            wss.clients.forEach(client => {
                if (client !== ws && client.readyState === WebSocket.OPEN) {
                    client.send(message);
                }
            });
        } catch (error) {
            console.error('WebSocket message error:', error);
        }
    });
    
    ws.on('close', () => {
        console.log('WebSocket connection closed');
    });
});

// Start server
server.listen(PORT, '0.0.0.0', () => {
    console.log(`Open MCT server running on port ${PORT}`);
    console.log(`WebSocket server running on port ${process.env.OPENMCT_WS_PORT || 8198}`);
    
    // Start demo telemetry if enabled
    if (process.env.OPENMCT_ENABLE_DEMO === 'true') {
        require('./telemetry-server').startDemoTelemetry(wss);
    }
});
EOF

    # Create telemetry-server.js for demo data
    cat > "${CORE_DIR}/../docker/telemetry-server.js" << 'EOF'
function startDemoTelemetry(wss) {
    console.log('Starting demo telemetry streams...');
    
    // Satellite telemetry
    setInterval(() => {
        const satelliteData = {
            stream: 'satellite_position',
            timestamp: Date.now(),
            value: Math.sin(Date.now() / 10000) * 90,  // Latitude
            data: {
                latitude: Math.sin(Date.now() / 10000) * 90,
                longitude: Math.cos(Date.now() / 10000) * 180,
                altitude: 400 + Math.random() * 10,
                velocity: 7.8 + Math.random() * 0.1
            }
        };
        
        broadcast(wss, satelliteData);
    }, 1000);
    
    // Sensor network
    setInterval(() => {
        const sensorData = {
            stream: 'sensor_network',
            timestamp: Date.now(),
            value: 20 + Math.random() * 10,  // Temperature
            data: {
                temperature: 20 + Math.random() * 10,
                humidity: 40 + Math.random() * 20,
                pressure: 1013 + Math.random() * 10
            }
        };
        
        broadcast(wss, sensorData);
    }, 2000);
    
    // System metrics
    setInterval(() => {
        const metricsData = {
            stream: 'system_metrics',
            timestamp: Date.now(),
            value: Math.random() * 100,  // CPU usage
            data: {
                cpu: Math.random() * 100,
                memory: Math.random() * 100,
                disk: Math.random() * 100,
                network: Math.random() * 1000
            }
        };
        
        broadcast(wss, metricsData);
    }, 3000);
}

function broadcast(wss, data) {
    wss.clients.forEach(client => {
        if (client.readyState === 1) {  // WebSocket.OPEN
            client.send(JSON.stringify(data));
        }
    });
}

module.exports = { startDemoTelemetry };
EOF
}

# Uninstall Open MCT
uninstall_openmct() {
    echo "Uninstalling Open MCT..."
    
    # Stop container if running
    stop_openmct
    
    # Remove container
    docker rm -f "$OPENMCT_CONTAINER_NAME" 2>/dev/null || true
    
    # Remove image
    docker rmi "$OPENMCT_IMAGE" 2>/dev/null || true
    
    # Optionally remove data
    read -p "Remove all Open MCT data? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$OPENMCT_DATA_DIR" "$OPENMCT_CONFIG_DIR" "$OPENMCT_PLUGINS_DIR"
        echo "Data removed"
    fi
    
    echo "Open MCT uninstalled"
}

# Start Open MCT
start_openmct() {
    local wait_flag=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait)
                wait_flag=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo "Starting Open MCT..."
    
    # Check if already running
    if docker ps --format "table {{.Names}}" | grep -q "^${OPENMCT_CONTAINER_NAME}$"; then
        echo "Open MCT is already running"
        return 0
    fi
    
    # Start container
    docker run -d \
        --name "$OPENMCT_CONTAINER_NAME" \
        --restart "$OPENMCT_RESTART_POLICY" \
        -p "${OPENMCT_PORT}:8099" \
        -p "${OPENMCT_WS_PORT}:${OPENMCT_WS_PORT}" \
        -v "${OPENMCT_DATA_DIR}:/data" \
        -v "${OPENMCT_CONFIG_DIR}:/config" \
        -v "${OPENMCT_PLUGINS_DIR}:/plugins" \
        -e OPENMCT_PORT="$OPENMCT_PORT" \
        -e OPENMCT_WS_PORT="$OPENMCT_WS_PORT" \
        -e OPENMCT_DATA_DIR="/data" \
        -e OPENMCT_ENABLE_DEMO="$OPENMCT_ENABLE_DEMO" \
        -e OPENMCT_MAX_STREAMS="$OPENMCT_MAX_STREAMS" \
        -e OPENMCT_AUTH_ENABLED="$OPENMCT_AUTH_ENABLED" \
        --memory="$OPENMCT_MEMORY_LIMIT" \
        --cpus="$OPENMCT_CPU_LIMIT" \
        "$OPENMCT_IMAGE" || {
        echo "Error: Failed to start Open MCT container"
        exit 1
    }
    
    if [[ "$wait_flag" == true ]]; then
        echo "Waiting for Open MCT to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if timeout 5 curl -sf "http://localhost:${OPENMCT_PORT}/health" > /dev/null 2>&1; then
                echo "Open MCT is ready!"
                echo "Access dashboard at: http://localhost:${OPENMCT_PORT}"
                return 0
            fi
            
            attempt=$((attempt + 1))
            sleep 1
        done
        
        echo "Error: Open MCT failed to start within 30 seconds"
        docker logs "$OPENMCT_CONTAINER_NAME"
        exit 1
    else
        echo "Open MCT started. Access at: http://localhost:${OPENMCT_PORT}"
    fi
}

# Stop Open MCT
stop_openmct() {
    echo "Stopping Open MCT..."
    
    if docker ps --format "table {{.Names}}" | grep -q "^${OPENMCT_CONTAINER_NAME}$"; then
        docker stop "$OPENMCT_CONTAINER_NAME" || {
            echo "Error: Failed to stop Open MCT container"
            exit 1
        }
        echo "Open MCT stopped"
    else
        echo "Open MCT is not running"
    fi
}

# Restart Open MCT
restart_openmct() {
    echo "Restarting Open MCT..."
    stop_openmct
    sleep 2
    start_openmct "$@"
}

# Show status
show_status() {
    local format="${1:-text}"
    
    if docker ps --format "table {{.Names}}" | grep -q "^${OPENMCT_CONTAINER_NAME}$"; then
        local health_status="unknown"
        
        if timeout 5 curl -sf "http://localhost:${OPENMCT_PORT}/health" > /dev/null 2>&1; then
            health_status="healthy"
        else
            health_status="unhealthy"
        fi
        
        if [[ "$format" == "--json" ]]; then
            echo "{\"status\": \"running\", \"health\": \"$health_status\", \"port\": $OPENMCT_PORT, \"container\": \"$OPENMCT_CONTAINER_NAME\"}"
        else
            echo "Open MCT Status: Running"
            echo "Health: $health_status"
            echo "Port: $OPENMCT_PORT"
            echo "WebSocket Port: $OPENMCT_WS_PORT"
            echo "Container: $OPENMCT_CONTAINER_NAME"
            echo "Dashboard: http://localhost:${OPENMCT_PORT}"
        fi
    else
        if [[ "$format" == "--json" ]]; then
            echo "{\"status\": \"stopped\"}"
        else
            echo "Open MCT Status: Stopped"
        fi
    fi
}

# Show logs
show_logs() {
    local tail_lines="${1:-100}"
    
    if [[ "$1" == "--tail" ]]; then
        tail_lines="${2:-100}"
    fi
    
    if docker ps -a --format "table {{.Names}}" | grep -q "^${OPENMCT_CONTAINER_NAME}$"; then
        docker logs --tail "$tail_lines" "$OPENMCT_CONTAINER_NAME"
    else
        echo "No Open MCT container found"
    fi
}

# Handle content management
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        add)
            add_telemetry_source "$@"
            ;;
        list)
            list_telemetry_sources "$@"
            ;;
        get)
            get_telemetry_data "$@"
            ;;
        remove)
            remove_telemetry_source "$@"
            ;;
        execute)
            execute_telemetry_command "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand: $subcommand"
            echo "Valid subcommands: add, list, get, remove, execute"
            exit 1
            ;;
    esac
}

# Add telemetry source
add_telemetry_source() {
    local source_type="${1:-}"
    local source_config="${2:-}"
    
    if [[ -z "$source_type" ]]; then
        echo "Error: Source type required (websocket, mqtt, traccar)"
        exit 1
    fi
    
    echo "Adding telemetry source: $source_type"
    
    # Store configuration
    local config_file="${OPENMCT_CONFIG_DIR}/sources_$(date +%s).json"
    echo "{\"type\": \"$source_type\", \"config\": \"$source_config\"}" > "$config_file"
    
    echo "Telemetry source added: $(basename "$config_file")"
}

# List telemetry sources
list_telemetry_sources() {
    echo "Configured telemetry sources:"
    echo "============================="
    
    if [[ -d "$OPENMCT_CONFIG_DIR" ]]; then
        shopt -s nullglob
        for config in "${OPENMCT_CONFIG_DIR}"/sources_*.json; do
            if [[ -f "$config" ]]; then
                echo "- $(basename "$config"): $(jq -r .type "$config" 2>/dev/null || echo "unknown")"
            fi
        done
        shopt -u nullglob
    fi
    
    if [[ "$OPENMCT_ENABLE_DEMO" == "true" ]]; then
        echo ""
        echo "Demo streams (active):"
        echo "- satellite_position"
        echo "- sensor_network"
        echo "- system_metrics"
    fi
}

# Get telemetry data
get_telemetry_data() {
    local stream="${1:-}"
    
    if [[ -z "$stream" ]]; then
        echo "Error: Stream name required"
        exit 1
    fi
    
    echo "Fetching telemetry data for stream: $stream"
    
    # Query from SQLite database
    sqlite3 -header -column "${OPENMCT_DATA_DIR}/telemetry.db" \
        "SELECT datetime(timestamp/1000, 'unixepoch') as time, value, data 
         FROM telemetry 
         WHERE stream = '$stream' 
         ORDER BY timestamp DESC 
         LIMIT 10;"
}

# Remove telemetry source
remove_telemetry_source() {
    local source_id="${1:-}"
    
    if [[ -z "$source_id" ]]; then
        echo "Error: Source ID required"
        exit 1
    fi
    
    local config_file="${OPENMCT_CONFIG_DIR}/${source_id}"
    
    if [[ -f "$config_file" ]]; then
        rm "$config_file"
        echo "Telemetry source removed: $source_id"
    else
        echo "Error: Source not found: $source_id"
        exit 1
    fi
}

# Execute telemetry command
execute_telemetry_command() {
    local command="${1:-}"
    
    case "$command" in
        clear-history)
            echo "Clearing telemetry history..."
            sqlite3 "${OPENMCT_DATA_DIR}/telemetry.db" "DELETE FROM telemetry;"
            echo "History cleared"
            ;;
        export)
            local output="${2:-telemetry_export.csv}"
            echo "Exporting telemetry data to: $output"
            sqlite3 -header -csv "${OPENMCT_DATA_DIR}/telemetry.db" \
                "SELECT * FROM telemetry ORDER BY timestamp;" > "$output"
            echo "Export complete: $output"
            ;;
        import)
            local input="${2:-}"
            if [[ -z "$input" ]]; then
                echo "Error: Input file required"
                exit 1
            fi
            echo "Importing telemetry data from: $input"
            sqlite3 "${OPENMCT_DATA_DIR}/telemetry.db" ".import $input telemetry"
            echo "Import complete"
            ;;
        *)
            echo "Error: Unknown command: $command"
            echo "Valid commands: clear-history, export, import"
            exit 1
            ;;
    esac
}