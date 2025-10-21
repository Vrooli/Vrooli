#!/bin/bash
set -e

echo "=== Testing Structure ==="

cd ../../api

if ! go vet ./...; then
  echo "❌ Structure (vet) tests failed"
  exit 1
fi

echo "✅ Structure tests passed"
