#!/usr/bin/env bash
# Scenario Generator V1 - Startup and Deployment Script

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/scenario-generator-v1/deployment"
PROJECT_DIR="${APP_ROOT}/scenarios/scenario-generator-v1"

# Colors for output
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
CYAN='\033[1;36m'
NC='\033[0m'

# Configuration
API_PORT="${API_PORT:-8080}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_URL="${POSTGRES_URL:-postgres://postgres:postgres@localhost:${POSTGRES_PORT}/postgres?sslmode=disable}"

info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

banner() {
    echo -e "${CYAN}"
    cat << 'EOF'
    ███████╗ ██████╗███████╗███╗   ██╗ █████╗ ██████╗ ██╗ ██████╗ 
    ██╔════╝██╔════╝██╔════╝████╗  ██║██╔══██╗██╔══██╗██║██╔═══██╗
    ███████╗██║     █████╗  ██╔██╗ ██║███████║██████╔╝██║██║   ██║
    ╚════██║██║     ██╔══╝  ██║╚██╗██║██╔══██║██╔══██╗██║██║   ██║
    ███████║╚██████╗███████╗██║ ╚████║██║  ██║██║  ██║██║╚██████╔╝
    ╚══════╝ ╚═════╝╚══════╝╚═╝  ╚═══╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝ ╚═════╝ 
    
         ██████╗ ███████╗███╗   ██╗███████╗██████╗  █████╗ ████████╗ ██████╗ ██████╗ 
        ██╔════╝ ██╔════╝████╗  ██║██╔════╝██╔══██╗██╔══██╗╚══██╔══╝██╔═══██╗██╔══██╗
        ██║  ███╗█████╗  ██╔██╗ ██║█████╗  ██████╔╝███████║   ██║   ██║   ██║██████╔╝
        ██║   ██║██╔══╝  ██║╚██╗██║██╔══╝  ██╔══██╗██╔══██║   ██║   ██║   ██║██╔══██╗
        ╚██████╔╝███████╗██║ ╚████║███████╗██║  ██║██║  ██║   ██║   ╚██████╔╝██║  ██║
         ╚═════╝ ╚══════╝╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
                                                                                       
                         🤖 AI-Powered Scenario Generator v1.0.0 🤖
EOF
    echo -e "${NC}"
}

check_dependencies() {
    info "Checking dependencies..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        error "Go is not installed. Please install Go 1.21 or later."
    fi
    
    # Check if PostgreSQL is accessible
    if ! command -v psql &> /dev/null; then
        warn "psql command not found. Database initialization may fail."
    fi
    
    # Check if jq is installed (for CLI)
    if ! command -v jq &> /dev/null; then
        warn "jq is not installed. CLI may not work properly."
    fi
    
    # Check if Claude Code is available
    if command -v claude &> /dev/null; then
        success "Claude Code CLI found"
    else
        warn "Claude Code CLI not found. Generation features will be limited."
    fi
    
    success "Dependency check completed"
}

wait_for_postgres() {
    info "Waiting for PostgreSQL to be ready..."
    
    local attempts=0
    local max_attempts=30
    
    while [ $attempts -lt $max_attempts ]; do
        if pg_isready -h localhost -p "$POSTGRES_PORT" -U postgres &> /dev/null; then
            success "PostgreSQL is ready"
            return 0
        fi
        
        attempts=$((attempts + 1))
        echo -n "."
        sleep 1
    done
    
    error "PostgreSQL is not ready after ${max_attempts} seconds"
}

initialize_database() {
    info "Initializing database schema..."
    
    cd "$PROJECT_DIR"
    
    # Check if schema file exists
    if [ ! -f "initialization/postgres/schema.sql" ]; then
        error "Database schema file not found: initialization/postgres/schema.sql"
    fi
    
    # Initialize schema
    if psql "$POSTGRES_URL" -f "initialization/postgres/schema.sql" &> /dev/null; then
        success "Database schema initialized"
    else
        warn "Database schema may already exist or initialization failed"
    fi
}

build_api() {
    info "Building Go API..."
    
    cd "$PROJECT_DIR/api"
    
    # Download dependencies
    go mod tidy
    
    # Build the API
    if go build -o scenario-generator-api main.go; then
        success "API built successfully"
    else
        error "Failed to build API"
    fi
}

start_api() {
    info "Starting API server on port $API_PORT..."
    
    cd "$PROJECT_DIR/api"
    
    # Set environment variables
    export API_PORT="$API_PORT"
    export POSTGRES_URL="$POSTGRES_URL"
    export POSTGRES_PORT="$POSTGRES_PORT"
    
    # Start the API server
    if [ -f "./scenario-generator-api" ]; then
        ./scenario-generator-api &
        local api_pid=$!
        echo "$api_pid" > /tmp/scenario-generator-api.pid
        
        # Wait for API to be ready
        local attempts=0
        local max_attempts=30
        
        while [ $attempts -lt $max_attempts ]; do
            if curl -s "http://localhost:$API_PORT/health" &> /dev/null; then
                success "API server is running (PID: $api_pid)"
                return 0
            fi
            
            attempts=$((attempts + 1))
            sleep 1
        done
        
        error "API server failed to start"
    else
        error "API binary not found. Run build first."
    fi
}

install_ui_dependencies() {
    info "Installing UI dependencies..."
    
    cd "$PROJECT_DIR/ui"
    
    if command -v npm &> /dev/null; then
        npm install
        success "UI dependencies installed"
    else
        warn "npm not found. UI will not be available."
        return 1
    fi
}

start_ui() {
    info "Starting UI development server..."
    
    cd "$PROJECT_DIR/ui"
    
    if command -v npm &> /dev/null; then
        # Set API URL
        export REACT_APP_API_URL="http://localhost:$API_PORT"
        
        npm start &
        local ui_pid=$!
        echo "$ui_pid" > /tmp/scenario-generator-ui.pid
        
        success "UI development server started (PID: $ui_pid)"
        info "UI will be available at http://localhost:3000"
    else
        warn "npm not found. Skipping UI startup."
    fi
}

show_status() {
    echo
    info "🚀 Scenario Generator V1 Status:"
    echo
    echo -e "${GREEN}Services Running:${NC}"
    echo -e "  📡 API Server: http://localhost:$API_PORT"
    echo -e "  🗄️  Database: PostgreSQL on port $POSTGRES_PORT"
    
    if [ -f "/tmp/scenario-generator-ui.pid" ]; then
        echo -e "  🌐 UI Dashboard: http://localhost:3000"
    fi
    
    echo
    echo -e "${CYAN}Available Tools:${NC}"
    echo -e "  🔧 CLI: scenario-generator help"
    echo -e "  🔍 API Health: curl http://localhost:$API_PORT/health"
    echo -e "  📊 API Docs: http://localhost:$API_PORT/api/scenarios"
    
    echo
    echo -e "${YELLOW}Quick Commands:${NC}"
    echo -e "  • List scenarios: scenario-generator list"
    echo -e "  • Generate scenario: scenario-generator generate \"My App\" \"Description\" \"Detailed prompt\""
    echo -e "  • View templates: scenario-generator templates"
    echo -e "  • Check status: scenario-generator status"
}

stop_services() {
    info "Stopping services..."
    
    # Stop API
    if [ -f "/tmp/scenario-generator-api.pid" ]; then
        local api_pid=$(cat /tmp/scenario-generator-api.pid)
        if kill "$api_pid" 2>/dev/null; then
            success "API server stopped"
        fi
        rm -f /tmp/scenario-generator-api.pid
    fi
    
    # Stop UI
    if [ -f "/tmp/scenario-generator-ui.pid" ]; then
        local ui_pid=$(cat /tmp/scenario-generator-ui.pid)
        if kill "$ui_pid" 2>/dev/null; then
            success "UI server stopped"
        fi
        rm -f /tmp/scenario-generator-ui.pid
    fi
    
    success "All services stopped"
}

main() {
    local command="${1:-start}"
    
    case "$command" in
        "start"|"deploy")
            banner
            check_dependencies
            wait_for_postgres
            initialize_database
            build_api
            start_api
            install_ui_dependencies && start_ui
            show_status
            ;;
        "stop")
            stop_services
            ;;
        "restart")
            stop_services
            sleep 2
            main start
            ;;
        "status")
            show_status
            ;;
        "build")
            info "Building components..."
            build_api
            install_ui_dependencies
            success "Build completed"
            ;;
        "help")
            echo "Usage: $0 [start|stop|restart|status|build|help]"
            echo
            echo "Commands:"
            echo "  start    - Start all services (default)"
            echo "  deploy   - Alias for start"
            echo "  stop     - Stop all services"
            echo "  restart  - Restart all services"
            echo "  status   - Show service status"
            echo "  build    - Build API and install UI dependencies"
            echo "  help     - Show this help message"
            ;;
        *)
            error "Unknown command: $command. Use 'help' for usage."
            ;;
    esac
}

# Handle signals for graceful shutdown
trap 'stop_services; exit 0' SIGINT SIGTERM

main "$@"
