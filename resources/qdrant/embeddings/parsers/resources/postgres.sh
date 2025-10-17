#!/usr/bin/env bash
# PostgreSQL Parser for Qdrant Embeddings
# Extracts semantic information from PostgreSQL SQL files
#
# Handles:
# - Schema definitions (tables, indexes, constraints)
# - Migrations and version control
# - Seed data and fixtures
# - Functions and stored procedures
# - Database initialization patterns

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract schema information
# 
# Analyzes table definitions and structure
#
# Arguments:
#   $1 - Path to SQL file
# Returns: JSON with schema information
#######################################
extractor::lib::postgres::extract_schema() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract table definitions
    local tables=$(grep -iE "CREATE\s+TABLE(\s+IF\s+NOT\s+EXISTS)?\s+[\"']?([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*CREATE\s+TABLE(\s+IF\s+NOT\s+EXISTS)?\s+[\"'\'']*([a-zA-Z_][a-zA-Z0-9_]*).*/\2/' | \
        sort -u | jq -R . | jq -s '.')
    
    # Count different schema objects
    local table_count=$(echo "$tables" | jq 'length')
    local index_count=$(grep -ic "CREATE\s\+INDEX" "$file" 2>/dev/null || echo "0")
    local view_count=$(grep -ic "CREATE\s\+VIEW" "$file" 2>/dev/null || echo "0")
    local trigger_count=$(grep -ic "CREATE\s\+TRIGGER" "$file" 2>/dev/null || echo "0")
    local constraint_count=$(grep -ic "CONSTRAINT\s\+" "$file" 2>/dev/null || echo "0")
    
    # Check for specific patterns
    local has_foreign_keys="false"
    if grep -qiE "FOREIGN\s+KEY|REFERENCES" "$file" 2>/dev/null; then
        has_foreign_keys="true"
    fi
    
    local has_primary_keys="false"
    if grep -qiE "PRIMARY\s+KEY" "$file" 2>/dev/null; then
        has_primary_keys="true"
    fi
    
    local has_unique_constraints="false"
    if grep -qiE "UNIQUE" "$file" 2>/dev/null; then
        has_unique_constraints="true"
    fi
    
    jq -n \
        --argjson tables "$tables" \
        --arg table_count "$table_count" \
        --arg index_count "$index_count" \
        --arg view_count "$view_count" \
        --arg trigger_count "$trigger_count" \
        --arg constraint_count "$constraint_count" \
        --arg foreign_keys "$has_foreign_keys" \
        --arg primary_keys "$has_primary_keys" \
        --arg unique_constraints "$has_unique_constraints" \
        '{
            tables: $tables,
            table_count: ($table_count | tonumber),
            index_count: ($index_count | tonumber),
            view_count: ($view_count | tonumber),
            trigger_count: ($trigger_count | tonumber),
            constraint_count: ($constraint_count | tonumber),
            has_foreign_keys: ($foreign_keys == "true"),
            has_primary_keys: ($primary_keys == "true"),
            has_unique_constraints: ($unique_constraints == "true")
        }'
}

#######################################
# Extract migration information
# 
# Detects migration patterns and versioning
#
# Arguments:
#   $1 - Path to SQL file
# Returns: JSON with migration information
#######################################
extractor::lib::postgres::extract_migration() {
    local file="$1"
    local filename=$(basename "$file")
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Check if filename suggests migration
    local is_migration="false"
    local migration_version=""
    local migration_name=""
    
    # Common migration patterns
    if [[ "$filename" =~ ^([0-9]+)[_-](.*)\.sql$ ]]; then
        is_migration="true"
        migration_version="${BASH_REMATCH[1]}"
        migration_name="${BASH_REMATCH[2]}"
    elif [[ "$filename" =~ ^V([0-9]+[._0-9]*)__(.*)\.sql$ ]]; then
        # Flyway pattern
        is_migration="true"
        migration_version="${BASH_REMATCH[1]}"
        migration_name="${BASH_REMATCH[2]}"
    elif [[ "$filename" == *"migration"* ]] || [[ "$filename" == *"migrate"* ]]; then
        is_migration="true"
    fi
    
    # Check for rollback/down migration
    local has_rollback="false"
    if grep -qiE "^--\s*DOWN|^--\s*ROLLBACK|DROP\s+TABLE\s+IF\s+EXISTS" "$file" 2>/dev/null; then
        has_rollback="true"
    fi
    
    # Check for transactional migration
    local is_transactional="false"
    if grep -qiE "BEGIN\s*;|START\s+TRANSACTION" "$file" 2>/dev/null && \
       grep -qiE "COMMIT\s*;" "$file" 2>/dev/null; then
        is_transactional="true"
    fi
    
    # Detect migration type
    local migration_type="unknown"
    if grep -qiE "CREATE\s+TABLE" "$file" 2>/dev/null; then
        migration_type="schema_creation"
    elif grep -qiE "ALTER\s+TABLE" "$file" 2>/dev/null; then
        migration_type="schema_alteration"
    elif grep -qiE "INSERT\s+INTO" "$file" 2>/dev/null; then
        migration_type="data_migration"
    elif grep -qiE "CREATE\s+INDEX" "$file" 2>/dev/null; then
        migration_type="index_creation"
    fi
    
    jq -n \
        --arg is_migration "$is_migration" \
        --arg version "$migration_version" \
        --arg name "$migration_name" \
        --arg has_rollback "$has_rollback" \
        --arg is_transactional "$is_transactional" \
        --arg type "$migration_type" \
        '{
            is_migration: ($is_migration == "true"),
            version: $version,
            name: $name,
            has_rollback: ($has_rollback == "true"),
            is_transactional: ($is_transactional == "true"),
            type: $type
        }'
}

#######################################
# Extract functions and procedures
# 
# Analyzes stored procedures and functions
#
# Arguments:
#   $1 - Path to SQL file
# Returns: JSON with function information
#######################################
extractor::lib::postgres::extract_functions() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract function names
    local functions=$(grep -iE "CREATE\s+(OR\s+REPLACE\s+)?FUNCTION\s+[\"']?([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*FUNCTION\s+[\"'\'']*([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' | \
        sort -u | jq -R . | jq -s '.')
    
    # Extract procedure names
    local procedures=$(grep -iE "CREATE\s+(OR\s+REPLACE\s+)?PROCEDURE\s+[\"']?([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*PROCEDURE\s+[\"'\'']*([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' | \
        sort -u | jq -R . | jq -s '.')
    
    # Count functions and procedures
    local function_count=$(echo "$functions" | jq 'length')
    local procedure_count=$(echo "$procedures" | jq 'length')
    
    # Detect function languages
    local has_plpgsql="false"
    local has_sql="false"
    local has_plpython="false"
    
    if grep -qiE "LANGUAGE\s+plpgsql" "$file" 2>/dev/null; then
        has_plpgsql="true"
    fi
    if grep -qiE "LANGUAGE\s+sql" "$file" 2>/dev/null; then
        has_sql="true"
    fi
    if grep -qiE "LANGUAGE\s+(plpython|python)" "$file" 2>/dev/null; then
        has_plpython="true"
    fi
    
    jq -n \
        --argjson functions "$functions" \
        --argjson procedures "$procedures" \
        --arg function_count "$function_count" \
        --arg procedure_count "$procedure_count" \
        --arg has_plpgsql "$has_plpgsql" \
        --arg has_sql "$has_sql" \
        --arg has_plpython "$has_plpython" \
        '{
            functions: $functions,
            procedures: $procedures,
            function_count: ($function_count | tonumber),
            procedure_count: ($procedure_count | tonumber),
            has_plpgsql: ($has_plpgsql == "true"),
            has_sql_functions: ($has_sql == "true"),
            has_plpython: ($has_plpython == "true")
        }'
}

#######################################
# Extract data operations
# 
# Analyzes DML operations (INSERT, UPDATE, DELETE)
#
# Arguments:
#   $1 - Path to SQL file
# Returns: JSON with data operation information
#######################################
extractor::lib::postgres::extract_data_operations() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Count different DML operations
    local insert_count=$(grep -ic "INSERT\s\+INTO" "$file" 2>/dev/null || echo "0")
    local update_count=$(grep -ic "UPDATE\s\+" "$file" 2>/dev/null || echo "0")
    local delete_count=$(grep -ic "DELETE\s\+FROM" "$file" 2>/dev/null || echo "0")
    local truncate_count=$(grep -ic "TRUNCATE\s\+" "$file" 2>/dev/null || echo "0")
    
    # Check for bulk operations
    local has_bulk_insert="false"
    if grep -qiE "INSERT\s+INTO.*VALUES\s*\([^)]+\)\s*,\s*\(" "$file" 2>/dev/null; then
        has_bulk_insert="true"
    fi
    
    # Check for COPY operations
    local has_copy="false"
    if grep -qiE "COPY\s+" "$file" 2>/dev/null; then
        has_copy="true"
    fi
    
    # Check for upsert patterns
    local has_upsert="false"
    if grep -qiE "ON\s+CONFLICT" "$file" 2>/dev/null; then
        has_upsert="true"
    fi
    
    # Determine if it's seed data
    local is_seed_data="false"
    local filename=$(basename "$file")
    if [[ "$filename" == *"seed"* ]] || [[ "$filename" == *"fixture"* ]] || \
       [[ "$filename" == *"sample"* ]] || [[ "$filename" == *"test"* ]]; then
        is_seed_data="true"
    elif [[ $insert_count -gt 5 && $update_count -eq 0 && $delete_count -eq 0 ]]; then
        # Heuristic: many inserts with no updates/deletes suggests seed data
        is_seed_data="true"
    fi
    
    jq -n \
        --arg insert "$insert_count" \
        --arg update "$update_count" \
        --arg delete "$delete_count" \
        --arg truncate "$truncate_count" \
        --arg bulk_insert "$has_bulk_insert" \
        --arg copy "$has_copy" \
        --arg upsert "$has_upsert" \
        --arg seed "$is_seed_data" \
        '{
            insert_count: ($insert | tonumber),
            update_count: ($update | tonumber),
            delete_count: ($delete | tonumber),
            truncate_count: ($truncate | tonumber),
            has_bulk_insert: ($bulk_insert == "true"),
            has_copy_operations: ($copy == "true"),
            has_upsert: ($upsert == "true"),
            is_seed_data: ($seed == "true")
        }'
}

#######################################
# Analyze SQL file purpose
# 
# Determines the primary purpose of the SQL file
#
# Arguments:
#   $1 - Path to SQL file
# Returns: JSON with purpose analysis
#######################################
extractor::lib::postgres::analyze_purpose() {
    local file="$1"
    local purposes=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"purposes": [], "primary_purpose": "unknown"}'
        return
    fi
    
    local filename=$(basename "$file" | tr '[:upper:]' '[:lower:]')
    local content=$(cat "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]')
    
    # Check filename patterns
    if [[ "$filename" == *"schema"* ]] || [[ "$filename" == *"structure"* ]]; then
        purposes+=("schema_definition")
    elif [[ "$filename" == *"migration"* ]] || [[ "$filename" =~ ^[0-9]+.*\.sql$ ]]; then
        purposes+=("migration")
    elif [[ "$filename" == *"seed"* ]] || [[ "$filename" == *"fixture"* ]]; then
        purposes+=("seed_data")
    elif [[ "$filename" == *"init"* ]] || [[ "$filename" == *"setup"* ]]; then
        purposes+=("initialization")
    elif [[ "$filename" == *"function"* ]] || [[ "$filename" == *"procedure"* ]]; then
        purposes+=("stored_procedures")
    elif [[ "$filename" == *"trigger"* ]]; then
        purposes+=("triggers")
    elif [[ "$filename" == *"view"* ]]; then
        purposes+=("views")
    elif [[ "$filename" == *"index"* ]]; then
        purposes+=("indexes")
    fi
    
    # Check content patterns
    if echo "$content" | grep -q "create table"; then
        purposes+=("table_creation")
    fi
    
    if echo "$content" | grep -q "alter table"; then
        purposes+=("schema_modification")
    fi
    
    if echo "$content" | grep -q "create extension"; then
        purposes+=("extension_setup")
    fi
    
    if echo "$content" | grep -q "grant\|revoke"; then
        purposes+=("permissions")
    fi
    
    if echo "$content" | grep -qE "create (or replace )?function"; then
        purposes+=("function_definition")
    fi
    
    if echo "$content" | grep -q "create trigger"; then
        purposes+=("trigger_definition")
    fi
    
    if echo "$content" | grep -q "create view"; then
        purposes+=("view_definition")
    fi
    
    # Determine primary purpose
    local primary_purpose="general_sql"
    if [[ ${#purposes[@]} -gt 0 ]]; then
        primary_purpose="${purposes[0]}"
    fi
    
    local purposes_json=$(printf '%s\n' "${purposes[@]}" | sort -u | jq -R . | jq -s '.')
    
    jq -n \
        --argjson purposes "$purposes_json" \
        --arg primary "$primary_purpose" \
        '{
            purposes: $purposes,
            primary_purpose: $primary
        }'
}

#######################################
# Extract all PostgreSQL information
# 
# Main extraction function that combines all analyses
#
# Arguments:
#   $1 - SQL file path or directory
#   $2 - Component type (migration, schema, seed, etc.)
#   $3 - Resource name
# Returns: JSON lines with all SQL information
#######################################
extractor::lib::postgres::extract_all() {
    local path="$1"
    local component_type="${2:-sql}"
    local resource_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        local file_ext="${filename##*.}"
        
        # Check if it's a SQL file
        case "$file_ext" in
            sql|ddl|dml)
                ;;
            *)
                return 1
                ;;
        esac
        
        # Get file statistics
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        local line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
        
        # Extract all components
        local schema=$(extractor::lib::postgres::extract_schema "$file")
        local migration=$(extractor::lib::postgres::extract_migration "$file")
        local functions=$(extractor::lib::postgres::extract_functions "$file")
        local data_ops=$(extractor::lib::postgres::extract_data_operations "$file")
        local purpose=$(extractor::lib::postgres::analyze_purpose "$file")
        
        # Get key metrics
        local table_count=$(echo "$schema" | jq -r '.table_count')
        local is_migration=$(echo "$migration" | jq -r '.is_migration')
        local primary_purpose=$(echo "$purpose" | jq -r '.primary_purpose')
        local is_seed=$(echo "$data_ops" | jq -r '.is_seed_data')
        
        # Build content summary
        local content="PostgreSQL: $filename | Type: $component_type | Resource: $resource_name"
        content="$content | Purpose: $primary_purpose"
        
        if [[ "$is_migration" == "true" ]]; then
            local version=$(echo "$migration" | jq -r '.version')
            [[ -n "$version" && "$version" != "null" ]] && content="$content | Version: $version"
        fi
        
        [[ $table_count -gt 0 ]] && content="$content | Tables: $table_count"
        
        local function_count=$(echo "$functions" | jq -r '.function_count')
        [[ $function_count -gt 0 ]] && content="$content | Functions: $function_count"
        
        [[ "$is_seed" == "true" ]] && content="$content | Contains Seed Data"
        
        # Output comprehensive SQL analysis
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg file_size "$file_size" \
            --arg line_count "$line_count" \
            --argjson schema "$schema" \
            --argjson migration "$migration" \
            --argjson functions "$functions" \
            --argjson data_ops "$data_ops" \
            --argjson purpose "$purpose" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    filename: $filename,
                    component_type: $component_type,
                    database_type: "postgresql",
                    file_size: ($file_size | tonumber),
                    line_count: ($line_count | tonumber),
                    schema: $schema,
                    migration: $migration,
                    functions: $functions,
                    data_operations: $data_ops,
                    purpose: $purpose,
                    content_type: "postgres_sql",
                    extraction_method: "postgres_parser"
                }
            }' | jq -c
            
        # Output entries for each table (for better searchability)
        echo "$schema" | jq -r '.tables[]' 2>/dev/null | while read -r table; do
            [[ -z "$table" || "$table" == "null" ]] && continue
            
            local table_content="PostgreSQL Table: $table | File: $filename | Resource: $resource_name"
            
            jq -n \
                --arg content "$table_content" \
                --arg resource "$resource_name" \
                --arg source_file "$file" \
                --arg table_name "$table" \
                --arg component_type "$component_type" \
                '{
                    content: $content,
                    metadata: {
                        resource: $resource,
                        source_file: $source_file,
                        table_name: $table_name,
                        component_type: $component_type,
                        content_type: "postgres_table",
                        extraction_method: "postgres_parser"
                    }
                }' | jq -c
        done
        
        # Output entries for each function (for better searchability)
        echo "$functions" | jq -r '.functions[]' 2>/dev/null | while read -r func; do
            [[ -z "$func" || "$func" == "null" ]] && continue
            
            local func_content="PostgreSQL Function: $func | File: $filename | Resource: $resource_name"
            
            jq -n \
                --arg content "$func_content" \
                --arg resource "$resource_name" \
                --arg source_file "$file" \
                --arg function_name "$func" \
                --arg component_type "$component_type" \
                '{
                    content: $content,
                    metadata: {
                        resource: $resource,
                        source_file: $source_file,
                        function_name: $function_name,
                        component_type: $component_type,
                        content_type: "postgres_function",
                        extraction_method: "postgres_parser"
                    }
                }' | jq -c
        done
        
    elif [[ -d "$path" ]]; then
        # Directory - find all SQL files
        local sql_files=()
        while IFS= read -r file; do
            sql_files+=("$file")
        done < <(find "$path" -type f \( -name "*.sql" -o -name "*.ddl" -o -name "*.dml" \) 2>/dev/null)
        
        if [[ ${#sql_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        # Sort files to process migrations in order
        IFS=$'\n' sql_files=($(sort <<< "${sql_files[*]}"))
        
        for file in "${sql_files[@]}"; do
            extractor::lib::postgres::extract_all "$file" "$component_type" "$resource_name"
        done
    fi
}

#######################################
# Check if file is a PostgreSQL file
# 
# Basic validation for SQL files
#
# Arguments:
#   $1 - File path
# Returns: 0 if PostgreSQL file, 1 otherwise
#######################################
extractor::lib::postgres::is_sql_file() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local file_ext="${file##*.}"
    
    case "$file_ext" in
        sql|ddl|dml)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Export all functions
export -f extractor::lib::postgres::extract_schema
export -f extractor::lib::postgres::extract_migration
export -f extractor::lib::postgres::extract_functions
export -f extractor::lib::postgres::extract_data_operations
export -f extractor::lib::postgres::analyze_purpose
export -f extractor::lib::postgres::extract_all
export -f extractor::lib::postgres::is_sql_file