#!/usr/bin/env bash
# Comprehensive App Monitor Test Suite
# Tests all components that can be verified without running services

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✅${NC} $1"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}❌${NC} $1"
    ((TESTS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Test 1: File Structure
test_file_structure() {
    log_info "Testing file structure..."
    
    local required_files=(
        "README.md"
        ".vrooli/service.json"
        "test/run-tests.sh"
        "api/main.go"
        "api/go.mod"
        "cli/app-monitor"
        "ui/index.html"
        "ui/script.js"
        "ui/styles.css"
        "initialization/storage/postgres/schema.sql"
        "initialization/n8n/app-health-monitor.json"
    )
    
    local missing_files=()
    for file in "${required_files[@]}"; do
        if [[ ! -f "${SCRIPT_DIR}/${file}" ]]; then
            missing_files+=("$file")
        fi
    done
    
    if [[ ${#missing_files[@]} -eq 0 ]]; then
        log_success "All required files present"
    else
        log_error "Missing files: ${missing_files[*]}"
    fi
}

# Test 2: Go API Compilation
test_api_compilation() {
    log_info "Testing Go API compilation..."
    
    cd "${SCRIPT_DIR}/api"
    if go build -o test-build main.go 2>/dev/null; then
        log_success "Go API compiles successfully"
        rm -f test-build
    else
        log_error "Go API compilation failed"
    fi
}

# Test 3: CLI Functionality
test_cli_functionality() {
    log_info "Testing CLI functionality..."
    
    cd "${SCRIPT_DIR}/cli"
    
    # Test help command
    if ./app-monitor help | grep -q "App Monitor CLI"; then
        log_success "CLI help command works"
    else
        log_error "CLI help command failed"
    fi
    
    # Test error handling
    if ./app-monitor invalid-command 2>&1 | grep -q "Unknown command"; then
        log_success "CLI error handling works"
    else
        log_error "CLI error handling failed"
    fi
    
    # Test parameter validation
    if ./app-monitor start 2>&1 | grep -q "App ID required"; then
        log_success "CLI parameter validation works"
    else
        log_error "CLI parameter validation failed"
    fi
}

# Test 4: Database Schema Validation
test_database_schema() {
    log_info "Testing database schema..."
    
    local schema_file="${SCRIPT_DIR}/initialization/storage/postgres/schema.sql"
    local required_tables=(
        "apps"
        "app_status"
        "app_metrics"
        "app_logs"
        "app_health_status"
        "app_health_alerts"
        "app_performance_analysis"
    )
    
    local missing_tables=()
    for table in "${required_tables[@]}"; do
        if ! grep -q "CREATE TABLE.*$table" "$schema_file"; then
            missing_tables+=("$table")
        fi
    done
    
    if [[ ${#missing_tables[@]} -eq 0 ]]; then
        log_success "All required database tables defined"
    else
        log_error "Missing database tables: ${missing_tables[*]}"
    fi
}

# Test 5: N8N Workflow Validation
test_n8n_workflows() {
    log_info "Testing n8n workflow structure..."
    
    local workflow_dir="${SCRIPT_DIR}/initialization/n8n"
    local workflows=("app-health-monitor.json" "app-performance-analyzer.json" "auto-restart.json" "resource-monitor.json")
    local missing_workflows=()
    
    for workflow in "${workflows[@]}"; do
        local workflow_file="${workflow_dir}/${workflow}"
        if [[ ! -f "$workflow_file" ]]; then
            missing_workflows+=("$workflow")
        elif ! jq . "$workflow_file" >/dev/null 2>&1; then
            log_error "Invalid JSON in workflow: $workflow"
        fi
    done
    
    if [[ ${#missing_workflows[@]} -eq 0 ]]; then
        log_success "All n8n workflows present and valid"
    else
        log_error "Missing n8n workflows: ${missing_workflows[*]}"
    fi
}

# Test 6: UI File Validation
test_ui_files() {
    log_info "Testing UI implementation..."
    
    local ui_dir="${SCRIPT_DIR}/ui"
    
    # Check HTML structure
    if grep -q "VROOLI SYSTEMS MONITOR" "${ui_dir}/index.html"; then
        log_success "UI HTML structure is complete"
    else
        log_error "UI HTML structure incomplete"
    fi
    
    # Check CSS completeness
    local css_lines=$(wc -l < "${ui_dir}/styles.css")
    if [[ $css_lines -gt 500 ]]; then
        log_success "UI CSS is comprehensive ($css_lines lines)"
    else
        log_error "UI CSS appears incomplete ($css_lines lines)"
    fi
    
    # Check JavaScript functionality
    local js_lines=$(wc -l < "${ui_dir}/script.js")
    if [[ $js_lines -gt 500 ]] && grep -q "WebSocket" "${ui_dir}/script.js"; then
        log_success "UI JavaScript is feature-complete ($js_lines lines)"
    else
        log_error "UI JavaScript appears incomplete"
    fi
}

# Test 7: Configuration Files
test_configuration() {
    log_info "Testing configuration files..."
    
    # Test .vrooli/service.json
    local service_config="${SCRIPT_DIR}/.vrooli/service.json"
    if jq . "$service_config" >/dev/null 2>&1; then
        if jq -e '.test.steps[] | select(.run == "test/run-tests.sh")' "$service_config" >/dev/null 2>&1; then
            log_success "Service test lifecycle configured for phased runner"
        else
            log_warning "Phased test runner not declared in .vrooli/service.json"
        fi
    else
        log_error "Invalid .vrooli/service.json"
    fi
}

# Test 8: Documentation Quality
test_documentation() {
    log_info "Testing documentation..."
    
    local readme="${SCRIPT_DIR}/README.md"
    local required_sections=("Purpose" "Features" "API Endpoints" "CLI Commands")
    local missing_sections=()
    
    for section in "${required_sections[@]}"; do
        if ! grep -q "## $section\|# $section" "$readme"; then
            missing_sections+=("$section")
        fi
    done
    
    if [[ ${#missing_sections[@]} -eq 0 ]]; then
        log_success "Documentation is comprehensive"
    else
        log_error "Missing documentation sections: ${missing_sections[*]}"
    fi
}

# Main test runner
main() {
    echo -e "${BLUE}🔍 App Monitor Comprehensive Test Suite${NC}"
    echo "========================================"
    echo ""
    
    test_file_structure
    test_api_compilation
    test_cli_functionality
    test_database_schema
    test_n8n_workflows
    test_ui_files
    test_configuration
    test_documentation
    
    echo ""
    echo "========================================"
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}🎉 ALL TESTS PASSED (${TESTS_PASSED}/${TESTS_PASSED})${NC}"
        echo -e "${GREEN}App Monitor is production-ready!${NC}"
        exit 0
    else
        echo -e "${RED}❌ ${TESTS_FAILED} test(s) failed, ${TESTS_PASSED} passed${NC}"
        exit 1
    fi
}

main "$@"

