#!/bin/sh

# Wait for backend server to start. It could run fine alone, but we don't want a user trying to access the site during this time
${PROJECT_DIR}/scripts/wait-for.sh server:5329 -t 1000 -- echo 'Backend server is up. Starting app'

cd ${PROJECT_DIR}/packages/ui

# Production build extra steps
# if [ "${NODE_ENV}" = "production" ]; then
#     # Build project
#     yarn build
# fi

# Finally, start project
yarn start-${NODE_ENV}