#!/usr/bin/env bash

# ROS2 Resource - Smoke Test Phase
# Quick health validation (<30s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

# Run smoke tests
ros2_test_smoke