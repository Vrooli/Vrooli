#!/bin/bash

# Script for dependency-aware staged type checking
# This ensures packages are type-checked in the correct order
# to avoid build failures due to missing dependencies

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to log with color
log() {
    local color=$1
    local message=$2
    echo -e "${color}[TYPE-CHECK]${NC} ${message}"
}

# Function to check if a package has TypeScript files
has_ts_files() {
    local package_dir="$1"
    find "$package_dir/src" -name "*.ts" -o -name "*.tsx" 2>/dev/null | head -1 | grep -q .
}

# Function to type-check a single package
type_check_package() {
    local package_name="$1"
    local package_dir="packages/$package_name"
    
    if [[ ! -d "$package_dir" ]]; then
        log "$RED" "Package directory $package_dir not found"
        return 1
    fi
    
    if ! has_ts_files "$package_dir"; then
        log "$YELLOW" "No TypeScript files found in $package_name, skipping..."
        return 0
    fi
    
    log "$BLUE" "Type-checking $package_name..."
    local start_time=$(date +%s)
    
    cd "$package_dir"
    if pnpm run type-check; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log "$GREEN" "✓ $package_name completed in ${duration}s"
        cd - > /dev/null
        return 0
    else
        log "$RED" "✗ $package_name failed"
        cd - > /dev/null
        return 1
    fi
}

# Function to type-check packages in parallel (for packages at same dependency level)
type_check_parallel() {
    local packages=("$@")
    local pids=()
    local results=()
    
    log "$BLUE" "Type-checking packages in parallel: ${packages[*]}"
    
    # Start all packages in background
    for package in "${packages[@]}"; do
        type_check_package "$package" &
        pids+=($!)
        results+=("pending")
    done
    
    # Wait for all to complete and collect results
    local all_success=true
    for i in "${!pids[@]}"; do
        if wait "${pids[$i]}"; then
            results[$i]="success"
        else
            results[$i]="failed"
            all_success=false
        fi
    done
    
    # Report results
    for i in "${!packages[@]}"; do
        if [[ "${results[$i]}" == "success" ]]; then
            log "$GREEN" "✓ ${packages[$i]} (parallel)"
        else
            log "$RED" "✗ ${packages[$i]} (parallel)"
        fi
    done
    
    $all_success
}

# Main execution function
main() {
    log "$BLUE" "Starting dependency-aware type checking..."
    local start_time=$(date +%s)
    local overall_success=true
    
    # Stage 1: Shared package (no dependencies)
    log "$BLUE" "=== Stage 1: Foundation packages ==="
    if ! type_check_package "shared"; then
        overall_success=false
        log "$RED" "Shared package failed - aborting due to dependency chain"
        exit 1
    fi
    
    # Stage 2: Server and Jobs (depend on shared)
    log "$BLUE" "=== Stage 2: Service packages ==="
    if ! type_check_parallel "server" "jobs"; then
        overall_success=false
        log "$RED" "Service packages failed"
        # Don't exit here - UI might still work
    fi
    
    # Stage 3: UI (depends on shared)
    log "$BLUE" "=== Stage 3: Frontend packages ==="
    if ! type_check_package "ui"; then
        overall_success=false
        log "$RED" "UI package failed"
    fi
    
    # Report final results
    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))
    
    echo
    if $overall_success; then
        log "$GREEN" "🎉 All packages type-checked successfully in ${total_duration}s"
        exit 0
    else
        log "$RED" "💥 Some packages failed type checking (completed in ${total_duration}s)"
        exit 1
    fi
}

# Handle command line arguments
case "${1:-}" in
    "--help"|"-h")
        echo "Usage: $0 [--help]"
        echo "Performs dependency-aware type checking of all packages"
        echo "Packages are checked in stages based on their dependencies:"
        echo "  Stage 1: shared"
        echo "  Stage 2: server, jobs (parallel)"
        echo "  Stage 3: ui"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac