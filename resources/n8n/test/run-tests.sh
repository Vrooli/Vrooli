#!/usr/bin/env bash
################################################################################
# n8n Test Runner - v2.0 Universal Contract Compliant
# 
# Main test entry point for n8n resource validation
# Usage: ./test/run-tests.sh [all|smoke|integration|unit]
################################################################################

set -euo pipefail

# Get directory paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
N8N_DIR="${APP_ROOT}/resources/n8n"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Source test implementation
source "${N8N_DIR}/lib/test.sh"

# Parse test phase argument
TEST_PHASE="${1:-all}"

case "$TEST_PHASE" in
    all)
        n8n::test::all
        exit $?
        ;;
    smoke)
        n8n::test::smoke
        exit $?
        ;;
    integration)
        n8n::test::integration
        exit $?
        ;;
    unit)
        n8n::test::unit
        exit $?
        ;;
    *)
        log::error "Unknown test phase: $TEST_PHASE"
        log::info "Usage: $0 [all|smoke|integration|unit]"
        exit 1
        ;;
esac