#!/usr/bin/env bash
################################################################################
# Vrooli CLI Installation Script
#
# Installs the Go-based 'vrooli' CLI into ~/.vrooli/bin.
#
# Usage:
#   ./install.sh [--uninstall]
#
################################################################################

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)}"
VROOLI_ROOT="${APP_ROOT}"
INSTALL_DIR="${HOME}/.vrooli/bin"
INSTALLED_BINARY="${INSTALL_DIR}/vrooli"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_info() { echo -e "$1"; }

# Check if directory is in PATH
check_path() {
	local dir="$1"
	# Match only full path segments separated by ':'
	if echo "$PATH" | grep -qE "(^|:)$dir(:|$)"; then
		return 0
	fi
	return 1
}

# Add directory to PATH in shell config
add_to_path() {
	local dir="$1"
	local shell_config=""
	
	# Determine shell configuration file
	if [[ -n "${BASH_VERSION:-}" ]]; then
		if [[ -f "$HOME/.bashrc" ]]; then
			shell_config="$HOME/.bashrc"
		elif [[ -f "$HOME/.bash_profile" ]]; then
			shell_config="$HOME/.bash_profile"
		fi
	elif [[ -n "${ZSH_VERSION:-}" ]]; then
		shell_config="$HOME/.zshrc"
	fi
	
	if [[ -n "$shell_config" ]]; then
		# Check if PATH export already exists (path-segment safe)
		if ! grep -qE "(^|:)${dir}(:|$)" "$shell_config" 2>/dev/null; then
			echo "" >> "$shell_config"
			echo "# Added by Vrooli CLI installer" >> "$shell_config"
			echo "export PATH=\"$dir:\$PATH\"" >> "$shell_config"
			print_success "Added $dir to PATH in $shell_config"
			print_info "   Run 'source $shell_config' or open a new terminal"
		fi
	else
		print_warning "Could not determine shell configuration file"
		print_info "   Add this to your shell configuration:"
		print_info "   export PATH=\"$dir:\$PATH\""
	fi
}

# Check if CLI is already installed and up-to-date
check_existing_installation() {
	[[ -x "$INSTALLED_BINARY" ]]
}

is_legacy_bash_install() {
	local location="$1"
	[[ -L "$location" ]] || return 1

	local resolved
	resolved=$(readlink -f "$location" 2>/dev/null || true)
	[[ -n "$resolved" ]] || return 1
	[[ "$resolved" == */cli/vrooli ]]
}

remove_legacy_installations() {
	local locations=(
		"$HOME/.local/bin/vrooli"
		"$HOME/bin/vrooli"
		"/usr/local/bin/vrooli"
	)
	local removed=false

	for location in "${locations[@]}"; do
		if is_legacy_bash_install "$location"; then
			if [[ -w "${location%/*}" ]]; then
				print_info "Removing legacy Bash CLI install: $location"
				rm -f "$location"
				removed=true
			else
				print_warning "Legacy Bash CLI install remains at $location (directory not writable)"
			fi
		fi
	done

	if [[ "$removed" == "true" ]]; then
		print_info "Removed legacy Bash entrypoints so ~/.vrooli/bin/vrooli can take precedence"
	fi
}

# Install the CLI
install_cli() {
	local force="${1:-false}"

	if [[ ! -f "$APP_ROOT/go.mod" ]]; then
		print_error "Go module not found: $APP_ROOT/go.mod"
		exit 1
	fi

	print_info "🚀 Installing Vrooli CLI..."
	echo ""
	print_info "Installation directory: $INSTALL_DIR"

	mkdir -p "$INSTALL_DIR"
	remove_legacy_installations

	if [[ "$force" != "true" ]] && check_existing_installation; then
		print_info "Refreshing project-level Go binaries..."
	fi

	if ! command -v make >/dev/null 2>&1; then
		print_error "make is required to install the Go CLI"
		exit 1
	fi

	(
		cd "$APP_ROOT"
		make install
	)

	if [[ ! -x "$INSTALLED_BINARY" ]]; then
		print_error "Failed to install Go CLI binary at $INSTALLED_BINARY"
		exit 1
	fi

	print_success "Installed Go CLI binary: $INSTALLED_BINARY"

	# Check if directory is in PATH
	if ! check_path "$INSTALL_DIR"; then
		print_warning "$INSTALL_DIR is not in your PATH"
		add_to_path "$INSTALL_DIR"
	fi

	# Set VROOLI_ROOT environment variable
	local shell_config=""
	if [[ -n "${BASH_VERSION:-}" ]]; then
		shell_config="${HOME}/.bashrc"
	elif [[ -n "${ZSH_VERSION:-}" ]]; then
		shell_config="${HOME}/.zshrc"
	fi
	
	if [[ -n "$shell_config" ]] && [[ -f "$shell_config" ]]; then
		if ! grep -q "export VROOLI_ROOT=" "$shell_config" 2>/dev/null; then
			echo "" >> "$shell_config"
			echo "# Vrooli CLI root directory" >> "$shell_config"
			echo "export VROOLI_ROOT=\"$VROOLI_ROOT\"" >> "$shell_config"
			print_success "Added VROOLI_ROOT to $shell_config"
		fi
	fi
	
	echo ""
	print_success "✅ Vrooli CLI installed successfully!"
	echo ""

	# Test if command works from the current shell
	if check_path "$INSTALL_DIR" && command -v vrooli >/dev/null 2>&1 && [[ "$(readlink -f "$(command -v vrooli)" 2>/dev/null || true)" == "$(readlink -f "$INSTALLED_BINARY")" ]]; then
		print_success "The 'vrooli' command is ready to use!"
		echo ""
		echo "Try these commands:"
		echo "  vrooli --help           # Show help"
		echo "  vrooli --version        # Show version"
		echo "  vrooli scenario list    # List available scenarios"
	else
		print_warning "The Go-based 'vrooli' command is not yet active in this shell session"
		echo ""
		echo "To start using it:"
		if [[ -n "$shell_config" ]]; then
			echo "  source $shell_config"
		else
			echo "  export PATH=\"\\$PATH:$INSTALL_DIR\""
		fi
		echo "  Or open a new terminal"
	fi
}

# Uninstall the CLI
uninstall_cli() {
	print_info "🗑️  Uninstalling Vrooli CLI..."
	echo ""
	
	local removed=false
	
	# Check current and legacy locations
	local locations=(
		"$HOME/.vrooli/bin/vrooli"
		"$HOME/.local/bin/vrooli"
		"$HOME/bin/vrooli"
		"/usr/local/bin/vrooli"
	)
	
	for location in "${locations[@]}"; do
		if [[ -L "$location" ]] || [[ -f "$location" ]]; then
			print_info "Removing: $location"
			rm -f "$location"
			removed=true
		fi
	done
	
	if [[ "$removed" == "true" ]]; then
		print_success "✅ Vrooli CLI uninstalled"
		echo ""
		echo "Note: Environment variables (VROOLI_ROOT) and PATH modifications"
		echo "were not removed from your shell configuration files."
		echo "You can remove them manually if desired."
	else
		print_warning "No Vrooli CLI installation found"
	fi
}

# Main execution
main() {
	case "${1:-}" in
		--uninstall|-u)
			uninstall_cli
			;;
		--force|-f)
			install_cli true
			;;
		--help|-h)
			cat << EOF
Vrooli CLI Installation Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --force, -f         Force reinstallation even if already up-to-date
    --uninstall, -u     Uninstall the Vrooli CLI
    --help, -h          Show this help message

DESCRIPTION:
    This script installs the Go-based 'vrooli' binary into ~/.vrooli/bin.
    
    Features:
    • Builds the project-level Go binaries via 'make install'
    • Installs the canonical CLI into ~/.vrooli/bin
    • Adds ~/.vrooli/bin to PATH if needed
    • Removes legacy Bash symlinks that would shadow the Go binary
    • Sets VROOLI_ROOT environment variable

EOF
			;;
		"")
			install_cli false
			;;
		*)
			print_error "Unknown option: $1"
			echo "Run '$0 --help' for usage information"
			exit 1
			;;
	esac
}

# Run main function
main "$@"
