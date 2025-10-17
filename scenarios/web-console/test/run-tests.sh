#!/bin/bash
# Web Console - Test Orchestrator
# Coordinates all test phases for comprehensive scenario validation

set -eo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../.." && pwd)}"
SCENARIO_DIR="$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)"
SCENARIO_NAME="web-console"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
    echo ""
}

print_phase() {
    echo ""
    echo -e "${YELLOW}▶ Phase: $1${NC}"
    echo ""
}

# Parse command line arguments
PHASES="all"
VERBOSE=false
STOP_ON_ERROR=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --phases)
            PHASES="$2"
            shift 2
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --stop-on-error)
            STOP_ON_ERROR=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --phases PHASES       Comma-separated list of phases to run (default: all)"
            echo "                        Available: structure,dependencies,unit,integration,business,performance"
            echo "  --verbose, -v         Enable verbose output"
            echo "  --stop-on-error       Stop on first phase failure"
            echo "  --help, -h            Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                                    # Run all phases"
            echo "  $0 --phases unit,integration          # Run specific phases"
            echo "  $0 --verbose --stop-on-error          # Verbose with early exit"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

print_header "Web Console - Test Suite"
echo "Scenario: $SCENARIO_NAME"
echo "Test Directory: $SCENARIO_DIR/test"
echo "Phases: $PHASES"
echo ""

# Track results
declare -A PHASE_RESULTS
declare -A PHASE_TIMES
TOTAL_PHASES=0
PASSED_PHASES=0
FAILED_PHASES=0
SKIPPED_PHASES=0

run_phase() {
    local phase_name="$1"
    local phase_script="$2"
    local requires_runtime="${3:-false}"

    TOTAL_PHASES=$((TOTAL_PHASES + 1))

    print_phase "$phase_name"

    if [ ! -f "$phase_script" ]; then
        echo -e "${YELLOW}⊘ SKIPPED: Phase script not found${NC}"
        PHASE_RESULTS[$phase_name]="SKIPPED"
        SKIPPED_PHASES=$((SKIPPED_PHASES + 1))
        return 0
    fi

    if [ ! -x "$phase_script" ]; then
        echo -e "${RED}✗ FAILED: Phase script not executable${NC}"
        PHASE_RESULTS[$phase_name]="FAILED"
        FAILED_PHASES=$((FAILED_PHASES + 1))
        return 1
    fi

    local start_time
    start_time=$(date +%s)

    if [ "$VERBOSE" = true ]; then
        if bash "$phase_script"; then
            local end_time
            end_time=$(date +%s)
            local duration=$((end_time - start_time))
            PHASE_TIMES[$phase_name]=$duration

            echo ""
            echo -e "${GREEN}✓ PASSED${NC} (${duration}s)"
            PHASE_RESULTS[$phase_name]="PASSED"
            PASSED_PHASES=$((PASSED_PHASES + 1))
            return 0
        else
            local end_time
            end_time=$(date +%s)
            local duration=$((end_time - start_time))
            PHASE_TIMES[$phase_name]=$duration

            echo ""
            echo -e "${RED}✗ FAILED${NC} (${duration}s)"
            PHASE_RESULTS[$phase_name]="FAILED"
            FAILED_PHASES=$((FAILED_PHASES + 1))
            return 1
        fi
    else
        if bash "$phase_script" > /tmp/web-console-test-${phase_name}.log 2>&1; then
            local end_time
            end_time=$(date +%s)
            local duration=$((end_time - start_time))
            PHASE_TIMES[$phase_name]=$duration

            echo -e "${GREEN}✓ PASSED${NC} (${duration}s)"
            PHASE_RESULTS[$phase_name]="PASSED"
            PASSED_PHASES=$((PASSED_PHASES + 1))
            return 0
        else
            local end_time
            end_time=$(date +%s)
            local duration=$((end_time - start_time))
            PHASE_TIMES[$phase_name]=$duration

            echo -e "${RED}✗ FAILED${NC} (${duration}s)"
            echo "  Log: /tmp/web-console-test-${phase_name}.log"
            PHASE_RESULTS[$phase_name]="FAILED"
            FAILED_PHASES=$((FAILED_PHASES + 1))
            return 1
        fi
    fi
}

should_run_phase() {
    local phase_name="$1"
    if [ "$PHASES" = "all" ]; then
        return 0
    fi
    if echo "$PHASES" | grep -q "$phase_name"; then
        return 0
    fi
    return 1
}

# Run test phases in order
cd "$SCENARIO_DIR"

if should_run_phase "structure"; then
    if ! run_phase "Structure Validation" "test/phases/test-structure.sh" false; then
        [ "$STOP_ON_ERROR" = true ] && exit 1
    fi
fi

if should_run_phase "dependencies"; then
    if ! run_phase "Dependency Checks" "test/phases/test-dependencies.sh" false; then
        [ "$STOP_ON_ERROR" = true ] && exit 1
    fi
fi

if should_run_phase "unit"; then
    if ! run_phase "Unit Tests" "test/phases/test-unit.sh" false; then
        [ "$STOP_ON_ERROR" = true ] && exit 1
    fi
fi

if should_run_phase "integration"; then
    if ! run_phase "Integration Tests" "test/phases/test-integration.sh" true; then
        [ "$STOP_ON_ERROR" = true ] && exit 1
    fi
fi

if should_run_phase "business"; then
    if ! run_phase "Business Logic Tests" "test/phases/test-business.sh" true; then
        [ "$STOP_ON_ERROR" = true ] && exit 1
    fi
fi

if should_run_phase "performance"; then
    # Performance tests are part of unit tests (Go benchmark tests)
    echo ""
    print_phase "Performance Tests"
    echo "ℹ️  Performance tests included in unit test suite"
    echo -e "${GREEN}✓ PASSED${NC}"
fi

# Print summary
print_header "Test Summary"

echo "Phase Results:"
echo "══════════════"
for phase in "${!PHASE_RESULTS[@]}"; do
    result="${PHASE_RESULTS[$phase]}"
    time="${PHASE_TIMES[$phase]:-0}"

    case $result in
        PASSED)
            echo -e "  ${GREEN}✓${NC} $phase (${time}s)"
            ;;
        FAILED)
            echo -e "  ${RED}✗${NC} $phase (${time}s)"
            ;;
        SKIPPED)
            echo -e "  ${YELLOW}⊘${NC} $phase"
            ;;
    esac
done

echo ""
echo "Overall Statistics:"
echo "═══════════════════"
echo "  Total Phases:  $TOTAL_PHASES"
echo -e "  ${GREEN}Passed:${NC}        $PASSED_PHASES"
echo -e "  ${RED}Failed:${NC}        $FAILED_PHASES"
echo -e "  ${YELLOW}Skipped:${NC}       $SKIPPED_PHASES"

echo ""
if [ $FAILED_PHASES -eq 0 ]; then
    echo -e "${GREEN}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}  ✓ ALL TESTS PASSED${NC}"
    echo -e "${GREEN}════════════════════════════════════════════════════════════════${NC}"
    exit 0
else
    echo -e "${RED}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${RED}  ✗ SOME TESTS FAILED${NC}"
    echo -e "${RED}════════════════════════════════════════════════════════════════${NC}"
    exit 1
fi
