#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"
cd "$TESTING_PHASE_SCENARIO_DIR"

main() {
    testing::phase::log "INFO" "Checking scenario dependencies..."

    if ! command -v node >/dev/null 2>&1; then
        testing::phase::log "ERROR" "Node.js is not installed"
        return 1
    fi
    testing::phase::log "INFO" "Node.js version: $(node --version)"

    if ! command -v npm >/dev/null 2>&1; then
        testing::phase::log "ERROR" "npm is not installed"
        return 1
    fi
    testing::phase::log "INFO" "npm version: $(npm --version)"

    if [ ! -f "api/package.json" ]; then
        testing::phase::log "ERROR" "api/package.json not found"
        return 1
    fi

    if [ ! -d "api/node_modules" ]; then
        testing::phase::log "WARN" "api/node_modules not found, installing dependencies..."
        if ! (cd api && npm install >/dev/null 2>&1); then
            testing::phase::log "ERROR" "Failed to install dependencies"
            return 1
        fi
    fi

    testing::phase::log "INFO" "Verifying critical dependencies..."
    local critical_deps=("express" "cors" "@faker-js/faker" "uuid" "joi" "csv-writer" "js2xmlparser")
    local missing_deps=()

    for dep in "${critical_deps[@]}"; do
        if ! (cd api && npm list "$dep" >/dev/null 2>&1); then
            missing_deps+=("$dep")
        fi
    done

    if [ ${#missing_deps[@]} -gt 0 ]; then
        testing::phase::log "ERROR" "Missing critical dependencies: ${missing_deps[*]}"
        return 1
    fi
    testing::phase::log "INFO" "All critical dependencies are installed"

    testing::phase::log "INFO" "Running security audit..."
    local audit_output audit_exit
    audit_output=$(cd api && npm audit --audit-level=high 2>&1 || true)
    audit_exit=$?
    if [ $audit_exit -ne 0 ]; then
        local high_vulns critical_vulns
        high_vulns=$(echo "$audit_output" | grep -oP '\d+(?= high)' | head -1)
        critical_vulns=$(echo "$audit_output" | grep -oP '\d+(?= critical)' | head -1)
        if [ -n "$high_vulns" ] && [ "$high_vulns" -gt 0 ]; then
            testing::phase::log "WARN" "Found $high_vulns high severity vulnerabilities"
        fi
        if [ -n "$critical_vulns" ] && [ "$critical_vulns" -gt 0 ]; then
            testing::phase::log "WARN" "Found $critical_vulns critical severity vulnerabilities"
        fi
    else
        testing::phase::log "INFO" "No high/critical security vulnerabilities found"
    fi

    testing::phase::log "INFO" "Checking for outdated packages..."
    local outdated_output
    outdated_output=$(cd api && npm outdated 2>&1 || true)
    if [ -n "$outdated_output" ]; then
        local outdated_count
        outdated_count=$(echo "$outdated_output" | tail -n +2 | wc -l | tr -d ' ')
        if [ "$outdated_count" -gt 0 ]; then
            testing::phase::log "WARN" "Found $outdated_count outdated packages"
        else
            testing::phase::log "INFO" "All packages are up to date"
        fi
    else
        testing::phase::log "INFO" "All packages are up to date"
    fi

    return 0
}

if main; then
    testing::phase::end_with_summary "Dependency checks completed"
else
    testing::phase::end_with_summary "Dependency checks failed"
    exit 1
fi
