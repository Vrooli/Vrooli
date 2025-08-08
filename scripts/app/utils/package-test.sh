#!/bin/bash
# Generic test runner script for all packages that handles both full test runs and specific file tests
# Usage:
#   package-test.sh <package> [test-file]
#   Examples:
#     package-test.sh ui                    # Run all UI tests
#     package-test.sh ui SignupView.test.tsx # Run specific test file
#     package-test.sh server auth.test.ts    # Run specific server test

# Get the directory where this script is located
APP_UTILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${APP_UTILS_DIR}/../../lib/utils/var.sh"

PROJECT_ROOT="${var_ROOT_DIR}"

# Function to show usage
show_usage() {
    echo "Usage: $0 <package> [test-file]"
    echo "Packages: ui, server, shared, jobs"
    echo "Examples:"
    echo "  $0 ui                    # Run all UI tests"
    echo "  $0 ui SignupView.test.tsx # Run specific UI test file"
    echo "  $0 server auth.test.ts    # Run specific server test"
    exit 1
}

# Check if at least package name is provided
if [ $# -eq 0 ]; then
    show_usage
fi

PACKAGE="$1"
TEST_FILE="$2"

# Navigate to project root for vitest workspace
cd "$PROJECT_ROOT"

# Handle package-specific logic
case "$PACKAGE" in
    "ui")
        if [ -z "$TEST_FILE" ]; then
            # Run all UI tests (including roundtrip and scenarios)
            npx vitest run --project ui --project ui-roundtrip --project ui-scenarios
        else
            # Find and run specific test file
            if [[ "$TEST_FILE" != *"packages/ui"* ]]; then
                # Try to find the test file
                FOUND_FILE=$(find packages/ui/src -name "$TEST_FILE" -type f | head -1)
                if [ -n "$FOUND_FILE" ]; then
                    TEST_PATH="$FOUND_FILE"
                else
                    # If not found, use glob pattern (vitest will handle the error)
                    TEST_PATH="packages/ui/src/**/$TEST_FILE"
                fi
            else
                TEST_PATH="$TEST_FILE"
            fi
            npx vitest run --project ui "$TEST_PATH"
        fi
        ;;
        
    "server")
        if [ -z "$TEST_FILE" ]; then
            # Run all server tests
            node --no-warnings $(which vitest) run --project server
        else
            # Find and run specific test file
            if [[ "$TEST_FILE" != *"packages/server"* ]]; then
                # Try to find the test file
                FOUND_FILE=$(find packages/server/src -name "$TEST_FILE" -type f | head -1)
                if [ -n "$FOUND_FILE" ]; then
                    TEST_PATH="$FOUND_FILE"
                else
                    # If not found, use glob pattern
                    TEST_PATH="packages/server/src/**/$TEST_FILE"
                fi
            else
                TEST_PATH="$TEST_FILE"
            fi
            node --no-warnings $(which vitest) run --project server "$TEST_PATH"
        fi
        ;;
        
    "shared")
        if [ -z "$TEST_FILE" ]; then
            # Run all shared tests
            npx vitest run --project shared
        else
            # Find and run specific test file
            if [[ "$TEST_FILE" != *"packages/shared"* ]]; then
                # Try to find the test file
                FOUND_FILE=$(find packages/shared/src -name "$TEST_FILE" -type f | head -1)
                if [ -n "$FOUND_FILE" ]; then
                    TEST_PATH="$FOUND_FILE"
                else
                    # If not found, use glob pattern
                    TEST_PATH="packages/shared/src/**/$TEST_FILE"
                fi
            else
                TEST_PATH="$TEST_FILE"
            fi
            npx vitest run --project shared "$TEST_PATH"
        fi
        ;;
        
    "jobs")
        if [ -z "$TEST_FILE" ]; then
            # Run all jobs tests
            npx vitest run --project jobs
        else
            # Find and run specific test file
            if [[ "$TEST_FILE" != *"packages/jobs"* ]]; then
                # Try to find the test file
                FOUND_FILE=$(find packages/jobs/src -name "$TEST_FILE" -type f | head -1)
                if [ -n "$FOUND_FILE" ]; then
                    TEST_PATH="$FOUND_FILE"
                else
                    # If not found, use glob pattern
                    TEST_PATH="packages/jobs/src/**/$TEST_FILE"
                fi
            else
                TEST_PATH="$TEST_FILE"
            fi
            npx vitest run --project jobs "$TEST_PATH"
        fi
        ;;
        
    *)
        echo "Error: Unknown package '$PACKAGE'"
        show_usage
        ;;
esac