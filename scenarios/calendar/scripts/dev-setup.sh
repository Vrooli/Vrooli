#!/bin/bash
# Calendar Scenario Development Setup Script
# Automates common development tasks

set -e

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT_NAME="$(basename "${BASH_SOURCE[0]}")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} Calendar Development Setup${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

show_help() {
    echo "Calendar Development Setup Script"
    echo ""
    echo "Usage: $SCRIPT_NAME [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  setup     - Full development environment setup"
    echo "  deps      - Install dependencies only"
    echo "  test      - Run test suite"
    echo "  lint      - Run linters and formatters"
    echo "  clean     - Clean build artifacts and caches"
    echo "  reset     - Reset development environment"
    echo "  check     - Check development environment health"
    echo "  help      - Show this help message"
    echo ""
}

check_prerequisites() {
    print_info "Checking prerequisites..."
    
    local missing_tools=()
    
    # Required tools
    if ! command -v go >/dev/null 2>&1; then
        missing_tools+=("go")
    fi
    
    if ! command -v node >/dev/null 2>&1; then
        missing_tools+=("node")
    fi
    
    if ! command -v npm >/dev/null 2>&1; then
        missing_tools+=("npm")
    fi
    
    # Recommended tools
    if ! command -v jq >/dev/null 2>&1; then
        print_warning "jq not installed (recommended for JSON processing)"
    fi
    
    if ! command -v golangci-lint >/dev/null 2>&1; then
        print_warning "golangci-lint not installed (recommended for Go linting)"
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        print_error "Missing required tools: ${missing_tools[*]}"
        echo ""
        echo "Please install the missing tools and try again."
        echo ""
        echo "Installation suggestions:"
        echo "  Go: https://golang.org/doc/install"
        echo "  Node.js: https://nodejs.org/"
        echo "  jq: https://stedolan.github.io/jq/"
        echo "  golangci-lint: https://golangci-lint.run/usage/install/"
        return 1
    fi
    
    print_success "All prerequisites are installed"
}

setup_go_environment() {
    print_info "Setting up Go environment..."
    
    cd "$SCENARIO_DIR/api"
    
    # Clean and tidy modules
    go clean -cache
    go mod tidy
    go mod download
    
    # Verify modules
    go mod verify
    
    print_success "Go environment setup completed"
}

setup_node_environment() {
    print_info "Setting up Node.js environment..."
    
    cd "$SCENARIO_DIR/ui"
    
    # Install dependencies
    npm ci
    
    # Verify installation
    npm audit --audit-level=moderate
    
    print_success "Node.js environment setup completed"
}

create_env_files() {
    print_info "Creating environment configuration files..."
    
    # Create .env file for development
    cat > "$SCENARIO_DIR/.env.development" << EOF
# Calendar Development Environment Configuration
# Copy this to .env and modify as needed

# API Configuration
# API_PORT and UI_PORT are dynamically assigned by Vrooli lifecycle
# Ranges: API 15000-19999, UI 35000-39999
API_PORT=\${API_PORT}
UI_PORT=\${UI_PORT}

# Database Configuration
# These should be set via environment or resource-postgres commands
POSTGRES_HOST=\${POSTGRES_HOST}
POSTGRES_PORT=\${POSTGRES_PORT}
POSTGRES_USER=\${POSTGRES_USER}
POSTGRES_PASSWORD=\${POSTGRES_PASSWORD}
POSTGRES_DB=\${POSTGRES_DB:-calendar_system}

# Resource URLs - Set by Vrooli resource management
QDRANT_URL=\${QDRANT_URL}
AUTH_SERVICE_URL=\${AUTH_SERVICE_URL}
NOTIFICATION_SERVICE_URL=\${NOTIFICATION_SERVICE_URL}
OLLAMA_URL=\${OLLAMA_URL}

# Security - MUST be set via environment for production
JWT_SECRET=\${JWT_SECRET}

# Feature Flags
ENABLE_DEBUG_LOGGING=true
ENABLE_CORS=true
ENABLE_METRICS=true

# Development Tools
HOT_RELOAD=true
PRETTY_LOGS=true
EOF

    print_success "Environment template created"
    print_warning "IMPORTANT: All values must be set via environment variables"
    print_info "No hardcoded defaults are provided per Vrooli standards"
}

setup_git_hooks() {
    print_info "Setting up Git hooks..."
    
    cd "$SCENARIO_DIR"
    
    # Create pre-commit hook
    mkdir -p .git/hooks
    
    cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# Calendar pre-commit hook

echo "Running pre-commit checks..."

# Format Go code
if [ -d api ]; then
    cd api
    if command -v gofumpt >/dev/null 2>&1; then
        gofumpt -w .
    else
        gofmt -w .
    fi
    cd ..
fi

# Format UI code
if [ -d ui ] && [ -f ui/package.json ]; then
    cd ui
    if npm run format:check >/dev/null 2>&1; then
        echo "UI code formatting OK"
    else
        echo "Formatting UI code..."
        npm run format || true
    fi
    cd ..
fi

# Run quick tests
echo "Running quick tests..."
make test >/dev/null 2>&1 || echo "Some tests failed - consider running 'make test' manually"

echo "Pre-commit checks completed"
EOF

    chmod +x .git/hooks/pre-commit
    
    print_success "Git hooks installed"
}

setup_vscode_config() {
    print_info "Setting up VS Code configuration..."
    
    mkdir -p "$SCENARIO_DIR/.vscode"
    
    # VS Code settings
    cat > "$SCENARIO_DIR/.vscode/settings.json" << EOF
{
  "go.useLanguageServer": true,
  "go.formatTool": "gofumpt",
  "go.lintTool": "golangci-lint",
  "go.testFlags": ["-v"],
  "typescript.preferences.importModuleSpecifier": "relative",
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  },
  "files.exclude": {
    "**/node_modules": true,
    "**/dist": true,
    "**/*.log": true
  },
  "search.exclude": {
    "**/node_modules": true,
    "**/dist": true,
    "**/*.log": true
  }
}
EOF

    # VS Code launch configuration
    cat > "$SCENARIO_DIR/.vscode/launch.json" << EOF
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Calendar API",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "\${workspaceFolder}/api",
      "env": {
        "VROOLI_LIFECYCLE_MANAGED": "true",
        "API_PORT": "\${env:API_PORT}"
      },
      "args": [],
      "preLaunchTask": "echo 'Set API_PORT environment variable before debugging'"
    }
  ]
}
EOF

    # VS Code tasks
    cat > "$SCENARIO_DIR/.vscode/tasks.json" << EOF
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Build API",
      "type": "shell",
      "command": "make",
      "args": ["build"],
      "group": {
        "kind": "build",
        "isDefault": true
      },
      "presentation": {
        "echo": true,
        "reveal": "always",
        "focus": false,
        "panel": "shared"
      }
    },
    {
      "label": "Run Tests",
      "type": "shell",
      "command": "make",
      "args": ["test"],
      "group": "test",
      "presentation": {
        "echo": true,
        "reveal": "always",
        "focus": false,
        "panel": "shared"
      }
    },
    {
      "label": "Start Development Server",
      "type": "shell",
      "command": "make",
      "args": ["dev"],
      "group": "build",
      "presentation": {
        "echo": true,
        "reveal": "always",
        "focus": false,
        "panel": "shared"
      }
    }
  ]
}
EOF

    print_success "VS Code configuration created"
}

run_health_check() {
    print_info "Running development environment health check..."
    
    local issues=0
    
    # Check Go environment
    cd "$SCENARIO_DIR/api"
    if ! go version >/dev/null 2>&1; then
        print_error "Go not working"
        issues=$((issues + 1))
    else
        print_success "Go environment OK"
    fi
    
    # Check Node environment
    cd "$SCENARIO_DIR/ui"
    if ! npm --version >/dev/null 2>&1; then
        print_error "npm not working"
        issues=$((issues + 1))
    else
        print_success "Node.js environment OK"
    fi
    
    # Check build capability
    cd "$SCENARIO_DIR"
    if make build >/dev/null 2>&1; then
        print_success "Build system OK"
    else
        print_warning "Build system has issues"
        issues=$((issues + 1))
    fi
    
    # Check test capability
    if make test >/dev/null 2>&1; then
        print_success "Test system OK"
    else
        print_warning "Some tests are failing"
    fi
    
    if [ $issues -eq 0 ]; then
        print_success "Development environment is healthy!"
        return 0
    else
        print_warning "$issues issues found in development environment"
        return 1
    fi
}

clean_environment() {
    print_info "Cleaning development environment..."
    
    cd "$SCENARIO_DIR"
    
    # Clean build artifacts
    make clean >/dev/null 2>&1 || true
    
    # Clean Go cache
    if [ -d api ]; then
        cd api
        go clean -cache -modcache -testcache || true
        cd ..
    fi
    
    # Clean Node cache
    if [ -d ui ]; then
        cd ui
        rm -rf node_modules/.cache || true
        npm cache clean --force || true
        cd ..
    fi
    
    # Clean temporary files
    find . -name "*.log" -delete || true
    find . -name "*.tmp" -delete || true
    
    print_success "Environment cleaned"
}

full_setup() {
    print_header
    
    print_info "Starting full development environment setup..."
    echo ""
    
    check_prerequisites || exit 1
    setup_go_environment
    setup_node_environment
    create_env_files
    setup_git_hooks
    setup_vscode_config
    
    echo ""
    print_success "🎉 Development environment setup completed!"
    echo ""
    print_info "Next steps:"
    echo "  1. Review .env.development and copy to .env if needed"
    echo "  2. Start services: make run"
    echo "  3. Run tests: make test"
    echo "  4. Start development: make dev"
    echo ""
    print_info "VS Code users: Open this directory in VS Code for best experience"
}

# Main execution
main() {
    case "${1:-help}" in
        "setup")
            full_setup
            ;;
        "deps")
            check_prerequisites && setup_go_environment && setup_node_environment
            ;;
        "test")
            cd "$SCENARIO_DIR" && make test
            ;;
        "lint")
            cd "$SCENARIO_DIR" && make lint
            ;;
        "clean")
            clean_environment
            ;;
        "reset")
            clean_environment && full_setup
            ;;
        "check")
            run_health_check
            ;;
        "help"|*)
            show_help
            ;;
    esac
}

main "$@"