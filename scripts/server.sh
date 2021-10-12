#!/bin/sh

# Before backend can start, it must first wait for the database and redis to finish initializing
${PROJECT_DIR}/scripts/wait-for.sh ${DB_CONN} -t 1000 -- echo 'Database is up'
${PROJECT_DIR}/scripts/wait-for.sh ${REDIS_CONN} -t 1000 -- echo 'Redis is up'
echo 'Starting backend...'

cd ${PROJECT_DIR}/packages/server
if [ "${DB_PULL}" = true ]; then
    echo 'Generating schema.prisma file from database'
    prisma db pull --schema src/prisma/schema.prisma
fi
if [ "${DB_PUSH}" = true ]; then
    echo 'Updating database to match schema.prisma file'
    prisma db push --schema src/prisma/schema.prisma
fi
if [[ -n "${NEW_MIGRATION_STRING// /}" ]]; then
    echo 'Creating new migration file from schema.prisma'
    prisma migrate dev --${NEW_MIGRATION_STRING} --schema src/prisma/schema.prisma
fi
if [ "${NODE_ENV}" = "production" ]; then
    echo 'Environment is set to production, so migrating to latest database'
    prisma migrate deploy --schema src/prisma/schema.prisma
fi
echo 'Ensuring database is populated with minimal data'
node src/db/seeds/init.js --esm
if [ "${CREATE_MOCK_DATA}" = true ]; then
    echo 'Populating database with mock data'
    node src/db/seeds/mock.js --esm
fi
echo 'Generating Prisma schema'
prisma generate --schema src/prisma/schema.prisma

# Clean any unused files
yarn clean

cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}