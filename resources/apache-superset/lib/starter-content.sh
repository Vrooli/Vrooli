#!/usr/bin/env bash
# Apache Superset Starter Content Library
# Provides pre-built dashboards and templates for immediate value

# Load dependencies
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi
# Source dependencies only if not already loaded
[[ -z "${SUPERSET_PORT:-}" ]] && source "${SCRIPT_DIR}/../config/defaults.sh"

# Create KPI Dashboard Template
superset::create_kpi_dashboard() {
    log::info "Creating KPI Dashboard Template..."
    
    local token=$(superset::get_auth_token)
    if [[ -z "$token" ]]; then
        log::error "Failed to get authentication token"
        return 1
    fi
    
    # First, ensure we have a database connection for Vrooli's Postgres
    local db_id=$(superset::ensure_database_connection "Vrooli Main DB" "postgresql://vrooli:vrooli@host.docker.internal:5433/vrooli")
    if [[ -z "$db_id" ]]; then
        log::error "Failed to create database connection"
        return 1
    fi
    
    # Create dataset for KPI metrics
    local dataset_id=$(superset::create_dataset "$db_id" "kpi_metrics" "
        SELECT 
            'Revenue' as metric_name,
            50000 as current_value,
            45000 as previous_value,
            10.0 as growth_rate,
            CURRENT_TIMESTAMP as last_updated
        UNION ALL
        SELECT 
            'Active Users' as metric_name,
            1250 as current_value,
            1100 as previous_value,
            13.6 as growth_rate,
            CURRENT_TIMESTAMP as last_updated
        UNION ALL
        SELECT 
            'System Uptime' as metric_name,
            99.9 as current_value,
            99.7 as previous_value,
            0.2 as growth_rate,
            CURRENT_TIMESTAMP as last_updated
        UNION ALL
        SELECT 
            'Response Time' as metric_name,
            125 as current_value,
            180 as previous_value,
            -30.5 as growth_rate,
            CURRENT_TIMESTAMP as last_updated
    ")
    
    # Create simple KPI dashboard JSON first
    local dashboard_json=$(cat << 'DASHBOARD_JSON'
{
    "dashboard_title": "Executive KPI Dashboard"
}
DASHBOARD_JSON
)
    
    # Create the dashboard (CSRF should be exempt for API routes)
    local csrf_token=$(superset::get_csrf_token "$token")
    local response=$(curl -s -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: ${csrf_token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/" \
        -d "$dashboard_json")
    
    local dashboard_id=$(echo "$response" | jq -r '.id')
    if [[ "$dashboard_id" != "null" ]] && [[ -n "$dashboard_id" ]]; then
        log::info "✅ Created KPI Dashboard (ID: ${dashboard_id})"
        
        # Create charts for the dashboard
        superset::create_kpi_charts "$dashboard_id" "$dataset_id" "$token"
        
        return 0
    else
        log::error "Failed to create KPI dashboard: $response"
        return 1
    fi
}

# Create Scenario Health Monitor Dashboard
superset::create_scenario_health_dashboard() {
    log::info "Creating Scenario Health Monitor Dashboard..."
    
    local token=$(superset::get_auth_token)
    if [[ -z "$token" ]]; then
        log::error "Failed to get authentication token"
        return 1
    fi
    
    
    # Create simple dashboard JSON
    local dashboard_json=$(cat << 'DASHBOARD_JSON'
{
    "dashboard_title": "Scenario Health Monitor"
}
DASHBOARD_JSON
)
    
    local csrf_token=$(superset::get_csrf_token "$token")
    local response=$(curl -s -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: ${csrf_token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/" \
        -d "$dashboard_json")
    
    local dashboard_id=$(echo "$response" | jq -r '.id')
    if [[ "$dashboard_id" != "null" ]] && [[ -n "$dashboard_id" ]]; then
        log::info "✅ Created Scenario Health Monitor Dashboard (ID: ${dashboard_id})"
        return 0
    else
        log::error "Failed to create scenario health dashboard: $response"
        return 1
    fi
}

# Create Resource Usage Analytics Dashboard
superset::create_resource_usage_dashboard() {
    log::info "Creating Resource Usage Analytics Dashboard..."
    
    local token=$(superset::get_auth_token)
    if [[ -z "$token" ]]; then
        log::error "Failed to get authentication token"
        return 1
    fi
    
    
    # Create simple dashboard JSON
    local dashboard_json=$(cat << 'DASHBOARD_JSON'
{
    "dashboard_title": "Resource Usage Analytics"
}
DASHBOARD_JSON
)
    
    local csrf_token=$(superset::get_csrf_token "$token")
    local response=$(curl -s -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: ${csrf_token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/" \
        -d "$dashboard_json")
    
    local dashboard_id=$(echo "$response" | jq -r '.id')
    if [[ "$dashboard_id" != "null" ]] && [[ -n "$dashboard_id" ]]; then
        log::info "✅ Created Resource Usage Analytics Dashboard (ID: ${dashboard_id})"
        return 0
    else
        log::error "Failed to create resource usage dashboard: $response"
        return 1
    fi
}

# Helper: Ensure database connection exists
superset::ensure_database_connection() {
    local name="$1"
    local uri="$2"
    
    local token=$(superset::get_auth_token)
    
    # Check if database already exists
    local existing=$(curl -s -H "Authorization: Bearer ${token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/database/" | \
        jq -r ".result[] | select(.database_name == \"$name\") | .id")
    
    if [[ -n "$existing" ]]; then
        echo "$existing"
        return 0
    fi
    
    # Create new database connection
    local csrf_token=$(superset::get_csrf_token "$token")
    local response=$(curl -s -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: ${csrf_token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/database/" \
        -d "{
            \"database_name\": \"$name\",
            \"sqlalchemy_uri\": \"$uri\",
            \"expose_in_sqllab\": true,
            \"allow_ctas\": true,
            \"allow_cvas\": true,
            \"allow_dml\": true,
            \"allow_multi_schema_metadata_fetch\": true,
            \"extra\": \"{\\\"allows_virtual_table_explore\\\": true}\"
        }")
    
    echo "$response" | jq -r '.id'
}

# Helper: Create dataset from SQL
superset::create_dataset() {
    local database_id="$1"
    local table_name="$2"
    local sql_query="$3"
    
    local token=$(superset::get_auth_token)
    
    # Create virtual dataset
    local csrf_token=$(superset::get_csrf_token "$token")
    local response=$(curl -s -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: ${csrf_token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/dataset/" \
        -d "{
            \"database\": $database_id,
            \"table_name\": \"$table_name\",
            \"sql\": \"$sql_query\",
            \"is_managed_externally\": false
        }")
    
    echo "$response" | jq -r '.id'
}

# Helper: Create KPI charts
superset::create_kpi_charts() {
    local dashboard_id="$1"
    local dataset_id="$2"
    local token="$3"
    
    # Create metric cards for each KPI
    local metrics=("Revenue" "Active Users" "System Uptime" "Response Time")
    
    for metric in "${metrics[@]}"; do
        local chart_json=$(cat << CHART_JSON
{
    "slice_name": "KPI: $metric",
    "viz_type": "big_number_total",
    "datasource_id": $dataset_id,
    "datasource_type": "table",
    "params": {
        "metric": {
            "expressionType": "SIMPLE",
            "column": {
                "column_name": "current_value"
            },
            "aggregate": "SUM"
        },
        "adhoc_filters": [
            {
                "expressionType": "SIMPLE",
                "subject": "metric_name",
                "operator": "==",
                "comparator": "$metric",
                "clause": "WHERE"
            }
        ],
        "header_font_size": 0.3,
        "subheader_font_size": 0.2,
        "y_axis_format": ",.0f",
        "time_format": "%Y-%m-%d"
    },
    "dashboards": [$dashboard_id]
}
CHART_JSON
)
        
            curl -s -X POST \
            -H "Authorization: Bearer ${token}" \
            -H "Content-Type: application/json" \
                "http://localhost:${SUPERSET_PORT}/api/v1/chart/" \
            -d "$chart_json" > /dev/null
    done
}

# Create sample alert configuration
superset::create_alert_configs() {
    log::info "Creating sample alert configurations..."
    
    local token=$(superset::get_auth_token)
    if [[ -z "$token" ]]; then
        log::error "Failed to get authentication token"
        return 1
    fi
    
    
    # Create alert for high response time
    local alert_json=$(cat << 'ALERT_JSON'
{
    "name": "High Response Time Alert",
    "type": "Alert",
    "crontab": "*/5 * * * *",
    "sql": "SELECT AVG(response_time) as avg_response FROM metrics WHERE time > NOW() - INTERVAL '5 minutes'",
    "database_id": 1,
    "validator_type": "operator",
    "validator_config_json": "{\"op\": \">\", \"threshold\": 500}",
    "grace_period": 300,
    "recipients": [
        {
            "type": "email",
            "recipient_config_json": "{\"target\": \"admin@vrooli.com\"}"
        }
    ],
    "slice_id": null,
    "dashboard_id": null,
    "chart_id": null,
    "force_screenshot": false,
    "log_retention": 90,
    "active": true,
    "description": "Alert when average response time exceeds 500ms"
}
ALERT_JSON
)
    
    local response=$(curl -s -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        "http://localhost:${SUPERSET_PORT}/api/v1/report/" \
        -d "$alert_json")
    
    local alert_id=$(echo "$response" | jq -r '.id')
    if [[ "$alert_id" != "null" ]] && [[ -n "$alert_id" ]]; then
        log::info "✅ Created High Response Time Alert (ID: ${alert_id})"
    else
        log::warn "Could not create alert (may require additional configuration)"
    fi
    
    # Create alert for low system uptime
    alert_json=$(cat << 'ALERT_JSON'
{
    "name": "Low System Uptime Alert",
    "type": "Alert", 
    "crontab": "*/10 * * * *",
    "sql": "SELECT uptime_percentage FROM system_health WHERE time = (SELECT MAX(time) FROM system_health)",
    "database_id": 1,
    "validator_type": "operator",
    "validator_config_json": "{\"op\": \"<\", \"threshold\": 99.0}",
    "grace_period": 600,
    "active": true,
    "description": "Alert when system uptime drops below 99%"
}
ALERT_JSON
)
    
    response=$(curl -s -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        "http://localhost:${SUPERSET_PORT}/api/v1/report/" \
        -d "$alert_json")
    
    alert_id=$(echo "$response" | jq -r '.id')
    if [[ "$alert_id" != "null" ]] && [[ -n "$alert_id" ]]; then
        log::info "✅ Created Low System Uptime Alert (ID: ${alert_id})"
    fi
}

# Initialize all starter content
superset::init_starter_content() {
    log::info "Initializing Apache Superset starter content..."
    
    # Wait for Superset to be ready
    local max_attempts=30
    local attempt=0
    while [[ $attempt -lt $max_attempts ]]; do
        if timeout 5 curl -sf "http://localhost:${SUPERSET_PORT}/health" > /dev/null 2>&1; then
            break
        fi
        attempt=$((attempt + 1))
        sleep 2
    done
    
    if [[ $attempt -ge $max_attempts ]]; then
        log::error "Superset is not responding after 60 seconds"
        return 1
    fi
    
    local success=0
    local failed=0
    
    # Create each starter dashboard
    if superset::create_kpi_dashboard; then
        ((success++))
    else
        ((failed++))
    fi
    
    if superset::create_scenario_health_dashboard; then
        ((success++))
    else
        ((failed++))
    fi
    
    if superset::create_resource_usage_dashboard; then
        ((success++))
    else
        ((failed++))
    fi
    
    # Create alert configurations
    if superset::create_alert_configs; then
        ((success++))
    else
        ((failed++))
    fi
    
    log::info "Starter content initialization complete: ${success} successful, ${failed} failed"
    
    if [[ $failed -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

# Export functions
export -f superset::create_kpi_dashboard
export -f superset::create_scenario_health_dashboard
export -f superset::create_resource_usage_dashboard
export -f superset::create_alert_configs
export -f superset::init_starter_content
export -f superset::ensure_database_connection
export -f superset::create_dataset
export -f superset::get_csrf_token
export -f superset::get_auth_token
export -f superset::create_kpi_charts