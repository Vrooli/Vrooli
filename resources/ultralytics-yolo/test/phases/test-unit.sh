#!/bin/bash
# Unit tests for Ultralytics YOLO - Library function validation (<60s)

set -euo pipefail

# Get resource directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source core library
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Run unit tests
yolo::test_unit