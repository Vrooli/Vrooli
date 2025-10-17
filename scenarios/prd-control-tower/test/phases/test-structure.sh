#!/usr/bin/env bash
#
# Structure Test: Validate PRD Control Tower project structure
#

set -eo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

echo "🔍 Testing PRD Control Tower structure..."

# Required directories
required_dirs=(
    "api"
    "cli"
    "ui"
    "ui/src"
    "ui/src/pages"
    "test/phases"
    "initialization/postgres"
    "data/prd-drafts/scenario"
    "data/prd-drafts/resource"
    ".vrooli"
)

# Required files
required_files=(
    "PRD.md"
    "README.md"
    "Makefile"
    ".vrooli/service.json"
    "api/go.mod"
    "api/main.go"
    "cli/prd-control-tower"
    "cli/install.sh"
    "ui/package.json"
    "ui/vite.config.ts"
    "ui/index.html"
    "ui/src/main.tsx"
    "initialization/postgres/schema.sql"
)

# Check directories
echo "  Checking directories..."
for dir in "${required_dirs[@]}"; do
    if [[ ! -d "$dir" ]]; then
        echo "    ✗ Missing directory: $dir"
        exit 1
    fi
done

# Check files
echo "  Checking required files..."
for file in "${required_files[@]}"; do
    if [[ ! -f "$file" ]]; then
        echo "    ✗ Missing file: $file"
        exit 1
    fi
done

# Check executables
echo "  Checking executable permissions..."
if [[ ! -x "cli/prd-control-tower" ]]; then
    echo "    ✗ CLI not executable: cli/prd-control-tower"
    exit 1
fi
if [[ ! -x "cli/install.sh" ]]; then
    echo "    ✗ Install script not executable: cli/install.sh"
    exit 1
fi

echo "✅ Structure test passed"
