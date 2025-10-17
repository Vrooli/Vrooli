#!/bin/bash
set -e
echo "=== Structure Checks ==="
make fmt lint typecheck
echo "✅ Structure checks completed"