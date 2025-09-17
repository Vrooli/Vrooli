visited_tracker::validate_cli() {
    local scenario_dir="$1"
    local ensure_install="${2:-false}"
    local install_script="${scenario_dir}/cli/install.sh"
    local cli_binary="${scenario_dir}/cli/visited-tracker"
    local errors=0

    if [ ! -f "$install_script" ]; then
        log::error "❌ CLI install script missing: $install_script"
        return 1
    fi

    if [ ! -x "$install_script" ]; then
        log::error "❌ CLI install script is not executable: $install_script"
        errors=$((errors + 1))
    else
        log::success "✅ CLI install script available"
    fi

    if [ -f "$cli_binary" ]; then
        if [ ! -x "$cli_binary" ]; then
            log::warning "⚠️  CLI entrypoint exists but is not executable; attempting to fix permissions"
            chmod +x "$cli_binary" || true
        fi
    else
        log::warning "⚠️  CLI entrypoint missing; it can be installed dynamically"
    fi

    if command -v visited-tracker >/dev/null 2>&1; then
        log::success "✅ visited-tracker CLI already installed"
    elif [ "$ensure_install" = "true" ]; then
        if "$install_script" >/dev/null 2>&1; then
            if command -v visited-tracker >/dev/null 2>&1; then
                log::success "✅ visited-tracker CLI installed for tests"
            else
                log::error "❌ CLI installation script ran but CLI is still unavailable"
                errors=$((errors + 1))
            fi
        else
            log::error "❌ CLI installation script failed"
            errors=$((errors + 1))
        fi
    else
        log::warning "⚠️  visited-tracker CLI not yet installed; dependencies phase will install"
    fi

    return $errors
}
