#!/usr/bin/env bash
# Windmill Test Helpers
# Common functions and utilities for testing Windmill resource functionality

# Set up test environment
setup_windmill_test() {
    # Set test environment variables
    export WINDMILL_PROJECT_NAME="windmill-test"
    export WINDMILL_SERVER_PORT="15681"  # Different port for testing
    export WINDMILL_WORKER_REPLICAS="1"
    export WINDMILL_ENABLE_LSP="no"
    export WINDMILL_ENABLE_MULTIPLAYER="no"
    export WINDMILL_DISABLE_NATIVE_WORKER="yes"
    export WINDMILL_LOG_LEVEL="debug"
    export YES="yes"  # Auto-confirm for tests
    
    # Set test-specific directories
    export WINDMILL_DATA_DIR="/tmp/windmill-test-data"
    export WINDMILL_BACKUP_DIR="/tmp/windmill-test-backup"
    export WINDMILL_ENV_FILE="/tmp/windmill-test.env"
    
    # Generate test credentials
    export WINDMILL_SUPERADMIN_EMAIL="test@windmill.test"
    export WINDMILL_SUPERADMIN_PASSWORD="test-password-123"
    export WINDMILL_JWT_SECRET="test-jwt-secret-1234567890abcdef"
    export WINDMILL_DB_PASSWORD="test-db-password-123"
    
    # Create test directories
    mkdir -p "$WINDMILL_DATA_DIR" "$WINDMILL_BACKUP_DIR"
    
    # Source the configuration to apply test overrides
    if [[ -f "$SCRIPT_DIR/config/defaults.sh" ]]; then
        source "$SCRIPT_DIR/config/defaults.sh"
        windmill::export_config
    fi
}

# Clean up test environment
teardown_windmill_test() {
    # Stop any test containers
    if docker ps --format "{{.Names}}" | grep -q "windmill-test"; then
        docker stop $(docker ps --format "{{.Names}}" | grep "windmill-test") 2>/dev/null || true
        docker rm $(docker ps -a --format "{{.Names}}" | grep "windmill-test") 2>/dev/null || true
    fi
    
    # Remove test volumes
    docker volume ls --format "{{.Name}}" | grep "windmill-test" | xargs docker volume rm 2>/dev/null || true
    
    # Remove test networks
    docker network ls --format "{{.Name}}" | grep "windmill-test" | xargs docker network rm 2>/dev/null || true
    
    # Clean up test directories
    rm -rf "$WINDMILL_DATA_DIR" "$WINDMILL_BACKUP_DIR" "$WINDMILL_ENV_FILE" 2>/dev/null || true
    
    # Clean up any test artifacts
    rm -f /tmp/windmill-test* 2>/dev/null || true
}

# Wait for test service to be ready
wait_for_test_service() {
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -s -f "http://localhost:$WINDMILL_SERVER_PORT/api/version" >/dev/null 2>&1; then
            return 0
        fi
        
        sleep 2
        attempt=$((attempt + 1))
    done
    
    return 1
}

# Create a test environment file
create_test_env_file() {
    cat > "$WINDMILL_ENV_FILE" << EOF
WINDMILL_PROJECT_NAME=$WINDMILL_PROJECT_NAME
WINDMILL_SERVER_PORT=$WINDMILL_SERVER_PORT
WINDMILL_WORKER_REPLICAS=$WINDMILL_WORKER_REPLICAS
WINDMILL_WORKER_MEMORY_LIMIT=512M
WINDMILL_SUPERADMIN_EMAIL=$WINDMILL_SUPERADMIN_EMAIL
WINDMILL_SUPERADMIN_PASSWORD=$WINDMILL_SUPERADMIN_PASSWORD
WINDMILL_JWT_SECRET=$WINDMILL_JWT_SECRET
WINDMILL_DB_PASSWORD=$WINDMILL_DB_PASSWORD
WINDMILL_LOG_LEVEL=$WINDMILL_LOG_LEVEL
WINDMILL_ENABLE_LSP=$WINDMILL_ENABLE_LSP
WINDMILL_ENABLE_MULTIPLAYER=$WINDMILL_ENABLE_MULTIPLAYER
COMPOSE_PROFILES=internal-db,workers
EOF
}

# Assert that a service is running
assert_service_running() {
    local service_name="$1"
    
    if ! docker ps --format "{{.Names}}" | grep -q "$service_name"; then
        echo "ASSERTION FAILED: Service $service_name is not running"
        return 1
    fi
    
    return 0
}

# Assert that a port is listening
assert_port_listening() {
    local port="$1"
    
    if ! nc -z localhost "$port" 2>/dev/null; then
        echo "ASSERTION FAILED: Port $port is not listening"
        return 1
    fi
    
    return 0
}

# Assert that API is responding
assert_api_responding() {
    local base_url="$1"
    local endpoint="${2:-/api/version}"
    
    if ! curl -s -f "$base_url$endpoint" >/dev/null 2>&1; then
        echo "ASSERTION FAILED: API at $base_url$endpoint is not responding"
        return 1
    fi
    
    return 0
}

# Create mock API responses for testing
create_mock_responses() {
    local mock_dir="$BATS_TEST_DIRNAME/fixtures/mock-responses"
    mkdir -p "$mock_dir"
    
    # Mock version response
    cat > "$mock_dir/version.json" << 'EOF'
{
  "version": "1.100.0",
  "build": "test-build"
}
EOF
    
    # Mock health response
    cat > "$mock_dir/health.json" << 'EOF'
{
  "status": "healthy",
  "timestamp": "2025-01-26T12:00:00Z"
}
EOF
    
    # Mock workspaces response
    cat > "$mock_dir/workspaces.json" << 'EOF'
[
  {
    "id": "test-workspace",
    "name": "Test Workspace",
    "owner": "test@windmill.test"
  }
]
EOF
}

# Generate test Docker Compose file
create_test_compose_file() {
    local compose_file="$1"
    
    cat > "$compose_file" << EOF
version: '3.8'

services:
  windmill-db:
    image: postgres:16
    container_name: "\${WINDMILL_PROJECT_NAME:-windmill-test}-db"
    environment:
      POSTGRES_PASSWORD: \${WINDMILL_DB_PASSWORD}
      POSTGRES_DB: windmill
      POSTGRES_USER: postgres
    volumes:
      - windmill_test_db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d windmill"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - windmill-test-network

  windmill-server:
    image: ghcr.io/windmill-labs/windmill:main
    container_name: "\${WINDMILL_PROJECT_NAME:-windmill-test}-server"
    ports:
      - "\${WINDMILL_SERVER_PORT:-15681}:8000"
    environment:
      - DATABASE_URL=postgresql://postgres:\${WINDMILL_DB_PASSWORD}@windmill-db:5432/windmill
      - MODE=server
      - RUST_LOG=\${WINDMILL_LOG_LEVEL:-debug}
      - JWT_SECRET=\${WINDMILL_JWT_SECRET}
    depends_on:
      windmill-db:
        condition: service_healthy
    networks:
      - windmill-test-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/api/version"]
      interval: 10s
      timeout: 5s
      retries: 3

volumes:
  windmill_test_db_data:

networks:
  windmill-test-network:
    driver: bridge
EOF
}

# Test configuration validation
test_config_validation() {
    local expected_port="$1"
    local expected_workers="$2"
    
    [[ "$WINDMILL_SERVER_PORT" == "$expected_port" ]] || return 1
    [[ "$WINDMILL_WORKER_REPLICAS" == "$expected_workers" ]] || return 1
    
    return 0
}

# Check if test prerequisites are met
check_test_prerequisites() {
    # Check Docker availability
    if ! command -v docker >/dev/null 2>&1; then
        echo "Docker is required for testing"
        return 1
    fi
    
    # Check Docker Compose availability
    if ! docker compose version >/dev/null 2>&1; then
        echo "Docker Compose is required for testing"
        return 1
    fi
    
    # Check available ports
    if nc -z localhost "$WINDMILL_SERVER_PORT" 2>/dev/null; then
        echo "Test port $WINDMILL_SERVER_PORT is already in use"
        return 1
    fi
    
    return 0
}

# Helper function to run command with timeout
run_with_timeout() {
    local timeout="$1"
    shift
    
    timeout "$timeout" "$@"
}

# Generate random test data
generate_test_data() {
    local type="$1"
    
    case "$type" in
        "csv")
            echo "name,age,city"
            echo "Alice,30,New York"
            echo "Bob,25,San Francisco"
            echo "Charlie,35,Chicago"
            ;;
        "json")
            echo '{"users":[{"name":"Alice","age":30},{"name":"Bob","age":25}]}'
            ;;
        "webhook")
            echo '{"event_type":"user_signup","data":{"user_id":"test_123","email":"test@example.com","timestamp":"2025-01-26T12:00:00Z"}}'
            ;;
        *)
            echo "Unknown test data type: $type"
            return 1
            ;;
    esac
}

# Cleanup function for individual tests
cleanup_test() {
    # Remove any test containers that might be running
    docker ps --format "{{.Names}}" | grep "windmill-test" | xargs docker stop 2>/dev/null || true
    docker ps -a --format "{{.Names}}" | grep "windmill-test" | xargs docker rm 2>/dev/null || true
    
    # Remove test files
    rm -f /tmp/windmill-test* 2>/dev/null || true
}

# Common setup that should be called at the beginning of each test file
common_test_setup() {
    # Load test environment
    setup_windmill_test
    
    # Check prerequisites
    check_test_prerequisites || skip "Test prerequisites not met"
    
    # Create mock responses
    create_mock_responses
}

# Common teardown that should be called at the end of each test file
common_test_teardown() {
    # Cleanup test environment
    teardown_windmill_test
    
    # Remove any leftover artifacts
    cleanup_test
}