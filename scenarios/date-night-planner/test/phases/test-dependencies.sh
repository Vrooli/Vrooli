#!/bin/bash
set -e
echo "=== Dependency Tests ==="
# Check required dependencies
command -v go >/dev/null 2>&1 || { echo >&2 "Go is required but not installed."; exit 1; }
command -v curl >/dev/null 2>&1 || { echo >&2 "curl is required but not installed."; exit 1; }
echo "All dependencies satisfied"
exit 0