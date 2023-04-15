#!/bin/sh
HERE=$(dirname $0)
source "${HERE}/prettify.sh"

# If in development mode
if [ "${NODE_ENV}" = "development" ]; then
    # Convert shared packages to typescript.
    # In production, this should already be done
    source "${HERE}/shared.sh"
    # Install prisma dependency, in case we need to run scripts to reset/apply migrations
    yarn global add prisma@4.12.0
    yarn global bin
fi

info 'Waiting for database and redis to start...'
${PROJECT_DIR}/scripts/wait-for.sh db:5432 -t 120 -- echo 'Database is up'
${PROJECT_DIR}/scripts/wait-for.sh redis:6379 -t 60 -- echo 'Redis is up'

info 'Starting server...'
cd ${PROJECT_DIR}/packages/server
yarn start-${NODE_ENV}
success 'Server started'
