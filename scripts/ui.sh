#!/bin/sh
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

# If in development mode, convert shared packages to typescript
# In production, this should already be done
if [ "${NODE_ENV}" = "development" ]; then
    "${HERE}/shared.sh"
fi

cd ${PROJECT_DIR}/packages/ui

# Finally, start project
info 'Starting app...'
yarn start-${NODE_ENV}
success 'App started'
