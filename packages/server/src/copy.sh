#!/bin/bash

shopt -s nullglob

for file in models/base/*.ts; do
    filename=$(basename "$file")

    # Skip "index.ts"
    if [[ $filename != "index.ts" ]]; then
        # Create a corresponding file in "models/format"
        touch "models/format/$filename"

        # Extract all import lines and modify relative imports
        grep '^import' "$file" | sed 's/from "\.\./from "..\/\.\./g' >"models/format/$filename"
    fi
done
