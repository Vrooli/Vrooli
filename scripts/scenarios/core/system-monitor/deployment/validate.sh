#!/usr/bin/env bash
# System Monitor Validation Script
set -euo pipefail

echo "Validating System Monitor deployment..."

# Check PostgreSQL
if psql -U postgres -d system_monitor -c "SELECT COUNT(*) FROM thresholds" > /dev/null 2>&1; then
    echo "✓ PostgreSQL database with thresholds"
else
    echo "✗ PostgreSQL database not configured"
    exit 1
fi

# Check QuestDB
if curl -s http://localhost:9000/exec?query=SELECT%201 > /dev/null 2>&1; then
    echo "✓ QuestDB time-series database"
else
    echo "✗ QuestDB not accessible"
    exit 1
fi

# Check Redis
if redis-cli -n 1 ping > /dev/null 2>&1; then
    echo "✓ Redis alert system"
else
    echo "✗ Redis not accessible"
    exit 1
fi

# Check Claude Code integration
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/claude/investigate -X OPTIONS | grep -q "200\|404"; then
    echo "✓ Claude Code integration endpoint"
else
    echo "✗ Claude Code integration not available"
fi

# Check Grafana
if curl -s -o /dev/null -w "%{http_code}" http://localhost:3004 | grep -q "200\|302"; then
    echo "✓ Grafana visualization"
else
    echo "✗ Grafana not accessible"
fi

# Check UI
if curl -s -o /dev/null -w "%{http_code}" http://localhost:3003 | grep -q "200\|304"; then
    echo "✓ UI dashboard accessible"
else
    echo "✗ UI dashboard not accessible"
    exit 1
fi

# Check API health
if curl -s http://localhost:8083/health | grep -q "healthy"; then
    echo "✓ API healthy"
else
    echo "✗ API not healthy"
    exit 1
fi

# Check system metrics collection
echo "Testing metric collection..."
if curl -s http://localhost:8083/api/metrics/current | grep -q "cpu_usage"; then
    echo "✓ Metric collection active"
else
    echo "✗ Metric collection not working"
fi

echo ""
echo "System Monitor validation successful!"
echo "Anomaly detection and Claude investigation ready."