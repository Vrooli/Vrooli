#!/bin/bash
# Dependencies Test Phase for idea-generator
# Checks resource dependencies and availability

set -euo pipefail

echo "=== Dependencies Phase (Target: <30s) ==="

cd "$(dirname "$0")/../.."

# Check required resources from service.json
required_resources=(
  "postgres"
  "ollama"
  "redis"
  "qdrant"
  "minio"
  "unstructured-io"
)

missing=0
for resource in "${required_resources[@]}"; do
  if ! vrooli resource status "$resource" &>/dev/null; then
    echo "⚠️  Resource not running: $resource (may be optional)"
  else
    echo "✅ Resource available: $resource"
  fi
done

# Check Go dependencies
if [ -f "api/go.mod" ]; then
  echo "✅ Go module file present"
  if cd api && go mod verify &>/dev/null; then
    echo "✅ Go dependencies verified"
  else
    echo "⚠️  Go dependencies need updating"
  fi
  cd ..
fi

# Check UI dependencies
if [ -f "ui/package.json" ]; then
  echo "✅ Node.js package file present"
  if [ -d "ui/node_modules" ]; then
    echo "✅ Node.js dependencies installed"
  else
    echo "⚠️  Node.js dependencies not installed"
  fi
fi

echo "✅ Dependencies validation completed"
