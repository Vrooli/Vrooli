#!/bin/bash
# PostGIS Status Functions

# Get script directory
POSTGIS_STATUS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${POSTGIS_STATUS_LIB_DIR}/common.sh"

# Get PostGIS status
postgis_status() {
    local format_type="${1:-text}"
    local verbose="${2:-false}"
    
    # Initialize
    postgis_init_dirs
    
    # Check PostgreSQL connection
    local postgres_available=false
    if postgis_check_postgres; then
        postgres_available=true
    fi
    
    # Check PostGIS installation
    local postgis_installed=false
    if [ "$postgres_available" = "true" ] && postgis_is_installed; then
        postgis_installed=true
    fi
    
    # Check if enabled in default database
    local postgis_enabled=false
    local postgis_version="not_installed"
    if [ "$postgis_installed" = "true" ]; then
        if postgis_is_enabled "$POSTGIS_PG_DATABASE"; then
            postgis_enabled=true
            postgis_version=$(postgis_get_version "$POSTGIS_PG_DATABASE")
        fi
    fi
    
    # Determine health status
    local health="unhealthy"
    local health_message="PostGIS not configured"
    
    if [ "$postgres_available" = "false" ]; then
        health="unhealthy"
        health_message="PostgreSQL not available"
    elif [ "$postgis_installed" = "false" ]; then
        health="warning"
        health_message="PostGIS extension not installed in PostgreSQL"
    elif [ "$postgis_enabled" = "false" ]; then
        health="warning"
        health_message="PostGIS not enabled in database $POSTGIS_PG_DATABASE"
    else
        health="healthy"
        health_message="PostGIS is installed and enabled"
    fi
    
    # Count spatial tables if enabled
    local spatial_tables=0
    if [ "$postgis_enabled" = "true" ]; then
        spatial_tables=$(PGPASSWORD="$POSTGIS_PG_PASSWORD" psql -h "$POSTGIS_PG_HOST" -p "$POSTGIS_PG_PORT" -U "$POSTGIS_PG_USER" -d "$POSTGIS_PG_DATABASE" \
            -c "SELECT COUNT(*) FROM geometry_columns" -t -A 2>/dev/null || echo "0")
    fi
    
    # Format output
    if [ "$format_type" = "json" ]; then
        cat <<EOF
{
  "name": "$POSTGIS_RESOURCE_NAME",
  "category": "$POSTGIS_RESOURCE_CATEGORY",
  "description": "$POSTGIS_RESOURCE_DESCRIPTION",
  "installed": $postgis_installed,
  "running": $postgis_enabled,
  "healthy": "$health",
  "health_message": "$health_message",
  "version": "$postgis_version",
  "postgres_host": "$POSTGIS_PG_HOST",
  "postgres_port": $POSTGIS_PG_PORT,
  "database": "$POSTGIS_PG_DATABASE",
  "spatial_tables": $spatial_tables,
  "extensions": "$POSTGIS_EXTENSIONS",
  "data_dir": "$POSTGIS_DATA_DIR"
}
EOF
    else
        log::header "PostGIS Status"
        log::info "Description: $POSTGIS_RESOURCE_DESCRIPTION"
        log::info "Category: $POSTGIS_RESOURCE_CATEGORY"
        
        echo
        log::info "Basic Status:"
        if [ "$postgis_installed" = "true" ]; then
            log::success "Installed: Yes"
        else
            log::warning "Installed: No (extension not in PostgreSQL)"
        fi
        
        if [ "$postgis_enabled" = "true" ]; then
            log::success "Enabled: Yes (in $POSTGIS_PG_DATABASE)"
        else
            log::warning "Enabled: No"
        fi
        
        if [ "$health" = "healthy" ]; then
            log::success "Health: $health"
        elif [ "$health" = "warning" ]; then
            log::warning "Health: $health"
        else
            log::error "Health: $health"
        fi
        
        echo
        log::info "Configuration:"
        log::info "   PostgreSQL Host: $POSTGIS_PG_HOST:$POSTGIS_PG_PORT"
        log::info "   Database: $POSTGIS_PG_DATABASE"
        log::info "   Version: $postgis_version"
        log::info "   Extensions: $POSTGIS_EXTENSIONS"
        log::info "   Spatial Tables: $spatial_tables"
        log::info "   Default SRID: $POSTGIS_DEFAULT_SRID"
        log::info "   Data Directory: $POSTGIS_DATA_DIR"
        
        echo
        log::info "Status Message:"
        if [ "$health" = "healthy" ]; then
            log::success "$health_message"
        elif [ "$health" = "warning" ]; then
            log::warning "$health_message"
        else
            log::error "$health_message"
        fi
        
        if [ "$verbose" = "true" ] && [ "$postgis_enabled" = "true" ]; then
            echo
            log::info "Spatial Reference Systems:"
            local srid_count=$(PGPASSWORD="$POSTGIS_PG_PASSWORD" psql -h "$POSTGIS_PG_HOST" -p "$POSTGIS_PG_PORT" -U "$POSTGIS_PG_USER" -d "$POSTGIS_PG_DATABASE" \
                -c "SELECT COUNT(*) FROM spatial_ref_sys" -t -A 2>/dev/null || echo "0")
            log::info "   Total SRIDs: $srid_count"
        fi
    fi
}

# Export function for use by other scripts
export -f postgis_status