#!/usr/bin/env bash
set -euo pipefail

# var.sh defines directory variables and should always be sourced
# No source guard needed as variables are idempotent

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
var_ROOT_DIR="$APP_ROOT"
export var_ROOT_DIR

# DERIVE all paths from ROOT - zero subshells!
export var_SCRIPTS_DIR="$var_ROOT_DIR/scripts"
export var_LIB_DIR="$var_SCRIPTS_DIR/lib"
export var_LIB_UTILS_DIR="$var_LIB_DIR/utils"
export var_LIB_DEPLOY_DIR="$var_LIB_DIR/deploy"
export var_LIB_DEPS_DIR="$var_LIB_DIR/deps"
export var_LIB_TOOLS_DIR="$var_LIB_DIR/tools"
export var_LIB_LIFECYCLE_DIR="$var_LIB_DIR/lifecycle"
export var_LIB_NETWORK_DIR="$var_LIB_DIR/network"
export var_LIB_SERVICE_DIR="$var_LIB_DIR/service"
export var_LIB_SYSTEM_DIR="$var_LIB_DIR/system"
export var_TEST_DIR="$var_ROOT_DIR/__test"
export var_RESOURCES_DIR="$var_ROOT_DIR/resources"
export var_SCENARIOS_DIR="$var_ROOT_DIR/scenarios"
export var_SCRIPTS_RESOURCES_DIR="$var_SCRIPTS_DIR/resources"
export var_SCRIPTS_RESOURCES_LIB_DIR="$var_SCRIPTS_RESOURCES_DIR/lib"
export var_SCRIPTS_RESOURCES_TESTS_LIB_DIR="$var_SCRIPTS_RESOURCES_DIR/tests/lib"
export var_SCRIPTS_SCENARIOS_DIR="$var_SCRIPTS_DIR/scenarios"
export var_SCRIPTS_SCENARIOS_INJECTION_DIR="$var_SCRIPTS_SCENARIOS_DIR/injection"
export var_JQ_RESOURCES_EXPR='(.dependencies.resources // {})'
export var_JQ_SCENARIO_DEPENDENCIES_EXPR='(.dependencies.scenarios // {})'

# Vrooli configuration directory and files
export var_VROOLI_CONFIG_DIR="$var_ROOT_DIR/.vrooli"
export var_SERVICE_JSON_FILE="$var_VROOLI_CONFIG_DIR/service.json"
export var_EXAMPLES_DIR="$var_VROOLI_CONFIG_DIR/examples"
export var_SCHEMAS_DIR="$var_VROOLI_CONFIG_DIR/schemas"

# Common files
export var_LIFECYCLE_ENGINE_FILE="$var_LIB_LIFECYCLE_DIR/engine.sh"
export var_REPOSITORY_FILE="$var_LIB_SERVICE_DIR/repository.sh"
export var_SYSTEM_COMMANDS_FILE="$var_LIB_SYSTEM_DIR/system_commands.sh"
export var_TRASH_FILE="$var_LIB_SYSTEM_DIR/trash.sh"
export var_EXIT_CODES_FILE="$var_LIB_UTILS_DIR/exit_codes.sh"
export var_FLOW_FILE="$var_LIB_UTILS_DIR/flow.sh"
export var_LOG_FILE="$var_LIB_UTILS_DIR/log.sh"
export var_RESOURCES_COMMON_FILE="$var_SCRIPTS_RESOURCES_DIR/common.sh"
export var_PORT_REGISTRY_FILE="$var_SCRIPTS_RESOURCES_DIR/port_registry.sh"
export var_RUNTIME_ENGINE="$var_SCRIPTS_SCENARIOS_INJECTION_DIR/engine.sh"

# Detect if we're in Vrooli monorepo based on root folder name (case-insensitive)
# Use parameter expansion instead of basename subshell
ROOT_FOLDER_NAME="${var_ROOT_DIR##*/}"
if [[ "${ROOT_FOLDER_NAME,,}" == "vrooli" ]]; then
    export VROOLI_CONTEXT="monorepo"

    export var_PACKAGES_DIR="$var_ROOT_DIR/packages"
    export var_BACKUPS_DIR="$var_ROOT_DIR/backups"
    export var_DATA_DIR="$var_ROOT_DIR/data"
    export var_DEST_DIR="$var_ROOT_DIR/dist"

    # Environment files
    export var_ENV_DEV_FILE="$var_ROOT_DIR/.env-dev"
    export var_ENV_PROD_FILE="$var_ROOT_DIR/.env-prod"
    export var_ENV_EXAMPLE_FILE="$var_ROOT_DIR/.env-example"

    # Docker directory and compose files
    export var_DOCKER_DIR="$var_ROOT_DIR/docker"
    export var_DOCKER_COMPOSE_DEV_FILE="$var_DOCKER_DIR/docker-compose.yml"
    export var_DOCKER_COMPOSE_PROD_FILE="$var_DOCKER_DIR/docker-compose-prod.yml"

    # Key pairs
    export var_STAGING_CI_SSH_PRIV_KEY_FILE="$var_ROOT_DIR/ci_ssh_priv_staging.pem"
    export var_STAGING_CI_SSH_PUB_KEY_FILE="$var_ROOT_DIR/ci_ssh_pub_staging.pem"
    export var_PRODUCTION_CI_SSH_PRIV_KEY_FILE="$var_ROOT_DIR/ci_ssh_priv_production.pem"
    export var_PRODUCTION_CI_SSH_PUB_KEY_FILE="$var_ROOT_DIR/ci_ssh_pub_production.pem"
    export var_STAGING_JWT_PRIV_KEY_FILE="$var_ROOT_DIR/jwt_priv_staging.pem"
    export var_STAGING_JWT_PUB_KEY_FILE="$var_ROOT_DIR/jwt_pub_staging.pem"
    export var_PRODUCTION_JWT_PRIV_KEY_FILE="$var_ROOT_DIR/jwt_priv_production.pem"
    export var_PRODUCTION_JWT_PUB_KEY_FILE="$var_ROOT_DIR/jwt_pub_production.pem"

    # Remote server
    export var_REMOTE_ROOT_DIR="/root/Vrooli"
    export var_REMOTE_DEST_DIR="$var_REMOTE_ROOT_DIR/dist"

    # Package directories/files
    export var_POSTGRES_ENTRYPOINT_DIR="$var_PACKAGES_DIR/postgres/entrypoint"
    export var_DB_SCHEMA_FILE="$var_PACKAGES_DIR/server/src/db/schema.prisma"

    # Script directories
    export var_APP_DIR="$var_SCRIPTS_DIR/app"
    export var_APP_LIFECYCLE_DIR="$var_APP_DIR/lifecycle"
    export var_APP_LIFECYCLE_DEPLOY_DIR="$var_APP_LIFECYCLE_DIR/deploy"
    export var_APP_LIFECYCLE_DEVELOP_DIR="$var_APP_LIFECYCLE_DIR/develop"
    export var_APP_LIFECYCLE_SETUP_DIR="$var_APP_LIFECYCLE_DIR/setup"
    export var_APP_PACKAGE_DIR="$var_APP_DIR/package"
    export var_APP_UTILS_DIR="$var_APP_DIR/utils"
else
    export VROOLI_CONTEXT="standalone"
fi
