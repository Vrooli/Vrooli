#!/bin/sh
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

# If in development mode
if [ "${NODE_ENV}" = "development" ]; then
    # Convert shared packages to typescript. In production, this should already be done
    . "${HERE}/shared.sh"
fi

echo "DB_USER is ${DB_USER}"
echo "DB_PASSWORD is ${DB_PASSWORD}"
echo "JWT_PRIV is ${JWT_PRIV}"
echo "DB_URL is ${DB_URL}"


cd ${PROJECT_DIR}/packages/server
yarn pre-develop && yarn build
if [ $? -ne 0 ]; then
    error "Failed to build server"
    exit 1
fi

if [ "${DB_PULL}" = true ]; then
    info 'Generating schema.prisma file from database...'
    yarn prisma db pull
    if [ $? -ne 0 ]; then
        error "Failed to generate schema.prisma file from database"
        exit 1
    fi
    success 'Schema.prisma file generated'
else
    info 'Running migrations...'
    yarn prisma migrate deploy
    if [ $? -ne 0 ]; then
        error "Migration or Server start failed, keeping container running for debug..."
        tail -f /dev/null
    fi
    success 'Migrations completed'
fi

info 'Generating Prisma schema...'
yarn prisma generate
if [ $? -ne 0 ]; then
    error "Failed to generate Prisma schema"
    exit 1
fi

info 'Starting server...'
cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}
success 'Server started'
