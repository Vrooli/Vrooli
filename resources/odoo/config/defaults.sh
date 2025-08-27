#!/usr/bin/env bash
################################################################################
# Odoo Resource Configuration Defaults
# 
# Default configuration for Odoo Community Edition ERP platform
################################################################################

# Resource identification
export ODOO_RESOURCE_NAME="odoo"
export ODOO_RESOURCE_CATEGORY="business-application"
export ODOO_RESOURCE_DISPLAY_NAME="Odoo Community Edition"

# Paths  
export ODOO_BASE_DIR="${APP_ROOT}/resources/odoo"
export ODOO_DATA_DIR="${var_DATA_DIR:-${VROOLI_ROOT:-${HOME}/Vrooli}/data}/resources/odoo"
export ODOO_CONFIG_DIR="$ODOO_BASE_DIR/config"
export ODOO_LOG_FILE="${var_LOG_DIR:-${VROOLI_ROOT:-${HOME}/Vrooli}/logs}/odoo.log"

# Docker configuration
export ODOO_CONTAINER_NAME="vrooli-odoo"
export ODOO_PG_CONTAINER_NAME="vrooli-odoo-postgres"
export ODOO_NETWORK_NAME="vrooli-odoo-net"
export ODOO_IMAGE="odoo:17.0"
export ODOO_PG_IMAGE="postgres:15-alpine"

# Network configuration
export ODOO_PORT="${ODOO_PORT:-8069}"
export ODOO_LONGPOLLING_PORT="${ODOO_LONGPOLLING_PORT:-8072}"
export ODOO_PG_PORT="${ODOO_PG_PORT:-5469}"

# Database configuration
export ODOO_DB_NAME="odoo"
export ODOO_DB_USER="odoo"
export ODOO_DB_PASSWORD="${ODOO_DB_PASSWORD:-odoo_secure_password_$(date +%s)}"
export ODOO_ADMIN_EMAIL="${ODOO_ADMIN_EMAIL:-admin@example.com}"
export ODOO_ADMIN_PASSWORD="${ODOO_ADMIN_PASSWORD:-admin}"