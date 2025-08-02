#!/bin/bash
# Claude Code Sandbox Wrapper Script
# Provides safe, isolated environment for testing claude-code

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCKER_DIR="${SCRIPT_DIR}/docker"
TEST_FILES_DIR="${SCRIPT_DIR}/test-files"

# Configuration
CLAUDE_SANDBOX_CONFIG="${CLAUDE_SANDBOX_CONFIG:-${HOME}/.claude-sandbox}"
CLAUDE_SANDBOX_IMAGE="vrooli/claude-code-sandbox:latest"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running or you don't have permissions"
        exit 1
    fi
}

# Build the sandbox image
build_sandbox() {
    print_info "Building Claude Code sandbox image..."
    cd "${DOCKER_DIR}"
    
    if docker compose build claude-sandbox; then
        print_success "Sandbox image built successfully"
    else
        print_error "Failed to build sandbox image"
        exit 1
    fi
}

# Setup sandbox authentication
setup_auth() {
    if [[ -d "${CLAUDE_SANDBOX_CONFIG}" ]]; then
        print_warning "Sandbox config already exists at ${CLAUDE_SANDBOX_CONFIG}"
        read -p "Do you want to reconfigure authentication? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            return 0
        fi
    fi
    
    print_info "Setting up sandbox authentication..."
    mkdir -p "${CLAUDE_SANDBOX_CONFIG}"
    
    # Create temporary auth setup script
    cat > /tmp/claude-auth-setup.sh << 'EOF'
#!/bin/bash
export CLAUDE_CONFIG_DIR="${CLAUDE_SANDBOX_CONFIG}"
echo "🔐 Authenticating Claude Code for sandbox use..."
echo "This will use your Claude subscription but with isolated credentials."
echo ""
claude login
claude doctor
EOF
    
    chmod +x /tmp/claude-auth-setup.sh
    
    # Run authentication setup
    CLAUDE_CONFIG_DIR="${CLAUDE_SANDBOX_CONFIG}" bash /tmp/claude-auth-setup.sh
    rm -f /tmp/claude-auth-setup.sh
    
    if [[ -f "${CLAUDE_SANDBOX_CONFIG}/.credentials.json" ]]; then
        print_success "Authentication setup complete"
        chmod 600 "${CLAUDE_SANDBOX_CONFIG}/.credentials.json"
    else
        print_error "Authentication setup failed"
        exit 1
    fi
}

# Check authentication
check_auth() {
    if [[ ! -d "${CLAUDE_SANDBOX_CONFIG}" ]] || [[ ! -f "${CLAUDE_SANDBOX_CONFIG}/.credentials.json" ]]; then
        print_warning "Sandbox authentication not configured"
        setup_auth
    fi
}

# Run claude in sandbox
run_sandbox() {
    local command="${1:-}"
    shift || true
    
    check_docker
    check_auth
    
    # Ensure test files directory exists
    mkdir -p "${TEST_FILES_DIR}"
    
    # Build image if it doesn't exist
    if ! docker images | grep -q "${CLAUDE_SANDBOX_IMAGE}"; then
        build_sandbox
    fi
    
    print_info "Starting Claude Code sandbox..."
    print_warning "⚠️  SANDBOX MODE: Limited to test-files directory only"
    echo ""
    
    case "${command}" in
        "interactive"|"")
            # Interactive mode
            docker compose -f "${DOCKER_DIR}/docker-compose.yml" \
                run --rm \
                -e CLAUDE_SANDBOX_CONFIG="${CLAUDE_SANDBOX_CONFIG}" \
                claude-sandbox
            ;;
        "run")
            # Run a single command
            docker compose -f "${DOCKER_DIR}/docker-compose.yml" \
                run --rm \
                -e CLAUDE_SANDBOX_CONFIG="${CLAUDE_SANDBOX_CONFIG}" \
                claude-sandbox "$@"
            ;;
        "exec")
            # Execute in running container
            docker compose -f "${DOCKER_DIR}/docker-compose.yml" \
                exec claude-sandbox claude "$@"
            ;;
        *)
            # Pass through to claude
            docker compose -f "${DOCKER_DIR}/docker-compose.yml" \
                run --rm \
                -e CLAUDE_SANDBOX_CONFIG="${CLAUDE_SANDBOX_CONFIG}" \
                claude-sandbox "${command}" "$@"
            ;;
    esac
}

# Stop sandbox
stop_sandbox() {
    print_info "Stopping Claude Code sandbox..."
    cd "${DOCKER_DIR}"
    docker compose down
    print_success "Sandbox stopped"
}

# Clean up sandbox
cleanup_sandbox() {
    print_warning "This will remove sandbox containers and images"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        stop_sandbox
        docker rmi "${CLAUDE_SANDBOX_IMAGE}" 2>/dev/null || true
        print_success "Sandbox cleaned up"
    fi
}

# Show usage
usage() {
    cat << EOF
Claude Code Sandbox - Safe testing environment

Usage: $(basename "$0") [COMMAND] [OPTIONS]

Commands:
    setup           Set up sandbox authentication
    build           Build the sandbox Docker image
    run [args]      Run claude with arguments in sandbox
    interactive     Start interactive claude session (default)
    exec [args]     Execute command in running sandbox
    stop            Stop the sandbox container
    cleanup         Remove sandbox containers and images
    help            Show this help message

Environment Variables:
    CLAUDE_SANDBOX_CONFIG   Directory for sandbox credentials (default: ~/.claude-sandbox)

Examples:
    # First time setup
    $(basename "$0") setup
    
    # Interactive session
    $(basename "$0")
    
    # Run specific command
    $(basename "$0") run --help
    $(basename "$0") run -p "Analyze this code" --print
    
    # Execute in running container
    $(basename "$0") exec doctor

Safety Features:
    ✓ Runs as non-root user
    ✓ Isolated network (no internet by default)
    ✓ Limited to test-files directory
    ✓ Resource limits enforced
    ✓ Separate authentication from main claude
EOF
}

# Main command handling
case "${1:-interactive}" in
    setup)
        setup_auth
        ;;
    build)
        check_docker
        build_sandbox
        ;;
    stop)
        stop_sandbox
        ;;
    cleanup)
        cleanup_sandbox
        ;;
    help|--help|-h)
        usage
        ;;
    run|interactive|exec)
        run_sandbox "$@"
        ;;
    *)
        # Assume it's a claude command
        run_sandbox "$@"
        ;;
esac