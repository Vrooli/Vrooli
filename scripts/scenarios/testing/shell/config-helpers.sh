#!/usr/bin/env bash
# Configuration parsing helpers for JSON files
# Provides common jq utilities to reduce duplication across test scripts
set -euo pipefail

# Get a boolean value from JSON config with a default
# Usage: config::get_boolean CONFIG_FILE JQ_PATH DEFAULT_VALUE
# Example: config::get_boolean ".vrooli/testing.json" ".unit.enabled" "true"
# Returns: "true" or "false"
config::get_boolean() {
    local config_file="$1"
    local jq_path="$2"
    local default_value="${3:-false}"

    if [ ! -f "$config_file" ]; then
        echo "$default_value"
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        echo "$default_value"
        return 0
    fi

    local value
    value=$(jq -r "${jq_path} // \"${default_value}\"" "$config_file" 2>/dev/null || echo "$default_value")

    # Normalize to "true" or "false"
    if [ "$value" = "true" ] || [ "$value" = "1" ]; then
        echo "true"
    else
        echo "false"
    fi
}

# Get a string value from JSON config with a default
# Usage: config::get_string CONFIG_FILE JQ_PATH DEFAULT_VALUE
# Example: config::get_string ".vrooli/testing.json" ".unit.dir" "api"
# Returns: String value or default
config::get_string() {
    local config_file="$1"
    local jq_path="$2"
    local default_value="${3:-}"

    if [ ! -f "$config_file" ]; then
        echo "$default_value"
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        echo "$default_value"
        return 0
    fi

    local value
    value=$(jq -r "${jq_path} // \"${default_value}\"" "$config_file" 2>/dev/null || echo "$default_value")

    # Return empty string if jq returned "null"
    if [ "$value" = "null" ]; then
        echo "$default_value"
    else
        echo "$value"
    fi
}

# Get a number value from JSON config with a default
# Usage: config::get_number CONFIG_FILE JQ_PATH DEFAULT_VALUE
# Example: config::get_number ".vrooli/testing.json" ".unit.timeout" "120"
# Returns: Number value or default
config::get_number() {
    local config_file="$1"
    local jq_path="$2"
    local default_value="${3:-0}"

    if [ ! -f "$config_file" ]; then
        echo "$default_value"
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        echo "$default_value"
        return 0
    fi

    local value
    value=$(jq -r "${jq_path} // ${default_value}" "$config_file" 2>/dev/null || echo "$default_value")

    # Return default if jq returned "null" or non-numeric
    if [ "$value" = "null" ] || ! [[ "$value" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
        echo "$default_value"
    else
        echo "$value"
    fi
}

# Get an array from JSON config
# Usage: config::get_array CONFIG_FILE JQ_PATH
# Example: mapfile -t items < <(config::get_array ".vrooli/testing.json" ".unit.languages[]")
# Returns: Array elements, one per line (suitable for mapfile)
config::get_array() {
    local config_file="$1"
    local jq_path="$2"

    if [ ! -f "$config_file" ]; then
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        return 0
    fi

    jq -r "$jq_path // empty" "$config_file" 2>/dev/null || true
}

# Check if a field exists in JSON config
# Usage: config::has_field CONFIG_FILE JQ_PATH
# Example: if config::has_field ".vrooli/testing.json" ".unit.enabled"; then ...
# Returns: 0 if field exists, 1 otherwise
config::has_field() {
    local config_file="$1"
    local jq_path="$2"

    if [ ! -f "$config_file" ]; then
        return 1
    fi

    if ! command -v jq >/dev/null 2>&1; then
        return 1
    fi

    jq -e "$jq_path" "$config_file" >/dev/null 2>&1
}

# Get object keys from JSON config
# Usage: mapfile -t keys < <(config::get_keys CONFIG_FILE JQ_PATH)
# Example: mapfile -t phases < <(config::get_keys ".vrooli/testing.json" ".phases | keys[]")
# Returns: Object keys, one per line (suitable for mapfile)
config::get_keys() {
    local config_file="$1"
    local jq_path="$2"

    if [ ! -f "$config_file" ]; then
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        return 0
    fi

    jq -r "$jq_path // empty" "$config_file" 2>/dev/null || true
}

# Get entire JSON object as compact JSON string
# Usage: config::get_object CONFIG_FILE JQ_PATH
# Example: object_json=$(config::get_object ".vrooli/testing.json" ".performance.checks")
# Returns: Compact JSON string or "{}"
config::get_object() {
    local config_file="$1"
    local jq_path="$2"
    local default_value="${3:-{}}"

    if [ ! -f "$config_file" ]; then
        echo "$default_value"
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        echo "$default_value"
        return 0
    fi

    local value
    value=$(jq -c "$jq_path // $default_value" "$config_file" 2>/dev/null || echo "$default_value")

    if [ "$value" = "null" ]; then
        echo "$default_value"
    else
        echo "$value"
    fi
}

# Merge two JSON objects (second overwrites first)
# Usage: config::merge_objects JSON1 JSON2
# Example: merged=$(config::merge_objects '{"a":1}' '{"b":2}')
# Returns: Merged JSON object
config::merge_objects() {
    local json1="$1"
    local json2="$2"

    if ! command -v jq >/dev/null 2>&1; then
        echo "$json1"
        return 0
    fi

    echo "$json1" | jq -c ". * ($json2 | fromjson)" --argjson json2 "$json2" 2>/dev/null || echo "$json1"
}

# Check if jq is available
# Usage: if config::jq_available; then ...
# Returns: 0 if jq is available, 1 otherwise
config::jq_available() {
    command -v jq >/dev/null 2>&1
}

# Export all functions
export -f config::get_boolean
export -f config::get_string
export -f config::get_number
export -f config::get_array
export -f config::has_field
export -f config::get_keys
export -f config::get_object
export -f config::merge_objects
export -f config::jq_available
