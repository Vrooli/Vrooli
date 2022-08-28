#!/bin/sh
HERE=`dirname $0`
source "${HERE}/shared.sh"

cd ${PROJECT_DIR}/packages/docs

# Finally, start project
yarn start-${NODE_ENV}