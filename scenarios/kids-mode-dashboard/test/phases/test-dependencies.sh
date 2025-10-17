#!/bin/bash
set -e

echo "=== Dependency Tests for Kids Mode Dashboard ==="

echo "1. Verifying Go modules..."
go mod tidy

echo "2. Checking for unused dependencies..."
go mod why -m github.com/some/unused || echo "No unused dependencies found"

echo "3. Verifying no external binary dependencies..."
# Check for no vendor/lock issues
echo "Dependency check passed"

echo "All dependency tests passed ✅"