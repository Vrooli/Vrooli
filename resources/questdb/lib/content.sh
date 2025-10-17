#!/usr/bin/env bash
# QuestDB Content Functions - Business Functionality
# These functions handle the actual business use of QuestDB (queries, data operations, etc.)

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
QUESTDB_LIB_DIR="${APP_ROOT}/resources/questdb/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/api.sh"
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/tables.sh"
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/inject.sh"

# Add content (generic interface - maps to appropriate sub-functionality)
questdb::content::add() {
    local content_type="${1:-table}"
    shift
    
    case "$content_type" in
        table)
            questdb::content::tables create "$@"
            ;;
        data|sql|csv|json)
            questdb::content::inject "$@"
            ;;
        *)
            log::error "Unknown content type: $content_type"
            echo "Usage: resource-questdb content add [table|data|sql|csv|json] [args...]"
            echo ""
            echo "Examples:"
            echo "  resource-questdb content add table sensors schema.sql"
            echo "  resource-questdb content add data data.csv"
            echo "  resource-questdb content add sql schema.sql"
            return 1
            ;;
    esac
}

# List content (show tables by default)
questdb::content::list() {
    questdb::tables::list "$@"
}

# Get content (show table info or execute query)
questdb::content::get() {
    local target="${1:-}"
    
    if [[ -z "$target" ]]; then
        log::error "Target required (table name or 'tables')"
        echo "Usage: resource-questdb content get [table_name|tables]"
        echo ""
        echo "Examples:"
        echo "  resource-questdb content get tables"
        echo "  resource-questdb content get sensors"
        return 1
    fi
    
    if [[ "$target" == "tables" ]]; then
        questdb::tables::list
    else
        questdb::tables::info "$target"
    fi
}

# Remove content (drop tables)
questdb::content::remove() {
    local table_name="${1:-}"
    
    if [[ -z "$table_name" ]]; then
        log::error "Table name required"
        echo "Usage: resource-questdb content remove <table_name>"
        return 1
    fi
    
    questdb::tables::drop "$table_name"
}

# Execute content (run SQL queries)
questdb::content::execute() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        log::error "SQL query required"
        echo "Usage: resource-questdb content execute \"SQL QUERY\""
        echo ""
        echo "Examples:"
        echo "  resource-questdb content execute \"SHOW TABLES\""
        echo "  resource-questdb content execute \"SELECT * FROM sensors LIMIT 10\""
        return 1
    fi
    
    questdb::api::query "$query"
}

# Execute SQL query (content subcommand)
questdb::content::query() {
    questdb::content::execute "$@"
}

# Inject data into QuestDB (content subcommand)
questdb::content::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-questdb content inject <file.sql|file.csv|file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-questdb content inject data.sql"
        echo "  resource-questdb content inject shared:initialization/storage/questdb/sample.csv"
        echo "  resource-questdb content inject time_series.json"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${APP_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Use existing injection function based on file type
    local extension="${file##*.}"
    case "$extension" in
        sql)
            questdb::inject::sql "$file"
            ;;
        csv|json)
            if command -v questdb::api::bulk_insert_csv &>/dev/null && [[ "$extension" == "csv" ]]; then
                questdb::api::bulk_insert_csv "$file"
            else
                log::error "Data injection for $extension files not yet implemented"
                return 1
            fi
            ;;
        *)
            log::error "Unsupported file type: $extension (supported: sql, csv, json)"
            return 1
            ;;
    esac
}

# Table operations (content subcommand)
questdb::content::tables() {
    local action="${1:-list}"
    shift
    
    case "$action" in
        list)
            questdb::tables::list "$@"
            ;;
        create)
            local table_name="${1:-}"
            local schema_file="${2:-}"
            
            if [[ -z "$table_name" ]]; then
                log::error "Table name required for create"
                echo "Usage: resource-questdb content tables create <table_name> [schema_file]"
                echo ""
                echo "Examples:"
                echo "  resource-questdb content tables create sensors"
                echo "  resource-questdb content tables create trades schema.sql"
                return 1
            fi
            
            if [[ -n "$schema_file" ]]; then
                questdb::tables::create_from_file "$table_name" "$schema_file"
            else
                questdb::tables::create_defaults "$table_name"
            fi
            ;;
        drop)
            local table_name="${1:-}"
            
            if [[ -z "$table_name" ]]; then
                log::error "Table name required for drop"
                echo "Usage: resource-questdb content tables drop <table_name>"
                return 1
            fi
            
            questdb::tables::drop "$table_name"
            ;;
        info)
            local table_name="${1:-}"
            
            if [[ -z "$table_name" ]]; then
                log::error "Table name required for info"
                echo "Usage: resource-questdb content tables info <table_name>"
                return 1
            fi
            
            questdb::tables::info "$table_name"
            ;;
        *)
            log::error "Unknown table action: $action"
            echo "Usage: resource-questdb content tables [list|create|drop|info] [args...]"
            return 1
            ;;
    esac
}

# Make API request (content subcommand)
questdb::content::api() {
    local endpoint="${1:-}"
    local method="${2:-GET}"
    local data="${3:-}"
    
    if [[ -z "$endpoint" ]]; then
        log::error "API endpoint required"
        echo "Usage: resource-questdb content api <endpoint> [method] [data]"
        echo ""
        echo "Examples:"
        echo "  resource-questdb content api /status GET"
        echo "  resource-questdb content api /exec POST '{\"query\":\"SELECT * FROM trades\"}'"
        return 1
    fi
    
    # Use curl to make API request directly
    local base_url="${QUESTDB_BASE_URL:-http://localhost:9000}"
    local url="$base_url$endpoint"
    
    case "$method" in
        GET)
            curl -s "$url" | jq . 2>/dev/null || curl -s "$url"
            ;;
        POST)
            if [[ -n "$data" ]]; then
                curl -s -X POST -H "Content-Type: application/json" -d "$data" "$url" | jq . 2>/dev/null || curl -s -X POST -H "Content-Type: application/json" -d "$data" "$url"
            else
                curl -s -X POST "$url" | jq . 2>/dev/null || curl -s -X POST "$url"
            fi
            ;;
        *)
            log::error "Unsupported HTTP method: $method (use GET or POST)"
            return 1
            ;;
    esac
}

# Open web console (content subcommand)
questdb::content::console() {
    local console_url="${QUESTDB_BASE_URL:-http://localhost:9000}"
    log::info "Opening QuestDB console at: $console_url"
    
    if command -v xdg-open &>/dev/null; then
        xdg-open "$console_url" 2>/dev/null &
    elif command -v open &>/dev/null; then
        open "$console_url" 2>/dev/null &
    else
        echo "Please open $console_url in your browser"
    fi
}