#!/bin/sh
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

# If in development mode
if [ "${NODE_ENV}" = "development" ]; then
    # Convert shared packages to typescript.
    # In production, this should already be done
    . "${HERE}/shared.sh"
fi

success 'Prisma schema generated'
info 'Waiting for database and redis to start...'
${PROJECT_DIR}/scripts/wait-for.sh db:5432 -t 120 -- echo 'Database is up'
${PROJECT_DIR}/scripts/wait-for.sh redis:6379 -t 60 -- echo 'Redis is up'

# Install prisma dependency
# TODO shouldn't need these 2 lines, since Prisma is added in Dockerfile. But for some reason we do. Otherwise, prisma not found
yarn global add prisma@4.14.0
yarn global bin

cd ${PROJECT_DIR}/packages/server
yarn pre-build-prisma

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
        error "Migration or Server start failed, keeping container running for debug..."
        tail -f /dev/null
    fi
    success 'Migrations completed'
fi

info 'Generating Prisma schema...'
/usr/local/bin/prisma generate
if [ $? -ne 0 ]; then
    error "Failed to generate Prisma schema"
    exit 1
fi

info 'Getting secrets...'
. ${HERE}/getSecrets.sh -e ${NODE_ENV} -s VALYXA_API_KEY -s DB_PASSWORD -s ADMIN_WALLET -s ADMIN_PASSWORD -s VALYXA_PASSWORD -s VAPID_PRIVATE_KEY -s STRIPE_SECRET_KEY -s STRIPE_WEBHOOK_SECRET -s SITE_EMAIL_PASSWORD -s AWS_ACCESS_KEY_ID -s AWS_SECRET_ACCESS_KEY

info 'Starting server...'
cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}
success 'Server started'
