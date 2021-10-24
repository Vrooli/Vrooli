#!/bin/sh

# Wait for backend server to start. It could run fine alone, but we don't want a user trying to access the site during this time
${PROJECT_DIR}/scripts/wait-for.sh server:5000 -t 1000 -- echo 'Backend server is up. Starting UI'

cd ${PROJECT_DIR}/packages/ui && PORT=${VIRTUAL_PORT}

# Production build extra steps
if [ "${NODE_ENV}" = "production" ]; then
    # Build project
    yarn build
    # If PRECONNECT set in index.html, convert it to correct url (localhost server port for local testing, and website url for remote deployment)
    ${PROJECT_DIR}/scripts/preconnect.sh
fi

# Finally, start project
yarn start-${NODE_ENV}