#!/usr/bin/env bash
# farmOS Core Library Functions

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Source API library
source "${RESOURCE_DIR}/lib/api.sh"

# Source installation helper
source "${RESOURCE_DIR}/lib/install.sh"

# Help function
farmos::help() {
    cat << EOF
farmOS Resource - Agricultural Management Platform

USAGE:
    resource-farmos <command> [options]

EXAMPLES:
    # Start farmOS with demo data
    resource-farmos manage start --wait
    
    # Create a new field
    resource-farmos farm create-field --name "North Field" --size 10 --unit acres
    
    # Log an activity
    resource-farmos farm log-activity --type planting --field "North Field" --crop "Corn"
    
    # Export records
    resource-farmos farm export --format csv --output farm_records.csv
    
    # Connect IoT sensors
    resource-farmos iot connect --broker mqtt://localhost:1883
    
    # Check status
    resource-farmos status

CONFIGURATION:
    Port: ${FARMOS_PORT}
    Admin User: ${FARMOS_ADMIN_USER}
    API Endpoint: ${FARMOS_API_BASE}
    Demo Data: ${FARMOS_DEMO_DATA}

For more information, see: ${RESOURCE_DIR}/README.md
EOF
}

# Info function
farmos::info() {
    local format="${1:-text}"
    
    if [[ "$format" == "json" ]] || [[ "$format" == "--json" ]]; then
        cat "${RESOURCE_DIR}/config/runtime.json"
    else
        echo "farmOS Resource Information:"
        echo "  Startup Order: 450"
        echo "  Dependencies: none"
        echo "  Startup Time: 60-90s"
        echo "  Priority: medium"
        echo "  Port: ${FARMOS_PORT}"
        echo "  API Version: ${FARMOS_API_VERSION}"
    fi
}

# Management functions
farmos::manage::install() {
    echo "Installing farmOS..."
    
    # Create necessary directories
    mkdir -p "${HOME}/.farmos/data"
    mkdir -p "${HOME}/.farmos/modules"
    mkdir -p "${HOME}/.farmos/themes"
    
    # Pull Docker images
    echo "Pulling farmOS Docker images..."
    docker pull "${FARMOS_DOCKER_IMAGE}"
    docker pull postgres:14-alpine
    
    echo "farmOS installation complete"
    return 0
}

farmos::manage::uninstall() {
    local force="${1:-}"
    
    if [[ "$force" != "--force" ]]; then
        echo "This will remove farmOS and all data. Use --force to confirm."
        return 1
    fi
    
    echo "Uninstalling farmOS..."
    
    # Stop services if running
    farmos::manage::stop || true
    
    # Remove containers and volumes
    docker-compose -f "${FARMOS_COMPOSE_FILE}" down -v 2>/dev/null || true
    
    # Remove data directories (unless --keep-data is specified)
    if [[ "$2" != "--keep-data" ]]; then
        rm -rf "${HOME}/.farmos"
    fi
    
    echo "farmOS uninstalled"
    return 0
}

farmos::manage::start() {
    local wait_flag="${1:-}"
    
    echo "Starting farmOS services..."
    
    # Start services using Docker Compose
    docker-compose -f "${FARMOS_COMPOSE_FILE}" up -d
    
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "Waiting for farmOS to be ready..."
        local max_attempts=24  # 2 minutes with 5-second intervals
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if farmos::health_check; then
                echo "farmOS is ready!"
                
                # Auto-complete installation if needed
                farmos::install::auto_setup
                
                return 0
            fi
            echo -n "."
            sleep 5
            ((attempt++))
        done
        
        echo ""
        echo "Error: farmOS failed to start within timeout period"
        return 1
    fi
    
    return 0
}

farmos::manage::stop() {
    echo "Stopping farmOS services..."
    
    docker-compose -f "${FARMOS_COMPOSE_FILE}" stop
    
    echo "farmOS services stopped"
    return 0
}

farmos::manage::restart() {
    echo "Restarting farmOS services..."
    
    farmos::manage::stop
    sleep 2
    farmos::manage::start --wait
    
    return 0
}

# Health check function
farmos::health_check() {
    # farmOS doesn't have a /health endpoint, check the homepage or API
    timeout 5 curl -sf "${FARMOS_BASE_URL}/" > /dev/null 2>&1 || \
    timeout 5 curl -sf "${FARMOS_API_BASE}" > /dev/null 2>&1
}

# Status function
farmos::status() {
    local format="${1:-text}"
    
    if farmos::health_check; then
        local status="running"
        local health="healthy"
    else
        # Check if containers are running
        if docker ps | grep -q farmos; then
            local status="running"
            local health="unhealthy"
        else
            local status="stopped"
            local health="n/a"
        fi
    fi
    
    if [[ "$format" == "--json" ]] || [[ "$format" == "json" ]]; then
        cat << EOF
{
  "status": "${status}",
  "health": "${health}",
  "port": ${FARMOS_PORT},
  "url": "${FARMOS_BASE_URL}",
  "api": "${FARMOS_API_BASE}"
}
EOF
    else
        echo "farmOS Status:"
        echo "  Service: ${status}"
        echo "  Health: ${health}"
        echo "  URL: ${FARMOS_BASE_URL}"
        echo "  API: ${FARMOS_API_BASE}"
        echo "  Admin: ${FARMOS_ADMIN_USER}"
    fi
}

# Logs function
farmos::logs() {
    local tail="${1:-50}"
    local follow="${2:-}"
    
    if [[ "$follow" == "--follow" ]]; then
        docker-compose -f "${FARMOS_COMPOSE_FILE}" logs -f --tail="$tail"
    else
        docker-compose -f "${FARMOS_COMPOSE_FILE}" logs --tail="$tail"
    fi
}

# Credentials function
farmos::credentials() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]] || [[ "$format" == "json" ]]; then
        cat << EOF
{
  "admin_user": "${FARMOS_ADMIN_USER}",
  "admin_password": "${FARMOS_ADMIN_PASSWORD}",
  "api_endpoint": "${FARMOS_API_BASE}",
  "oauth_enabled": ${FARMOS_OAUTH_ENABLED}
}
EOF
    else
        echo "farmOS Credentials:"
        echo "  Admin User: ${FARMOS_ADMIN_USER}"
        echo "  Admin Password: ${FARMOS_ADMIN_PASSWORD}"
        echo "  API Endpoint: ${FARMOS_API_BASE}"
        echo "  OAuth Enabled: ${FARMOS_OAUTH_ENABLED}"
    fi
}

# Content management functions
farmos::content::add() {
    local entity_type="${1:-field}"
    shift
    
    case "$entity_type" in
        field)
            farmos::api::field::create "$@"
            ;;
        asset|equipment|animal)
            farmos::api::asset::create "$entity_type" "$@"
            ;;
        log|activity|observation)
            farmos::api::log::create "$entity_type" "$@"
            ;;
        *)
            echo "Error: Unknown entity type: $entity_type"
            echo "Valid types: field, asset, equipment, animal, log, activity, observation"
            return 1
            ;;
    esac
}

farmos::content::list() {
    local entity_type="${1:-all}"
    
    case "$entity_type" in
        field|fields)
            farmos::api::field::list
            ;;
        asset|assets)
            farmos::api::asset::list
            ;;
        log|logs)
            farmos::api::log::list
            ;;
        all)
            echo "=== Fields ==="
            farmos::api::field::list
            echo ""
            echo "=== Assets ==="
            farmos::api::asset::list
            echo ""
            echo "=== Recent Logs ==="
            farmos::api::log::list "" 10
            ;;
        *)
            echo "Error: Unknown entity type: $entity_type"
            return 1
            ;;
    esac
}

farmos::content::get() {
    local entity_type="${1:-}"
    local entity_id="${2:-}"
    
    if [[ -z "$entity_type" ]] || [[ -z "$entity_id" ]]; then
        echo "Error: Entity type and ID required"
        echo "Usage: resource-farmos content get <type> <id>"
        return 1
    fi
    
    case "$entity_type" in
        field)
            farmos::api::field::get "$entity_id"
            ;;
        *)
            echo "Error: Get operation not yet implemented for: $entity_type"
            return 1
            ;;
    esac
}

farmos::content::remove() {
    local entity_type="${1:-}"
    local entity_id="${2:-}"
    
    if [[ -z "$entity_type" ]] || [[ -z "$entity_id" ]]; then
        echo "Error: Entity type and ID required"
        echo "Usage: resource-farmos content remove <type> <id>"
        return 1
    fi
    
    case "$entity_type" in
        field)
            farmos::api::field::delete "$entity_id"
            ;;
        *)
            echo "Error: Remove operation not yet implemented for: $entity_type"
            return 1
            ;;
    esac
}

farmos::content::execute() {
    local operation="${1:-}"
    shift
    
    case "$operation" in
        seed-demo)
            farmos::api::seed_demo
            ;;
        export)
            farmos::api::export "$@"
            ;;
        authenticate)
            farmos::api::authenticate "$@"
            ;;
        *)
            echo "Error: Unknown operation: $operation"
            echo "Valid operations: seed-demo, export, authenticate"
            return 1
            ;;
    esac
}

# Farm-specific operations (P0 requirements)
farmos::farm::create_field() {
    local name="${1:-}"
    local size="${2:-}"
    local unit="${3:-acres}"
    local description="${4:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Field name is required"
        echo "Usage: resource-farmos farm create-field --name NAME [--size SIZE] [--unit UNIT] [--description DESC]"
        return 1
    fi
    
    # Use API to create field
    farmos::api::field::create "$name" "$size" "$unit" "$description"
}

farmos::farm::log_activity() {
    local type="${1:-activity}"
    local name="${2:-}"
    local field="${3:-}"
    local notes="${4:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Activity name is required"
        echo "Usage: resource-farmos farm log-activity --type TYPE --name NAME [--field FIELD] [--notes NOTES]"
        return 1
    fi
    
    # Use API to create log
    farmos::api::log::create "$type" "$name" "$field" "$notes"
}

farmos::farm::export() {
    local entity="${1:-all}"
    local format="${2:-json}"
    local output="${3:-farm_export.${format}}"
    
    # Use API to export data
    farmos::api::export "$entity" "$format" "$output"
}

farmos::farm::import() {
    echo "Importing farm data..."
    # This would require parsing import file and calling appropriate API methods
    echo "Import functionality requires file parsing - implementation pending"
    return 0
}

farmos::farm::seed_demo() {
    echo "Seeding demo data..."
    
    # Check if farmOS is running
    if ! farmos::health_check; then
        echo "Error: farmOS must be running to seed demo data"
        return 1
    fi
    
    # Use API to seed demo data
    farmos::api::seed_demo
}

# IoT integration functions (P1 requirements)
farmos::iot::connect() {
    echo "Connecting to IoT broker..."
    # Implementation would connect to MQTT broker
    echo "IoT integration coming soon"
    return 0
}

farmos::iot::ingest() {
    echo "Ingesting sensor data..."
    # Implementation would handle sensor data ingestion
    echo "Sensor data ingestion coming soon"
    return 0
}

farmos::iot::sync() {
    echo "Syncing sensor data to farmOS..."
    # Implementation would sync data to farmOS entities
    echo "Data sync functionality coming soon"
    return 0
}