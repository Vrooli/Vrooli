#!/usr/bin/env bash
# Test helper for Huginn management script tests
# Provides common utilities, mocks, and fixtures for bats tests

# Test environment setup
export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(dirname "${BASH_SOURCE[0]}")}"
export HUGINN_ROOT_DIR="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"

# Debug: ensure correct path resolution
if [[ ! -d "$HUGINN_ROOT_DIR/config" ]]; then
    # Try different path resolution
    export HUGINN_ROOT_DIR="/home/matthalloran8/Vrooli/scripts/resources/automation/huginn"
fi
export HUGINN_TEST_DIR="$(mktemp -d)"
# Point to the actual project root .vrooli directory
export HUGINN_TEST_CONFIG_DIR="/home/matthalloran8/Vrooli/.vrooli"

# Test script directory with proper path resolution
export SCRIPT_DIR="$HUGINN_TEST_DIR"

# Create mock directory structure
setup_test_environment() {
    mkdir -p "$HUGINN_TEST_DIR"/{config,lib,docker,examples,test-fixtures}
    mkdir -p "$HUGINN_TEST_DIR"/examples/{agents,scenarios,workflows}
    
    # Set up proper RESOURCES_DIR after test directory is created
    local test_parent="$(dirname "$HUGINN_TEST_DIR")"
    export RESOURCES_DIR="$test_parent/resources"
    mkdir -p "$RESOURCES_DIR"
    
    # Copy actual configuration files to test directory and modify for testing
    cp -r "$HUGINN_ROOT_DIR/config"/* "$HUGINN_TEST_DIR/config/" 2>/dev/null || true
    cp -r "$HUGINN_ROOT_DIR/lib"/* "$HUGINN_TEST_DIR/lib/" 2>/dev/null || true
    
    # Remove readonly declarations from config files for testing
    if [[ -f "$HUGINN_TEST_DIR/config/defaults.sh" ]]; then
        sed -i 's/readonly /export /g' "$HUGINN_TEST_DIR/config/defaults.sh"
    fi
    
    # Create mock common.sh
    cat > "$RESOURCES_DIR/common.sh" << 'EOF'
#!/usr/bin/env bash
# Mock common.sh for testing

log::info() { echo "[INFO] $*"; }
log::success() { echo "[SUCCESS] $*"; }
log::warning() { echo "[WARNING] $*"; }
log::error() { echo "[ERROR] $*" >&2; }

# Mock system functions
system::is_command() { command -v "$1" >/dev/null 2>&1; }

# Mock port checking functions  
check_port_in_use() { return 1; }  # Always return port not in use
is_port_in_use() { return 1; }     # Always return port not in use

# Mock flow control functions
flow::confirm() {
    case "${1:-}" in
        "y"|"Y"|"yes"|"YES") return 0 ;;
        *) return 1 ;;
    esac
}

flow::is_yes() {
    case "${1:-}" in
        "y"|"Y"|"yes"|"YES") return 0 ;;
        *) return 1 ;;
    esac
}

# Ensure consistent container names
export CONTAINER_NAME="${CONTAINER_NAME:-huginn}"
export DB_CONTAINER_NAME="${DB_CONTAINER_NAME:-huginn-postgres}"
export RESOURCE_PORT="${RESOURCE_PORT:-4111}"
EOF

    # Create mock args.sh
    local helpers_dir="$(dirname "$RESOURCES_DIR")/helpers/utils"
    mkdir -p "$helpers_dir"
    cat > "$helpers_dir/args.sh" << 'EOF'
#!/usr/bin/env bash
# Mock args.sh for testing

declare -A ARGS_VALUES
declare -a ARGS_REGISTERED

args::reset() { 
    ARGS_VALUES=()
    ARGS_REGISTERED=()
}

args::register() {
    local name=""
    while [[ $# -gt 0 ]]; do
        case $1 in
            --name) name="$2"; shift 2 ;;
            --default) ARGS_VALUES["$name"]="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    ARGS_REGISTERED+=("$name")
}

args::register_help() { args::register --name "help" --default "no"; }
args::register_yes() { args::register --name "yes" --default "no"; }

args::parse() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --action) ARGS_VALUES["action"]="$2"; shift 2 ;;
            --force) ARGS_VALUES["force"]="$2"; shift 2 ;;
            --operation) ARGS_VALUES["operation"]="$2"; shift 2 ;;
            --agent-id) ARGS_VALUES["agent-id"]="$2"; shift 2 ;;
            --scenario-id) ARGS_VALUES["scenario-id"]="$2"; shift 2 ;;
            --count) ARGS_VALUES["count"]="$2"; shift 2 ;;
            --yes) ARGS_VALUES["yes"]="yes"; shift ;;
            *) shift ;;
        esac
    done
}

args::get() {
    echo "${ARGS_VALUES[$1]:-}"
}

args::is_asking_for_help() {
    [[ "$1" == "--help" || "$1" == "-h" ]]
}

args::usage() {
    echo "Usage: $1"
}
EOF
}

# Mock Docker commands for Huginn and PostgreSQL
mock_docker() {
    export DOCKER_MOCK_MODE="${1:-success}"
    
    docker() {
        case "$DOCKER_MOCK_MODE" in
            "success")
                case "$1" in
                    "container")
                        case "$2" in
                            "inspect")
                                if [[ "$5" == "huginn" || "$5" == "$CONTAINER_NAME" ]]; then
                                    if [[ "$4" == "{{.State.Running}}" ]]; then
                                        echo "true"
                                    elif [[ "$4" == "{{.State.Status}}" ]]; then
                                        echo "running"
                                    elif [[ "$3" == "--format={{.State.Health.Status}}" ]]; then
                                        echo "healthy"
                                    else
                                        echo '{"State": {"Running": true, "Status": "running", "Health": {"Status": "healthy"}}}'
                                    fi
                                elif [[ "$5" == "huginn-postgres" || "$5" == "$DB_CONTAINER_NAME" ]]; then
                                    if [[ "$4" == "{{.State.Running}}" ]]; then
                                        echo "true"
                                    elif [[ "$4" == "{{.State.Status}}" ]]; then
                                        echo "running"
                                    else
                                        echo '{"State": {"Running": true, "Status": "running"}}'
                                    fi
                                else
                                    return 0
                                fi
                                ;;
                            *) return 0 ;;
                        esac
                        ;;
                    "ps") 
                        if [[ "$*" =~ "--format" ]]; then
                            echo -e "huginn\nhuginn-postgres"
                        else
                            echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS    NAMES"
                            echo "abc123def456   huginn/huginn:latest   rails server   2 hours ago   Up 2 hours   0.0.0.0:4111->3000/tcp   huginn"
                            echo "def456ghi789   postgres:15   postgres   2 hours ago   Up 2 hours   5432/tcp   huginn-postgres"
                        fi
                        ;;
                    "exec")
                        # Handle docker exec for Rails runner
                        if [[ "$2" == "-i" && "$3" == "$CONTAINER_NAME" ]]; then
                            # Mock Rails runner output
                            case "$DOCKER_EXEC_MOCK" in
                                "agents_list")
                                    echo "✅ [1] RSS Weather Monitor"
                                    echo "   Type: RssAgent | Events: 42"
                                    echo "   Last Run: 12/28 14:30 | Schedule: Every 1 hour"
                                    echo ""
                                    echo "✅ [2] Website Status Checker"
                                    echo "   Type: WebsiteAgent | Events: 156"
                                    echo "   Last Run: 12/28 15:00 | Schedule: Every 30 minutes"
                                    ;;
                                "agent_show")
                                    echo "🔍 Agent Details: RSS Weather Monitor"
                                    echo "=================================================="
                                    echo "ID: 1"
                                    echo "Type: Agents::RssAgent"
                                    echo "Owner: admin"
                                    echo "Created: 2025-12-01 10:00"
                                    echo "Events Generated: 42"
                                    ;;
                                "scenarios_list")
                                    echo "📁 [1] Weather Monitoring Suite"
                                    echo "   Description: Complete weather monitoring workflow"
                                    echo "   Agents: 3"
                                    echo "   Created: 2025-12-01"
                                    ;;
                                "events_recent")
                                    echo "📄 12/28 15:00 - RSS Weather Monitor:"
                                    echo "   {\"title\": \"Weather Update\", \"temperature\": \"72F\"}"
                                    ;;
                                "db_check")
                                    echo "DB_OK"
                                    ;;
                                "system_stats")
                                    echo '{"users":1,"agents":5,"scenarios":2,"events":248,"links":7,"active_agents":3,"recent_events":15}'
                                    ;;
                                "version")
                                    echo "Huginn Latest on Rails 7.0.0"
                                    ;;
                                *)
                                    echo "mock rails runner output"
                                    ;;
                            esac
                        elif [[ "$2" == "$CONTAINER_NAME" ]]; then
                            echo "mock exec output"
                        fi
                        return 0
                        ;;
                    "logs")
                        echo "2025-12-28 15:00:00 INFO  Huginn started successfully"
                        echo "2025-12-28 15:00:01 INFO  Database connected"
                        echo "2025-12-28 15:00:02 INFO  Listening on port 3000"
                        return 0
                        ;;
                    "build"|"run"|"start"|"stop"|"restart"|"rm"|"rmi"|"volume"|"network") 
                        return 0 
                        ;;
                    "info") 
                        return 0 
                        ;;
                    *) return 0 ;;
                esac
                ;;
            "not_installed")
                case "$1" in
                    "ps")
                        if [[ "$*" =~ "--format" ]]; then
                            echo ""  # No containers
                        else
                            echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS    NAMES"
                        fi
                        ;;
                    "container")
                        return 1
                        ;;
                    *) return 1 ;;
                esac
                ;;
            "not_running")
                case "$1" in
                    "container")
                        case "$2" in
                            "inspect")
                                if [[ "$4" == "{{.State.Running}}" ]]; then
                                    echo "false"
                                elif [[ "$3" == "--format={{.State.Health.Status}}" ]]; then
                                    echo "unhealthy"
                                else
                                    echo '{"State": {"Running": false, "Status": "exited"}}'
                                fi
                                ;;
                            *) return 0 ;;
                        esac
                        ;;
                    "ps")
                        if [[ "$*" =~ "--format" ]]; then
                            echo -e "huginn\nhuginn-postgres"
                        else
                            echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS    NAMES"
                            echo "abc123def456   huginn/huginn:latest   rails server   2 hours ago   Exited (1)   huginn"
                            echo "def456ghi789   postgres:15   postgres   2 hours ago   Up 2 hours   huginn-postgres"
                        fi
                        ;;
                    *) return 0 ;;
                esac
                ;;
            "failure")
                return 1
                ;;
        esac
    }
    export -f docker
}

# Mock curl commands
mock_curl() {
    export CURL_MOCK_MODE="${1:-success}"
    
    curl() {
        case "$CURL_MOCK_MODE" in
            "success")
                if [[ "$*" =~ "4111" ]] || [[ "$*" =~ "localhost:3000" ]]; then
                    echo '<!DOCTYPE html><html><head><title>Huginn</title></head></html>'
                    return 0
                else
                    echo "OK"
                    return 0
                fi
                ;;
            "timeout")
                return 28  # curl timeout error code
                ;;
            "failure")
                return 1
                ;;
        esac
    }
    export -f curl
}

# Mock PostgreSQL commands
mock_pg_isready() {
    export PG_MOCK_MODE="${1:-success}"
    
    pg_isready() {
        case "$PG_MOCK_MODE" in
            "success")
                echo "localhost:5432 - accepting connections"
                return 0
                ;;
            "failure")
                echo "localhost:5432 - no response"
                return 1
                ;;
        esac
    }
    export -f pg_isready
}

# Test fixtures location
export TEST_FIXTURES_DIR="$HUGINN_ROOT_DIR/test-fixtures"

# Create test fixtures
create_test_fixtures() {
    mkdir -p "$TEST_FIXTURES_DIR/mock-responses"
    mkdir -p "$TEST_FIXTURES_DIR/sample-data"
    
    # Create mock agent response
    cat > "$TEST_FIXTURES_DIR/mock-responses/agents.json" << 'EOF'
[
  {
    "id": 1,
    "name": "RSS Weather Monitor",
    "type": "Agents::RssAgent",
    "schedule": "every_1h",
    "disabled": false,
    "events_count": 42
  },
  {
    "id": 2,
    "name": "Website Status Checker",
    "type": "Agents::WebsiteAgent",
    "schedule": "every_30m",
    "disabled": false,
    "events_count": 156
  }
]
EOF

    # Create mock scenario response
    cat > "$TEST_FIXTURES_DIR/mock-responses/scenarios.json" << 'EOF'
[
  {
    "id": 1,
    "name": "Weather Monitoring Suite",
    "description": "Complete weather monitoring workflow",
    "agents_count": 3,
    "created_at": "2025-12-01T10:00:00Z"
  }
]
EOF
}

# Test assertions
assert_success() {
    if [[ "$status" -ne 0 ]]; then
        echo "Expected success but got exit code: $status"
        echo "Output: $output"
        return 1
    fi
}

assert_failure() {
    if [[ "$status" -eq 0 ]]; then
        echo "Expected failure but got success"
        echo "Output: $output"
        return 1
    fi
}

assert_output_contains() {
    local expected="$1"
    if [[ "$output" != *"$expected"* ]]; then
        echo "Expected output to contain: $expected"
        echo "Actual output: $output"
        return 1
    fi
}

assert_output_not_contains() {
    local unexpected="$1"
    if [[ "$output" == *"$unexpected"* ]]; then
        echo "Expected output to NOT contain: $unexpected"
        echo "Actual output: $output"
        return 1
    fi
}

assert_file_exists() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        echo "Expected file to exist: $file"
        return 1
    fi
}

assert_file_not_exists() {
    local file="$1"
    if [[ -f "$file" ]]; then
        echo "Expected file to NOT exist: $file"
        return 1
    fi
}

assert_directory_exists() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        echo "Expected directory to exist: $dir"
        return 1
    fi
}

# Cleanup function
teardown_test_environment() {
    if [[ -n "$HUGINN_TEST_DIR" && -d "$HUGINN_TEST_DIR" ]]; then
        rm -rf "$HUGINN_TEST_DIR"
    fi
    
    # Reset function exports
    unset -f docker curl pg_isready 2>/dev/null || true
}

# Source the actual scripts for testing
source_huginn_scripts() {
    # Set up environment
    export SCRIPT_DIR="$HUGINN_TEST_DIR" 
    
    # Source mock common.sh first (provides log:: functions)
    source "$RESOURCES_DIR/common.sh"
    
    # Source mock args.sh (provides args:: functions)
    source "$(dirname "$RESOURCES_DIR")/helpers/utils/args.sh"
    
    # Source configuration
    if ! command -v huginn::export_config >/dev/null 2>&1; then
        source "$HUGINN_TEST_DIR/config/defaults.sh"
    fi
    if ! command -v huginn::show_installing >/dev/null 2>&1; then
        source "$HUGINN_TEST_DIR/config/messages.sh"
    fi
    
    # Export configuration
    huginn::export_config
    
    # Source library modules from the actual huginn directory
    source "$HUGINN_ROOT_DIR/lib/common.sh"
    source "$HUGINN_ROOT_DIR/lib/docker.sh"
    source "$HUGINN_ROOT_DIR/lib/install.sh"
    source "$HUGINN_ROOT_DIR/lib/status.sh"
    source "$HUGINN_ROOT_DIR/lib/api.sh"
}

# Initialize test environment
setup_test_environment
create_test_fixtures