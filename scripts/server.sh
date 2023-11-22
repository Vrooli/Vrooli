#!/bin/sh
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

# If in development mode
if [ "${NODE_ENV}" = "development" ]; then
    # Convert shared packages to typescript.
    # In production, this should already be done
    . "${HERE}/shared.sh"
fi

info "Getting ${NODE_ENV} secrets..."
TMP_FILE=$(mktemp) && { ./getSecrets.sh ${NODE_ENV} ${TMP_FILE} jwt_priv jwt_pub OPENAI_API_KEY DB_PASSWORD ADMIN_WALLET ADMIN_PASSWORD VALYXA_PASSWORD VAPID_PRIVATE_KEY STRIPE_SECRET_KEY STRIPE_WEBHOOK_SECRET SITE_EMAIL_PASSWORD AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY 2>/dev/null && . "$TMP_FILE"; } || echo "Failed to get secrets."
rm "$TMP_FILE"
export DB_URL="postgresql://${DB_USER}:${DB_PASSWORD}@db:5432"

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

info 'Starting server...'
cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}
success 'Server started'
