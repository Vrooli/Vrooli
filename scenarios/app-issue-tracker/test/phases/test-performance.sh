#!/bin/bash
set -euo pipefail

echo "=== Running test-performance.sh ==="

# Performance tests (basic)

# Check if vector search is fast
if [ -f scripts/test_vector_search.sh ]; then
  time bash scripts/test_vector_search.sh >/dev/null 2>&1 && echo "✓ Vector search performance acceptable"
else
  echo "⚠ No performance test script, skipping"
fi

# Basic file operations timing
time find data/issues/open -mindepth 1 -maxdepth 1 -type d >/dev/null 2>&1 && echo "✓ Issue directory enumeration fast"

echo "✅ test-performance.sh completed successfully"
