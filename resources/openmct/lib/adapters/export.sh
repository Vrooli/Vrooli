#!/usr/bin/env bash
# Export Adapter for Open MCT - Postgres/Qdrant Integration

set -euo pipefail

# Export Configuration
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5433}"
POSTGRES_DB="${POSTGRES_DB:-vrooli}"
POSTGRES_USER="${POSTGRES_USER:-vrooli}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"

QDRANT_HOST="${QDRANT_HOST:-localhost}"
QDRANT_PORT="${QDRANT_PORT:-6333}"
QDRANT_COLLECTION="${QDRANT_COLLECTION:-telemetry_anomalies}"

# Export telemetry to PostgreSQL
export_to_postgres() {
    local stream="${1:-}"
    local start_time="${2:-}"
    local end_time="${3:-}"
    
    if [[ -z "$stream" ]]; then
        echo "Usage: export_to_postgres <stream> [start_time] [end_time]"
        exit 1
    fi
    
    echo "Exporting stream '$stream' to PostgreSQL..."
    
    # Create table if not exists
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" << EOF
CREATE TABLE IF NOT EXISTS telemetry_${stream} (
    id SERIAL PRIMARY KEY,
    timestamp BIGINT NOT NULL,
    value REAL,
    data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_telemetry_${stream}_timestamp ON telemetry_${stream}(timestamp);
EOF
    
    # Export data from SQLite to PostgreSQL
    local query="SELECT timestamp, value, data FROM telemetry WHERE stream = '$stream'"
    if [[ -n "$start_time" ]]; then
        query="$query AND timestamp >= $start_time"
    fi
    if [[ -n "$end_time" ]]; then
        query="$query AND timestamp <= $end_time"
    fi
    
    sqlite3 "${OPENMCT_DATA_DIR}/telemetry.db" "$query" | while IFS='|' read -r timestamp value data; do
        PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
            -c "INSERT INTO telemetry_${stream} (timestamp, value, data) VALUES ($timestamp, $value, '$data')" \
            2>/dev/null || echo "Failed to insert record"
    done
    
    # Get count
    count=$(PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
        -t -c "SELECT COUNT(*) FROM telemetry_${stream}")
    
    echo "✓ Exported to PostgreSQL table: telemetry_${stream}"
    echo "  Total records: $count"
}

# Export anomalies to Qdrant
export_to_qdrant() {
    local stream="${1:-}"
    local threshold="${2:-}"
    
    if [[ -z "$stream" || -z "$threshold" ]]; then
        echo "Usage: export_to_qdrant <stream> <threshold>"
        echo "  Exports data points where value exceeds threshold as anomalies"
        exit 1
    fi
    
    echo "Exporting anomalies from stream '$stream' to Qdrant..."
    echo "Threshold: values > $threshold"
    
    # Create collection if not exists
    curl -X PUT "http://${QDRANT_HOST}:${QDRANT_PORT}/collections/${QDRANT_COLLECTION}" \
        -H "Content-Type: application/json" \
        -d '{
            "vectors": {
                "size": 4,
                "distance": "Cosine"
            }
        }' 2>/dev/null || true
    
    # Find anomalies
    local anomalies=$(sqlite3 "${OPENMCT_DATA_DIR}/telemetry.db" \
        "SELECT timestamp, value, data FROM telemetry WHERE stream = '$stream' AND value > $threshold")
    
    local count=0
    echo "$anomalies" | while IFS='|' read -r timestamp value data; do
        if [[ -n "$timestamp" ]]; then
            # Create vector from telemetry data (simplified - could use actual feature extraction)
            vector="[$value, $(echo "$value" | awk '{print sin($1)}'), $(echo "$value" | awk '{print cos($1)}'), $timestamp]"
            
            # Create point
            point=$(cat <<EOF
{
    "id": "$count",
    "vector": $vector,
    "payload": {
        "stream": "$stream",
        "timestamp": $timestamp,
        "value": $value,
        "data": $data,
        "anomaly_type": "threshold_exceeded"
    }
}
EOF
            )
            
            # Insert into Qdrant
            curl -X PUT "http://${QDRANT_HOST}:${QDRANT_PORT}/collections/${QDRANT_COLLECTION}/points" \
                -H "Content-Type: application/json" \
                -d "{\"points\": [$point]}" \
                2>/dev/null || echo "Failed to insert anomaly"
            
            count=$((count + 1))
        fi
    done
    
    echo "✓ Exported $count anomalies to Qdrant collection: $QDRANT_COLLECTION"
}

# Schedule automatic exports
schedule_export() {
    local interval="${1:-3600}"  # Default 1 hour
    local target="${2:-postgres}"  # postgres or qdrant
    
    echo "Scheduling $target export every ${interval} seconds..."
    
    # Create export script
    cat > "${OPENMCT_CONFIG_DIR}/export_schedule.sh" << EOF
#!/usr/bin/env bash
while true; do
    echo "Running scheduled export to $target..."
    
    # Get all streams
    streams=\$(sqlite3 "${OPENMCT_DATA_DIR}/telemetry.db" "SELECT DISTINCT stream FROM telemetry")
    
    for stream in \$streams; do
        if [[ "$target" == "postgres" ]]; then
            ${BASH_SOURCE[0]} export-postgres "\$stream"
        elif [[ "$target" == "qdrant" ]]; then
            # Use standard deviation as threshold (simplified)
            avg=\$(sqlite3 "${OPENMCT_DATA_DIR}/telemetry.db" \
                "SELECT AVG(value) FROM telemetry WHERE stream = '\$stream'")
            threshold=\$(echo "\$avg * 1.5" | bc)
            ${BASH_SOURCE[0]} export-qdrant "\$stream" "\$threshold"
        fi
    done
    
    sleep $interval
done
EOF
    
    chmod +x "${OPENMCT_CONFIG_DIR}/export_schedule.sh"
    
    # Run in background
    nohup "${OPENMCT_CONFIG_DIR}/export_schedule.sh" > "${OPENMCT_DATA_DIR}/export.log" 2>&1 &
    echo $! > "${OPENMCT_CONFIG_DIR}/export_schedule.pid"
    
    echo "✓ Export scheduled (PID: $(cat "${OPENMCT_CONFIG_DIR}/export_schedule.pid"))"
}

# Stop scheduled exports
stop_schedule() {
    if [[ -f "${OPENMCT_CONFIG_DIR}/export_schedule.pid" ]]; then
        pid=$(cat "${OPENMCT_CONFIG_DIR}/export_schedule.pid")
        kill "$pid" 2>/dev/null && echo "✓ Stopped scheduled export (PID: $pid)"
        rm "${OPENMCT_CONFIG_DIR}/export_schedule.pid"
    else
        echo "No scheduled export found"
    fi
}

# Test connections
test_connections() {
    echo "Testing export connections..."
    echo ""
    
    # Test PostgreSQL
    echo -n "PostgreSQL ($POSTGRES_HOST:$POSTGRES_PORT): "
    if PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
        -c "SELECT 1" &>/dev/null; then
        echo "✓ Connected"
    else
        echo "✗ Failed"
    fi
    
    # Test Qdrant
    echo -n "Qdrant ($QDRANT_HOST:$QDRANT_PORT): "
    if curl -s "http://${QDRANT_HOST}:${QDRANT_PORT}/collections" &>/dev/null; then
        echo "✓ Connected"
    else
        echo "✗ Failed"
    fi
}

# Main command handler
case "${1:-}" in
    export-postgres)
        shift
        export_to_postgres "$@"
        ;;
    export-qdrant)
        shift
        export_to_qdrant "$@"
        ;;
    schedule)
        shift
        schedule_export "$@"
        ;;
    stop-schedule)
        stop_schedule
        ;;
    test)
        test_connections
        ;;
    *)
        echo "Export Adapter for Open MCT"
        echo "============================"
        echo ""
        echo "Commands:"
        echo "  export-postgres <stream> [start] [end] - Export to PostgreSQL"
        echo "  export-qdrant <stream> <threshold>     - Export anomalies to Qdrant"
        echo "  schedule [interval] [target]           - Schedule automatic exports"
        echo "  stop-schedule                          - Stop scheduled exports"
        echo "  test                                   - Test connections"
        echo ""
        echo "Environment Variables:"
        echo "  POSTGRES_HOST     - PostgreSQL host (default: localhost)"
        echo "  POSTGRES_PORT     - PostgreSQL port (default: 5433)"
        echo "  POSTGRES_DB       - PostgreSQL database (default: vrooli)"
        echo "  POSTGRES_USER     - PostgreSQL user (default: vrooli)"
        echo "  POSTGRES_PASSWORD - PostgreSQL password"
        echo "  QDRANT_HOST      - Qdrant host (default: localhost)"
        echo "  QDRANT_PORT      - Qdrant port (default: 6333)"
        echo "  QDRANT_COLLECTION - Qdrant collection (default: telemetry_anomalies)"
        ;;
esac