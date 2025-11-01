#!/bin/bash
# Back-compat wrapper for legacy performance entrypoints.
# Delegates to the phased performance test.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ "${1:-all}" != "all" ]; then
  echo "⚠️  performance-test.sh arguments are deprecated; running phased performance checks instead" >&2
fi

"${SCRIPT_DIR}/run-tests.sh" performance
