#!/bin/bash
set -e
echo "=== Unit Tests ==="
if [ -d api ]; then cd api && go test -v ./... -short; fi
if [ -d ui ]; then cd ui && npm test -- --watchAll=false; fi
echo "✅ Unit tests completed"