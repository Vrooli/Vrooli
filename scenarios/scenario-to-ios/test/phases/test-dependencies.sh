#!/bin/bash
set -e

echo "=== Dependency Tests ==="

# Navigate to scenario root
cd "$(dirname "$0")/../.."

# Check Go dependencies
echo "Checking Go module dependencies..."
cd api
go mod tidy
echo "✅ Go dependencies are clean"

echo "✅ All dependency tests completed"
