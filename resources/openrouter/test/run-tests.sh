#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_PHASE="${1:-all}"

run_phase() {
    local phase="$1"
    bash "${SCRIPT_DIR}/phases/test-${phase}.sh"
}

case "${TEST_PHASE}" in
    smoke|integration|unit|benchmark)
        run_phase "${TEST_PHASE}"
        ;;
    all)
        run_phase smoke
        run_phase integration
        run_phase unit
        run_phase benchmark
        ;;
    *)
        echo "Usage: $0 [smoke|integration|unit|benchmark|all]" >&2
        exit 1
        ;;
esac
