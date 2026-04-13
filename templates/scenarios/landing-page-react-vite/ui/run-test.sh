#!/bin/bash
# Test runner that filters out framework's coverage CLI args
# All coverage config is in vite.config.ts

# Filter out coverage.* args since vite.config.ts handles all coverage
filtered_args=()
for arg in "$@"; do
    if [[ ! "$arg" =~ ^--coverage\. ]]; then
        filtered_args+=("$arg")
    fi
done

# Run vitest with filtered args
NODE_ENV=test exec node_modules/.bin/vitest run "${filtered_args[@]}"
