#!/bin/bash
# Structure test phase - validates project structure
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "20s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Checking project structure..."

# Required directories
required_dirs=(
    "api"
    "ui"
    "cli"
    "initialization"
    "initialization/postgres"
    "initialization/n8n"
    "initialization/qdrant"
    "test"
    "test/phases"
    ".vrooli"
)

for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        log::success "Directory $dir exists"
    else
        testing::phase::add_error "Required directory $dir missing"
    fi
done

# Required files
required_files=(
    ".vrooli/service.json"
    "Makefile"
    "PRD.md"
    "README.md"
    "api/main.go"
    "api/go.mod"
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
    "initialization/postgres/schema.sql"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        log::success "File $file exists"
    else
        testing::phase::add_error "Required file $file missing"
    fi
done

# Check service.json structure
log::info "Validating service.json..."
if jq -e '.service.name' .vrooli/service.json &>/dev/null; then
    service_name=$(jq -r '.service.name' .vrooli/service.json)
    if [ "$service_name" = "stream-of-consciousness-analyzer" ]; then
        log::success "service.json has correct service name"
    else
        testing::phase::add_error "service.json service name mismatch"
    fi
else
    testing::phase::add_error "service.json is invalid"
fi

# Check for required resources
log::info "Checking required resources..."
required_resources=("n8n" "postgres" "redis" "qdrant" "ollama")

for resource in "${required_resources[@]}"; do
    if jq -e ".resources.$resource" .vrooli/service.json &>/dev/null; then
        log::success "Resource $resource configured"
    else
        testing::phase::add_error "Resource $resource not configured"
    fi
done

# Check lifecycle configuration
log::info "Checking lifecycle configuration..."
if jq -e '.lifecycle.version' .vrooli/service.json &>/dev/null; then
    lifecycle_version=$(jq -r '.lifecycle.version' .vrooli/service.json)
    if [ "$lifecycle_version" = "2.0.0" ]; then
        log::success "Lifecycle version 2.0.0 configured"
    else
        log::warn "Lifecycle version is $lifecycle_version (expected 2.0.0)"
    fi
else
    testing::phase::add_error "Lifecycle configuration missing"
fi

testing::phase::end_with_summary "Structure tests completed"
