#!/usr/bin/env bash
################################################################################
# Setup Condition: UI Bundle Check
#
# Checks if UI production bundle is outdated compared to source files
# Used for scenarios that serve pre-built UI bundles (not dev servers)
#
# Checks performed:
# 1. Bundle file exists (dist/index.html or similar)
# 2. Bundle is newer than all source files in src/
# 3. Bundle is newer than config files (package.json, vite.config.ts, etc.)
# 4. Bundle is newer than shared package dependencies referenced via file: specs
#
# Input: JSON object with optional fields:
#   - bundle_path (default: ui/dist/index.html)
#   - source_dir (default: ui/src)
#   - watch_file_dependencies (default: true)
#   - dependency_excludes (optional glob array against resolved dependency paths)
# Returns: 0 if setup needed (bundle missing/outdated), 1 if bundle current
################################################################################

set -euo pipefail

# Get check configuration from argument
CHECK_CONFIG="${1:-'{}'}"

# Extract configuration with defaults
BUNDLE_PATH=$(echo "$CHECK_CONFIG" | jq -r '.bundle_path // "ui/dist/index.html"' 2>/dev/null || echo "ui/dist/index.html")
SOURCE_DIR=$(echo "$CHECK_CONFIG" | jq -r '.source_dir // "ui/src"' 2>/dev/null || echo "ui/src")
WATCH_FILE_DEPENDENCIES=$(echo "$CHECK_CONFIG" | jq -r 'if has("watch_file_dependencies") then .watch_file_dependencies else true end' 2>/dev/null || echo "true")

# Resolve paths relative to APP_ROOT or current directory if not absolute
# Set default APP_ROOT if not provided
: "${APP_ROOT:=$(pwd)}"

if [[ ! "$BUNDLE_PATH" =~ ^/ ]]; then
    BUNDLE_PATH="${APP_ROOT}/$BUNDLE_PATH"
fi

if [[ ! "$SOURCE_DIR" =~ ^/ ]]; then
    SOURCE_DIR="${APP_ROOT}/$SOURCE_DIR"
fi

# Check 1: Bundle must exist
if [[ ! -f "$BUNDLE_PATH" ]]; then
    echo "[DEBUG] UI bundle missing: $BUNDLE_PATH" >&2
    exit 0  # Setup needed
fi

# Check 2: Source files must not be newer than bundle
# Check all files (code, assets, markdown, etc.) - let build tools decide what matters
if [[ -d "$SOURCE_DIR" ]]; then
    if find "$SOURCE_DIR" -type f -newer "$BUNDLE_PATH" 2>/dev/null | head -1 | grep -q .; then
        echo "[DEBUG] UI bundle outdated (source files modified): $BUNDLE_PATH" >&2
        exit 0  # Setup needed
    fi
fi

# Check 3: Configuration files must not be newer than bundle
BUNDLE_DIR=$(dirname "$BUNDLE_PATH")
UI_DIR=$(dirname "$BUNDLE_DIR")  # Usually ui/ if bundle is ui/dist/index.html

for config_file in "package.json" "vite.config.ts" "vite.config.js" "tsconfig.json" "index.html"; do
    config_path="$UI_DIR/$config_file"
    if [[ -f "$config_path" ]] && [[ "$config_path" -nt "$BUNDLE_PATH" ]]; then
        echo "[DEBUG] UI bundle outdated ($config_file modified): $BUNDLE_PATH" >&2
        exit 0  # Setup needed
    fi
done

path_is_excluded() {
    local candidate="$1"
    while IFS= read -r pattern; do
        [[ -z "$pattern" ]] && continue
        if [[ "$candidate" == $pattern ]]; then
            return 0
        fi
    done < <(echo "$CHECK_CONFIG" | jq -r '.dependency_excludes[]? // empty' 2>/dev/null)
    return 1
}

shared_dependency_newer_than_bundle() {
    local dep_path="$1"
    find "$dep_path" -type f \
        -not -path '*/node_modules/*' \
        -not -path '*/.git/*' \
        -not -path '*/coverage/*' \
        -newer "$BUNDLE_PATH" 2>/dev/null | head -1 | grep -q .
}

if [[ "$WATCH_FILE_DEPENDENCIES" == "true" ]]; then
    PACKAGE_JSON_PATH="$UI_DIR/package.json"
    if [[ -f "$PACKAGE_JSON_PATH" ]]; then
        declare -A seen_dep_paths=()
        dep_specs=$(jq -r '
            [
              .dependencies,
              .devDependencies,
              .peerDependencies,
              .optionalDependencies
            ]
            | map(. // {})
            | add
            | to_entries[]
            | select((.value | type) == "string" and (.value | startswith("file:")))
            | .value
        ' "$PACKAGE_JSON_PATH" 2>/dev/null || true)

        while IFS= read -r dep_spec; do
            [[ -z "$dep_spec" ]] && continue

            dep_rel="${dep_spec#file:}"
            dep_path="$dep_rel"
            if [[ "$dep_path" != /* ]]; then
                dep_path="$UI_DIR/$dep_path"
            fi
            dep_path=$(realpath -m "$dep_path")

            if path_is_excluded "$dep_path"; then
                continue
            fi
            if [[ -n "${seen_dep_paths[$dep_path]:-}" ]]; then
                continue
            fi
            seen_dep_paths["$dep_path"]=1

            if [[ ! -d "$dep_path" ]]; then
                echo "[DEBUG] UI bundle outdated (shared package path missing): $dep_path" >&2
                exit 0  # Setup needed to reinstall dependencies
            fi

            if shared_dependency_newer_than_bundle "$dep_path"; then
                echo "[DEBUG] UI bundle outdated (shared package modified): $dep_path" >&2
                exit 0  # Setup needed
            fi
        done <<< "$dep_specs"
    fi
fi

# All checks passed - bundle is current
exit 1  # Setup not needed
