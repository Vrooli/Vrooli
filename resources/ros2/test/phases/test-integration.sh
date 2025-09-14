#!/usr/bin/env bash

# ROS2 Resource - Integration Test Phase
# End-to-end functionality testing (<120s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

# Run integration tests
ros2_test_integration