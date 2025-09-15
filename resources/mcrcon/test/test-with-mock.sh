#!/usr/bin/env bash
# Test mcrcon with mock RCON server

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# Mock server configuration
export MOCK_RCON_HOST="127.0.0.1"
export MOCK_RCON_PORT="25576"  # Use different port to avoid conflicts
export MOCK_RCON_PASSWORD="test123"

# Configure mcrcon to use mock server
export MCRCON_HOST="$MOCK_RCON_HOST"
export MCRCON_PORT="$MOCK_RCON_PORT"
export MCRCON_PASSWORD="$MOCK_RCON_PASSWORD"

# Test counter
TESTS_RUN=0
TESTS_PASSED=0
MOCK_PID=""

# Cleanup function
cleanup() {
    if [[ -n "$MOCK_PID" ]]; then
        echo -e "${YELLOW}Stopping mock RCON server (PID: $MOCK_PID)...${NC}"
        kill "$MOCK_PID" 2>/dev/null || true
        wait "$MOCK_PID" 2>/dev/null || true
    fi
}

trap cleanup EXIT

# Start mock server
start_mock_server() {
    echo -e "${YELLOW}Starting mock RCON server...${NC}"
    python3 "${SCRIPT_DIR}/mock_rcon_server.py" > /tmp/mock_rcon.log 2>&1 &
    MOCK_PID=$!
    
    # Wait for server to start
    local count=0
    while ! nc -z "$MOCK_RCON_HOST" "$MOCK_RCON_PORT" 2>/dev/null; do
        sleep 0.5
        count=$((count + 1))
        if [[ $count -ge 20 ]]; then
            echo -e "${RED}Mock server failed to start${NC}"
            cat /tmp/mock_rcon.log
            exit 1
        fi
    done
    echo -e "${GREEN}Mock server started on ${MOCK_RCON_HOST}:${MOCK_RCON_PORT}${NC}"
}

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected="${3:-}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "  Testing $test_name... "
    
    local output
    if output=$(eval "$test_command" 2>&1); then
        if [[ -n "$expected" ]]; then
            if echo "$output" | grep -q "$expected"; then
                echo -e "${GREEN}✓${NC}"
                TESTS_PASSED=$((TESTS_PASSED + 1))
                return 0
            else
                echo -e "${RED}✗ (Expected: $expected, Got: $output)${NC}"
                return 1
            fi
        else
            echo -e "${GREEN}✓${NC}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        fi
    else
        echo -e "${RED}✗ (Command failed)${NC}"
        echo "  Output: $output"
        return 1
    fi
}

# Main test execution
main() {
    echo "=============================="
    echo "mcrcon Mock Server Tests"
    echo "=============================="
    
    # Start mock server
    start_mock_server
    
    echo ""
    echo "Testing Basic Commands:"
    echo "-----------------------"
    
    # Test connection
    run_test "Connection test" \
        "'${RESOURCE_DIR}/cli.sh' content test $MOCK_RCON_HOST $MOCK_RCON_PORT" \
        ""
    
    # Test execute command
    run_test "Execute 'list' command" \
        "'${RESOURCE_DIR}/cli.sh' content execute 'list'" \
        "players online"
    
    # Test player commands
    echo ""
    echo "Testing Player Commands:"
    echo "------------------------"
    
    run_test "List players" \
        "'${RESOURCE_DIR}/cli.sh' content player list" \
        ""
    
    run_test "Player info" \
        "'${RESOURCE_DIR}/cli.sh' content player info Steve" \
        ""
    
    run_test "Teleport player" \
        "'${RESOURCE_DIR}/cli.sh' content player teleport Steve Alex" \
        ""
    
    run_test "Give item to player" \
        "'${RESOURCE_DIR}/cli.sh' content player give Steve diamond 64" \
        ""
    
    # Test world commands
    echo ""
    echo "Testing World Commands:"
    echo "-----------------------"
    
    run_test "Save world" \
        "'${RESOURCE_DIR}/cli.sh' content world save" \
        ""
    
    run_test "Get world info" \
        "'${RESOURCE_DIR}/cli.sh' content world info" \
        ""
    
    run_test "Set world spawn" \
        "'${RESOURCE_DIR}/cli.sh' content world set-spawn 100 64 100" \
        ""
    
    # Test server commands
    echo ""
    echo "Testing Server Commands:"
    echo "------------------------"
    
    run_test "Say message" \
        "'${RESOURCE_DIR}/cli.sh' content execute 'say Hello from Vrooli!'" \
        "Hello from Vrooli"
    
    run_test "Help command" \
        "'${RESOURCE_DIR}/cli.sh' content execute 'help'" \
        "Available commands"
    
    # Test multi-server support
    echo ""
    echo "Testing Multi-Server Support:"
    echo "-----------------------------"
    
    # Add mock server to configuration
    run_test "Add server to config" \
        "'${RESOURCE_DIR}/cli.sh' content add-server mock $MOCK_RCON_HOST $MOCK_RCON_PORT $MOCK_RCON_PASSWORD" \
        ""
    
    run_test "List configured servers" \
        "'${RESOURCE_DIR}/cli.sh' content list" \
        ""
    
    run_test "Execute on specific server" \
        "'${RESOURCE_DIR}/cli.sh' content execute 'list' mock" \
        ""
    
    # Summary
    echo ""
    echo "=============================="
    echo "Test Results: ${TESTS_PASSED}/${TESTS_RUN} passed"
    echo "=============================="
    
    if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed${NC}"
        exit 1
    fi
}

# Check dependencies
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}Error: python3 is required for mock server${NC}"
    exit 1
fi

if ! command -v nc &> /dev/null; then
    echo -e "${YELLOW}Warning: nc (netcat) not found, using alternative check${NC}"
fi

main "$@"