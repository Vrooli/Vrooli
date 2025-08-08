#!/usr/bin/env bash
set -euo pipefail

_HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Common directories
export var_LIB_UTILS_DIR="$_HERE"
export var_LIB_DIR=$(cd "$var_LIB_UTILS_DIR"/.. && pwd)
export var_LIB_DEPLOY_DIR="$var_LIB_DIR/deploy"
export var_LIB_DEPS_DIR="$var_LIB_DIR/deps"
export var_LIB_LIFECYCLE_DIR="$var_LIB_DIR/lifecycle"
export var_LIB_NETWORK_DIR="$var_LIB_DIR/network"
export var_LIB_SERVICE_DIR="$var_LIB_DIR/service"
export var_LIB_SYSTEM_DIR="$var_LIB_DIR/system"
export var_SCRIPTS_DIR=$(cd "$var_LIB_DIR"/.. && pwd)
export var_SCRIPTS_TEST_DIR="$var_SCRIPTS_DIR/__test"
export var_SCRIPTS_RESOURCES_DIR="$var_SCRIPTS_DIR/resources"
export var_SCRIPTS_SCENARIOS_DIR="$var_SCRIPTS_DIR/scenarios"
export var_ROOT_DIR=$(cd "$var_SCRIPTS_DIR"/.. && pwd)

# Detect if we're in Vrooli monorepo or standalone app based on package.json
if [[ -f "$var_ROOT_DIR/package.json" ]] && \
   jq -e '.name == "vrooli" and .workspaces' "$var_ROOT_DIR/package.json" >/dev/null 2>&1; then
    export VROOLI_CONTEXT="monorepo"

    export var_PACKAGES_DIR="$var_ROOT_DIR/packages"
    export var_BACKUPS_DIR="$var_ROOT_DIR/backups"
    export var_DATA_DIR="$var_ROOT_DIR/data"
    export var_DEST_DIR="$var_ROOT_DIR/dist"

    # Vrooli configuration directory and files
    export var_VROOLI_CONFIG_DIR="$var_ROOT_DIR/.vrooli"
    export var_SERVICE_JSON_FILE="$var_VROOLI_CONFIG_DIR/service.json"
    export var_EXAMPLES_DIR="$var_VROOLI_CONFIG_DIR/examples"
    export var_SCHEMAS_DIR="$var_VROOLI_CONFIG_DIR/schemas"

    # Environment files
    export var_ENV_DEV_FILE="$var_ROOT_DIR/.env-dev"
    export var_ENV_PROD_FILE="$var_ROOT_DIR/.env-prod"
    export var_ENV_EXAMPLE_FILE="$var_ROOT_DIR/.env-example"

    # Docker Compose files
    export var_DOCKER_COMPOSE_DEV_FILE="$var_ROOT_DIR/docker-compose.yml"
    export var_DOCKER_COMPOSE_PROD_FILE="$var_ROOT_DIR/docker-compose-prod.yml"

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
