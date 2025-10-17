#!/bin/bash

# OpenEMS Test Runner
# Orchestrates all test phases

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
PASSED=0
FAILED=0
SKIPPED=0

# Run test phase
run_phase() {
    local phase="$1"
    local script="${SCRIPT_DIR}/phases/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${YELLOW}⚠️  Skipping $phase tests (script not found)${NC}"
        ((SKIPPED++))
        return 0
    fi
    
    echo -e "${GREEN}▶️  Running $phase tests...${NC}"
    
    if bash "$script"; then
        echo -e "${GREEN}✅ $phase tests passed${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}❌ $phase tests failed${NC}"
        ((FAILED++))
        return 1
    fi
}

# Main test execution
main() {
    echo "🧪 OpenEMS Test Suite"
    echo "===================="
    
    local phase="${1:-all}"
    
    case "$phase" in
        smoke)
            run_phase "smoke"
            ;;
        integration)
            run_phase "integration"
            ;;
        unit)
            run_phase "unit"
            ;;
        der-simulation)
            run_phase "der-simulation"
            ;;
        api-validation)
            run_phase "api-validation"
            ;;
        all)
            run_phase "smoke" || true
            run_phase "integration" || true
            run_phase "unit" || true
            run_phase "der-simulation" || true
            run_phase "api-validation" || true
            ;;
        *)
            echo -e "${RED}❌ Unknown test phase: $phase${NC}"
            echo "Available phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    # Print summary
    echo ""
    echo "📊 Test Summary"
    echo "=============="
    echo -e "${GREEN}✅ Passed: $PASSED${NC}"
    echo -e "${RED}❌ Failed: $FAILED${NC}"
    echo -e "${YELLOW}⚠️  Skipped: $SKIPPED${NC}"
    
    # Return appropriate exit code
    if [[ $FAILED -gt 0 ]]; then
        exit 1
    else
        exit 0
    fi
}

# Execute main function
main "$@"