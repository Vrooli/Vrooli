#!/bin/bash
# Structure validation phase - <15 seconds
# Validates required files, configuration, and directory structure
set -euo pipefail

# Resolve directories early so set -e doesn't kill the script on arithmetic
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${SCENARIO_DIR}/test/utils/cli.sh"

pushd "$SCENARIO_DIR" >/dev/null

echo "=== Structure Phase (Target: <15s) ==="
start_time=$(date +%s)

error_count=0

# Check required files
echo "🔍 Checking required files..."
required_files=(
    ".vrooli/service.json"
    "README.md"
    "PRD.md"
)

missing_files=()
for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        missing_files+=("$file")
        error_count=$((error_count + 1))
    fi
done

if [ ${#missing_files[@]} -gt 0 ]; then
    log::error "❌ Missing required files:"
    printf "   - %s\n" "${missing_files[@]}"
else
    log::success "✅ All required files present"
fi

# Check required directories
echo "🔍 Checking directory structure..."
required_dirs=("api" "cli" "ui" "data" "test")
missing_dirs=()
for dir in "${required_dirs[@]}"; do
    if [ ! -d "$dir" ]; then
        missing_dirs+=("$dir")
        error_count=$((error_count + 1))
    fi
done

if [ ${#missing_dirs[@]} -gt 0 ]; then
    log::error "❌ Missing required directories:"
    printf "   - %s\n" "${missing_dirs[@]}"
else
    log::success "✅ All required directories present"
fi

# Validate service.json schema
echo "🔍 Validating service.json..."
if command -v jq >/dev/null 2>&1; then
    if ! jq empty < .vrooli/service.json >/dev/null 2>&1; then
        log::error "❌ Invalid JSON in service.json"
        error_count=$((error_count + 1))
    else
        log::success "✅ service.json is valid JSON"
        
        # Check required fields
        required_fields=("service.name" "service.version" "ports" "lifecycle")
        for field in "${required_fields[@]}"; do
            if ! jq -e ".$field" < .vrooli/service.json >/dev/null 2>&1; then
                log::error "❌ Missing required field in service.json: $field"
                error_count=$((error_count + 1))
            fi
        done
        
        if [ $error_count -eq 0 ]; then
            service_name=$(jq -r '.service.name' .vrooli/service.json)
            if [ "$service_name" = "visited-tracker" ]; then
                log::success "✅ service.json contains correct service name"
            else
                log::error "❌ Incorrect service name in service.json: $service_name"
                error_count=$((error_count + 1))
            fi
        fi
    fi
else
    log::warning "⚠️  jq not available, skipping JSON validation"
fi

# Check Go module structure
echo "🔍 Validating Go module..."
if [ -f "api/go.mod" ]; then
    if grep -q "module " api/go.mod; then
        log::success "✅ Go module properly defined"
    else
        log::error "❌ Invalid go.mod structure"
        error_count=$((error_count + 1))
    fi
else
    log::error "❌ go.mod missing"
    error_count=$((error_count + 1))
fi

# Check Node.js package.json structure
echo "🔍 Validating Node.js package..."
if [ -f "ui/package.json" ]; then
    if command -v jq >/dev/null 2>&1; then
        if jq -e '.name' ui/package.json >/dev/null 2>&1; then
            package_name=$(jq -r '.name' ui/package.json)
            log::success "✅ Node.js package properly defined: $package_name"
        else
            log::error "❌ Invalid package.json structure"
            error_count=$((error_count + 1))
        fi
    fi
else
    log::error "❌ ui/package.json missing"
    error_count=$((error_count + 1))
fi

# Check CLI tooling can be installed on demand
echo "🔍 Validating CLI tooling..."
if ! visited_tracker::validate_cli "$SCENARIO_DIR" false; then
    cli_errors=$?
    if [ $cli_errors -eq 0 ]; then
        cli_errors=1
    fi
    error_count=$((error_count + cli_errors))
fi

# Check modern test structure
echo "🔍 Validating test infrastructure..."
test_structure_valid=true

if [ ! -f "test/run-tests.sh" ]; then
    log::error "❌ Modern test orchestrator missing: test/run-tests.sh"
    error_count=$((error_count + 1))
    test_structure_valid=false
fi

if [ ! -x "test/run-tests.sh" ] && [ -f "test/run-tests.sh" ]; then
    log::error "❌ Test orchestrator not executable: test/run-tests.sh"
    error_count=$((error_count + 1))
    test_structure_valid=false
fi

required_phases=("test-structure.sh" "test-dependencies.sh" "test-unit.sh")
for phase in "${required_phases[@]}"; do
    if [ ! -f "test/phases/$phase" ]; then
        log::error "❌ Missing test phase: test/phases/$phase"
        error_count=$((error_count + 1))
        test_structure_valid=false
    fi
done

if [ "$test_structure_valid" = "true" ]; then
    log::success "✅ Modern test infrastructure complete"
fi

# Check data directory structure
echo "🔍 Checking data directory..."
if [ -d "data" ]; then
    if [ -d "data/campaigns" ]; then
        log::success "✅ Data directory structure correct"
    else
        log::warning "⚠️  data/campaigns directory missing (will be created on setup)"
    fi
else
    log::warning "⚠️  data directory missing (will be created on setup)"
fi

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))
echo ""
if [ $error_count -eq 0 ]; then
    log::success "✅ Structure validation completed successfully in ${duration}s"
else
    log::error "❌ Structure validation failed with $error_count errors in ${duration}s"
fi

if [ $duration -gt 15 ]; then
    log::warning "⚠️  Structure phase exceeded 15s target"
fi

# Exit with appropriate code
popd >/dev/null

if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi
