#!/bin/bash
set -euo pipefail

echo "=== Structure Tests Phase for Vrooli Assistant ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Check required directories
required_dirs=("api" "ui" "cli" ".vrooli")
missing_dirs=()
for dir in "${required_dirs[@]}"; do
  if [ ! -d "$SCENARIO_DIR/$dir" ]; then
    missing_dirs+=("$dir")
  fi
done

if [ ${#missing_dirs[@]} -eq 0 ]; then
  echo "✅ All required directories present"
else
  echo "❌ Missing directories: ${missing_dirs[*]}"
  exit 1
fi

# Check required files (customized for vrooli-assistant structure)
required_files=(".vrooli/service.json" "api/go.mod" "ui/electron/package.json" "cli/vrooli-assistant")
missing_files=()
for file in "${required_files[@]}"; do
  if [ ! -f "$SCENARIO_DIR/$file" ]; then
    missing_files+=("$file")
  fi
done

if [ ${#missing_files[@]} -eq 0 ]; then
  echo "✅ All required files present"
else
  echo "❌ Missing files: ${missing_files[*]}"
  exit 1
fi

# Check Makefile if present
if [ -f "$SCENARIO_DIR/Makefile" ]; then
  if grep -q "run\|test\|stop" "$SCENARIO_DIR/Makefile"; then
    echo "✅ Makefile has basic lifecycle targets"
  else
    echo "⚠️  Makefile missing lifecycle targets"
  fi
fi

echo "✅ Structure tests phase completed"
