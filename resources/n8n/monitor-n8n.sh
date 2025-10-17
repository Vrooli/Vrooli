#!/usr/bin/env bash
# Simple n8n monitoring script
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 n8n Monitoring Dashboard"
echo "==========================="
echo ""

# Check container status
echo "📦 Container Status:"
if docker ps --filter "name=n8n" --format '{{.Names}}' | grep -q '^n8n$'; then
    echo -e "   ${GREEN}✓ Container running${NC}"
    UPTIME=$(docker ps --filter "name=n8n" --format '{{.Status}}' | head -1)
    echo "   Uptime: $UPTIME"
else
    echo -e "   ${RED}✗ Container not running${NC}"
    exit 1
fi
echo ""

# Check memory usage
echo "💾 Memory Usage:"
MEMORY_STATS=$(docker stats n8n --no-stream --format "{{.MemUsage}} ({{.MemPerc}})")
MEMORY_PERCENT=$(docker stats n8n --no-stream --format "{{.MemPerc}}" | sed 's/%//')

echo "   Current: $MEMORY_STATS"

# Memory threshold warnings
if (( $(echo "$MEMORY_PERCENT > 90" | bc -l) )); then
    echo -e "   ${RED}⚠️  CRITICAL: Memory usage above 90%${NC}"
elif (( $(echo "$MEMORY_PERCENT > 80" | bc -l) )); then
    echo -e "   ${RED}⚠️  WARNING: Memory usage above 80%${NC}"
elif (( $(echo "$MEMORY_PERCENT > 70" | bc -l) )); then
    echo -e "   ${YELLOW}⚠️  CAUTION: Memory usage above 70%${NC}"
else
    echo -e "   ${GREEN}✓ Memory usage healthy${NC}"
fi
echo ""

# Check CPU usage
echo "🔥 CPU Usage:"
CPU_USAGE=$(docker stats n8n --no-stream --format "{{.CPUPerc}}")
echo "   Current: $CPU_USAGE"
echo ""

# Check health endpoint
echo "🏥 Health Check:"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 3 http://localhost:5678/healthz 2>/dev/null || echo "0")
if [ "$HTTP_CODE" = "200" ]; then
    echo -e "   ${GREEN}✓ Health endpoint responding (HTTP $HTTP_CODE)${NC}"
else
    echo -e "   ${RED}✗ Health endpoint not responding (HTTP $HTTP_CODE)${NC}"
fi
echo ""

# Check API responsiveness
echo "🔌 API Status:"
START_TIME=$(date +%s%N)
API_RESPONSE=$(curl -s --max-time 3 http://localhost:5678/api/v1 2>/dev/null | jq -r '.message' 2>/dev/null || echo "timeout")
END_TIME=$(date +%s%N)
RESPONSE_TIME=$(echo "scale=3; ($END_TIME - $START_TIME) / 1000000000" | bc)

if [ "$API_RESPONSE" = "not found" ]; then
    echo -e "   ${GREEN}✓ API responding normally${NC}"
    echo "   Response time: ${RESPONSE_TIME}s"
elif [ "$API_RESPONSE" = "timeout" ]; then
    echo -e "   ${RED}✗ API timeout${NC}"
else
    echo -e "   ${YELLOW}⚠️  Unexpected API response: $API_RESPONSE${NC}"
fi
echo ""

# Check database connection
echo "🗄️  Database Connection:"
DB_WORKFLOWS=$(docker exec vrooli-n8n-postgres psql -U n8n -d n8n -t -c "SELECT COUNT(*) FROM workflow_entity;" 2>/dev/null | xargs)
if [ -n "$DB_WORKFLOWS" ]; then
    echo -e "   ${GREEN}✓ PostgreSQL connected${NC}"
    echo "   Workflows in database: $DB_WORKFLOWS"
else
    echo -e "   ${RED}✗ Database connection failed${NC}"
fi
echo ""

# Check for recent errors in logs
echo "📝 Recent Log Analysis:"
ERROR_COUNT=$(docker logs n8n --since "1h" 2>&1 | grep -i "error" | wc -l)
WARN_COUNT=$(docker logs n8n --since "1h" 2>&1 | grep -i "warn" | wc -l)

if [ "$ERROR_COUNT" -eq 0 ] && [ "$WARN_COUNT" -eq 0 ]; then
    echo -e "   ${GREEN}✓ No errors or warnings in last hour${NC}"
else
    [ "$ERROR_COUNT" -gt 0 ] && echo -e "   ${RED}⚠️  Errors in last hour: $ERROR_COUNT${NC}"
    [ "$WARN_COUNT" -gt 0 ] && echo -e "   ${YELLOW}⚠️  Warnings in last hour: $WARN_COUNT${NC}"
fi
echo ""

# Configuration summary
echo "⚙️  Configuration:"
echo "   Memory Limit: $(docker inspect n8n --format '{{.HostConfig.Memory}}' | awk '{print $1/1024/1024/1024 "GB"}')"
echo "   Webhook Timeout: $(docker exec n8n env | grep N8N_WEBHOOK_TIMEOUT | cut -d= -f2)s"
echo "   Execution Timeout: $(docker exec n8n env | grep '^EXECUTIONS_TIMEOUT=' | cut -d= -f2)s"
echo "   Max Payload Size: $(docker exec n8n env | grep N8N_PAYLOAD_SIZE_MAX | cut -d= -f2)MB"
echo ""

# Overall status
echo "📊 Overall Status:"
if [ "$HTTP_CODE" = "200" ] && [ "$ERROR_COUNT" -eq 0 ] && (( $(echo "$MEMORY_PERCENT < 80" | bc -l) )); then
    echo -e "   ${GREEN}✅ System Healthy - All checks passed${NC}"
elif [ "$HTTP_CODE" = "200" ] && (( $(echo "$MEMORY_PERCENT < 90" | bc -l) )); then
    echo -e "   ${YELLOW}⚠️  System Operational - Minor issues detected${NC}"
else
    echo -e "   ${RED}❌ System Degraded - Critical issues detected${NC}"
fi
echo ""
echo "Last checked: $(date '+%Y-%m-%d %H:%M:%S')"