#!/usr/bin/env bash
################################################################################
# CLI Output Formatter for Vrooli CLI
# 
# Provides consistent output formatting using format.sh library.
# Standardizes how CLI commands format their output for both text and JSON.
#
# Usage:
#   source cli/lib/output-formatter.sh
#   cli::format_output "json" "kv" key1 value1 key2 value2
#   cli::format_table "text" headers -- rows
#
################################################################################

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
source "${var_LOG_FILE}" 2>/dev/null || true
source "${VROOLI_ROOT}/scripts/lib/utils/format.sh" 2>/dev/null || true

################################################################################
# Main CLI Output Formatter
# Args: format_type data_type [data...]
# format_type: json|text
# data_type: kv|list|table|raw|status|error
################################################################################
cli::format_output() {
    local format="${1:-text}"
    local data_type="${2:-kv}"
    shift 2 || true
    
    case "$data_type" in
        kv|key_value)
            format::key_value "$format" "$@"
            ;;
        list|array)
            format::list "$format" "$@"
            ;;
        table)
            format::table "$format" "$@"
            ;;
        raw)
            format::raw "$format" "$@"
            ;;
        status)
            cli::format_status "$format" "$@"
            ;;
        error)
            cli::format_error "$format" "$@"
            ;;
        *)
            log::error "Unknown data type: $data_type"
            return 1
            ;;
    esac
}

################################################################################
# Format Status Information
# Args: format status_type message [details...]
################################################################################
cli::format_status() {
    local format="${1:-text}"
    local status_type="${2:-info}"
    local message="${3:-}"
    shift 3 || true
    
    if [[ "$format" == "json" ]]; then
        local details_json="{}"
        if [[ $# -gt 0 ]]; then
            local details_array=()
            while [[ $# -gt 0 ]]; do
                details_array+=("$1")
                shift
            done
            details_json=$(format::list json "${details_array[@]}")
        fi
        
        format::key_value json \
            "status" "$status_type" \
            "message" "$message" \
            "details" "$details_json"
    else
        format::status "$status_type" "$message"
        if [[ $# -gt 0 ]]; then
            for detail in "$@"; do
                echo "  $detail"
            done
        fi
    fi
}

################################################################################
# Format Error Information
# Args: format error_message [error_code] [details...]
################################################################################
cli::format_error() {
    local format="${1:-text}"
    local error_message="${2:-Unknown error}"
    local error_code="${3:-1}"
    shift 3 || true
    
    if [[ "$format" == "json" ]]; then
        local details_json="{}"
        if [[ $# -gt 0 ]]; then
            local details_array=()
            while [[ $# -gt 0 ]]; do
                details_array+=("$1")
                shift
            done
            details_json=$(format::list json "${details_array[@]}")
        fi
        
        format::key_value json \
            "error" true \
            "message" "$error_message" \
            "code" "$error_code" \
            "details" "$details_json"
    else
        format::status "error" "$error_message"
        if [[ $# -gt 0 ]]; then
            for detail in "$@"; do
                echo "  $detail"
            done
        fi
    fi
}

################################################################################
# Format Command Result
# Args: format success message [data...]
################################################################################
cli::format_result() {
    local format="${1:-text}"
    local success="${2:-true}"
    local message="${3:-}"
    shift 3 || true
    
    if [[ "$format" == "json" ]]; then
        local data_json="{}"
        if [[ $# -gt 0 ]]; then
            # Assume remaining args are key-value pairs
            data_json=$(format::key_value json "$@")
        fi
        
        format::key_value json \
            "success" "$success" \
            "message" "$message" \
            "data" "$data_json"
    else
        if [[ "$success" == "true" ]]; then
            format::status "success" "$message"
        else
            format::status "error" "$message"
        fi
        
        if [[ $# -gt 0 ]]; then
            # Display key-value pairs in text format
            while [[ $# -gt 0 ]]; do
                local key="$1"
                local value="$2"
                shift 2 || break
                echo "  $key: $value"
            done
        fi
    fi
}

################################################################################
# Format List with Headers
# Args: format title items...
################################################################################
cli::format_list() {
    local format="${1:-text}"
    local title="${2:-}"
    shift 2 || true
    
    if [[ "$format" == "json" ]]; then
        local items_json
        items_json=$(format::list json "$@")
        
        if [[ -n "$title" ]]; then
            format::key_value json "$title" "$items_json"
        else
            echo "$items_json"
        fi
    else
        if [[ -n "$title" ]]; then
            echo "$title:"
        fi
        format::list text "$@"
    fi
}

################################################################################
# Format Table with Title
# Args: format title headers... -- rows...
################################################################################
cli::format_table() {
    local format="${1:-text}"
    local title="${2:-}"
    shift 2
    
    # Collect headers
    local headers=()
    while [[ $# -gt 0 ]] && [[ "$1" != "--" ]]; do
        headers+=("$1")
        shift
    done
    
    # Skip the -- separator
    [[ "$1" == "--" ]] && shift
    
    # Collect rows
    local rows=()
    while [[ $# -gt 0 ]]; do
        rows+=("$1")
        shift
    done
    
    if [[ "$format" == "json" ]]; then
        local table_json
        table_json=$(format::table json "${headers[@]}" -- "${rows[@]}")
        
        if [[ -n "$title" ]]; then
            format::key_value json "$title" "$table_json"
        else
            echo "$table_json"
        fi
    else
        if [[ -n "$title" ]]; then
            echo "$title:"
            echo ""
        fi
        format::table text "${headers[@]}" -- "${rows[@]}"
    fi
}

################################################################################
# Format Section Header
# Args: format title [subtitle]
################################################################################
cli::format_header() {
    local format="${1:-text}"
    local title="${2:-}"
    local subtitle="${3:-}"
    
    if [[ "$format" == "json" ]]; then
        local header_json
        header_json=$(format::key_value json "title" "$title")
        
        if [[ -n "$subtitle" ]]; then
            header_json=$(format::json_object "title" "$title" "subtitle" "$subtitle")
        fi
        
        format::key_value json "header" "$header_json"
    else
        echo ""
        echo "$title"
        if [[ -n "$subtitle" ]]; then
            echo "$subtitle"
        fi
        echo "$(printf '=%.0s' {1..50})"
        echo ""
    fi
}

################################################################################
# Format Summary Information
# Args: format summary_items...
################################################################################
cli::format_summary() {
    local format="${1:-text}"
    shift
    
    if [[ "$format" == "json" ]]; then
        # Assume args are key-value pairs
        format::key_value json "$@"
    else
        echo ""
        echo "Summary:"
        while [[ $# -gt 0 ]]; do
            local key="$1"
            local value="$2"
            shift 2 || break
            
            # Format key for display
            local display_key="${key//_/ }"
            display_key="$(echo "$display_key" | sed 's/\b\(.\)/\u\1/g')"
            
            echo "  $display_key: $value"
        done
        echo ""
    fi
}

################################################################################
# Format Help Text
# Args: format help_text
################################################################################
cli::format_help() {
    local format="${1:-text}"
    local help_text="${2:-}"
    
    if [[ "$format" == "json" ]]; then
        format::key_value json "help" "$help_text"
    else
        echo "$help_text"
    fi
}

# Export functions
export -f cli::format_output
export -f cli::format_status
export -f cli::format_error
export -f cli::format_result
export -f cli::format_list
export -f cli::format_table
export -f cli::format_header
export -f cli::format_summary
export -f cli::format_help 