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

# If in development mode
if [ "${NODE_ENV}" = "development" ]; then
    echo "INFO: Development mode detected. Ensuring shared packages are built..."
    # Rebuild shared package as source is mounted in dev
    (cd "${PROJECT_DIR}" && pnpm --filter @vrooli/shared run build)
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to build shared packages for jobs. Keeping container running for debug..." >&2
        # Consider if tail -f is needed here or if exit 1 is better depending on CI/CD
    fi

    echo "INFO: Performing server pre-develop steps (dependency for jobs)..."
    (cd "${PROJECT_DIR}/packages/server" && pnpm run pre-develop)
    # Add error handling if the above step is critical

    echo "INFO: Performing jobs pre-develop steps..."
    cd "${PROJECT_DIR}/packages/jobs"
    pnpm run build && pnpm run pre-develop
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed jobs build or pre-develop steps" >&2
        exit 1
    fi
else
    echo "INFO: Production mode detected. Performing jobs pre-prod steps..."
    cd "${PROJECT_DIR}/packages/jobs"
    pnpm run pre-prod
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed pre-prod steps" >&2
        exit 1
    fi
fi

echo 'INFO: Starting jobs...'
cd "${PROJECT_DIR}/packages/jobs"
pnpm run start:${NODE_ENV}
echo 'SUCCESS: Jobs started'
