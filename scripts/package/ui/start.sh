#!/bin/sh

# Check for required variables
if [ -z "$PROJECT_DIR" ]; then
  echo "ERROR: PROJECT_DIR environment variable is not set." >&2
  exit 1
fi
if [ -z "$NODE_ENV" ]; then
  echo "ERROR: NODE_ENV environment variable is not set." >&2
  exit 1
fi

cd "${PROJECT_DIR}/packages/ui"
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to cd to ${PROJECT_DIR}/packages/ui. Current PWD: $(pwd)" >&2
    exit 1
fi

echo 'INFO: Starting app...'
pnpm run start:${NODE_ENV}
echo 'SUCCESS: App started'
