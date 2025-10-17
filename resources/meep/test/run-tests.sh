#!/usr/bin/env bash
# MEEP Test Runner - Delegates to test phases

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly SCRIPT_DIR

# Source test library
source "${SCRIPT_DIR}/lib/test.sh"

# Run requested test phase
phase="${1:-all}"
shift || true

meep::test "$phase" "$@"