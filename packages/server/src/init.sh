#!/bin/bash

shopt -s nullglob

# Check if necessary directories exist, if not, create them
[ ! -d "models/base" ] && mkdir "models/base"
[ ! -d "models/format" ] && mkdir "models/format"

# Move files, skipping "types.ts"
for file in models/*.ts; do
    filename=$(basename "$file")
    if [[ $filename != "types.ts" ]]; then
        mv "$file" "models/base/$filename"
    fi
done
