#!/bin/bash
# Builds the React app, making sure to include environment 
# variables and post-build commands.
HERE=`dirname $0`
source "${HERE}/prettify.sh"

# Load variables from .env file
if [ -f "${HERE}/../.env" ]; then
    source "${HERE}/../.env"
else
    error "Could not find .env file. This may break the build."
fi

# Navigate to UI directory
cd ${HERE}/../packages/ui

# Create local .env file
touch .env
# Set environment variables
echo "REACT_APP_SERVER_LOCATION=${SERVER_LOCATION}" >> .env
echo "REACT_APP_PORT_SERVER=${PORT_SERVER}" >> .env
echo "REACT_APP_SERVER_URL=${SERVER_URL}" >> .env
echo "REACT_APP_SITE_IP=${SITE_IP}" >> .env
# Set trap to remove .env file on exit
trap "rm .env" EXIT

# Build React app
info "Building React app..."
yarn build

# Use production icons
mv build/prod/* build/

success "Build successful!"