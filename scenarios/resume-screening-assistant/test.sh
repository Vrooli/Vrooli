#!/bin/bash
# Compatibility wrapper delegating to the phased test orchestrator.
set -euo pipefail

SCENARIO_DIR="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"

if [ ! -x "$SCENARIO_DIR/test/run-tests.sh" ]; then
  echo "Expected phased test runner missing at test/run-tests.sh" >&2
  exit 1
fi

exec "$SCENARIO_DIR/test/run-tests.sh" "$@"
