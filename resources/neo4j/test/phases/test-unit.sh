#!/usr/bin/env bash
################################################################################
# Neo4j Resource - Unit Test Phase
# 
# Library function validation for Neo4j operations
# Must complete in <60 seconds per v2.0 contract
################################################################################

set -euo pipefail

# Get directories
PHASE_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${PHASE_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run unit tests with timeout
timeout 60 neo4j_test_unit || {
    echo "Error: Unit tests failed or timed out"
    exit 1
}