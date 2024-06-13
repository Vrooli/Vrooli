#!/bin/bash
# Sets up NPM, Yarn, global dependencies, and anything else
# required to get the project up and running.
ORIGINAL_DIR=$(pwd)
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Default values
REINSTALL_MODULES=""
ON_REMOTE=""
ENVIRONMENT=$NODE_ENV
ENV_FILES_SET_UP=""
USE_KUBERNETES=false

# Read arguments
for arg in "$@"; do
    case $arg in
    -h | --help)
        echo "Usage: $0 [-h HELP] [-e ENV_SETUP] [-k KUBERNETES] [-m MODULES_REINSTALL] [-p PROD] [-r REMOTE]"
        echo "  -h --help: Show this help message"
        echo "  -e --env-setup: (Y/n) True if you want to create secret files for the environment variables. If not provided, will prompt."
        echo "  -k --kubernetes: If set, will use Kubernetes instead of Docker Compose"
        echo "  -m --modules-reinstall: (y/N) If set to \"y\", will delete all node_modules directories and reinstall"
        echo "  -p --prod: If set, will skip steps that are only required for development"
        echo "  -r --remote: (Y/n) True if this script is being run on a remote server"
        exit 0
        ;;
    -e | --env-setup)
        ENV_FILES_SET_UP="${2}"
        shift
        shift
        ;;
    -k | --kubernetes)
        USE_KUBERNETES=true
        shift
        ;;
    -m | --modules-reinstall)
        REINSTALL_MODULES="${2}"
        shift
        shift
        ;;
    -p | --prod)
        ENVIRONMENT="production"
        shift
        ;;
    -r | --remote)
        ON_REMOTE="${2}"
        shift
        shift
        ;;
    esac
done

header "Making sure the system clock is accurate"
sudo hwclock -s
info "System clock is now: $(date)"

# Limit the number of apt-get update and upgrade calls
should_run_apt_get_update() {
    local last_update=$(stat -c %Y /var/lib/apt/lists/)
    local current_time=$(date +%s)
    local update_interval=$((24 * 60 * 60)) # 24 hours

    if ((current_time - last_update > update_interval)); then
        return 0 # true, should run
    else
        return 1 # false, should not run
    fi
}
should_run_apt_get_upgrade() {
    local last_upgrade=$(stat -c %Y /var/lib/dpkg/status)
    local current_time=$(date +%s)
    local upgrade_interval=$((7 * 24 * 60 * 60)) # 1 week

    if ((current_time - last_upgrade > upgrade_interval)); then
        return 0 # true, should run
    else
        return 1 # false, should not run
    fi
}

# Check and run apt-get update and upgrade
if should_run_apt_get_update; then
    header "Updating apt-get package lists"
    sudo apt-get update
else
    info "Skipping apt-get update - last update was less than 24 hours ago"
fi
if should_run_apt_get_upgrade; then
    header "Upgrading apt-get packages"
    RUNLEVEL=1 sudo apt-get -y upgrade
else
    info "Skipping apt-get upgrade - last upgrade was less than 1 week ago"
fi

header "Setting script permissions"
chmod +x "${HERE}/"*.sh

# If this script is being run on a remote server, enable PasswordAuthentication
if [ -z "${ON_REMOTE}" ]; then
    prompt "Is this script being run on a remote server? (Y/n)"
    read -n1 -r ON_REMOTE
    echo
fi
if [[ "$ON_REMOTE" =~ ^[Yy]([Ee][Ss])?$ ]]; then
    header "Enabling PasswordAuthentication"
    sudo sed -i 's/#\?PasswordAuthentication .*/PasswordAuthentication yes/g' /etc/ssh/sshd_config
    sudo sed -i 's/#\?PubkeyAuthentication .*/PubkeyAuthentication yes/g' /etc/ssh/sshd_config
    sudo sed -i 's/#\?AuthorizedKeysFile .*/AuthorizedKeysFile .ssh\/authorized_keys/g' /etc/ssh/sshd_config
    if [ ! -d ~/.ssh ]; then
        mkdir ~/.ssh
        chmod 700 ~/.ssh
    fi
    if [ ! -f ~/.ssh/authorized_keys ]; then
        touch ~/.ssh/authorized_keys
    fi
    chmod 600 ~/.ssh/authorized_keys
    # Try restarting service. Can either be called "sshd" or "ssh"
    sudo service sshd restart
    # If sshd fails, try to restart ssh
    if [ $? -ne 0 ]; then
        warning "Failed to restart sshd, trying ssh..."
        sudo systemctl restart ssh
        # If ssh also fails, exit with an error
        if [ $? -ne 0 ]; then
            echo "Failed to restart ssh. Exiting with error."
            cd "$ORIGINAL_DIR"
            exit 1
        fi
    fi
else
    # Otherwise, make sure mailx is installed. This may be used by some scripts which
    # track errors on a remote server and notify the developer via email.
    header "Installing mailx"
    # TODO - Not working for some reason
    # info "Select option 2 (Internet Site) then enter \"http://mirrors.kernel.org/ubuntu\" when prompted."
    #sudo apt-get install -y mailutils
    # While we're here, also check if .env and .env-prod exist. If not, create them using .env-example.
    if [ ! -f "${HERE}/../.env" ]; then
        header "Creating .env file"
        cp "${HERE}/../.env-example" "${HERE}/../.env"
        warning "Please update the .env file with your own values."
    fi
    if [ ! -f "${HERE}/../.env-prod" ]; then
        header "Creating .env-prod file"
        cp "${HERE}/../.env-example" "${HERE}/../.env-prod"
        warning "Please update the .env-prod file with your own values."
    fi
fi

header "Installing nvm"
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
. ~/.nvm/nvm.sh

header "Installing Node (includes npm)"
nvm install 18.19.0
nvm alias default v18.19.0

header "Installing Yarn"
npm install -g yarn

header "Installing jq for JSON parsing"
sudo apt-get install -y jq

# If using Kubernetes, install relevant tools
if $USE_KUBERNETES; then
    # Check if Kubernetes is installed
    if ! [ -x "$(command -v kubectl)" ]; then
        info "Kubernetes not found. Installing Kubernetes..."

        # Install Kubernetes
        curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
        chmod +x ./kubectl
        sudo mv ./kubectl /usr/local/bin/kubectl

        if ! [ -x "$(command -v kubectl)" ]; then
            error "Failed to install Kubernetes"
            cd "$ORIGINAL_DIR"
            exit 1
        else
            success "Kubernetes installed successfully"
        fi
    else
        success "Kubernetes is already installed"
    fi

    # Check if Helm is installed (used to manage Kubernetes charts)
    if ! [ -x "$(command -v helm)" ]; then
        info "Helm not found. Installing Helm..."

        # Install Helm
        curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3
        chmod 700 get_helm.sh
        trap 'rm -f get_helm.sh' EXIT
        ./get_helm.sh

        if ! [ -x "$(command -v helm)" ]; then
            error "Failed to install Helm"
            cd "$ORIGINAL_DIR"
            exit 1
        else
            success "Helm installed successfully"
        fi
    else
        success "Helm is already installed"
    fi

    # If in a development environment, install Minikube for running Kubernetes locally
    if [ "${ENVIRONMENT}" = "development" ]; then
        if ! [ -x "$(command -v minikube)" ]; then
            info "Minikube not found. Installing Minikube..."

            # Install Minikube
            curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
            sudo install minikube-linux-amd64 /usr/local/bin/minikube

            if ! [ -x "$(command -v minikube)" ]; then
                error "Failed to install Minikube"
                cd "$ORIGINAL_DIR"
                exit 1
            else
                success "Minikube installed successfully"
            fi

            # Rename Minikube context to dev-cluster
            kubectl config rename-context minikube vrooli-dev-cluster
            # Set dev-cluster as the current context
            kubectl config use-context vrooli-dev-cluster

        else
            success "Minikube is already installed"
        fi
    else
        echo "TODO"
        # TODO set up production Kubernetes cluster vrooli-prod-cluster. Might look something like this:
        # # Set the cluster
        # kubectl config set-cluster vrooli-prod-cluster --server=API_SERVER_ENDPOINT --certificate-authority=CA_DATA_PATH --embed-certs=true
        # # Set the credentials
        # kubectl config set-credentials prod-user --client-certificate=CLIENT_CERT_PATH --client-key=CLIENT_KEY_PATH --embed-certs=true
        # # Or, for token-based authentication
        # kubectl config set-credentials prod-user --token=YOUR_BEARER_TOKEN
        # # Set the context
        # kubectl config set-context vrooli-prod-cluster --cluster=vrooli-prod-cluster --user=prod-user
        # # Switch to the new context
        # kubectl config use-context vrooli-prod-cluster
    fi
fi

# We still need Docker and Docker Compose for Kubernetes, mostly for local development.
# May be able to skip some of this for production, but for now just install everything.
if ! command -v docker &>/dev/null; then
    info "Docker is not installed. Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    trap 'rm -f get-docker.sh' EXIT
    sudo sh get-docker.sh
    # Check if Docker installation failed
    if ! command -v docker &>/dev/null; then
        echo "Error: Docker installation failed."
        cd "$ORIGINAL_DIR"
        exit 1
    fi
else
    info "Detected: $(docker --version)"
fi

# Try to start Docker (if already running, this should be a no-op)
sudo service docker start

# Verify Docker is running by attempting a command
if ! docker version >/dev/null 2>&1; then
    error "Failed to start Docker or Docker is not running. If you are in Windows Subsystem for Linux (WSL), please start Docker Desktop and try again."
    cd "$ORIGINAL_DIR"
    exit 1
fi

if ! command -v docker-compose &>/dev/null; then
    info "Docker Compose is not installed. Installing Docker Compose..."
    sudo curl -L "https://github.com/docker/compose/releases/download/v2.15.1/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod a+rx /usr/local/bin/docker-compose
    # Check if Docker Compose installation failed
    if ! command -v docker-compose &>/dev/null; then
        echo "Error: Docker Compose installation failed."
        cd "$ORIGINAL_DIR"
        exit 1
    fi
else
    info "Detected: $(docker-compose --version)"
fi

header "Create nginx-proxy network"
docker network create nginx-proxy
# Ignore errors if the network already exists
if [ $? -ne 0 ]; then
    true
fi

# Development-specific setup
if [ "${ENVIRONMENT}" = "development" ]; then
    header "Installing global dependencies"
    installedPackages=$(yarn global list)
    check_and_add_to_install_list() {
        package="$1"
        version="$2"
        fullPackageName="$package"
        if [ ! -z "$version" ]; then
            fullPackageName="$package@$version"
        fi
        # Check if package is installed globally
        if ! echo "$installedPackages" | grep -E "info \"$package(@$version)?" >/dev/null; then
            info "Installing $fullPackageName"
            toInstall="$toInstall $fullPackageName"
        fi
    }
    toInstall=""
    check_and_add_to_install_list "typescript" "5.3.3"
    # check_and_add_to_install_list "ts-node" "4.7.0"
    check_and_add_to_install_list "nodemon" "3.0.2"
    check_and_add_to_install_list "prisma" "5.7.1"
    check_and_add_to_install_list "vite" "5.2.13"
    # Install all at once if there are packages to install
    if [ ! -z "$toInstall" ]; then
        yarn global add $toInstall
        if [ $? -ne 0 ]; then
            error "Failed to install global dependencies: $toInstall"
            info "Trying to install each package individually..."
            # Split the toInstall string into an array
            IFS=' ' read -r -a individualPackages <<<"$toInstall"
            # Loop through each package and try to install it individually
            for pkg in "${individualPackages[@]}"; do
                info "Attempting to install $pkg individually..."
                yarn global add "$pkg"
                if [ $? -ne 0 ]; then
                    error "Failed to install $pkg"
                    cd "$ORIGINAL_DIR"
                    exit 1
                else
                    info "$pkg installed successfully"
                fi
            done
        fi
    fi

    # If reinstalling modules, delete all node_modules directories before installing dependencies
    if [[ "$REINSTALL_MODULES" =~ ^[Yy]([Ee][Ss])?$ ]]; then
        header "Deleting all node_modules directories"
        find "${HERE}/.." -maxdepth 4 -name "node_modules" -type d -exec rm -rf {} \;
        header "Deleting yarn.lock"
        rm "${HERE}/../yarn.lock"
    fi
    header "Installing local dependencies"
    cd "${HERE}/.." && yarn cache clean && yarn

    header "Generating type models for Prisma"
    cd "${HERE}/../packages/server" && prisma generate --schema ./src/db/schema.prisma

    "${HERE}/shared.sh"

    # Install AWS CLI, for uploading to S3 bucket. This is used for Kubernetes deployments.
    sudo apt-get install awscli
else
    # Less needs to be done for production environments
    info "Skipping global dependencies installation - production environment detected"
    info "Skipping local dependencies installation - production environment detected"
    info "Skipping type models generation - production environment detected"
    info "Skipping shared.sh - production environment detected"
fi

header "Setting up secrets vault"
# Add HashiCorp's GPG key if it's not already added
VAULT_KEYRING="/usr/share/keyrings/hashicorp-archive-keyring.gpg"
if [ ! -f "$VAULT_KEYRING" ]; then
    wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o "$VAULT_KEYRING"
fi
# Determine the distribution codename
DISTRO_CODENAME=$(lsb_release -cs | tail -n 1)
if [ -z "$DISTRO_CODENAME" ]; then
    error "Error determining the distribution codename. Defaulting to 'bionic'."
    DISTRO_CODENAME="bionic"
fi
# Check if the repository for the current distribution exists
if ! curl --output /dev/null --silent --head --fail "https://apt.releases.hashicorp.com/dists/$DISTRO_CODENAME/Release"; then
    warning "No release file found for $DISTRO_CODENAME. Using 'bionic' as fallback."
    DISTRO_CODENAME="bionic"
fi
# Add HashiCorp's APT repository
VAULT_LIST="/etc/apt/sources.list.d/hashicorp.list"
echo "deb [signed-by=$VAULT_KEYRING] https://apt.releases.hashicorp.com $DISTRO_CODENAME main" | sudo tee "$VAULT_LIST"
# Update APT and install Vault
sudo apt update && sudo apt install -y vault
# Setup vault based on environment
FLAGS=""
if [ "${ENVIRONMENT}" = "production" ]; then
    FLAGS="${FLAGS}-p "
fi
if $USE_KUBERNETES; then
    FLAGS="${FLAGS}-k "
fi
"${HERE}/vaultSetup.sh" ${FLAGS}

header "Generating JWT key pair for authentication"
. "${HERE}/genJwt.sh"

if [ "${ENV_FILES_SET_UP}" = "" ]; then
    prompt "Have you already set up your .env and .env-prod files, and would like to generate secret files? (Y/n)"
    read -n1 -r ENV_FILES_SET_UP
    echo
fi
if [[ "$ENV_FILES_SET_UP" =~ ^[Yy]([Ee][Ss])?$ ]]; then
    if [ "${ENVIRONMENT}" = "development" ]; then
        info "Setting up secrets for development environment..."
        "${HERE}/setSecrets.sh" -e development
    else
        info "Setting up secrets for production environment..."
        "${HERE}/setSecrets.sh" -e production
    fi
fi

cd "$ORIGINAL_DIR"
info "Done! You may need to restart your editor for syntax highlighting to work correctly."
