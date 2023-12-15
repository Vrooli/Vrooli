#!/bin/sh
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

# If in development mode
if [ "${NODE_ENV}" = "development" ]; then
    # Convert shared package to typescript. In production, this should already be done
    . "${HERE}/shared.sh"
    # Perform pre-develop steps
    cd ${PROJECT_DIR}/packages/server
    yarn pre-develop
    cd ${PROJECT_DIR}/packages/jobs
    yarn build && yarn pre-develop
    if [ $? -ne 0 ]; then
        error "Failed pre-develop steps"
        exit 1
    fi
else
    # Perform pre-prod steps
    cd ${PROJECT_DIR}/packages/jobs
    yarn pre-prod
    if [ $? -ne 0 ]; then
        error "Failed pre-prod steps"
        exit 1
    fi
fi

info 'Starting jobs...'
cd ${PROJECT_DIR}/packages/jobs
yarn start-${NODE_ENV}
success 'Jobs started'
