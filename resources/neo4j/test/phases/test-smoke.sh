#!/usr/bin/env bash
################################################################################
# Neo4j Resource - Smoke Test Phase
# 
# Quick validation that Neo4j is running and responsive
# Must complete in <30 seconds per v2.0 contract
################################################################################

set -euo pipefail

# Get directories
PHASE_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${PHASE_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run smoke tests with timeout
timeout 30 neo4j_test_smoke || {
    echo "Error: Smoke tests failed or timed out"
    exit 1
}