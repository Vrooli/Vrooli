#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"
cd "$TESTING_PHASE_SCENARIO_DIR"

main() {
    testing::phase::log "INFO" "Validating scenario structure..."

    local required_dirs=("api" "test" "test/phases")
    local missing_dirs=()

    for dir in "${required_dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            missing_dirs+=("$dir")
        fi
    done

    if [ ${#missing_dirs[@]} -gt 0 ]; then
        testing::phase::log "ERROR" "Missing required directories: ${missing_dirs[*]}"
        return 1
    fi

    testing::phase::log "INFO" "All required directories exist"

    local required_files=(
        "api/server.js"
        "api/package.json"
        ".vrooli/service.json"
        "PRD.md"
        "README.md"
    )
    local missing_files=()

    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            missing_files+=("$file")
        fi
    done

    if [ ${#missing_files[@]} -gt 0 ]; then
        testing::phase::log "ERROR" "Missing required files: ${missing_files[*]}"
        return 1
    fi

    testing::phase::log "INFO" "All required files exist"

    testing::phase::log "INFO" "Validating service.json structure..."
    if ! command -v jq >/dev/null 2>&1; then
        testing::phase::log "WARN" "jq not installed, skipping service.json validation"
    else
        local service_json=".vrooli/service.json"
        local required_fields=("service" "version" "lifecycle")

        for field in "${required_fields[@]}"; do
            if ! jq -e ".$field" "$service_json" >/dev/null 2>&1; then
                testing::phase::log "ERROR" "Missing required field in service.json: $field"
                return 1
            fi
        done

        testing::phase::log "INFO" "service.json structure is valid"
    fi

    testing::phase::log "INFO" "Validating package.json structure..."
    local package_json="api/package.json"
    local required_scripts=("start" "test")

    for script in "${required_scripts[@]}"; do
        if ! jq -e ".scripts.$script" "$package_json" >/dev/null 2>&1; then
            testing::phase::log "ERROR" "Missing required npm script: $script"
            return 1
        fi
    done

    testing::phase::log "INFO" "package.json structure is valid"

    testing::phase::log "INFO" "Checking for test files..."
    local test_files
    test_files=$(find api/__tests__ -name "*.test.js" 2>/dev/null | wc -l)
    if [ "$test_files" -eq 0 ]; then
        testing::phase::log "WARN" "No test files found in api/__tests__/"
    else
        testing::phase::log "INFO" "Found $test_files test file(s)"
    fi

    testing::phase::log "INFO" "Validating JavaScript syntax..."
    if command -v node >/dev/null 2>&1; then
        if ! node --check api/server.js >/dev/null 2>&1; then
            testing::phase::log "ERROR" "Syntax error in api/server.js"
            return 1
        fi
        testing::phase::log "INFO" "JavaScript syntax is valid"
    else
        testing::phase::log "WARN" "Node.js not available, skipping syntax validation"
    fi

    testing::phase::log "INFO" "Checking documentation..."
    local doc
    for doc in PRD.md README.md; do
        if [ -f "$doc" ]; then
            if [ ! -s "$doc" ]; then
                testing::phase::log "WARN" "$doc exists but is empty"
            else
                local lines
                lines=$(wc -l < "$doc")
                testing::phase::log "INFO" "$doc: $lines lines"
            fi
        fi
    done

    testing::phase::log "INFO" "Verifying API endpoints..."
    local endpoints_defined
    endpoints_defined=$(grep -c "app\\.\(get\|post\|put\|delete\)" api/server.js || echo "0")
    if [ "$endpoints_defined" -eq 0 ]; then
        testing::phase::log "ERROR" "No API endpoints defined in server.js"
        return 1
    fi

    testing::phase::log "INFO" "Found $endpoints_defined API endpoint(s) defined"

    if ! grep -q "'/health'" api/server.js; then
        testing::phase::log "WARN" "No /health endpoint found - recommended for monitoring"
    else
        testing::phase::log "INFO" "Health check endpoint is defined"
    fi

    return 0
}

if main; then
    testing::phase::end_with_summary "Structure validation completed"
else
    testing::phase::end_with_summary "Structure validation failed"
    exit 1
fi
