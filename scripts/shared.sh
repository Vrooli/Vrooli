#!/bin/sh
HERE=`dirname $0`

# Loop through shared folder and convert typescript to javascript
echo 'Converting shared typescript to javascript'
for package in $(ls -d ${HERE}/../packages/shared/*); do
    echo "Converting ${package}"
    cd ${package}
    yarn tsc 
    cd ..
done