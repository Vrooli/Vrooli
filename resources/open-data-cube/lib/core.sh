#!/bin/bash

# Open Data Cube Core Library
# Provides core functionality for ODC resource management

set -euo pipefail

RESOURCE_NAME="open-data-cube"
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${RESOURCE_DIR}/config"
DOCKER_DIR="${RESOURCE_DIR}/docker"
DATA_DIR="${RESOURCE_DIR}/data"

# Load configuration
source "${CONFIG_DIR}/defaults.sh"

# Port allocation - dynamically allocated by the lifecycle system
# For testing/development, use defaults from port_registry.sh
if [ -z "${ODC_PORT}" ]; then
    ODC_PORT="8850"
fi
if [ -z "${ODC_DB_PORT}" ]; then
    ODC_DB_PORT="5450"  
fi
if [ -z "${DATACUBE_OWS_PORT}" ]; then
    DATACUBE_OWS_PORT="8851"
fi

# Container names
ODC_API_CONTAINER="${RESOURCE_NAME}-api"
ODC_DB_CONTAINER="${RESOURCE_NAME}-db"
ODC_OWS_CONTAINER="${RESOURCE_NAME}-ows"
ODC_REDIS_CONTAINER="${RESOURCE_NAME}-redis"

# Show runtime information
show_info() {
    echo "Open Data Cube Runtime Configuration:"
    echo "  Resource Directory: ${RESOURCE_DIR}"
    echo "  API Port: ${ODC_PORT}"
    echo "  Database Port: ${ODC_DB_PORT}"
    echo "  OWS Port: ${DATACUBE_OWS_PORT}"
    echo "  Data Directory: ${DATA_DIR}"
    echo ""
    echo "Container Names:"
    echo "  API: ${ODC_API_CONTAINER}"
    echo "  Database: ${ODC_DB_CONTAINER}"
    echo "  OWS: ${ODC_OWS_CONTAINER}"
    echo "  Redis: ${ODC_REDIS_CONTAINER}"
}

# Handle lifecycle management
handle_manage() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        install)
            install_odc "$@"
            ;;
        start)
            start_odc "$@"
            ;;
        stop)
            stop_odc "$@"
            ;;
        restart)
            stop_odc
            start_odc "$@"
            ;;
        uninstall)
            uninstall_odc "$@"
            ;;
        *)
            echo "Usage: manage [install|start|stop|restart|uninstall]"
            exit 1
            ;;
    esac
}

# Install ODC dependencies
install_odc() {
    echo "Installing Open Data Cube dependencies..."
    
    # Ensure Docker is installed
    if ! command -v docker &> /dev/null; then
        echo "Error: Docker is not installed"
        exit 1
    fi
    
    # Create necessary directories
    mkdir -p "${DATA_DIR}"/{products,indexed,cache}
    mkdir -p "${DOCKER_DIR}"
    
    # Create docker-compose file
    create_docker_compose
    
    # Pull Docker images
    echo "Pulling ODC Docker images..."
    docker pull opendatacube/datacube-core:latest
    docker pull postgis/postgis:13-3.1
    docker pull opendatacube/ows:latest
    docker pull redis:alpine
    
    echo "ODC installation complete"
}

# Create Docker Compose configuration
create_docker_compose() {
    cat > "${DOCKER_DIR}/docker-compose.yml" << EOF
version: '3.8'

services:
  odc-db:
    image: postgis/postgis:13-3.1
    container_name: ${ODC_DB_CONTAINER}
    environment:
      POSTGRES_DB: datacube
      POSTGRES_USER: datacube
      POSTGRES_PASSWORD: datacube_password
    ports:
      - "${ODC_DB_PORT}:5432"
    volumes:
      - odc-db-data:/var/lib/postgresql/data
    networks:
      - odc-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U datacube"]
      interval: 10s
      timeout: 5s
      retries: 5

  odc-api:
    image: opendatacube/datacube-core:latest
    container_name: ${ODC_API_CONTAINER}
    environment:
      DB_HOSTNAME: odc-db
      DB_PORT: 5432
      DB_DATABASE: datacube
      DB_USERNAME: datacube
      DB_PASSWORD: datacube_password
      DATACUBE_CONFIG_PATH: /etc/datacube.conf
    ports:
      - "${ODC_PORT}:8080"
    volumes:
      - ${DATA_DIR}:/data
      - ./datacube.conf:/etc/datacube.conf:ro
    networks:
      - odc-network
    depends_on:
      odc-db:
        condition: service_healthy
    command: ["datacube", "system", "init"]

  odc-ows:
    image: opendatacube/ows:latest
    container_name: ${ODC_OWS_CONTAINER}
    environment:
      DB_HOSTNAME: odc-db
      DB_PORT: 5432
      DB_DATABASE: datacube
      DB_USERNAME: datacube
      DB_PASSWORD: datacube_password
      DATACUBE_OWS_CFG: config.datacube_ows_cfg.ows_cfg
    ports:
      - "${DATACUBE_OWS_PORT}:8000"
    volumes:
      - ${DATA_DIR}:/data
    networks:
      - odc-network
    depends_on:
      - odc-db
      - redis

  redis:
    image: redis:alpine
    container_name: ${ODC_REDIS_CONTAINER}
    networks:
      - odc-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

networks:
  odc-network:
    driver: bridge

volumes:
  odc-db-data:
EOF

    # Create datacube configuration
    cat > "${DOCKER_DIR}/datacube.conf" << EOF
[datacube]
db_database: datacube
db_hostname: odc-db
db_username: datacube
db_password: datacube_password
db_port: 5432

[locations]
data: /data
EOF
}

# Start ODC stack
start_odc() {
    local wait_flag=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait)
                wait_flag=true
                ;;
        esac
        shift
    done
    
    echo "Starting Open Data Cube stack..."
    
    # Check port allocation
    if [[ -z "${ODC_PORT}" || -z "${ODC_DB_PORT}" || -z "${DATACUBE_OWS_PORT}" ]]; then
        echo "Error: Ports not allocated. Run 'manage install' first"
        exit 1
    fi
    
    cd "${DOCKER_DIR}"
    docker-compose up -d
    
    if [[ "$wait_flag" == true ]]; then
        echo "Waiting for ODC services to be ready..."
        wait_for_health
    fi
    
    echo "ODC stack started successfully"
    echo "  API: http://localhost:${ODC_PORT}"
    echo "  OWS: http://localhost:${DATACUBE_OWS_PORT}"
}

# Stop ODC stack
stop_odc() {
    echo "Stopping Open Data Cube stack..."
    
    cd "${DOCKER_DIR}" 2>/dev/null || true
    docker-compose down 2>/dev/null || true
    
    # Stop individual containers if compose fails
    docker stop ${ODC_API_CONTAINER} 2>/dev/null || true
    docker stop ${ODC_OWS_CONTAINER} 2>/dev/null || true
    docker stop ${ODC_DB_CONTAINER} 2>/dev/null || true
    docker stop ${ODC_REDIS_CONTAINER} 2>/dev/null || true
    
    echo "ODC stack stopped"
}

# Uninstall ODC
uninstall_odc() {
    echo "Uninstalling Open Data Cube..."
    
    # Stop services
    stop_odc
    
    # Remove containers
    docker rm ${ODC_API_CONTAINER} 2>/dev/null || true
    docker rm ${ODC_OWS_CONTAINER} 2>/dev/null || true
    docker rm ${ODC_DB_CONTAINER} 2>/dev/null || true
    docker rm ${ODC_REDIS_CONTAINER} 2>/dev/null || true
    
    # Remove volumes (optional)
    read -p "Remove data volumes? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker volume rm open-data-cube_odc-db-data 2>/dev/null || true
    fi
    
    echo "ODC uninstalled"
}

# Wait for services to be healthy
wait_for_health() {
    local max_attempts=30
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if check_health; then
            echo "All ODC services are healthy"
            return 0
        fi
        
        echo "Waiting for services... ($((attempt+1))/$max_attempts)"
        sleep 5
        ((attempt++))
    done
    
    echo "Warning: Services did not become healthy in time"
    return 1
}

# Check health of ODC services
check_health() {
    # Check database
    if ! docker exec ${ODC_DB_CONTAINER} pg_isready -U datacube &>/dev/null; then
        return 1
    fi
    
    # Check Redis
    if ! docker exec ${ODC_REDIS_CONTAINER} redis-cli ping &>/dev/null; then
        return 1
    fi
    
    # Check API endpoint
    if ! timeout 5 curl -sf "http://localhost:${ODC_PORT}/health" &>/dev/null; then
        return 1
    fi
    
    # Check OWS endpoint
    if ! timeout 5 curl -sf "http://localhost:${DATACUBE_OWS_PORT}/ping" &>/dev/null; then
        return 1
    fi
    
    return 0
}

# Show ODC status
show_status() {
    echo "Open Data Cube Status:"
    echo ""
    
    # Check containers
    for container in ${ODC_API_CONTAINER} ${ODC_DB_CONTAINER} ${ODC_OWS_CONTAINER} ${ODC_REDIS_CONTAINER}; do
        if docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
            echo "  ${container}: Running"
        else
            echo "  ${container}: Stopped"
        fi
    done
    
    echo ""
    
    # Check health
    if check_health; then
        echo "Health: All services healthy"
    else
        echo "Health: Some services unhealthy"
    fi
    
    echo ""
    echo "Endpoints:"
    echo "  API: http://localhost:${ODC_PORT:-not allocated}"
    echo "  OWS: http://localhost:${DATACUBE_OWS_PORT:-not allocated}"
    echo "  Database: localhost:${ODC_DB_PORT:-not allocated}"
}

# Show logs
show_logs() {
    local service="${1:-all}"
    
    case "$service" in
        api)
            docker logs -f ${ODC_API_CONTAINER}
            ;;
        db|database)
            docker logs -f ${ODC_DB_CONTAINER}
            ;;
        ows)
            docker logs -f ${ODC_OWS_CONTAINER}
            ;;
        redis)
            docker logs -f ${ODC_REDIS_CONTAINER}
            ;;
        all|*)
            docker-compose -f "${DOCKER_DIR}/docker-compose.yml" logs -f
            ;;
    esac
}

# Show credentials
show_credentials() {
    echo "Open Data Cube Credentials:"
    echo ""
    echo "Database:"
    echo "  Host: localhost"
    echo "  Port: ${ODC_DB_PORT:-not allocated}"
    echo "  Database: datacube"
    echo "  Username: datacube"
    echo "  Password: datacube_password"
    echo ""
    echo "API Endpoint: http://localhost:${ODC_PORT:-not allocated}"
    echo "OWS Endpoint: http://localhost:${DATACUBE_OWS_PORT:-not allocated}"
    echo ""
    echo "Connection String:"
    echo "  postgresql://datacube:datacube_password@localhost:${ODC_DB_PORT}/datacube"
}

# Handle content management
handle_content() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            list_datasets "$@"
            ;;
        add)
            add_dataset "$@"
            ;;
        remove)
            remove_dataset "$@"
            ;;
        get)
            get_dataset "$@"
            ;;
        *)
            echo "Usage: content [list|add|remove|get]"
            exit 1
            ;;
    esac
}

# List datasets
list_datasets() {
    echo "Available datasets:"
    docker exec ${ODC_API_CONTAINER} datacube product list 2>/dev/null || echo "No datasets available"
}

# Add dataset
add_dataset() {
    local path="${1:-}"
    
    if [[ -z "$path" ]]; then
        echo "Usage: content add <path>"
        exit 1
    fi
    
    echo "Adding dataset from ${path}..."
    docker exec ${ODC_API_CONTAINER} datacube dataset add "$path"
}

# Remove dataset
remove_dataset() {
    local id="${1:-}"
    
    if [[ -z "$id" ]]; then
        echo "Usage: content remove <dataset-id>"
        exit 1
    fi
    
    echo "Removing dataset ${id}..."
    docker exec ${ODC_API_CONTAINER} datacube dataset archive "$id"
}

# Get dataset details
get_dataset() {
    local id="${1:-}"
    
    if [[ -z "$id" ]]; then
        echo "Usage: content get <dataset-id>"
        exit 1
    fi
    
    docker exec ${ODC_API_CONTAINER} datacube dataset info "$id"
}

# Handle dataset operations
handle_dataset() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        index)
            index_dataset "$@"
            ;;
        list)
            list_indexed_datasets "$@"
            ;;
        search)
            search_datasets "$@"
            ;;
        *)
            echo "Usage: dataset [index|list|search]"
            exit 1
            ;;
    esac
}

# Index dataset
index_dataset() {
    local path="${1:-}"
    
    if [[ -z "$path" ]]; then
        echo "Usage: dataset index <path>"
        exit 1
    fi
    
    echo "Indexing dataset from ${path}..."
    docker exec ${ODC_API_CONTAINER} datacube dataset add --auto-match "$path"
}

# List indexed datasets
list_indexed_datasets() {
    echo "Indexed datasets:"
    docker exec ${ODC_API_CONTAINER} datacube dataset search
}

# Search datasets
search_datasets() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        echo "Usage: dataset search <query>"
        exit 1
    fi
    
    docker exec ${ODC_API_CONTAINER} datacube dataset search "$query"
}

# Handle query operations
handle_query() {
    local type="${1:-}"
    shift || true
    
    case "$type" in
        area)
            query_by_area "$@"
            ;;
        time)
            query_by_time "$@"
            ;;
        product)
            query_by_product "$@"
            ;;
        *)
            echo "Usage: query [area|time|product]"
            exit 1
            ;;
    esac
}

# Query by area
query_by_area() {
    local geojson="${1:-}"
    
    if [[ -z "$geojson" ]]; then
        echo "Usage: query area <geojson>"
        exit 1
    fi
    
    echo "Querying by area..."
    docker exec ${ODC_API_CONTAINER} python3 -c "
import datacube
dc = datacube.Datacube()
import json
geom = json.loads('${geojson}')
data = dc.load(product='ls8_nbar_albers', geopolygon=geom)
print(data)
"
}

# Query by time range
query_by_time() {
    local range="${1:-}"
    
    if [[ -z "$range" ]]; then
        echo "Usage: query time <start-date>/<end-date>"
        exit 1
    fi
    
    echo "Querying by time range: ${range}..."
    local start_date="${range%/*}"
    local end_date="${range#*/}"
    
    docker exec ${ODC_API_CONTAINER} python3 -c "
import datacube
dc = datacube.Datacube()
data = dc.load(product='ls8_nbar_albers', time=('${start_date}', '${end_date}'))
print(data)
"
}

# Query by product
query_by_product() {
    local product="${1:-}"
    
    if [[ -z "$product" ]]; then
        echo "Usage: query product <product-name>"
        exit 1
    fi
    
    echo "Querying product: ${product}..."
    docker exec ${ODC_API_CONTAINER} python3 -c "
import datacube
dc = datacube.Datacube()
data = dc.load(product='${product}')
print(data)
"
}

# Handle export operations
handle_export() {
    local format="${1:-}"
    shift || true
    
    case "$format" in
        geotiff)
            export_geotiff "$@"
            ;;
        geojson)
            export_geojson "$@"
            ;;
        netcdf)
            export_netcdf "$@"
            ;;
        *)
            echo "Usage: export [geotiff|geojson|netcdf] <output-path>"
            exit 1
            ;;
    esac
}

# Export to GeoTIFF
export_geotiff() {
    local output="${1:-output.tif}"
    
    echo "Exporting to GeoTIFF: ${output}..."
    docker exec ${ODC_API_CONTAINER} python3 -c "
import datacube
import rasterio
dc = datacube.Datacube()
# Example export - would use actual query results
data = dc.load(product='ls8_nbar_albers', limit=1)
data.to_netcdf('${output}'.replace('.tif', '.nc'))
print('Export complete: ${output}')
"
}

# Export to GeoJSON
export_geojson() {
    local output="${1:-output.json}"
    
    echo "Exporting to GeoJSON: ${output}..."
    docker exec ${ODC_API_CONTAINER} python3 -c "
import datacube
import json
dc = datacube.Datacube()
# Example export - would use actual query results
datasets = dc.find_datasets(product='ls8_nbar_albers', limit=1)
features = [{'type': 'Feature', 'geometry': ds.extent.geom, 'properties': {'id': str(ds.id)}} for ds in datasets]
geojson = {'type': 'FeatureCollection', 'features': features}
with open('${output}', 'w') as f:
    json.dump(geojson, f)
print('Export complete: ${output}')
"
}

# Export to NetCDF
export_netcdf() {
    local output="${1:-output.nc}"
    
    echo "Exporting to NetCDF: ${output}..."
    docker exec ${ODC_API_CONTAINER} python3 -c "
import datacube
dc = datacube.Datacube()
# Example export - would use actual query results
data = dc.load(product='ls8_nbar_albers', limit=1)
data.to_netcdf('${output}')
print('Export complete: ${output}')
"
}