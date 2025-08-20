#!/usr/bin/env bash
# Migration Verification Script
# Validates that the document-manager scenario has been properly migrated to modern lifecycle system

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATION_ERRORS=0

check_file_exists() {
    local file="$1"
    local description="$2"
    
    if [[ -f "$SCENARIO_DIR/$file" ]]; then
        log_success "$description exists: $file"
        return 0
    else
        log_error "$description missing: $file"
        ((MIGRATION_ERRORS++))
        return 1
    fi
}

check_file_not_exists() {
    local file="$1"
    local description="$2"
    
    if [[ ! -f "$SCENARIO_DIR/$file" ]] && [[ ! -d "$SCENARIO_DIR/$file" ]]; then
        log_success "$description removed: $file"
        return 0
    else
        log_warning "$description still exists (should be removed): $file"
        ((MIGRATION_ERRORS++))
        return 1
    fi
}

check_json_structure() {
    local file="$1"
    local required_key="$2"
    local description="$3"
    
    if [[ -f "$SCENARIO_DIR/$file" ]]; then
        if jq -e "$required_key" "$SCENARIO_DIR/$file" >/dev/null 2>&1; then
            log_success "$description has correct structure"
            return 0
        else
            log_error "$description missing required key: $required_key"
            ((MIGRATION_ERRORS++))
            return 1
        fi
    else
        log_error "$description file not found: $file"
        ((MIGRATION_ERRORS++))
        return 1
    fi
}

check_yaml_structure() {
    local file="$1"
    local required_key="$2"
    local description="$3"
    
    if [[ -f "$SCENARIO_DIR/$file" ]]; then
        # Convert YAML to JSON and check with jq
        if python3 -c "import yaml, json, sys; json.dump(yaml.safe_load(open('$SCENARIO_DIR/$file')), sys.stdout)" 2>/dev/null | jq -e "$required_key" >/dev/null 2>&1; then
            log_success "$description has correct structure"
            return 0
        else
            log_warning "$description may be missing key: $required_key (YAML check)"
            return 0  # Don't fail for YAML structure issues
        fi
    else
        log_error "$description file not found: $file"
        ((MIGRATION_ERRORS++))
        return 1
    fi
}

main() {
    log_info "=== Document Manager Migration Verification ==="
    log_info "Scenario Directory: $SCENARIO_DIR"
    echo
    
    # Check modern lifecycle structure
    log_info "Checking modern lifecycle structure..."
    check_file_exists ".vrooli/service.json" "Modern service configuration"
    check_json_structure ".vrooli/service.json" ".lifecycle" "Service configuration"
    check_json_structure ".vrooli/service.json" ".resources" "Resource configuration"
    check_json_structure ".vrooli/service.json" ".ports" "Port configuration"
    
    echo
    
    # Check Go API backend
    log_info "Checking Go API backend..."
    check_file_exists "api/main.go" "Go API main file"
    check_file_exists "api/go.mod" "Go module configuration"
    
    echo
    
    # Check CLI wrapper
    log_info "Checking CLI wrapper..."
    check_file_exists "cli/document-manager" "CLI script"
    check_file_exists "cli/install.sh" "CLI installation script"
    
    if [[ -f "$SCENARIO_DIR/cli/document-manager" ]]; then
        if [[ -x "$SCENARIO_DIR/cli/document-manager" ]]; then
            log_success "CLI script is executable"
        else
            log_error "CLI script is not executable"
            ((MIGRATION_ERRORS++))
        fi
    fi
    
    echo
    
    # Check modern test configuration
    log_info "Checking test configuration..."
    check_file_exists "scenario-test.yaml" "Test configuration"
    check_json_structure "scenario-test.yaml" ".metadata.lifecycle_version" "Test lifecycle version"
    
    echo
    
    # Check legacy files removed
    log_info "Checking legacy files removal..."
    check_file_not_exists "deployment/startup.sh" "Legacy startup script"
    check_file_not_exists "deployment" "Legacy deployment directory"
    
    echo
    
    # Check Go API structure
    log_info "Checking Go API structure..."
    if [[ -f "$SCENARIO_DIR/api/main.go" ]]; then
        if grep -q "PORT.*SERVICE_PORT" "$SCENARIO_DIR/api/main.go"; then
            log_success "API uses environment variables for port configuration"
        else
            log_warning "API may not use environment variables correctly"
        fi
        
        if grep -q "RESOURCE_PORTS" "$SCENARIO_DIR/.vrooli/service.json"; then
            log_success "Service configuration uses RESOURCE_PORTS pattern"
        else
            log_warning "Service configuration may not use RESOURCE_PORTS correctly"
        fi
    fi
    
    echo
    
    # Check service.json structure details
    log_info "Checking service.json lifecycle structure..."
    if [[ -f "$SCENARIO_DIR/.vrooli/service.json" ]]; then
        local phases=("setup" "develop" "test" "stop")
        for phase in "${phases[@]}"; do
            if jq -e ".lifecycle.$phase" "$SCENARIO_DIR/.vrooli/service.json" >/dev/null 2>&1; then
                log_success "Lifecycle phase '$phase' defined"
            else
                log_error "Lifecycle phase '$phase' missing"
                ((MIGRATION_ERRORS++))
            fi
        done
        
        # Check for resource injection usage
        if jq -e '.lifecycle.setup.steps[] | select(.run | contains("scripts/resources/injection/engine.sh"))' "$SCENARIO_DIR/.vrooli/service.json" >/dev/null 2>&1; then
            log_success "Uses resource injection engine"
        else
            log_error "Does not use resource injection engine"
            ((MIGRATION_ERRORS++))
        fi
        
        # Check for hardcoded ports (should not exist)
        if jq -r '.lifecycle.develop.steps[].run' "$SCENARIO_DIR/.vrooli/service.json" 2>/dev/null | grep -E ':(5678|5681|6333|5432|6379)' >/dev/null; then
            log_warning "Found hardcoded ports in develop phase (should use environment variables)"
            ((MIGRATION_ERRORS++))
        else
            log_success "No hardcoded ports found in lifecycle configuration"
        fi
    fi
    
    echo
    
    # Summary
    if [[ $MIGRATION_ERRORS -eq 0 ]]; then
        log_success "=== Migration Verification PASSED ==="
        log_info "Document Manager scenario has been successfully migrated to modern lifecycle system"
        echo
        log_info "Next steps:"
        echo "  1. Build the Go API: cd api && go mod download && go build -o document-manager-api main.go"
        echo "  2. Install CLI: cd cli && sudo ./install.sh"
        echo "  3. Test with orchestrator: vrooli scenario run document-manager"
        return 0
    else
        log_error "=== Migration Verification FAILED ==="
        log_error "Found $MIGRATION_ERRORS issues that need to be addressed"
        return 1
    fi
}

main "$@"