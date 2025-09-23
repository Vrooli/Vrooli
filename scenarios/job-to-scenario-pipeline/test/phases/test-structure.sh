#!/bin/bash
set -euo pipefail

echo "=== Structure Tests ==="

# Verify directory structure
required_dirs=("api" "cli" "ui" "initialization" "data")
for dir in "${required_dirs[@]}"; do
    if [[ -d "$dir" ]]; then
        echo "✓ $dir directory present"
    else
        echo "✗ Missing $dir directory"
        exit 1
    fi
done

required_files=("Makefile" "README.md" ".vrooli/service.json")
for file in "${required_files[@]}"; do
    if [[ -f "$file" ]]; then
        echo "✓ $file present"
    else
        echo "✗ Missing $file"
        exit 1
    fi
done

echo "Structure verified"
exit 0