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
    # Assuming PROJECT_DIR is the monorepo root and pnpm is available
    # Rebuild shared package as source is mounted in dev
    (cd "${PROJECT_DIR}" && pnpm --filter @vrooli/shared run build)
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to build shared packages. Keeping container running so you can debug..." >&2
        tail -f /dev/null
    fi
fi

echo "INFO: Building server..."
cd "${PROJECT_DIR}/packages/server"

# Pass environment variables explicitly to ensure they're available to Prisma
DB_URL="${DB_URL}" REDIS_URL="${REDIS_URL}" pnpm run build && \
DB_URL="${DB_URL}" REDIS_URL="${REDIS_URL}" pnpm run prisma:migrate && \
DB_URL="${DB_URL}" REDIS_URL="${REDIS_URL}" pnpm run prisma:generate
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to build server. Keeping container running so you can debug..." >&2
    tail -f /dev/null
fi

echo 'INFO: Starting server...'
cd "${PROJECT_DIR}/packages/server"
# Pass environment variables explicitly including seeding variables
DB_URL="${DB_URL}" \
REDIS_URL="${REDIS_URL}" \
NODE_ENV="${NODE_ENV}" \
ADMIN_WALLET="${ADMIN_WALLET}" \
ADMIN_PASSWORD="${ADMIN_PASSWORD}" \
SITE_EMAIL_USERNAME="${SITE_EMAIL_USERNAME}" \
VALYXA_PASSWORD="${VALYXA_PASSWORD}" \
pnpm run start-${NODE_ENV}
