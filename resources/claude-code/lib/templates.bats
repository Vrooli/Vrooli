#!/usr/bin/env bats
# Tests for Claude Code templates.sh functions
bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure
source "${APP_ROOT}/__test/fixtures/setup.bash"

setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "claude-code-templates"
    
    # Load dependencies
    SCRIPT_DIR="${APP_ROOT}/resources/claude-code"
    
    # shellcheck disable=SC1091
    source "${APP_ROOT}/lib/utils/var.sh" || true
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh" || true
    
    # Load configuration if available
    if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
        source "${SCRIPT_DIR}/config/defaults.sh"
    fi
    
    # Load the template functions
    source "${BATS_TEST_DIRNAME}/templates.sh"
    
    # Export functions for subshells
    export -f claude_code::templates_list
    export -f claude_code::template_load
    export -f claude_code::template_run
    export -f claude_code::template_create
    export -f claude_code::template_info
    export -f claude_code::template_validate
}

setup() {
    # Reset mocks for each test
    mock::claude::reset >/dev/null 2>&1 || true
    
    # Create temporary test templates directory
    export TEST_TEMPLATES_DIR="${BATS_TMPDIR}/templates-test"
    export TEMPLATES_DIR="$TEST_TEMPLATES_DIR/prompts"
    mkdir -p "$TEMPLATES_DIR"
}

teardown() {
    # Clean up test files
    trash::safe_remove "$TEST_TEMPLATES_DIR" --test-cleanup
}

#######################################
# Templates List Tests
#######################################

@test "claude_code::templates_list - should handle missing templates directory" {
    # Remove templates directory to test handling
    trash::safe_remove "$TEMPLATES_DIR" --test-cleanup
    
    run claude_code::templates_list "text"
    
    assert_failure
    assert_output --partial "No templates directory found"
}

@test "claude_code::templates_list - should return empty JSON array for missing directory" {
    # Remove templates directory to test JSON handling
    trash::safe_remove "$TEMPLATES_DIR" --test-cleanup
    
    run claude_code::templates_list "json"
    
    assert_failure
    assert_output "[]"
}

@test "claude_code::templates_list - should list templates in text format" {
    # Create test template files
    echo "# Code Review Template" > "$TEMPLATES_DIR/code-review.md"
    echo "# Security Audit Template" > "$TEMPLATES_DIR/security-audit.md"
    
    run claude_code::templates_list "text"
    
    assert_success
    assert_output --partial "code-review"
    assert_output --partial "security-audit"
}

@test "claude_code::templates_list - should list templates in JSON format" {
    # Create test template files
    echo "# Refactoring Template" > "$TEMPLATES_DIR/refactoring.md"
    
    run claude_code::templates_list "json"
    
    assert_success
    assert_output --partial "refactoring"
    # Should be valid JSON
    run bash -c "echo '$output' | jq ."
    assert_success
}

@test "claude_code::templates_list - should handle empty templates directory" {
    # Templates directory exists but is empty
    
    run claude_code::templates_list "text"
    
    assert_success
    assert_output --partial "No templates found"
}

#######################################
# Template Load Tests
#######################################

@test "claude_code::template_load - should require template name" {
    run claude_code::template_load "" 
    
    assert_failure
    assert_output --partial "Template name is required"
}

@test "claude_code::template_load - should handle missing template file" {
    run claude_code::template_load "nonexistent-template"
    
    assert_failure
    assert_output --partial "Template not found"
}

@test "claude_code::template_load - should load valid template" {
    # Create a test template
    cat > "$TEMPLATES_DIR/test-template.md" << 'EOF'
# Test Template
Please analyze the code: {{CODE}}
Focus on: {{FOCUS_AREA}}
EOF
    
    run claude_code::template_load "test-template" "CODE=function test() {}" "FOCUS_AREA=security"
    
    assert_success
    assert_output --partial "Template loaded"
    assert_output --partial "function test() {}"
    assert_output --partial "security"
}

@test "claude_code::template_load - should handle template without variables" {
    # Create a simple template without variables
    cat > "$TEMPLATES_DIR/simple-template.md" << 'EOF'
# Simple Template
Please review this code and provide feedback.
EOF
    
    run claude_code::template_load "simple-template"
    
    assert_success
    assert_output --partial "Please review this code"
}

#######################################
# Template Run Tests
#######################################

@test "claude_code::template_run - should require template name" {
    run claude_code::template_run "" "5" ""
    
    assert_failure
    assert_output --partial "Template name is required"
}

@test "claude_code::template_run - should execute template with Claude" {
    # Create a test template
    cat > "$TEMPLATES_DIR/run-template.md" << 'EOF'
# Executable Template
Create a function that: {{TASK}}
EOF
    
    # Mock successful Claude execution
    mock::claude::scenario::setup_development
    
    run claude_code::template_run "run-template" "5" "Read,Write" "TASK=adds two numbers"
    
    assert_success
    assert_output --partial "Executing template"
}

@test "claude_code::template_run - should handle authentication errors" {
    # Create a test template
    cat > "$TEMPLATES_DIR/auth-template.md" << 'EOF'
# Auth Test Template
Test template content
EOF
    
    # Mock authentication error
    mock::claude::scenario::setup_auth_required
    
    run claude_code::template_run "auth-template" "5" ""
    
    assert_failure
    assert_output --partial "Authentication"
}

#######################################
# Template Create Tests
#######################################

@test "claude_code::template_create - should require template name" {
    run claude_code::template_create "" "Test description"
    
    assert_failure
    assert_output --partial "Template name is required"
}

@test "claude_code::template_create - should create new template interactively" {
    # Mock user input for template creation
    # This is a simplified test - real implementation might use more complex input
    
    run claude_code::template_create "new-template" "Custom template description"
    
    assert_success
    assert_output --partial "Creating template"
    assert_output --partial "new-template"
}

@test "claude_code::template_create - should handle existing template name" {
    # Create existing template
    echo "# Existing Template" > "$TEMPLATES_DIR/existing.md"
    
    run claude_code::template_create "existing" "Description"
    
    assert_failure
    assert_output --partial "already exists"
}

#######################################
# Template Info Tests
#######################################

@test "claude_code::template_info - should require template name" {
    run claude_code::template_info ""
    
    assert_failure
    assert_output --partial "Template name is required"
}

@test "claude_code::template_info - should show info for existing template" {
    # Create a test template with metadata
    cat > "$TEMPLATES_DIR/info-template.md" << 'EOF'
# Info Template
<!-- Description: Template for testing info functionality -->
<!-- Variables: CODE, LANGUAGE -->
<!-- Tags: testing, info -->

Please analyze this {{LANGUAGE}} code:
{{CODE}}
EOF
    
    run claude_code::template_info "info-template"
    
    assert_success
    assert_output --partial "Template Information"
    assert_output --partial "info-template"
}

@test "claude_code::template_info - should handle missing template" {
    run claude_code::template_info "missing-template"
    
    assert_failure
    assert_output --partial "Template not found"
}

#######################################
# Template Validate Tests
#######################################

@test "claude_code::template_validate - should require template name" {
    run claude_code::template_validate ""
    
    assert_failure
    assert_output --partial "Template name is required"
}

@test "claude_code::template_validate - should validate template syntax" {
    # Create a valid template
    cat > "$TEMPLATES_DIR/valid-template.md" << 'EOF'
# Valid Template
This is a valid template with {{VARIABLE}}.
EOF
    
    run claude_code::template_validate "valid-template"
    
    assert_success
    assert_output --partial "Template is valid"
}

@test "claude_code::template_validate - should detect invalid template syntax" {
    # Create an invalid template (malformed variables)
    cat > "$TEMPLATES_DIR/invalid-template.md" << 'EOF'
# Invalid Template
This template has malformed {{UNCLOSED_VARIABLE syntax.
EOF
    
    run claude_code::template_validate "invalid-template"
    
    assert_failure
    assert_output --partial "validation errors"
}

@test "claude_code::template_validate - should handle missing template" {
    run claude_code::template_validate "missing-template"
    
    assert_failure
    assert_output --partial "Template not found"
}