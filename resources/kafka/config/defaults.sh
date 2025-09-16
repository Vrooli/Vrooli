#!/bin/bash
# Apache Kafka Resource - Default Configuration
# v2.0 Contract Compliant

# Resource identification
export KAFKA_RESOURCE_NAME="kafka"
export KAFKA_CONTAINER_NAME="kafka-broker"

# Port configuration (from port registry)
export KAFKA_PORT="${KAFKA_PORT:-29092}"
export KAFKA_CONTROLLER_PORT="${KAFKA_CONTROLLER_PORT:-29093}"
export KAFKA_EXTERNAL_PORT="${KAFKA_EXTERNAL_PORT:-29094}"
export KAFKA_JMX_PORT="${KAFKA_JMX_PORT:-29099}"

# Docker configuration
export KAFKA_IMAGE="${KAFKA_IMAGE:-apache/kafka:latest}"
export KAFKA_NETWORK="${KAFKA_NETWORK:-vrooli-network}"

# Kafka configuration
export KAFKA_NODE_ID="${KAFKA_NODE_ID:-1}"
export KAFKA_PROCESS_ROLES="${KAFKA_PROCESS_ROLES:-broker,controller}"
export KAFKA_LOG_LEVEL="${KAFKA_LOG_LEVEL:-INFO}"

# Storage configuration
export KAFKA_DATA_DIR="${KAFKA_DATA_DIR:-/var/lib/kafka/data}"
export KAFKA_LOG_DIR="${KAFKA_LOG_DIR:-/var/lib/kafka/logs}"
export KAFKA_VOLUME_NAME="${KAFKA_VOLUME_NAME:-kafka-data}"

# Memory configuration
export KAFKA_HEAP_OPTS="${KAFKA_HEAP_OPTS:--Xmx1G -Xms1G}"

# Retention configuration
export KAFKA_LOG_RETENTION_HOURS="${KAFKA_LOG_RETENTION_HOURS:-168}"  # 7 days
export KAFKA_LOG_SEGMENT_BYTES="${KAFKA_LOG_SEGMENT_BYTES:-1073741824}"  # 1GB

# Replication configuration (for single node)
export KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR="${KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR:-1}"
export KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR="${KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR:-1}"
export KAFKA_TRANSACTION_STATE_LOG_MIN_ISR="${KAFKA_TRANSACTION_STATE_LOG_MIN_ISR:-1}"

# Timeout configuration
export KAFKA_STARTUP_TIMEOUT="${KAFKA_STARTUP_TIMEOUT:-120}"
export KAFKA_SHUTDOWN_TIMEOUT="${KAFKA_SHUTDOWN_TIMEOUT:-30}"
export KAFKA_HEALTH_CHECK_TIMEOUT="${KAFKA_HEALTH_CHECK_TIMEOUT:-5}"

# Test configuration
export KAFKA_TEST_TOPIC="${KAFKA_TEST_TOPIC:-test-topic}"
export KAFKA_TEST_MESSAGE="${KAFKA_TEST_MESSAGE:-Hello Kafka}"

# Listener configuration
export KAFKA_LISTENERS="PLAINTEXT://:${KAFKA_PORT},CONTROLLER://:${KAFKA_CONTROLLER_PORT},EXTERNAL://:${KAFKA_EXTERNAL_PORT}"
export KAFKA_ADVERTISED_LISTENERS="PLAINTEXT://localhost:${KAFKA_PORT},EXTERNAL://localhost:${KAFKA_EXTERNAL_PORT}"
export KAFKA_LISTENER_SECURITY_PROTOCOL_MAP="CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT,EXTERNAL:PLAINTEXT"
export KAFKA_CONTROLLER_QUORUM_VOTERS="${KAFKA_NODE_ID}@localhost:${KAFKA_CONTROLLER_PORT}"
export KAFKA_CONTROLLER_LISTENER_NAMES="CONTROLLER"

# Security configuration (disabled by default)
export KAFKA_SECURITY_ENABLED="${KAFKA_SECURITY_ENABLED:-false}"
export KAFKA_SSL_ENABLED="${KAFKA_SSL_ENABLED:-false}"
export KAFKA_SASL_ENABLED="${KAFKA_SASL_ENABLED:-false}"

# Logging configuration
export KAFKA_LOG_FILE="${KAFKA_LOG_FILE:-/tmp/kafka-resource.log}"
export KAFKA_DEBUG="${KAFKA_DEBUG:-false}"

# Health check configuration
export KAFKA_HEALTH_CHECK_INTERVAL="${KAFKA_HEALTH_CHECK_INTERVAL:-30}"
export KAFKA_HEALTH_CHECK_RETRIES="${KAFKA_HEALTH_CHECK_RETRIES:-3}"

# Function to validate configuration
validate_config() {
    # Check if ports are available using nc (netcat) which doesn't require sudo
    for port in $KAFKA_PORT $KAFKA_CONTROLLER_PORT $KAFKA_EXTERNAL_PORT; do
        # Use timeout with nc to check if port is in use
        if timeout 1 bash -c "echo >/dev/tcp/localhost/$port" 2>/dev/null; then
            echo "Error: Port $port is already in use"
            return 1
        fi
    done
    
    # Check Docker availability
    if ! command -v docker &> /dev/null; then
        echo "Error: Docker is not installed"
        return 1
    fi
    
    return 0
}

# Function to display configuration
show_config() {
    echo "=== Kafka Configuration ==="
    echo "Container: $KAFKA_CONTAINER_NAME"
    echo "Image: $KAFKA_IMAGE"
    echo "Broker Port: $KAFKA_PORT"
    echo "Controller Port: $KAFKA_CONTROLLER_PORT"
    echo "External Port: $KAFKA_EXTERNAL_PORT"
    echo "JMX Port: $KAFKA_JMX_PORT"
    echo "Data Directory: $KAFKA_DATA_DIR"
    echo "Log Retention: $KAFKA_LOG_RETENTION_HOURS hours"
    echo "Heap Memory: $KAFKA_HEAP_OPTS"
    echo "=========================="
}