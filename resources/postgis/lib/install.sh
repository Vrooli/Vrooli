#!/bin/bash
# PostGIS Standalone Installation Functions

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# PostGIS Docker configuration
# Check if custom Dockerfile exists for enhanced image with pgRouting
if [[ -f "${APP_ROOT}/resources/postgis/Dockerfile" ]]; then
    POSTGIS_IMAGE="vrooli/postgis-routing:16-3.4"
    POSTGIS_USE_CUSTOM_IMAGE=true
else
    POSTGIS_IMAGE="postgis/postgis:16-3.4-alpine"
    POSTGIS_USE_CUSTOM_IMAGE=false
fi
POSTGIS_CONTAINER="postgis-main"
POSTGIS_STANDALONE_PORT="${POSTGIS_STANDALONE_PORT:-5434}"
POSTGIS_INSTALL_LIB_DIR="${APP_ROOT}/resources/postgis/lib"

# Source var.sh for access to project variables
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source dependencies
source "${POSTGIS_INSTALL_LIB_DIR}/common.sh"

# Install PostGIS as standalone container
postgis_install() {
    log::header "Installing PostGIS (Standalone)"
    
    # Initialize directories
    postgis_init_dirs
    
    # Check if container already exists
    if docker ps -a --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        log::info "PostGIS container already exists"
        
        # Start if not running
        if ! docker ps --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
            log::info "Starting PostGIS container..."
            docker start "${POSTGIS_CONTAINER}"
            sleep 5
        fi
        
        log::success "PostGIS is running on port ${POSTGIS_STANDALONE_PORT}"
        return 0
    fi
    
    # Create Docker network if not exists
    if ! docker network ls --format "{{.Name}}" | grep -q "^vrooli-network$"; then
        log::info "Creating Docker network: vrooli-network"
        docker network create vrooli-network 2>/dev/null || true
    fi
    
    # Build or pull PostGIS image
    if [[ "$POSTGIS_USE_CUSTOM_IMAGE" == "true" ]]; then
        log::info "Building custom PostGIS image with pgRouting support..."
        if ! docker build -t "${POSTGIS_IMAGE}" "${APP_ROOT}/resources/postgis" 2>&1 | tail -5; then
            log::warning "Failed to build custom image, falling back to standard image"
            POSTGIS_IMAGE="postgis/postgis:16-3.4-alpine"
            docker pull "${POSTGIS_IMAGE}" || {
                log::error "Failed to pull PostGIS image"
                return 1
            }
        else
            log::success "Custom PostGIS image with pgRouting built successfully"
        fi
    else
        log::info "Pulling PostGIS image: ${POSTGIS_IMAGE}"
        docker pull "${POSTGIS_IMAGE}" || {
            log::error "Failed to pull PostGIS image"
            return 1
        }
    fi
    
    # Create PostGIS container
    log::info "Creating PostGIS container with spatial extensions..."
    docker run -d \
        --name "${POSTGIS_CONTAINER}" \
        --network vrooli-network \
        -p "${POSTGIS_STANDALONE_PORT}:5432" \
        -e POSTGRES_USER=vrooli \
        -e POSTGRES_PASSWORD=vrooli \
        -e POSTGRES_DB=spatial \
        -e POSTGRES_INITDB_ARGS="--encoding=UTF8" \
        -v "${POSTGIS_DATA_DIR}/data:/var/lib/postgresql/data" \
        --health-cmd="pg_isready -U vrooli -d spatial" \
        --health-interval=10s \
        --health-timeout=5s \
        --health-retries=5 \
        --restart unless-stopped \
        "${POSTGIS_IMAGE}" \
        -c shared_buffers=256MB \
        -c max_connections=100 || {
        log::error "Failed to create PostGIS container"
        return 1
    }
    
    # Wait for container to be ready
    log::info "Waiting for PostGIS to be ready..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if docker exec "${POSTGIS_CONTAINER}" pg_isready -U vrooli -d spatial &>/dev/null; then
            log::success "PostGIS is ready"
            break
        fi
        
        attempt=$((attempt + 1))
        sleep 2
    done
    
    if [ $attempt -eq $max_attempts ]; then
        log::error "PostGIS failed to start within timeout"
        return 1
    fi
    
    # Enable PostGIS extensions
    log::info "Enabling PostGIS extensions..."
    docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -c "CREATE EXTENSION IF NOT EXISTS postgis;" 2>/dev/null || true
    docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -c "CREATE EXTENSION IF NOT EXISTS postgis_raster;" 2>/dev/null || true
    docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -c "CREATE EXTENSION IF NOT EXISTS postgis_topology;" 2>/dev/null || true
    
    # Try to enable pgRouting if using custom image
    if [[ "$POSTGIS_USE_CUSTOM_IMAGE" == "true" ]]; then
        log::info "Enabling pgRouting extension..."
        if docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -c "CREATE EXTENSION IF NOT EXISTS pgrouting;" 2>/dev/null; then
            log::success "pgRouting extension enabled for advanced routing capabilities"
        else
            log::warning "pgRouting extension not available in this image"
        fi
    fi
    
    # Create sample spatial data
    log::info "Creating sample spatial tables..."
    create_sample_data_standalone
    
    # Register CLI with Vrooli
    local resource_dir="${APP_ROOT}/resources/postgis"
    if [ -f "${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh" ]; then
        "${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh" "${resource_dir}" 2>/dev/null || true
    fi
    
    # Verify installation
    local version=$(docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -t -c "SELECT PostGIS_Version();" 2>/dev/null | xargs)
    
    if [ -n "$version" ]; then
        # Register PostGIS CLI with vrooli
        if [ -n "${var_SCRIPTS_RESOURCES_LIB_DIR:-}" ] && [ -f "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" ]; then
            "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "${APP_ROOT}/resources/postgis" 2>/dev/null || true
        fi
        
        # Start health server
        if [[ -f "${POSTGIS_INSTALL_LIB_DIR}/health.sh" ]]; then
            source "${POSTGIS_INSTALL_LIB_DIR}/health.sh"
            log::info "Starting health server..."
            postgis::health::start_server
        fi
        
        log::success "PostGIS $version installed and running on port ${POSTGIS_STANDALONE_PORT}"
        log::info "Connection: psql -h localhost -p ${POSTGIS_STANDALONE_PORT} -U vrooli -d spatial"
        log::info "Health endpoint: http://localhost:5435/health"
        return 0
    else
        log::error "Failed to verify PostGIS installation"
        return 1
    fi
}

# Start PostGIS container
postgis_start() {
    if ! docker ps -a --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        log::info "PostGIS not installed. Installing..."
        postgis_install
        return $?
    fi
    
    if docker ps --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        log::info "PostGIS is already running"
        # Ensure health server is also running
        if [[ -f "${POSTGIS_INSTALL_LIB_DIR}/health.sh" ]]; then
            source "${POSTGIS_INSTALL_LIB_DIR}/health.sh"
            if ! postgis::health::is_running; then
                log::info "Starting health server..."
                postgis::health::start_server
            fi
        fi
        return 0
    fi
    
    log::info "Starting PostGIS container..."
    docker start "${POSTGIS_CONTAINER}"
    sleep 3
    
    if docker ps --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        # Start health server
        if [[ -f "${POSTGIS_INSTALL_LIB_DIR}/health.sh" ]]; then
            source "${POSTGIS_INSTALL_LIB_DIR}/health.sh"
            log::info "Starting health server..."
            postgis::health::start_server
        fi
        
        log::success "PostGIS started successfully"
        return 0
    else
        log::error "Failed to start PostGIS"
        return 1
    fi
}

# Stop PostGIS container
postgis_stop() {
    # Stop health server first
    if [[ -f "${POSTGIS_INSTALL_LIB_DIR}/health.sh" ]]; then
        source "${POSTGIS_INSTALL_LIB_DIR}/health.sh"
        if postgis::health::is_running; then
            log::info "Stopping health server..."
            postgis::health::stop_server
        fi
    fi
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        log::info "PostGIS is not running"
        return 0
    fi
    
    log::info "Stopping PostGIS container..."
    docker stop "${POSTGIS_CONTAINER}"
    log::success "PostGIS stopped"
    return 0
}

# Uninstall PostGIS (stop and remove container)
postgis_uninstall() {
    log::header "Uninstalling PostGIS"
    
    # Stop health server first
    if [[ -f "${POSTGIS_INSTALL_LIB_DIR}/health.sh" ]]; then
        source "${POSTGIS_INSTALL_LIB_DIR}/health.sh"
        if postgis::health::is_running; then
            log::info "Stopping health server..."
            postgis::health::stop_server
        fi
    fi
    
    # Stop container if running
    if docker ps --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        log::info "Stopping PostGIS container..."
        docker stop "${POSTGIS_CONTAINER}"
    fi
    
    # Remove container if exists
    if docker ps -a --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        log::info "Removing PostGIS container..."
        docker rm "${POSTGIS_CONTAINER}"
    fi
    
    log::success "PostGIS uninstalled"
    log::info "Note: Data is preserved in ${POSTGIS_DATA_DIR}"
    return 0
}

# Create sample spatial data for testing
create_sample_data_standalone() {
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
    
    # Execute SQL in PostGIS container
    if docker cp "$sql_file" "${POSTGIS_CONTAINER}:/tmp/sample_data.sql" 2>/dev/null && \
       docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -f /tmp/sample_data.sql 2>/dev/null; then
        log::success "Sample spatial data created"
        
        # Show sample query
        log::info "Sample query - Cities within 5000km of NYC:"
        docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial \
            -c "SELECT name, ROUND(distance_km::numeric, 2) as distance_km FROM distances_from_nyc WHERE distance_km < 5000" 2>/dev/null
    else
        log::warning "Failed to create sample data"
    fi
}

# Export functions
export -f postgis_install
export -f postgis_start
export -f postgis_stop
export -f postgis_uninstall