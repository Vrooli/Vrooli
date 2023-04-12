#!/bin/sh
HERE=$(dirname $0)
source "${HERE}/prettify.sh"

# If in development mode, convert shared packages to typescript
# In production, this should already be done
if [ "${NODE_ENV}" = "development" ]; then
    source "${HERE}/shared.sh"
fi

info 'Waiting for database and redis to start...'
${PROJECT_DIR}/scripts/wait-for.sh redis:5432 -t 120 -- echo 'Database is up'
# ${PROJECT_DIR}/scripts/wait-for.sh ${REDIS_CONN} -t 60 -- echo 'Redis is up'

PRISMA_SCHEMA_FILE="src/db/schema.prisma"

# TODO shouldn't need these 2 lines, but for some reason we do. Otherwise, prisma not found
yarn global add prisma@4.11.0
yarn global bin

cd ${PROJECT_DIR}/packages/server
if [ "${DB_PULL}" = true ]; then
    info 'Generating schema.prisma file from database...'
    /usr/local/bin/prisma db pull
    if [ $? -ne 0 ]; then
        error "Failed to generate schema.prisma file from database"
        exit 1
    fi
    success 'Schema.prisma file generated'
else
    info 'Running migrations...'
    /usr/local/bin/prisma migrate deploy
    if [ $? -ne 0 ]; then
        error "Failed to run migrations"
        exit 1
    fi
    success 'Migrations completed'
fi

info 'Generating Prisma schema...'
/usr/local/bin/prisma generate
if [ $? -ne 0 ]; then
    error "Failed to generate Prisma schema"
    exit 1
fi
success 'Prisma schema generated'

info 'Starting server...'
cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}
success 'Server started'
