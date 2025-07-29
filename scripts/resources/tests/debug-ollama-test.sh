#!/bin/bash
# Debug version of ollama test

set -euo pipefail

echo "🧪 Starting Ollama Integration Test (DEBUG VERSION)"
echo "Resource: ollama"
echo "Timeout: ${TEST_TIMEOUT:-60}s"
echo ""

# Test metadata
TEST_RESOURCE="ollama"
TEST_TIMEOUT="${TEST_TIMEOUT:-60}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Debug environment
echo "=== DEBUG: Environment Check ==="
echo "HEALTHY_RESOURCES_STR: ${HEALTHY_RESOURCES_STR:-NOT_SET}"
echo "HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]:-NOT_SET}"
echo "SCRIPT_DIR: ${SCRIPT_DIR:-NOT_SET}"
echo ""

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
echo "=== DEBUG: After SCRIPT_DIR calculation ==="
echo "SCRIPT_DIR: $SCRIPT_DIR"
echo "HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]:-NOT_SET}"
echo ""

echo "=== DEBUG: Sourcing assertions.sh ==="
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
echo "HEALTHY_RESOURCES after sourcing assertions.sh: ${HEALTHY_RESOURCES[*]:-NOT_SET}"

echo "=== DEBUG: Sourcing cleanup.sh ==="
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
echo "HEALTHY_RESOURCES after sourcing cleanup.sh: ${HEALTHY_RESOURCES[*]:-NOT_SET}"
echo ""

# Ollama configuration
OLLAMA_BASE_URL="http://localhost:11434"

# Test setup
setup_test() {
    echo "🔧 Setting up Ollama integration test..."
    
    echo "=== DEBUG: In setup_test function ==="
    echo "HEALTHY_RESOURCES: ${HEALTHY_RESOURCES[*]:-NOT_SET}"
    echo "TEST_RESOURCE: $TEST_RESOURCE"
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Verify Ollama is available
    echo "=== DEBUG: About to call require_resource ==="
    require_resource "$TEST_RESOURCE"
    echo "=== DEBUG: require_resource completed ==="
    
    echo "✓ Test setup complete"
}

# Main execution
main() {
    setup_test
    echo "✅ Debug test completed successfully"
}

main "$@"