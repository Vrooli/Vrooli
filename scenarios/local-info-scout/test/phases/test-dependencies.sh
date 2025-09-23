#!/bin/bash
set -e

echo "=== Testing Dependencies ==="

cd ../api
go mod tidy

if ! go mod verify; then
  echo "❌ Dependency verification failed"
  exit 1
fi

echo "✅ Dependencies tests passed"