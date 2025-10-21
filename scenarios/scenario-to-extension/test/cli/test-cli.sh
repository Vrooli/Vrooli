#!/usr/bin/env bash
# CLI integration tests for scenario-to-extension
# This script runs BATS tests for the CLI interface

set -euo pipefail

# Colors for output
GREEN='\033[1;32m'
RED='\033[1;31m'
BLUE='\033[1;34m'
YELLOW='\033[1;33m'
RESET='\033[0m'

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo -e "${BLUE}=== CLI Tests Phase (Target: <60s) ===${RESET}"

# Check if BATS is installed
if ! command -v bats &> /dev/null; then
    echo -e "${RED}[ERROR] BATS not found. Install with: sudo apt-get install bats${RESET}"
    exit 1
fi

# Check if CLI binary exists
if [ ! -f "$SCENARIO_DIR/cli/scenario-to-extension" ]; then
    echo -e "${YELLOW}[WARNING] CLI binary not found. Running install script...${RESET}"
    if ! (cd "$SCENARIO_DIR/cli" && ./install.sh); then
        echo -e "${RED}[ERROR] Failed to install CLI binary${RESET}"
        exit 1
    fi
fi

# Check if CLI is in PATH
if ! command -v scenario-to-extension &> /dev/null; then
    echo -e "${YELLOW}[WARNING] CLI not in PATH. Adding $SCENARIO_DIR/cli to PATH for tests${RESET}"
    export PATH="$SCENARIO_DIR/cli:$PATH"
fi

# Get API_PORT from lifecycle system if scenario is running
if command -v vrooli &> /dev/null; then
    STATUS_JSON=$(vrooli scenario status scenario-to-extension --json 2>/dev/null || echo '{}')
    RUNNING_API_PORT=$(echo "$STATUS_JSON" | jq -r '.scenario_data.allocated_ports.API_PORT // .ports.API_PORT // empty' 2>/dev/null || echo "")

    if [[ -n "$RUNNING_API_PORT" ]] && [[ "$RUNNING_API_PORT" != "null" ]]; then
        echo -e "${GREEN}[INFO] Found running scenario on API port: $RUNNING_API_PORT${RESET}"
        export API_PORT="$RUNNING_API_PORT"
    else
        echo -e "${YELLOW}[INFO] Scenario not running - CLI→API integration tests will fail${RESET}"
    fi
fi

# Run BATS tests
echo -e "${BLUE}🔍 Running CLI tests with BATS...${RESET}"

START_TIME=$(date +%s)

# Run BATS test suite
bats "$SCRIPT_DIR/cli-tests.bats"
TEST_EXIT_CODE=$?

END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo ""
    echo -e "${GREEN}[SUCCESS] ✅ CLI tests completed in ${ELAPSED}s${RESET}"
    exit 0
else
    echo ""
    echo -e "${RED}[FAILED] ❌ CLI tests failed in ${ELAPSED}s${RESET}"
    exit 1
fi
