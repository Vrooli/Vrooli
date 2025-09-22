#!/bin/bash
set -euo pipefail

echo "🔍 Checking Secure Document Processing structure compliance"

SCENARIO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." &amp;&amp; pwd )"

# Required directories
required_dirs=(
    "api"
    "ui"
    "cli"
    "initialization"
    ".vrooli"
    "test"
)

for dir in "${required_dirs[@]}"; do
    if [ ! -d "$SCENARIO_DIR/$dir" ]; then
        echo "❌ Missing required directory: $dir"
        exit 1
    fi
    echo "✅ Directory exists: $dir"
done

# Required files
required_files=(
    ".vrooli/service.json"
    "api/go.mod"
    "api/main.go"
    "ui/package.json"
    "ui/server.js"
    "cli/install.sh"
    "cli/secure-document-processing"
    "Makefile"
    "README.md"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$SCENARIO_DIR/$file" ]; then
        echo "❌ Missing required file: $file"
        exit 1
    fi
    echo "✅ File exists: $file"
done

# Check for initialization files
if [ ! -f "initialization/storage/postgres/schema.sql" ]; then
    echo "⚠️  Optional file missing: initialization/storage/postgres/schema.sql"
fi

echo "✅ All structure checks passed"
