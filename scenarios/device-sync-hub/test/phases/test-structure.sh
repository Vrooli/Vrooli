#!/bin/bash
# Structure Tests for Device Sync Hub
# Validates required files and directory structure

set -euo pipefail

SCENARIO_ROOT="${PWD}"

echo "=== Device Sync Hub Structure Tests ==="
echo "[INFO] Validating scenario structure..."

# Required files
required_files=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "PROBLEMS.md"
    "Makefile"
    "api/main.go"
    "api/device-sync-hub-api"
    "cli/device-sync-hub"
    "cli/install.sh"
    "ui/index.html"
    "ui/app.js"
    "ui/server.js"
    "initialization/postgres/schema.sql"
)

missing_files=0
for file in "${required_files[@]}"; do
    if [ -f "$SCENARIO_ROOT/$file" ] || [ -d "$SCENARIO_ROOT/$file" ]; then
        echo "[PASS] Found: $file"
    else
        echo "[FAIL] Missing: $file"
        missing_files=$((missing_files + 1))
    fi
done

# Required directories
required_dirs=(
    "api"
    "cli"
    "ui"
    "test"
    "test/phases"
    "initialization"
    "initialization/postgres"
    "data"
)

missing_dirs=0
for dir in "${required_dirs[@]}"; do
    if [ -d "$SCENARIO_ROOT/$dir" ]; then
        echo "[PASS] Found directory: $dir"
    else
        echo "[FAIL] Missing directory: $dir"
        missing_dirs=$((missing_dirs + 1))
    fi
done

# Validate service.json schema compliance
if [ -f "$SCENARIO_ROOT/.vrooli/service.json" ]; then
    echo "[INFO] Checking service.json structure..."
    if command -v jq >/dev/null 2>&1; then
        if jq empty "$SCENARIO_ROOT/.vrooli/service.json" 2>/dev/null; then
            echo "[PASS] service.json is valid JSON"
        else
            echo "[FAIL] service.json has JSON syntax errors"
            exit 1
        fi
    fi
fi

echo ""
if [ "$missing_files" -eq 0 ] && [ "$missing_dirs" -eq 0 ]; then
    echo "[PASS] All required files and directories present"
    exit 0
else
    echo "[FAIL] Missing $missing_files files and $missing_dirs directories"
    exit 1
fi
