#!/usr/bin/env bats
# Data Pipeline Integration Test Template
# Tests data processing pipelines involving storage, processing, and analytics resources
#
# Copy this template and customize it for your specific data pipeline needs.
# Example: Real-time analytics pipeline with streaming data

#######################################
# TEST CONFIGURATION
# Customize these variables for your pipeline
#######################################

# Define the pipeline resources
PIPELINE_RESOURCES=("questdb" "redis" "n8n" "ollama")
PIPELINE_NAME="realtime_analytics"

# Data flow stages
DATA_STAGES=(
    "data_ingestion"
    "data_validation"
    "real_time_processing" 
    "batch_analytics"
    "result_storage"
)

# Pipeline configuration
MAX_PROCESSING_TIME=120    # 2 minutes
BATCH_SIZE=100
STREAMING_INTERVAL=5       # seconds
DATA_RETENTION_DAYS=7

#######################################
# SETUP AND TEARDOWN
#######################################

# Load the test infrastructure
if [[ -n "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
    source "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"
else
    TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${TEST_DIR}/../../core/common_setup.bash"
fi

setup() {
    # Setup integration test environment
    setup_integration_test "${PIPELINE_RESOURCES[@]}"
    
    # Configure pipeline-specific environment
    export PIPELINE_ID="pipeline_${PIPELINE_NAME}_$$"
    export PIPELINE_DATA_DIR="$TEST_TMPDIR/pipeline_data"
    export PIPELINE_CONFIG_FILE="$TEST_TMPDIR/pipeline_config.json"
    export PIPELINE_SCHEMA_FILE="$TEST_TMPDIR/data_schema.sql"
    
    # Create pipeline directories
    mkdir -p "$PIPELINE_DATA_DIR"/{raw,processed,analytics,exports}
    
    # Generate test data schema
    cat > "$PIPELINE_SCHEMA_FILE" << 'EOF'
-- Real-time metrics table
CREATE TABLE realtime_metrics (
    timestamp TIMESTAMP,
    metric_name STRING,
    metric_value DOUBLE,
    source_system STRING,
    tags STRING
) timestamp (timestamp) PARTITION BY DAY;

-- Aggregated analytics table  
CREATE TABLE analytics_summary (
    date DATE,
    hour INT,
    metric_name STRING,
    avg_value DOUBLE,
    min_value DOUBLE,
    max_value DOUBLE,
    count LONG
) timestamp (date);

-- Data quality metrics
CREATE TABLE data_quality (
    timestamp TIMESTAMP,
    table_name STRING,
    quality_score DOUBLE,
    issues_detected INT,
    records_processed LONG
) timestamp (timestamp);
EOF

    # Generate pipeline configuration
    cat > "$PIPELINE_CONFIG_FILE" << EOF
{
    "pipeline_id": "$PIPELINE_ID",
    "name": "$PIPELINE_NAME",
    "version": "1.0.0",
    "resources": $(printf '%s\n' "${PIPELINE_RESOURCES[@]}" | jq -R . | jq -s .),
    "stages": $(printf '%s\n' "${DATA_STAGES[@]}" | jq -R . | jq -s .),
    "config": {
        "batch_size": $BATCH_SIZE,
        "streaming_interval": $STREAMING_INTERVAL,
        "retention_days": $DATA_RETENTION_DAYS,
        "processing_timeout": $MAX_PROCESSING_TIME
    },
    "data_sources": [
        "sensor_data",
        "user_events", 
        "system_metrics",
        "external_apis"
    ]
}
EOF

    # Setup mock verification expectations
    mock::verify::expect_calls "docker" "run.*questdb" 1
    mock::verify::expect_calls "docker" "run.*redis" 1
    mock::verify::expect_calls "docker" "run.*n8n" 1
    mock::verify::expect_calls "http" ".*questdb.*exec.*" "10"  # Multiple SQL queries
    mock::verify::expect_calls "command" "redis-cli.*" "5"     # Redis operations
}

teardown() {
    # Validate all pipeline expectations
    mock::verify::validate_all
    
    # Clean up pipeline data
    rm -rf "$PIPELINE_DATA_DIR"
    rm -f "$PIPELINE_CONFIG_FILE" "$PIPELINE_SCHEMA_FILE"
    
    # Standard teardown
    teardown_test_environment
}

#######################################
# PIPELINE INFRASTRUCTURE TESTS
#######################################

@test "pipeline: data storage systems are ready" {
    # Test QuestDB is ready for time-series data
    run curl -s "$QUESTDB_BASE_URL/status"
    assert_success
    assert_json_field_equals "$output" ".status" "OK"
    
    # Test Redis is ready for caching and streaming
    run redis-cli -h localhost -p "$REDIS_PORT" ping
    assert_success
    assert_output "PONG"
    
    # Test N8N workflow engine is ready
    run curl -s "$N8N_BASE_URL/healthz"
    assert_success
    assert_json_field_equals "$output" ".status" "ok"
}

@test "pipeline: data schema initialization" {
    # Create tables in QuestDB
    while IFS= read -r sql_line; do
        if [[ "$sql_line" =~ ^CREATE[[:space:]]+TABLE ]]; then
            # Execute CREATE TABLE statement
            run curl -s -G "$QUESTDB_BASE_URL/exec" \
                --data-urlencode "query=$sql_line" \
                --data-urlencode "fmt=json"
            assert_success
            assert_json_field_not_contains "$output" "error"
        fi
    done < "$PIPELINE_SCHEMA_FILE"
    
    # Verify tables were created
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT table_name FROM tables() WHERE table_name LIKE '%metrics%' OR table_name LIKE '%analytics%' OR table_name LIKE '%quality%'" \
        --data-urlencode "fmt=json"
    assert_success
    
    # Should have created 3 tables
    local table_count
    table_count=$(echo "$output" | jq '.dataset | length')
    assert_equals "$table_count" "3"
}

@test "pipeline: cache and streaming setup" {
    # Setup Redis streams for real-time data
    run redis-cli -h localhost -p "$REDIS_PORT" XADD pipeline_events "*" event_type "pipeline_start" pipeline_id "$PIPELINE_ID"
    assert_success
    
    # Setup Redis keys for caching
    run redis-cli -h localhost -p "$REDIS_PORT" SET "pipeline:${PIPELINE_ID}:status" "initializing"
    assert_success
    assert_output "OK"
    
    # Setup Redis sets for tracking processed items
    run redis-cli -h localhost -p "$REDIS_PORT" SADD "pipeline:${PIPELINE_ID}:processed" "init_marker"
    assert_success
    assert_output "(integer) 1"
}

#######################################
# DATA INGESTION TESTS
#######################################

@test "pipeline: stage 1 - data ingestion" {
    local stage_start_time=$(date +%s)
    
    # Generate test sensor data
    local sensor_data_file="$PIPELINE_DATA_DIR/raw/sensor_data.jsonl"
    for i in {1..10}; do
        local timestamp=$(date -d "$i seconds ago" -Iseconds)
        echo '{"timestamp":"'"$timestamp"'","sensor_id":"temp_01","value":'"$((20 + i))"',"location":"datacenter"}' >> "$sensor_data_file"
    done
    
    # Verify data file creation
    assert_file_exists "$sensor_data_file"
    local line_count=$(wc -l < "$sensor_data_file")
    assert_equals "$line_count" "10"
    
    # Simulate data ingestion via Redis stream
    local data_ingested=0
    while IFS= read -r json_line; do
        local timestamp=$(echo "$json_line" | jq -r '.timestamp')
        local sensor_id=$(echo "$json_line" | jq -r '.sensor_id')
        local value=$(echo "$json_line" | jq -r '.value')
        
        run redis-cli -h localhost -p "$REDIS_PORT" XADD sensor_stream "*" \
            timestamp "$timestamp" sensor_id "$sensor_id" value "$value" pipeline_id "$PIPELINE_ID"
        assert_success
        data_ingested=$((data_ingested + 1))
    done < "$sensor_data_file"
    
    # Verify all data was ingested
    assert_equals "$data_ingested" "10"
    
    # Check stream length
    run redis-cli -h localhost -p "$REDIS_PORT" XLEN sensor_stream
    assert_success
    assert_greater_or_equal "$output" "10"
    
    local stage_duration=$(( $(date +%s) - stage_start_time ))
    echo "Data ingestion completed in ${stage_duration}s"
}

@test "pipeline: stage 2 - data validation" {
    local stage_start_time=$(date +%s)
    
    # Setup validation rules in Redis
    run redis-cli -h localhost -p "$REDIS_PORT" HSET "validation:rules" \
        "temperature_min" "-50" \
        "temperature_max" "100" \
        "required_fields" "timestamp,sensor_id,value" \
        "max_age_seconds" "3600"
    assert_success
    
    # Read stream data for validation
    run redis-cli -h localhost -p "$REDIS_PORT" XREAD COUNT 5 STREAMS sensor_stream 0
    assert_success
    
    # Simulate validation process
    local validation_results="$PIPELINE_DATA_DIR/processed/validation_results.json"
    cat > "$validation_results" << 'EOF'
{
    "validation_summary": {
        "total_records": 10,
        "valid_records": 9,
        "invalid_records": 1,
        "validation_errors": [
            {"record_id": "5", "error": "temperature out of range", "value": 150}
        ]
    },
    "quality_score": 0.9
}
EOF

    # Store validation results in QuestDB
    local quality_score=$(jq -r '.quality_score' "$validation_results")
    local total_records=$(jq -r '.validation_summary.total_records' "$validation_results")
    local invalid_count=$(jq -r '.validation_summary.invalid_records' "$validation_results")
    
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=INSERT INTO data_quality VALUES (now(), 'sensor_data', $quality_score, $invalid_count, $total_records)" \
        --data-urlencode "fmt=json"
    assert_success
    
    local stage_duration=$(( $(date +%s) - stage_start_time ))
    echo "Data validation completed in ${stage_duration}s, quality score: $quality_score"
}

@test "pipeline: stage 3 - real-time processing" {
    local stage_start_time=$(date +%s)
    
    # Setup N8N workflow for real-time processing
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/workflows/execute" \
        '{"executionId":"realtime_001","status":"running","workflow":"realtime_processor"}' 200
    
    # Trigger real-time processing workflow
    run curl -s -X POST "$N8N_BASE_URL/api/v1/workflows/execute" \
        -H "Content-Type: application/json" \
        -d '{"workflowId":"realtime_processor","data":{"pipeline_id":"'"$PIPELINE_ID"'","stream":"sensor_stream"}}'
    assert_success
    
    local execution_id=$(echo "$output" | jq -r '.executionId')
    assert_output_not_empty "$execution_id"
    
    # Simulate real-time metric calculations
    local current_hour=$(date +%H)
    local current_date=$(date +%Y-%m-%d)
    
    # Insert real-time metrics
    for metric in "cpu_usage" "memory_usage" "temperature"; do
        local value=$((RANDOM % 100))
        run curl -s -G "$QUESTDB_BASE_URL/exec" \
            --data-urlencode "query=INSERT INTO realtime_metrics VALUES (now(), '$metric', $value, 'test_system', 'environment=test,pipeline=$PIPELINE_ID')" \
            --data-urlencode "fmt=json"
        assert_success
    done
    
    # Verify metrics were inserted
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT COUNT(*) as count FROM realtime_metrics WHERE tags LIKE '%$PIPELINE_ID%'" \
        --data-urlencode "fmt=json"
    assert_success
    
    local metric_count=$(echo "$output" | jq -r '.dataset[0][0]')
    assert_greater_or_equal "$metric_count" "3"
    
    local stage_duration=$(( $(date +%s) - stage_start_time ))
    echo "Real-time processing completed in ${stage_duration}s, $metric_count metrics processed"
}

@test "pipeline: stage 4 - batch analytics" {
    local stage_start_time=$(date +%s)
    
    # Setup Ollama for AI-powered analytics
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/generate" \
        '{"response":"Based on the metrics data, I observe normal system performance with temperature readings within acceptable ranges and stable CPU/memory utilization.","done":true}' 200
    
    # Perform batch analytics query
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT metric_name, AVG(metric_value) as avg_value, MIN(metric_value) as min_value, MAX(metric_value) as max_value, COUNT(*) as count FROM realtime_metrics WHERE tags LIKE '%$PIPELINE_ID%' GROUP BY metric_name" \
        --data-urlencode "fmt=json"
    assert_success
    
    # Extract analytics data
    local analytics_data="$PIPELINE_DATA_DIR/analytics/batch_results.json"
    echo "$output" > "$analytics_data"
    
    # Verify analytics results
    assert_file_exists "$analytics_data"
    local analytics_count=$(jq '.dataset | length' "$analytics_data")
    assert_greater_or_equal "$analytics_count" "1"
    
    # Use AI to analyze the results
    local analytics_summary=$(jq -c '.dataset' "$analytics_data")
    run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model":"llama3.1:8b","prompt":"Analyze this metrics data: '"$analytics_summary"'","stream":false}'
    assert_success
    
    local ai_analysis=$(echo "$output" | jq -r '.response')
    assert_output_not_empty "$ai_analysis"
    
    # Store analytics summary
    local current_date=$(date +%Y-%m-%d)
    local current_hour=$(date +%H)
    
    # Insert aggregated analytics
    local first_metric=$(jq -r '.dataset[0][0]' "$analytics_data")
    local avg_value=$(jq -r '.dataset[0][1]' "$analytics_data")
    local min_value=$(jq -r '.dataset[0][2]' "$analytics_data")
    local max_value=$(jq -r '.dataset[0][3]' "$analytics_data")
    local count=$(jq -r '.dataset[0][4]' "$analytics_data")
    
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=INSERT INTO analytics_summary VALUES ('$current_date', $current_hour, '$first_metric', $avg_value, $min_value, $max_value, $count)" \
        --data-urlencode "fmt=json"
    assert_success
    
    local stage_duration=$(( $(date +%s) - stage_start_time ))
    echo "Batch analytics completed in ${stage_duration}s"
    echo "AI Analysis: $ai_analysis"
}

@test "pipeline: stage 5 - result storage and export" {
    local stage_start_time=$(date +%s)
    
    # Export data to different formats
    local export_dir="$PIPELINE_DATA_DIR/exports"
    
    # Export to CSV
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT * FROM analytics_summary ORDER BY date DESC, hour DESC LIMIT 100" \
        --data-urlencode "fmt=csv" \
        -o "$export_dir/analytics_export.csv"
    assert_success
    assert_file_exists "$export_dir/analytics_export.csv"
    
    # Export to JSON
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT * FROM data_quality ORDER BY timestamp DESC LIMIT 50" \
        --data-urlencode "fmt=json" \
        -o "$export_dir/quality_export.json"
    assert_success
    assert_file_exists "$export_dir/quality_export.json"
    
    # Cache final results in Redis
    run redis-cli -h localhost -p "$REDIS_PORT" SET "pipeline:${PIPELINE_ID}:final_results" \
        '{"status":"completed","exports":["analytics_export.csv","quality_export.json"],"timestamp":"'"$(date -Iseconds)"'"}'
    assert_success
    assert_output "OK"
    
    # Update pipeline status
    run redis-cli -h localhost -p "$REDIS_PORT" SET "pipeline:${PIPELINE_ID}:status" "completed"
    assert_success
    
    local stage_duration=$(( $(date +%s) - stage_start_time ))
    echo "Result storage completed in ${stage_duration}s"
}

#######################################
# END-TO-END PIPELINE TESTS
#######################################

@test "pipeline: complete data pipeline execution" {
    local pipeline_start_time=$(date +%s)
    
    echo "Starting complete data pipeline: $PIPELINE_NAME"
    
    # Simulate complete data flow
    local test_data_points=20
    
    # Generate and ingest test data
    for i in $(seq 1 $test_data_points); do
        local timestamp=$(date -d "$i minutes ago" -Iseconds)
        local value=$((50 + (i % 20)))
        
        # Add to Redis stream
        run redis-cli -h localhost -p "$REDIS_PORT" XADD complete_pipeline_stream "*" \
            timestamp "$timestamp" metric "system_load" value "$value" batch_id "complete_test"
        assert_success
    done
    
    # Process stream data
    run redis-cli -h localhost -p "$REDIS_PORT" XLEN complete_pipeline_stream
    assert_success
    assert_greater_or_equal "$output" "$test_data_points"
    
    # Batch insert into QuestDB
    for i in $(seq 1 $test_data_points); do
        local timestamp=$(date -d "$i minutes ago" -Iseconds)
        local value=$((50 + (i % 20)))
        
        run curl -s -G "$QUESTDB_BASE_URL/exec" \
            --data-urlencode "query=INSERT INTO realtime_metrics VALUES ('$timestamp', 'system_load', $value, 'complete_test', 'batch=complete_test')" \
            --data-urlencode "fmt=json"
        assert_success
    done
    
    # Verify data insertion
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT COUNT(*) as count FROM realtime_metrics WHERE source_system = 'complete_test'" \
        --data-urlencode "fmt=json"
    assert_success
    
    local inserted_count=$(echo "$output" | jq -r '.dataset[0][0]')
    assert_equals "$inserted_count" "$test_data_points"
    
    # Generate final analytics
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT AVG(metric_value) as avg, MIN(metric_value) as min, MAX(metric_value) as max FROM realtime_metrics WHERE source_system = 'complete_test'" \
        --data-urlencode "fmt=json"
    assert_success
    
    local avg_value=$(echo "$output" | jq -r '.dataset[0][0]')
    assert_not_equals "$avg_value" "null"
    
    local pipeline_duration=$(( $(date +%s) - pipeline_start_time ))
    assert_less_than "$pipeline_duration" "$MAX_PROCESSING_TIME"
    
    echo "Complete pipeline executed successfully in ${pipeline_duration}s"
    echo "Processed $test_data_points data points, average value: $avg_value"
}

@test "pipeline: data consistency and integrity" {
    # Test data consistency across different storage systems
    
    # Insert test record in both systems
    local test_id="consistency_test_$$"
    local test_value=99.99
    local test_timestamp=$(date -Iseconds)
    
    # Insert in QuestDB
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=INSERT INTO realtime_metrics VALUES ('$test_timestamp', 'consistency_test', $test_value, '$test_id', 'test=consistency')" \
        --data-urlencode "fmt=json"
    assert_success
    
    # Cache in Redis
    run redis-cli -h localhost -p "$REDIS_PORT" HSET "consistency:$test_id" \
        "timestamp" "$test_timestamp" \
        "value" "$test_value" \
        "source" "$test_id"
    assert_success
    
    # Verify data in QuestDB
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT metric_value FROM realtime_metrics WHERE source_system = '$test_id'" \
        --data-urlencode "fmt=json"
    assert_success
    
    local questdb_value=$(echo "$output" | jq -r '.dataset[0][0]')
    assert_equals "$questdb_value" "$test_value"
    
    # Verify data in Redis
    run redis-cli -h localhost -p "$REDIS_PORT" HGET "consistency:$test_id" "value"
    assert_success
    assert_output "$test_value"
    
    echo "Data consistency verified across QuestDB and Redis"
}

@test "pipeline: monitoring and alerting" {
    # Test pipeline monitoring capabilities
    
    # Setup monitoring thresholds
    local max_processing_time=30
    local min_quality_score=0.8
    local max_error_rate=0.1
    
    # Simulate monitoring data
    run redis-cli -h localhost -p "$REDIS_PORT" HMSET "monitoring:$PIPELINE_ID" \
        "avg_processing_time" "15" \
        "current_quality_score" "0.95" \
        "error_rate" "0.02" \
        "last_update" "$(date -Iseconds)"
    assert_success
    
    # Check monitoring metrics
    run redis-cli -h localhost -p "$REDIS_PORT" HGET "monitoring:$PIPELINE_ID" "avg_processing_time"
    assert_success
    local processing_time="$output"
    assert_less_than "$processing_time" "$max_processing_time"
    
    run redis-cli -h localhost -p "$REDIS_PORT" HGET "monitoring:$PIPELINE_ID" "current_quality_score"
    assert_success
    local quality_score="$output"
    assert_greater_than "$quality_score" "$min_quality_score"
    
    run redis-cli -h localhost -p "$REDIS_PORT" HGET "monitoring:$PIPELINE_ID" "error_rate"
    assert_success
    local error_rate="$output"
    assert_less_than "$error_rate" "$max_error_rate"
    
    echo "Pipeline monitoring: processing_time=${processing_time}s, quality=${quality_score}, errors=${error_rate}"
}

#######################################
# PIPELINE HELPER FUNCTIONS
#######################################

# Helper: Generate test time-series data
generate_time_series_data() {
    local metric_name="$1"
    local count="${2:-100}"
    local output_file="$3"
    
    for i in $(seq 1 "$count"); do
        local timestamp=$(date -d "$i seconds ago" -Iseconds)
        local value=$(( (RANDOM % 100) + 1 ))
        echo '{"timestamp":"'"$timestamp"'","metric":"'"$metric_name"'","value":'"$value"'}' >> "$output_file"
    done
}

# Helper: Check pipeline stage status
pipeline_check_stage_status() {
    local stage_name="$1"
    
    redis-cli -h localhost -p "$REDIS_PORT" HGET "pipeline:${PIPELINE_ID}:stages" "$stage_name"
}

# Helper: Update pipeline stage status
pipeline_update_stage_status() {
    local stage_name="$1"
    local status="$2"
    
    redis-cli -h localhost -p "$REDIS_PORT" HSET "pipeline:${PIPELINE_ID}:stages" "$stage_name" "$status"
}

# Helper: Get pipeline metrics summary
pipeline_get_metrics_summary() {
    curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT metric_name, COUNT(*) as count, AVG(metric_value) as avg_value FROM realtime_metrics WHERE tags LIKE '%$PIPELINE_ID%' GROUP BY metric_name" \
        --data-urlencode "fmt=json"
}