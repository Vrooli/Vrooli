#!/bin/sh
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

# Make sure @codemirror is copied to the public directory
"${HERE}/codemirror.sh"

cd ${PROJECT_DIR}/packages/ui
info 'Starting app...'
yarn start-${NODE_ENV}
success 'App started'
