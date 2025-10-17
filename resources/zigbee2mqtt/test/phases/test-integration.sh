#!/usr/bin/env bash
################################################################################
# Zigbee2MQTT Integration Tests
# 
# End-to-end functionality tests (<120s)
################################################################################

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source libraries
source "${RESOURCE_DIR}/lib/test.sh"

# Run integration tests
zigbee2mqtt::test::integration