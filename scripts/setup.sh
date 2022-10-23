#!/bin/bash
# Sets up NPM, Yarn, global dependencies, and anything else 
# required to get the project up and running.
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

header "Checking for package updates"
sudo apt-get update
header "Running upgrade"
sudo apt-get -y upgrade

header "Installing nvm"
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | bash
source ~/.nvm/nvm.sh

header "Installing Node (includes npm)"
nvm install 16.16.0
nvm alias default v16.16.0

header "Installing Yarn"
npm install -g yarn

header "Installing global dependencies"
yarn global add apollo@2.34.0 typescript ts-node nodemon prisma@3.14.0 react-scripts serve

header "Installing local dependencies"
cd "${HERE}/.." && yarn cache clean && yarn

"${HERE}/shared.sh"

# header "Combining node_modules from all packages into one"

header "Generating type models for Prisma"
cd "${HERE}/../packages/server" && yarn prisma-generate

info "Done! You may need to restart your editor for syntax highlighting to work correctly." 
info "If you haven't already, copy .env-example to .env and edit it to match your environment."