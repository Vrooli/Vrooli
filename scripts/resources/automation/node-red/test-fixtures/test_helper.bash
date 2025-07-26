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
export NODE_RED_TEST_CONFIG_DIR="$NODE_RED_TEST_DIR/.vrooli"

# Setup test environment without conflicting readonly variables
# These will be set by the config files being tested

# Test script directory
export SCRIPT_DIR="$NODE_RED_TEST_DIR"
export RESOURCES_DIR="$NODE_RED_TEST_DIR/../.."

# Create mock directory structure
setup_test_environment() {
    mkdir -p "$NODE_RED_TEST_DIR"/{config,lib,docker,flows,nodes}
    mkdir -p "$NODE_RED_TEST_CONFIG_DIR"
    
    # Copy actual configuration files to test directory and modify for testing
    cp -r "$NODE_RED_ROOT_DIR/config"/* "$NODE_RED_TEST_DIR/config/"
    cp -r "$NODE_RED_ROOT_DIR/lib"/* "$NODE_RED_TEST_DIR/lib/"
    
    # Remove readonly declarations from config files for testing
    if [[ -f "$NODE_RED_TEST_DIR/config/defaults.sh" ]]; then
        sed -i 's/readonly /export /g' "$NODE_RED_TEST_DIR/config/defaults.sh"
    fi
    
    # Create mock common.sh
    cat > "$NODE_RED_TEST_DIR/../common.sh" << 'EOF'
#!/usr/bin/env bash
# Mock common.sh for testing

log::info() { echo "[INFO] $*"; }
log::success() { echo "[SUCCESS] $*"; }
log::warning() { echo "[WARNING] $*"; }
log::error() { echo "[ERROR] $*" >&2; }

# Mock system functions
system::is_command() { command -v "$1" >/dev/null 2>&1; }
EOF

    # Create mock args.sh
    mkdir -p "$NODE_RED_TEST_DIR/../helpers/utils"
    cat > "$NODE_RED_TEST_DIR/../helpers/utils/args.sh" << 'EOF'
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
}

# Mock Docker commands
mock_docker() {
    export DOCKER_MOCK_MODE="${1:-success}"
    
    # Create docker mock function
    docker() {
        case "$DOCKER_MOCK_MODE" in
            "success")
                case "$1" in
                    "container"|"inspect")
                        if [[ "$3" == "$CONTAINER_NAME" ]]; then
                            echo '{"State": {"Running": true}}'
                        fi
                        ;;
                    "ps") echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS" ;;
                    "build") return 0 ;;
                    "run") return 0 ;;
                    "start"|"stop"|"restart") return 0 ;;
                    "rm"|"rmi") return 0 ;;
                    "volume") return 0 ;;
                    "network") return 0 ;;
                    "stats") echo "CONTAINER     CPU %     MEM USAGE / LIMIT" ;;
                    "exec") 
                        case "$3" in
                            "ls") echo "file1 file2" ;;
                            "docker") echo "mocked docker in container" ;;
                            *) echo "mock exec output" ;;
                        esac
                        ;;
                    *) return 0 ;;
                esac
                ;;
            "not_installed")
                case "$1" in
                    "container"|"inspect") return 1 ;;
                    *) return 0 ;;
                esac
                ;;
            "not_running")
                case "$1" in
                    "container"|"inspect")
                        if [[ "$3" == "$CONTAINER_NAME" ]]; then
                            echo '{"State": {"Running": false}}'
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
                    cat "$BATS_TEST_DIRNAME/sample-flows/flows-response.json" 2>/dev/null || echo '[]'
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
                    *) echo "mock jq output" ;;
                esac
                ;;
            "failure")
                return 1
                ;;
        esac
    }
    export -f jq
}

# Create test fixtures
create_test_fixtures() {
    # Create directories if they don't exist
    mkdir -p "$BATS_TEST_DIRNAME/sample-flows"
    mkdir -p "$BATS_TEST_DIRNAME/mock-responses"
    
    # Sample flow file (only create if it doesn't exist)
    if [[ ! -f "$BATS_TEST_DIRNAME/sample-flows/flows-response.json" ]]; then
        cat > "$BATS_TEST_DIRNAME/sample-flows/flows-response.json" << 'EOF'
[
    {
        "id": "flow1",
        "type": "tab",
        "label": "Test Flow 1",
        "disabled": false
    },
    {
        "id": "node1",
        "type": "inject",
        "name": "Test Inject",
        "z": "flow1"
    }
]
EOF
    fi

    # Sample settings file (only create if it doesn't exist)
    if [[ ! -f "$BATS_TEST_DIRNAME/test-settings.js" ]]; then
        cat > "$BATS_TEST_DIRNAME/test-settings.js" << 'EOF'
module.exports = {
    flowFile: 'flows.json',
    userDir: '/data/',
    uiPort: 1880
};
EOF
    fi

    # Mock responses (only create if they don't exist)
    if [[ ! -f "$BATS_TEST_DIRNAME/mock-responses/health.json" ]]; then
        echo '{"status": "ok"}' > "$BATS_TEST_DIRNAME/mock-responses/health.json"
    fi
    if [[ ! -f "$BATS_TEST_DIRNAME/mock-responses/settings.json" ]]; then
        echo '{"version": "3.0.0"}' > "$BATS_TEST_DIRNAME/mock-responses/settings.json"
    fi
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
    # Set up environment
    export SCRIPT_DIR="$NODE_RED_TEST_DIR" 
    export RESOURCES_DIR="$NODE_RED_TEST_DIR/../.."
    
    # Source mock common.sh first (provides log:: functions)
    source "$NODE_RED_TEST_DIR/../common.sh"
    
    # Source mock args.sh (provides args:: functions)
    source "$NODE_RED_TEST_DIR/../helpers/utils/args.sh"
    
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
}

# Initialize test environment
setup_test_environment
create_test_fixtures