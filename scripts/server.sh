#!/bin/sh
HERE=`dirname $0`
source "${HERE}/shared.sh"

# Before backend can start, it must first wait for the database and redis to finish initializing
echo 'Waiting for databas and redis to start...'
${PROJECT_DIR}/scripts/wait-for.sh ${DB_CONN} -t 120 -- echo 'Database is up'
${PROJECT_DIR}/scripts/wait-for.sh ${REDIS_CONN} -t 60 -- echo 'Redis is up'

PRISMA_SCHEMA_FILE="src/db/schema.prisma"

cd ${PROJECT_DIR}/packages/server
if [ "${DB_PULL}" = true ]; then
    echo 'Generating schema.prisma file from database...'
    yarn prisma db pull
    echo 'Schema.prisma file generated'
else 
    echo 'Running migrations...'
    yarn prisma migrate deploy
    echo 'Migrations completed'
fi

echo 'Generating Prisma schema...'
yarn prisma generate
echo 'Prisma schema generated'

echo 'Starting server...'
cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}
echo 'Server started'