# Mock Registry: Advanced Mock Management

Understanding the lazy-loaded mock system that powers the Vrooli testing infrastructure.

## 🎯 Overview

The Mock Registry is a revolutionary approach to test mocking that:
- **Loads mocks on-demand** (~60% faster startup)
- **Prevents conflicts** between different mock systems
- **Provides unified interface** for all mock types
- **Tracks usage** for debugging and optimization

## 🏗️ Architecture

```
Mock Registry
├── System Mocks          # Docker, HTTP, Commands
│   ├── docker.bash       # Complete Docker CLI mock
│   ├── http.bash         # HTTP/curl/wget mocking
│   └── commands.bash     # jq, systemctl, git, etc.
├── Resource Mocks        # Vrooli Resources
│   ├── ai/               # AI services
│   ├── automation/       # Automation tools
│   ├── agents/           # Agent services
│   ├── storage/          # Data storage
│   ├── search/           # Search engines
│   └── execution/        # Code execution
└── Registry System       # Management layer
    ├── mock_registry.bash   # Core registry
    └── Legacy fallback      # Backward compatibility
```

## 🚀 Quick Start

### Basic Mock Loading
```bash
# Load individual mocks on demand
mock::load system docker
mock::load system http
mock::load resource ollama

# Check what's loaded
mock::list_loaded

# Check if specific mock is loaded
if mock::is_loaded system docker; then
    echo "Docker mock available"
fi
```

### Pre-configured Setup Functions
```bash
# Minimal setup (fastest)
mock::setup_minimal

# Resource-specific setup
mock::setup_resource "ollama"

# Multi-resource setup
mock::setup_integration "ollama" "whisper" "n8n"
```

## 🔧 Core Functions

### `mock::load(category, name)`
Load a specific mock module on-demand.

```bash
# System mocks
mock::load system docker      # Docker CLI commands
mock::load system http        # curl, wget, nc
mock::load system commands    # jq, systemctl, git, etc.

# Resource mocks (auto-detects category)
mock::load resource ollama    # Loads from ai/ollama.bash
mock::load resource n8n       # Loads from automation/n8n.bash
mock::load resource qdrant    # Loads from storage/qdrant.bash
```

**Benefits:**
- Only loads what you need
- Prevents duplicate loading
- Automatic category detection for resources
- Fallback to legacy mocks

### `mock::setup_minimal()`
Fast setup for basic testing.

```bash
setup() {
    mock::setup_minimal
}
```

**What it loads:**
- Essential system mocks: `docker`, `http`, `commands`
- Basic environment variables
- Temporary directories
- Command tracking setup

**Performance:** ~50ms startup time

### `mock::setup_resource(resource)`
Complete setup for single resource testing.

```bash
setup() {
    mock::setup_resource "ollama"
}
```

**What it does:**
1. Calls `mock::setup_minimal()`
2. Loads resource-specific mocks
3. Configures environment variables
4. Sets up container names and ports
5. Configures test isolation

**Performance:** ~150ms startup time

### `mock::setup_integration(resources...)`
Setup for multi-resource testing.

```bash
setup() {
    mock::setup_integration "ollama" "whisper" "n8n"
}
```

**What it does:**
1. Calls `mock::setup_minimal()`
2. Loads all resource mocks
3. Configures all environments
4. Ensures port conflict avoidance
5. Sets up shared test namespace

**Performance:** ~300ms startup time

## 📁 Mock Categories

### System Mocks
Located in `/mocks/system/`

#### docker.bash
Complete Docker CLI simulation with:
- All major commands: `ps`, `run`, `exec`, `logs`, `inspect`
- Container state management
- Image management
- Network and volume operations
- Docker Compose support

```bash
# Example usage after loading
mock::load system docker

# Set container state
mock::docker::set_container_state "myapp" "running" "nginx:latest"

# Container appears in docker ps
run docker ps
assert_output_contains "myapp"
```

#### http.bash
HTTP client mocking with:
- curl and wget simulation
- Endpoint state management
- Response customization
- Resource-specific patterns

```bash
# Example usage
mock::load system http

# Set endpoint response
mock::http::set_endpoint_response "http://api.example.com/status" '{"status":"ok"}' 200

# Test HTTP call
run curl -s http://api.example.com/status
assert_json_field_equals "$output" ".status" "ok"
```

#### commands.bash
System command mocking:
- jq (JSON processing)
- systemctl (service management)
- git, ssh, openssl
- lsof, netstat, ps

```bash
# Example usage
mock::load system commands

# Set service state
mock::service::set_state "nginx" "active"

# Test service status
run systemctl status nginx
assert_output_contains "active (running)"
```

### Resource Mocks
Located in `/mocks/resources/{category}/`

#### Resource Categories

**AI Resources (`ai/`)**
- `ollama.bash` - LLM service mocking
- `whisper.bash` - Speech-to-text mocking
- `comfyui.bash` - Image generation mocking
- `unstructured-io.bash` - Document processing mocking

**Automation Resources (`automation/`)**
- `n8n.bash` - Workflow automation mocking
- `node-red.bash` - Visual programming mocking
- `huginn.bash` - Agent system mocking
- `windmill.bash` - Developer automation mocking

**Storage Resources (`storage/`)**
- `qdrant.bash` - Vector database mocking
- `minio.bash` - Object storage mocking
- `postgres.bash` - PostgreSQL mocking
- `redis.bash` - Redis cache mocking

## 🎛️ Advanced Configuration

### Custom Mock Responses

#### HTTP Endpoints
```bash
# Set specific endpoint behavior
mock::http::set_endpoint_state "http://ollama:11434" "healthy"
mock::http::set_endpoint_response "http://ollama:11434/api/tags" '{"models":[]}'

# Set response delays
mock::http::set_endpoint_delay "http://slow-api:8080" "2"

# Set response sequences for multiple calls
mock::http::set_endpoint_sequence "http://api:8080/status" "starting,running,healthy"
```

#### Container States
```bash
# Configure Docker container mock states
mock::docker::set_container_state "ollama_container" "running" "ollama/ollama:latest"
mock::docker::set_container_state "broken_container" "stopped"

# Set image availability
mock::docker::set_image_available "ollama/ollama:latest" "true"
```

#### Service States
```bash
# Configure systemctl service states
mock::service::set_state "docker" "active"
mock::service::set_state "nginx" "failed"
mock::service::set_state "postgresql" "inactive"
```

### Environment Customization

#### Resource Environment Variables
Each resource gets automatically configured environment:

```bash
# For Ollama
OLLAMA_PORT="11434"
OLLAMA_BASE_URL="http://localhost:11434"
OLLAMA_CONTAINER_NAME="test_12345_ollama"

# For Whisper  
WHISPER_PORT="8090"
WHISPER_BASE_URL="http://localhost:8090"
WHISPER_CONTAINER_NAME="test_12345_whisper"

# Universal
RESOURCE_NAME="ollama"           # Current resource
TEST_NAMESPACE="test_12345"      # Unique test ID
CONTAINER_NAME_PREFIX="test_12345_"
```

#### Custom Resource Configuration
```bash
# Override default ports
OLLAMA_PORT="9999" setup_resource_test "ollama"

# Custom base URL
OLLAMA_BASE_URL="http://custom-host:11434" setup_resource_test "ollama"
```

## 🔍 Mock Tracking and Debugging

### Command Call Tracking
Every mock command is automatically tracked:

```bash
setup() {
    setup_standard_mocks
}

@test "command tracking works" {
    # Execute commands
    docker ps
    curl -s http://example.com
    jq '.test' <<< '{"test":"value"}'
    
    # Verify tracking
    assert_file_exists "${MOCK_RESPONSES_DIR}/command_calls.log"
    assert_command_called "docker" "ps"
    assert_command_called "curl"
    assert_command_called "jq"
}
```

### Mock Usage Tracking
Track which mocks are actually used:

```bash
@test "mock usage verification" {
    setup_resource_test "ollama"
    
    # Use some mocks
    docker ps
    curl -s "$OLLAMA_BASE_URL/health"
    
    # Verify specific mocks were used
    assert_mock_used "docker"
    assert_mock_used "http"
}
```

### Debug Information
```bash
# List all loaded mocks
mock::list_loaded

# Check if specific mock is loaded
mock::is_loaded system docker    # Returns 0 if loaded

# View mock registry state
declare -p LOADED_MOCKS
```

## 🎭 Mock Modes

### HTTP Mock Modes
```bash
# Normal mode (default)
export HTTP_MOCK_MODE="normal"

# Offline mode (all requests fail)
export HTTP_MOCK_MODE="offline"

# Slow mode (applies configured delays)
export HTTP_MOCK_MODE="slow"
```

### Docker Mock Modes
```bash
# Normal mode (default)
export DOCKER_MOCK_MODE="normal"

# Offline mode (Docker daemon unavailable)
export DOCKER_MOCK_MODE="offline"

# Error mode (all commands fail)
export DOCKER_MOCK_MODE="error"
```

### Performance Mode
```bash
# Enable performance mode (faster, less realistic)
export TEST_PERFORMANCE_MODE="true"

# Effects:
# - sleep commands become instant
# - Reduced mock overhead
# - Simpler command outputs
```

## 🔧 Creating Custom Mocks

### System Mock Template
```bash
#!/usr/bin/env bash
# Custom System Mock Template

# Prevent duplicate loading
if [[ "${MY_CUSTOM_MOCKS_LOADED:-}" == "true" ]]; then
    return 0
fi
export MY_CUSTOM_MOCKS_LOADED="true"

# Mock state storage
declare -A MOCK_CUSTOM_STATE

# Mock function
my_custom_command() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "my_custom_command $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Custom logic here
    echo "Mock response"
}

# Export functions
export -f my_custom_command

echo "[CUSTOM_MOCKS] Custom system mocks loaded"
```

### Resource Mock Template
```bash
#!/usr/bin/env bash
# Custom Resource Mock Template

# Prevent duplicate loading
if [[ "${MY_RESOURCE_MOCKS_LOADED:-}" == "true" ]]; then
    return 0
fi
export MY_RESOURCE_MOCKS_LOADED="true"

# Resource-specific configuration
setup_my_resource_environment() {
    export MY_RESOURCE_PORT="${MY_RESOURCE_PORT:-8080}"
    export MY_RESOURCE_BASE_URL="http://localhost:${MY_RESOURCE_PORT}"
    export MY_RESOURCE_CONTAINER_NAME="${CONTAINER_NAME_PREFIX}my_resource"
}

# Resource health check
check_my_resource_health() {
    local port="${MY_RESOURCE_PORT:-8080}"
    local url="${MY_RESOURCE_BASE_URL:-http://localhost:$port}"
    
    assert_port_in_use "$port"
    assert_http_endpoint_reachable "$url/health"
}

# Export functions
export -f setup_my_resource_environment check_my_resource_health

echo "[MY_RESOURCE_MOCKS] My resource mocks loaded"
```

### Loading Custom Mocks
```bash
# Add to mock registry detection
mock::detect_resource_category() {
    local resource="$1"
    
    case "$resource" in
        "my-custom-resource")
            echo "custom"
            ;;
        # ... existing cases
    esac
}

# Load in tests
setup() {
    mock::load system my_custom
    mock::load resource my-custom-resource
}
```

## 🚀 Performance Optimization

### Lazy Loading Benefits
```bash
# Traditional approach (loads everything)
source all_mocks.bash  # ~500ms, 50MB memory

# Lazy loading approach (loads only what's needed)
mock::setup_minimal     # ~50ms, 5MB memory
mock::load system docker  # +20ms when first used
```

### Best Practices

1. **Load only what you need**
```bash
# Good: Minimal setup for basic tests
setup_standard_mocks

# Bad: Integration setup for simple tests
setup_integration_test "ollama" "whisper" "n8n"
```

2. **Cache expensive operations**
```bash
# Good: Parse JSON once, use multiple times
local response=$(curl -s "$API_URL/status")
assert_json_valid "$response"
assert_json_field_equals "$response" ".status" "ok"
assert_json_field_exists "$response" ".version"

# Bad: Multiple API calls
assert_json_field_equals "$(curl -s "$API_URL/status")" ".status" "ok"
assert_json_field_exists "$(curl -s "$API_URL/status")" ".version"
```

3. **Use performance mode for speed-critical tests**
```bash
setup() {
    export TEST_PERFORMANCE_MODE="true"
    setup_standard_mocks
}
```

## 🐛 Troubleshooting

### Common Issues

**"Mock not found for category:name"**
```bash
# Cause: Mock file doesn't exist or isn't in expected location
# Solution: Check file exists at correct path
ls -la /path/to/bats-fixtures/mocks/system/docker.bash

# Or check category detection
mock::detect_resource_category "my-resource"
```

**"Command call tracking not available"**
```bash
# Cause: MOCK_RESPONSES_DIR not set
# Solution: Ensure setup function calls mock setup
setup() {
    setup_standard_mocks  # This sets MOCK_RESPONSES_DIR
}
```

**"Function already exists" errors**
```bash
# Cause: Mock loaded multiple times
# Solution: Mocks have duplicate loading protection, but check:
declare -f mock_function_name  # See if already loaded
```

### Debug Mode
```bash
# Enable verbose mock loading
export MOCK_DEBUG="true"

# Enable command tracking debug
export MOCK_TRACE_COMMANDS="true"

# Enable performance timing
export MOCK_PERFORMANCE_TIMING="true"
```

## 📚 Next Steps

- **Try examples**: Start with [basic examples](examples/)
- **Learn setup modes**: Review [Setup Guide](setup-guide.md)
- **Resource patterns**: Explore [Resource Testing](resource-testing.md)
- **Debug issues**: Check [Troubleshooting](troubleshooting.md)