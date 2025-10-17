#!/bin/bash
# AI Model Orchestra Controller - Dependencies Phase Tests
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$TEST_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "ℹ️  $1"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }
log_warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }

echo "📦 Running dependencies tests for AI Model Orchestra Controller..."

# Test Go installation and version
test_go_installation() {
    log_info "Checking Go installation..."
    
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed"
        return 1
    fi
    
    local go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    local major=$(echo "$go_version" | cut -d. -f1)
    local minor=$(echo "$go_version" | cut -d. -f2)
    
    if [ "$major" -lt 1 ] || [ "$major" -eq 1 -a "$minor" -lt 21 ]; then
        log_error "Go version $go_version is too old (minimum: 1.21)"
        return 1
    fi
    
    log_success "Go $go_version is installed"
    return 0
}

# Test Go modules download
test_go_modules() {
    log_info "Testing Go modules..."
    
    cd "$SCENARIO_ROOT/api"
    
    # Download modules
    if ! go mod download 2>/dev/null; then
        log_error "Failed to download Go modules"
        return 1
    fi
    
    # Verify modules
    if ! go mod verify 2>/dev/null; then
        log_error "Go modules verification failed"
        return 1
    fi
    
    log_success "Go modules downloaded and verified"
    return 0
}

# Test Go build capability
test_go_build() {
    log_info "Testing Go build..."
    
    cd "$SCENARIO_ROOT/api"
    
    # Try building without actually creating binary
    if ! go build -o /dev/null . 2>/dev/null; then
        log_error "Go build failed"
        return 1
    fi
    
    log_success "Go build test passed"
    return 0
}

# Test required system tools
test_system_tools() {
    log_info "Checking system tools..."
    
    local required_tools=(
        "curl"
        "jq" 
        "make"
    )
    
    for tool in "${required_tools[@]}"; do
        if command -v "$tool" >/dev/null 2>&1; then
            log_success "Found: $tool"
        else
            log_error "Missing required tool: $tool"
            return 1
        fi
    done
    
    return 0
}

# Test Node.js for UI server (optional)
test_nodejs() {
    log_info "Checking Node.js for UI server..."
    
    if ! command -v node >/dev/null 2>&1; then
        log_warn "Node.js not found - UI server won't work"
        return 0  # Not critical for core functionality
    fi
    
    local node_version=$(node --version | sed 's/v//' | cut -d. -f1)
    
    if [ "$node_version" -lt 18 ]; then
        log_warn "Node.js version $node_version is old (recommended: 18+)"
        return 0
    fi
    
    log_success "Node.js $(node --version) is available"
    return 0
}

# Test resource availability (if services are running)
test_resource_availability() {
    log_info "Checking resource availability..."
    
    # These are optional - services might not be running during dependency tests
    
    # Check PostgreSQL
    if command -v pg_isready >/dev/null 2>&1; then
        if pg_isready -h localhost -p "${RESOURCE_PORTS_POSTGRES:-5432}" >/dev/null 2>&1; then
            log_success "PostgreSQL is available"
        else
            log_warn "PostgreSQL not running (will be started by scenario)"
        fi
    else
        log_warn "PostgreSQL client tools not found"
    fi
    
    # Check Redis
    if command -v redis-cli >/dev/null 2>&1; then
        if timeout 2 redis-cli -h localhost -p "${RESOURCE_PORTS_REDIS:-6379}" ping >/dev/null 2>&1; then
            log_success "Redis is available"
        else
            log_warn "Redis not running (will be started by scenario)"
        fi
    else
        log_warn "Redis client tools not found"
    fi
    
    # Check if Ollama is available
    if timeout 5 curl -s "http://localhost:${RESOURCE_PORTS_OLLAMA:-11434}/api/tags" >/dev/null 2>&1; then
        log_success "Ollama is available"
    else
        log_warn "Ollama not running (will be started by scenario)"
    fi
    
    return 0  # Resource checks are not critical for dependency phase
}

# Test Docker availability (for resource management)
test_docker() {
    log_info "Checking Docker availability..."
    
    if ! command -v docker >/dev/null 2>&1; then
        log_warn "Docker not found - container monitoring won't work"
        return 0  # Not critical
    fi
    
    # Test Docker daemon connectivity
    if ! timeout 5 docker info >/dev/null 2>&1; then
        log_warn "Docker daemon not accessible"
        return 0
    fi
    
    log_success "Docker is available"
    return 0
}

# Test CLI installation capability
test_cli_installation() {
    log_info "Testing CLI installation..."
    
    local install_script="$SCENARIO_ROOT/cli/install.sh"
    
    if [ ! -f "$install_script" ]; then
        log_error "CLI install script not found"
        return 1
    fi
    
    if [ ! -x "$install_script" ]; then
        log_error "CLI install script not executable"
        return 1
    fi
    
    # Check if .vrooli/bin directory can be created
    local vrooli_bin="${HOME}/.vrooli/bin"
    if ! mkdir -p "$vrooli_bin" 2>/dev/null; then
        log_error "Cannot create .vrooli/bin directory"
        return 1
    fi
    
    log_success "CLI installation prerequisites met"
    return 0
}

# Test environment variable handling
test_environment() {
    log_info "Testing environment variable handling..."
    
    # Test that default values work
    local test_host="${ORCHESTRATOR_HOST:-localhost}"
    local test_port="${API_PORT:-8080}"
    
    if [ -z "$test_host" ] || [ -z "$test_port" ]; then
        log_error "Environment variable defaults not working"
        return 1
    fi
    
    log_success "Environment variable handling works"
    return 0
}

# Test file permissions
test_permissions() {
    log_info "Checking file permissions..."
    
    # Check that key files are readable
    local key_files=(
        ".vrooli/service.json"
        "api/main.go"
        "ui/dashboard.html"
    )
    
    for file in "${key_files[@]}"; do
        local full_path="$SCENARIO_ROOT/$file"
        if [ ! -r "$full_path" ]; then
            log_error "File not readable: $file"
            return 1
        fi
    done
    
    # Check that executable files are executable
    local exec_files=(
        "cli/ai-orchestra"
        "cli/install.sh"
        "test/run-tests.sh"
        "test/phases/test-structure.sh"
    )
    
    for file in "${exec_files[@]}"; do
        local full_path="$SCENARIO_ROOT/$file"
        if [ -f "$full_path" ] && [ ! -x "$full_path" ]; then
            log_error "File not executable: $file"
            return 1
        fi
    done
    
    log_success "File permissions are correct"
    return 0
}

# Run all dependency tests
echo "Starting dependency validation tests..."

# Execute all tests
test_go_installation || exit 1
test_go_modules || exit 1  
test_go_build || exit 1
test_system_tools || exit 1
test_nodejs # Non-critical
test_resource_availability # Non-critical
test_docker # Non-critical
test_cli_installation || exit 1
test_environment || exit 1
test_permissions || exit 1

echo ""
log_success "All dependency tests passed!"
echo "✅ Dependencies phase completed successfully"