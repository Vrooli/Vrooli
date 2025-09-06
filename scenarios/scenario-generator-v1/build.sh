#!/usr/bin/env bash
# Scenario Generator V1 - Build and Run Script

set -euo pipefail

# Colors for output
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
CYAN='\033[1;36m'
NC='\033[0m'

# Configuration
API_PORT="${API_PORT:-8080}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"

info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

# Build the API
build_api() {
    info "Building Scenario Generator API..."
    
    cd api
    
    # Download dependencies
    info "Downloading Go dependencies..."
    go mod download
    
    # Build the binary
    info "Compiling API server..."
    go build -o scenario-generator-api .
    
    if [ $? -eq 0 ]; then
        success "API built successfully"
    else
        error "Failed to build API"
    fi
    
    cd ..
}

# Initialize database
init_database() {
    info "Initializing database schema..."
    
    # Check if PostgreSQL is running
    if ! pg_isready -h localhost -p "$POSTGRES_PORT" &> /dev/null; then
        error "PostgreSQL is not running on port $POSTGRES_PORT"
    fi
    
    # Create database if it doesn't exist
    psql -h localhost -p "$POSTGRES_PORT" -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'vrooli'" | grep -q 1 || \
    psql -h localhost -p "$POSTGRES_PORT" -U postgres -c "CREATE DATABASE vrooli;"
    
    # Apply schema
    psql -h localhost -p "$POSTGRES_PORT" -U postgres -d vrooli < initialization/postgres/schema.sql 2>/dev/null || true
    
    success "Database initialized"
}

# Run the API
run_api() {
    info "Starting Scenario Generator API on port $API_PORT..."
    
    export API_PORT
    export POSTGRES_PORT
    export POSTGRES_URL="postgres://postgres:postgres@localhost:${POSTGRES_PORT}/vrooli?sslmode=disable"
    
    cd api
    ./scenario-generator-api
}

# Main execution
case "${1:-build}" in
    build)
        build_api
        ;;
    init)
        init_database
        ;;
    run)
        run_api
        ;;
    all)
        build_api
        init_database
        run_api
        ;;
    *)
        echo "Usage: $0 {build|init|run|all}"
        echo "  build - Build the API binary"
        echo "  init  - Initialize the database"
        echo "  run   - Run the API server"
        echo "  all   - Build, initialize, and run"
        exit 1
        ;;
esac