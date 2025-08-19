#!/bin/bash
# PostGIS Installation Functions

# Get script directory
POSTGIS_INSTALL_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${POSTGIS_INSTALL_LIB_DIR}/common.sh"

# Install PostGIS extension in PostgreSQL
postgis_install() {
    log::header "Installing PostGIS"
    
    # Initialize directories
    postgis_init_dirs
    
    # Check PostgreSQL availability
    log::info "Checking PostgreSQL connection..."
    if ! postgis_check_postgres; then
        log::error "PostgreSQL is not available. Please ensure postgres resource is running."
        return 1
    fi
    log::success "PostgreSQL is available"
    
    # Check if PostGIS extension is available
    log::info "Checking PostGIS availability in PostgreSQL..."
    if ! postgis_is_installed; then
        log::warning "PostGIS extension not found in PostgreSQL"
        log::info "PostGIS must be installed at the PostgreSQL server level"
        log::info "For Docker-based PostgreSQL, use postgis/postgis image"
        log::info "For system PostgreSQL, install postgresql-XX-postgis-3 package"
        
        # Try to use postgres resource to enable PostGIS
        log::info "Attempting to use postgres resource to enable PostGIS..."
        if command -v resource-postgres >/dev/null 2>&1; then
            if resource-postgres enable-extension postgis; then
                log::success "PostGIS extension enabled via postgres resource"
            else
                log::warning "Could not enable PostGIS via postgres resource"
                log::info "You may need to manually install PostGIS in your PostgreSQL instance"
            fi
        fi
    else
        log::success "PostGIS extension is available in PostgreSQL"
    fi
    
    # Enable PostGIS in default database
    log::info "Enabling PostGIS in database: $POSTGIS_PG_DATABASE"
    if postgis_enable_database "$POSTGIS_PG_DATABASE"; then
        log::success "PostGIS enabled successfully"
    else
        log::error "Failed to enable PostGIS"
        return 1
    fi
    
    # Create sample spatial data
    log::info "Creating sample spatial tables..."
    create_sample_data
    
    log::success "PostGIS installation complete"
    return 0
}

# Uninstall PostGIS (disable in databases, keep PostgreSQL extension)
postgis_uninstall() {
    log::header "Uninstalling PostGIS"
    
    log::info "Disabling PostGIS in database: $POSTGIS_PG_DATABASE"
    if postgis_disable_database "$POSTGIS_PG_DATABASE"; then
        log::success "PostGIS disabled successfully"
    else
        log::warning "Failed to disable PostGIS or already disabled"
    fi
    
    log::info "Note: PostGIS extension remains available in PostgreSQL"
    log::info "To fully remove, uninstall at PostgreSQL server level"
    
    log::success "PostGIS uninstall complete"
    return 0
}

# Create sample spatial data for testing
create_sample_data() {
    local sql_file="$POSTGIS_SQL_DIR/sample_data.sql"
    
    cat > "$sql_file" <<'EOF'
-- Create a sample table with geographic data
CREATE TABLE IF NOT EXISTS sample_locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    description TEXT,
    location GEOGRAPHY(Point, 4326),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data (major cities)
INSERT INTO sample_locations (name, description, location) VALUES
    ('New York', 'New York City, USA', ST_GeogFromText('POINT(-74.006 40.7128)')),
    ('London', 'London, UK', ST_GeogFromText('POINT(-0.1276 51.5074)')),
    ('Tokyo', 'Tokyo, Japan', ST_GeogFromText('POINT(139.6503 35.6762)')),
    ('Paris', 'Paris, France', ST_GeogFromText('POINT(2.3522 48.8566)')),
    ('Sydney', 'Sydney, Australia', ST_GeogFromText('POINT(151.2093 -33.8688)'))
ON CONFLICT DO NOTHING;

-- Create spatial index
CREATE INDEX IF NOT EXISTS idx_sample_locations_geography 
ON sample_locations USING GIST (location);

-- Create a view for distance calculations from NYC
CREATE OR REPLACE VIEW distances_from_nyc AS
SELECT 
    name,
    ST_Distance(
        location,
        ST_GeogFromText('POINT(-74.006 40.7128)')
    ) / 1000 as distance_km
FROM sample_locations
WHERE name != 'New York'
ORDER BY distance_km;
EOF
    
    if postgis_execute_sql "$sql_file" "$POSTGIS_PG_DATABASE"; then
        format_success "Sample spatial data created"
        
        # Show sample query
        format_info "Sample query - Cities within 5000km of NYC:"
        PGPASSWORD="$POSTGIS_PG_PASSWORD" psql -h "$POSTGIS_PG_HOST" -p "$POSTGIS_PG_PORT" \
            -U "$POSTGIS_PG_USER" -d "$POSTGIS_PG_DATABASE" \
            -c "SELECT name, ROUND(distance_km::numeric, 2) as distance_km FROM distances_from_nyc WHERE distance_km < 5000" 2>/dev/null
    else
        format_warning "Failed to create sample data"
    fi
}

# Export functions
export -f postgis_install
export -f postgis_uninstall