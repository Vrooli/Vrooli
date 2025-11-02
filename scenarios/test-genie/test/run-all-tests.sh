#!/bin/bash
# Compatibility shim: delegates to the shared phased test runner.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "⚠️  'run-all-tests.sh' is deprecated – forwarding to run-tests.sh." >&2
exec "$SCRIPT_DIR/run-tests.sh" "$@"
