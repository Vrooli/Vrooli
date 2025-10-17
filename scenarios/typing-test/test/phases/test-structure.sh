#!/bin/bash
set -euo pipefail

echo "=== Code Structure Verification for Typing Test ==="

# Required directories
required_dirs=(api ui cli .vrooli initialization test test/phases)
missing_dirs=()

for dir in "${required_dirs[@]}"; do
    if [[ ! -d "$dir" ]]; then
        missing_dirs+=("$dir")
    fi
done

if [[ ${#missing_dirs[@]} -gt 0 ]]; then
    echo "❌ Missing directories: ${missing_dirs[*]}"
    exit 1
fi

# Required files
required_files=(
    "api/main.go"
    "ui/index.html"
    "ui/package.json"
    "cli/install.sh"
    ".vrooli/service.json"
    "PRD.md"
    "test/run-tests.sh"
)

missing_files=()

for file in "${required_files[@]}"; do
    if [[ ! -f "$file" ]]; then
        missing_files+=("$file")
    fi
done

if [[ ${#missing_files[@]} -gt 0 ]]; then
    echo "❌ Missing files: ${missing_files[*]}"
    exit 1
else
    echo "✅ All required structure present"
fi

# Check for common issues
if [[ -f api/go.mod ]] &amp;&amp; ! grep -q "module" api/go.mod; then
    echo "⚠️  Potential issue in go.mod"
fi

echo "✅ Structure verification passed"
