#!/bin/bash
# ERPNext Configuration

# Default port
ERPNEXT_PORT="${ERPNEXT_PORT:-8020}"

# Configuration defaults
ERPNEXT_VERSION="${ERPNEXT_VERSION:-v15}"
ERPNEXT_ADMIN_PASSWORD="${ERPNEXT_ADMIN_PASSWORD:-admin}"
ERPNEXT_DB_NAME="${ERPNEXT_DB_NAME:-erpnext}"
ERPNEXT_SITE_NAME="${ERPNEXT_SITE_NAME:-vrooli.local}"

# Docker image
ERPNEXT_IMAGE="${ERPNEXT_IMAGE:-frappe/erpnext:${ERPNEXT_VERSION}}"

# Export for use in docker-compose
export ERPNEXT_PORT
export ERPNEXT_VERSION
export ERPNEXT_ADMIN_PASSWORD
export ERPNEXT_DB_NAME
export ERPNEXT_SITE_NAME
export ERPNEXT_IMAGE