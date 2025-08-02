#!/bin/bash
# ====================================================================
# Quick Test - Simplified Entry Point
# ====================================================================
#
# Test a single resource quickly without complex options.
# Perfect for developers who just want to check if something works.
#
# Usage:
#   ./quick-test.sh [resource-name]
#   ./quick-test.sh ollama
#   ./quick-test.sh          # Test all available resources
#
# ====================================================================

set -euo pipefail

# Colors for output
if [[ -t 1 ]]; then
    GREEN='\033[0;32m'
    BLUE='\033[0;34m'
    YELLOW='\033[1;33m'
    NC='\033[0m'
else
    GREEN='' BLUE='' YELLOW='' NC=''
fi

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Show usage
show_usage() {
    cat << EOF
🚀 Quick Test - Simple Resource Testing

USAGE:
    $0 [resource-name]

EXAMPLES:
    $0              # Test all available resources
    $0 ollama       # Test just Ollama
    $0 n8n          # Test just n8n
    $0 minio        # Test just MinIO

AVAILABLE RESOURCES:
    AI Services:      ollama, whisper, comfyui, unstructured-io
    Automation:       n8n, node-red, windmill, huginn
    Agents:           agent-s2, browserless, claude-code
    Search:           searxng
    Storage:          minio, postgres, redis, qdrant, questdb, vault

TIP: Run './validate-setup.sh' first to check if services are running

EOF
}

# Main function
main() {
    echo -e "${BLUE}🚀 Vrooli Quick Test${NC}"
    echo

    # Handle help
    if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
        show_usage
        exit 0
    fi

    # Check if run.sh exists
    if [[ ! -f "$SCRIPT_DIR/run.sh" ]]; then
        echo -e "${YELLOW}❌ Cannot find run.sh in $SCRIPT_DIR${NC}"
        echo "Make sure you're running from the tests directory"
        exit 1
    fi

    # Determine what to test
    local test_args=()
    if [[ $# -eq 0 ]]; then
        echo -e "${BLUE}🎯 Testing all available resources...${NC}"
        test_args=("--single-only")
    else
        local resource="$1"
        echo -e "${BLUE}🎯 Testing resource: $resource${NC}"
        test_args=("--resource" "$resource")
    fi

    # Add standard options for quick testing
    test_args+=("--verbose" "--fail-fast" "--timeout" "600")

    echo -e "${BLUE}📋 Running: ./run.sh ${test_args[*]}${NC}"
    echo

    # Run the test
    if "$SCRIPT_DIR/run.sh" "${test_args[@]}"; then
        echo
        echo -e "${GREEN}✅ Quick test completed successfully!${NC}"
        echo -e "${BLUE}💡 For more options, try: ./run.sh --help-beginner${NC}"
    else
        echo
        echo -e "${YELLOW}❌ Quick test failed${NC}"
        echo -e "${BLUE}💡 For troubleshooting, see: docs/TROUBLESHOOTING.md${NC}"
        echo -e "${BLUE}💡 Check what's running: ../index.sh --action discover${NC}"
        exit 1
    fi
}

main "$@"