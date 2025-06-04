#!/bin/sh

# Check for required variables
if [ -z "$PROJECT_DIR" ]; then
  echo "ERROR: PROJECT_DIR environment variable is not set." >&2
  exit 1
fi

cd "${PROJECT_DIR}/packages/shared"

echo "INFO: Converting shared typescript to javascript (running build)"
# Type check, then convert shared packages to javascript
pnpm run build
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to build shared package" >&2
    exit 1
fi
# Add JSON files to dist
echo "INFO: Copying translation files to dist..."
cp -r src/translations/locales dist/translations

echo "SUCCESS: Finished building shared package and copying translations"
