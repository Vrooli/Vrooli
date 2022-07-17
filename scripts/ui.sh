#!/bin/sh

# Wait for backend server to start. It could run fine alone, but we don't want a user trying to access the site during this time
${PROJECT_DIR}/scripts/wait-for.sh server:5329 -t 1000 -- echo 'Backend server is up. Starting app'

cd ${PROJECT_DIR}/packages/ui

# Determine which favicons to use TODO this doesn't work, and doesn't account for build folder (which should be checked if it exists if in production)
# Use dev version if NODE_ENV is development or REACT_APP_SERVER_LOCATION is local. 
# User prod version otherwise
if [ "${NODE_ENV}" = "development" ] || [ "${REACT_APP_SERVER_LOCATION}" = "local" ]; then
    cp -p ${PROJECT_DIR}/packages/ui/public/dev/* ${PROJECT_DIR}/packages/ui/public/
    echo "Using development favicons"
else
    cp -p ${PROJECT_DIR}/packages/ui/public/prod/* ${PROJECT_DIR}/packages/ui/public/
    echo "Using production favicons"
fi

# Finally, start project
yarn start-${NODE_ENV}