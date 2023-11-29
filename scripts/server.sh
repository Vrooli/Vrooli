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

# Install prisma dependency
# TODO shouldn't need these 2 lines, since Prisma is added in Dockerfile. But for some reason we do. Otherwise, prisma not found
yarn global add prisma@4.14.0
yarn global bin

cd ${PROJECT_DIR}/packages/server
yarn pre-develop && yarn build
if [ $? -ne 0 ]; then
    error "Failed to build server"
    exit 1
fi

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

info 'Starting server...'
cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}
success 'Server started'
