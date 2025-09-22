#!/bin/bash
set -e

echo "Running unit tests for system-monitor"

cd ../api && go test -v ./... -short

echo "✅ Unit tests passed"
