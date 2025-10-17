#!/usr/bin/env bash
########################################
# Test Conditional Workflow Features
########################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSERLESS_DIR="${APP_ROOT}/resources/browserless"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test output directory
TEST_OUTPUT="/tmp/browserless-conditional-test-$(date +%Y%m%d_%H%M%S)"
mkdir -p "$TEST_OUTPUT"

########################################
# Test basic condition evaluation
########################################
test_basic_conditions() {
    log::info "Testing basic condition evaluation"
    
    # Create a simple test workflow
    cat > "$TEST_OUTPUT/test-conditions.yaml" << 'EOF'
workflow:
  name: "test-conditions"
  description: "Test basic conditional operations"
  version: "1.0.0"
  
  parameters:
    test_url:
      type: string
      default: "https://example.com"
  
  steps:
    - name: "navigate"
      action: "navigate"
      url: "${params.test_url}"
      
    - name: "test-url-condition"
      action: "condition"
      condition_type: "url"
      url_pattern: "example.com"
      then_steps:
        - action: "log"
          message: "URL condition passed: On example.com"
      else_steps:
        - action: "log"
          message: "URL condition failed: Not on example.com"
    
    - name: "test-element-condition"
      action: "condition"
      condition_type: "element_visible"
      selector: "h1"
      then_steps:
        - action: "extract_text"
          selector: "h1"
          variable: "heading_text"
        - action: "log"
          message: "Found heading: {{heading_text}}"
      else_steps:
        - action: "log"
          message: "No h1 element found"
    
    - name: "test-text-condition"
      action: "condition"
      condition_type: "element_text"
      selector: "h1"
      text: "Example"
      match_type: "contains"
      then_steps:
        - action: "log"
          message: "Heading contains 'Example'"
        - action: "screenshot"
          path: "${test_output}/example-page.png"
      else_steps:
        - action: "log"
          message: "Heading does not contain 'Example'"
EOF
    
    # Run the test workflow
    if "${BROWSERLESS_DIR}/lib/workflow/interpreter.sh" "$TEST_OUTPUT/test-conditions.yaml" \
        --param test_output="$TEST_OUTPUT" 2>&1 | tee "$TEST_OUTPUT/conditions.log"; then
        log::success "Basic conditions test passed"
        
        # Verify log contains expected messages
        if grep -q "URL condition passed" "$TEST_OUTPUT/conditions.log"; then
            log::success "URL condition evaluated correctly"
        else
            log::error "URL condition did not evaluate as expected"
            return 1
        fi
    else
        log::error "Basic conditions test failed"
        return 1
    fi
}

########################################
# Test nested conditions
########################################
test_nested_conditions() {
    log::info "Testing nested conditional branching"
    
    cat > "$TEST_OUTPUT/test-nested.yaml" << 'EOF'
workflow:
  name: "test-nested"
  description: "Test nested conditions"
  version: "1.0.0"
  
  steps:
    - name: "navigate"
      action: "navigate"
      url: "https://httpbin.org/html"
      
    - name: "outer-condition"
      action: "condition"
      condition_type: "url"
      url_pattern: "httpbin.org"
      then_steps:
        - action: "log"
          message: "On httpbin.org - checking for content"
          
        - action: "condition"
          condition_type: "element_visible"
          selector: "h1"
          then_steps:
            - action: "log"
              message: "Found h1 in httpbin page"
            - action: "evaluate"
              script: "return { nested: true, level: 2 };"
              output: "${test_output}/nested-result.json"
          else_steps:
            - action: "log"
              message: "No h1 found in httpbin page"
      else_steps:
        - action: "log"
          message: "Not on httpbin.org"
EOF
    
    if "${BROWSERLESS_DIR}/lib/workflow/interpreter.sh" "$TEST_OUTPUT/test-nested.yaml" \
        --param test_output="$TEST_OUTPUT" 2>&1 | tee "$TEST_OUTPUT/nested.log"; then
        log::success "Nested conditions test passed"
        
        # Check if nested result was created
        if [[ -f "$TEST_OUTPUT/nested-result.json" ]]; then
            log::success "Nested condition executed inner branch"
        else
            log::warning "Nested result file not created"
        fi
    else
        log::error "Nested conditions test failed"
        return 1
    fi
}

########################################
# Test error detection
########################################
test_error_detection() {
    log::info "Testing error detection conditions"
    
    cat > "$TEST_OUTPUT/test-errors.yaml" << 'EOF'
workflow:
  name: "test-errors"
  description: "Test error detection"
  version: "1.0.0"
  
  steps:
    - name: "navigate-to-404"
      action: "navigate"
      url: "https://httpbin.org/status/404"
      
    - name: "check-404"
      action: "condition"
      condition_type: "element_text"
      selector: "body"
      text: "404"
      match_type: "contains"
      then_steps:
        - action: "log"
          message: "404 error detected successfully"
        - action: "evaluate"
          script: "return { error_detected: true, code: 404 };"
          output: "${test_output}/error-detection.json"
      else_steps:
        - action: "log"
          message: "404 not detected"
    
    - name: "navigate-to-success"
      action: "navigate"
      url: "https://httpbin.org/status/200"
      
    - name: "check-no-error"
      action: "condition"
      condition_type: "has_errors"
      error_patterns: "error,Error,404,500"
      then_steps:
        - action: "log"
          message: "Unexpected error found on success page"
      else_steps:
        - action: "log"
          message: "No errors on success page - correct"
EOF
    
    if "${BROWSERLESS_DIR}/lib/workflow/interpreter.sh" "$TEST_OUTPUT/test-errors.yaml" \
        --param test_output="$TEST_OUTPUT" 2>&1 | tee "$TEST_OUTPUT/errors.log"; then
        log::success "Error detection test passed"
        
        # Verify error was detected
        if [[ -f "$TEST_OUTPUT/error-detection.json" ]]; then
            local error_code=$(jq -r '.code' "$TEST_OUTPUT/error-detection.json")
            if [[ "$error_code" == "404" ]]; then
                log::success "404 error correctly identified"
            fi
        fi
    else
        log::error "Error detection test failed"
        return 1
    fi
}

########################################
# Test sub-workflow with conditions
########################################
test_subworkflow_conditions() {
    log::info "Testing sub-workflows with conditions"
    
    cat > "$TEST_OUTPUT/test-subworkflow.yaml" << 'EOF'
workflow:
  name: "test-subworkflow"
  description: "Test sub-workflows with conditions"
  version: "1.0.0"
  
  sub_workflows:
    check_content:
      steps:
        - name: "check-title"
          action: "evaluate"
          script: "return document.title;"
          variable: "page_title"
          
        - name: "verify-title"
          action: "condition"
          condition: "{{page_title}}.length > 0"
          condition_type: "javascript"
          then_steps:
            - action: "log"
              message: "Page has title: {{page_title}}"
          else_steps:
            - action: "log"
              message: "Page has no title"
  
  steps:
    - name: "navigate"
      action: "navigate"
      url: "https://example.com"
      
    - name: "check-page-loaded"
      action: "condition"
      condition_type: "element_visible"
      selector: "body"
      then_steps:
        - action: "call_sub_workflow"
          sub_workflow: "check_content"
        - action: "log"
          message: "Content check completed"
      else_steps:
        - action: "log"
          message: "Page failed to load"
EOF
    
    if "${BROWSERLESS_DIR}/lib/workflow/interpreter.sh" "$TEST_OUTPUT/test-subworkflow.yaml" \
        --param test_output="$TEST_OUTPUT" 2>&1 | tee "$TEST_OUTPUT/subworkflow.log"; then
        log::success "Sub-workflow conditions test passed"
    else
        log::error "Sub-workflow conditions test failed"
        return 1
    fi
}

########################################
# Main test runner
########################################
main() {
    log::info "Starting conditional workflow tests"
    log::info "Test output directory: $TEST_OUTPUT"
    
    local failed=0
    
    # Check if browserless is running
    if ! curl -sf http://localhost:3000/health > /dev/null 2>&1; then
        log::warning "Browserless not running, starting it..."
        vrooli resource browserless develop &
        sleep 5
    fi
    
    # Run tests
    test_basic_conditions || ((failed++))
    test_nested_conditions || ((failed++))
    test_error_detection || ((failed++))
    test_subworkflow_conditions || ((failed++))
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All conditional workflow tests passed!"
        log::info "Test artifacts saved to: $TEST_OUTPUT"
        return 0
    else
        log::error "$failed test(s) failed"
        log::info "Check logs in: $TEST_OUTPUT"
        return 1
    fi
}

# Run tests if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi