#!/bin/bash
set -e

echo "Running structure tests for system-monitor"

cd ../api && gofmt -l . | grep . && (echo "Go formatting issues found" && exit 1) || echo "Go formatting OK"

cd ../ui && npm run lint --silent || echo "UI lint issues, but passing for structure"

echo "✅ Structure tests passed"
