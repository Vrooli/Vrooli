#!/bin/bash
# AI Model Orchestra Controller - Structure Phase Tests
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$TEST_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "ℹ️  $1"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }
log_warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }

echo "🏗️  Running structure tests for AI Model Orchestra Controller..."

# Test required files exist
test_required_files() {
    local files=(
        ".vrooli/service.json"
        "PRD.md"
        "Makefile"
        "README.md"
        "api/main.go"
        "api/go.mod"
        "cli/ai-orchestra"
        "cli/install.sh"
        "ui/dashboard.html"
        "ui/dashboard.css"
        "ui/dashboard.js"
        "ui/server.js"
    )
    
    log_info "Checking required files..."
    
    for file in "${files[@]}"; do
        if [ -f "$SCENARIO_ROOT/$file" ]; then
            log_success "Found: $file"
        else
            log_error "Missing required file: $file"
            return 1
        fi
    done
    
    return 0
}

# Test required directories exist
test_required_directories() {
    local dirs=(
        "api"
        "cli" 
        "ui"
        "test"
        "test/phases"
        ".vrooli"
    )
    
    log_info "Checking required directories..."
    
    for dir in "${dirs[@]}"; do
        if [ -d "$SCENARIO_ROOT/$dir" ]; then
            log_success "Found: $dir/"
        else
            log_error "Missing required directory: $dir/"
            return 1
        fi
    done
    
    return 0
}

# Test service.json is valid JSON
test_service_json() {
    local service_json="$SCENARIO_ROOT/.vrooli/service.json"
    
    log_info "Validating service.json..."
    
    if [ ! -f "$service_json" ]; then
        log_error "service.json not found"
        return 1
    fi
    
    if ! jq empty "$service_json" 2>/dev/null; then
        log_error "service.json is not valid JSON"
        return 1
    fi
    
    # Check required fields
    local required_fields=(
        ".service.name"
        ".service.displayName"
        ".service.description"
        ".ports.api"
        ".resources"
        ".lifecycle"
    )
    
    for field in "${required_fields[@]}"; do
        if ! jq -e "$field" "$service_json" >/dev/null 2>&1; then
            log_error "Missing required field in service.json: $field"
            return 1
        fi
    done
    
    log_success "service.json is valid"
    return 0
}

# Test Go mod is valid
test_go_mod() {
    local go_mod="$SCENARIO_ROOT/api/go.mod"
    
    log_info "Validating Go module..."
    
    if [ ! -f "$go_mod" ]; then
        log_error "go.mod not found"
        return 1
    fi
    
    # Check module name
    if ! grep -q "module ai-model-orchestra-controller" "$go_mod"; then
        log_error "go.mod has incorrect module name"
        return 1
    fi
    
    # Check Go version
    if ! grep -q "go 1\.[0-9][0-9]" "$go_mod"; then
        log_error "go.mod missing or invalid Go version"
        return 1
    fi
    
    log_success "go.mod is valid"
    return 0
}

# Test CLI script is executable
test_cli_executable() {
    local cli_script="$SCENARIO_ROOT/cli/ai-orchestra"
    
    log_info "Checking CLI executable..."
    
    if [ ! -f "$cli_script" ]; then
        log_error "CLI script not found"
        return 1
    fi
    
    if [ ! -x "$cli_script" ]; then
        log_error "CLI script is not executable"
        return 1
    fi
    
    # Check shebang
    if ! head -1 "$cli_script" | grep -q "#!/bin/bash"; then
        log_error "CLI script missing bash shebang"
        return 1
    fi
    
    log_success "CLI script is executable"
    return 0
}

# Test UI files are properly modularized
test_ui_modular() {
    local ui_dir="$SCENARIO_ROOT/ui"
    
    log_info "Checking UI modularization..."
    
    # Check that dashboard.html references external files
    if grep -q "<style>" "$ui_dir/dashboard.html"; then
        log_error "dashboard.html contains embedded styles (should be external)"
        return 1
    fi
    
    if grep -q "<script>" "$ui_dir/dashboard.html"; then
        if ! grep -q "dashboard.js" "$ui_dir/dashboard.html"; then
            log_error "dashboard.html contains embedded scripts (should be external)"
            return 1
        fi
    fi
    
    # Check external files exist and have content
    if [ ! -s "$ui_dir/dashboard.css" ]; then
        log_error "dashboard.css is missing or empty"
        return 1
    fi
    
    if [ ! -s "$ui_dir/dashboard.js" ]; then
        log_error "dashboard.js is missing or empty"
        return 1
    fi
    
    log_success "UI is properly modularized"
    return 0
}

# Test PRD exists and has content
test_prd_content() {
    local prd_file="$SCENARIO_ROOT/PRD.md"
    
    log_info "Checking PRD content..."
    
    if [ ! -f "$prd_file" ]; then
        log_error "PRD.md not found"
        return 1
    fi
    
    # Check for key PRD sections
    local required_sections=(
        "Capability Definition"
        "Success Metrics"
        "Technical Architecture"
        "API Contract"
        "CLI Interface Contract"
    )
    
    for section in "${required_sections[@]}"; do
        if ! grep -q "$section" "$prd_file"; then
            log_error "PRD missing required section: $section"
            return 1
        fi
    done
    
    log_success "PRD has required content"
    return 0
}

# Test Makefile has required targets
test_makefile() {
    local makefile="$SCENARIO_ROOT/Makefile"
    
    log_info "Checking Makefile..."
    
    if [ ! -f "$makefile" ]; then
        log_error "Makefile not found"
        return 1
    fi
    
    # Check for required targets
    local required_targets=(
        "run:"
        "stop:"
        "test:"
        "build:"
        "clean:"
        "help:"
    )
    
    for target in "${required_targets[@]}"; do
        if ! grep -q "^$target" "$makefile"; then
            log_error "Makefile missing required target: $target"
            return 1
        fi
    done
    
    log_success "Makefile has required targets"
    return 0
}

# Run all structure tests
echo "Starting structure validation tests..."

# Execute all tests
test_required_files || exit 1
test_required_directories || exit 1
test_service_json || exit 1
test_go_mod || exit 1
test_cli_executable || exit 1
test_ui_modular || exit 1
test_prd_content || exit 1
test_makefile || exit 1

echo ""
log_success "All structure tests passed!"
echo "✅ Structure phase completed successfully"