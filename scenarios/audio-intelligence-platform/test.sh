#!/usr/bin/env bash
# Compatibility wrapper that delegates to the phased test runner
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCENARIO_DIR="${APP_ROOT}/scenarios/audio-intelligence-platform"

"$SCENARIO_DIR/test/run-tests.sh" "$@"
