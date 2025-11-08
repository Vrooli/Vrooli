#!/bin/bash
# Backwards-compatible unit test runner entrypoint
set -euo pipefail

UNIT_RUNNER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${UNIT_RUNNER_DIR}/../shell/unit.sh"

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    testing::unit::run_all_tests "$@"
fi
