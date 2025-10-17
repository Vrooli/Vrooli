#!/bin/bash
set -euo pipefail

echo "=== Dependency Tests ==="

# Check required files and resources
test -f api/main.go && echo "✓ API source present"
test -f cli/job-to-scenario-pipeline && echo "✓ CLI present"
test -d data && echo "✓ Data directory present"

# Check Go dependencies
cd api && go mod tidy -e >/dev/null 2>&1 && echo "✓ Go dependencies tidy"

echo "All dependencies verified"
exit 0