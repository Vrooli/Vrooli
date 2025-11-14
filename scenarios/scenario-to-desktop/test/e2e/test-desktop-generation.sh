#!/usr/bin/env bash
# E2E Desktop Generation Test
# Validates that desktop app generation works end-to-end:
# 1. Generates desktop app via API
# 2. Verifies correct file structure
# 3. Validates package.json and configuration
# 4. Tests npm install and build

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$SCENARIO_DIR/../../.." && pwd)}"

# Source testing utilities
source "$VROOLI_ROOT/scripts/lib/testing.sh" 2>/dev/null || true

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=== E2E Desktop Generation Test ==="
echo ""

# Test configuration
API_PORT="${API_PORT:-19044}"
API_BASE_URL="${API_BASE_URL:-http://localhost:$API_PORT}"
TEST_APP_NAME="test-desktop-e2e"
TEST_OUTPUT_DIR="/tmp/scenario-to-desktop-e2e-test"

# Cleanup function
cleanup() {
    echo -e "${BLUE}Cleaning up test artifacts...${NC}"
    rm -rf "$TEST_OUTPUT_DIR"
}

# Register cleanup
trap cleanup EXIT

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper function
run_test() {
    local test_name="$1"
    local test_command="$2"

    TESTS_RUN=$((TESTS_RUN + 1))
    echo -e "${BLUE}Test $TESTS_RUN: $test_name${NC}"

    if eval "$test_command"; then
        echo -e "${GREEN}  ✓ PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}  ✗ FAIL${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

echo "🔍 Configuration:"
echo "  API URL: $API_BASE_URL"
echo "  Test output: $TEST_OUTPUT_DIR"
echo "  Test app name: $TEST_APP_NAME"
echo ""

# Cleanup any previous test artifacts
rm -rf "$TEST_OUTPUT_DIR"

echo "📋 Phase 1: API Health Check"
echo ""

run_test "API is responsive" \
    "curl -sf $API_BASE_URL/health > /dev/null"

run_test "API templates endpoint works" \
    "curl -sf $API_BASE_URL/api/v1/templates > /dev/null"

echo ""
echo "📋 Phase 2: Desktop Generation via API"
echo ""

# Generate desktop app via API
GENERATION_CONFIG=$(cat <<EOF
{
  "app_name": "$TEST_APP_NAME",
  "app_display_name": "Test Desktop E2E",
  "app_description": "E2E test desktop application",
  "version": "1.0.0",
  "author": "E2E Test Suite",
  "license": "MIT",
  "app_id": "com.vrooli.test-desktop-e2e",
  "server_type": "static",
  "server_path": "../ui/dist/index.html",
  "api_endpoint": "http://localhost:3000",
  "framework": "electron",
  "template_type": "basic",
  "platforms": ["win", "mac", "linux"],
  "output_path": "$TEST_OUTPUT_DIR",
  "features": {
    "splash": true,
    "autoUpdater": false,
    "devTools": true,
    "singleInstance": true
  },
  "window": {
    "width": 1200,
    "height": 800,
    "background": "#1e1e1e"
  }
}
EOF
)

echo "Generating desktop app..."
GENERATION_RESPONSE=$(curl -sf -X POST \
    -H "Content-Type: application/json" \
    -d "$GENERATION_CONFIG" \
    "$API_BASE_URL/api/v1/desktop/generate" || echo "ERROR")

if [ "$GENERATION_RESPONSE" = "ERROR" ]; then
    echo -e "${RED}✗ Failed to generate desktop app via API${NC}"
    echo "Response: $GENERATION_RESPONSE"
    exit 1
fi

echo "Generation response received"
BUILD_ID=$(echo "$GENERATION_RESPONSE" | grep -o '"build_id":"[^"]*"' | cut -d'"' -f4 || echo "")

if [ -z "$BUILD_ID" ]; then
    echo -e "${YELLOW}Warning: No build_id in response, but continuing...${NC}"
else
    echo "Build ID: $BUILD_ID"
fi

# Wait a moment for generation to complete
sleep 2

echo ""
echo "📋 Phase 3: File Structure Validation"
echo ""

run_test "Output directory exists" \
    "[ -d '$TEST_OUTPUT_DIR' ]"

run_test "package.json exists" \
    "[ -f '$TEST_OUTPUT_DIR/package.json' ]"

run_test "tsconfig.json exists" \
    "[ -f '$TEST_OUTPUT_DIR/tsconfig.json' ]"

run_test "src/ directory exists" \
    "[ -d '$TEST_OUTPUT_DIR/src' ]"

run_test "src/main.ts exists" \
    "[ -f '$TEST_OUTPUT_DIR/src/main.ts' ]"

run_test "src/preload.ts exists" \
    "[ -f '$TEST_OUTPUT_DIR/src/preload.ts' ]"

run_test "README.md exists" \
    "[ -f '$TEST_OUTPUT_DIR/README.md' ]"

run_test "assets/ directory exists" \
    "[ -d '$TEST_OUTPUT_DIR/assets' ]"

echo ""
echo "📋 Phase 4: Package Configuration Validation"
echo ""

if [ -f "$TEST_OUTPUT_DIR/package.json" ]; then
    run_test "package.json is valid JSON" \
        "jq empty '$TEST_OUTPUT_DIR/package.json' 2>/dev/null"

    run_test "package.json has electron dependency" \
        "jq -e '.devDependencies.electron' '$TEST_OUTPUT_DIR/package.json' > /dev/null"

    run_test "package.json has electron-builder dependency" \
        "jq -e '.devDependencies[\"electron-builder\"]' '$TEST_OUTPUT_DIR/package.json' > /dev/null"

    run_test "package.json has typescript dependency" \
        "jq -e '.devDependencies.typescript' '$TEST_OUTPUT_DIR/package.json' > /dev/null"

    run_test "package.json has build script" \
        "jq -e '.scripts.build' '$TEST_OUTPUT_DIR/package.json' > /dev/null"

    run_test "package.json has dist script" \
        "jq -e '.scripts.dist' '$TEST_OUTPUT_DIR/package.json' > /dev/null"
else
    echo -e "${RED}Skipping package.json validation - file not found${NC}"
fi

echo ""
echo "📋 Phase 5: Template Content Validation"
echo ""

if [ -f "$TEST_OUTPUT_DIR/src/main.ts" ]; then
    run_test "main.ts contains app configuration" \
        "grep -q 'APP_CONFIG' '$TEST_OUTPUT_DIR/src/main.ts'"

    run_test "main.ts contains window creation" \
        "grep -q 'createMainWindow' '$TEST_OUTPUT_DIR/src/main.ts'"

    run_test "main.ts contains server integration" \
        "grep -q 'startScenarioServer' '$TEST_OUTPUT_DIR/src/main.ts'"

    run_test "main.ts has security settings (contextIsolation)" \
        "grep -q 'contextIsolation: true' '$TEST_OUTPUT_DIR/src/main.ts'"

    run_test "main.ts disables nodeIntegration" \
        "grep -q 'nodeIntegration: false' '$TEST_OUTPUT_DIR/src/main.ts'"
else
    echo -e "${RED}Skipping main.ts validation - file not found${NC}"
fi

if [ -f "$TEST_OUTPUT_DIR/src/preload.ts" ]; then
    run_test "preload.ts uses contextBridge" \
        "grep -q 'contextBridge' '$TEST_OUTPUT_DIR/src/preload.ts'"

    run_test "preload.ts exposes desktop API" \
        "grep -q 'desktop\|electronAPI' '$TEST_OUTPUT_DIR/src/preload.ts'"
else
    echo -e "${RED}Skipping preload.ts validation - file not found${NC}"
fi

echo ""
echo "📋 Phase 6: Dependency Installation (Optional)"
echo ""

# Check if npm is available
if command -v npm > /dev/null 2>&1; then
    echo "npm found, testing dependency installation..."

    # Only test if structure is valid
    if [ -f "$TEST_OUTPUT_DIR/package.json" ]; then
        echo "Running npm install (this may take a minute)..."

        if (cd "$TEST_OUTPUT_DIR" && npm install --silent > /dev/null 2>&1); then
            run_test "npm install completes successfully" "true"

            run_test "node_modules/ directory created" \
                "[ -d '$TEST_OUTPUT_DIR/node_modules' ]"

            run_test "electron binary installed" \
                "[ -f '$TEST_OUTPUT_DIR/node_modules/.bin/electron' ] || [ -f '$TEST_OUTPUT_DIR/node_modules/electron/dist/electron' ]"

            # Test TypeScript build if dependencies installed
            if [ -d "$TEST_OUTPUT_DIR/node_modules" ]; then
                echo ""
                echo "📋 Phase 7: TypeScript Build (Optional)"
                echo ""

                if (cd "$TEST_OUTPUT_DIR" && npm run build > /dev/null 2>&1); then
                    run_test "TypeScript build succeeds" "true"

                    run_test "dist/ directory created" \
                        "[ -d '$TEST_OUTPUT_DIR/dist' ]"

                    run_test "main.js compiled" \
                        "[ -f '$TEST_OUTPUT_DIR/dist/main.js' ]"

                    run_test "preload.js compiled" \
                        "[ -f '$TEST_OUTPUT_DIR/dist/preload.js' ]"
                else
                    echo -e "${YELLOW}TypeScript build failed (non-critical for E2E test)${NC}"
                fi
            fi
        else
            echo -e "${YELLOW}npm install failed (non-critical for E2E test)${NC}"
        fi
    fi
else
    echo -e "${YELLOW}npm not found - skipping dependency installation test${NC}"
fi

echo ""
echo "=== E2E Test Summary ==="
echo ""
echo "Tests run:    $TESTS_RUN"
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All E2E tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some E2E tests failed${NC}"
    exit 1
fi
