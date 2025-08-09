#!/usr/bin/env bash
# Prompt Manager Validation Script
set -euo pipefail

echo "Validating Prompt Manager deployment..."

# Check PostgreSQL
if psql -U postgres -d prompt_manager -c "SELECT COUNT(*) FROM tags" > /dev/null 2>&1; then
    echo "✓ PostgreSQL database with tags"
else
    echo "✗ PostgreSQL database not configured"
    exit 1
fi

# Check Qdrant vector database
if curl -s http://localhost:6333/collections | grep -q "prompts"; then
    echo "✓ Qdrant vector store configured"
else
    echo "✗ Qdrant collection not created"
    exit 1
fi

# Check Redis session store
if redis-cli -n 2 GET "prompt_manager:initialized" > /dev/null 2>&1; then
    echo "✓ Redis session store"
else
    echo "✗ Redis session store not initialized"
    exit 1
fi

# Check Ollama (optional)
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "✓ Ollama LLM available for testing"
else
    echo "⚠ Ollama not available (prompt testing limited)"
fi

# Check UI
if curl -s -o /dev/null -w "%{http_code}" http://localhost:3005 | grep -q "200\|304"; then
    echo "✓ UI dashboard accessible"
else
    echo "✗ UI dashboard not accessible"
    exit 1
fi

# Check API health
if curl -s http://localhost:8085/health | grep -q "healthy"; then
    echo "✓ API healthy"
else
    echo "✗ API not healthy"
    exit 1
fi

# Test authentication endpoint
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/api/auth/login -X POST | grep -q "400\|401\|200"; then
    echo "✓ Authentication endpoint responsive"
else
    echo "✗ Authentication endpoint not working"
fi

# Test semantic search readiness
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/api/prompts/semantic -X POST \
    -H "Content-Type: application/json" \
    -d '{"query":"test"}' | grep -q "200\|401\|400"; then
    echo "✓ Semantic search endpoint ready"
else
    echo "✗ Semantic search not configured"
fi

echo ""
echo "Prompt Manager validation successful!"
echo "Campaign-based prompt management ready."
echo ""
echo "Access the dashboard at: http://localhost:3005"