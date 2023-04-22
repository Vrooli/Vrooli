#!/bin/bash

# Directories to match for relative import conversion
dirs=("api" "assets" "components" "forms" "tools" "utils" "views" "styles")

# Loop through all TypeScript and JavaScript files in the src folder
find src -type f \( -iname "*.ts" -o -iname "*.tsx" -o -iname "*.js" -o -iname "*.jsx" \) |
    while read -r file; do
        echo "Processing: $file"

        # Calculate the number of directories up to the src folder
        num_dirs_up=$(awk -F"/" '{print NF-2}' <<<"$file")

        # Build the relative import prefix
        rel_import_prefix=""
        for ((i = 0; i < $num_dirs_up; i++)); do
            rel_import_prefix="../$rel_import_prefix"
        done

        # Loop through the directories to match
        for dir in "${dirs[@]}"; do
            # Replace the absolute imports with relative imports
            sed -i.bak -E "s~import (.*) from '\"$dir~import \1 from '\"$rel_import_prefix$dir~g" "$file"
        done

        # Remove the backup file created by sed
        rm "${file}.bak"
    done

echo "Import paths updated."
