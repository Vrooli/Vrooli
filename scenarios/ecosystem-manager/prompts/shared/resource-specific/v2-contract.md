# 📐 v2.0 Resource Standards

## Mandatory Requirements

Every resource in Vrooli MUST meet these v2.0 contract requirements. No exceptions.

## Directory Structure

```bash
resource-name/
├── .vrooli/
│   ├── app-identity.json      # Resource identity and versioning
│   └── service.json           # v2.0 contract definition
├── lib/                       # Core functionality
│   ├── common.sh             # Shared utilities (REQUIRED)
│   ├── health.sh             # Health check implementation (REQUIRED)
│   ├── inject.sh             # Resource injection logic (REQUIRED)
│   ├── status.sh             # Status reporting (REQUIRED)
│   └── content.sh            # Content management (REQUIRED)
├── cli.sh                    # CLI entry point (REQUIRED)
├── manage.sh                 # Lifecycle management (REQUIRED)
├── README.md                 # Comprehensive documentation (REQUIRED)
├── PRD.md                    # Product Requirements Document (REQUIRED)
└── test/
    └── integration.bats      # Integration tests (REQUIRED)
```

## Health Check Implementation

### Required Contract Structure
```bash
#!/bin/bash  
# lib/health.sh - REQUIRED file

# Must implement check_health() function
# Must handle timeouts and retries
# Must use consistent logging patterns

# Source common utilities
source "$(dirname "$0")/common.sh"

# For comprehensive implementation examples and patterns:
{{INCLUDE: resource-specific/health-checks.md}}
```

## Lifecycle Hooks

### Required Implementations

#### 1. Setup Hook
```bash
# In manage.sh
setup() {
    log_info "Setting up ${RESOURCE_NAME}..."
    
    # Check prerequisites
    check_prerequisites || return 1
    
    # Initialize configuration
    init_config || return 1
    
    # Create required directories
    mkdir -p "${DATA_DIR}" "${LOG_DIR}" "${CONFIG_DIR}"
    
    # Pull/install required components
    install_components || return 1
    
    # Initialize database/storage if needed
    init_storage || return 1
    
    log_success "${RESOURCE_NAME} setup complete"
}
```

#### 2. Develop Hook
```bash
develop() {
    log_info "Starting ${RESOURCE_NAME} in development mode..."
    
    # Ensure setup has been run
    check_setup || setup || return 1
    
    # Start dependent services
    start_dependencies || return 1
    
    # Start main service
    start_service || return 1
    
    # Wait for health
    wait_for_health || return 1
    
    # Show access information
    show_urls
    
    log_success "${RESOURCE_NAME} is running"
}
```

#### 3. Test Hook
```bash
test() {
    log_info "Testing ${RESOURCE_NAME}..."
    
    # Run unit tests if present
    if [ -f "test/unit.bats" ]; then
        bats test/unit.bats || return 1
    fi
    
    # Run integration tests
    if [ -f "test/integration.bats" ]; then
        bats test/integration.bats || return 1
    fi
    
    # Test health endpoints
    check_all_endpoints || return 1
    
    # Test CLI commands
    test_cli_commands || return 1
    
    log_success "All tests passed"
}
```

#### 4. Stop Hook
```bash
stop() {
    log_info "Stopping ${RESOURCE_NAME}..."
    
    # Graceful shutdown
    if [ -f "${PID_FILE}" ]; then
        local pid=$(cat "${PID_FILE}")
        kill -TERM "$pid" 2>/dev/null
        
        # Wait for graceful shutdown
        local count=0
        while kill -0 "$pid" 2>/dev/null && [ $count -lt 10 ]; do
            sleep 1
            count=$((count + 1))
        done
        
        # Force kill if still running
        if kill -0 "$pid" 2>/dev/null; then
            log_warn "Force killing ${RESOURCE_NAME}"
            kill -KILL "$pid" 2>/dev/null
        fi
        
        rm -f "${PID_FILE}"
    fi
    
    # Stop dependencies
    stop_dependencies
    
    # Cleanup
    cleanup_temp_files
    
    log_success "${RESOURCE_NAME} stopped"
}
```

## CLI Integration

### Required Commands
Every resource MUST provide these CLI commands:

```bash
#!/bin/bash
# cli.sh

case "$1" in
    status)
        # Show current status
        show_status
        ;;
    
    health)
        # Check health
        check_health && echo "Healthy" || echo "Unhealthy"
        ;;
    
    logs)
        # Show logs
        tail -f "${LOG_FILE:-/var/log/${RESOURCE_NAME}.log}"
        ;;
    
    content)
        # Content management
        case "$2" in
            list)
                list_content "$3"
                ;;
            add)
                add_content "$3" "$4"
                ;;
            remove)
                remove_content "$3"
                ;;
            *)
                echo "Usage: $0 content {list|add|remove}"
                ;;
        esac
        ;;
    
    help)
        # Show help
        cat <<EOF
${RESOURCE_NAME} CLI Commands:
  status              Show current status
  health              Check health
  logs                Show logs
  content list        List content
  content add <item>  Add content
  content remove <id> Remove content
  help                Show this help
EOF
        ;;
    
    *)
        echo "Usage: $0 {status|health|logs|content|help}"
        exit 1
        ;;
esac
```

## Content Management

### Required Pattern
```bash
#!/bin/bash
# lib/content.sh

# List content based on type
list_content() {
    local content_type="${1:-all}"
    
    case "$content_type" in
        models)
            list_models
            ;;
        workflows)
            list_workflows
            ;;
        configs)
            list_configs
            ;;
        all)
            echo "=== Models ==="
            list_models
            echo "=== Workflows ==="
            list_workflows
            echo "=== Configs ==="
            list_configs
            ;;
        *)
            echo "Unknown content type: $content_type"
            return 1
            ;;
    esac
}

# Add content
add_content() {
    local content_type="$1"
    local content_path="$2"
    
    if [ ! -f "$content_path" ]; then
        echo "File not found: $content_path"
        return 1
    fi
    
    case "$content_type" in
        model)
            add_model "$content_path"
            ;;
        workflow)
            add_workflow "$content_path"
            ;;
        config)
            add_config "$content_path"
            ;;
        *)
            echo "Unknown content type: $content_type"
            return 1
            ;;
    esac
}

# Remove content
remove_content() {
    local content_id="$1"
    
    if [ -z "$content_id" ]; then
        echo "Content ID required"
        return 1
    fi
    
    # Find and remove content
    if remove_by_id "$content_id"; then
        echo "Content removed: $content_id"
        return 0
    else
        echo "Failed to remove: $content_id"
        return 1
    fi
}
```

## Error Handling

### Required Pattern
```bash
# Proper error handling with cleanup
execute_with_cleanup() {
    local cleanup_needed=false
    
    # Set trap for cleanup
    trap 'cleanup_on_error' ERR
    
    # Execute main logic
    main_logic || {
        local exit_code=$?
        log_error "Operation failed with code $exit_code"
        cleanup_on_error
        return $exit_code
    }
    
    # Success
    trap - ERR
    return 0
}

cleanup_on_error() {
    log_warn "Cleaning up after error..."
    
    # Stop any started services
    stop_services
    
    # Remove temporary files
    rm -f /tmp/${RESOURCE_NAME}-*
    
    # Reset state
    reset_state
}
```

## Logging Standards

### Required Pattern
```bash
# Consistent logging functions
log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $*" | tee -a "${LOG_FILE}"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" | tee -a "${LOG_FILE}" >&2
}

log_warn() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] WARN: $*" | tee -a "${LOG_FILE}"
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✅ $*" | tee -a "${LOG_FILE}"
}

log_debug() {
    if [ "${DEBUG:-false}" = "true" ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] DEBUG: $*" | tee -a "${LOG_FILE}"
    fi
}
```

## Port Management

### Required Implementation
```bash
# Get port with fallback
get_port() {
    local env_var="${1}"
    local default_port="${2}"
    
    # Check environment variable
    local port="${!env_var}"
    
    # Check if port is available
    if [ -n "$port" ] && ! nc -z localhost "$port" 2>/dev/null; then
        echo "$port"
        return 0
    fi
    
    # Use default if available
    if [ -n "$default_port" ] && ! nc -z localhost "$default_port" 2>/dev/null; then
        echo "$default_port"
        return 0
    fi
    
    # Find next available port
    find_available_port 20000 40000
}

find_available_port() {
    local start="${1:-20000}"
    local end="${2:-40000}"
    
    for port in $(seq $start $end); do
        if ! nc -z localhost "$port" 2>/dev/null; then
            echo "$port"
            return 0
        fi
    done
    
    log_error "No available ports in range $start-$end"
    return 1
}
```

## Docker Support

### Optional but Recommended
```dockerfile
# Dockerfile
FROM alpine:latest

# Install dependencies
RUN apk add --no-cache bash curl jq

# Copy resource files
COPY . /app/

# Set working directory
WORKDIR /app

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD ./lib/health.sh || exit 1

# Run resource
CMD ["./manage.sh", "develop"]
```

## Testing Requirements

### Integration Test Template
```bash
#!/usr/bin/env bats
# test/integration.bats

setup() {
    export RESOURCE_NAME="test-resource"
    export RESOURCE_PORT="29999"
}

@test "health check responds" {
    run bash lib/health.sh
    [ "$status" -eq 0 ]
}

@test "CLI status command works" {
    run bash cli.sh status
    [ "$status" -eq 0 ]
}

@test "content can be listed" {
    run bash cli.sh content list
    [ "$status" -eq 0 ]
}

@test "lifecycle hooks work" {
    run bash manage.sh setup
    [ "$status" -eq 0 ]
    
    run bash manage.sh stop
    [ "$status" -eq 0 ]
}
```

## Documentation Requirements

### README.md Template
```markdown
# Resource Name

## Overview
[What this resource does]

## Features
- [Feature 1]
- [Feature 2]

## Requirements
- [Requirement 1]
- [Requirement 2]

## Installation
\`\`\`bash
./manage.sh setup
\`\`\`

## Usage
\`\`\`bash
# Start the resource
./manage.sh develop

# Check health
./cli.sh health

# View logs
./cli.sh logs
\`\`\`

## Configuration
[Environment variables and configuration options]

## CLI Commands
[List all available CLI commands]

## API Reference
[If applicable, document API endpoints]

## Troubleshooting
[Common issues and solutions]

## Integration Examples
[How to integrate with other resources]

## Lessons Learned
[Patterns and insights for Qdrant memory]
```

## Compliance Checklist

Before considering a resource v2.0 compliant:

- [ ] Directory structure matches template
- [ ] Health checks implemented with timeout
- [ ] All lifecycle hooks present and functional
- [ ] CLI provides all required commands
- [ ] Content management implemented
- [ ] Error handling with cleanup
- [ ] Logging follows standards
- [ ] Port management implemented
- [ ] Integration tests present
- [ ] README.md complete
- [ ] PRD.md present with requirements

## Common Violations

### ❌ Missing Health Timeout
```bash
# BAD
curl http://localhost:$PORT/health

# GOOD
timeout 5 curl -sf http://localhost:$PORT/health
```

### ❌ No Error Cleanup
```bash
# BAD
start_service || return 1

# GOOD
start_service || {
    cleanup_on_error
    return 1
}
```

### ❌ Incomplete CLI
```bash
# BAD - Missing required commands
case "$1" in
    status) show_status ;;
esac

# GOOD - All required commands
case "$1" in
    status|health|logs|content|help) ... ;;
esac
```

## Remember

- **v2.0 is mandatory** - No resource works without it
- **Health checks save debugging time** - Make them robust
- **CLI consistency matters** - Users expect standard commands
- **Document for memory** - Your patterns help future resources
- **Test everything** - Broken resources break scenarios

Every resource following v2.0 standards integrates seamlessly, debugs easily, and scales reliably.