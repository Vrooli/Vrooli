#!/usr/bin/env bash
# Neo4j Resource - Common Functions

# Get script directory
NEO4J_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NEO4J_RESOURCE_DIR="$(dirname "$NEO4J_LIB_DIR")"

# Source utilities
source "$NEO4J_RESOURCE_DIR/../../../lib/utils/var.sh"
source "$NEO4J_RESOURCE_DIR/../../../lib/utils/format.sh"

# Neo4j configuration
export NEO4J_CONTAINER_NAME="vrooli-neo4j"
export NEO4J_VERSION="5.15.0"
export NEO4J_HTTP_PORT="7474"
export NEO4J_BOLT_PORT="7687"
export NEO4J_DATA_DIR="${var_ROOT_DIR}/.vrooli/neo4j/data"
export NEO4J_LOGS_DIR="${var_ROOT_DIR}/.vrooli/neo4j/logs"
export NEO4J_IMPORT_DIR="${var_ROOT_DIR}/.vrooli/neo4j/import"
export NEO4J_PLUGINS_DIR="${var_ROOT_DIR}/.vrooli/neo4j/plugins"
export NEO4J_CONF_DIR="${var_ROOT_DIR}/.vrooli/neo4j/conf"
export NEO4J_AUTH="neo4j/VrooliNeo4j2024!"

# Docker image
export NEO4J_IMAGE="neo4j:${NEO4J_VERSION}"

# Max heap size
export NEO4J_HEAP_SIZE="512M"