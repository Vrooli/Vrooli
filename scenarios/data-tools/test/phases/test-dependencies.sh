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

# Check for required resources
echo "Checking resource dependencies..."
resources_ok=true

# PostgreSQL
if ! curl -sf http://localhost:15432/health > /dev/null 2>&1; then
  echo "⚠️  PostgreSQL may not be running"
  resources_ok=false
fi

# Redis
if ! redis-cli ping > /dev/null 2>&1; then
  echo "⚠️  Redis may not be running"
  resources_ok=false
fi

if [ "$resources_ok" = true ]; then
  echo "✅ All resource dependencies available"
else
  echo "ℹ️  Some optional resources not running (this is OK for basic tests)"
fi

echo "✅ Dependency checks completed"
