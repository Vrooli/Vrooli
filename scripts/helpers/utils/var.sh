#!/usr/bin/env bash
set -euo pipefail

_HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Common directories
export var_UTILS_DIR="$_HERE"
export var_HELPERS_DIR=$(cd "$var_UTILS_DIR"/.. && pwd)
export var_SCRIPTS_DIR=$(cd "$var_HELPERS_DIR"/.. && pwd)
export var_SCRIPT_TESTS_DIR="$var_SCRIPTS_DIR/__tests"
export var_ROOT_DIR=$(cd "$var_SCRIPTS_DIR"/.. && pwd)
export var_PACKAGES_DIR="$var_ROOT_DIR/packages"
export var_BACKUPS_DIR="$var_ROOT_DIR/backups"
export var_DATA_DIR="$var_ROOT_DIR/data"
export var_DEST_DIR="$var_ROOT_DIR/dist"

# Environment files
export var_ENV_DEV_FILE="$var_ROOT_DIR/.env-dev"
export var_ENV_PROD_FILE="$var_ROOT_DIR/.env-prod"

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
export var_REMOTE_ROOT_DIR="/root/StartupFromScratch"
export var_REMOTE_DEST_DIR="$var_REMOTE_ROOT_DIR/dist"

# Package directories/files
export var_POSTGRES_ENTRYPOINT_DIR="$var_PACKAGES_DIR/postgres/entrypoint"
export var_DB_SCHEMA_FILE="$var_PACKAGES_DIR/server/src/db/schema.prisma"