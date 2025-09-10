#!/bin/bash
# Apache Kafka Resource - Unit Test Phase
# v2.0 Contract: Must complete in <60s

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Source test library
source "$SCRIPT_DIR/lib/test.sh"

# Run unit test
test_unit