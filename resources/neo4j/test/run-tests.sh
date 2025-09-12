#!/usr/bin/env bash
################################################################################
# Neo4j Resource - Main Test Runner
# 
# Orchestrates all test phases for Neo4j validation
################################################################################

set -euo pipefail

# Get script directory
TEST_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
APP_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Parse test phase argument
PHASE="${1:-all}"

echo "Neo4j Test Runner"
echo "================="
echo "Phase: $PHASE"
echo

case "$PHASE" in
    smoke)
        neo4j_test_smoke
        ;;
    integration)
        neo4j_test_integration
        ;;
    unit)
        neo4j_test_unit
        ;;
    all)
        neo4j_test_all
        ;;
    *)
        echo "Error: Unknown test phase: $PHASE"
        echo "Usage: $0 [smoke|integration|unit|all]"
        exit 1
        ;;
esac