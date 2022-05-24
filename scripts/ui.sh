#!/bin/sh

# Wait for backend server to start. It could run fine alone, but we don't want a user trying to access the site during this time
${PROJECT_DIR}/scripts/wait-for.sh server:5329 -t 1000 -- echo 'Backend server is up. Starting app'

cd ${PROJECT_DIR}/packages/ui

# Development build extra steps
if [ "${NODE_ENV}" = "development" ]; then
    # Replace favicon images with the development version. 
    # This makes it easier to see which downloaded version of the site is which. 
    # NOTE: This does not specify if the build is run locally or on a server. If you really 
    # want to get fancy with it, you could make a set of images for these cases.
    mv ${PROJECT_DIR}/packages/ui/public/dev/* ${PROJECT_DIR}/packages/ui/public/
fi

# Finally, start project
yarn start-${NODE_ENV}