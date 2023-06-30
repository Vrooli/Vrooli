#!/bin/bash
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

# Loop through shared folder and convert typescript to javascript
header 'Converting shared typescript to javascript'
cd "${HERE}/../packages/shared"
yarn tsc
cd ../../scripts
success "Finished converting shared typescript to javascript"
