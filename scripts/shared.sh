#!/bin/sh
HERE=`dirname $0`
source "${HERE}/prettify.sh"

# Loop through shared folder and convert typescript to javascript
info 'Converting shared typescript to javascript'
for package in $(ls -d ${HERE}/../packages/shared/*); do
    info "Converting typescript to javascript in ${package}..."
    cd ${package}
    yarn tsc 
    success "Converted typescript to javascript in ${package}"
    cd ..
done