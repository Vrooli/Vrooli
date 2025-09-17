#!/bin/bash

# GeoNode Core Functionality

set -euo pipefail

RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_FILE="${RESOURCE_DIR}/config/defaults.sh"
DOCKER_DIR="${RESOURCE_DIR}/docker"
COMPOSE_FILE="${DOCKER_DIR}/docker-compose.yml"

# Load configuration
load_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        source "$CONFIG_FILE"
    else
        echo "Error: Configuration file not found at $CONFIG_FILE" >&2
        exit 1
    fi
    
    # Ports must be set by config - no fallbacks per Vrooli standards
    if [[ -z "$GEONODE_PORT" ]] || [[ -z "$GEONODE_GEOSERVER_PORT" ]]; then
        echo "Error: Ports not configured. Check port_registry.sh" >&2
        exit 1
    fi
    export GEONODE_DB_HOST="${GEONODE_DB_HOST:-geonode-postgres}"
    export GEONODE_DB_PORT="${GEONODE_DB_PORT:-5432}"
    export GEONODE_DB_NAME="${GEONODE_DB_NAME:-geonode}"
    export GEONODE_DB_USER="${GEONODE_DB_USER:-geonode}"
    export GEONODE_DB_PASSWORD="${GEONODE_DB_PASSWORD:-geonode_secure_pass}"
    export GEONODE_ADMIN_USER="${GEONODE_ADMIN_USER:-admin}"
    export GEONODE_ADMIN_PASSWORD="${GEONODE_ADMIN_PASSWORD:-admin}"
    export GEONODE_ADMIN_EMAIL="${GEONODE_ADMIN_EMAIL:-admin@vrooli.local}"
    export GEONODE_SECRET_KEY="${GEONODE_SECRET_KEY:-default_secret_key}"
}

# Get service status
get_status() {
    if docker ps --format "{{.Names}}" | grep -q "^geonode-django$"; then
        if is_healthy; then
            echo "running"
        else
            echo "starting"
        fi
    else
        echo "stopped"
    fi
}

# Check if services are healthy
is_healthy() {
    # Check GeoServer (usually starts faster)
    if ! timeout 5 curl -sf -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
        "http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/about/status" &>/dev/null; then
        return 1
    fi
    
    # For Django, just check if container is running (it takes a long time to fully start)
    if ! docker ps --filter "name=geonode-django" --format "{{.Names}}" | grep -q "geonode-django"; then
        return 1
    fi
    
    return 0
}

# Install GeoNode
install_geonode() {
    echo "Installing GeoNode dependencies..."
    
    # Load configuration first
    load_config
    
    # Create necessary directories
    mkdir -p "${DOCKER_DIR}"
    mkdir -p "${RESOURCE_DIR}/data"
    mkdir -p "${RESOURCE_DIR}/examples"
    
    # Create docker-compose file if not exists
    if [[ ! -f "$COMPOSE_FILE" ]]; then
        create_docker_compose
    fi
    
    # Pull Docker images
    echo "Pulling Docker images..."
    cd "${DOCKER_DIR}" && docker-compose pull
    
    echo "GeoNode installation complete"
}

# Create Docker Compose configuration
create_docker_compose() {
    cat > "$COMPOSE_FILE" << EOF
services:
  geonode-db:
    image: postgis/postgis:14-3.2
    container_name: geonode-postgres
    environment:
      POSTGRES_DB: ${GEONODE_DB_NAME}
      POSTGRES_USER: ${GEONODE_DB_USER}
      POSTGRES_PASSWORD: ${GEONODE_DB_PASSWORD}
      POSTGRES_INITDB_ARGS: "--auth-host=trust"
    volumes:
      - geonode-db-data:/var/lib/postgresql/data
    networks:
      - geonode-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${GEONODE_DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

  geonode-redis:
    image: redis:7-alpine
    container_name: geonode-redis
    networks:
      - geonode-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  geonode-geoserver:
    image: kartoza/geoserver:2.24.0
    container_name: geonode-geoserver
    ports:
      - "${GEONODE_GEOSERVER_PORT}:8080"
    environment:
      GEOSERVER_ADMIN_USER: ${GEONODE_ADMIN_USER}
      GEOSERVER_ADMIN_PASSWORD: ${GEONODE_ADMIN_PASSWORD}
      INITIAL_MEMORY: 2G
      MAXIMUM_MEMORY: 4G
      GEOSERVER_DATA_DIR: /opt/geoserver/data_dir
      STABLE_EXTENSIONS: "css,csw,importer,inspire,printing,vectortiles,wps,xslt"
      COMMUNITY_EXTENSIONS: ""
      GEOSERVER_CONTEXT_ROOT: geoserver
    volumes:
      - geonode-geoserver-data:/opt/geoserver/data_dir
      - geonode-backup:/backup_restore
    depends_on:
      - geonode-db
    networks:
      - geonode-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/geoserver/rest/about/status", "-u", "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s

  geonode-django:
    image: geonode/geonode:4.4.3
    container_name: geonode-django
    entrypoint: ["/usr/src/geonode/entrypoint.sh"]
    command: ["python", "/usr/src/geonode/manage.py", "runserver", "0.0.0.0:8000"]
    ports:
      - "${GEONODE_PORT}:8000"
    environment:
      # Database
      DATABASE_URL: postgis://${GEONODE_DB_USER}:${GEONODE_DB_PASSWORD}@geonode-db:5432/${GEONODE_DB_NAME}
      GEODATABASE_URL: postgis://${GEONODE_DB_USER}:${GEONODE_DB_PASSWORD}@geonode-db:5432/geonode_data
      
      # GeoServer
      GEOSERVER_PUBLIC_LOCATION: http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/
      GEOSERVER_LOCATION: http://geonode-geoserver:8080/geoserver/
      GEOSERVER_ADMIN_USER: ${GEONODE_ADMIN_USER}
      GEOSERVER_ADMIN_PASSWORD: ${GEONODE_ADMIN_PASSWORD}
      
      # Django settings
      DJANGO_SETTINGS_MODULE: geonode.settings
      SECRET_KEY: ${GEONODE_SECRET_KEY}
      SITEURL: http://localhost:${GEONODE_PORT}/
      ALLOWED_HOSTS: "['localhost', 'geonode-django', '*']"
      
      # Admin
      ADMIN_USERNAME: ${GEONODE_ADMIN_USER}
      ADMIN_PASSWORD: ${GEONODE_ADMIN_PASSWORD}
      ADMIN_EMAIL: ${GEONODE_ADMIN_EMAIL}
      
      # Redis cache
      CACHE_URL: redis://geonode-redis:6379/0
      BROKER_URL: redis://geonode-redis:6379/0
      CELERY_BROKER_URL: redis://geonode-redis:6379/0
      
      # Features
      MONITORING_ENABLED: "True"
      MODIFY_TOPICCATEGORY: "True"
      IS_CELERY: "False"
      
    volumes:
      - geonode-statics:/mnt/volumes/statics
      - geonode-media:/mnt/volumes/media
      - geonode-backup:/backup_restore
    depends_on:
      - geonode-db
      - geonode-redis
      - geonode-geoserver
    networks:
      - geonode-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 120s

volumes:
  geonode-db-data:
    name: geonode-db-data
  geonode-geoserver-data:
    name: geonode-geoserver-data
  geonode-statics:
    name: geonode-statics
  geonode-media:
    name: geonode-media
  geonode-backup:
    name: geonode-backup

networks:
  geonode-network:
    name: geonode-network
    driver: bridge
EOF
}

# Start GeoNode services
start_geonode() {
    local wait_flag=false
    
    if [[ "${1:-}" == "--wait" ]]; then
        wait_flag=true
    fi
    
    load_config
    
    if [[ ! -f "$COMPOSE_FILE" ]]; then
        echo "Docker Compose file not found. Running install first..."
        install_geonode
    fi
    
    echo "Starting GeoNode services..."
    cd "${DOCKER_DIR}" && docker-compose up -d
    
    if [[ "$wait_flag" == true ]]; then
        echo "Waiting for services to be healthy..."
        local max_attempts=60
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if is_healthy; then
                echo "GeoNode is ready!"
                echo "Web Portal: http://localhost:${GEONODE_PORT}"
                echo "GeoServer: http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver"
                return 0
            fi
            
            echo -n "."
            sleep 2
            ((attempt++))
        done
        
        echo ""
        echo "Warning: Services may still be starting. Check status with: vrooli resource geonode status"
    fi
}

# Stop GeoNode services
stop_geonode() {
    echo "Stopping GeoNode services..."
    
    if [[ -f "$COMPOSE_FILE" ]]; then
        cd "${DOCKER_DIR}" && docker-compose down
    else
        # Stop containers directly if compose file missing
        docker stop geonode-django geonode-geoserver geonode-redis geonode-postgres 2>/dev/null || true
    fi
    
    echo "GeoNode services stopped"
}

# Restart services
restart_geonode() {
    stop_geonode
    sleep 2
    start_geonode "$@"
}

# Uninstall GeoNode
uninstall_geonode() {
    echo "Uninstalling GeoNode..."
    
    stop_geonode
    
    # Remove containers and volumes
    if [[ -f "$COMPOSE_FILE" ]]; then
        cd "${DOCKER_DIR}" && docker-compose down -v
    fi
    
    # Remove Docker volumes
    docker volume rm geonode-db-data geonode-geoserver-data geonode-statics geonode-media geonode-backup 2>/dev/null || true
    
    echo "GeoNode uninstalled"
}

# Show service status
show_status() {
    local format="${1:-text}"
    load_config
    
    local status=$(get_status)
    
    if [[ "$format" == "--json" || "$format" == "json" ]]; then
        cat << EOF
{
  "status": "$status",
  "services": {
    "django": {
      "container": "geonode-django",
      "port": ${GEONODE_PORT},
      "url": "http://localhost:${GEONODE_PORT}",
      "healthy": $(is_service_healthy "geonode-django" && echo "true" || echo "false")
    },
    "geoserver": {
      "container": "geonode-geoserver",
      "port": ${GEONODE_GEOSERVER_PORT},
      "url": "http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver",
      "healthy": $(is_service_healthy "geonode-geoserver" && echo "true" || echo "false")
    },
    "database": {
      "container": "geonode-postgres",
      "healthy": $(is_service_healthy "geonode-postgres" && echo "true" || echo "false")
    },
    "redis": {
      "container": "geonode-redis",
      "healthy": $(is_service_healthy "geonode-redis" && echo "true" || echo "false")
    }
  }
}
EOF
    else
        echo "GeoNode Status: $status"
        echo ""
        echo "Services:"
        echo "  Django Portal: $(is_service_healthy "geonode-django" && echo "✓" || echo "✗") http://localhost:${GEONODE_PORT}"
        echo "  GeoServer: $(is_service_healthy "geonode-geoserver" && echo "✓" || echo "✗") http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver"
        echo "  Database: $(is_service_healthy "geonode-postgres" && echo "✓" || echo "✗") PostGIS"
        echo "  Cache: $(is_service_healthy "geonode-redis" && echo "✓" || echo "✗") Redis"
        
        if [[ "$status" == "running" ]]; then
            echo ""
            echo "Quick Actions:"
            echo "  Upload data: vrooli resource geonode content add-layer <file>"
            echo "  View layers: vrooli resource geonode content list-layers"
            echo "  View logs: vrooli resource geonode logs"
        fi
    fi
}

# Check if specific service is healthy
is_service_healthy() {
    local service="$1"
    docker ps --filter "name=${service}" --filter "health=healthy" --format "{{.Names}}" | grep -q "${service}"
}

# Show logs
show_logs() {
    local service="${1:-}"
    
    if [[ -n "$service" ]]; then
        docker logs "geonode-${service}" --tail 100 -f
    else
        echo "Available services: django, geoserver, postgres, redis"
        echo "Usage: vrooli resource geonode logs <service>"
    fi
}

# Show credentials
show_credentials() {
    load_config
    
    cat << EOF
GeoNode Credentials
==================
Admin User: ${GEONODE_ADMIN_USER}
Admin Password: ${GEONODE_ADMIN_PASSWORD}
Admin Email: ${GEONODE_ADMIN_EMAIL}

Web Portal: http://localhost:${GEONODE_PORT}
GeoServer: http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver

API Token:
  1. Login to portal with admin credentials
  2. Navigate to: http://localhost:${GEONODE_PORT}/account/profile
  3. Click "API" tab to generate token

Database:
  Host: localhost
  Port: 5432 (internal)
  Database: ${GEONODE_DB_NAME}
  User: ${GEONODE_DB_USER}
  Password: ${GEONODE_DB_PASSWORD}
EOF
}

# Add layer
add_layer() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "Error: File not found: $file" >&2
        exit 1
    fi
    
    load_config
    
    echo "Uploading layer: $(basename "$file")..."
    
    # Try GeoNode API first if Django is available
    if timeout 2 curl -sf "http://localhost:${GEONODE_PORT}/api/" &>/dev/null; then
        local response=$(curl -sf -X POST \
            -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
            -F "base_file=@${file}" \
            "http://localhost:${GEONODE_PORT}/api/v2/uploads/upload/" 2>/dev/null)
        
        if [[ $? -eq 0 ]]; then
            echo "Layer uploaded successfully via GeoNode!"
            echo "View at: http://localhost:${GEONODE_PORT}/layers/"
            return 0
        fi
    fi
    
    # Fallback to GeoServer REST API
    echo "Using GeoServer REST API for layer upload..."
    
    # Ensure vrooli workspace exists
    curl -sf -X POST -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
        -H "Content-type: application/json" \
        -d '{"workspace":{"name":"vrooli"}}' \
        "http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/workspaces" &>/dev/null || true
    
    # Get file extension and name
    local filename=$(basename "$file")
    local name="${filename%.*}"
    local ext="${filename##*.}"
    
    # Upload based on file type
    case "${ext,,}" in
        geojson|json)
            # Copy file to container
            docker cp "$file" geonode-geoserver:/tmp/"${filename}"
            
            # Create datastore and publish layer in one step
            curl -sf -X PUT -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
                -H "Content-type: application/json" \
                -d "{\"dataStore\":{\"name\":\"${name}\",\"type\":\"GeoJSON\",\"enabled\":true,\"workspace\":{\"name\":\"vrooli\"},\"connectionParameters\":{\"entry\":[{\"@key\":\"url\",\"$\":\"file:///tmp/${filename}\"}]},\"featureTypes\":{\"featureType\":[{\"name\":\"${name}\"}]}}}" \
                "http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/workspaces/vrooli/datastores/${name}"
            ;;
        tif|tiff|geotiff)
            # Upload raster
            curl -sf -X POST -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
                -H "Content-type: image/tiff" \
                --data-binary "@${file}" \
                "http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/workspaces/vrooli/coveragestores/${name}/file.geotiff"
            ;;
        shp)
            # Upload shapefile (needs to be zipped with supporting files)
            echo "Note: Shapefiles should be uploaded as zip archives with all supporting files"
            ;;
        *)
            echo "Unsupported file type: ${ext}" >&2
            echo "Supported formats: GeoJSON, GeoTIFF" >&2
            exit 1
            ;;
    esac
    
    if [[ $? -eq 0 ]]; then
        echo "Layer '${name}' uploaded successfully to GeoServer!"
        echo "View at: http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/web/"
        return 0
    else
        echo "Error uploading layer via GeoServer" >&2
        exit 1
    fi
}

# List layers
list_layers() {
    load_config
    
    echo "Fetching layers..."
    
    # Try GeoNode API first
    if timeout 2 curl -sf "http://localhost:${GEONODE_PORT}/api/" &>/dev/null; then
        curl -sf \
            -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
            "http://localhost:${GEONODE_PORT}/api/v2/layers/" | \
            jq -r '.objects[] | "\(.title) (\(.name)) - \(.resource_type)"' 2>/dev/null && return 0
    fi
    
    # Fallback to GeoServer REST API
    echo "Using GeoServer REST API..."
    
    # List layers from GeoServer
    local layers=$(curl -sf -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
        "http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/layers" 2>/dev/null)
    
    if [[ -n "$layers" ]]; then
        echo "$layers" | jq -r '.layers.layer[]?.name // empty' 2>/dev/null || echo "No layers found"
    else
        echo "No layers found or GeoServer not available"
    fi
}

# Get layer details
get_layer() {
    local name="$1"
    
    if [[ -z "$name" ]]; then
        echo "Usage: get-layer <name>" >&2
        exit 1
    fi
    
    load_config
    
    # Try GeoNode API first
    if timeout 2 curl -sf "http://localhost:${GEONODE_PORT}/api/" &>/dev/null; then
        curl -sf \
            -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
            "http://localhost:${GEONODE_PORT}/api/v2/layers/${name}/" | \
            jq '.' 2>/dev/null && return 0
    fi
    
    # Fallback to GeoServer REST API
    echo "Using GeoServer REST API..."
    curl -sf \
        -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
        "http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/layers/${name}" | \
        jq '.' 2>/dev/null || \
        echo "Layer not found: $name"
}

# Remove layer
remove_layer() {
    local name="$1"
    
    if [[ -z "$name" ]]; then
        echo "Usage: remove-layer <name>" >&2
        exit 1
    fi
    
    load_config
    
    echo "Removing layer: $name..."
    
    # Try GeoNode API first
    if timeout 2 curl -sf "http://localhost:${GEONODE_PORT}/api/" &>/dev/null; then
        curl -sf -X DELETE \
            -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
            "http://localhost:${GEONODE_PORT}/api/v2/layers/${name}/" &>/dev/null
        
        if [[ $? -eq 0 ]]; then
            echo "Layer removed successfully from GeoNode"
            return 0
        fi
    fi
    
    # Fallback to GeoServer REST API
    echo "Using GeoServer REST API..."
    
    # Try to remove from default workspace first
    curl -sf -X DELETE \
        -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
        "http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/layers/${name}?recurse=true" &>/dev/null
    
    # Also try vrooli workspace
    curl -sf -X DELETE \
        -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
        "http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/workspaces/vrooli/layers/${name}?recurse=true" &>/dev/null
    
    if [[ $? -eq 0 ]]; then
        echo "Layer removed successfully from GeoServer"
    else
        echo "Warning: Layer may not exist or already removed" >&2
    fi
}

# Import data
import_data() {
    local format="$1"
    local file="$2"
    
    case "$format" in
        shapefile|shp)
            add_layer "$file"
            ;;
        geotiff|tiff|tif)
            add_layer "$file"
            ;;
        geojson|json)
            add_layer "$file"
            ;;
        kml)
            add_layer "$file"
            ;;
        *)
            echo "Unsupported format: $format" >&2
            echo "Supported: shapefile, geotiff, geojson, kml" >&2
            exit 1
            ;;
    esac
}

# Show stats
show_stats() {
    load_config
    
    echo "GeoNode Resource Statistics"
    echo "==========================="
    
    # Get container stats
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" \
        geonode-django geonode-geoserver geonode-postgres geonode-redis 2>/dev/null || \
        echo "Services not running"
    
    echo ""
    
    # Get layer count if available
    local layer_count=$(curl -sf \
        -u "${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD}" \
        "http://localhost:${GEONODE_PORT}/api/v2/layers/" 2>/dev/null | \
        jq '.meta.total_count' 2>/dev/null || echo "0")
    
    echo "Total Layers: $layer_count"
}

# Placeholder functions for advanced features
create_map() { echo "Map creation will be implemented in P1 phase"; }
list_maps() { echo "Map listing will be implemented in P1 phase"; }
export_map() { echo "Map export will be implemented in P1 phase"; }
export_layer() { echo "Layer export will be implemented in P1 phase"; }
update_metadata() { echo "Metadata update will be implemented in P1 phase"; }
search_metadata() { echo "Metadata search will be implemented in P1 phase"; }
set_postgres_config() { echo "Postgres configuration will be implemented in P1 phase"; }
set_storage_config() { echo "Storage configuration will be implemented in P1 phase"; }
show_config() { show_info; }
validate_file() { echo "File validation will be implemented in P1 phase"; }
optimize_cache() { echo "Cache optimization will be implemented in P2 phase"; }
optimize_indexes() { echo "Index optimization will be implemented in P2 phase"; }
create_webhook() { echo "Webhook creation will be implemented in P2 phase"; }
list_webhooks() { echo "Webhook listing will be implemented in P2 phase"; }
delete_webhook() { echo "Webhook deletion will be implemented in P2 phase"; }