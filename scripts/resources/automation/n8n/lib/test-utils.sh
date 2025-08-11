#!/usr/bin/env bash
# n8n Test Utilities
# Shared utilities for BATS test files to reduce duplication and improve consistency

#######################################
# Standard Docker mocking for tests
#######################################
n8n::mock_docker_healthy() {
    # Mock docker commands for healthy n8n container
    docker() {
        case "$1 $2" in
            "ps --format")
                echo "$N8N_CONTAINER_NAME"
                ;;
            "ps -a --format")
                echo "$N8N_CONTAINER_NAME"
                ;;
            "exec $N8N_CONTAINER_NAME env")
                echo "N8N_BASIC_AUTH_ACTIVE=true"
                echo "N8N_BASIC_AUTH_USER=admin"
                ;;
            "logs $N8N_CONTAINER_NAME --tail")
                echo "n8n ready"
                ;;
            "stats $N8N_CONTAINER_NAME --no-stream")
                echo "CPU: 2.5% | Memory: 150MB"
                ;;
            "inspect $N8N_CONTAINER_NAME --format"*)
                if [[ "$*" =~ "N8N_BASIC_AUTH_ACTIVE" ]]; then
                    echo "true"
                elif [[ "$*" =~ "N8N_BASIC_AUTH_USER" ]]; then
                    echo "admin"
                fi
                ;;
            *)
                return 0  # Default success for other docker commands
                ;;
        esac
    }
}

#######################################
# Mock Docker for unhealthy container
#######################################
n8n::mock_docker_unhealthy() {
    docker() {
        case "$1" in
            "ps")
                if [[ "$*" =~ "--format" ]] && [[ ! "$*" =~ "-a" ]]; then
                    # Container not running
                    return 1
                else
                    # Container exists but not running
                    echo "$N8N_CONTAINER_NAME"
                fi
                ;;
            *)
                return 1  # Default failure
                ;;
        esac
    }
}

#######################################
# Mock API responses for testing
# Args: $1 - response_type (healthy|degraded|unhealthy)
#######################################
n8n::mock_api_responses() {
    local response_type="$1"
    curl() {
        local url=""
        local api_key=""
        # Parse curl arguments
        while [[ $# -gt 0 ]]; do
            case "$1" in
                -H)
                    if [[ "$2" =~ X-N8N-API-KEY:.*([a-zA-Z0-9]+) ]]; then
                        api_key="present"
                    fi
                    shift 2
                    ;;
                http*)
                    url="$1"
                    shift
                    ;;
                *)
                    shift
                    ;;
            esac
        done
        case "$response_type" in
            "healthy")
                if [[ "$url" =~ "/healthz" ]]; then
                    echo '{"status":"ok"}'
                    return 0
                elif [[ "$url" =~ "/api/v1/workflows" ]]; then
                    if [[ -n "$api_key" ]]; then
                        echo '{"data":[{"id":1,"name":"test"}]}'
                        return 0
                    else
                        echo '{"message":"Unauthorized"}'
                        return 1
                    fi
                fi
                ;;
            "degraded")
                if [[ "$url" =~ "/healthz" ]]; then
                    echo '{"status":"ok"}'
                    return 0
                elif [[ "$url" =~ "/api/v1/workflows" ]]; then
                    echo '{"message":"Unauthorized"}'
                    return 1
                fi
                ;;
            "unhealthy")
                return 1  # All API calls fail
                ;;
        esac
        return 1
    }
}

#######################################
# Setup standard test environment
# Args: $1 - test_type (healthy|degraded|unhealthy)
#######################################
n8n::setup_test_environment() {
    local test_type="${1:-healthy}"
    # Export test variables
    export N8N_CONTAINER_NAME="n8n-test"
    export N8N_PORT="5678"
    export N8N_BASE_URL="http://localhost:5678"
    export N8N_DATA_DIR="/tmp/n8n-test"
    export N8N_API_KEY=""
    # Create mock data directory
    mkdir -p "$N8N_DATA_DIR"
    # Mock system commands
    system::is_command() {
        case "$1" in
            docker|curl|jq)
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    # Setup mocks based on test type
    case "$test_type" in
        "healthy")
            export N8N_API_KEY="test-api-key"
            n8n::mock_docker_healthy
            n8n::mock_api_responses "healthy"
            ;;
        "degraded")
            n8n::mock_docker_healthy
            n8n::mock_api_responses "degraded"
            ;;
        "unhealthy")
            n8n::mock_docker_unhealthy
            n8n::mock_api_responses "unhealthy"
            ;;
    esac
    # Mock secrets resolution
    secrets::resolve() {
        case "$1" in
            "N8N_API_KEY")
                echo "${N8N_API_KEY:-}"
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    # Mock logging functions
    log::info() { echo "INFO: $*" >&2; }
    log::warn() { echo "WARN: $*" >&2; }
    log::error() { echo "ERROR: $*" >&2; }
    log::success() { echo "SUCCESS: $*" >&2; }
    log::header() { echo "HEADER: $*" >&2; }
}

#######################################
# Cleanup test environment
#######################################
n8n::cleanup_test_environment() {
    # Remove test directories
    [[ -n "${N8N_DATA_DIR:-}" ]] && rm -rf "$N8N_DATA_DIR" 2>/dev/null || true
    # Unset test variables
    unset N8N_CONTAINER_NAME N8N_PORT N8N_BASE_URL N8N_DATA_DIR N8N_API_KEY
    # Restore original functions (if they were overridden)
    unset -f docker curl system::is_command secrets::resolve
    unset -f log::info log::warn log::error log::success log::header
}

#######################################
# Create test fixtures for various scenarios
# Args: $1 - fixture_type
#######################################
n8n::create_test_fixture() {
    local fixture_type="$1"
    case "$fixture_type" in
        "database")
            # Create a mock database file
            touch "$N8N_DATA_DIR/database.sqlite"
            echo "SQLite database" > "$N8N_DATA_DIR/database.sqlite"
            ;;
        "corrupted_database")
            # Create a corrupted database
            mkdir -p "$N8N_DATA_DIR"
            echo "corrupted" > "$N8N_DATA_DIR/database.sqlite"
            chmod 000 "$N8N_DATA_DIR/database.sqlite"
            ;;
        "backup")
            # Create a backup directory
            local backup_dir="${HOME}/n8n-backup-$(date +%Y%m%d-%H%M%S)"
            mkdir -p "$backup_dir"
            echo "SQLite backup" > "$backup_dir/database.sqlite"
            echo "$backup_dir"
            ;;
        "config_file")
            # Create a test config file
            mkdir -p "$(dirname "$N8N_DATA_DIR")"
            cat > "$N8N_DATA_DIR/config.json" << 'EOF'
{
  "database": {
    "type": "sqlite",
    "sqlite": {
      "database": "database.sqlite"
    }
  }
}
EOF
            ;;
    esac
}

#######################################
# Assert that a function produces expected output
# Args: $1 - function_name, $2 - expected_pattern, $3 - expected_exit_code (optional)
#######################################
n8n::assert_function_behavior() {
    local function_name="$1"
    local expected_pattern="$2"
    local expected_exit_code="${3:-0}"
    local output
    local actual_exit_code
    # Capture function output and exit code
    if output=$($function_name 2>&1); then
        actual_exit_code=0
    else
        actual_exit_code=$?
    fi
    # Check exit code
    if [[ "$actual_exit_code" != "$expected_exit_code" ]]; then
        echo "Expected exit code $expected_exit_code, got $actual_exit_code" >&2
        return 1
    fi
    # Check output pattern
    if [[ -n "$expected_pattern" ]] && ! echo "$output" | grep -q "$expected_pattern"; then
        echo "Expected pattern '$expected_pattern' not found in output: $output" >&2
        return 1
    fi
    return 0
}

#######################################
# Mock network connectivity tests
# Args: $1 - connectivity_type (online|offline)
#######################################
n8n::mock_network_connectivity() {
    local connectivity_type="$1"
    case "$connectivity_type" in
        "online")
            ping() {
                return 0
            }
            nc() {
                if [[ "$*" =~ "-z" ]]; then
                    return 0  # Port is open
                fi
                return 0
            }
            ;;
        "offline")
            ping() {
                return 1
            }
            nc() {
                return 1  # All ports closed
            }
            ;;
    esac
}

#######################################
# Generate test data for API responses
# Args: $1 - data_type
#######################################
n8n::generate_test_data() {
    local data_type="$1"
    case "$data_type" in
        "workflow")
            cat << 'EOF'
{
  "id": 1,
  "name": "Test Workflow",
  "active": true,
  "nodes": [
    {
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [240, 300]
    }
  ],
  "connections": {},
  "createdAt": "2024-01-01T00:00:00.000Z",
  "updatedAt": "2024-01-01T00:00:00.000Z"
}
EOF
            ;;
        "api_error")
            cat << 'EOF'
{
  "message": "Invalid API key",
  "code": 401
}
EOF
            ;;
        "health_response")
            cat << 'EOF'
{
  "status": "ok"
}
EOF
            ;;
    esac
}
