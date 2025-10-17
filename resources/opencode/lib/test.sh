#!/bin/bash
# OpenCode Test Functions - Health checks for the official CLI

source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::test::smoke() {
    log::info "Running OpenCode smoke test..."
    opencode::status::check
}

opencode::test::integration() {
    log::info "Running OpenCode integration tests..."
    local failures=0

    opencode::ensure_dirs
    opencode::ensure_config

    log::info "Test 1: CLI availability"
    if opencode::ensure_cli; then
        log::success "✅ OpenCode binary located"
    else
        log::error "❌ OpenCode binary missing"
        return 1
    fi

    log::info "Test 2: --version"
    if version_output=$(opencode::run_cli --version 2>/dev/null); then
        log::success "✅ Version ${version_output}"
    else
        log::error "❌ Failed to execute 'opencode --version'"
        failures=$((failures + 1))
    fi

    log::info "Test 3: non-interactive command (models)"
    if opencode::run_cli models >/dev/null 2>&1; then
        log::success "✅ 'opencode models' executed"
    else
        log::warning "⚠️  'opencode models' failed (verify credentials or network access)"
        failures=$((failures + 1))
    fi

    log::info "Test 4: configuration file"
    if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
        if command -v jq &>/dev/null && ! jq empty "${OPENCODE_CONFIG_FILE}" >/dev/null 2>&1; then
            log::error "❌ Config file is not valid JSON"
            failures=$((failures + 1))
        else
            log::success "✅ Config present at ${OPENCODE_CONFIG_FILE}"
        fi
    else
        log::error "❌ Config file missing"
        failures=$((failures + 1))
    fi

    log::info "Test 5: server lifecycle"
    if opencode::docker::start; then
        log::success "✅ OpenCode server started"
        if opencode::docker::stop; then
            log::success "✅ OpenCode server stopped"
        else
            log::warning "⚠️  Unable to stop OpenCode server cleanly"
            failures=$((failures + 1))
        fi
    else
        log::error "❌ Failed to start OpenCode server"
        failures=$((failures + 1))
    fi

    if [[ ${failures} -eq 0 ]]; then
        log::success "OpenCode integration tests passed"
        return 0
    fi

    log::error "OpenCode integration tests failed (${failures} issue(s))"
    return 1
}

opencode::test::all() {
    log::info "Running all OpenCode tests..."
    local result=0

    if ! opencode::test::smoke; then
        result=1
    fi

    echo

    if ! opencode::test::integration; then
        result=1
    fi

    echo

    if [[ ${result} -eq 0 ]]; then
        log::success "All OpenCode tests passed"
    else
        log::error "Some OpenCode tests failed"
    fi

    return ${result}
}
