#!/bin/bash
# Compatibility wrapper that delegates to the phased test orchestrator.
set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

exec "$SCENARIO_DIR/test/run-tests.sh" "$@"
