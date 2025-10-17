#!/bin/bash
# Structure tests for Morning Vision Walk scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::step "Verifying directory structure"

# Check required directories
REQUIRED_DIRS=("api" "ui" "cli" "test" "test/phases" ".vrooli" "initialization" "initialization/postgres" "initialization/n8n")

for dir in "${REQUIRED_DIRS[@]}"; do
    if [[ -d "$dir" ]]; then
        testing::phase::success "Directory exists: $dir"
    else
        testing::phase::error "Missing directory: $dir"
    fi
done

testing::phase::step "Verifying configuration files"

# Check required configuration files
REQUIRED_FILES=(
    ".vrooli/service.json"
    "api/main.go"
    "api/go.mod"
    "ui/server.js"
    "ui/package.json"
    "cli/install.sh"
    "initialization/postgres/schema.sql"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        testing::phase::success "File exists: $file"
    else
        testing::phase::error "Missing file: $file"
    fi
done

testing::phase::step "Verifying test infrastructure"

# Check test files
TEST_FILES=(
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-business.sh"
    "test/phases/test-performance.sh"
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
)

for file in "${TEST_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        testing::phase::success "Test file exists: $file"
    else
        testing::phase::warn "Missing test file: $file"
    fi
done

testing::phase::step "Verifying service.json structure"

if [[ -f ".vrooli/service.json" ]]; then
    # Validate JSON structure
    if jq empty .vrooli/service.json 2>/dev/null; then
        testing::phase::success "service.json is valid JSON"

        # Check required fields
        SERVICE_NAME=$(jq -r '.service.name // empty' .vrooli/service.json)
        LIFECYCLE_VERSION=$(jq -r '.lifecycle.version // empty' .vrooli/service.json)

        if [[ "$SERVICE_NAME" == "morning-vision-walk" ]]; then
            testing::phase::success "Service name correct: $SERVICE_NAME"
        else
            testing::phase::error "Invalid service name: $SERVICE_NAME"
        fi

        if [[ "$LIFECYCLE_VERSION" == "2.0.0" ]]; then
            testing::phase::success "Using lifecycle v2.0"
        else
            testing::phase::warn "Lifecycle version: $LIFECYCLE_VERSION"
        fi

        # Check resource configuration
        POSTGRES=$(jq -r '.resources.postgres.enabled // false' .vrooli/service.json)
        REDIS=$(jq -r '.resources.redis.enabled // false' .vrooli/service.json)
        QDRANT=$(jq -r '.resources.qdrant.enabled // false' .vrooli/service.json)
        N8N=$(jq -r '.resources.n8n.enabled // false' .vrooli/service.json)

        testing::phase::info "Resources configured:"
        testing::phase::info "  PostgreSQL: $POSTGRES"
        testing::phase::info "  Redis: $REDIS"
        testing::phase::info "  Qdrant: $QDRANT"
        testing::phase::info "  n8n: $N8N"
    else
        testing::phase::error "service.json is invalid JSON"
    fi
fi

testing::phase::step "Verifying Go module structure"

if [[ -f "api/go.mod" ]]; then
    MODULE_NAME=$(grep "^module " api/go.mod | cut -d' ' -f2)
    testing::phase::success "Go module: $MODULE_NAME"

    # Check for required dependencies
    REQUIRED_DEPS=("github.com/gorilla/mux" "github.com/gorilla/websocket" "github.com/rs/cors")

    for dep in "${REQUIRED_DEPS[@]}"; do
        if grep -q "$dep" api/go.mod; then
            testing::phase::success "Dependency present: $dep"
        else
            testing::phase::warn "Missing dependency: $dep"
        fi
    done
fi

testing::phase::step "Verifying initialization files"

# Check PostgreSQL schema
if [[ -f "initialization/postgres/schema.sql" ]]; then
    TABLE_COUNT=$(grep -c "CREATE TABLE" initialization/postgres/schema.sql || echo "0")
    testing::phase::info "PostgreSQL schema contains $TABLE_COUNT tables"
else
    testing::phase::warn "No PostgreSQL schema file"
fi

# Check n8n workflows
N8N_WORKFLOWS=$(find initialization/n8n -name "*.json" 2>/dev/null | wc -l)
if [[ $N8N_WORKFLOWS -gt 0 ]]; then
    testing::phase::success "Found $N8N_WORKFLOWS n8n workflow(s)"

    # List workflows
    for workflow in initialization/n8n/*.json; do
        if [[ -f "$workflow" ]]; then
            WORKFLOW_NAME=$(basename "$workflow" .json)
            testing::phase::info "  - $WORKFLOW_NAME"
        fi
    done
else
    testing::phase::warn "No n8n workflows found"
fi

testing::phase::step "Verifying code structure"

# Check for main handler functions
if [[ -f "api/main.go" ]]; then
    HANDLERS=(
        "healthHandler"
        "startConversationHandler"
        "sendMessageHandler"
        "endConversationHandler"
        "generateInsightsHandler"
        "prioritizeTasksHandler"
        "gatherContextHandler"
        "getSessionHandler"
        "exportSessionHandler"
        "handleWebSocket"
    )

    for handler in "${HANDLERS[@]}"; do
        if grep -q "func $handler" api/main.go; then
            testing::phase::success "Handler implemented: $handler"
        else
            testing::phase::warn "Handler missing: $handler"
        fi
    done
fi

testing::phase::end_with_summary "Structure verification completed"
