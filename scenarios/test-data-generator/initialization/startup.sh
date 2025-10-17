#!/bin/bash

# Test Data Generator Scenario Startup Script
# This script initializes the test-data-generator scenario

set -e

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_DIR="$SCENARIO_DIR/api"
UI_DIR="$SCENARIO_DIR/ui"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_color() {
    local color=$1
    shift
    echo -e "${color}$*${NC}"
}

print_error() {
    print_color $RED "❌ Error: $*"
}

print_success() {
    print_color $GREEN "✅ $*"
}

print_info() {
    print_color $BLUE "ℹ️  $*"
}

print_warning() {
    print_color $YELLOW "⚠️  $*"
}

# Check if Node.js is available
check_nodejs() {
    if ! command -v node &> /dev/null; then
        print_error "Node.js is not installed"
        print_info "Please install Node.js 16+ and try again"
        exit 1
    fi
    
    local node_version=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
    if [ "$node_version" -lt 16 ]; then
        print_warning "Node.js version $node_version detected. Version 16+ recommended."
    else
        print_success "Node.js $(node --version) detected"
    fi
}

# Install API dependencies
install_api_deps() {
    print_info "Installing API dependencies..."
    
    cd "$API_DIR"
    
    if [ ! -f package.json ]; then
        print_error "package.json not found in API directory"
        exit 1
    fi
    
    npm install
    print_success "API dependencies installed"
}

# Install UI dependencies
install_ui_deps() {
    print_info "Installing UI dependencies..."
    
    cd "$UI_DIR"
    
    if [ ! -f package.json ]; then
        print_error "package.json not found in UI directory"
        exit 1
    fi
    
    npm install
    print_success "UI dependencies installed"
}

# Check ports availability
check_ports() {
    local api_port=${API_PORT:-3001}
    local ui_port=${UI_PORT:-3002}
    
    if command -v lsof &> /dev/null; then
        if lsof -i ":$api_port" &> /dev/null; then
            print_warning "Port $api_port is already in use"
        fi
        
        if lsof -i ":$ui_port" &> /dev/null; then
            print_warning "Port $ui_port is already in use"
        fi
    fi
    
    print_info "API will run on port $api_port"
    print_info "UI will run on port $ui_port"
}

# Create necessary directories
create_directories() {
    print_info "Creating necessary directories..."
    
    local dirs=(
        "$SCENARIO_DIR/data"
        "$SCENARIO_DIR/logs"
        "$SCENARIO_DIR/.tmp"
    )
    
    for dir in "${dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
            print_success "Created directory: $dir"
        fi
    done
}

# Validate scenario structure
validate_structure() {
    print_info "Validating scenario structure..."
    
    local required_files=(
        "$SCENARIO_DIR/README.md"
        "$SCENARIO_DIR/scenario-test.yaml"
        "$API_DIR/package.json"
        "$API_DIR/server.js"
        "$UI_DIR/package.json"
        "$UI_DIR/server.js"
        "$UI_DIR/index.html"
    )
    
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            print_error "Required file missing: $file"
            exit 1
        fi
    done
    
    print_success "Scenario structure is valid"
}

# Main initialization
main() {
    print_info "Initializing Test Data Generator scenario..."
    echo
    
    validate_structure
    check_nodejs
    check_ports
    create_directories
    install_api_deps
    install_ui_deps
    
    echo
    print_success "Test Data Generator scenario initialized successfully!"
    echo
    print_info "To start the services:"
    print_info "  API: cd $API_DIR && npm start"
    print_info "  UI:  cd $UI_DIR && npm start"
    echo
    print_info "Or use the development commands:"
    print_info "  API: cd $API_DIR && npm run dev"
    print_info "  UI:  cd $UI_DIR && npm run dev"
    echo
    print_info "CLI installation:"
    print_info "  cd $SCENARIO_DIR/cli && ./install.sh"
    echo
}

main "$@"