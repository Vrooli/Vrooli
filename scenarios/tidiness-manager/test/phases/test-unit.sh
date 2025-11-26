#!/bin/bash
# Orchestrates language unit tests with coverage thresholds.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/unit.sh"

# Set DATABASE_URL for auto-campaigns tests
# Extract postgres credentials from running container
POSTGRES_USER=$(docker exec vrooli-postgres-main env | grep "^POSTGRES_USER=" | cut -d'=' -f2)
POSTGRES_PASSWORD=$(docker exec vrooli-postgres-main env | grep "^POSTGRES_PASSWORD=" | cut -d'=' -f2)
POSTGRES_PORT="5433"  # vrooli-postgres-main is exposed on 5433
export DATABASE_URL="postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/tidiness-manager?sslmode=disable"

testing::unit::validate_all
