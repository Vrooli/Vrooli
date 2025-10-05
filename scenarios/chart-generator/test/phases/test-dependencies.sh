#!/bin/bash
set -euo pipefail

echo "=== Test Dependencies ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Check Go dependencies
cd "${SCENARIO_DIR}/api"
if ! go mod verify; then
  echo "❌ Go module verification failed"
  exit 1
fi

# Check for required Go packages
required_packages=(
  "github.com/gorilla/mux"
  "github.com/lib/pq"
)

for pkg in "${required_packages[@]}"; do
  if ! grep -q "$pkg" go.mod; then
    echo "❌ Missing required package: $pkg"
    exit 1
  fi
done

# Check CLI exists and is executable
if [[ ! -x "${SCENARIO_DIR}/cli/chart-generator" ]]; then
  echo "❌ CLI binary not found or not executable"
  exit 1
fi

# Check CLI responds to help
if ! "${SCENARIO_DIR}/cli/chart-generator" help &>/dev/null; then
  echo "❌ CLI help command failed"
  exit 1
fi

echo "✅ Dependency tests passed"
