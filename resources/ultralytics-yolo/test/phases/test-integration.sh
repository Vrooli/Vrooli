#!/bin/bash
# Integration tests for Ultralytics YOLO - End-to-end functionality (<120s)

set -euo pipefail

# Get resource directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source core library
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Run integration tests
yolo::test_integration