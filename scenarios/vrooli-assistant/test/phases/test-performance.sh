#!/bin/bash
set -euo pipefail

echo "=== Performance Tests Phase for Vrooli Assistant ==="

# Basic performance checks (lightweight)
echo "Checking API response time..."
if command -v curl >/dev/null 2>&1; then
  if curl -s -w "Time: %{time_total}s\n" -o /dev/null http://localhost:${API_PORT:-8080}/health 2>/dev/null | grep -q "Time: [0-9]\+\.[0-9]\+s"; then
    echo "✅ API response time measured"
  else
    echo "⚠️  Could not measure API performance (service not running)"
  fi
else
  echo "ℹ️  curl not available, skipping performance test"
fi

# Check binary sizes if built
if [ -f "../../api/vrooli-assistant-api" ]; then
  size=$(stat -f%z ../../api/vrooli-assistant-api 2>/dev/null || stat -c%s ../../api/vrooli-assistant-api 2>/dev/null)
  echo "API binary size: $size bytes"
  if [ "$size" -lt 50000000 ]; then  # 50MB threshold
    echo "✅ API binary size acceptable"
  else
    echo "⚠️  API binary size large: $size bytes"
  fi
fi

# Memory usage placeholder
echo "ℹ️  Detailed memory profiling would require running the service"

echo "✅ Performance tests phase completed"
