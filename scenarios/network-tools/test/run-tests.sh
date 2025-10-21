#!/bin/bash
# run-tests.sh - Test runner that delegates to run-all-tests.sh
# This file exists to satisfy v2.0 contract requirements for test/run-tests.sh
# The actual implementation is in test/run-all-tests.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Delegate to the actual test runner
exec "${SCRIPT_DIR}/run-all-tests.sh" "$@"
