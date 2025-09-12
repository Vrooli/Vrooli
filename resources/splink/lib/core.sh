#!/usr/bin/env bash
# Splink Core Library - Lifecycle and service management

set -euo pipefail

# Resource configuration
RESOURCE_NAME="splink"
SPLINK_PORT="${SPLINK_PORT:-8096}"
SPLINK_BACKEND="${SPLINK_BACKEND:-duckdb}"
SPLINK_CONTAINER="vrooli-splink"
SPLINK_IMAGE="splink:latest"

# Paths
VROOLI_ROOT="${VROOLI_ROOT:-$HOME/Vrooli}"
RESOURCE_DIR="${VROOLI_ROOT}/resources/${RESOURCE_NAME}"
DATA_DIR="${VROOLI_ROOT}/.vrooli/data/${RESOURCE_NAME}"
LOG_DIR="${VROOLI_ROOT}/.vrooli/logs/resources"
LOG_FILE="${LOG_DIR}/${RESOURCE_NAME}.log"

# Ensure directories exist
mkdir -p "$DATA_DIR" "$LOG_DIR"

# Install dependencies
install_dependencies() {
    echo "Installing Splink dependencies..."
    
    # Create data directories
    mkdir -p "${DATA_DIR}/datasets" "${DATA_DIR}/results" "${DATA_DIR}/jobs"
    
    # Build Docker image
    if [[ -f "${RESOURCE_DIR}/Dockerfile" ]]; then
        echo "Building Splink Docker image..."
        docker build -t "$SPLINK_IMAGE" "$RESOURCE_DIR" || {
            echo "Error: Failed to build Docker image"
            return 1
        }
    else
        echo "Creating Dockerfile..."
        create_dockerfile
        docker build -t "$SPLINK_IMAGE" "$RESOURCE_DIR" || {
            echo "Error: Failed to build Docker image"
            return 1
        }
    fi
    
    echo "Splink dependencies installed successfully"
}

# Create Dockerfile if it doesn't exist
create_dockerfile() {
    cat > "${RESOURCE_DIR}/Dockerfile" << 'EOF'
FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Install Python packages
RUN pip install --no-cache-dir \
    splink==3.9.14 \
    duckdb==0.10.0 \
    fastapi==0.109.0 \
    uvicorn==0.27.0 \
    pydantic==2.5.0 \
    pandas==2.1.4 \
    pyarrow==14.0.2 \
    plotly==5.18.0 \
    redis==5.0.1 \
    psycopg2-binary==2.9.9 \
    boto3==1.34.0

# Copy application code
COPY api/ /app/api/
COPY lib/ /app/lib/

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV SPLINK_PORT=8096
ENV SPLINK_BACKEND=duckdb

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:${SPLINK_PORT}/health || exit 1

# Start the service
CMD ["python", "-m", "uvicorn", "api.main:app", "--host", "0.0.0.0", "--port", "8096"]
EOF
}

# Start the service
start_service() {
    local wait_flag="${1:-}"
    
    echo "Starting Splink service..."
    
    # Check if already running
    if check_service_health; then
        echo "Splink is already running"
        return 0
    fi
    
    # Ensure image exists
    if ! docker images | grep -q "$SPLINK_IMAGE"; then
        echo "Docker image not found. Running install first..."
        install_dependencies
    fi
    
    # Start container
    docker run -d \
        --name "$SPLINK_CONTAINER" \
        --restart unless-stopped \
        -p "${SPLINK_PORT}:8096" \
        -v "${DATA_DIR}:/data" \
        -v "${RESOURCE_DIR}/api:/app/api" \
        -v "${RESOURCE_DIR}/lib:/app/lib" \
        -e SPLINK_PORT="${SPLINK_PORT}" \
        -e SPLINK_BACKEND="${SPLINK_BACKEND}" \
        -e POSTGRES_HOST="${POSTGRES_HOST:-localhost}" \
        -e POSTGRES_PORT="${POSTGRES_PORT:-5433}" \
        -e REDIS_HOST="${REDIS_HOST:-localhost}" \
        -e REDIS_PORT="${REDIS_PORT:-6380}" \
        -e MINIO_HOST="${MINIO_HOST:-localhost}" \
        -e MINIO_PORT="${MINIO_PORT:-9000}" \
        --network "${DOCKER_NETWORK:-bridge}" \
        "$SPLINK_IMAGE" &>> "$LOG_FILE" || {
            echo "Error: Failed to start Splink container"
            return 1
        }
    
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "Waiting for Splink to be ready..."
        wait_for_health
    fi
    
    echo "Splink started successfully"
}

# Stop the service
stop_service() {
    echo "Stopping Splink service..."
    
    if docker ps -q -f name="$SPLINK_CONTAINER" | grep -q .; then
        docker stop "$SPLINK_CONTAINER" &>> "$LOG_FILE"
        docker rm "$SPLINK_CONTAINER" &>> "$LOG_FILE"
        echo "Splink stopped successfully"
    else
        echo "Splink is not running"
    fi
}

# Restart the service
restart_service() {
    echo "Restarting Splink service..."
    stop_service
    sleep 2
    start_service "$@"
}

# Uninstall the service
uninstall_service() {
    echo "Uninstalling Splink..."
    
    # Stop service if running
    stop_service
    
    # Remove Docker image
    if docker images | grep -q "$SPLINK_IMAGE"; then
        docker rmi "$SPLINK_IMAGE" &>> "$LOG_FILE"
    fi
    
    # Optionally remove data (prompt user)
    read -p "Remove Splink data directory? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$DATA_DIR"
        echo "Data directory removed"
    fi
    
    echo "Splink uninstalled successfully"
}

# Check service health
check_service_health() {
    timeout 5 curl -sf "http://localhost:${SPLINK_PORT}/health" &> /dev/null
}

# Wait for service to be healthy
wait_for_health() {
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if check_service_health; then
            echo "Splink is ready"
            return 0
        fi
        
        echo "Waiting for Splink to start... (attempt $((attempt + 1))/$max_attempts)"
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "Error: Splink failed to start within timeout"
    return 1
}

# Get service PID
get_service_pid() {
    docker inspect -f '{{.State.Pid}}' "$SPLINK_CONTAINER" 2>/dev/null || echo "N/A"
}

# Get service uptime
get_service_uptime() {
    if docker ps -f name="$SPLINK_CONTAINER" --format "{{.Status}}" | grep -q Up; then
        docker ps -f name="$SPLINK_CONTAINER" --format "{{.Status}}" | sed 's/Up //'
    else
        echo "N/A"
    fi
}

# Execute linkage operation
execute_linkage() {
    local operation="${1:-deduplicate}"
    shift || true
    
    if ! check_service_health; then
        echo "Error: Splink service is not running"
        return 1
    fi
    
    case "$operation" in
        deduplicate)
            execute_deduplication "$@"
            ;;
        link)
            execute_linking "$@"
            ;;
        estimate)
            execute_estimation "$@"
            ;;
        *)
            echo "Error: Unknown operation: $operation"
            echo "Available operations: deduplicate, link, estimate"
            return 1
            ;;
    esac
}

# Execute deduplication
execute_deduplication() {
    local dataset=""
    local settings="{}"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dataset)
                dataset="$2"
                shift 2
                ;;
            --settings)
                settings="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$dataset" ]]; then
        echo "Error: --dataset is required"
        return 1
    fi
    
    echo "Starting deduplication for dataset: $dataset"
    
    curl -X POST "http://localhost:${SPLINK_PORT}/linkage/deduplicate" \
        -H "Content-Type: application/json" \
        -d "{\"dataset_id\": \"$dataset\", \"settings\": $settings}" \
        2>/dev/null | jq '.' 2>/dev/null || echo "Response received"
}

# Execute linking
execute_linking() {
    local dataset1=""
    local dataset2=""
    local settings="{}"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dataset1)
                dataset1="$2"
                shift 2
                ;;
            --dataset2)
                dataset2="$2"
                shift 2
                ;;
            --settings)
                settings="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$dataset1" ]] || [[ -z "$dataset2" ]]; then
        echo "Error: --dataset1 and --dataset2 are required"
        return 1
    fi
    
    echo "Starting linkage between $dataset1 and $dataset2"
    
    curl -X POST "http://localhost:${SPLINK_PORT}/linkage/link" \
        -H "Content-Type: application/json" \
        -d "{\"dataset1_id\": \"$dataset1\", \"dataset2_id\": \"$dataset2\", \"settings\": $settings}" \
        2>/dev/null | jq '.' 2>/dev/null || echo "Response received"
}

# Execute parameter estimation
execute_estimation() {
    local dataset="${1:-}"
    
    if [[ -z "$dataset" ]]; then
        echo "Error: Dataset is required"
        return 1
    fi
    
    echo "Starting parameter estimation for dataset: $dataset"
    
    curl -X POST "http://localhost:${SPLINK_PORT}/linkage/estimate" \
        -H "Content-Type: application/json" \
        -d "{\"dataset_id\": \"$dataset\"}" \
        2>/dev/null | jq '.' 2>/dev/null || echo "Response received"
}

# List linkage jobs
list_jobs() {
    if ! check_service_health; then
        echo "Error: Splink service is not running"
        return 1
    fi
    
    curl -sf "http://localhost:${SPLINK_PORT}/linkage/jobs" 2>/dev/null | jq '.' 2>/dev/null || {
        echo "No jobs found or service unavailable"
    }
}

# Get linkage results
get_results() {
    local job_id="${1:-}"
    
    if [[ -z "$job_id" ]]; then
        echo "Error: Job ID is required"
        return 1
    fi
    
    if ! check_service_health; then
        echo "Error: Splink service is not running"
        return 1
    fi
    
    curl -sf "http://localhost:${SPLINK_PORT}/linkage/results/${job_id}" 2>/dev/null | jq '.' 2>/dev/null || {
        echo "Results not found for job: $job_id"
    }
}

# Remove linkage job
remove_job() {
    local job_id="${1:-}"
    
    if [[ -z "$job_id" ]]; then
        echo "Error: Job ID is required"
        return 1
    fi
    
    if ! check_service_health; then
        echo "Error: Splink service is not running"
        return 1
    fi
    
    curl -X DELETE "http://localhost:${SPLINK_PORT}/linkage/jobs/${job_id}" 2>/dev/null
    echo "Job $job_id removed"
}