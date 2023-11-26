#!/bin/bash
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

cd "${HERE}/../packages/shared"

header 'Converting shared typescript to javascript'
# Type check, then convert shared packages to javascript
yarn tsc --noEmit && yarn swc src -d dist
if [ $? -ne 0 ]; then
    error "Failed to convert shared typescript to javascript"
    exit 1
fi
# Add JSON files to dist
cp -r src/translations/locales dist/translations
# Modify the compiled JS files to add import assertions for JSON files.
# Adding `"importAssertions": true` to the swc config isn't working, so we have to do it manually
header 'Adding import assertions for JSON files'
find dist -type f -name '*.js' -exec sed -i'' -e 's/from "\(.*\.json\)";/from "\1" assert { type: "json" };/g' {} +

cd ../../scripts
success "Finished converting shared typescript to javascript"
