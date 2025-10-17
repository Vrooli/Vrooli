#!/bin/bash
set -e
echo "=== Dependency Checks ==="

# Check Go dependencies
if [ -d api ] && [ -f api/go.mod ]; then
  echo "Checking Go dependencies..."
  cd api && go mod tidy && go mod verify
  echo "✅ Go dependencies verified"
  cd ..
fi

# Check Node.js dependencies (for UI)
if [ -d ui ] && [ -f ui/package.json ]; then
  echo "Checking Node.js dependencies..."
  cd ui
  if [ -f package-lock.json ]; then
    echo "✅ UI dependencies lock file present"
  fi
  cd ..
fi

# Check for optional resources (all are optional for this scenario)
echo "Checking optional resource dependencies..."
resources_ok=true

# PostgreSQL (optional - for calculation history)
if ! curl -sf http://localhost:5433/health > /dev/null 2>&1; then
  echo "ℹ️  PostgreSQL not running (optional - needed for calculation history)"
  resources_ok=false
fi

# Redis (optional - for caching)
if ! redis-cli -p 6379 ping > /dev/null 2>&1; then
  echo "ℹ️  Redis not running (optional - needed for caching)"
  resources_ok=false
fi

# Ollama (optional - for AI insights)
if ! curl -sf http://localhost:11434/api/tags > /dev/null 2>&1; then
  echo "ℹ️  Ollama not running (optional - needed for AI explanations)"
  resources_ok=false
fi

if [ "$resources_ok" = true ]; then
  echo "✅ All optional resources available"
else
  echo "ℹ️  Some optional resources not running (scenario works without them)"
fi

echo "✅ Dependency checks completed"
