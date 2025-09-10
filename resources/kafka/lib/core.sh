#!/bin/bash
# Apache Kafka Resource - Core Library Functions
# v2.0 Contract Implementation

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source configuration
source "$SCRIPT_DIR/config/defaults.sh"

# Install Kafka
install_kafka() {
    local force=""
    local skip_validation=""
    
    for arg in "$@"; do
        case "$arg" in
            --force)
                force="--force"
                ;;
            --skip-validation)
                skip_validation="--skip-validation"
                ;;
        esac
    done
    
    echo "Installing Apache Kafka resource..."
    
    # Check if already installed
    if docker ps -a --format '{{.Names}}' | grep -q "^${KAFKA_CONTAINER_NAME}$"; then
        echo "Kafka is already installed"
        return 2  # Already installed (skipped)
    fi
    
    # Validate configuration
    if ! validate_config; then
        echo "Configuration validation failed"
        return 1
    fi
    
    # Create Docker network if it doesn't exist
    if ! docker network ls | grep -q "$KAFKA_NETWORK"; then
        echo "Creating Docker network: $KAFKA_NETWORK"
        docker network create "$KAFKA_NETWORK" || return 1
    fi
    
    # Create volume for data persistence
    if ! docker volume ls | grep -q "$KAFKA_VOLUME_NAME"; then
        echo "Creating Docker volume: $KAFKA_VOLUME_NAME"
        docker volume create "$KAFKA_VOLUME_NAME" || return 1
    fi
    
    # Pull Kafka image
    echo "Pulling Kafka image: $KAFKA_IMAGE"
    docker pull "$KAFKA_IMAGE" || return 1
    
    if [ "$skip_validation" != "--skip-validation" ]; then
        echo "Validating installation..."
        # Verify image was pulled
        if ! docker images | grep -q "apache/kafka"; then
            echo "Error: Kafka image not found"
            return 1
        fi
    fi
    
    echo "Kafka installed successfully"
    return 0
}

# Start Kafka broker
start_kafka() {
    local wait_flag=""
    local timeout="${KAFKA_STARTUP_TIMEOUT}"
    
    for arg in "$@"; do
        case "$arg" in
            --wait)
                wait_flag="--wait"
                ;;
            --timeout)
                shift
                timeout="${1:-$KAFKA_STARTUP_TIMEOUT}"
                ;;
        esac
    done
    
    echo "Starting Kafka broker..."
    
    # Check if already running
    if docker ps --format '{{.Names}}' | grep -q "^${KAFKA_CONTAINER_NAME}$"; then
        echo "Kafka is already running"
        return 0
    fi
    
    # Remove any existing stopped container
    if docker ps -a --format '{{.Names}}' | grep -q "^${KAFKA_CONTAINER_NAME}$"; then
        echo "Removing stopped container..."
        docker rm "$KAFKA_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Generate cluster ID for KRaft mode
    CLUSTER_ID=$(docker run --rm "$KAFKA_IMAGE" /opt/kafka/bin/kafka-storage.sh random-uuid)
    echo "Generated Cluster ID: $CLUSTER_ID"
    
    # Start Kafka container
    docker run -d \
        --name "$KAFKA_CONTAINER_NAME" \
        --network "$KAFKA_NETWORK" \
        -p "${KAFKA_PORT}:${KAFKA_PORT}" \
        -p "${KAFKA_CONTROLLER_PORT}:${KAFKA_CONTROLLER_PORT}" \
        -p "${KAFKA_EXTERNAL_PORT}:${KAFKA_EXTERNAL_PORT}" \
        -p "${KAFKA_JMX_PORT}:${KAFKA_JMX_PORT}" \
        -v "${KAFKA_VOLUME_NAME}:${KAFKA_DATA_DIR}" \
        -e KAFKA_NODE_ID="$KAFKA_NODE_ID" \
        -e KAFKA_PROCESS_ROLES="$KAFKA_PROCESS_ROLES" \
        -e KAFKA_LISTENERS="$KAFKA_LISTENERS" \
        -e KAFKA_ADVERTISED_LISTENERS="$KAFKA_ADVERTISED_LISTENERS" \
        -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP="$KAFKA_LISTENER_SECURITY_PROTOCOL_MAP" \
        -e KAFKA_CONTROLLER_QUORUM_VOTERS="$KAFKA_CONTROLLER_QUORUM_VOTERS" \
        -e KAFKA_CONTROLLER_LISTENER_NAMES="$KAFKA_CONTROLLER_LISTENER_NAMES" \
        -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR="$KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR" \
        -e KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR="$KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR" \
        -e KAFKA_TRANSACTION_STATE_LOG_MIN_ISR="$KAFKA_TRANSACTION_STATE_LOG_MIN_ISR" \
        -e KAFKA_LOG_RETENTION_HOURS="$KAFKA_LOG_RETENTION_HOURS" \
        -e KAFKA_LOG_SEGMENT_BYTES="$KAFKA_LOG_SEGMENT_BYTES" \
        -e KAFKA_HEAP_OPTS="$KAFKA_HEAP_OPTS" \
        -e CLUSTER_ID="$CLUSTER_ID" \
        "$KAFKA_IMAGE" || return 1
    
    if [ "$wait_flag" == "--wait" ]; then
        echo "Waiting for Kafka to be ready (timeout: ${timeout}s)..."
        local elapsed=0
        while [ $elapsed -lt $timeout ]; do
            if check_health >/dev/null 2>&1; then
                echo "Kafka is ready!"
                return 0
            fi
            sleep 5
            elapsed=$((elapsed + 5))
            echo "Waiting... ($elapsed/$timeout seconds)"
        done
        echo "Error: Kafka failed to start within $timeout seconds"
        return 1
    fi
    
    echo "Kafka started (use --wait to wait for readiness)"
    return 0
}

# Stop Kafka broker
stop_kafka() {
    echo "Stopping Kafka broker..."
    
    if ! docker ps --format '{{.Names}}' | grep -q "^${KAFKA_CONTAINER_NAME}$"; then
        echo "Kafka is not running"
        return 2
    fi
    
    # Graceful shutdown
    docker stop --time="$KAFKA_SHUTDOWN_TIMEOUT" "$KAFKA_CONTAINER_NAME" || return 1
    docker rm "$KAFKA_CONTAINER_NAME" >/dev/null 2>&1
    
    echo "Kafka stopped successfully"
    return 0
}

# Restart Kafka broker
restart_kafka() {
    echo "Restarting Kafka broker..."
    stop_kafka || true
    sleep 2
    start_kafka "$@"
}

# Uninstall Kafka
uninstall_kafka() {
    local force=""
    local keep_data=""
    
    for arg in "$@"; do
        case "$arg" in
            --force)
                force="--force"
                ;;
            --keep-data)
                keep_data="--keep-data"
                ;;
        esac
    done
    
    if [ "$force" != "--force" ]; then
        read -p "Are you sure you want to uninstall Kafka? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Uninstall cancelled"
            return 1
        fi
    fi
    
    echo "Uninstalling Kafka..."
    
    # Stop and remove container
    stop_kafka || true
    
    # Remove volume if not keeping data
    if [ "$keep_data" != "--keep-data" ]; then
        echo "Removing data volume..."
        docker volume rm "$KAFKA_VOLUME_NAME" 2>/dev/null || true
    fi
    
    echo "Kafka uninstalled successfully"
    return 0
}

# Check Kafka health
check_health() {
    timeout "$KAFKA_HEALTH_CHECK_TIMEOUT" docker exec "$KAFKA_CONTAINER_NAME" \
        /opt/kafka/bin/kafka-broker-api-versions.sh \
        --bootstrap-server "localhost:${KAFKA_PORT}" >/dev/null 2>&1
}

# Show detailed status
show_status() {
    local json_flag=""
    
    for arg in "$@"; do
        case "$arg" in
            --json)
                json_flag="--json"
                ;;
        esac
    done
    
    if [ "$json_flag" == "--json" ]; then
        # JSON output
        local status="stopped"
        local health="unhealthy"
        
        if docker ps --format '{{.Names}}' | grep -q "^${KAFKA_CONTAINER_NAME}$"; then
            status="running"
            if check_health; then
                health="healthy"
            fi
        fi
        
        cat << EOF
{
  "resource": "kafka",
  "status": "$status",
  "health": "$health",
  "container": "$KAFKA_CONTAINER_NAME",
  "ports": {
    "broker": $KAFKA_PORT,
    "controller": $KAFKA_CONTROLLER_PORT,
    "external": $KAFKA_EXTERNAL_PORT
  }
}
EOF
    else
        # Human-readable output
        echo "=== Kafka Status ==="
        
        if docker ps --format '{{.Names}}' | grep -q "^${KAFKA_CONTAINER_NAME}$"; then
            echo "Status: Running"
            
            if check_health; then
                echo "Health: Healthy"
            else
                echo "Health: Unhealthy (broker not responding)"
            fi
            
            echo ""
            echo "Container Info:"
            docker ps --filter "name=${KAFKA_CONTAINER_NAME}" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        else
            echo "Status: Stopped"
        fi
        
        echo ""
        echo "Configuration:"
        echo "  Broker Port: $KAFKA_PORT"
        echo "  Controller Port: $KAFKA_CONTROLLER_PORT"
        echo "  External Port: $KAFKA_EXTERNAL_PORT"
        echo "==================="
    fi
}

# Show logs
show_logs() {
    local tail_lines="50"
    
    for arg in "$@"; do
        case "$arg" in
            --tail)
                shift
                tail_lines="${1:-50}"
                ;;
        esac
    done
    
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${KAFKA_CONTAINER_NAME}$"; then
        echo "Error: Kafka container not found"
        return 1
    fi
    
    docker logs --tail "$tail_lines" "$KAFKA_CONTAINER_NAME"
}

# Show credentials
show_credentials() {
    echo "=== Kafka Connection Details ==="
    echo "Bootstrap Server: localhost:${KAFKA_PORT}"
    echo "External Access: localhost:${KAFKA_EXTERNAL_PORT}"
    echo ""
    echo "Producer Example:"
    echo "  kafka-console-producer.sh --bootstrap-server localhost:${KAFKA_PORT} --topic my-topic"
    echo ""
    echo "Consumer Example:"
    echo "  kafka-console-consumer.sh --bootstrap-server localhost:${KAFKA_PORT} --topic my-topic --from-beginning"
    echo ""
    echo "Client Configuration:"
    echo "  bootstrap.servers=localhost:${KAFKA_PORT}"
    echo "================================"
}

# Topic management functions

# Add topic
add_topic() {
    local topic_name="${1:-}"
    local partitions="3"
    local replication="1"
    
    if [ -z "$topic_name" ]; then
        echo "Error: Topic name required"
        echo "Usage: resource-kafka content add <topic-name> [--partitions N] [--replication N]"
        return 1
    fi
    
    shift
    for arg in "$@"; do
        case "$arg" in
            --partitions)
                shift
                partitions="${1:-3}"
                ;;
            --replication)
                shift
                replication="${1:-1}"
                ;;
        esac
    done
    
    echo "Creating topic: $topic_name (partitions=$partitions, replication=$replication)"
    
    docker exec "$KAFKA_CONTAINER_NAME" \
        /opt/kafka/bin/kafka-topics.sh \
        --create \
        --topic "$topic_name" \
        --partitions "$partitions" \
        --replication-factor "$replication" \
        --bootstrap-server "localhost:${KAFKA_PORT}"
}

# List topics
list_topics() {
    echo "=== Kafka Topics ==="
    docker exec "$KAFKA_CONTAINER_NAME" \
        /opt/kafka/bin/kafka-topics.sh \
        --list \
        --bootstrap-server "localhost:${KAFKA_PORT}"
}

# Describe topic
describe_topic() {
    local topic_name="${1:-}"
    
    if [ -z "$topic_name" ]; then
        echo "Error: Topic name required"
        echo "Usage: resource-kafka content get <topic-name>"
        return 1
    fi
    
    docker exec "$KAFKA_CONTAINER_NAME" \
        /opt/kafka/bin/kafka-topics.sh \
        --describe \
        --topic "$topic_name" \
        --bootstrap-server "localhost:${KAFKA_PORT}"
}

# Remove topic
remove_topic() {
    local topic_name="${1:-}"
    
    if [ -z "$topic_name" ]; then
        echo "Error: Topic name required"
        echo "Usage: resource-kafka content remove <topic-name>"
        return 1
    fi
    
    echo "Deleting topic: $topic_name"
    
    docker exec "$KAFKA_CONTAINER_NAME" \
        /opt/kafka/bin/kafka-topics.sh \
        --delete \
        --topic "$topic_name" \
        --bootstrap-server "localhost:${KAFKA_PORT}"
}

# Execute arbitrary Kafka command
execute_command() {
    local command="$*"
    
    if [ -z "$command" ]; then
        echo "Error: Command required"
        echo "Usage: resource-kafka content execute <kafka-command>"
        return 1
    fi
    
    echo "Executing: $command"
    docker exec "$KAFKA_CONTAINER_NAME" /opt/kafka/bin/$command
}