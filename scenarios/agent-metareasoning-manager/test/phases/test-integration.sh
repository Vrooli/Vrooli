#!/bin/bash
set -e
echo "=== Integration Tests ==="
cd ../api && go test -v -tags=integration ./...
echo "✅ Integration tests passed"