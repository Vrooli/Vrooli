#!/bin/bash
# Database initialization script for PRD Control Tower
# This script initializes the PostgreSQL schema required for draft management

set -e

# Colors for output
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
RED='\033[1;31m'
RESET='\033[0m'

echo -e "${BLUE}🔧 Initializing PRD Control Tower database...${RESET}"

# Get the script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SCHEMA_FILE="${PROJECT_ROOT}/initialization/postgres/schema.sql"
MIGRATIONS_DIR="${PROJECT_ROOT}/initialization/postgres/migrations"

# Check if schema file exists
if [ ! -f "${SCHEMA_FILE}" ]; then
    echo -e "${RED}❌ Schema file not found: ${SCHEMA_FILE}${RESET}"
    exit 1
fi

# Get PostgreSQL connection details from resource
echo -e "${BLUE}  → Fetching PostgreSQL connection details...${RESET}"
POSTGRES_STATUS=$(vrooli resource status postgres --json 2>/dev/null || echo '{}')

if [ "${POSTGRES_STATUS}" = "{}" ]; then
    echo -e "${RED}❌ PostgreSQL resource not running. Start it with:${RESET}"
    echo -e "${YELLOW}   vrooli resource start postgres${RESET}"
    exit 1
fi

# Try to initialize via docker exec (prefer vrooli-postgres containers)
CONTAINER_NAME=$(docker ps --format "{{.Names}}" | grep -m1 'vrooli-postgres' || true)
if [ -z "${CONTAINER_NAME}" ]; then
    CONTAINER_NAME=$(docker ps --filter "name=postgres" --format "{{.Names}}" | head -1)
fi

if [ -z "${CONTAINER_NAME}" ]; then
    echo -e "${RED}❌ PostgreSQL container not found${RESET}"
    exit 1
fi

# Get credentials from container environment
CONTAINER_ENV=$(docker exec "${CONTAINER_NAME}" env 2>/dev/null)
POSTGRES_USER=$(echo "${CONTAINER_ENV}" | grep "^POSTGRES_USER=" | cut -d= -f2 | tr -d '\r')
POSTGRES_DB=$(echo "${CONTAINER_ENV}" | grep "^POSTGRES_DB=" | cut -d= -f2 | tr -d '\r')

# Default to 'vrooli' if not found
if [ -z "${POSTGRES_USER}" ]; then
    POSTGRES_USER="vrooli"
fi
if [ -z "${POSTGRES_DB}" ]; then
    POSTGRES_DB="vrooli"
fi

echo -e "${BLUE}  → Initializing schema in ${POSTGRES_DB} database...${RESET}"

# Copy schema to container and execute
docker cp "${SCHEMA_FILE}" "${CONTAINER_NAME}:/tmp/schema.sql" >/dev/null 2>&1

# Execute schema file
OUTPUT=$(docker exec "${CONTAINER_NAME}" sh -c "psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} -f /tmp/schema.sql" 2>&1 | grep -v "NOTICE: relation")
EXIT_CODE=$?

# Clean up schema file
docker exec "${CONTAINER_NAME}" rm /tmp/schema.sql >/dev/null 2>&1 || true

if [ ${EXIT_CODE} -ne 0 ]; then
    echo -e "${RED}❌ Database initialization failed:${RESET}"
    echo "${OUTPUT}"
    exit 1
fi

# Apply migrations if present
if [ -d "${MIGRATIONS_DIR}" ]; then
    shopt -s nullglob
    MIGRATION_FILES=(${MIGRATIONS_DIR}/*.sql)
    shopt -u nullglob

    if [ ${#MIGRATION_FILES[@]} -gt 0 ]; then
        echo -e "${BLUE}  → Applying migrations...${RESET}"
        for migration in $(printf "%s\n" "${MIGRATION_FILES[@]}" | sort); do
            base_name="$(basename "${migration}")"
            echo -e "${BLUE}     • ${base_name}${RESET}"
            docker cp "${migration}" "${CONTAINER_NAME}:/tmp/${base_name}" >/dev/null 2>&1
            docker exec "${CONTAINER_NAME}" sh -c "psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} -f /tmp/${base_name}" >/dev/null 2>&1
            docker exec "${CONTAINER_NAME}" rm "/tmp/${base_name}" >/dev/null 2>&1 || true
        done
    fi
fi

echo -e "${GREEN}✅ Database initialized successfully${RESET}"
