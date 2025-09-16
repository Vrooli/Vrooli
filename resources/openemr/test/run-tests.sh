#!/usr/bin/env bash
# OpenEMR Test Runner
# Delegates to the main test.sh library

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run specified test phase or all tests
phase="${1:-all}"

openemr::test "$phase"