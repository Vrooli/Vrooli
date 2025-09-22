#!/bin/bash
set -e
echo "=== Unit Tests ==="
cd ../api && go test -v ./... -short
echo "✅ Unit tests passed"