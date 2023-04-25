#!/bin/sh
HERE=$(dirname $0)
source "${HERE}/prettify.sh"

success 'Prisma schema generated'
info 'Waiting for database and redis to start...'
${PROJECT_DIR}/scripts/wait-for.sh db:5432 -t 120 -- echo 'Database is up'
${PROJECT_DIR}/scripts/wait-for.sh redis:6379 -t 60 -- echo 'Redis is up'

cd ${PROJECT_DIR}/packages/main

# Run pre-build script
npm run pre-build

cd server
if [ "${DB_PULL}" = true ]; then
    info 'Generating schema.prisma file from database...'
    npm run prisma-pull
    if [ $? -ne 0 ]; then
        error "Failed to generate schema.prisma file from database"
        exit 1
    fi
    success 'Schema.prisma file generated'
else
    info 'Running migrations...'
    pwd
    ls -la
    npm run prisma-migrate
    if [ $? -ne 0 ]; then
        error "Failed to run migrations"
        exit 1
    fi
    success 'Migrations completed'
fi

info 'Generating Prisma schema...'
npm run prisma-generate
if [ $? -ne 0 ]; then
    error "Failed to generate Prisma schema"
    exit 1
fi
cd ..

info 'Starting main application...'
npm run start-${NODE_ENV}
success 'Main application started'
