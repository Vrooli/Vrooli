#!/bin/sh

# Before backend can start, it must first wait for the database and redis to finish initializing
${PROJECT_DIR}/scripts/wait-for.sh ${DB_CONN} -t 1000 -- echo 'Database is up'
${PROJECT_DIR}/scripts/wait-for.sh ${REDIS_CONN} -t 1000 -- echo 'Redis is up'
echo 'Starting backend...'

cd ${PROJECT_DIR}/packages/server
echo 'Migrating to latest database'
knex migrate:latest --env development --knexfile ./src/db/knexfile.js --esm
echo 'Ensuring database is populated with minimal data'
knex seed:run --knexfile ./src/db/knexfile.js --specific init.js --esm
if [ "${CREATE_MOCK_DATA}" = true ]; then
    echo 'Populating database with mock data'
    knex seed:run --knexfile ./src/db/knexfile.js --specific mock.js --esm
fi
if [ "${PRISMA_INTROSPECT}" = true ]; then
    echo 'Generating schema.prisma file from database'
    prisma introspect --schema src/prisma/schema.prisma && prisma generate --schema src/prisma/schema.prisma
fi
echo 'Generating Prisma schema'
prisma generate --schema src/prisma/schema.prisma

# Clean any unused files
yarn clean

cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}