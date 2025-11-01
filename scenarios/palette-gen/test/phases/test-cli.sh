#!/bin/bash
# CLI Tests Phase for Palette Gen
# Validates CLI commands work correctly

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== CLI Tests Phase ===${NC}"

# Check if bats is available
if ! command -v bats &> /dev/null; then
    echo -e "${YELLOW}⚠️  BATS not installed - installing BATS test framework${NC}"

    # Try to install bats via npm (most portable approach)
    if command -v npm &> /dev/null; then
        npm install -g bats || {
            echo -e "${YELLOW}⚠️  Could not install BATS globally, trying local install${NC}"
            cd "$SCENARIO_DIR"
            npm install --save-dev bats || {
                echo -e "${YELLOW}⚠️  BATS installation failed - skipping CLI tests${NC}"
                echo -e "${BLUE}To enable CLI tests, install BATS: npm install -g bats${NC}"
                exit 0
            }
            # Use locally installed bats
            export PATH="$SCENARIO_DIR/node_modules/.bin:$PATH"
        }
    else
        echo -e "${YELLOW}⚠️  npm not available - skipping CLI tests${NC}"
        echo -e "${BLUE}To enable CLI tests, install BATS: https://github.com/bats-core/bats-core${NC}"
        exit 0
    fi
fi

# Run CLI tests if bats is available
if command -v bats &> /dev/null; then
    echo -e "${BLUE}Running CLI test suite...${NC}"
    cd "$SCRIPT_DIR/../cli"

    # Export API_PORT from lifecycle if available
    if [[ -z "$API_PORT" ]]; then
        # Try to read from service status
        API_PORT=$(grep -o '"API_PORT":[0-9]*' "$SCENARIO_DIR/.vrooli/service.json" 2>/dev/null | grep -o '[0-9]*' || echo "16917")
    fi
    export API_PORT

    if bats run-cli-tests.sh; then
        echo -e "${GREEN}✅ CLI tests passed${NC}"
        exit 0
    else
        echo -e "${RED}❌ CLI tests failed${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠️  BATS not available - skipping CLI tests${NC}"
    exit 0
fi
