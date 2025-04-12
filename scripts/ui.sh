#!/bin/sh
HERE=$(dirname $0)
. "${HERE}/utils.sh"

cd ${PROJECT_DIR}/packages/ui
info 'Starting app...'
yarn start-${NODE_ENV}
success 'App started'
