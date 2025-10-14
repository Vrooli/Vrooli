#!/usr/bin/env bash
# Idea Generator - Phased Test Suite
# Runs all test phases in sequence

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/idea-generator"
SCENARIO_DIR="$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_section() { echo -e "${BLUE}[====]${NC} $*"; }

# Parse arguments
SKIP_PHASES=""
RUN_ONLY=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip)
            SKIP_PHASES="$2"
            shift 2
            ;;
        --only)
            RUN_ONLY="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [--skip PHASE] [--only PHASE]"
            echo ""
            echo "Options:"
            echo "  --skip PHASE   Skip specific phase (structure|dependencies|unit|integration|business|performance)"
            echo "  --only PHASE   Run only specific phase"
            echo "  --help         Show this help"
            echo ""
            echo "Examples:"
            echo "  $0                              # Run all phases"
            echo "  $0 --only integration           # Run only integration tests"
            echo "  $0 --skip performance           # Skip performance tests"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Test phases in order
PHASES=(
    "structure:test/phases/test-structure.sh"
    "dependencies:test/phases/test-dependencies.sh"
    "unit:test/phases/test-unit.sh"
    "integration:test/phases/test-integration.sh"
    "business:test/phases/test-business.sh"
    "performance:test/phases/test-performance.sh"
)

cd "$SCENARIO_DIR"

log_section "Starting Phased Test Suite for idea-generator"
echo ""

FAILED_PHASES=()
PASSED_PHASES=()
SKIPPED_PHASES=()

for phase_spec in "${PHASES[@]}"; do
    PHASE_NAME="${phase_spec%%:*}"
    PHASE_SCRIPT="${phase_spec#*:}"

    # Check if we should skip this phase
    if [[ -n "$RUN_ONLY" ]] && [[ "$PHASE_NAME" != "$RUN_ONLY" ]]; then
        SKIPPED_PHASES+=("$PHASE_NAME")
        continue
    fi

    if [[ "$SKIP_PHASES" == *"$PHASE_NAME"* ]]; then
        SKIPPED_PHASES+=("$PHASE_NAME")
        continue
    fi

    # Check if phase script exists
    if [ ! -f "$PHASE_SCRIPT" ]; then
        log_warn "Phase script not found: $PHASE_SCRIPT"
        SKIPPED_PHASES+=("$PHASE_NAME (missing)")
        continue
    fi

    log_section "Running Phase: $PHASE_NAME"

    if bash "$PHASE_SCRIPT"; then
        PASSED_PHASES+=("$PHASE_NAME")
        log_info "✓ Phase $PHASE_NAME passed"
    else
        FAILED_PHASES+=("$PHASE_NAME")
        log_error "✗ Phase $PHASE_NAME failed"
    fi

    echo ""
done

# Print summary
log_section "Test Summary"
echo ""
echo -e "${GREEN}Passed:${NC}  ${#PASSED_PHASES[@]} phases"
for phase in "${PASSED_PHASES[@]}"; do
    echo -e "  ${GREEN}✓${NC} $phase"
done

if [ ${#FAILED_PHASES[@]} -gt 0 ]; then
    echo ""
    echo -e "${RED}Failed:${NC}  ${#FAILED_PHASES[@]} phases"
    for phase in "${FAILED_PHASES[@]}"; do
        echo -e "  ${RED}✗${NC} $phase"
    done
fi

if [ ${#SKIPPED_PHASES[@]} -gt 0 ]; then
    echo ""
    echo -e "${YELLOW}Skipped:${NC} ${#SKIPPED_PHASES[@]} phases"
    for phase in "${SKIPPED_PHASES[@]}"; do
        echo -e "  ${YELLOW}○${NC} $phase"
    done
fi

echo ""

if [ ${#FAILED_PHASES[@]} -gt 0 ]; then
    log_error "Test suite failed with ${#FAILED_PHASES[@]} failed phase(s)"
    exit 1
else
    log_info "All tests passed! (${#PASSED_PHASES[@]}/${#PHASES[@]} phases)"
    exit 0
fi
