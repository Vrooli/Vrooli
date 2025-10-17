#!/usr/bin/env bash

# ROS2 Resource - Test Runner
# Delegates to test phases based on requested test type

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Determine which test to run
TEST_TYPE="${1:-all}"

# Run the appropriate test
ros2_test "${TEST_TYPE}"