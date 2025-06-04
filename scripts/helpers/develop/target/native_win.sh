#!/usr/bin/env bash
# Posix-compliant script for native Windows development
set -euo pipefail

DEVELOP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/flow.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/log.sh"

nativeWin::start_development_native_win() {
    log::header "Starting native Windows development environment..."

    # Check if running on Windows
    if [[ "$(uname)" != *"NT"* ]]; then
        log::error "This script must be run on Windows"
        exit 1
    fi

    # Ensure correct Node.js version is active
    log::info "Verifying Node.js version..."
    nvm use lts

    # Check and install dependencies if needed
    if [ ! -d "node_modules" ]; then
        log::info "Installing dependencies..."
        pnpm install
    fi

    nativeWin::cleanup() {
        log::info "Cleaning up development environment..."
        kill $TYPE_CHECK_PID $LINT_PID
        exit 0
    }
    if ! flow::is_yes "$DETACHED"; then
        trap nativeWin::cleanup SIGINT SIGTERM
    fi

    # Run TypeScript type checking in watch mode in background
    log::info "Starting TypeScript type checking in watch mode..."
    pnpm run type-check:watch &
    TYPE_CHECK_PID=$!

    # Start ESLint in watch mode in background
    log::info "Starting ESLint in watch mode..."
    pnpm run lint:watch &
    LINT_PID=$!

    # Start development server
    log::info "Starting development server..."
    pnpm run dev

    log::success "Development environment started successfully!"
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    nativeWin::start_development_native_win "$@"
fi
