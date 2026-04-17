#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
LIB_DEPS_DIR="${APP_ROOT}/scripts/lib/deps"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/retry.sh"

# Function to install ast-grep for structural code search
ast_grep::install() {
    # Check if ast-grep is already installed
    if system::is_command "ast-grep"; then
        log::info "ast-grep is already installed"
        return
    fi

    log::header "🌳 Installing ast-grep for structural code search"

    # Determine the installation directory based on sudo availability
    INSTALL_DIR="/usr/local/bin"
    local use_sudo=true

    # Check if we should use local installation
    if [ "${SUDO_MODE:-error}" = "skip" ] || ! sudo::can_use_sudo; then
        use_sudo=false
        log::info "Installing ast-grep to user directory."
        INSTALL_DIR="$HOME/.local/bin"

        # Ensure the local bin directory exists
        if sudo::is_running_as_sudo; then
            sudo::mkdir_as_user "$INSTALL_DIR"
        else
            mkdir -p "$INSTALL_DIR"
        fi

        # Ensure the local bin directory is in PATH
        if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
            export PATH="$INSTALL_DIR:$PATH"
            log::info "Added $INSTALL_DIR to PATH for current session."
        fi
    fi

    # Determine architecture for static binary
    local arch
    case "$(uname -m)" in
        x86_64) arch="x86_64-unknown-linux-gnu" ;;
        aarch64|arm64) arch="aarch64-unknown-linux-gnu" ;;
        *) 
            log::error "Unsupported architecture: $(uname -m)"
            log::info "ast-grep may be available via cargo install ast-grep if you have Rust installed"
            return 1
            ;;
    esac

    # Build list of candidate download URLs
    local -a versions_to_try=()

    log::info "🔍 Fetching latest ast-grep release information..."
    local latest_release_info=""
    if latest_release_info=$(curl -fsSL --retry 3 --retry-delay 2 "https://api.github.com/repos/ast-grep/ast-grep/releases/latest" 2>/dev/null); then
        local latest_tag
        latest_tag=$(echo "$latest_release_info" | jq -r '.tag_name' 2>/dev/null || echo "")
        if [[ -n "$latest_tag" && "$latest_tag" != "null" ]]; then
            while IFS=$'\t' read -r asset_name asset_url; do
                [[ -z "$asset_url" || "$asset_url" == "null" ]] && continue
                versions_to_try+=("${latest_tag}|${asset_url}")
                log::info "Found release asset ${asset_name} for ${latest_tag}"
            done < <(echo "$latest_release_info" | jq -r --arg arch "$arch" '.assets[] | select(.name | contains($arch)) | [.name, .browser_download_url] | @tsv')
        fi
    else
        log::warning "Could not fetch latest release from GitHub API, using fallback versions"
    fi

    # Add fallback versions covering both historical naming schemes
    local fallback_versions=("0.39.5" "0.39.4" "0.39.3" "0.39.2" "0.39.1" "0.38.4" "0.37.3")
    for version in "${fallback_versions[@]}"; do
        versions_to_try+=("${version}|https://github.com/ast-grep/ast-grep/releases/download/${version}/app-${arch}.zip")
        versions_to_try+=("${version}|https://github.com/ast-grep/ast-grep/releases/download/${version}/ast-grep-${arch}.zip")
    done

    local version_info
    for version_info in "${versions_to_try[@]}"; do
        IFS='|' read -r version download_url <<< "$version_info"
        [[ -z "$download_url" ]] && continue
        log::header "📥 Attempting to download ast-grep ${version} for ${arch}"
        tmpdir=$(mktemp -d)
        
        # Try downloading the release zip file
        if curl -fsSL "$download_url" -o "$tmpdir/ast-grep.zip" && 
           unzip -q "$tmpdir/ast-grep.zip" -d "$tmpdir"; then

            # Find the ast-grep binary (it might be in a subdirectory)
            local ast_grep_binary
            ast_grep_binary=$(find "$tmpdir" -name "ast-grep" -type f -executable | head -1)
            
            if [[ -n "$ast_grep_binary" ]]; then
                # Move and make executable
                if [ "$use_sudo" = "true" ]; then
                    sudo::exec_with_fallback "cp '$ast_grep_binary' '${INSTALL_DIR}/ast-grep'"
                    sudo::exec_with_fallback "chmod +x '${INSTALL_DIR}/ast-grep'"
                else
                    cp "$ast_grep_binary" "${INSTALL_DIR}/ast-grep"
                    chmod +x "${INSTALL_DIR}/ast-grep"
                fi
                
                trash::safe_remove "$tmpdir" --no-confirm
                log::success "ast-grep ${version} installed to ${INSTALL_DIR}"
                
                # Verify installation
                if ast_grep::verify; then
                    return 0
                else
                    log::warning "ast-grep installed but verification failed"
                fi
            else
                log::warning "Could not find ast-grep binary in downloaded archive"
                trash::safe_remove "$tmpdir" --no-confirm
            fi
        else
            log::warning "Download of ast-grep ${version} failed, trying next version"
            trash::safe_remove "$tmpdir" --no-confirm
        fi
    done

    # Final fallback: try npm installation if Node.js is available
    if system::is_command npm; then
        log::info "Attempting fallback installation via npm..."
        if [[ "$use_sudo" == "true" ]]; then
            if npm install -g @ast-grep/cli; then
                log::success "ast-grep installed via npm"
                return 0
            else
                log::warning "npm installation also failed"
            fi
        else
            local npm_prefix="${HOME}/.local"
            mkdir -p "${npm_prefix}/bin"
            if NPM_CONFIG_PREFIX="$npm_prefix" npm install -g @ast-grep/cli; then
                export PATH="${npm_prefix}/bin:$PATH"
                log::success "ast-grep installed via npm (user prefix)"
                return 0
            else
                log::warning "npm installation also failed"
            fi
        fi
    fi

    log::error "All ast-grep installation methods failed"
    log::info "You can try installing manually with:"
    log::info "  • cargo install ast-grep (requires Rust)"
    log::info "  • npm install -g @ast-grep/cli (requires Node.js)"
    log::info "  • Download from https://github.com/ast-grep/ast-grep/releases"
    return 1
}

# Function to check if ast-grep is working properly
ast_grep::verify() {
    if ! system::is_command "ast-grep"; then
        log::error "ast-grep is not installed or not in PATH"
        return 1
    fi

    local version_output
    if version_output=$(ast-grep --version 2>&1); then
        log::success "ast-grep is working: ${version_output}"
        return 0
    else
        log::error "ast-grep is installed but not working properly"
        return 1
    fi
}

# Function to test ast-grep with a simple pattern
ast_grep::test() {
    if ! ast_grep::verify; then
        return 1
    fi

    log::info "Testing ast-grep with a simple pattern..."
    
    # Create a temporary test file
    local test_file
    test_file=$(mktemp --suffix=.js)
    cat > "$test_file" << 'EOF'
function testFunction() {
    console.log("Hello, world!");
    return true;
}

const arrowFunction = () => {
    return "test";
};
EOF

    # Test pattern matching
    if ast-grep --pattern 'function $NAME() { $$$ }' "$test_file" >/dev/null 2>&1; then
        log::success "ast-grep pattern matching test passed"
        trash::safe_remove "$test_file" --no-confirm
        return 0
    else
        log::warning "ast-grep pattern matching test failed"
        trash::safe_remove "$test_file" --no-confirm
        return 1
    fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    ast_grep::install "$@"
fi
