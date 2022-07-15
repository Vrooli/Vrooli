#!/bin/sh

# Before backend can start, it must first wait for the database and redis to finish initializing
${PROJECT_DIR}/scripts/wait-for.sh ${DB_CONN} -t 120 -- echo 'Database is up'
${PROJECT_DIR}/scripts/wait-for.sh ${REDIS_CONN} -t 60 -- echo 'Redis is up'
echo 'Starting backend...'

PRISMA_SCHEMA_FILE="src/db/schema.prisma"

cd ${PROJECT_DIR}/packages/server
if [ "${DB_PULL}" = true ]; then
    echo 'Generating schema.prisma file from database'
    prisma db pull
fi
if [ "${DB_PUSH}" = true ]; then
    echo 'Updating database to match schema.prisma file'
    prisma db push
fi
# If production and database migrations exist, migrate to latest
if [ "${NODE_ENV}" = "production" ] && [ "$(ls -A src/db/migrations)" ]; then
    echo 'Environment is set to production, so migrating to latest database'
    prisma migrate deploy
fi
echo 'Generating Prisma schema'
prisma generate

echo 'Converting shared directory to javascript'
cd ${PROJECT_DIR}/packages/shared
yarn build

echo 'Starting server'
cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}