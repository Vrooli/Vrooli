#!/usr/bin/env bash
# Resource Dependencies Extractor
# Extracts dependency information from various package management files
#
# Supports multiple package managers:
# - Node.js (package.json)
# - Python (requirements.txt, Pipfile, pyproject.toml)
# - Go (go.mod)
# - Ruby (Gemfile)
# - Rust (Cargo.toml)
# - PHP (composer.json)

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract Node.js dependencies
# 
# Parses package.json for npm/yarn dependencies
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with Node.js dependencies
#######################################
qdrant::extract::nodejs_dependencies() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    if [[ ! -f "$dir/package.json" ]]; then
        return 1
    fi
    
    log::debug "Extracting Node.js dependencies for $resource_name" >&2
    
    # Extract dependency objects
    local deps_json=$(jq -r '.dependencies // {}' "$dir/package.json" 2>/dev/null)
    local dev_deps_json=$(jq -r '.devDependencies // {}' "$dir/package.json" 2>/dev/null)
    local peer_deps_json=$(jq -r '.peerDependencies // {}' "$dir/package.json" 2>/dev/null)
    
    # Count dependencies
    local dep_count=$(echo "$deps_json" | jq 'length')
    local dev_dep_count=$(echo "$dev_deps_json" | jq 'length')
    local peer_dep_count=$(echo "$peer_deps_json" | jq 'length')
    
    # Get key packages (top 10)
    local key_packages_json=$(echo "$deps_json" | jq -r 'keys | .[0:10]')
    
    # Extract package metadata
    local package_name=$(jq -r '.name // empty' "$dir/package.json")
    local package_version=$(jq -r '.version // empty' "$dir/package.json")
    local package_license=$(jq -r '.license // empty' "$dir/package.json")
    
    # Build content
    local content="Resource: $resource_name | Type: Dependencies | Platform: Node.js"
    [[ -n "$package_name" ]] && content="$content | Package: $package_name@$package_version"
    content="$content | Dependencies: $dep_count | DevDependencies: $dev_dep_count"
    [[ $dep_count -gt 0 ]] && content="$content | Key packages: $(echo "$key_packages_json" | jq -r 'join(", ")')"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg resource "$resource_name" \
        --arg source_file "$dir/package.json" \
        --arg platform "nodejs" \
        --arg package_name "$package_name" \
        --arg package_version "$package_version" \
        --arg package_license "$package_license" \
        --argjson dependencies "$deps_json" \
        --argjson dev_dependencies "$dev_deps_json" \
        --argjson peer_dependencies "$peer_deps_json" \
        --argjson key_packages "$key_packages_json" \
        --arg dep_count "$dep_count" \
        --arg dev_dep_count "$dev_dep_count" \
        --arg peer_dep_count "$peer_dep_count" \
        '{
            content: $content,
            metadata: {
                resource: $resource,
                source_file: $source_file,
                component_type: "dependencies",
                platform: $platform,
                package_name: $package_name,
                package_version: $package_version,
                package_license: $package_license,
                dependencies: $dependencies,
                dev_dependencies: $dev_dependencies,
                peer_dependencies: $peer_dependencies,
                key_packages: $key_packages,
                dependency_count: ($dep_count | tonumber),
                dev_dependency_count: ($dev_dep_count | tonumber),
                peer_dependency_count: ($peer_dep_count | tonumber),
                content_type: "resource_dependencies",
                extraction_method: "package_json_parser"
            }
        }' | jq -c
}

#######################################
# Extract Python dependencies
# 
# Parses requirements.txt, Pipfile, or pyproject.toml
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with Python dependencies
#######################################
qdrant::extract::python_dependencies() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    # Check for various Python dependency files
    local dep_file=""
    local dep_format=""
    
    if [[ -f "$dir/requirements.txt" ]]; then
        dep_file="$dir/requirements.txt"
        dep_format="requirements"
    elif [[ -f "$dir/Pipfile" ]]; then
        dep_file="$dir/Pipfile"
        dep_format="pipfile"
    elif [[ -f "$dir/pyproject.toml" ]]; then
        dep_file="$dir/pyproject.toml"
        dep_format="pyproject"
    else
        return 1
    fi
    
    log::debug "Extracting Python dependencies from $dep_format for $resource_name" >&2
    
    local deps_json="[]"
    local dep_count=0
    
    case "$dep_format" in
        requirements)
            # Parse requirements.txt
            local deps_raw=$(grep -E "^[a-zA-Z]" "$dep_file" 2>/dev/null | sed 's/[<>=!].*//')
            if [[ -n "$deps_raw" ]]; then
                deps_json=$(echo "$deps_raw" | jq -R . | jq -s .)
                dep_count=$(echo "$deps_json" | jq 'length')
            fi
            ;;
        pipfile)
            # Parse Pipfile (basic parsing)
            local deps_raw=$(grep -A 100 '^\[packages\]' "$dep_file" 2>/dev/null | \
                grep -E '^[a-z]' | cut -d= -f1 | tr -d ' "')
            if [[ -n "$deps_raw" ]]; then
                deps_json=$(echo "$deps_raw" | jq -R . | jq -s .)
                dep_count=$(echo "$deps_json" | jq 'length')
            fi
            ;;
        pyproject)
            # Parse pyproject.toml (basic parsing)
            local deps_raw=$(grep -A 20 'dependencies = \[' "$dep_file" 2>/dev/null | \
                grep '"' | sed 's/[",].*//' | tr -d ' ')
            if [[ -n "$deps_raw" ]]; then
                deps_json=$(echo "$deps_raw" | jq -R . | jq -s .)
                dep_count=$(echo "$deps_json" | jq 'length')
            fi
            ;;
    esac
    
    # Get key packages (top 10)
    local key_packages_json=$(echo "$deps_json" | jq '.[0:10]')
    
    # Build content
    local content="Resource: $resource_name | Type: Dependencies | Platform: Python"
    content="$content | Format: $dep_format | Dependencies: $dep_count"
    [[ $dep_count -gt 0 ]] && content="$content | Key packages: $(echo "$key_packages_json" | jq -r 'join(", ")')"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg resource "$resource_name" \
        --arg source_file "$dep_file" \
        --arg platform "python" \
        --arg format "$dep_format" \
        --argjson dependencies "$deps_json" \
        --argjson key_packages "$key_packages_json" \
        --arg dep_count "$dep_count" \
        '{
            content: $content,
            metadata: {
                resource: $resource,
                source_file: $source_file,
                component_type: "dependencies",
                platform: $platform,
                format: $format,
                dependencies: $dependencies,
                key_packages: $key_packages,
                dependency_count: ($dep_count | tonumber),
                content_type: "resource_dependencies",
                extraction_method: "python_parser"
            }
        }' | jq -c
}

#######################################
# Extract Go dependencies
# 
# Parses go.mod for Go modules
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with Go dependencies
#######################################
qdrant::extract::go_dependencies() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    if [[ ! -f "$dir/go.mod" ]]; then
        return 1
    fi
    
    log::debug "Extracting Go dependencies for $resource_name" >&2
    
    # Extract module name
    local module_name=$(grep "^module " "$dir/go.mod" | cut -d' ' -f2)
    
    # Extract Go version
    local go_version=$(grep "^go " "$dir/go.mod" | cut -d' ' -f2)
    
    # Extract dependencies
    local deps_json="[]"
    local deps_raw=$(awk '/^require \(/, /^\)/ { if ($1 != "require" && $1 != ")" && $1 != "") print $1 }' "$dir/go.mod" 2>/dev/null)
    
    if [[ -z "$deps_raw" ]]; then
        # Try single-line requires
        deps_raw=$(grep "^require " "$dir/go.mod" | cut -d' ' -f2)
    fi
    
    if [[ -n "$deps_raw" ]]; then
        deps_json=$(echo "$deps_raw" | jq -R . | jq -s .)
    fi
    
    local dep_count=$(echo "$deps_json" | jq 'length')
    
    # Build content
    local content="Resource: $resource_name | Type: Dependencies | Platform: Go"
    content="$content | Module: $module_name"
    [[ -n "$go_version" ]] && content="$content | Go: $go_version"
    content="$content | Dependencies: $dep_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg resource "$resource_name" \
        --arg source_file "$dir/go.mod" \
        --arg platform "go" \
        --arg module_name "$module_name" \
        --arg go_version "$go_version" \
        --argjson dependencies "$deps_json" \
        --arg dep_count "$dep_count" \
        '{
            content: $content,
            metadata: {
                resource: $resource,
                source_file: $source_file,
                component_type: "dependencies",
                platform: $platform,
                module_name: $module_name,
                go_version: $go_version,
                dependencies: $dependencies,
                dependency_count: ($dep_count | tonumber),
                content_type: "resource_dependencies",
                extraction_method: "go_mod_parser"
            }
        }' | jq -c
}

#######################################
# Extract Rust dependencies
# 
# Parses Cargo.toml for Rust crates
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with Rust dependencies
#######################################
qdrant::extract::rust_dependencies() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    if [[ ! -f "$dir/Cargo.toml" ]]; then
        return 1
    fi
    
    log::debug "Extracting Rust dependencies for $resource_name" >&2
    
    # Extract package name and version (basic parsing)
    local package_name=$(grep "^name = " "$dir/Cargo.toml" | head -1 | sed 's/.*= *"//' | tr -d '"')
    local package_version=$(grep "^version = " "$dir/Cargo.toml" | head -1 | sed 's/.*= *"//' | tr -d '"')
    
    # Extract dependencies (basic parsing)
    local deps_raw=$(awk '/^\[dependencies\]/, /^\[/' "$dir/Cargo.toml" | \
        grep -E '^[a-z]' | cut -d= -f1 | tr -d ' ')
    
    local deps_json="[]"
    if [[ -n "$deps_raw" ]]; then
        deps_json=$(echo "$deps_raw" | jq -R . | jq -s .)
    fi
    
    local dep_count=$(echo "$deps_json" | jq 'length')
    
    # Build content
    local content="Resource: $resource_name | Type: Dependencies | Platform: Rust"
    [[ -n "$package_name" ]] && content="$content | Package: $package_name@$package_version"
    content="$content | Dependencies: $dep_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg resource "$resource_name" \
        --arg source_file "$dir/Cargo.toml" \
        --arg platform "rust" \
        --arg package_name "$package_name" \
        --arg package_version "$package_version" \
        --argjson dependencies "$deps_json" \
        --arg dep_count "$dep_count" \
        '{
            content: $content,
            metadata: {
                resource: $resource,
                source_file: $source_file,
                component_type: "dependencies",
                platform: $platform,
                package_name: $package_name,
                package_version: $package_version,
                dependencies: $dependencies,
                dependency_count: ($dep_count | tonumber),
                content_type: "resource_dependencies",
                extraction_method: "cargo_parser"
            }
        }' | jq -c
}

#######################################
# Extract all dependency information
# 
# Main function that calls all dependency extractors
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON lines for all dependency information
#######################################
qdrant::extract::resource_dependencies_all() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    # Extract Node.js dependencies
    qdrant::extract::nodejs_dependencies "$dir" 2>/dev/null || true
    
    # Extract Python dependencies
    qdrant::extract::python_dependencies "$dir" 2>/dev/null || true
    
    # Extract Go dependencies
    qdrant::extract::go_dependencies "$dir" 2>/dev/null || true
    
    # Extract Rust dependencies
    qdrant::extract::rust_dependencies "$dir" 2>/dev/null || true
}

# Export functions for use by resources.sh
export -f qdrant::extract::nodejs_dependencies
export -f qdrant::extract::python_dependencies
export -f qdrant::extract::go_dependencies
export -f qdrant::extract::rust_dependencies
export -f qdrant::extract::resource_dependencies_all