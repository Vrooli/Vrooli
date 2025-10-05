#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "INFO: Checking scenario dependencies..."

# Check Node.js version
if ! command -v node &> /dev/null; then
    testing::phase::log "ERROR" "Node.js is not installed"
    exit 1
fi

NODE_VERSION=$(node --version)
testing::phase::log "INFO" "Node.js version: $NODE_VERSION"

# Check npm version
if ! command -v npm &> /dev/null; then
    testing::phase::log "ERROR" "npm is not installed"
    exit 1
fi

NPM_VERSION=$(npm --version)
testing::phase::log "INFO" "npm version: $NPM_VERSION"

# Check if package.json exists
if [ ! -f "api/package.json" ]; then
    testing::phase::log "ERROR" "api/package.json not found"
    exit 1
fi

# Check if node_modules exists
if [ ! -d "api/node_modules" ]; then
    testing::phase::log "WARN" "api/node_modules not found, installing dependencies..."
    cd api && npm install &> /dev/null
    if [ $? -ne 0 ]; then
        testing::phase::log "ERROR" "Failed to install dependencies"
        exit 1
    fi
    cd ..
fi

# Verify critical dependencies
testing::phase::log "INFO" "Verifying critical dependencies..."

CRITICAL_DEPS=("express" "cors" "@faker-js/faker" "uuid" "joi" "csv-writer" "js2xmlparser")
MISSING_DEPS=()

cd api
for dep in "${CRITICAL_DEPS[@]}"; do
    if ! npm list "$dep" &> /dev/null; then
        MISSING_DEPS+=("$dep")
    fi
done
cd ..

if [ ${#MISSING_DEPS[@]} -gt 0 ]; then
    testing::phase::log "ERROR" "Missing critical dependencies: ${MISSING_DEPS[*]}"
    exit 1
fi

testing::phase::log "INFO" "All critical dependencies are installed"

# Check for security vulnerabilities (audit)
testing::phase::log "INFO" "Running security audit..."
cd api
AUDIT_OUTPUT=$(npm audit --audit-level=high 2>&1)
AUDIT_EXIT=$?

if [ $AUDIT_EXIT -ne 0 ]; then
    HIGH_VULNS=$(echo "$AUDIT_OUTPUT" | grep -oP '\d+(?= high)' | head -1)
    CRITICAL_VULNS=$(echo "$AUDIT_OUTPUT" | grep -oP '\d+(?= critical)' | head -1)

    if [ -n "$HIGH_VULNS" ] && [ "$HIGH_VULNS" -gt 0 ]; then
        testing::phase::log "WARN" "Found $HIGH_VULNS high severity vulnerabilities"
    fi

    if [ -n "$CRITICAL_VULNS" ] && [ "$CRITICAL_VULNS" -gt 0 ]; then
        testing::phase::log "WARN" "Found $CRITICAL_VULNS critical severity vulnerabilities"
    fi
else
    testing::phase::log "INFO" "No high/critical security vulnerabilities found"
fi
cd ..

# Check for outdated packages
testing::phase::log "INFO" "Checking for outdated packages..."
cd api
OUTDATED_OUTPUT=$(npm outdated 2>&1)
if [ -n "$OUTDATED_OUTPUT" ]; then
    OUTDATED_COUNT=$(echo "$OUTDATED_OUTPUT" | tail -n +2 | wc -l)
    if [ "$OUTDATED_COUNT" -gt 0 ]; then
        testing::phase::log "WARN" "Found $OUTDATED_COUNT outdated packages"
    fi
else
    testing::phase::log "INFO" "All packages are up to date"
fi
cd ..

testing::phase::end_with_summary "Dependency checks completed"
