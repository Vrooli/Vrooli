#!/usr/bin/env bash
# Neo4j Resource - Common Functions

# Define directories from the current script location
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
NEO4J_LIB_DIR="${SCRIPT_DIR}"
NEO4J_RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${NEO4J_RESOURCE_DIR}/../.." && builtin pwd)"

# Source utilities
source "${REPO_ROOT}/scripts/lib/utils/var.sh"
source "${REPO_ROOT}/scripts/lib/utils/format.sh"

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