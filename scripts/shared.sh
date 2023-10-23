#!/bin/bash
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

# Loop through shared folder and convert typescript to javascript
header 'Converting shared typescript to javascript'
cd "${HERE}/../packages/shared"
yarn tsc
if [ $? -ne 0 ]; then
    error "Failed to convert shared typescript to javascript"
    exit 1
fi
cd ../../scripts
success "Finished converting shared typescript to javascript"
