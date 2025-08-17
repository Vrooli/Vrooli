#!/usr/bin/env bash
################################################################################
# Generic Output Formatter Library
# 
# Provides consistent formatting for JSON and text output across CLI commands.
# Supports key-value pairs, lists, tables, and nested structures.
#
# Usage:
#   source format.sh
#   format::output "json" key1 value1 key2 value2
#   format::list "text" "${array[@]}"
#   format::table "json" "${headers[@]}" -- "${rows[@]}"
#   format::table "text" "${headers[@]}" -- "${rows[@]}"
#
################################################################################

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/var.sh" 2>/dev/null || true

################################################################################
# Main Output Formatter
# Args: format_type data_type [data...]
# format_type: json|text
# data_type: kv|list|table|raw
################################################################################
format::output() {
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
        *)
            echo "Error: Unknown data type: $data_type" >&2
            return 1
            ;;
    esac
}

################################################################################
# Format Key-Value Pairs
# Args: format key1 value1 [key2 value2 ...]
# Example: format::key_value json name "John" age 30 active true
################################################################################
format::key_value() {
    local format="${1:-text}"
    shift
    
    if [[ "$format" == "json" ]]; then
        local json_obj="{"
        local first=true
        
        while [[ $# -gt 0 ]]; do
            local key="${1:-}"
            local value="${2:-}"
            shift 2 || break
            
            [[ "$first" == "true" ]] && first=false || json_obj="${json_obj},"
            
            # Auto-detect value type
            if [[ "$value" =~ ^[0-9]+$ ]]; then
                # Integer
                json_obj="${json_obj}\"${key}\":${value}"
            elif [[ "$value" =~ ^[0-9]+\.[0-9]+$ ]]; then
                # Float
                json_obj="${json_obj}\"${key}\":${value}"
            elif [[ "$value" == "true" || "$value" == "false" || "$value" == "null" ]]; then
                # Boolean/null
                json_obj="${json_obj}\"${key}\":${value}"
            elif [[ "$value" =~ ^\{.*\}$ ]] || [[ "$value" =~ ^\[.*\]$ ]]; then
                # Already JSON
                json_obj="${json_obj}\"${key}\":${value}"
            else
                # String
                # Escape backslashes, quotes, and control characters
                value="${value//\\/\\\\}"
                value="${value//\"/\\\"}"
                value="${value//$'\n'/\\n}"
                value="${value//$'\r'/\\r}"
                value="${value//$'\t'/\\t}"
                json_obj="${json_obj}\"${key}\":\"${value}\""
            fi
        done
        
        json_obj="${json_obj}}"
        echo "$json_obj"
    else
        # Text format
        while [[ $# -gt 0 ]]; do
            local key="${1:-}"
            local value="${2:-}"
            shift 2 || break
            
            # Format key for display (capitalize, replace underscores)
            local display_key="${key//_/ }"
            display_key="$(echo "$display_key" | sed 's/\b\(.\)/\u\1/g')"
            
            echo "${display_key}: ${value}"
        done
    fi
}

################################################################################
# Format List/Array
# Args: format item1 [item2 ...]
# Example: format::list json "apple" "banana" "cherry"
################################################################################
format::list() {
    local format="${1:-text}"
    shift
    
    if [[ "$format" == "json" ]]; then
        local json_array="["
        local first=true
        
        for item in "$@"; do
            [[ "$first" == "true" ]] && first=false || json_array="${json_array},"
            
            # Auto-detect type
            if [[ "$item" =~ ^[0-9]+$ ]] || [[ "$item" =~ ^[0-9]+\.[0-9]+$ ]] || 
               [[ "$item" == "true" || "$item" == "false" || "$item" == "null" ]]; then
                json_array="${json_array}${item}"
            else
                item="${item//\\/\\\\}"  # Escape backslashes
                item="${item//\"/\\\"}"  # Escape quotes
                item="${item//$'\n'/\\n}"
                item="${item//$'\r'/\\r}"
                item="${item//$'\t'/\\t}"
                json_array="${json_array}\"${item}\""
            fi
        done
        
        json_array="${json_array}]"
        echo "$json_array"
    else
        # Text format - bulleted list
        for item in "$@"; do
            echo "  - ${item}"
        done
    fi
}

################################################################################
# Format Table
# Args: format headers... -- rows...
# Example: format::table json "Name" "Age" -- "John:30" "Jane:25"
################################################################################
format::table() {
    local format="${1:-text}"
    shift
    
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
        echo "["
        local first_row=true
        
        for row in "${rows[@]}"; do
            [[ "$first_row" == "true" ]] && first_row=false || echo ","
            
            # Split row by delimiter (: or |)
            IFS=':' read -ra values <<< "$row"
            [[ ${#values[@]} -eq 0 ]] && IFS='|' read -ra values <<< "$row"
            
            echo -n "  {"
            local first_col=true
            
            for i in "${!headers[@]}"; do
                [[ "$first_col" == "true" ]] && first_col=false || echo -n ", "
                
                local header="${headers[$i]}"
                local value="${values[$i]:-}"
                
                # Auto-detect type
                if [[ "$value" =~ ^[0-9]+$ ]] || [[ "$value" =~ ^[0-9]+\.[0-9]+$ ]] || [[ "$value" == "true" || "$value" == "false" || "$value" == "null" ]]; then
                    echo -n "\"${header}\": ${value}"
                else
                    value="${value//\"/\\\"}"
                    echo -n "\"${header}\": \"${value}\""
                fi
            done
            
            echo -n "}"
        done
        
        echo ""
        echo "]"
    else
        # Text format - aligned table
        # Calculate column widths
        local widths=()
        for header in "${headers[@]}"; do
            widths+=(${#header})
        done
        
        # Update widths based on data
        for row in "${rows[@]}"; do
            IFS=':' read -ra values <<< "$row"
            [[ ${#values[@]} -eq 0 ]] && IFS='|' read -ra values <<< "$row"
            
            for i in "${!values[@]}"; do
                local len=${#values[$i]}
                [[ $len -gt ${widths[$i]:-0} ]] && widths[$i]=$len
            done
        done
        
        # Print headers
        for i in "${!headers[@]}"; do
            printf "%-${widths[$i]}s  " "${headers[$i]}"
        done
        echo ""
        
        # Print separator with simple dashes
        for i in "${!widths[@]}"; do
            printf "%-${widths[$i]}s" "" | tr ' ' '-'
            [[ $i -lt $((${#widths[@]} - 1)) ]] && printf "  "
        done
        echo ""
        
        # Print rows
        for row in "${rows[@]}"; do
            IFS=':' read -ra values <<< "$row"
            [[ ${#values[@]} -eq 0 ]] && IFS='|' read -ra values <<< "$row"
            
            for i in "${!values[@]}"; do
                printf "%-${widths[$i]:-10}s  " "${values[$i]:-}"
            done
            echo ""
        done
    fi
}

################################################################################
# Format Raw Output (pass-through with optional processing)
# Args: format content
################################################################################
format::raw() {
    local format="${1:-text}"
    local content="${2:-}"
    
    if [[ "$format" == "json" ]]; then
        # Try to validate/format as JSON, fallback to string
        if echo "$content" | jq -e '.' >/dev/null 2>&1; then
            echo "$content" | jq '.'
        else
            # Wrap as JSON string
            content="${content//\"/\\\"}"
            content="${content//$'\n'/\\n}"
            echo "\"${content}\""
        fi
    else
        echo "$content"
    fi
}

################################################################################
# Build Complex JSON Object
# Usage: 
#   json_obj=$(format::json_object \
#     system "$(format::key_value json cpu 50 memory 80)" \
#     resources "$(format::list json postgres redis)")
################################################################################
format::json_object() {
    local json="{"
    local first=true
    
    while [[ $# -gt 0 ]]; do
        local key="${1:-}"
        local value="${2:-}"
        shift 2 || break
        
        [[ "$first" == "true" ]] && first=false || json="${json},"
        json="${json}\"${key}\":${value}"
    done
    
    json="${json}}"
    echo "$json"
}

################################################################################
# Pretty Print JSON
# Args: json_string
################################################################################
format::json_pretty() {
    local json="${1:-}"
    echo "$json" | jq '.' 2>/dev/null || echo "$json"
}

################################################################################
# Convert JSON to Text Summary
# Args: json_string [template]
################################################################################
format::json_to_text() {
    local json="${1:-}"
    local template="${2:-}"
    
    if [[ -n "$template" ]]; then
        # Use template for formatting
        echo "$json" | jq -r "$template"
    else
        # Auto-format
        echo "$json" | jq -r 'to_entries | .[] | "\(.key): \(.value)"'
    fi
}

################################################################################
# Format Status Message with Icon
# Args: status_type message
# status_type: success|error|warning|info
################################################################################
format::status() {
    local status="${1:-info}"
    local message="${2:-}"
    
    case "$status" in
        success|ok)
            echo "✅ ${message}"
            ;;
        error|fail)
            echo "❌ ${message}"
            ;;
        warning|warn)
            echo "⚠️  ${message}"
            ;;
        info)
            echo "ℹ️  ${message}"
            ;;
        running)
            echo "🟢 ${message}"
            ;;
        stopped)
            echo "🔴 ${message}"
            ;;
        *)
            echo "${message}"
            ;;
    esac
}

################################################################################
# Format Progress Bar
# Args: current total [width]
################################################################################
format::progress_bar() {
    local current="${1:-0}"
    local total="${2:-100}"
    local width="${3:-20}"
    
    local percent=$((current * 100 / total))
    local filled=$((width * current / total))
    local empty=$((width - filled))
    
    printf "["
    printf "%${filled}s" | tr ' ' '='
    printf "%${empty}s" | tr ' ' '.'
    printf "] %3d%%\n" "$percent"
}


# Export functions
export -f format::output
export -f format::key_value
export -f format::list
export -f format::table
export -f format::raw
export -f format::json_object
export -f format::json_pretty
export -f format::json_to_text
export -f format::status
export -f format::progress_bar