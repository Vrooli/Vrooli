#!/bin/bash
set -e
echo "=== Performance Tests ==="
# Run performance benchmarks
cd ../api && go test -bench=. -benchmem ./...
echo "✅ Performance tests passed"