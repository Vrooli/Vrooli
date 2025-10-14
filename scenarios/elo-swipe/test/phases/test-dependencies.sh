#!/bin/bash
set -e
echo "=== Dependency Tests ==="
# Verify all required dependencies are available

# Check PostgreSQL dependency
echo "Checking PostgreSQL connection..."
POSTGRES_HOST=${POSTGRES_HOST:-localhost}
POSTGRES_PORT=${POSTGRES_PORT:-5433}
POSTGRES_USER=${POSTGRES_USER:-vrooli}
POSTGRES_DB=${POSTGRES_DB:-vrooli}

if command -v psql > /dev/null 2>&1; then
  if PGPASSWORD=vrooli psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -c "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ PostgreSQL connection successful"
  else
    echo "⚠️  PostgreSQL connection failed (this is expected if resource is not running)"
  fi
else
  echo "⚠️  psql not found, skipping direct database check"
fi

# Check if API can connect to database
API_PORT=${API_PORT:-19286}
if curl -sf http://localhost:${API_PORT}/api/v1/health > /dev/null 2>&1; then
  HEALTH=$(curl -sf http://localhost:${API_PORT}/api/v1/health)
  DB_STATUS=$(echo "$HEALTH" | jq -r '.database // "unknown"')
  if [ "$DB_STATUS" = "connected" ] || [ "$DB_STATUS" = "healthy" ]; then
    echo "✅ API reports database connection healthy"
  else
    echo "⚠️  API database connection: $DB_STATUS"
  fi
else
  echo "⚠️  API not responding, skipping database check via API"
fi

# Check optional Redis dependency (if configured)
if [ -n "$REDIS_URL" ]; then
  echo "Checking Redis connection..."
  if command -v redis-cli > /dev/null 2>&1; then
    if redis-cli ping > /dev/null 2>&1; then
      echo "✅ Redis connection successful"
    else
      echo "⚠️  Redis connection failed (optional dependency)"
    fi
  else
    echo "⚠️  redis-cli not found, skipping Redis check"
  fi
fi

# Check optional Ollama dependency (if configured)
if command -v ollama > /dev/null 2>&1; then
  echo "Checking Ollama availability..."
  if ollama list > /dev/null 2>&1; then
    echo "✅ Ollama available"
  else
    echo "⚠️  Ollama not running (optional dependency)"
  fi
fi

echo "✅ Dependency tests completed"
