#!/usr/bin/env bash

# AI Model Orchestra Controller Setup Script
# This script initializes the AI Model Orchestra Controller scenario

set -euo pipefail

# Script metadata
readonly SCRIPT_NAME="AI Model Orchestra Controller Setup"
readonly SCRIPT_VERSION="1.0.0"
readonly SCENARIO_NAME="ai-model-orchestra-controller"

# Paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../../.." && builtin pwd)}"
readonly SCRIPT_DIR="${APP_ROOT}/scenarios/ai-model-orchestra-controller/initialization/scripts"
readonly SCENARIO_DIR="${APP_ROOT}/scenarios/ai-model-orchestra-controller"
readonly CONFIG_DIR="$SCENARIO_DIR/initialization/configuration"
readonly WORKFLOWS_DIR="$SCENARIO_DIR/initialization/workflows"
readonly UI_DIR="$SCENARIO_DIR/ui"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_header() {
    echo
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo
}

# Error handling
error_exit() {
    log_error "$1"
    exit 1
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check system requirements
check_requirements() {
    log_header "Checking System Requirements"
    
    local requirements_met=true
    
    # Check Node.js
    if command_exists node; then
        local node_version=$(node --version | sed 's/v//')
        local node_major=$(echo "$node_version" | cut -d. -f1)
        if [ "$node_major" -ge 18 ]; then
            log_success "Node.js $node_version (✓ >= 18.0.0)"
        else
            log_error "Node.js $node_version (✗ < 18.0.0 required)"
            requirements_met=false
        fi
    else
        log_error "Node.js not found (required >= 18.0.0)"
        requirements_met=false
    fi
    
    # Check npm
    if command_exists npm; then
        local npm_version=$(npm --version)
        log_success "npm $npm_version"
    else
        log_error "npm not found"
        requirements_met=false
    fi
    
    # Check for required services
    local required_services=("postgres" "redis" "ollama")
    for service in "${required_services[@]}"; do
        if command_exists "$service"; then
            log_success "$service command available"
        else
            log_warn "$service command not found (service may need to be installed/configured)"
        fi
    done
    
    # Check for Docker
    if command_exists docker; then
        log_success "Docker available"
    else
        log_warn "Docker not found (needed for container monitoring)"
    fi
    
    if [ "$requirements_met" = false ]; then
        error_exit "System requirements not met. Please install missing dependencies."
    fi
}

# Install Node.js dependencies
install_dependencies() {
    log_header "Installing Node.js Dependencies"
    
    # Create package.json if it doesn't exist
    local package_json="$SCENARIO_DIR/package.json"
    if [ ! -f "$package_json" ]; then
        log_info "Creating package.json..."
        cat > "$package_json" << 'EOF'
{
  "name": "ai-model-orchestra-controller",
  "version": "1.0.0",
  "description": "Intelligent AI model routing and resource management system",
  "main": "initialization/configuration/api-server.js",
  "scripts": {
    "start": "node initialization/configuration/api-server.js",
    "dev": "nodemon initialization/configuration/api-server.js",
    "test": "echo \"No tests specified\" && exit 0"
  },
  "dependencies": {
    "express": "^4.18.2",
    "cors": "^2.8.5",
    "helmet": "^6.1.5",
    "express-rate-limit": "^6.7.0",
    "body-parser": "^1.20.2",
    "pg": "^8.11.0",
    "redis": "^4.6.7",
    "dockerode": "^3.3.5",
    "node-cron": "^3.0.2"
  },
  "devDependencies": {
    "nodemon": "^2.0.22"
  },
  "keywords": [
    "ai",
    "orchestration",
    "load-balancing",
    "vrooli"
  ],
  "author": "Vrooli AI Infrastructure Team",
  "license": "MIT"
}
EOF
        log_success "package.json created"
    fi
    
    # Install dependencies
    log_info "Installing npm dependencies..."
    cd "$SCENARIO_DIR"
    npm install
    log_success "Dependencies installed"
}

# Create required configuration files
create_configurations() {
    log_header "Creating Configuration Files"
    
    # Generate resource-urls.json using the dynamic script
    log_info "Generating resource-urls.json using port registry..."
    
    if [ -f "$SCRIPT_DIR/create-resource-urls.sh" ]; then
        bash "$SCRIPT_DIR/create-resource-urls.sh"
        log_success "resource-urls.json generated dynamically"
    else
        log_warn "create-resource-urls.sh not found, creating fallback configuration"
        local resource_urls="$CONFIG_DIR/resource-urls.json"
        cat > "$resource_urls" << 'EOF'
{
  "version": "1.0.0",
  "generated_at": "fallback",
  "resources": {
    "ai": {
      "ollama": {
        "url": "http://${ORCHESTRATOR_HOST:-localhost}:${RESOURCE_PORTS_OLLAMA:-11434}",
        "api_base": "http://${ORCHESTRATOR_HOST:-localhost}:${RESOURCE_PORTS_OLLAMA:-11434}/api"
      }
    },
    "storage": {
      "postgres": {
        "url": "postgres://${POSTGRES_USER:-postgres}:${POSTGRES_PASSWORD:-postgres}@${ORCHESTRATOR_HOST:-localhost}:${RESOURCE_PORTS_POSTGRES:-5432}/${POSTGRES_DB:-orchestrator}?sslmode=disable",
        "maxConnections": 25
      },
      "redis": {
        "url": "redis://${ORCHESTRATOR_HOST:-localhost}:${RESOURCE_PORTS_REDIS:-6379}"
      }
    },
    "docker": {
      "api": {
        "socketPath": "/var/run/docker.sock"
      }
    }
  }
}
EOF
        log_success "Fallback resource-urls.json created"
    fi
}

# Initialize database schema
initialize_database() {
    log_header "Initializing Database Schema"
    
    # Create SQL schema file
    local schema_file="$CONFIG_DIR/database-schema.sql"
    if [ ! -f "$schema_file" ]; then
        log_info "Creating database schema..."
        cat > "$schema_file" << 'EOF'
-- AI Model Orchestra Controller Database Schema

-- Model metrics table
CREATE TABLE IF NOT EXISTS model_metrics (
  id SERIAL PRIMARY KEY,
  model_name VARCHAR(255) NOT NULL UNIQUE,
  request_count INTEGER DEFAULT 0,
  success_count INTEGER DEFAULT 0,
  error_count INTEGER DEFAULT 0,
  avg_response_time_ms FLOAT DEFAULT 0,
  current_load FLOAT DEFAULT 0,
  memory_usage_mb FLOAT DEFAULT 0,
  size_gb FLOAT DEFAULT 0,
  last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_health_check TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  healthy BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Orchestrator requests table
CREATE TABLE IF NOT EXISTS orchestrator_requests (
  id SERIAL PRIMARY KEY,
  request_id VARCHAR(255) UNIQUE NOT NULL,
  task_type VARCHAR(100) NOT NULL,
  selected_model VARCHAR(255) NOT NULL,
  fallback_used BOOLEAN DEFAULT FALSE,
  response_time_ms INTEGER,
  success BOOLEAN DEFAULT TRUE,
  error_message TEXT,
  resource_pressure FLOAT,
  cost_estimate FLOAT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- System resources table
CREATE TABLE IF NOT EXISTS system_resources (
  id SERIAL PRIMARY KEY,
  memory_available_gb FLOAT,
  memory_free_gb FLOAT,
  memory_total_gb FLOAT,
  memory_used_gb FLOAT,
  memory_usage_percent FLOAT,
  cpu_usage_percent FLOAT,
  cpu_load_1min FLOAT,
  swap_used_percent FLOAT,
  disk_usage_percent FLOAT,
  memory_pressure FLOAT,
  pressure_level VARCHAR(50),
  recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_model_metrics_name ON model_metrics(model_name);
CREATE INDEX IF NOT EXISTS idx_model_metrics_last_used ON model_metrics(last_used);
CREATE INDEX IF NOT EXISTS idx_orchestrator_requests_model ON orchestrator_requests(selected_model);
CREATE INDEX IF NOT EXISTS idx_orchestrator_requests_created ON orchestrator_requests(created_at);
CREATE INDEX IF NOT EXISTS idx_orchestrator_requests_success ON orchestrator_requests(success);
CREATE INDEX IF NOT EXISTS idx_system_resources_recorded ON system_resources(recorded_at);
CREATE INDEX IF NOT EXISTS idx_system_resources_pressure ON system_resources(pressure_level);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_model_metrics_updated_at 
  BEFORE UPDATE ON model_metrics 
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
EOF
        log_success "Database schema created"
    else
        log_info "Database schema already exists"
    fi
    
    # Try to create database tables if PostgreSQL is available
    if command_exists psql && command_exists pg_isready; then
        local pg_host="${ORCHESTRATOR_HOST:-localhost}"
        local pg_port="${RESOURCE_PORTS_POSTGRES:-5432}"
        local pg_user="${POSTGRES_USER:-postgres}"
        local pg_db="${POSTGRES_DB:-orchestrator}"
        
        if pg_isready -h "$pg_host" -p "$pg_port" >/dev/null 2>&1; then
            log_info "PostgreSQL is running, attempting to create tables..."
            # Note: This assumes proper credentials are configured via environment or .pgpass
            if PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h "$pg_host" -p "$pg_port" -U "$pg_user" -d "$pg_db" -f "$schema_file" >/dev/null 2>&1; then
                log_success "Database tables created successfully"
            else
                log_warn "Could not create database tables. Please run the schema manually:"
                log_warn "PGPASSWORD=\$POSTGRES_PASSWORD psql -h $pg_host -p $pg_port -U $pg_user -d $pg_db -f $schema_file"
            fi
        else
            log_warn "PostgreSQL not running. Database tables will be created when service starts."
        fi
    else
        log_warn "PostgreSQL client not available. Database schema will be applied when service starts."
    fi
}

# Configure Node-RED flows
configure_node_red() {
    log_header "Configuring Node-RED Flows"
    
    # Check if Node-RED is available
    if command_exists node-red; then
        log_info "Node-RED found, flows can be imported manually"
        log_info "Flow files available at:"
        log_info "  - $WORKFLOWS_DIR/orchestrator-main.json"
        log_info "  - $WORKFLOWS_DIR/resource-monitor.json"
    else
        log_warn "Node-RED not found. Flows will need to be imported manually when Node-RED is available."
    fi
    
    # Create flow import instructions
    local flow_instructions="$SCENARIO_DIR/FLOW_IMPORT_INSTRUCTIONS.md"
    cat > "$flow_instructions" << 'EOF'
# Node-RED Flow Import Instructions

## Overview
The AI Model Orchestra Controller includes two main Node-RED flows:

1. **orchestrator-main.json** - Main AI request routing and model selection
2. **resource-monitor.json** - System resource and model health monitoring

## Import Steps

1. **Start Node-RED:**
   ```bash
   node-red --port 1881
   ```

2. **Access Node-RED Interface:**
   Open http://\${ORCHESTRATOR_HOST:-localhost}:\${RESOURCE_PORTS_NODE_RED:-1881} in your browser

3. **Import Flows:**
   - Click the hamburger menu (☰) in the top right
   - Select "Import"
   - Choose "select a file to import"
   - Import both flow files from the workflows/ directory

4. **Deploy Flows:**
   - Click the "Deploy" button after importing
   - Verify flows are running without errors

## Flow Endpoints

After import and deployment, the flows will be available at:

- **Model Selection:** POST http://\${ORCHESTRATOR_HOST:-localhost}:\${RESOURCE_PORTS_NODE_RED:-1881}/flow/select-model
- **Request Routing:** POST http://\${ORCHESTRATOR_HOST:-localhost}:\${RESOURCE_PORTS_NODE_RED:-1881}/flow/route-request

## Testing

Test the flows with curl:

```bash
# Test model selection
curl -X POST http://\${ORCHESTRATOR_HOST:-localhost}:\${RESOURCE_PORTS_NODE_RED:-1881}/flow/select-model \\
  -H "Content-Type: application/json" \\
  -d '{"taskType": "completion", "requirements": {"priority": "normal"}}'

# Test request routing
curl -X POST http://\${ORCHESTRATOR_HOST:-localhost}:\${RESOURCE_PORTS_NODE_RED:-1881}/flow/route-request \\
  -H "Content-Type: application/json" \\
  -d '{"taskType": "completion", "prompt": "Hello, world!"}'
```
EOF
    log_success "Flow import instructions created: $flow_instructions"
}

# Create startup scripts
create_startup_scripts() {
    log_header "Creating Startup Scripts"
    
    # Create start script
    local start_script="$SCENARIO_DIR/start.sh"
    cat > "$start_script" << 'EOF'
#!/usr/bin/env bash

# AI Model Orchestra Controller Start Script

set -euo pipefail

readonly SCENARIO_DIR="${APP_ROOT}/scenarios/ai-model-orchestra-controller"

echo "🚀 Starting AI Model Orchestra Controller..."

# Start API server
echo "Starting API server..."
cd "$SCENARIO_DIR"
node initialization/configuration/api-server.js &
API_PID=$!

echo "✅ API server started (PID: $API_PID)"
echo "📊 Dashboard: http://\${ORCHESTRATOR_HOST:-localhost}:\${API_PORT:-8082}/dashboard"
echo "🔀 Model selection API: http://\${ORCHESTRATOR_HOST:-localhost}:\${API_PORT:-8082}/api/ai/select-model"
echo "🚦 Request routing API: http://\${ORCHESTRATOR_HOST:-localhost}:\${API_PORT:-8082}/api/ai/route-request"

# Save PID for stop script
echo $API_PID > "$SCENARIO_DIR/.api_pid"

# Wait for server to be ready
sleep 2

# Health check
if curl -s "http://\${ORCHESTRATOR_HOST:-localhost}:\${API_PORT:-8082}/health" >/dev/null; then
    echo "✅ Health check passed"
else
    echo "⚠️  Health check failed - service may still be starting"
fi

echo "🎉 AI Model Orchestra Controller is running!"
echo "Use ./stop.sh to stop the service"
EOF

    chmod +x "$start_script"
    log_success "Start script created: $start_script"
    
    # Create stop script
    local stop_script="$SCENARIO_DIR/stop.sh"
    cat > "$stop_script" << 'EOF'
#!/usr/bin/env bash

# AI Model Orchestra Controller Stop Script

set -euo pipefail

readonly SCENARIO_DIR="${APP_ROOT}/scenarios/ai-model-orchestra-controller"
readonly PID_FILE="$SCENARIO_DIR/.api_pid"

echo "🛑 Stopping AI Model Orchestra Controller..."

if [ -f "$PID_FILE" ]; then
    local API_PID=$(cat "$PID_FILE")
    if kill -0 "$API_PID" 2>/dev/null; then
        echo "Stopping API server (PID: $API_PID)..."
        kill "$API_PID"
        sleep 2
        
        # Force kill if still running
        if kill -0 "$API_PID" 2>/dev/null; then
            echo "Force stopping API server..."
            kill -9 "$API_PID"
        fi
        
        echo "✅ API server stopped"
    else
        echo "API server not running"
    fi
    
    rm -f "$PID_FILE"
else
    echo "PID file not found - service may not be running"
fi

echo "✅ AI Model Orchestra Controller stopped"
EOF

    chmod +x "$stop_script"
    log_success "Stop script created: $stop_script"
}

# Run health checks
run_health_checks() {
    log_header "Running Health Checks"
    
    # Check if required services are running
    local services_status=()
    
    local pg_host="${ORCHESTRATOR_HOST:-localhost}"
    local pg_port="${RESOURCE_PORTS_POSTGRES:-5432}"
    local redis_host="${ORCHESTRATOR_HOST:-localhost}"
    local redis_port="${RESOURCE_PORTS_REDIS:-6379}"
    local ollama_host="${ORCHESTRATOR_HOST:-localhost}"
    local ollama_port="${RESOURCE_PORTS_OLLAMA:-11434}"
    
    # Check PostgreSQL
    if command_exists pg_isready && pg_isready -h "$pg_host" -p "$pg_port" >/dev/null 2>&1; then
        services_status+=("✅ PostgreSQL: Running ($pg_host:$pg_port)")
    else
        services_status+=("⚠️  PostgreSQL: Not running ($pg_host:$pg_port)")
    fi
    
    # Check Redis
    if command_exists redis-cli && redis-cli -h "$redis_host" -p "$redis_port" ping >/dev/null 2>&1; then
        services_status+=("✅ Redis: Running ($redis_host:$redis_port)")
    else
        services_status+=("⚠️  Redis: Not running ($redis_host:$redis_port)")
    fi
    
    # Check Ollama
    if curl -s "http://$ollama_host:$ollama_port/api/tags" >/dev/null 2>&1; then
        services_status+=("✅ Ollama: Running ($ollama_host:$ollama_port)")
    else
        services_status+=("⚠️  Ollama: Not running ($ollama_host:$ollama_port)")
    fi
    
    # Display status
    for status in "${services_status[@]}"; do
        echo "$status"
    done
    
    echo
    log_info "Services can be started independently if needed:"
    log_info "- PostgreSQL: systemctl start postgresql"
    log_info "- Redis: systemctl start redis"
    log_info "- Ollama: systemctl start ollama"
}

# Main setup function
main() {
    log_header "$SCRIPT_NAME v$SCRIPT_VERSION"
    
    log_info "Setting up scenario: $SCENARIO_NAME"
    log_info "Scenario directory: $SCENARIO_DIR"
    
    # Run setup steps
    check_requirements
    install_dependencies
    create_configurations
    initialize_database
    configure_node_red
    create_startup_scripts
    run_health_checks
    
    # Final success message
    log_header "Setup Complete!"
    
    echo "🎉 AI Model Orchestra Controller setup completed successfully!"
    echo
    echo "📁 Scenario Location: $SCENARIO_DIR"
    echo "🚀 Start Service: ./start.sh"
    echo "🛑 Stop Service: ./stop.sh"
    echo "📊 Dashboard: http://\${ORCHESTRATOR_HOST:-localhost}:\${API_PORT:-8082}/dashboard (when running)"
    echo "📋 Flow Instructions: ./FLOW_IMPORT_INSTRUCTIONS.md"
    echo
    echo "Next Steps:"
    echo "1. Ensure required services are running (PostgreSQL, Redis, Ollama)"
    echo "2. Import Node-RED flows (see FLOW_IMPORT_INSTRUCTIONS.md)"
    echo "3. Start the orchestrator: ./start.sh"
    echo "4. Test the API endpoints"
    echo
    log_success "Setup completed! 🎉"
}

# Run main function
main "$@"