#!/bin/bash
set -euo pipefail

echo "=== Test Structure ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

required_dirs=(
  "api"
  "ui"
  "cli"
  ".vrooli"
  "initialization"
  "initialization/storage"
  "initialization/storage/postgres"
  "test"
  "test/phases"
)

for dir in "${required_dirs[@]}"; do
  if [[ ! -d "${SCENARIO_DIR}/${dir}" ]]; then
    echo "❌ Missing directory: ${dir}"
    exit 1
  fi
done

required_files=(
  "api/main.go"
  "api/chart_renderer.go"
  "ui/index.html"
  ".vrooli/service.json"
  "Makefile"
  "README.md"
  "PRD.md"
  "test/run-tests.sh"
  "cli/chart-generator"
  "initialization/storage/postgres/schema.sql"
)

for file in "${required_files[@]}"; do
  if [[ ! -f "${SCENARIO_DIR}/${file}" ]]; then
    echo "❌ Missing file: ${file}"
    exit 1
  fi
done

echo "✅ Structure tests passed"
