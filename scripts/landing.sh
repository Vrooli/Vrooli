#!/bin/sh

# Wait for backend server to start. It could run fine alone, but we don't want a user trying to access the site during this time
${PROJECT_DIR}/scripts/wait-for.sh server:3000 -t 1000 -- echo 'Backend server is up. Starting UI'

cd ${PROJECT_DIR}/packages/landing && PORT=${VIRTUAL_PORT}

# Finally, start project
yarn start-${NODE_ENV}