#!/usr/bin/env bash
# Shared Scoping Functions for Test Phases
# 
# Provides consistent scoping logic across all test phases.
# Supports filtering by resource, scenario, or path.

# Build find command with scope filters
# Usage: build_scoped_find <base_path> <file_pattern> [additional_find_args]
# Example: build_scoped_find "." "*.sh" "-type f"
build_scoped_find() {
    local base_path="${1:-.}"
    local file_pattern="$2"
    shift 2
    local additional_args="$*"
    
    local find_cmd="find"
    
    # Determine search paths based on scope
    if [[ -n "${SCOPE_RESOURCE:-}" ]]; then
        # Look in both resources and __test directories for resources
        local paths=(
            "$base_path/resources/*/$SCOPE_RESOURCE"
            "$base_path/resources/$SCOPE_RESOURCE"
            "$base_path/__test/fixtures/resources/$SCOPE_RESOURCE"
        )
        
        # Only add paths that exist
        local valid_paths=()
        for path in "${paths[@]}"; do
            # Expand glob and check if any matches exist
            for expanded in $path; do
                [[ -e "$expanded" ]] && valid_paths+=("$expanded")
            done
        done
        
        if [[ ${#valid_paths[@]} -eq 0 ]]; then
            # No paths found for this resource
            echo "# No paths found for resource: $SCOPE_RESOURCE" >&2
            return 1
        fi
        
        find_cmd="$find_cmd ${valid_paths[*]}"
        
    elif [[ -n "${SCOPE_SCENARIO:-}" ]]; then
        # Look in scenarios directories
        local paths=(
            "$base_path/scenarios/$SCOPE_SCENARIO"
            "$base_path/__test/fixtures/scenarios/$SCOPE_SCENARIO"
        )
        
        # Only add paths that exist
        local valid_paths=()
        for path in "${paths[@]}"; do
            # Expand glob and check if any matches exist
            for expanded in $path; do
                [[ -e "$expanded" ]] && valid_paths+=("$expanded")
            done
        done
        
        if [[ ${#valid_paths[@]} -eq 0 ]]; then
            echo "# No paths found for scenario: $SCOPE_SCENARIO" >&2
            return 1
        fi
        
        find_cmd="$find_cmd ${valid_paths[*]}"
        
    elif [[ -n "${SCOPE_PATH:-}" ]]; then
        # Use specific path
        if [[ ! -e "$base_path/$SCOPE_PATH" ]]; then
            echo "# Path does not exist: $SCOPE_PATH" >&2
            return 1
        fi
        find_cmd="$find_cmd $base_path/$SCOPE_PATH"
        
    else
        # No scope - search everything
        find_cmd="$find_cmd $base_path"
    fi
    
    # Add file pattern and additional arguments
    find_cmd="$find_cmd -name '$file_pattern' $additional_args"
    
    # Return the command for execution
    echo "$find_cmd"
}

# Get scoped files matching a pattern
# Usage: get_scoped_files <file_pattern> [base_path]
# Example: get_scoped_files "*.sh"
get_scoped_files() {
    local file_pattern="$1"
    local base_path="${2:-.}"
    
    local find_cmd
    find_cmd=$(build_scoped_find "$base_path" "$file_pattern" "-type f" 2>/dev/null)
    
    if [[ -z "$find_cmd" ]]; then
        return 1
    fi
    
    # Execute the find command
    eval "$find_cmd" 2>/dev/null | sort | uniq
}

# Check if a file is in scope
# Usage: is_in_scope <file_path>
# Returns 0 if in scope, 1 if not
is_in_scope() {
    local file_path="$1"
    
    # No scope means everything is in scope
    if [[ -z "${SCOPE_RESOURCE:-}" ]] && [[ -z "${SCOPE_SCENARIO:-}" ]] && [[ -z "${SCOPE_PATH:-}" ]]; then
        return 0
    fi
    
    # Check resource scope
    if [[ -n "${SCOPE_RESOURCE:-}" ]]; then
        if [[ "$file_path" =~ /resources/[^/]*/$SCOPE_RESOURCE/ ]] || \
           [[ "$file_path" =~ /resources/$SCOPE_RESOURCE/ ]] || \
           [[ "$file_path" =~ /fixtures/resources/$SCOPE_RESOURCE/ ]]; then
            return 0
        fi
        return 1
    fi
    
    # Check scenario scope
    if [[ -n "${SCOPE_SCENARIO:-}" ]]; then
        if [[ "$file_path" =~ /scenarios/[^/]*/$SCOPE_SCENARIO/ ]] || \
           [[ "$file_path" =~ /scenarios/$SCOPE_SCENARIO/ ]] || \
           [[ "$file_path" =~ /fixtures/scenarios/$SCOPE_SCENARIO/ ]]; then
            return 0
        fi
        return 1
    fi
    
    # Check path scope
    if [[ -n "${SCOPE_PATH:-}" ]]; then
        # Normalize paths for comparison
        local normalized_file
        normalized_file=$(realpath "$file_path" 2>/dev/null || echo "$file_path")
        local normalized_scope
        normalized_scope=$(realpath "$SCOPE_PATH" 2>/dev/null || echo "$SCOPE_PATH")
        
        if [[ "$normalized_file" =~ ^$normalized_scope ]]; then
            return 0
        fi
        return 1
    fi
    
    return 1
}

# Get scope description for logging
# Usage: get_scope_description
# Returns a human-readable scope description
get_scope_description() {
    if [[ -n "${SCOPE_RESOURCE:-}" ]]; then
        echo "resource '$SCOPE_RESOURCE'"
    elif [[ -n "${SCOPE_SCENARIO:-}" ]]; then
        echo "scenario '$SCOPE_SCENARIO'"
    elif [[ -n "${SCOPE_PATH:-}" ]]; then
        echo "path '$SCOPE_PATH'"
    else
        echo "all files"
    fi
}

# Parse scope arguments from command line
# Usage: parse_scope_args "$@"
# Sets SCOPE_RESOURCE, SCOPE_SCENARIO, SCOPE_PATH
parse_scope_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --resource=*)
                export SCOPE_RESOURCE="${1#*=}"
                ;;
            --scenario=*)
                export SCOPE_SCENARIO="${1#*=}"
                ;;
            --path=*)
                export SCOPE_PATH="${1#*=}"
                ;;
        esac
        shift
    done
    
    # Validate only one scope is set
    local scope_count=0
    [[ -n "${SCOPE_RESOURCE:-}" ]] && ((scope_count++))
    [[ -n "${SCOPE_SCENARIO:-}" ]] && ((scope_count++))
    [[ -n "${SCOPE_PATH:-}" ]] && ((scope_count++))
    
    if [[ $scope_count -gt 1 ]]; then
        echo "Error: Only one scope (--resource, --scenario, or --path) can be specified at a time" >&2
        return 1
    fi
    
    return 0
}

# Export functions for use in other scripts
export -f build_scoped_find
export -f get_scoped_files
export -f is_in_scope
export -f get_scope_description
export -f parse_scope_args