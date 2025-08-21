#!/usr/bin/env bash
# Generic CLI Installation Utility
# Used by both resource CLIs and scenario/app CLIs

set -euo pipefail

# Find best installation directory
find_install_dir() {
    local dirs=("$HOME/.local/bin" "$HOME/bin" "/usr/local/bin")
    
    for dir in "${dirs[@]}"; do
        if [[ -d "$dir" ]] && echo "$PATH" | grep -q "$dir"; then
            if [[ -w "$dir" ]] || [[ "$dir" == "/usr/local/bin" ]]; then
                echo "$dir"
                return 0
            fi
        fi
    done
    
    mkdir -p "$HOME/.local/bin"
    echo "$HOME/.local/bin"
}

# Install a CLI to the best available location
# Usage: install_cli <source_path> <cli_name>
install_cli() {
    local source_path="$1"
    local cli_name="$2"
    
    local install_dir=$(find_install_dir)
    local target="$install_dir/$cli_name"
    
    # Install with appropriate permissions
    if [[ -w "$install_dir" ]]; then
        cp "$source_path" "$target"
        chmod +x "$target"
    elif [[ "$install_dir" == "/usr/local/bin" ]] && command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
        sudo cp "$source_path" "$target"
        sudo chmod +x "$target"
    else
        # Fall back to user directory
        install_dir="$HOME/.local/bin"
        mkdir -p "$install_dir"
        target="$install_dir/$cli_name"
        cp "$source_path" "$target"
        chmod +x "$target"
    fi
    
    echo "✅ $cli_name installed to $target"
    
    # Check PATH
    if ! echo "$PATH" | grep -q "$install_dir"; then
        echo "⚠️  Add to PATH: export PATH=\"$install_dir:\$PATH\""
    fi
}