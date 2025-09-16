#!/usr/bin/env bash
# PaperMC unit tests

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly TEST_DIR="$(dirname "${SCRIPT_DIR}")"
readonly RESOURCE_DIR="$(dirname "${TEST_DIR}")"

# Source libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

echo "Running PaperMC unit tests..."
run_unit_tests