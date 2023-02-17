#!/bin/bash
# Sets up NPM, Yarn, global dependencies, and anything else 
# required to get the project up and running.
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

# Read arguments
while getopts ":r" opt; do
    case $opt in
    r)
        ON_REMOTE=$OPTARG
        ;;
    h)
        echo "Usage: $0 [-h HELP] [-r REMOTE]"
        echo "  -h --help: Show this help message"
        echo "  -r --remote: (Y/n) True if this script is being run on the remote server"
        exit 0
        ;;
    \?)
        echo "Invalid option: -$OPTARG" >&2
        exit 1
        ;;
    :)
        echo "Option -$OPTARG requires an argument." >&2
        exit 1
        ;;
    esac
done

header "Checking for package updates"
sudo apt-get update
header "Running upgrade"
sudo apt-get -y upgrade

# If this script is being run on the remote server, enable PasswordAuthentication
if [ -z "${ON_REMOTE}" ]; then
    prompt "Is this script being run on the remote server? (Y/n)"
    read -r ON_REMOTE
fi
if [ "${ON_REMOTE}" = "y" ] || [ "${ON_REMOTE}" = "Y" ] || [ "${ON_REMOTE}" = "yes" ] || [ "${ON_REMOTE}" = "Yes" ]; then
    header "Enabling PasswordAuthentication"
    sudo sed -i 's/PasswordAuthentication no/PasswordAuthentication yes/g' /etc/ssh/sshd_config
    sudo service sshd restart
fi

header "Installing nvm"
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | bash
source ~/.nvm/nvm.sh

header "Installing Node (includes npm)"
nvm install 16.16.0
nvm alias default v16.16.0

header "Installing Yarn"
npm install -g yarn

header "Installing global dependencies"
yarn global add apollo@2.34.0 typescript ts-node nodemon prisma@4.9.0 react-scripts serve

header "Installing local dependencies"
cd "${HERE}/.." && yarn cache clean && yarn

"${HERE}/shared.sh"

# header "Combining node_modules from all packages into one"

header "Generating type models for Prisma"
cd "${HERE}/../packages/server" && yarn prisma-generate

info "Done! You may need to restart your editor for syntax highlighting to work correctly." 
info "If you haven't already, copy .env-example to .env and edit it to match your environment."