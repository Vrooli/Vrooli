#!/usr/bin/env bash
# App Monitor Validation Script
set -euo pipefail

echo "Validating App Monitor deployment..."

# Check PostgreSQL
if psql -U postgres -d app_monitor -c "SELECT 1" > /dev/null 2>&1; then
    echo "✓ PostgreSQL database connected"
else
    echo "✗ PostgreSQL database not accessible"
    exit 1
fi

# Check Redis
if redis-cli ping > /dev/null 2>&1; then
    echo "✓ Redis connected"
else
    echo "✗ Redis not accessible"
    exit 1
fi

# Check Docker socket
if docker ps > /dev/null 2>&1; then
    echo "✓ Docker API accessible"
else
    echo "✗ Docker API not accessible"
    exit 1
fi

# Check UI
if curl -s -o /dev/null -w "%{http_code}" http://localhost:3001 | grep -q "200\|304"; then
    echo "✓ UI dashboard accessible"
else
    echo "✗ UI dashboard not accessible"
    exit 1
fi

# Check API
if curl -s http://localhost:8081/health | grep -q "healthy"; then
    echo "✓ API healthy"
else
    echo "✗ API not healthy"
    exit 1
fi

echo ""
echo "App Monitor validation successful!"
echo "All components are operational."