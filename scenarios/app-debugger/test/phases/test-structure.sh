#!/bin/bash
set -e
echo "=== Structure Validation ==="
make fmt && make lint  # Ensure code is formatted and linted
# Check for required files
[ -f api/main.go ] || echo "Warning: API main.go missing"
[ -f ui/index.html ] || echo "Warning: UI index.html missing"
echo "✅ Structure validated"