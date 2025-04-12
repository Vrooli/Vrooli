#!/bin/sh
HERE=$(dirname $0)
. "${HERE}/utils.sh"

cd "${HERE}/../packages/shared"

header 'Converting shared typescript to javascript'
# Type check, then convert shared packages to javascript
yarn build
if [ $? -ne 0 ]; then
    error "Failed to convert shared typescript to javascript"
    exit 1
fi
# Add JSON files to dist
cp -r src/translations/locales dist/translations

cd "${HERE}"
success "Finished converting shared typescript to javascript"
