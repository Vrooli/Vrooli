#!/usr/bin/env bash
# Apache Airflow Resource - Core Functionality

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Install Apache Airflow and dependencies
install() {
    echo "Installing Apache Airflow..."
    
    # Create necessary directories
    mkdir -p "${AIRFLOW_HOME}"
    mkdir -p "${AIRFLOW_DAG_DIR}"
    mkdir -p "${AIRFLOW_LOG_DIR}"
    mkdir -p "${AIRFLOW_PLUGIN_DIR}"
    mkdir -p "${AIRFLOW_DATA_DIR}"
    mkdir -p "${RESOURCE_DIR}/docker"
    
    # Create docker-compose file
    create_docker_compose
    
    # Create airflow configuration
    create_airflow_config
    
    # Create example DAGs if enabled
    if [[ "${AIRFLOW_ENABLE_EXAMPLE_DAGS}" == "true" ]]; then
        create_example_dags
    fi
    
    # Pull Docker images
    echo "Pulling Docker images..."
    docker pull "${AIRFLOW_DOCKER_IMAGE}" || {
        echo "Failed to pull Airflow image"
        return 1
    }
    docker pull postgres:13 || {
        echo "Failed to pull PostgreSQL image"
        return 1
    }
    docker pull redis:7 || {
        echo "Failed to pull Redis image"
        return 1
    }
    
    echo "Apache Airflow installed successfully"
    return 0
}

# Start Airflow services
start() {
    local wait_flag=false
    local timeout="${AIRFLOW_STARTUP_TIMEOUT}"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait) wait_flag=true ;;
            --timeout) timeout="$2"; shift ;;
            *) ;;
        esac
        shift
    done
    
    echo "Starting Apache Airflow services..."
    
    # Start services with docker-compose
    cd "${RESOURCE_DIR}/docker"
    docker-compose up -d || {
        echo "Failed to start Airflow services"
        return 1
    }
    
    if [[ "$wait_flag" == true ]]; then
        echo "Waiting for services to be ready (timeout: ${timeout}s)..."
        wait_for_health "$timeout"
    fi
    
    echo "Apache Airflow started successfully"
    echo "Web UI: http://localhost:${AIRFLOW_WEBSERVER_PORT}"
    echo "Username: ${AIRFLOW_ADMIN_USER}"
    echo "Password: ${AIRFLOW_ADMIN_PASSWORD}"
    
    return 0
}

# Stop Airflow services
stop() {
    echo "Stopping Apache Airflow services..."
    
    if [[ -f "${RESOURCE_DIR}/docker/docker-compose.yml" ]]; then
        cd "${RESOURCE_DIR}/docker"
        docker-compose down || {
            echo "Failed to stop Airflow services"
            return 1
        }
    fi
    
    echo "Apache Airflow stopped successfully"
    return 0
}

# Restart Airflow services
restart() {
    echo "Restarting Apache Airflow services..."
    stop
    sleep 2
    start "$@"
    return $?
}

# Uninstall Apache Airflow
uninstall() {
    local keep_data=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --keep-data) keep_data=true ;;
            *) ;;
        esac
        shift
    done
    
    echo "Uninstalling Apache Airflow..."
    
    # Stop services first
    stop
    
    # Remove Docker volumes if not keeping data
    if [[ "$keep_data" == false ]]; then
        echo "Removing Docker volumes..."
        cd "${RESOURCE_DIR}/docker"
        docker-compose down -v 2>/dev/null || true
    fi
    
    # Remove directories if not keeping data
    if [[ "$keep_data" == false ]]; then
        echo "Removing Airflow directories..."
        rm -rf "${AIRFLOW_HOME}"
    fi
    
    echo "Apache Airflow uninstalled successfully"
    return 0
}

# Show service status
status() {
    local json_flag=false
    
    for arg in "$@"; do
        case "$arg" in
            --json) json_flag=true ;;
        esac
    done
    
    if [[ "$json_flag" == true ]]; then
        check_health_json
    else
        echo "Apache Airflow Service Status:"
        echo "================================"
        
        # Check webserver
        if timeout 2 curl -sf "http://localhost:${AIRFLOW_WEBSERVER_PORT}/health" &>/dev/null; then
            echo "✓ Webserver: Running (port ${AIRFLOW_WEBSERVER_PORT})"
        else
            echo "✗ Webserver: Not responding"
        fi
        
        # Check scheduler
        if docker ps --format "table {{.Names}}" | grep -q "airflow-scheduler"; then
            echo "✓ Scheduler: Running"
        else
            echo "✗ Scheduler: Not running"
        fi
        
        # Check workers
        local worker_count=$(docker ps --format "table {{.Names}}" | grep -c "airflow-worker" || true)
        if [[ "$worker_count" -gt 0 ]]; then
            echo "✓ Workers: $worker_count running"
        else
            echo "✗ Workers: None running"
        fi
        
        # Check Redis
        if docker ps --format "table {{.Names}}" | grep -q "airflow-redis"; then
            echo "✓ Redis: Running"
        else
            echo "✗ Redis: Not running"
        fi
        
        # Check PostgreSQL
        if docker ps --format "table {{.Names}}" | grep -q "airflow-postgres"; then
            echo "✓ PostgreSQL: Running"
        else
            echo "✗ PostgreSQL: Not running"
        fi
    fi
    
    return 0
}

# Show service logs
logs() {
    local service="${1:-all}"
    local follow=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --follow|-f) follow=true ;;
            *) service="$1" ;;
        esac
        shift
    done
    
    cd "${RESOURCE_DIR}/docker"
    
    if [[ "$follow" == true ]]; then
        docker-compose logs -f "$service"
    else
        docker-compose logs --tail=100 "$service"
    fi
    
    return 0
}

# Helper: Create docker-compose configuration
create_docker_compose() {
    cat > "${RESOURCE_DIR}/docker/docker-compose.yml" << EOF
version: '3.8'

x-airflow-common:
  &airflow-common
  image: ${AIRFLOW_DOCKER_IMAGE}
  environment:
    &airflow-common-env
    AIRFLOW__CORE__EXECUTOR: ${AIRFLOW_EXECUTOR}
    AIRFLOW__DATABASE__SQL_ALCHEMY_CONN: postgresql+psycopg2://${AIRFLOW_POSTGRES_USER}:${AIRFLOW_POSTGRES_PASSWORD}@postgres/${AIRFLOW_POSTGRES_DB}
    AIRFLOW__CELERY__RESULT_BACKEND: db+postgresql://${AIRFLOW_POSTGRES_USER}:${AIRFLOW_POSTGRES_PASSWORD}@postgres/${AIRFLOW_POSTGRES_DB}
    AIRFLOW__CELERY__BROKER_URL: redis://:@redis:6379/0
    AIRFLOW__CORE__FERNET_KEY: ''
    AIRFLOW__CORE__DAGS_ARE_PAUSED_AT_CREATION: 'true'
    AIRFLOW__CORE__LOAD_EXAMPLES: '${AIRFLOW_ENABLE_EXAMPLE_DAGS}'
    AIRFLOW__API__AUTH_BACKENDS: 'airflow.api.auth.backend.basic_auth,airflow.api.auth.backend.session'
    AIRFLOW__WEBSERVER__EXPOSE_CONFIG: 'true'
    _PIP_ADDITIONAL_REQUIREMENTS: ''
  volumes:
    - ${AIRFLOW_DAG_DIR}:/opt/airflow/dags
    - ${AIRFLOW_LOG_DIR}:/opt/airflow/logs
    - ${AIRFLOW_PLUGIN_DIR}:/opt/airflow/plugins
  user: "\${AIRFLOW_UID:-50000}:\${AIRFLOW_GID:-0}"
  depends_on:
    &airflow-common-depends-on
    redis:
      condition: service_healthy
    postgres:
      condition: service_healthy

services:
  postgres:
    image: postgres:13
    container_name: airflow-postgres
    environment:
      POSTGRES_USER: ${AIRFLOW_POSTGRES_USER}
      POSTGRES_PASSWORD: ${AIRFLOW_POSTGRES_PASSWORD}
      POSTGRES_DB: ${AIRFLOW_POSTGRES_DB}
    volumes:
      - postgres-db-volume:/var/lib/postgresql/data
    ports:
      - "${AIRFLOW_POSTGRES_PORT}:5432"
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "${AIRFLOW_POSTGRES_USER}"]
      interval: 10s
      retries: 5
      start_period: 5s
    restart: always

  redis:
    image: redis:7
    container_name: airflow-redis
    expose:
      - 6379
    ports:
      - "${AIRFLOW_REDIS_PORT}:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 30s
      retries: 50
      start_period: 30s
    restart: always

  airflow-webserver:
    <<: *airflow-common
    container_name: airflow-webserver
    command: webserver
    ports:
      - "${AIRFLOW_WEBSERVER_PORT}:8080"
    healthcheck:
      test: ["CMD", "curl", "--fail", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 30s
    restart: always
    depends_on:
      <<: *airflow-common-depends-on
      airflow-init:
        condition: service_completed_successfully

  airflow-scheduler:
    <<: *airflow-common
    container_name: airflow-scheduler
    command: scheduler
    healthcheck:
      test: ["CMD", "curl", "--fail", "http://localhost:8974/health"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 30s
    restart: always
    depends_on:
      <<: *airflow-common-depends-on
      airflow-init:
        condition: service_completed_successfully

  airflow-worker:
    <<: *airflow-common
    container_name: airflow-worker
    command: celery worker
    healthcheck:
      test:
        - "CMD-SHELL"
        - 'celery --app airflow.providers.celery.executors.celery_executor.app inspect ping -d "celery@\$\${HOSTNAME}" || celery --app airflow.executors.celery_executor.app inspect ping -d "celery@\$\${HOSTNAME}"'
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 30s
    environment:
      <<: *airflow-common-env
      DUMB_INIT_SETSID: "0"
    restart: always
    depends_on:
      <<: *airflow-common-depends-on
      airflow-init:
        condition: service_completed_successfully

  airflow-triggerer:
    <<: *airflow-common
    container_name: airflow-triggerer
    command: triggerer
    healthcheck:
      test: ["CMD-SHELL", 'airflow jobs check --job-type TriggererJob --hostname "\$\${HOSTNAME}"']
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 30s
    restart: always
    depends_on:
      <<: *airflow-common-depends-on
      airflow-init:
        condition: service_completed_successfully

  airflow-init:
    <<: *airflow-common
    container_name: airflow-init
    entrypoint: /bin/bash
    command:
      - -c
      - |
        if [[ -z "\${AIRFLOW_UID}" ]]; then
          echo
          echo -e "\\033[1;33mWARNING!!!: AIRFLOW_UID not set!\\e[0m"
          echo "If you are on Linux, you SHOULD follow the instructions below to set "
          echo "AIRFLOW_UID environment variable, otherwise files will be owned by root."
          echo "For other operating systems you can get rid of the warning with manually created .env file:"
          echo "    See: https://airflow.apache.org/docs/apache-airflow/stable/howto/docker-compose/index.html#setting-the-right-airflow-user"
          echo
        fi
        one_meg=\$((1024 * 1024))
        mem_available=\$(($(getconf _PHYS_PAGES) * $(getconf PAGE_SIZE) / one_meg))
        cpus_available=\$(grep -cE 'cpu[0-9]+' /proc/stat)
        disk_available=\$(df / | tail -1 | awk '{print \$4}')
        warning_resources="false"
        if (( mem_available < 4000 )) ; then
          echo
          echo -e "\\033[1;33mWARNING!!!: Not enough memory available for Docker.\\e[0m"
          echo "At least 4GB of memory required. You have \$((mem_available / 1024))GB"
          echo
          warning_resources="true"
        fi
        if (( cpus_available < 2 )); then
          echo
          echo -e "\\033[1;33mWARNING!!!: Not enough CPUS available for Docker.\\e[0m"
          echo "At least 2 CPUs recommended. You have \${cpus_available}"
          echo
          warning_resources="true"
        fi
        if (( disk_available < one_meg * 10 )); then
          echo
          echo -e "\\033[1;33mWARNING!!!: Not enough Disk space available for Docker.\\e[0m"
          echo "At least 10 GBs recommended. You have \$((disk_available / one_meg / 1024 ))GB"
          echo
          warning_resources="true"
        fi
        if [[ \${warning_resources} == "true" ]]; then
          echo
          echo -e "\\033[1;33mWARNING!!!: You have not enough resources to run Airflow (see above)!\\e[0m"
          echo "Please follow the instructions to increase amount of resources available:"
          echo "   https://airflow.apache.org/docs/apache-airflow/stable/howto/docker-compose/index.html#before-you-begin"
          echo
        fi
        mkdir -p /sources/logs /sources/dags /sources/plugins
        chown -R "\${AIRFLOW_UID}:\${AIRFLOW_GID}" /sources/{logs,dags,plugins}
        exec /entrypoint airflow version
    environment:
      <<: *airflow-common-env
      _AIRFLOW_DB_MIGRATE: 'true'
      _AIRFLOW_WWW_USER_CREATE: 'true'
      _AIRFLOW_WWW_USER_USERNAME: \${AIRFLOW_ADMIN_USER:-admin}
      _AIRFLOW_WWW_USER_PASSWORD: \${AIRFLOW_ADMIN_PASSWORD:-admin}
      _PIP_ADDITIONAL_REQUIREMENTS: ''
    user: "0:0"
    volumes:
      - ${AIRFLOW_HOME}:/sources

  airflow-cli:
    <<: *airflow-common
    profiles:
      - debug
    environment:
      <<: *airflow-common-env
      CONNECTION_CHECK_MAX_COUNT: "0"
    command:
      - bash
      - -c
      - airflow

volumes:
  postgres-db-volume:

networks:
  default:
    name: ${AIRFLOW_DOCKER_NETWORK}
EOF
}

# Helper: Create Airflow configuration
create_airflow_config() {
    cat > "${RESOURCE_DIR}/docker/airflow.cfg" << EOF
[core]
executor = ${AIRFLOW_EXECUTOR}
parallelism = ${AIRFLOW_PARALLELISM}
dag_concurrency = ${AIRFLOW_DAG_CONCURRENCY}
max_active_runs_per_dag = ${AIRFLOW_MAX_ACTIVE_RUNS}
load_examples = ${AIRFLOW_ENABLE_EXAMPLE_DAGS}
dags_folder = /opt/airflow/dags
base_log_folder = /opt/airflow/logs
plugins_folder = /opt/airflow/plugins

[webserver]
base_url = http://localhost:${AIRFLOW_WEBSERVER_PORT}
expose_config = True
expose_hostname = True
expose_stacktrace = True

[scheduler]
min_file_process_interval = 30
dag_dir_list_interval = 300
print_stats_interval = 30
scheduler_heartbeat_sec = 5
max_threads = ${AIRFLOW_WORKER_CONCURRENCY}

[celery]
worker_concurrency = ${AIRFLOW_WORKER_CONCURRENCY}
worker_prefetch_multiplier = 1
worker_max_tasks_per_child = 1000

[logging]
logging_level = ${AIRFLOW_LOG_LEVEL}
fab_logging_level = WARN
EOF
}

# Helper: Create example DAGs
create_example_dags() {
    cat > "${AIRFLOW_DAG_DIR}/example_etl.py" << 'EOF'
from datetime import datetime, timedelta
from airflow import DAG
from airflow.operators.python import PythonOperator
from airflow.operators.bash import BashOperator

default_args = {
    'owner': 'vrooli',
    'depends_on_past': False,
    'start_date': datetime(2025, 1, 1),
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 1,
    'retry_delay': timedelta(minutes=5),
}

dag = DAG(
    'example_etl_pipeline',
    default_args=default_args,
    description='Example ETL pipeline for Vrooli',
    schedule_interval=timedelta(days=1),
    catchup=False,
)

def extract_data(**context):
    """Extract data from source"""
    print("Extracting data...")
    return {"records": 100, "source": "example_db"}

def transform_data(**context):
    """Transform extracted data"""
    ti = context['ti']
    data = ti.xcom_pull(task_ids='extract')
    print(f"Transforming {data['records']} records from {data['source']}")
    return {"transformed_records": data['records'] * 2}

def load_data(**context):
    """Load data to destination"""
    ti = context['ti']
    data = ti.xcom_pull(task_ids='transform')
    print(f"Loading {data['transformed_records']} records to warehouse")
    return "Success"

extract_task = PythonOperator(
    task_id='extract',
    python_callable=extract_data,
    dag=dag,
)

transform_task = PythonOperator(
    task_id='transform',
    python_callable=transform_data,
    dag=dag,
)

load_task = PythonOperator(
    task_id='load',
    python_callable=load_data,
    dag=dag,
)

cleanup_task = BashOperator(
    task_id='cleanup',
    bash_command='echo "Pipeline completed successfully"',
    dag=dag,
)

extract_task >> transform_task >> load_task >> cleanup_task
EOF
}

# Helper: Wait for services to be healthy
wait_for_health() {
    local timeout="${1:-${AIRFLOW_STARTUP_TIMEOUT}}"
    local elapsed=0
    
    while [[ "$elapsed" -lt "$timeout" ]]; do
        if timeout 5 curl -sf "http://localhost:${AIRFLOW_WEBSERVER_PORT}/health" &>/dev/null; then
            echo "Airflow is healthy"
            return 0
        fi
        
        echo -n "."
        sleep 5
        elapsed=$((elapsed + 5))
    done
    
    echo ""
    echo "Timeout waiting for Airflow to be healthy"
    return 1
}

# Helper: Check health and return JSON
check_health_json() {
    local healthy=true
    local services=()
    
    # Check webserver
    if timeout 2 curl -sf "http://localhost:${AIRFLOW_WEBSERVER_PORT}/health" &>/dev/null; then
        services+=('{"name":"webserver","status":"running","port":'${AIRFLOW_WEBSERVER_PORT}'}')
    else
        services+=('{"name":"webserver","status":"stopped","port":'${AIRFLOW_WEBSERVER_PORT}'}')
        healthy=false
    fi
    
    # Check scheduler
    if docker ps --format "table {{.Names}}" | grep -q "airflow-scheduler"; then
        services+=('{"name":"scheduler","status":"running"}')
    else
        services+=('{"name":"scheduler","status":"stopped"}')
        healthy=false
    fi
    
    # Output JSON
    echo -n '{"healthy":'
    if [[ "$healthy" == true ]]; then
        echo -n 'true'
    else
        echo -n 'false'
    fi
    echo -n ',"services":['
    printf '%s' "${services[0]}"
    for ((i=1; i<${#services[@]}; i++)); do
        printf ',%s' "${services[$i]}"
    done
    echo ']}'
}

# Export functions for use by CLI
export -f install
export -f start
export -f stop
export -f restart
export -f uninstall
export -f status
export -f logs

# Handle direct script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    command="${1:-}"
    shift || true
    
    case "$command" in
        install|start|stop|restart|uninstall|status|logs)
            "$command" "$@"
            ;;
        *)
            echo "Usage: $0 {install|start|stop|restart|uninstall|status|logs}"
            exit 1
            ;;
    esac
fi