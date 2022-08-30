#!/bin/bash
HERE=`dirname $0`
source "${HERE}/prettify.sh"

# Loop through shared folder and convert typescript to javascript
header 'Converting shared typescript to javascript'
cd "${HERE}/../packages/shared"
for package in $(ls -d ./*); do
    info "Converting typescript to javascript in ${HERE}/../packages/shared/${package}..."
    cd ${package}
    yarn tsc 
    if [ $? -ne 0 ]; then
        error "Failed to convert typescript to javascript in ${HERE}/../packages/shared/${package}"
        exit 1
    fi
    success "Converted typescript to javascript in ${HERE}/../packages/shared/${package}"
    cd ..
done
cd ../../scripts
success "Finished converting shared typescript to javascript"