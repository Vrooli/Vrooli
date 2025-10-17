#!/usr/bin/env bash

# ROS2 Resource - Unit Test Phase
# Library function validation (<60s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

# Run unit tests
ros2_test_unit