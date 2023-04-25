#!/bin/sh
HERE=$(dirname $0)
source "${HERE}/prettify.sh"

success 'Prisma schema generated'
info 'Waiting for database and redis to start...'
${PROJECT_DIR}/scripts/wait-for.sh db:5432 -t 120 -- echo 'Database is up'
${PROJECT_DIR}/scripts/wait-for.sh redis:6379 -t 60 -- echo 'Redis is up'

# Install prisma dependency
# TODO shouldn't need these 2 lines, since Prisma is added in Dockerfile. But for some reason we do. Otherwise, prisma not found
yarn global add prisma@4.12.0
yarn global bin

cd ${PROJECT_DIR}/packages/main
yarn pre-build

cd server
if [ "${DB_PULL}" = true ]; then
    info 'Generating schema.prisma file from database...'
    yarn prisma-pull
    if [ $? -ne 0 ]; then
        error "Failed to generate schema.prisma file from database"
        exit 1
    fi
    success 'Schema.prisma file generated'
else
    info 'Running migrations...'
    yarn prisma-migrate
    if [ $? -ne 0 ]; then
        error "Failed to run migrations"
        exit 1
    fi
    success 'Migrations completed'
fi

info 'Generating Prisma schema...'
yarn prisma-generate
if [ $? -ne 0 ]; then
    error "Failed to generate Prisma schema"
    exit 1
fi

info 'Starting server...'
yarn start-${NODE_ENV}
success 'Server started'
