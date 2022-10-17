#!/bin/sh
HERE=`dirname $0`
source "${HERE}/prettify.sh"

# If in development mode, convert shared packages to typescript
# In production, this should already be done
if [ "${NODE_ENV}" = "development" ]; then
    source "${HERE}/shared.sh"
fi 

# Wait for backend server to start. It could run fine alone, but we don't want a user trying to access the site during this time
info 'Waiting for backend server to start...'
${PROJECT_DIR}/scripts/wait-for.sh server:5329 -t 1000 -- echo 'Backend server is up'

cd ${PROJECT_DIR}/packages/ui

# Finally, start project
info 'Starting app...'
yarn start-${NODE_ENV}
success 'App started'