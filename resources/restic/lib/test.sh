#!/usr/bin/env bash
# Restic Test Library - Test functionality for restic resource

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Delegate to test runner
"${RESOURCE_DIR}/test/run-tests.sh" "$@"