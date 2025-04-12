#!/bin/sh
HERE=$(dirname $0)
start_time=$(date +%s)
. "${HERE}/utils.sh"

# If in development mode
if [ "${NODE_ENV}" = "development" ]; then
    # Convert shared packages to typescript. In production, this should already be done
    "${HERE}/shared.sh"
    # Perform pre-develop steps
    cd ${PROJECT_DIR}/packages/server
    yarn build && yarn pre-develop
    if [ $? -ne 0 ]; then
        error "Failed pre-develop steps. Could be db migration issue. Keeping container running for debug..."
        tail -f /dev/null
    fi
else
    # Perform pre-prod steps
    cd ${PROJECT_DIR}/packages/server
    yarn pre-prod
    if [ $? -ne 0 ]; then
        error "Failed pre-prod steps"
        exit 1
    fi
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

end_time=$(date +%s)
duration=$((end_time - start_time))
info "Server setup time: $duration seconds"

info 'Starting server...'
cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}
