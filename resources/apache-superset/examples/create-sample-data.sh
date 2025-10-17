#!/usr/bin/env bash
# Create sample data and dashboard for Apache Superset
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Creating sample data and dashboard..."

# Get auth token
TOKEN=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    "http://localhost:${SUPERSET_PORT}/api/v1/security/login" \
    -d "{\"username\": \"${SUPERSET_ADMIN_USERNAME}\", \"password\": \"${SUPERSET_ADMIN_PASSWORD}\", \"provider\": \"db\"}" | \
    jq -r '.access_token')

if [[ -z "$TOKEN" ]] || [[ "$TOKEN" == "null" ]]; then
    echo "Error: Failed to authenticate with Superset"
    exit 1
fi

# Create database connections for Vrooli resources
echo "Creating database connections..."

# 1. Vrooli PostgreSQL
curl -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    "http://localhost:${SUPERSET_PORT}/api/v1/database/" \
    -d '{
        "database_name": "Vrooli PostgreSQL",
        "sqlalchemy_uri": "postgresql://postgres:postgres@host.docker.internal:5433/postgres",
        "expose_in_sqllab": true,
        "allow_ctas": true,
        "allow_cvas": true,
        "allow_dml": true
    }' 2>/dev/null || echo "Vrooli PostgreSQL connection may already exist"

# 2. QuestDB Time Series
curl -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    "http://localhost:${SUPERSET_PORT}/api/v1/database/" \
    -d '{
        "database_name": "QuestDB Time Series",
        "sqlalchemy_uri": "postgresql://admin:quest@host.docker.internal:8812/qdb",
        "expose_in_sqllab": true,
        "allow_ctas": false,
        "allow_cvas": false,
        "allow_dml": false
    }' 2>/dev/null || echo "QuestDB connection may already exist"

# Create a sample dataset (using the metadata database)
echo "Creating sample dataset..."

DATASET_ID=$(curl -s -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    "http://localhost:${SUPERSET_PORT}/api/v1/dataset/" \
    -d '{
        "database": 1,
        "table_name": "logs",
        "sql": "SELECT NOW() as timestamp, RANDOM() * 100 as value, '\''scenario_'\'' || (RANDOM() * 10)::INT as scenario"
    }' | jq -r '.id')

echo "Sample dataset created with ID: ${DATASET_ID}"

# Create a sample chart
echo "Creating sample chart..."

CHART_ID=$(curl -s -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    "http://localhost:${SUPERSET_PORT}/api/v1/chart/" \
    -d '{
        "slice_name": "Scenario Activity",
        "viz_type": "line",
        "datasource_id": '${DATASET_ID}',
        "datasource_type": "table",
        "params": {
            "metrics": ["count"],
            "groupby": ["scenario"],
            "time_range": "Last day"
        }
    }' | jq -r '.id')

echo "Sample chart created with ID: ${CHART_ID}"

# Create KPI dashboard
echo "Creating KPI dashboard..."

DASHBOARD_ID=$(curl -s -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/" \
    -d "@${SCRIPT_DIR}/sample-dashboard.json" | jq -r '.id')

echo "KPI dashboard created with ID: ${DASHBOARD_ID}"

echo ""
echo "Sample data and dashboard created successfully!"
echo "Access the dashboard at: http://localhost:${SUPERSET_PORT}/superset/dashboard/${DASHBOARD_ID}/"
echo "Login with: ${SUPERSET_ADMIN_USERNAME} / ${SUPERSET_ADMIN_PASSWORD}"