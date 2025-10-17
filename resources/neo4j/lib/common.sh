#!/usr/bin/env bash
# Neo4j Resource - Common Functions

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
NEO4J_LIB_DIR="${APP_ROOT}/resources/neo4j/lib"
NEO4J_RESOURCE_DIR="${APP_ROOT}/resources/neo4j"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Source all configuration from defaults.sh to avoid duplication
# This ensures single source of truth for all Neo4j configuration
if [[ -f "${NEO4J_RESOURCE_DIR}/config/defaults.sh" ]]; then
    source "${NEO4J_RESOURCE_DIR}/config/defaults.sh"
fi

# Verify critical variables are set
if [[ -z "${NEO4J_HTTP_PORT}" ]] || [[ -z "${NEO4J_BOLT_PORT}" ]]; then
    echo "ERROR: Neo4j ports not configured. Check config/defaults.sh" >&2
    exit 1
fi