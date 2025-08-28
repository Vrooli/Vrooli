#!/usr/bin/env bash
# SQL Language Parser for Qdrant Embeddings
# Extracts database schemas, queries, procedures, and documentation from SQL files
#
# Handles:
# - DDL statements (CREATE TABLE, VIEW, INDEX)
# - DML statements (SELECT, INSERT, UPDATE, DELETE)
# - Stored procedures and functions
# - Triggers and constraints
# - Comments and documentation
# - Database migrations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract table definitions from SQL
# 
# Parses CREATE TABLE statements and their structure
#
# Arguments:
#   $1 - SQL file path
# Returns: JSON with table definitions
#######################################
extractor::lib::sql::extract_tables() {
    local file="$1"
    local tables=()
    
    # Extract CREATE TABLE statements (case insensitive, multi-line)
    while IFS= read -r table_def; do
        # Extract table name
        local table_name=$(echo "$table_def" | grep -iEo 'CREATE\s+TABLE\s+(IF\s+NOT\s+EXISTS\s+)?[`"]?([a-zA-Z_][a-zA-Z0-9_\.]*)[`"]?' | \
            sed -E 's/CREATE\s+TABLE\s+(IF\s+NOT\s+EXISTS\s+)?//i' | tr -d '`"' | head -1)
        
        if [[ -n "$table_name" ]]; then
            # Extract column count (rough estimate)
            local column_count=$(echo "$table_def" | grep -c "^\s*[a-zA-Z_]" || echo "0")
            
            # Check for common patterns
            local has_primary_key="false"
            local has_foreign_key="false"
            local has_indexes="false"
            
            echo "$table_def" | grep -qi "PRIMARY KEY" && has_primary_key="true"
            echo "$table_def" | grep -qi "FOREIGN KEY\|REFERENCES" && has_foreign_key="true"
            echo "$table_def" | grep -qi "INDEX\|KEY" && has_indexes="true"
            
            # Extract comment if present
            local table_comment=$(echo "$table_def" | grep -i "COMMENT\s*=" | sed -E "s/.*COMMENT\s*=\s*['\"]([^'\"]*)['\"].*/\1/" || echo "")
            
            tables+=("$(jq -n \
                --arg name "$table_name" \
                --arg columns "$column_count" \
                --arg pk "$has_primary_key" \
                --arg fk "$has_foreign_key" \
                --arg idx "$has_indexes" \
                --arg comment "$table_comment" \
                '{
                    name: $name,
                    column_count: ($columns | tonumber),
                    has_primary_key: ($pk == "true"),
                    has_foreign_keys: ($fk == "true"),
                    has_indexes: ($idx == "true"),
                    comment: $comment
                }')")
        fi
    done < <(awk '/CREATE\s+TABLE/I,/;/' "$file" 2>/dev/null)
    
    if [[ ${#tables[@]} -gt 0 ]]; then
        printf '%s\n' "${tables[@]}" | jq -s '.'
    else
        echo "[]"
    fi
}

#######################################
# Extract view definitions from SQL
# 
# Parses CREATE VIEW statements
#
# Arguments:
#   $1 - SQL file path
# Returns: JSON with view definitions
#######################################
extractor::lib::sql::extract_views() {
    local file="$1"
    local views=()
    
    while IFS= read -r view_def; do
        # Extract view name
        local view_name=$(echo "$view_def" | grep -iEo 'CREATE\s+(OR\s+REPLACE\s+)?VIEW\s+[`"]?([a-zA-Z_][a-zA-Z0-9_\.]*)[`"]?' | \
            sed -E 's/CREATE\s+(OR\s+REPLACE\s+)?VIEW\s+//i' | tr -d '`"' | head -1)
        
        if [[ -n "$view_name" ]]; then
            # Check if it's a materialized view
            local is_materialized="false"
            echo "$view_def" | grep -qi "MATERIALIZED" && is_materialized="true"
            
            # Extract referenced tables (rough)
            local referenced_tables=$(echo "$view_def" | grep -iEo 'FROM\s+[`"]?[a-zA-Z_][a-zA-Z0-9_\.]*[`"]?' | \
                sed -E 's/FROM\s+//i' | tr -d '`"' | sort -u | tr '\n' ',' | sed 's/,$//')
            
            views+=("$(jq -n \
                --arg name "$view_name" \
                --arg mat "$is_materialized" \
                --arg refs "$referenced_tables" \
                '{
                    name: $name,
                    is_materialized: ($mat == "true"),
                    referenced_tables: $refs
                }')")
        fi
    done < <(awk '/CREATE\s+(OR\s+REPLACE\s+)?VIEW/I,/;/' "$file" 2>/dev/null)
    
    if [[ ${#views[@]} -gt 0 ]]; then
        printf '%s\n' "${views[@]}" | jq -s '.'
    else
        echo "[]"
    fi
}

#######################################
# Extract stored procedures and functions
# 
# Parses CREATE PROCEDURE/FUNCTION statements
#
# Arguments:
#   $1 - SQL file path
# Returns: JSON with procedure/function definitions
#######################################
extractor::lib::sql::extract_procedures() {
    local file="$1"
    local procedures=()
    
    # Extract procedures
    while IFS= read -r proc_def; do
        local proc_name=$(echo "$proc_def" | grep -iEo 'CREATE\s+(OR\s+REPLACE\s+)?PROCEDURE\s+[`"]?([a-zA-Z_][a-zA-Z0-9_\.]*)[`"]?' | \
            sed -E 's/CREATE\s+(OR\s+REPLACE\s+)?PROCEDURE\s+//i' | tr -d '`"' | head -1)
        
        if [[ -n "$proc_name" ]]; then
            # Extract parameter count
            local param_count=$(echo "$proc_def" | grep -o "," | wc -l)
            [[ "$param_count" -gt 0 ]] && param_count=$((param_count + 1))
            
            procedures+=("$(jq -n \
                --arg name "$proc_name" \
                --arg type "procedure" \
                --arg params "$param_count" \
                '{
                    name: $name,
                    type: $type,
                    parameter_count: ($params | tonumber)
                }')")
        fi
    done < <(awk '/CREATE\s+(OR\s+REPLACE\s+)?PROCEDURE/I,/END;?/I' "$file" 2>/dev/null)
    
    # Extract functions
    while IFS= read -r func_def; do
        local func_name=$(echo "$func_def" | grep -iEo 'CREATE\s+(OR\s+REPLACE\s+)?FUNCTION\s+[`"]?([a-zA-Z_][a-zA-Z0-9_\.]*)[`"]?' | \
            sed -E 's/CREATE\s+(OR\s+REPLACE\s+)?FUNCTION\s+//i' | tr -d '`"' | head -1)
        
        if [[ -n "$func_name" ]]; then
            # Extract return type if specified
            local return_type=$(echo "$func_def" | grep -iEo 'RETURNS\s+[A-Z]+' | sed -E 's/RETURNS\s+//i' || echo "")
            
            procedures+=("$(jq -n \
                --arg name "$func_name" \
                --arg type "function" \
                --arg ret "$return_type" \
                '{
                    name: $name,
                    type: $type,
                    return_type: $ret
                }')")
        fi
    done < <(awk '/CREATE\s+(OR\s+REPLACE\s+)?FUNCTION/I,/END;?/I' "$file" 2>/dev/null)
    
    if [[ ${#procedures[@]} -gt 0 ]]; then
        printf '%s\n' "${procedures[@]}" | jq -s '.'
    else
        echo "[]"
    fi
}

#######################################
# Extract queries from SQL
# 
# Extracts SELECT, INSERT, UPDATE, DELETE statements
#
# Arguments:
#   $1 - SQL file path
# Returns: JSON with query information
#######################################
extractor::lib::sql::extract_queries() {
    local file="$1"
    local queries=()
    
    # Count different query types
    local select_count=$(grep -ic "^\s*SELECT\s" "$file" 2>/dev/null || echo "0")
    local insert_count=$(grep -ic "^\s*INSERT\s" "$file" 2>/dev/null || echo "0")
    local update_count=$(grep -ic "^\s*UPDATE\s" "$file" 2>/dev/null || echo "0")
    local delete_count=$(grep -ic "^\s*DELETE\s" "$file" 2>/dev/null || echo "0")
    
    # Check for complex query patterns
    local has_joins="false"
    local has_subqueries="false"
    local has_cte="false"
    local has_window="false"
    
    grep -qi "JOIN\s" "$file" && has_joins="true"
    grep -qi "SELECT.*FROM.*SELECT" "$file" && has_subqueries="true"
    grep -qi "WITH\s.*AS\s*(" "$file" && has_cte="true"
    grep -qi "OVER\s*(" "$file" && has_window="true"
    
    jq -n \
        --arg select "$select_count" \
        --arg insert "$insert_count" \
        --arg update "$update_count" \
        --arg delete "$delete_count" \
        --arg joins "$has_joins" \
        --arg subq "$has_subqueries" \
        --arg cte "$has_cte" \
        --arg window "$has_window" \
        '{
            select_count: ($select | tonumber),
            insert_count: ($insert | tonumber),
            update_count: ($update | tonumber),
            delete_count: ($delete | tonumber),
            has_joins: ($joins == "true"),
            has_subqueries: ($subq == "true"),
            has_cte: ($cte == "true"),
            has_window_functions: ($window == "true")
        }'
}

#######################################
# Extract indexes from SQL
# 
# Parses CREATE INDEX statements
#
# Arguments:
#   $1 - SQL file path
# Returns: JSON with index definitions
#######################################
extractor::lib::sql::extract_indexes() {
    local file="$1"
    local indexes=()
    
    while IFS= read -r index_def; do
        # Extract index name and type
        local index_name=$(echo "$index_def" | grep -iEo 'CREATE\s+(UNIQUE\s+)?INDEX\s+[`"]?([a-zA-Z_][a-zA-Z0-9_]*)[`"]?' | \
            sed -E 's/CREATE\s+(UNIQUE\s+)?INDEX\s+//i' | tr -d '`"' | head -1)
        
        if [[ -n "$index_name" ]]; then
            local is_unique="false"
            echo "$index_def" | grep -qi "UNIQUE" && is_unique="true"
            
            # Extract table name
            local table_name=$(echo "$index_def" | grep -iEo 'ON\s+[`"]?([a-zA-Z_][a-zA-Z0-9_\.]*)[`"]?' | \
                sed -E 's/ON\s+//i' | tr -d '`"' | head -1)
            
            indexes+=("$(jq -n \
                --arg name "$index_name" \
                --arg table "$table_name" \
                --arg unique "$is_unique" \
                '{
                    name: $name,
                    table: $table,
                    is_unique: ($unique == "true")
                }')")
        fi
    done < <(grep -i "CREATE.*INDEX" "$file" 2>/dev/null)
    
    if [[ ${#indexes[@]} -gt 0 ]]; then
        printf '%s\n' "${indexes[@]}" | jq -s '.'
    else
        echo "[]"
    fi
}

#######################################
# Extract migration information
# 
# Detects migration patterns and versioning
#
# Arguments:
#   $1 - SQL file path
# Returns: JSON with migration info
#######################################
extractor::lib::sql::extract_migration_info() {
    local file="$1"
    local filename=$(basename "$file")
    
    # Check if it looks like a migration file
    local is_migration="false"
    local migration_version=""
    local migration_type=""
    
    # Common migration patterns
    if [[ "$filename" =~ ^[0-9]+[_-] ]]; then
        is_migration="true"
        migration_version=$(echo "$filename" | grep -oE '^[0-9]+' || echo "")
    elif [[ "$filename" =~ V[0-9]+__ ]]; then
        # Flyway pattern
        is_migration="true"
        migration_version=$(echo "$filename" | grep -oE 'V[0-9]+' | tr -d 'V' || echo "")
    fi
    
    # Detect migration type
    if [[ "$is_migration" == "true" ]]; then
        grep -qi "CREATE TABLE" "$file" && migration_type="schema"
        grep -qi "ALTER TABLE" "$file" && migration_type="${migration_type:+$migration_type,}alter"
        grep -qi "INSERT INTO\|UPDATE\|DELETE" "$file" && migration_type="${migration_type:+$migration_type,}data"
        [[ -z "$migration_type" ]] && migration_type="unknown"
    fi
    
    jq -n \
        --arg is_mig "$is_migration" \
        --arg version "$migration_version" \
        --arg type "$migration_type" \
        '{
            is_migration: ($is_mig == "true"),
            version: $version,
            type: $type
        }'
}

#######################################
# Extract all SQL components from a file
# 
# Main extraction function that combines all SQL extractions
#
# Arguments:
#   $1 - SQL file path
#   $2 - Component type (database, migrations, queries, etc.)
#   $3 - Scenario/resource name
# Returns: JSON lines with all SQL information
#######################################
extractor::lib::sql::extract_all() {
    local dir="$1"
    local component_type="${2:-database}"
    local scenario_name="${3:-unknown}"
    
    # Find all SQL files
    local sql_files=()
    while IFS= read -r file; do
        sql_files+=("$file")
    done < <(find "$dir" -type f \( -name "*.sql" -o -name "*.ddl" -o -name "*.dml" \) 2>/dev/null)
    
    if [[ ${#sql_files[@]} -eq 0 ]]; then
        return 1
    fi
    
    for file in "${sql_files[@]}"; do
        local filename=$(basename "$file")
        local relative_path="${file#$dir/}"
        
        # Get file statistics
        local line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        
        # Extract components
        local tables=$(extractor::lib::sql::extract_tables "$file")
        local views=$(extractor::lib::sql::extract_views "$file")
        local procedures=$(extractor::lib::sql::extract_procedures "$file")
        local indexes=$(extractor::lib::sql::extract_indexes "$file")
        local queries=$(extractor::lib::sql::extract_queries "$file")
        local migration_info=$(extractor::lib::sql::extract_migration_info "$file")
        
        # Count totals
        local table_count=$(echo "$tables" | jq 'length')
        local view_count=$(echo "$views" | jq 'length')
        local proc_count=$(echo "$procedures" | jq 'length')
        local index_count=$(echo "$indexes" | jq 'length')
        
        # Build content summary
        local content="SQL: $filename | Component: $component_type"
        [[ $table_count -gt 0 ]] && content="$content | Tables: $table_count"
        [[ $view_count -gt 0 ]] && content="$content | Views: $view_count"
        [[ $proc_count -gt 0 ]] && content="$content | Procedures/Functions: $proc_count"
        [[ $index_count -gt 0 ]] && content="$content | Indexes: $index_count"
        
        # Add query summary
        local select_count=$(echo "$queries" | jq '.select_count')
        [[ $select_count -gt 0 ]] && content="$content | Queries: $select_count SELECT"
        
        # Check if it's a migration
        local is_migration=$(echo "$migration_info" | jq -r '.is_migration')
        [[ "$is_migration" == "true" ]] && content="$content | Migration: v$(echo "$migration_info" | jq -r '.version')"
        
        # Output main file overview
        jq -n \
            --arg content "$content" \
            --arg scenario "$scenario_name" \
            --arg source_file "$file" \
            --arg relative_path "$relative_path" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg line_count "$line_count" \
            --arg file_size "$file_size" \
            --arg table_count "$table_count" \
            --arg view_count "$view_count" \
            --arg proc_count "$proc_count" \
            --arg index_count "$index_count" \
            --argjson tables "$tables" \
            --argjson views "$views" \
            --argjson procedures "$procedures" \
            --argjson indexes "$indexes" \
            --argjson queries "$queries" \
            --argjson migration "$migration_info" \
            '{
                content: $content,
                metadata: {
                    scenario: $scenario,
                    source_file: $source_file,
                    relative_path: $relative_path,
                    filename: $filename,
                    component_type: $component_type,
                    language: "sql",
                    line_count: ($line_count | tonumber),
                    file_size: ($file_size | tonumber),
                    table_count: ($table_count | tonumber),
                    view_count: ($view_count | tonumber),
                    procedure_count: ($proc_count | tonumber),
                    index_count: ($index_count | tonumber),
                    database_objects: {
                        tables: $tables,
                        views: $views,
                        procedures: $procedures,
                        indexes: $indexes
                    },
                    query_statistics: $queries,
                    migration_info: $migration,
                    content_type: "sql_schema",
                    extraction_method: "sql_parser"
                }
            }' | jq -c
        
        # Output individual table definitions as separate entries
        if [[ $table_count -gt 0 ]]; then
            echo "$tables" | jq -c '.[]' | while read -r table; do
                local table_name=$(echo "$table" | jq -r '.name')
                local table_content="Table: $table_name | Database: $filename"
                local col_count=$(echo "$table" | jq -r '.column_count')
                [[ $col_count -gt 0 ]] && table_content="$table_content | Columns: $col_count"
                
                jq -n \
                    --arg content "$table_content" \
                    --arg scenario "$scenario_name" \
                    --arg source_file "$file" \
                    --arg table_name "$table_name" \
                    --arg component_type "$component_type" \
                    --argjson table_def "$table" \
                    '{
                        content: $content,
                        metadata: {
                            scenario: $scenario,
                            source_file: $source_file,
                            component_type: $component_type,
                            language: "sql",
                            object_type: "table",
                            object_name: $table_name,
                            table_definition: $table_def,
                            content_type: "sql_table",
                            extraction_method: "sql_parser"
                        }
                    }' | jq -c
            done
        fi
        
        # Output individual stored procedures/functions as separate entries
        if [[ $proc_count -gt 0 ]]; then
            echo "$procedures" | jq -c '.[]' | while read -r proc; do
                local proc_name=$(echo "$proc" | jq -r '.name')
                local proc_type=$(echo "$proc" | jq -r '.type')
                local proc_content="SQL $proc_type: $proc_name | Database: $filename"
                
                jq -n \
                    --arg content "$proc_content" \
                    --arg scenario "$scenario_name" \
                    --arg source_file "$file" \
                    --arg proc_name "$proc_name" \
                    --arg proc_type "$proc_type" \
                    --arg component_type "$component_type" \
                    --argjson proc_def "$proc" \
                    '{
                        content: $content,
                        metadata: {
                            scenario: $scenario,
                            source_file: $source_file,
                            component_type: $component_type,
                            language: "sql",
                            object_type: $proc_type,
                            object_name: $proc_name,
                            procedure_definition: $proc_def,
                            content_type: "sql_procedure",
                            extraction_method: "sql_parser"
                        }
                    }' | jq -c
            done
        fi
    done
}

#######################################
# Check if directory contains SQL files
# 
# Helper function to detect SQL presence
#
# Arguments:
#   $1 - Directory path
# Returns: 0 if SQL files found, 1 otherwise
#######################################
extractor::lib::sql::has_sql_files() {
    local dir="$1"
    
    if find "$dir" -type f \( -name "*.sql" -o -name "*.ddl" -o -name "*.dml" \) 2>/dev/null | grep -q .; then
        return 0
    else
        return 1
    fi
}

# Export all functions
export -f extractor::lib::sql::extract_tables
export -f extractor::lib::sql::extract_views
export -f extractor::lib::sql::extract_procedures
export -f extractor::lib::sql::extract_queries
export -f extractor::lib::sql::extract_indexes
export -f extractor::lib::sql::extract_migration_info
export -f extractor::lib::sql::extract_all
export -f extractor::lib::sql::has_sql_files