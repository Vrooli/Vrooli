#!/bin/bash
set -e

echo "Running dependency tests for system-monitor"

cd ../api && go mod tidy

cd ../ui && npm ci --dry-run

echo "✅ Dependency tests passed"
