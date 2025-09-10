#!/bin/bash
# Apache Kafka Resource - Smoke Test Phase
# v2.0 Contract: Must complete in <30s

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Source test library
source "$SCRIPT_DIR/lib/test.sh"

# Run smoke test
test_smoke