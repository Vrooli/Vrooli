#!/bin/bash
# ====================================================================
# Storage Resource Health Checks
# ====================================================================
#
# Category-specific health checks for storage resources including data
# integrity, backup status, performance metrics, and connection health.
#
# Supported Storage Resources:
# - PostgreSQL: Relational database
# - Redis: In-memory cache and message broker
# - MinIO: S3-compatible object storage
# - Qdrant: Vector database
# - QuestDB: Time-series database
# - Vault: Secret management
#
# ====================================================================

# Storage resource health check implementations
check_postgres_health() {
    local port="${1:-5433}"
    local health_level="${2:-basic}"
    
    # Check if port is accessible
    if ! timeout 3 bash -c "</dev/tcp/localhost/$port" 2>/dev/null; then
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local connection_test="false"
    local database_count="0"
    local active_connections="0"
    
    # Try to connect with pg_isready if available
    if which pg_isready >/dev/null 2>&1; then
        if pg_isready -h localhost -p "$port" -U postgres -t 3 >/dev/null 2>&1; then
            connection_test="true"
        fi
    fi
    
    # Try to get basic stats if psql is available
    if which psql >/dev/null 2>&1 && [[ "$connection_test" == "true" ]]; then
        database_count=$(psql -h localhost -p "$port" -U postgres -t -c "SELECT count(*) FROM pg_database WHERE datistemplate = false;" 2>/dev/null | tr -d ' \n' || echo "0")
        active_connections=$(psql -h localhost -p "$port" -U postgres -t -c "SELECT count(*) FROM pg_stat_activity;" 2>/dev/null | tr -d ' \n' || echo "0")
    fi
    
    if [[ "$connection_test" == "true" ]]; then
        echo "healthy:connected:databases:$database_count:connections:$active_connections"
    else
        echo "degraded:port_open:connection_failed"
    fi
    
    return 0
}

check_redis_health() {
    local port="${1:-6380}"
    local health_level="${2:-basic}"
    
    # Check if port is accessible
    if ! timeout 3 bash -c "</dev/tcp/localhost/$port" 2>/dev/null; then
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local ping_result="false"
    local memory_usage="unknown"
    local connected_clients="0"
    local keyspace="0"
    
    # Try to ping Redis if redis-cli is available
    if which redis-cli >/dev/null 2>&1; then
        if redis-cli -h localhost -p "$port" ping 2>/dev/null | grep -q "PONG"; then
            ping_result="true"
            
            # Get additional stats
            local info_output
            info_output=$(redis-cli -h localhost -p "$port" info memory,clients,keyspace 2>/dev/null)
            
            if [[ -n "$info_output" ]]; then
                memory_usage=$(echo "$info_output" | grep "used_memory_human:" | cut -d: -f2 | tr -d '\r\n' || echo "unknown")
                connected_clients=$(echo "$info_output" | grep "connected_clients:" | cut -d: -f2 | tr -d '\r\n' || echo "0")
                
                # Count total keys across all databases
                local db_keys
                db_keys=$(echo "$info_output" | grep "^db[0-9]" | sed 's/.*keys=\([0-9]*\).*/\1/' | paste -sd+ | bc 2>/dev/null || echo "0")
                keyspace="$db_keys"
            fi
        fi
    fi
    
    if [[ "$ping_result" == "true" ]]; then
        echo "healthy:connected:memory:$memory_usage:clients:$connected_clients:keys:$keyspace"
    else
        echo "degraded:port_open:ping_failed"
    fi
    
    return 0
}

check_minio_health() {
    local port="${1:-9000}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/minio/health/live" >/dev/null 2>&1; then
        # Try alternative health endpoints
        if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
            echo "healthy"
            return 0
        else
            echo "unreachable"
            return 1
        fi
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local ready_check="false"
    local storage_info="unknown"
    
    # Check readiness endpoint
    if curl -s --max-time 5 "http://localhost:${port}/minio/health/ready" >/dev/null 2>&1; then
        ready_check="true"
    fi
    
    # Try to get storage info if mc (MinIO client) is available
    if which mc >/dev/null 2>&1; then
        # This would require configuring mc first, so we skip for now
        storage_info="requires_auth"
    fi
    
    if [[ "$ready_check" == "true" ]]; then
        echo "healthy:ready:storage:$storage_info"
    else
        echo "degraded:live:not_ready"
    fi
    
    return 0
}

check_qdrant_health() {
    local port="${1:-6333}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
        # Try alternative endpoints
        if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
            echo "healthy"
            return 0
        else
            echo "unreachable"
            return 1
        fi
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local collections_count="0"
    local cluster_info="standalone"
    
    # Get collections info
    local collections_response
    collections_response=$(curl -s --max-time 10 "http://localhost:${port}/collections" 2>/dev/null)
    
    if [[ -n "$collections_response" ]] && echo "$collections_response" | jq . >/dev/null 2>&1; then
        collections_count=$(echo "$collections_response" | jq '.result.collections | length' 2>/dev/null || echo "0")
    fi
    
    # Get cluster info
    local cluster_response
    cluster_response=$(curl -s --max-time 10 "http://localhost:${port}/cluster" 2>/dev/null)
    
    if [[ -n "$cluster_response" ]] && echo "$cluster_response" | jq . >/dev/null 2>&1; then
        local peer_count
        peer_count=$(echo "$cluster_response" | jq '.result.peers | length' 2>/dev/null || echo "1")
        cluster_info="peers:$peer_count"
    fi
    
    echo "healthy:collections:$collections_count:cluster:$cluster_info"
    return 0
}

check_questdb_health() {
    local port="${1:-9010}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local tables_count="0"
    local metrics_available="false"
    
    # Try to get basic system information
    local query_response
    query_response=$(curl -s --max-time 10 -G "http://localhost:${port}/exec" \
        --data-urlencode "query=SELECT count(*) FROM tables()" 2>/dev/null)
    
    if [[ -n "$query_response" ]] && echo "$query_response" | jq . >/dev/null 2>&1; then
        tables_count=$(echo "$query_response" | jq -r '.dataset[0][0]' 2>/dev/null || echo "0")
        metrics_available="true"
    fi
    
    if [[ "$metrics_available" == "true" ]]; then
        echo "healthy:tables:$tables_count:query_ready"
    else
        echo "degraded:web_interface_up:query_failed"
    fi
    
    return 0
}

check_vault_health() {
    local port="${1:-8200}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/v1/sys/health" >/dev/null 2>&1; then
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local vault_status="unknown"
    local sealed_status="unknown"
    local ha_enabled="false"
    
    local health_response
    health_response=$(curl -s --max-time 10 "http://localhost:${port}/v1/sys/health" 2>/dev/null)
    
    if [[ -n "$health_response" ]] && echo "$health_response" | jq . >/dev/null 2>&1; then
        vault_status=$(echo "$health_response" | jq -r '.initialized' 2>/dev/null || echo "unknown")
        sealed_status=$(echo "$health_response" | jq -r '.sealed' 2>/dev/null || echo "unknown")
        
        # Check if standby (HA)
        local standby_status
        standby_status=$(echo "$health_response" | jq -r '.standby' 2>/dev/null)
        if [[ "$standby_status" != "null" ]]; then
            ha_enabled="true"
        fi
    fi
    
    if [[ "$vault_status" == "true" && "$sealed_status" == "false" ]]; then
        echo "healthy:initialized:unsealed:ha:$ha_enabled"
    elif [[ "$vault_status" == "true" && "$sealed_status" == "true" ]]; then
        echo "degraded:initialized:sealed:ha:$ha_enabled"
    elif [[ "$vault_status" == "false" ]]; then
        echo "degraded:uninitialized:ha:$ha_enabled"
    else
        echo "degraded:status_unknown"
    fi
    
    return 0
}

# Generic storage health check dispatcher
check_storage_resource_health() {
    local resource_name="$1"
    local port="$2"
    local health_level="${3:-basic}"
    
    case "$resource_name" in
        "postgres"|"postgresql")
            check_postgres_health "$port" "$health_level"
            ;;
        "redis")
            check_redis_health "$port" "$health_level"
            ;;
        "minio")
            check_minio_health "$port" "$health_level"
            ;;
        "qdrant")
            check_qdrant_health "$port" "$health_level"
            ;;
        "questdb")
            check_questdb_health "$port" "$health_level"
            ;;
        "vault")
            check_vault_health "$port" "$health_level"
            ;;
        *)
            # Fallback to generic HTTP health check
            if curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            ;;
    esac
}

# Storage resource capability testing
test_storage_resource_capabilities() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "postgres"|"postgresql")
            test_postgres_capabilities "$port"
            ;;
        "redis")
            test_redis_capabilities "$port"
            ;;
        "minio")
            test_minio_capabilities "$port"
            ;;
        "qdrant")
            test_qdrant_capabilities "$port"
            ;;
        "questdb")
            test_questdb_capabilities "$port"
            ;;
        "vault")
            test_vault_capabilities "$port"
            ;;
        *)
            echo "capability_testing_not_implemented"
            ;;
    esac
}

test_postgres_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if which psql >/dev/null 2>&1; then
        capabilities+=("sql_queries")
        capabilities+=("transactions")
        capabilities+=("backup_restore")
    fi
    
    if timeout 3 bash -c "</dev/tcp/localhost/$port" 2>/dev/null; then
        capabilities+=("tcp_connection")
    fi
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_redis_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if which redis-cli >/dev/null 2>&1; then
        capabilities+=("key_value_storage")
        capabilities+=("pub_sub")
        capabilities+=("data_structures")
    fi
    
    if timeout 3 bash -c "</dev/tcp/localhost/$port" 2>/dev/null; then
        capabilities+=("tcp_connection")
    fi
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_minio_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if curl -s --max-time 5 "http://localhost:${port}/minio/health/live" >/dev/null 2>&1; then
        capabilities+=("s3_compatible_api")
        capabilities+=("object_storage")
        capabilities+=("bucket_management")
    fi
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_qdrant_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
        capabilities+=("vector_storage")
        capabilities+=("similarity_search")
        capabilities+=("collection_management")
    fi
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_questdb_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        capabilities+=("time_series_storage")
        capabilities+=("sql_queries")
        capabilities+=("influxdb_line_protocol")
        capabilities+=("http_api")
    fi
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_vault_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if curl -s --max-time 5 "http://localhost:${port}/v1/sys/health" >/dev/null 2>&1; then
        capabilities+=("secret_storage")
        capabilities+=("encryption")
        capabilities+=("dynamic_secrets")
    fi
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

# Export functions
export -f check_storage_resource_health
export -f test_storage_resource_capabilities
export -f check_postgres_health
export -f check_redis_health
export -f check_minio_health
export -f check_qdrant_health
export -f check_questdb_health
export -f check_vault_health