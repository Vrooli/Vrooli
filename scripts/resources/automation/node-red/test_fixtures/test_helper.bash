#!/usr/bin/env bash
# Test helper for Node-RED management script tests
# Provides common utilities, mocks, and fixtures for bats tests

# Test environment setup
export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(dirname "${BASH_SOURCE[0]}")}"
export NODE_RED_ROOT_DIR="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"

# Debug: ensure correct path resolution
if [[ ! -d "$NODE_RED_ROOT_DIR/config" ]]; then
    # Try different path resolution
    export NODE_RED_ROOT_DIR="/home/matthalloran8/Vrooli/scripts/resources/automation/node-red"
fi
export NODE_RED_TEST_DIR="$(mktemp -d)"
# Point to the actual project root .vrooli directory
export NODE_RED_TEST_CONFIG_DIR="/home/matthalloran8/Vrooli/.vrooli"

# Setup test environment without conflicting readonly variables
# These will be set by the config files being tested

# Test script directory with proper path resolution
export SCRIPT_DIR="$NODE_RED_TEST_DIR"
# RESOURCES_DIR will be set properly in setup_test_environment

# Create mock directory structure
setup_test_environment() {
    mkdir -p "$NODE_RED_TEST_DIR"/{config,lib,docker,flows,nodes}
    # Don't create NODE_RED_TEST_CONFIG_DIR since it points to actual project .vrooli
    # mkdir -p "$NODE_RED_TEST_CONFIG_DIR"
    
    # Set up proper RESOURCES_DIR after test directory is created
    local test_parent="$(dirname "$NODE_RED_TEST_DIR")"
    export RESOURCES_DIR="$test_parent/resources"
    mkdir -p "$RESOURCES_DIR"
    
    # Copy actual configuration files to test directory and modify for testing
    cp -r "$NODE_RED_ROOT_DIR/config"/* "$NODE_RED_TEST_DIR/config/"
    cp -r "$NODE_RED_ROOT_DIR/lib"/* "$NODE_RED_TEST_DIR/lib/"
    
    # Remove readonly declarations from config files for testing
    if [[ -f "$NODE_RED_TEST_DIR/config/defaults.sh" ]]; then
        sed -i 's/readonly /export /g' "$NODE_RED_TEST_DIR/config/defaults.sh"
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

# Mock additional functions that might be called
update_resource_config() { return 0; }
import_flow() { return 0; }

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

# Ensure consistent container name
export CONTAINER_NAME="${CONTAINER_NAME:-node-red}"
export RESOURCE_PORT="${RESOURCE_PORT:-1880}"
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
            --build-image) ARGS_VALUES["build-image"]="$2"; shift 2 ;;
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

    # Create minimal settings.js file for Node-RED
    cat > "$NODE_RED_TEST_DIR/settings.js" << 'EOF'
module.exports = {
    httpAdminRoot: '/admin',
    httpNodeRoot: '/api',
    userDir: '/data',
    flowFile: 'flows.json',
    functionGlobalContext: {},
    debugMaxLength: 1000,
    adminAuth: {
        type: "credentials",
        users: [{
            username: "admin",
            password: "$2a$08$zZWtXTja0fB1pzD4sHCMyOCMYz2Z6dNbM6tl8sJogENOMcxWV9DN.",
            permissions: "*"
        }]
    }
};
EOF

    # Create basic flows.json
    echo '[]' > "$NODE_RED_TEST_DIR/flows/flows.json"
}

# Mock Docker commands
mock_docker() {
    export DOCKER_MOCK_MODE="${1:-success}"
    
    # Create docker mock function
    docker() {
        # Debug: uncomment to see what arguments are passed
        # echo "DEBUG: docker called with: $*" >&2
        case "$DOCKER_MOCK_MODE" in
            "success")
                case "$1" in
                    "container")
                        case "$2" in
                            "inspect")
                                if [[ "$3" == "-f" && "$4" == "{{.State.Running}}" ]]; then
                                    echo "true"
                                elif [[ "$3" == "-f" && "$4" == "{{.State.Status}}" ]]; then
                                    echo "running"
                                elif [[ "$3" == "--format={{.State.Health.Status}}" ]] || [[ "$3" == "--format='{{.State.Health.Status}}'" ]]; then
                                    echo "healthy"
                                elif [[ "$3" == "node-red" ]] || [[ "$4" == "node-red" ]] || [[ "$5" == "node-red" ]]; then
                                    # Mock container inspection output
                                    echo '{"State": {"Running": true, "Status": "running", "Health": {"Status": "healthy"}}}'
                                else
                                    return 0
                                fi
                                ;;
                            *) return 0 ;;
                        esac
                        ;;
                    "ps") 
                        if [[ "$*" =~ "--format" ]]; then
                            echo "node-red"
                        else
                            echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS    NAMES"
                            echo "abc123def456   nodered/node-red:latest   npm start   2 hours ago   Up 2 hours   0.0.0.0:1880->1880/tcp   node-red"
                        fi
                        ;;
                    "build") return 0 ;;
                    "run") return 0 ;;
                    "start"|"stop"|"restart") return 0 ;;
                    "rm"|"rmi") return 0 ;;
                    "volume") return 0 ;;
                    "network") return 0 ;;
                    "stats") echo "CONTAINER     CPU %     MEM USAGE / LIMIT     MEM %     NET I/O         BLOCK I/O" ;;
                    "exec")
                        # Handle docker exec $CONTAINER_NAME <command>
                        if [[ "$2" == "$CONTAINER_NAME" ]] || [[ "$2" == "node-red" ]]; then
                            case "$3" in
                                "docker")
                                    if [[ "$4" == "ps" ]]; then
                                        echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS    NAMES"
                                        echo "abc123def456   test/image   cmd      1 hour ago   Up 1 hour   0.0.0.0:8080->8080/tcp   test_container"
                                        return 0
                                    else
                                        echo "Docker version 20.10.0, build 1234567"
                                        return 0
                                    fi
                                    ;;
                                "ls")
                                    case "$4" in
                                        "/workspace") 
                                            echo "flows  nodes  settings.js  package.json"
                                            return 0
                                            ;;
                                        "/data/flows.json")
                                            echo "/data/flows.json"
                                            return 0
                                            ;;
                                        *)
                                            echo "settings.js flows.json package.json"
                                            return 0
                                            ;;
                                    esac
                                    ;;
                                "/bin/sh")
                                    if [[ "$4" == "-c" && "$5" == "which ls" ]]; then
                                        echo "/bin/ls"
                                        return 0
                                    else
                                        echo "mock shell output"
                                        return 0
                                    fi
                                    ;;
                                "test") return 0 ;;
                                *) echo "mock exec output"; return 0 ;;
                            esac
                        else
                            echo "mock exec output"
                            return 0
                        fi
                        ;;
                    "inspect")
                        # Handle direct docker inspect calls
                        if [[ "$2" == "--format={{.State.Health.Status}}" ]] || [[ "$2" == "--format='{{.State.Health.Status}}'" ]]; then
                            echo "healthy"
                        elif [[ "$2" == "--format={{.State.Running}}" ]] || [[ "$2" == "-f" && "$3" == "{{.State.Running}}" ]]; then
                            echo "true"
                        elif [[ "$2" == "--format={{.State.Status}}" ]] || [[ "$2" == "-f" && "$3" == "{{.State.Status}}" ]]; then
                            echo "running"
                        else
                            # Default container info
                            echo '{"State": {"Running": true, "Status": "running", "Health": {"Status": "healthy"}}}'
                        fi
                        ;;
                    "info") return 0 ;;  # Docker daemon check
                    *) return 0 ;;
                esac
                ;;
            "not_installed")
                case "$1" in
                    "container") return 1 ;;
                    "inspect") return 1 ;;  # Container doesn't exist, so inspect fails
                    *) return 0 ;;
                esac
                ;;
            "not_running")
                case "$1" in
                    "container")
                        case "$2" in
                            "inspect")
                                if [[ "$3" == "-f" && "$4" == "{{.State.Running}}" ]]; then
                                    echo "false"
                                elif [[ "$3" == "--format={{.State.Health.Status}}" ]] || [[ "$3" == "--format='{{.State.Health.Status}}'" ]]; then
                                    echo "unhealthy"
                                elif [[ "$3" == "node-red" ]] || [[ "$4" == "node-red" ]]; then
                                    echo '{"State": {"Running": false, "Status": "exited"}}'
                                else
                                    return 0
                                fi
                                ;;
                            *) return 0 ;;
                        esac
                        ;;
                    "inspect")
                        # Handle direct docker inspect calls for not_running
                        if [[ "$2" == "--format={{.State.Health.Status}}" ]] || [[ "$2" == "--format='{{.State.Health.Status}}'" ]]; then
                            echo "unhealthy"
                        elif [[ "$2" == "--format={{.State.Running}}" ]] || [[ "$2" == "-f" && "$3" == "{{.State.Running}}" ]]; then
                            echo "false"
                        elif [[ "$2" == "--format={{.State.Status}}" ]] || [[ "$2" == "-f" && "$3" == "{{.State.Status}}" ]]; then
                            echo "exited"
                        else
                            # Default container info for stopped container
                            echo '{"State": {"Running": false, "Status": "exited", "Health": {"Status": "unhealthy"}}}'
                        fi
                        ;;
                    "ps")
                        if [[ "$*" =~ "--format" ]]; then
                            echo ""  # No running containers
                        else
                            echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS    NAMES"
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
                if [[ "$*" =~ "flows" ]]; then
                    cat "$TEST_FIXTURES_DIR/sample-flows/flows-response.json" 2>/dev/null || echo '[]'
                elif [[ "$*" =~ "settings" ]]; then
                    echo '{"version": "3.0.0", "userDir": "/data"}'
                else
                    echo "OK"
                fi
                return 0
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

# Mock jq command
mock_jq() {
    export JQ_MOCK_MODE="${1:-success}"
    
    jq() {
        case "$JQ_MOCK_MODE" in
            "success")
                case "$1" in
                    ".") cat ;;
                    "length") echo "3" ;;
                    "-r")
                        # Handle different jq queries
                        if [[ "$*" =~ "version" ]]; then
                            echo "3.0.2"
                        elif [[ "$*" =~ "flows" ]]; then
                            cat "$TEST_FIXTURES_DIR/sample-flows/flows-response.json" 2>/dev/null || echo '[]'
                        else
                            echo "mock jq output"
                        fi
                        ;;
                    *)
                        # Default behavior - pass through the input
                        cat
                        ;;
                esac
                ;;
            "failure")
                return 1
                ;;
        esac
    }
    export -f jq
}

# Mock port checking commands
mock_port_commands() {
    # Mock lsof to always show port as free
    lsof() {
        case "$*" in
            *1880*) return 1 ;;  # Port not in use
            *) return 1 ;;
        esac
    }
    export -f lsof
    
    # Mock netstat to show no conflicts
    netstat() {
        case "$*" in
            *1880*) return 1 ;;  # Port not in use 
            *) echo "Proto Recv-Q Send-Q Local Address Foreign Address State" ;;
        esac
    }
    export -f netstat
    
    # Mock ss command as well
    ss() {
        case "$*" in
            *1880*) return 1 ;;  # Port not in use
            *) echo "Netid State Recv-Q Send-Q Local Address:Port Peer Address:Port" ;;
        esac  
    }
    export -f ss
}

# Test fixtures location
export TEST_FIXTURES_DIR="$NODE_RED_ROOT_DIR/test-fixtures/fixtures"

# Create test fixtures
create_test_fixtures() {
    # No longer create fixture copies - use centralized location directly
    true
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

# Cleanup function
teardown_test_environment() {
    if [[ -n "$NODE_RED_TEST_DIR" && -d "$NODE_RED_TEST_DIR" ]]; then
        rm -rf "$NODE_RED_TEST_DIR"
    fi
    
    # Reset function exports
    unset -f docker curl jq 2>/dev/null || true
}

# Reset environment for clean testing
reset_node_red_env() {
    # Unset any previously set variables that might conflict with readonly declarations
    # Note: We cannot unset readonly variables, so this prevents initial conflicts
    true
}

# Source the actual scripts for testing
source_node_red_scripts() {
    # Set up environment - use the same corrected RESOURCES_DIR
    export SCRIPT_DIR="$NODE_RED_TEST_DIR" 
    # Reuse the corrected RESOURCES_DIR from setup
    # export RESOURCES_DIR="$NODE_RED_TEST_DIR/../.." - this was the problem
    
    # Source mock common.sh first (provides log:: functions)
    source "$RESOURCES_DIR/common.sh"
    
    # Source mock args.sh (provides args:: functions)
    source "$(dirname "$RESOURCES_DIR")/helpers/utils/args.sh"
    
    # Source configuration (only if not already sourced)
    if ! command -v node_red::export_config >/dev/null 2>&1; then
        source "$NODE_RED_TEST_DIR/config/defaults.sh"
    fi
    if ! command -v node_red::show_installing >/dev/null 2>&1; then
        source "$NODE_RED_TEST_DIR/config/messages.sh"
    fi
    
    # Export configuration
    node_red::export_config
    
    # Source library modules
    source "$NODE_RED_TEST_DIR/lib/common.sh"
    source "$NODE_RED_TEST_DIR/lib/docker.sh"
    source "$NODE_RED_TEST_DIR/lib/install.sh"
    source "$NODE_RED_TEST_DIR/lib/status.sh"
    source "$NODE_RED_TEST_DIR/lib/api.sh"
    source "$NODE_RED_TEST_DIR/lib/testing.sh"
    
    # Create timeout wrapper function for tests that need external timeout
    node_red_timeout_wrapper() {
        local timeout_duration="$1"
        local function_name="$2"
        shift 2
        local args="$*"
        
        # Create complete environment recreation script
        local env_script="
        export SCRIPT_DIR='$SCRIPT_DIR'
        export RESOURCES_DIR='$RESOURCES_DIR'
        export RESOURCE_PORT='$RESOURCE_PORT'
        export CONTAINER_NAME='$CONTAINER_NAME'
        source '$RESOURCES_DIR/common.sh'
        source '$NODE_RED_TEST_DIR/config/defaults.sh'
        source '$NODE_RED_TEST_DIR/config/messages.sh'
        node_red::export_config
        source '$NODE_RED_TEST_DIR/lib/common.sh'
        source '$NODE_RED_TEST_DIR/lib/docker.sh'
        source '$NODE_RED_TEST_DIR/lib/install.sh'
        source '$NODE_RED_TEST_DIR/lib/status.sh'
        source '$NODE_RED_TEST_DIR/lib/api.sh'
        source '$NODE_RED_TEST_DIR/lib/testing.sh'
        $function_name $args
        "
        
        timeout "$timeout_duration" bash -c "$env_script"
    }
    export -f node_red_timeout_wrapper
}

# Initialize test environment
setup_test_environment
create_test_fixtures