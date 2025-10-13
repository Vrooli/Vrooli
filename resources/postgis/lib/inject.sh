#!/bin/bash
# PostGIS Injection Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POSTGIS_INJECT_LIB_DIR="${APP_ROOT}/resources/postgis/lib"

# Source dependencies
source "${POSTGIS_INJECT_LIB_DIR}/common.sh"

# Inject SQL files into PostGIS database
postgis_inject() {
    local source_path="$1"
    local database="${2:-$POSTGIS_PG_DATABASE}"
    
    if [ -z "$source_path" ]; then
        log::error "No source path provided"
        return 1
    fi
    
    if [ ! -e "$source_path" ]; then
        log::error "Source path does not exist: $source_path"
        return 1
    fi
    
    # Ensure PostGIS is enabled in target database
    if ! postgis_is_enabled "$database"; then
        log::warning "PostGIS not enabled in database $database, enabling now..."
        if ! postgis_enable_database "$database"; then
            log::error "Failed to enable PostGIS in database"
            return 1
        fi
    fi
    
    # Handle single file
    if [ -f "$source_path" ]; then
        if [[ "$source_path" == *.sql ]]; then
            log::info "Injecting SQL file: $source_path"
            postgis_execute_sql "$source_path" "$database"
            return $?
        else
            # Use universal GIS importer for all spatial formats
            log::info "Importing GIS file: $source_path"
            postgis_import_gis "$source_path" "$database"
            return $?
        fi
    fi
    
    # Handle directory
    if [ -d "$source_path" ]; then
        format_header "Injecting directory: $source_path"
        
        local success=0
        local failed=0
        
        # Process SQL files
        while IFS= read -r -d '' sql_file; do
            format_info "Processing: $(basename "$sql_file")"
            if postgis_execute_sql "$sql_file" "$database"; then
                ((success++))
            else
                ((failed++))
            fi
        done < <(find "$source_path" -name "*.sql" -type f -print0 | sort -z)
        
        # Process all supported GIS formats
        local gis_extensions="shp geojson json kml kmz gpx csv"
        for ext in $gis_extensions; do
            while IFS= read -r -d '' gis_file; do
                format_info "Processing: $(basename "$gis_file")"
                if postgis_import_gis "$gis_file" "$database"; then
                    ((success++))
                else
                    ((failed++))
                fi
            done < <(find "$source_path" -name "*.$ext" -type f -print0 | sort -z)
        done
        
        format_info "Injection complete: $success succeeded, $failed failed"
        
        if [ $failed -gt 0 ]; then
            return 1
        fi
        return 0
    fi
    
    format_error "Invalid source path: $source_path"
    return 1
}

# List injected data
postgis_list_injected() {
    local database="${1:-$POSTGIS_PG_DATABASE}"
    
    format_header "Spatial Tables in $database"
    
    if ! postgis_is_enabled "$database"; then
        format_warning "PostGIS not enabled in database $database"
        return 1
    fi
    
    # List tables with geometry columns
    PGPASSWORD="$POSTGIS_PG_PASSWORD" psql -h "$POSTGIS_PG_HOST" -p "$POSTGIS_PG_PORT" \
        -U "$POSTGIS_PG_USER" -d "$database" \
        -c "SELECT f_table_schema, f_table_name, f_geometry_column, type, srid 
            FROM geometry_columns 
            ORDER BY f_table_schema, f_table_name" 2>/dev/null
    
    # Count total spatial objects
    local total
    total=$(PGPASSWORD="$POSTGIS_PG_PASSWORD" psql -h "$POSTGIS_PG_HOST" -p "$POSTGIS_PG_PORT" \
        -U "$POSTGIS_PG_USER" -d "$database" \
        -c "SELECT COUNT(*) FROM geometry_columns" -t -A 2>/dev/null || echo "0")
    
    format_info "Total spatial tables: $total"
    return 0
}

# Export functions
export -f postgis_inject
export -f postgis_list_injected