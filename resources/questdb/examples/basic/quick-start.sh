#!/usr/bin/env bash
# QuestDB Quick Start Examples
# Basic usage patterns for common operations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../../.." && builtin pwd)}"
SCRIPT_DIR="$APP_ROOT/resources/questdb/examples/basic"
MANAGE_SCRIPT="$APP_ROOT/resources/questdb/manage.sh"

echo "🚀 QuestDB Quick Start Examples"
echo "================================"

# Check if QuestDB is running
echo ""
echo "1️⃣ Checking QuestDB status..."
"$MANAGE_SCRIPT" --action status

# List tables
echo ""
echo "2️⃣ Listing available tables..."
"$MANAGE_SCRIPT" --action tables

# Insert sample metrics via HTTP API
echo ""
echo "3️⃣ Inserting sample metrics..."
echo "   Using HTTP REST API..."

# System metrics
curl -s -G "http://localhost:9009/exec" \
  --data-urlencode "query=INSERT INTO system_metrics(timestamp, host, metric_name, metric_value, metric_unit) VALUES(now(), 'demo-host', 'cpu_usage', 45.2, 'percent')"

# AI inference metrics
curl -s -G "http://localhost:9009/exec" \
  --data-urlencode "query=INSERT INTO ai_inference(timestamp, model, task_type, response_time_ms, tokens_input, tokens_output, success) VALUES(now(), 'llama3.2', 'chat', 245.5, 50, 150, true)"

echo "   ✅ Sample data inserted"

# Insert via InfluxDB Line Protocol
echo ""
echo "4️⃣ Inserting via InfluxDB Line Protocol..."
echo "resource_health,resource=questdb,type=storage status=\"healthy\",response_time_ms=5.2 $(date +%s)000000000" | nc localhost 9003
echo "   ✅ Metric sent via ILP"

# Query recent data
echo ""
echo "5️⃣ Querying recent metrics..."
echo "   Last 5 system metrics:"
"$MANAGE_SCRIPT" --action query --query "SELECT * FROM system_metrics ORDER BY timestamp DESC LIMIT 5"

echo ""
echo "   AI inference summary:"
"$MANAGE_SCRIPT" --action query --query "SELECT model, COUNT(*) as count, AVG(response_time_ms) as avg_response FROM ai_inference GROUP BY model"

# Time-series specific queries
echo ""
echo "6️⃣ Time-series aggregation example..."
"$MANAGE_SCRIPT" --action query --query "SELECT timestamp, AVG(metric_value) FROM system_metrics WHERE metric_name = 'cpu_usage' AND timestamp > now() - 1h SAMPLE BY 5m"

# Show web console URL
echo ""
echo "7️⃣ Access points:"
echo "   🌐 Web Console: http://localhost:9009"
echo "   🐘 PostgreSQL: psql -h localhost -p 8812 -U admin -d qdb"
echo "   📊 REST API: http://localhost:9009/exec"
echo "   ⚡ ILP TCP: localhost:9003"

echo ""
echo "✅ Quick start complete! Explore more in the examples directory."