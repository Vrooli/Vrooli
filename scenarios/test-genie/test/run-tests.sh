#!/usr/bin/env bash
# Compatibility shim for test-genie phased testing.
#
# This script exists for backwards compatibility with tooling that expects
# test/run-tests.sh. The actual test orchestration is implemented natively
# in Go within the test-genie API.
#
# Usage:
#   ./run-tests.sh              Run all tests via test-genie CLI
#   ./run-tests.sh --list       List available phases (from Go catalog)
#   ./run-tests.sh --help       Show help
#
# For direct API access, use: test-genie run-tests test-genie

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CLI_PATH="${SCENARIO_DIR}/cli/test-genie"

# Check if CLI is available (local or installed)
if [[ -x "$CLI_PATH" ]]; then
    TEST_GENIE_CLI="$CLI_PATH"
elif command -v test-genie &>/dev/null; then
    TEST_GENIE_CLI="test-genie"
else
    echo "Error: test-genie CLI not found. Run 'make build' or install the CLI." >&2
    exit 1
fi

show_help() {
    cat <<'EOF'
test-genie test runner (Go-native implementation)

This script is a compatibility shim. Test phases are implemented natively in Go.

Usage:
  ./run-tests.sh [options]

Options:
  --list        List available test phases
  --help, -h    Show this help message

The Go orchestrator implements these phases:
  - structure      Validate scenario layout and configuration
  - dependencies   Check required tools and resources
  - unit           Run unit tests (Go, Node, Python)
  - integration    Validate CLI and run Bats acceptance tests
  - business       Verify requirements coverage
  - performance    Benchmark build times

To run tests, use: test-genie run-tests test-genie
Or through the lifecycle: make test
EOF
}

list_phases() {
    echo "Available phases (Go-native implementation):"
    echo "  - structure"
    echo "  - dependencies"
    echo "  - unit"
    echo "  - integration"
    echo "  - business"
    echo "  - performance"
}

# Parse arguments
case "${1:-}" in
    --help|-h)
        show_help
        exit 0
        ;;
    --list)
        list_phases
        exit 0
        ;;
    "")
        # Default: run tests via CLI
        echo "Running tests via test-genie CLI..."
        exec "$TEST_GENIE_CLI" run-tests test-genie
        ;;
    *)
        echo "Unknown option: $1" >&2
        echo "Use --help for usage information." >&2
        exit 1
        ;;
esac
