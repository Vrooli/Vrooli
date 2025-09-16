#!/usr/bin/env bash
# Apache Superset Core Functions

# Show help information
superset::show_help() {
    cat << EOF
Apache Superset - Enterprise Analytics Platform

Usage: resource-apache-superset [COMMAND] [OPTIONS]

Commands:
    help                    Show this help message
    info                    Show resource information
    manage install          Install Apache Superset
    manage start            Start Superset services
    manage stop             Stop Superset services
    manage restart          Restart Superset services
    manage uninstall        Uninstall Superset (preserves data)
    test smoke              Quick health check
    test integration        Full integration test
    test all                Run all tests
    content list            List dashboards and charts
    content add             Create new dashboard/chart
    content get             Export dashboard/chart
    content remove          Delete dashboard/chart
    content init-templates  Initialize starter dashboards and alerts
    status                  Show service status
    logs                    View service logs
    credentials             Display access credentials

Examples:
    # Install and start Superset
    resource-apache-superset manage install
    resource-apache-superset manage start --wait
    
    # Create a dashboard
    resource-apache-superset content add --type dashboard --name "KPI Dashboard"
    
    # Export a dashboard
    resource-apache-superset content get --id 1 --output dashboard.json
    
    # Check status
    resource-apache-superset status --json

Configuration:
    Port: ${SUPERSET_PORT}
    Admin User: ${SUPERSET_ADMIN_USERNAME}
    Data Directory: ${SUPERSET_DATA_DIR}

For more information, see: /home/matthalloran8/Vrooli/resources/apache-superset/README.md
EOF
}

# Show resource information
superset::show_info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "Apache Superset Resource Information:"
        echo "======================================"
        jq -r '. | to_entries[] | "\(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

# Manage lifecycle commands
superset::manage() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        install)
            superset::install "$@"
            ;;
        start)
            superset::start "$@"
            ;;
        stop)
            superset::stop "$@"
            ;;
        restart)
            superset::restart "$@"
            ;;
        uninstall)
            superset::uninstall "$@"
            ;;
        *)
            echo "Error: Unknown manage command '$action'"
            echo "Valid commands: install, start, stop, restart, uninstall"
            return 1
            ;;
    esac
}

# Install Superset
superset::install() {
    log::info "Installing Apache Superset..."
    
    # Create data directories
    mkdir -p "${SUPERSET_CONFIG_DIR}" "${SUPERSET_POSTGRES_DATA}" \
             "${SUPERSET_UPLOADS_DIR}" "${SUPERSET_CACHE_DIR}" \
             "${SUPERSET_DATA_DIR}/logs"
    
    # Pull Docker images
    log::info "Pulling Docker images..."
    docker pull "${SUPERSET_IMAGE}" || return 1
    docker pull "${SUPERSET_POSTGRES_IMAGE}" || return 1
    docker pull "${SUPERSET_REDIS_IMAGE}" || return 1
    
    # Create Docker network
    if ! docker network inspect "${SUPERSET_NETWORK}" &>/dev/null; then
        log::info "Creating Docker network ${SUPERSET_NETWORK}..."
        docker network create "${SUPERSET_NETWORK}" || return 1
    fi
    
    # Create Superset config
    superset::create_config
    
    log::success "Apache Superset installed successfully"
    return 0
}

# Start Superset services
superset::start() {
    local wait_flag=false
    [[ "${1:-}" == "--wait" ]] && wait_flag=true
    
    log::info "Starting Apache Superset services..."
    
    # Start PostgreSQL
    if ! docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_POSTGRES_CONTAINER}$"; then
        log::info "Starting PostgreSQL..."
        docker run -d \
            --name "${SUPERSET_POSTGRES_CONTAINER}" \
            --network "${SUPERSET_NETWORK}" \
            -p "${SUPERSET_POSTGRES_PORT}:5432" \
            -e POSTGRES_USER="${SUPERSET_POSTGRES_USER}" \
            -e POSTGRES_PASSWORD="${SUPERSET_POSTGRES_PASSWORD}" \
            -e POSTGRES_DB="${SUPERSET_POSTGRES_DB}" \
            -v "${SUPERSET_POSTGRES_DATA}:/var/lib/postgresql/data" \
            "${SUPERSET_POSTGRES_IMAGE}" || return 1
    else
        docker start "${SUPERSET_POSTGRES_CONTAINER}" || return 1
    fi
    
    # Start Redis
    if ! docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_REDIS_CONTAINER}$"; then
        log::info "Starting Redis..."
        docker run -d \
            --name "${SUPERSET_REDIS_CONTAINER}" \
            --network "${SUPERSET_NETWORK}" \
            -p "${SUPERSET_REDIS_PORT}:6379" \
            "${SUPERSET_REDIS_IMAGE}" || return 1
    else
        docker start "${SUPERSET_REDIS_CONTAINER}" || return 1
    fi
    
    # Wait for dependencies
    sleep 5
    
    # Database URI for Superset metadata
    local db_uri="postgresql+psycopg2://${SUPERSET_POSTGRES_USER}:${SUPERSET_POSTGRES_PASSWORD}@${SUPERSET_POSTGRES_CONTAINER}:5432/${SUPERSET_POSTGRES_DB}"
    
    # Initialize database (if needed)
    if ! docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_APP_CONTAINER}$"; then
        log::info "Initializing Superset database..."
        docker run --rm \
            --network "${SUPERSET_NETWORK}" \
            -e SUPERSET_SECRET_KEY="${SUPERSET_SECRET_KEY}" \
            -e SQLALCHEMY_DATABASE_URI="${db_uri}" \
            -v "${SUPERSET_CONFIG_DIR}:/app/pythonpath" \
            "${SUPERSET_IMAGE}" superset db upgrade || return 1
        
        # Create admin user
        log::info "Creating admin user..."
        docker run --rm \
            --network "${SUPERSET_NETWORK}" \
            -e SUPERSET_SECRET_KEY="${SUPERSET_SECRET_KEY}" \
            -e SQLALCHEMY_DATABASE_URI="${db_uri}" \
            "${SUPERSET_IMAGE}" superset fab create-admin \
            --username "${SUPERSET_ADMIN_USERNAME}" \
            --password "${SUPERSET_ADMIN_PASSWORD}" \
            --firstname Admin \
            --lastname User \
            --email "${SUPERSET_ADMIN_EMAIL}" || true
        
        # Initialize Superset
        docker run --rm \
            --network "${SUPERSET_NETWORK}" \
            -e SUPERSET_SECRET_KEY="${SUPERSET_SECRET_KEY}" \
            -e SQLALCHEMY_DATABASE_URI="${db_uri}" \
            -v "${SUPERSET_CONFIG_DIR}:/app/pythonpath" \
            "${SUPERSET_IMAGE}" superset init || return 1
    fi
    
    # Start Superset app
    if ! docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_APP_CONTAINER}$"; then
        log::info "Starting Superset application..."
        docker run -d \
            --name "${SUPERSET_APP_CONTAINER}" \
            --network "${SUPERSET_NETWORK}" \
            -p "${SUPERSET_PORT}:8088" \
            -e SUPERSET_SECRET_KEY="${SUPERSET_SECRET_KEY}" \
            -e SQLALCHEMY_DATABASE_URI="${db_uri}" \
            -e REDIS_HOST="${SUPERSET_REDIS_CONTAINER}" \
            -e REDIS_PORT=6379 \
            -v "${SUPERSET_CONFIG_DIR}:/app/pythonpath" \
            -v "${SUPERSET_UPLOADS_DIR}:/app/superset_home" \
            "${SUPERSET_IMAGE}" || return 1
    else
        docker start "${SUPERSET_APP_CONTAINER}" || return 1
    fi
    
    # Start Celery worker
    if ! docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_WORKER_CONTAINER}$"; then
        log::info "Starting Celery worker..."
        docker run -d \
            --name "${SUPERSET_WORKER_CONTAINER}" \
            --network "${SUPERSET_NETWORK}" \
            -e SUPERSET_SECRET_KEY="${SUPERSET_SECRET_KEY}" \
            -e SQLALCHEMY_DATABASE_URI="${db_uri}" \
            -e REDIS_HOST="${SUPERSET_REDIS_CONTAINER}" \
            -e REDIS_PORT=6379 \
            -v "${SUPERSET_CONFIG_DIR}:/app/pythonpath" \
            "${SUPERSET_IMAGE}" celery --app=superset.tasks.celery_app:app worker || return 1
    else
        docker start "${SUPERSET_WORKER_CONTAINER}" || return 1
    fi
    
    # Start Celery beat
    if ! docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_WORKER_BEAT_CONTAINER}$"; then
        log::info "Starting Celery beat..."
        docker run -d \
            --name "${SUPERSET_WORKER_BEAT_CONTAINER}" \
            --network "${SUPERSET_NETWORK}" \
            -e SUPERSET_SECRET_KEY="${SUPERSET_SECRET_KEY}" \
            -e SQLALCHEMY_DATABASE_URI="${db_uri}" \
            -e REDIS_HOST="${SUPERSET_REDIS_CONTAINER}" \
            -e REDIS_PORT=6379 \
            -v "${SUPERSET_CONFIG_DIR}:/app/pythonpath" \
            "${SUPERSET_IMAGE}" celery --app=superset.tasks.celery_app:app beat || return 1
    else
        docker start "${SUPERSET_WORKER_BEAT_CONTAINER}" || return 1
    fi
    
    if [[ "$wait_flag" == true ]]; then
        log::info "Waiting for Superset to be ready..."
        superset::wait_ready
    fi
    
    log::success "Apache Superset started successfully"
    echo "Access the UI at: http://localhost:${SUPERSET_PORT}"
    echo "Login with: ${SUPERSET_ADMIN_USERNAME} / ${SUPERSET_ADMIN_PASSWORD}"
    return 0
}

# Stop Superset services
superset::stop() {
    log::info "Stopping Apache Superset services..."
    
    docker stop "${SUPERSET_WORKER_BEAT_CONTAINER}" 2>/dev/null || true
    docker stop "${SUPERSET_WORKER_CONTAINER}" 2>/dev/null || true
    docker stop "${SUPERSET_APP_CONTAINER}" 2>/dev/null || true
    docker stop "${SUPERSET_REDIS_CONTAINER}" 2>/dev/null || true
    docker stop "${SUPERSET_POSTGRES_CONTAINER}" 2>/dev/null || true
    
    log::success "Apache Superset stopped"
    return 0
}

# Restart services
superset::restart() {
    superset::stop
    sleep 2
    superset::start "$@"
}

# Uninstall Superset
superset::uninstall() {
    local keep_data=false
    [[ "$1" == "--keep-data" ]] && keep_data=true
    
    log::info "Uninstalling Apache Superset..."
    
    # Stop and remove containers
    docker stop "${SUPERSET_WORKER_BEAT_CONTAINER}" 2>/dev/null || true
    docker stop "${SUPERSET_WORKER_CONTAINER}" 2>/dev/null || true
    docker stop "${SUPERSET_APP_CONTAINER}" 2>/dev/null || true
    docker stop "${SUPERSET_REDIS_CONTAINER}" 2>/dev/null || true
    docker stop "${SUPERSET_POSTGRES_CONTAINER}" 2>/dev/null || true
    
    docker rm "${SUPERSET_WORKER_BEAT_CONTAINER}" 2>/dev/null || true
    docker rm "${SUPERSET_WORKER_CONTAINER}" 2>/dev/null || true
    docker rm "${SUPERSET_APP_CONTAINER}" 2>/dev/null || true
    docker rm "${SUPERSET_REDIS_CONTAINER}" 2>/dev/null || true
    docker rm "${SUPERSET_POSTGRES_CONTAINER}" 2>/dev/null || true
    
    # Remove network
    docker network rm "${SUPERSET_NETWORK}" 2>/dev/null || true
    
    # Remove data if requested
    if [[ "$keep_data" == false ]]; then
        log::warning "Removing all Superset data..."
        rm -rf "${SUPERSET_DATA_DIR}"
    else
        log::info "Preserving Superset data in ${SUPERSET_DATA_DIR}"
    fi
    
    log::success "Apache Superset uninstalled"
    return 0
}

# Content management
superset::content() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            superset::content_list "$@"
            ;;
        add)
            superset::content_add "$@"
            ;;
        get)
            superset::content_get "$@"
            ;;
        remove)
            superset::content_remove "$@"
            ;;
        execute)
            superset::content_execute "$@"
            ;;
        init-templates)
            superset::content_init_templates "$@"
            ;;
        *)
            echo "Error: Unknown content command '$action'"
            echo "Valid commands: list, add, get, remove, execute, init-templates"
            return 1
            ;;
    esac
}

# List content (dashboards/charts)
superset::content_list() {
    local token=$(superset::get_auth_token)
    
    echo "Dashboards:"
    curl -s -H "Authorization: Bearer ${token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/" | \
        jq -r '.result[] | "  - \(.id): \(.dashboard_title)"'
    
    echo -e "\nCharts:"
    curl -s -H "Authorization: Bearer ${token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/chart/" | \
        jq -r '.result[] | "  - \(.id): \(.slice_name)"'
}

# Add content
superset::content_add() {
    local type="dashboard"
    local name="New Dashboard"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type) type="$2"; shift 2 ;;
            --name) name="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    local token=$(superset::get_auth_token)
    
    if [[ "$type" == "dashboard" ]]; then
        curl -X POST -H "Authorization: Bearer ${token}" \
            -H "Content-Type: application/json" \
            "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/" \
            -d "{\"dashboard_title\": \"${name}\"}"
    else
        echo "Chart creation requires more parameters. Use the web UI or API directly."
    fi
}

# Get/export content
superset::content_get() {
    local id=""
    local output=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id) id="$2"; shift 2 ;;
            --output) output="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$id" ]]; then
        echo "Error: --id required"
        return 1
    fi
    
    local token=$(superset::get_auth_token)
    local result=$(curl -s -H "Authorization: Bearer ${token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/${id}/export/")
    
    if [[ -n "$output" ]]; then
        echo "$result" > "$output"
        echo "Dashboard exported to $output"
    else
        echo "$result"
    fi
}

# Remove content
superset::content_remove() {
    local id=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id) id="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$id" ]]; then
        echo "Error: --id required"
        return 1
    fi
    
    local token=$(superset::get_auth_token)
    curl -X DELETE -H "Authorization: Bearer ${token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/${id}"
}

# Execute SQL query
superset::content_execute() {
    local query="${1:-SELECT 1}"
    local database_id="${2:-1}"
    
    local token=$(superset::get_auth_token)
    
    curl -X POST -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        "http://localhost:${SUPERSET_PORT}/api/v1/sqllab/execute/" \
        -d "{\"database_id\": ${database_id}, \"sql\": \"${query}\"}"
}

# Initialize starter templates
superset::content_init_templates() {
    # Load the starter content library
    local core_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${core_dir}/starter-content.sh"
    
    # Initialize all starter content
    superset::init_starter_content
}

# Show status
superset::status() {
    local format="${1:-text}"
    
    local app_running=$(docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_APP_CONTAINER}$" && echo "true" || echo "false")
    local postgres_running=$(docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_POSTGRES_CONTAINER}$" && echo "true" || echo "false")
    local redis_running=$(docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_REDIS_CONTAINER}$" && echo "true" || echo "false")
    local worker_running=$(docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_WORKER_CONTAINER}$" && echo "true" || echo "false")
    
    if [[ "$format" == "--json" ]]; then
        cat << EOF
{
  "service": "apache-superset",
  "status": "$([[ "$app_running" == "true" ]] && echo "running" || echo "stopped")",
  "components": {
    "app": "$app_running",
    "postgres": "$postgres_running",
    "redis": "$redis_running",
    "worker": "$worker_running"
  },
  "port": ${SUPERSET_PORT},
  "url": "http://localhost:${SUPERSET_PORT}"
}
EOF
    else
        echo "Apache Superset Status:"
        echo "======================"
        echo "Application: $([[ "$app_running" == "true" ]] && echo "✓ Running" || echo "✗ Stopped")"
        echo "PostgreSQL:  $([[ "$postgres_running" == "true" ]] && echo "✓ Running" || echo "✗ Stopped")"
        echo "Redis:       $([[ "$redis_running" == "true" ]] && echo "✓ Running" || echo "✗ Stopped")"
        echo "Worker:      $([[ "$worker_running" == "true" ]] && echo "✓ Running" || echo "✗ Stopped")"
        echo ""
        echo "URL: http://localhost:${SUPERSET_PORT}"
        
        if [[ "$app_running" == "true" ]]; then
            # Check health endpoint
            if timeout 5 curl -sf "http://localhost:${SUPERSET_PORT}/health" &>/dev/null; then
                echo "Health: ✓ Healthy"
            else
                echo "Health: ✗ Not responding"
            fi
        fi
    fi
}

# View logs
superset::logs() {
    local service="${1:-app}"
    local lines="${2:-50}"
    
    case "$service" in
        app)
            docker logs --tail "$lines" "${SUPERSET_APP_CONTAINER}"
            ;;
        postgres)
            docker logs --tail "$lines" "${SUPERSET_POSTGRES_CONTAINER}"
            ;;
        redis)
            docker logs --tail "$lines" "${SUPERSET_REDIS_CONTAINER}"
            ;;
        worker)
            docker logs --tail "$lines" "${SUPERSET_WORKER_CONTAINER}"
            ;;
        *)
            echo "Error: Unknown service '$service'"
            echo "Valid services: app, postgres, redis, worker"
            return 1
            ;;
    esac
}

# Display credentials
superset::credentials() {
    local format="${1:-text}"
    
    if [[ "$format" == "--format" ]] && [[ "$2" == "json" ]]; then
        cat << EOF
{
  "url": "http://localhost:${SUPERSET_PORT}",
  "username": "${SUPERSET_ADMIN_USERNAME}",
  "password": "${SUPERSET_ADMIN_PASSWORD}",
  "token": "$(superset::get_auth_token 2>/dev/null || echo '')"
}
EOF
    else
        echo "Apache Superset Credentials:"
        echo "============================"
        echo "URL:      http://localhost:${SUPERSET_PORT}"
        echo "Username: ${SUPERSET_ADMIN_USERNAME}"
        echo "Password: ${SUPERSET_ADMIN_PASSWORD}"
    fi
}

# Helper: Wait for Superset to be ready
superset::wait_ready() {
    local retries=30
    while [[ $retries -gt 0 ]]; do
        if timeout 5 curl -sf "http://localhost:${SUPERSET_PORT}/health" &>/dev/null; then
            return 0
        fi
        ((retries--))
        sleep 2
    done
    return 1
}

# Helper: Get authentication token
superset::get_auth_token() {
    curl -s -X POST \
        -H "Content-Type: application/json" \
        "http://localhost:${SUPERSET_PORT}/api/v1/security/login" \
        -d "{\"username\": \"${SUPERSET_ADMIN_USERNAME}\", \"password\": \"${SUPERSET_ADMIN_PASSWORD}\", \"provider\": \"db\"}" | \
        jq -r '.access_token'
}

# Get CSRF token for POST requests
superset::get_csrf_token() {
    local access_token="${1:-$(superset::get_auth_token)}"
    curl -s -X GET \
        -H "Authorization: Bearer ${access_token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/security/csrf_token/" | \
        jq -r '.result'
}

# Helper: Create Superset config file
superset::create_config() {
    cat > "${SUPERSET_CONFIG_DIR}/superset_config.py" << 'EOF'
import os
from celery.schedules import crontab

# Superset specific config
SECRET_KEY = os.environ.get('SUPERSET_SECRET_KEY', 'dev-secret-key')
SQLALCHEMY_DATABASE_URI = os.environ.get('SQLALCHEMY_DATABASE_URI')

# Flask-WTF flag for CSRF
# Disable CSRF for API automation
WTF_CSRF_ENABLED = False
WTF_CSRF_TIME_LIMIT = None

# Add endpoints that need to be exempt from CSRF protection
WTF_CSRF_EXEMPT_LIST = []

# Cache config
CACHE_CONFIG = {
    'CACHE_TYPE': 'RedisCache',
    'CACHE_DEFAULT_TIMEOUT': 300,
    'CACHE_KEY_PREFIX': 'superset_',
    'CACHE_REDIS_HOST': os.environ.get('REDIS_HOST', 'redis'),
    'CACHE_REDIS_PORT': int(os.environ.get('REDIS_PORT', 6379)),
    'CACHE_REDIS_DB': 1,
}

# Celery config
class CeleryConfig:
    broker_url = f"redis://{os.environ.get('REDIS_HOST', 'redis')}:{os.environ.get('REDIS_PORT', 6379)}/0"
    imports = ('superset.sql_lab', 'superset.tasks')
    result_backend = f"redis://{os.environ.get('REDIS_HOST', 'redis')}:{os.environ.get('REDIS_PORT', 6379)}/0"
    worker_prefetch_multiplier = 1
    task_acks_late = False
    beat_schedule = {
        'reports.scheduler': {
            'task': 'reports.scheduler',
            'schedule': crontab(minute='*', hour='*'),
        },
        'reports.prune_log': {
            'task': 'reports.prune_log',
            'schedule': crontab(minute=10, hour=0),
        },
    }

CELERY_CONFIG = CeleryConfig

# Feature flags
FEATURE_FLAGS = {
    'ENABLE_TEMPLATE_PROCESSING': True,
    'ENABLE_EXPLORE_JSON_CSRF_PROTECTION': False,
    'ENABLE_TEMPLATE_REMOVE_FILTERS': False,
    'DASHBOARD_RBAC': True,
}

# Embedding configuration
GUEST_TOKEN_JWT_SECRET = SECRET_KEY
GUEST_TOKEN_JWT_ALGO = 'HS256'
GUEST_TOKEN_HEADER_NAME = 'X-GuestToken'
GUEST_TOKEN_JWT_EXP_SECONDS = 300

# CORS configuration for embedding
ENABLE_CORS = True
CORS_OPTIONS = {
    'supports_credentials': True,
    'allow_headers': ['*'],
    'origins': ['http://localhost:*'],
}

# Row limits
ROW_LIMIT = 100000
VIS_ROW_LIMIT = 10000
SAMPLES_ROW_LIMIT = 1000
SQL_MAX_ROW = 100000

# WebServer config
WEBSERVER_THREADS = 8
WEBSERVER_PORT = 8088

# Allow uploads
UPLOAD_FOLDER = '/app/superset_home/uploads/'
ALLOWED_EXTENSIONS = {'csv', 'tsv', 'txt', 'xls', 'xlsx', 'json'}

# Default cache timeout
CACHE_DEFAULT_TIMEOUT = 60 * 60 * 24  # 1 day

# Pre-configured database connections for Vrooli
SQLALCHEMY_DATABASE_CONNECTIONS = {
    'vrooli_postgres': f"postgresql://user:pass@{os.environ.get('VROOLI_POSTGRES_HOST', 'host.docker.internal')}:{os.environ.get('VROOLI_POSTGRES_PORT', 5433)}/vrooli",
    'vrooli_questdb': f"postgresql://admin:quest@{os.environ.get('VROOLI_QUESTDB_HOST', 'host.docker.internal')}:{os.environ.get('VROOLI_QUESTDB_PORT', 8812)}/qdb",
}
EOF
}