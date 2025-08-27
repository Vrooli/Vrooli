#!/usr/bin/env bash
# Neo4j Configuration Defaults

# Service configuration
export NEO4J_NAME="neo4j"
export NEO4J_DISPLAY_NAME="Neo4j Graph Database"
export NEO4J_CATEGORY="storage"
export NEO4J_DESCRIPTION="Native property graph database with Cypher query language"

# Container configuration
export NEO4J_CONTAINER_NAME="${NEO4J_CONTAINER_NAME:-vrooli-neo4j}"
export NEO4J_VERSION="${NEO4J_VERSION:-5.15.0}"
export NEO4J_IMAGE="${NEO4J_IMAGE:-neo4j:${NEO4J_VERSION}}"

# Port configuration
export NEO4J_HTTP_PORT="${NEO4J_HTTP_PORT:-7474}"
export NEO4J_BOLT_PORT="${NEO4J_BOLT_PORT:-7687}"

# Authentication
export NEO4J_AUTH="${NEO4J_AUTH:-neo4j/VrooliNeo4j2024!}"

# Memory configuration
export NEO4J_HEAP_SIZE="${NEO4J_HEAP_SIZE:-512M}"

# Data directories (these use var_ROOT_DIR from framework)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
export NEO4J_DATA_DIR="${NEO4J_DATA_DIR:-${var_ROOT_DIR}/data/resources/neo4j/data}"
export NEO4J_LOGS_DIR="${NEO4J_LOGS_DIR:-${var_ROOT_DIR}/data/resources/neo4j/logs}"
export NEO4J_IMPORT_DIR="${NEO4J_IMPORT_DIR:-${var_ROOT_DIR}/data/resources/neo4j/import}"
export NEO4J_PLUGINS_DIR="${NEO4J_PLUGINS_DIR:-${var_ROOT_DIR}/data/resources/neo4j/plugins}"
export NEO4J_CONF_DIR="${NEO4J_CONF_DIR:-${var_ROOT_DIR}/data/resources/neo4j/conf}"

# Export config function
neo4j::export_config() {
    export NEO4J_NAME NEO4J_DISPLAY_NAME NEO4J_CATEGORY NEO4J_DESCRIPTION
    export NEO4J_CONTAINER_NAME NEO4J_VERSION NEO4J_IMAGE
    export NEO4J_HTTP_PORT NEO4J_BOLT_PORT NEO4J_AUTH NEO4J_HEAP_SIZE
    export NEO4J_DATA_DIR NEO4J_LOGS_DIR NEO4J_IMPORT_DIR NEO4J_PLUGINS_DIR NEO4J_CONF_DIR
}