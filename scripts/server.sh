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
else 
    echo 'Running migrations'
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