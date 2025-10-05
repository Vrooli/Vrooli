#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "INFO" "Validating scenario structure..."

# Check required directories
REQUIRED_DIRS=("api" "test" "test/phases")
MISSING_DIRS=()

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ ! -d "$dir" ]; then
        MISSING_DIRS+=("$dir")
    fi
done

if [ ${#MISSING_DIRS[@]} -gt 0 ]; then
    testing::phase::log "ERROR" "Missing required directories: ${MISSING_DIRS[*]}"
    exit 1
fi

testing::phase::log "INFO" "All required directories exist"

# Check required files
REQUIRED_FILES=(
    "api/server.js"
    "api/package.json"
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
)

MISSING_FILES=()

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        MISSING_FILES+=("$file")
    fi
done

if [ ${#MISSING_FILES[@]} -gt 0 ]; then
    testing::phase::log "ERROR" "Missing required files: ${MISSING_FILES[*]}"
    exit 1
fi

testing::phase::log "INFO" "All required files exist"

# Validate service.json structure
testing::phase::log "INFO" "Validating service.json structure..."

if ! command -v jq &> /dev/null; then
    testing::phase::log "WARN" "jq not installed, skipping service.json validation"
else
    SERVICE_JSON=".vrooli/service.json"

    # Check for required fields
    REQUIRED_FIELDS=("name" "version" "lifecycle")

    for field in "${REQUIRED_FIELDS[@]}"; do
        if ! jq -e ".$field" "$SERVICE_JSON" &> /dev/null; then
            testing::phase::log "ERROR" "Missing required field in service.json: $field"
            exit 1
        fi
    done

    testing::phase::log "INFO" "service.json structure is valid"
fi

# Validate package.json structure
testing::phase::log "INFO" "Validating package.json structure..."

PACKAGE_JSON="api/package.json"

# Check for required npm scripts
REQUIRED_SCRIPTS=("start" "test")

for script in "${REQUIRED_SCRIPTS[@]}"; do
    if ! jq -e ".scripts.$script" "$PACKAGE_JSON" &> /dev/null; then
        testing::phase::log "ERROR" "Missing required npm script: $script"
        exit 1
    fi
done

testing::phase::log "INFO" "package.json structure is valid"

# Check for test files
testing::phase::log "INFO" "Checking for test files..."

TEST_FILES=$(find api/__tests__ -name "*.test.js" 2>/dev/null | wc -l)

if [ "$TEST_FILES" -eq 0 ]; then
    testing::phase::log "WARN" "No test files found in api/__tests__/"
else
    testing::phase::log "INFO" "Found $TEST_FILES test file(s)"
fi

# Validate JavaScript syntax (if available)
testing::phase::log "INFO" "Validating JavaScript syntax..."

if command -v node &> /dev/null; then
    # Check server.js syntax
    if ! node --check api/server.js 2>&1; then
        testing::phase::log "ERROR" "Syntax error in api/server.js"
        exit 1
    fi

    testing::phase::log "INFO" "JavaScript syntax is valid"
else
    testing::phase::log "WARN" "Node.js not available, skipping syntax validation"
fi

# Check for documentation
testing::phase::log "INFO" "Checking documentation..."

DOC_FILES=("PRD.md" "README.md")
for doc in "${DOC_FILES[@]}"; do
    if [ -f "$doc" ]; then
        # Check if file is not empty
        if [ ! -s "$doc" ]; then
            testing::phase::log "WARN" "$doc exists but is empty"
        else
            LINES=$(wc -l < "$doc")
            testing::phase::log "INFO" "$doc: $LINES lines"
        fi
    fi
done

# Check API endpoints are defined
testing::phase::log "INFO" "Verifying API endpoints..."

ENDPOINTS_DEFINED=$(grep -c "app\.\(get\|post\|put\|delete\)" api/server.js || echo "0")

if [ "$ENDPOINTS_DEFINED" -eq 0 ]; then
    testing::phase::log "ERROR" "No API endpoints defined in server.js"
    exit 1
else
    testing::phase::log "INFO" "Found $ENDPOINTS_DEFINED API endpoint(s) defined"
fi

# Verify health check endpoint exists
if ! grep -q "'/health'" api/server.js; then
    testing::phase::log "WARN" "No /health endpoint found - recommended for monitoring"
else
    testing::phase::log "INFO" "Health check endpoint is defined"
fi

testing::phase::end_with_summary "Structure validation completed"
