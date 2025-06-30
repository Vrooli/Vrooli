#!/bin/bash

# Script to type-check only changed TypeScript files
# This provides massive performance improvements for incremental changes

set -e

# Function to check if we're in git repo
check_git_repo() {
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        echo "Error: Not in a git repository"
        exit 1
    fi
}

# Function to get changed TypeScript files
get_changed_files() {
    local compare_ref="${1:-HEAD~1}"
    git diff --name-only --diff-filter=ACMR "$compare_ref" HEAD | grep -E '\.(ts|tsx)$' || true
}

# Function to get staged TypeScript files
get_staged_files() {
    git diff --cached --name-only --diff-filter=ACMR | grep -E '\.(ts|tsx)$' || true
}

# Function to type-check specific files in their package context
type_check_files() {
    local files=("$@")
    local packages_checked=()
    
    for file in "${files[@]}"; do
        if [[ -f "$file" ]]; then
            # Determine which package the file belongs to
            if [[ "$file" == packages/server/* ]]; then
                if [[ ! " ${packages_checked[*]} " =~ " server " ]]; then
                    echo "Type-checking server package (contains: $file)..."
                    cd packages/server && NODE_OPTIONS='--max-old-space-size=4096' tsc --noEmit "$file"
                    packages_checked+=("server")
                fi
            elif [[ "$file" == packages/ui/* ]]; then
                if [[ ! " ${packages_checked[*]} " =~ " ui " ]]; then
                    echo "Type-checking UI package (contains: $file)..."
                    cd packages/ui && NODE_OPTIONS='--max-old-space-size=3072' tsc --noEmit "$file"
                    packages_checked+=("ui")
                fi
            elif [[ "$file" == packages/shared/* ]]; then
                if [[ ! " ${packages_checked[*]} " =~ " shared " ]]; then
                    echo "Type-checking shared package (contains: $file)..."
                    cd packages/shared && tsc --noEmit "$file"
                    packages_checked+=("shared")
                fi
            elif [[ "$file" == packages/jobs/* ]]; then
                if [[ ! " ${packages_checked[*]} " =~ " jobs " ]]; then
                    echo "Type-checking jobs package (contains: $file)..."
                    cd packages/jobs && NODE_OPTIONS='--max-old-space-size=3072' tsc --noEmit "$file"
                    packages_checked+=("jobs")
                fi
            fi
        fi
    done
    
    if [[ ${#packages_checked[@]} -eq 0 ]]; then
        echo "No TypeScript files to check."
        return 0
    fi
    
    echo "Type-checked packages: ${packages_checked[*]}"
}

# Main execution
main() {
    check_git_repo
    
    local mode="${1:-changed}"
    local files=()
    
    case "$mode" in
        "staged")
            echo "Checking staged TypeScript files..."
            readarray -t files < <(get_staged_files)
            ;;
        "changed"|*)
            echo "Checking changed TypeScript files (since HEAD~1)..."
            readarray -t files < <(get_changed_files)
            ;;
    esac
    
    if [[ ${#files[@]} -eq 0 ]]; then
        echo "No changed TypeScript files found."
        exit 0
    fi
    
    echo "Found ${#files[@]} changed TypeScript files:"
    printf '  %s\n' "${files[@]}"
    echo
    
    type_check_files "${files[@]}"
}

# Handle command line arguments
case "${1:-}" in
    "--staged")
        main "staged"
        ;;
    "--help"|"-h")
        echo "Usage: $0 [--staged|--help]"
        echo "  --staged    Check only staged files"
        echo "  (default)   Check files changed since HEAD~1"
        exit 0
        ;;
    *)
        main "changed"
        ;;
esac