#!/bin/bash
# OpenCode Test Functions - Health checks and integration tests

# Source common functions
source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::test::smoke() {
    # Basic health check - same as status check
    log::info "Running OpenCode smoke test..."
    opencode::status::check
}

opencode::test::integration() {
    log::info "Running OpenCode integration tests..."
    local test_result=0
    
    # Test 1: VS Code availability
    log::info "Test 1: VS Code availability"
    if [[ -n "${VSCODE_COMMAND}" ]]; then
        log::success "✅ VS Code command available: ${VSCODE_COMMAND}"
    else
        log::error "❌ VS Code command not found"
        test_result=1
    fi
    
    # Test 2: Extension installation
    log::info "Test 2: Extension installation"
    if opencode_is_installed; then
        local version=$(opencode_get_version)
        log::success "✅ Twinny extension installed: v${version}"
    else
        log::error "❌ Twinny extension not installed"
        test_result=1
    fi
    
    # Test 3: Configuration file
    log::info "Test 3: Configuration file"
    if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
        if command -v jq &>/dev/null && jq . "${OPENCODE_CONFIG_FILE}" >/dev/null 2>&1; then
            log::success "✅ Configuration file is valid JSON"
        else
            log::error "❌ Configuration file exists but is not valid JSON"
            test_result=1
        fi
    else
        log::warning "⚠️  Configuration file not found (this is OK for fresh installs)"
    fi
    
    # Test 4: Available models
    log::info "Test 4: Model availability"
    local models=$(opencode_get_models)
    local model_count=$(echo "${models}" | jq '. | length')
    if [[ "${model_count}" -gt 0 ]]; then
        log::success "✅ Found ${model_count} available models"
    else
        log::warning "⚠️  No models found (install Ollama or configure OpenRouter)"
    fi
    
    # Test 5: Data directory structure
    log::info "Test 5: Data directory structure"
    local required_dirs=(
        "${OPENCODE_DATA_DIR}"
        "${OPENCODE_DATA_DIR}/logs"
        "${OPENCODE_DATA_DIR}/cache"
    )
    
    local missing_dirs=0
    for dir in "${required_dirs[@]}"; do
        if [[ -d "${dir}" ]]; then
            log::success "✅ Directory exists: ${dir}"
        else
            log::warning "⚠️  Directory missing: ${dir}"
            missing_dirs=$((missing_dirs + 1))
        fi
    done
    
    if [[ $missing_dirs -eq 0 ]]; then
        log::success "✅ All required directories exist"
    else
        log::info "⚠️  ${missing_dirs} directories missing (will be created on first use)"
    fi
    
    # Summary
    if [[ $test_result -eq 0 ]]; then
        log::success "OpenCode integration tests passed"
    else
        log::error "OpenCode integration tests failed"
    fi
    
    return $test_result
}

opencode::test::all() {
    log::info "Running all OpenCode tests..."
    local overall_result=0
    
    # Run smoke test
    if ! opencode::test::smoke; then
        overall_result=1
    fi
    
    echo ""
    
    # Run integration test
    if ! opencode::test::integration; then
        overall_result=1
    fi
    
    echo ""
    
    # Summary
    if [[ $overall_result -eq 0 ]]; then
        log::success "All OpenCode tests passed"
    else
        log::error "Some OpenCode tests failed"
    fi
    
    return $overall_result
}