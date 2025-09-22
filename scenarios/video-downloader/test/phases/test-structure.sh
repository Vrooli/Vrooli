#!/bin/bash

set -euo pipefail

printf "=== Test Structure ===\\n"

# Check for required directories and files

required_dirs=("api" "ui" ".vrooli")

for dir in "${required_dirs[@]}"; do

  if [[ ! -d "$dir" ]]; then

    printf "❌ Missing directory: %s\\n" "$dir"

    exit 1

  fi

done

required_files=("api/go.mod" "ui/package.json" ".vrooli/service.json")

for file in "${required_files[@]}"; do

  if [[ ! -f "$file" ]]; then

    printf "❌ Missing file: %s\\n" "$file"

    exit 1

  fi

done

printf "✅ Structure tests passed\\n"