#!/bin/bash
# Wrapper script to handle vitest coverage args from framework
# pnpm interprets args as pnpm options unless we skip them entirely
# All coverage config is handled by vitest.config.ts

echo "DEBUG: Called with args: $@" >&2

# Just run vitest with basic args, ignoring framework's coverage args
NODE_ENV=test exec node_modules/.bin/vitest run --coverage --silent
