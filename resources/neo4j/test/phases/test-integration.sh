#!/usr/bin/env bash
################################################################################
# Neo4j Resource - Integration Test Phase
# 
# End-to-end functionality testing for Neo4j operations
# Must complete in <120 seconds per v2.0 contract
################################################################################

set -euo pipefail

# Get directories
PHASE_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${PHASE_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run integration tests with timeout
timeout 120 neo4j_test_integration || {
    echo "Error: Integration tests failed or timed out"
    exit 1
}