#!/bin/sh
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

# If in development mode, convert shared packages to typescript
# In production, this should already be done
if [ "${NODE_ENV}" = "development" ]; then
    source "${HERE}/shared.sh"
fi 

cd ${PROJECT_DIR}/packages/docs

# Finally, start project
info 'Starting docs...'
yarn start-${NODE_ENV}
success 'Docs started'