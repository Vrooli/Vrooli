#!/usr/bin/env bash
# QuestDB Data Injection Module
# Provides capability to inject SQL, CSV, and JSON data into QuestDB

# Get the script directory
QUESTDB_INJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required libraries
source "${QUESTDB_INJECT_DIR}/../../../../lib/utils/var.sh"
source "${QUESTDB_INJECT_DIR}/../../../../lib/utils/log.sh"
source "${QUESTDB_INJECT_DIR}/../config/defaults.sh"
source "${QUESTDB_INJECT_DIR}/api.sh"

#######################################
# Inject SQL file into QuestDB
# Arguments:
#   1 - Path to SQL file
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::inject::sql() {
    local sql_file="$1"
    
    if [[ ! -f "$sql_file" ]]; then
        log::error "SQL file not found: $sql_file"
        return 1
    fi
    
    local sql_content
    sql_content=$(cat "$sql_file")
    
    log::info "Injecting SQL from: $sql_file"
    
    # Execute SQL via API
    if questdb::api::query "$sql_content"; then
        log::success "SQL injection successful"
        return 0
    else
        log::error "SQL injection failed"
        return 1
    fi
}

#######################################
# Main injection handler
# Arguments:
#   1 - File path to inject
#   --type - File type (sql/csv/json)
# Returns:
#   0 on success, 1 on failure  
#######################################
questdb::inject() {
    local file_path="$1"
    local file_type=""
    
    # Parse arguments
    shift
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                file_type="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Auto-detect file type if not specified
    if [[ -z "$file_type" ]]; then
        case "${file_path##*.}" in
            sql)
                file_type="sql"
                ;;
            csv)
                file_type="csv"
                ;;
            json)
                file_type="json"
                ;;
            *)
                log::error "Unknown file type. Please specify --type sql|csv|json"
                return 1
                ;;
        esac
    fi
    
    # Handle based on type
    case "$file_type" in
        sql)
            questdb::inject::sql "$file_path"
            ;;
        csv)
            log::error "CSV injection not yet implemented"
            return 1
            ;;
        json)
            log::error "JSON injection not yet implemented"
            return 1
            ;;
        *)
            log::error "Unsupported file type: $file_type"
            return 1
            ;;
    esac
}
