#!/bin/bash
# Sets up NPM, Yarn, global dependencies, and anything else
# required to get the project up and running.
#
# Arguments (all optional):
# -f: Force install (y/N) - If set to "y", will delete all node_modules directories and reinstall
# -r: Run on remote server (y/N) - If set to "y", will run additional commands to set up the remote server
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

# Read arguments
REINSTALL_MODULES=""
ON_REMOTE=""
for arg in "$@"; do
    case $arg in
    -f | --force)
        REINSTALL_MODULES="${2}"
        shift
        shift
        ;;
    -r | --remote)
        ON_REMOTE="${2}"
        shift
        shift
        ;;
    -h | --help)
        echo "Usage: $0 [-h HELP] [-f FORCE] [-r REMOTE]"
        echo "  -h --help: Show this help message"
        echo "  -f --force: (y/N) If set to \"y\", will delete all node_modules directories and reinstall"
        echo "  -r --remote: (Y/n) True if this script is being run on the remote server"
        exit 0
        ;;
    esac
done

header "Checking for package updates"
sudo apt-get update
header "Running upgrade"
RUNLEVEL=1 sudo apt-get -y upgrade

# If this script is being run on the remote server, enable PasswordAuthentication
if [ -z "${ON_REMOTE}" ]; then
    prompt "Is this script being run on the remote server? (Y/n)"
    read -r ON_REMOTE
fi
if [ "${ON_REMOTE}" = "y" ] || [ "${ON_REMOTE}" = "Y" ] || [ "${ON_REMOTE}" = "yes" ] || [ "${ON_REMOTE}" = "Yes" ]; then
    header "Enabling PasswordAuthentication"
    sudo sed -i 's/PasswordAuthentication no/PasswordAuthentication yes/g' /etc/ssh/sshd_config
    sudo sed -i 's/PubkeyAuthentication no/PubkeyAuthentication yes/g' /etc/ssh/sshd_config
    sudo service sshd restart
else
    # Otherwise, make sure mailx is installed. This may be used by some scripts which
    # track errors on the remote server and notify the developer via email.
    header "Installing mailx"
    # TODO - Not working for some reason
    # info "Select option 2 (Internet Site) then enter \"http://mirrors.kernel.org/ubuntu\" when prompted."
    #sudo apt-get install -y mailutils
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
yarn global add apollo@2.34.0 typescript ts-node nodemon prisma@4.11.0 vite

if ! command -v docker &>/dev/null; then
    info "Docker is not installed. Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    # Check if Docker installation failed
    if ! command -v docker &>/dev/null; then
        echo "Error: Docker installation failed."
        exit 1
    fi
else
    info "Detected: $(docker --version)"
fi

if ! command -v docker-compose &>/dev/null; then
    info "Docker Compose is not installed. Installing Docker Compose..."
    sudo curl -L "https://github.com/docker/compose/releases/download/v2.15.1/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    # Check if Docker Compose installation failed
    if ! command -v docker-compose &>/dev/null; then
        echo "Error: Docker Compose installation failed."
        exit 1
    fi
else
    info "Detected: $(docker-compose --version)"
fi

# If reinstalling modules, delete all node_modules directories
if [ -z "${REINSTALL_MODULES}" ]; then
    prompt "Force install node_modules? This will delete all node_modules and the yarn.lock file. (y/N)"
    read -r REINSTALL_MODULES
fi
if [ "${REINSTALL_MODULES}" = "y" ] || [ "${REINSTALL_MODULES}" = "Y" ] || [ "${REINSTALL_MODULES}" = "yes" ] || [ "${REINSTALL_MODULES}" = "Yes" ]; then
    header "Deleting all node_modules directories"
    find "${HERE}/.." -maxdepth 4 -name "node_modules" -type d -exec rm -rf {} \;
    header "Deleting yarn.lock"
    rm "${HERE}/../yarn.lock"
fi
header "Installing local dependencies"
cd "${HERE}/.." && yarn cache clean && yarn

"${HERE}/shared.sh"

# header "Combining node_modules from all packages into one"

header "Generating type models for Prisma"
cd "${HERE}/../packages/server" && yarn prisma-generate

info "Done! You may need to restart your editor for syntax highlighting to work correctly."
info "If you haven't already, copy .env-example to .env and edit it to match your environment."
