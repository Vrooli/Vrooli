#!/usr/bin/env bash
# Test Structure Phase - Validate file and directory structure

set -e

SCENARIO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "Testing Git Control Tower structure..."

# Required files
required_files=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "Makefile"
    "api/main.go"
    "api/go.mod"
    "cli/git-control-tower"
    "initialization/postgres/schema.sql"
)

# Required directories
required_dirs=(
    "api"
    "cli"
    "initialization"
    "initialization/postgres"
    "test"
    "test/phases"
)

# Check required files
for file in "${required_files[@]}"; do
    if [[ ! -f "${SCENARIO_ROOT}/${file}" ]]; then
        echo "❌ Missing required file: ${file}"
        exit 1
    fi
done

# Check required directories
for dir in "${required_dirs[@]}"; do
    if [[ ! -d "${SCENARIO_ROOT}/${dir}" ]]; then
        echo "❌ Missing required directory: ${dir}"
        exit 1
    fi
done

# Check CLI is executable
if [[ ! -x "${SCENARIO_ROOT}/cli/git-control-tower" ]]; then
    echo "❌ CLI is not executable: cli/git-control-tower"
    exit 1
fi

echo "✅ Structure validation passed"
