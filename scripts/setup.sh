#!/bin/bash
# Sets up NPM, Yarn, global dependencies, and anything else
# required to get the project up and running.

# Exit codes
export ERROR_USAGE=64
export ERROR_NO_INTERNET=65

ORIGINAL_DIR=$(pwd)
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

# Default values
REINSTALL_MODULES=""
ON_REMOTE=""
ENVIRONMENT=${NODE_ENV:-development}
ENV_FILES_SET_UP=""
USE_KUBERNETES=false

usage() {
    cat <<EOF
Usage: $(basename "$0") [-h HELP] [-e ENV_SETUP] [-k KUBERNETES] [-m MODULES_REINSTALL] [-p PROD] [-r REMOTE]

Sets up NPM, Yarn, global dependencies, and anything else required to get the project up and running.

Options:
  -h, --help:       Show this help message
  -e, --env-setup:  (Y/n) True if you want to create secret files for the environment variables. If not provided, will prompt.
  -k, --kubernetes: If set, will use Kubernetes instead of Docker Compose
  -m, --modules-reinstall: (y/N) If set to "y", will delete all node_modules directories and reinstall
  -p, --prod:       If set, will skip steps that are only required for development
  -r, --remote:     (Y/n) True if this script is being run on a remote server

Exit Codes:
  0                     Success
  $ERROR_USAGE              Command line usage error
  $ERROR_NO_INTERNET        No internet access

EOF
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        key="$1"
        case $key in
        -h | --help)
            usage
            exit 0
            ;;
        -e | --env-setup)
            if [ -z "$2" ] || [[ "$2" == -* ]]; then
                echo "Error: Option $key requires an argument."
                exit $ERROR_USAGE
            fi
            ENV_FILES_SET_UP="${2}"
            shift # past argument
            shift # past value
            ;;
        -k | --kubernetes)
            USE_KUBERNETES=true
            shift # past argument
            ;;
        -m | --modules-reinstall)
            if [ -z "$2" ] || [[ "$2" == -* ]]; then
                echo "Error: Option $key requires an argument."
                usage
                exit $ERROR_USAGE
            fi
            REINSTALL_MODULES="${2}"
            shift # past argument
            shift # past value
            ;;
        -p | --prod)
            ENVIRONMENT="production"
            shift # past argument
            ;;
        -r | --remote)
            if [ -z "$2" ] || [[ "$2" == -* ]]; then
                echo "Error: Option $key requires an argument."
                usage
                exit $ERROR_USAGE
            fi
            ON_REMOTE="${2}"
            shift # past argument
            shift # past value
            ;;
        *)
            # Unknown option
            echo "Unknown option: $1"
            shift # past argument
            ;;
        esac
    done
}

set_script_permissions() {
    header "Setting script permissions"
    chmod +x "${HERE}/"*.sh
}

fix_system_clock() {
    header "Making sure the system clock is accurate"
    sudo hwclock -s
    info "System clock is now: $(date)"
}

check_internet() {
    header "Checking host internet access..."
    if ping -c 1 google.com &>/dev/null; then
        success "Host internet access: OK"
    else
        error "Host internet access: FAILED"
        exit $ERROR_NO_INTERNET
    fi
}

# Limits the number of apt-get update calls
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

# Limit the number of apt-get upgrade calls
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

run_apt_get_update_and_upgrade() {
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
}

setup_local_server() {
    # Make sure mailx is installed. This may be used by some scripts which
    # track errors on a remote server and notify the developer via email.
    header "Installing mailx"
    # TODO - Not working for some reason
    # info "Select option 2 (Internet Site) then enter \"http://mirrors.kernel.org/ubuntu\" when prompted."
    #sudo apt-get install -y mailutils

    # Check if .env and .env-prod exist. If not, create them using .env-example.
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

    # Check for keytool and install JDK if it's not available. This is
    # used for signing the app in the Google Play store
    if ! command -v keytool &>/dev/null; then
        header "Installing JDK for keytool"
        sudo apt update
        sudo apt install -y default-jdk
        success "JDK installed. keytool should now be available."
    else
        info "keytool is already installed"
    fi

    # Install Bats for testing bash scripts
    header "Installing Bats and dependencies for Bash script testing"
    BATS_TEST_DIR="${HERE}/tests/helpers"
    mkdir -p "$BATS_TEST_DIR"
    cd "$BATS_TEST_DIR"

    # Function to clone and confirm each bats dependency
    install_bats_dependency() {
        local repo_url=$1
        local dir_name=$2
        if [ ! -d "$dir_name" ]; then
            git clone "$repo_url" "$dir_name"
            success "$dir_name installed successfully at $(pwd)/$dir_name"
        else
            info "$dir_name is already installed"
        fi
    }

    # Install Bats-core
    if [ ! -d "bats-core" ]; then
        git clone https://github.com/bats-core/bats-core.git
        cd bats-core
        sudo ./install.sh /usr/local
        success "Bats-core installed successfully at $(pwd)"
        cd ..
    else
        info "Bats-core is already installed"
    fi

    # Install bats-support
    install_bats_dependency "https://github.com/bats-core/bats-support.git" "bats-support"

    # Install bats-mock
    install_bats_dependency "https://github.com/jasonkarns/bats-mock.git" "bats-mock"

    # Install bats-assert
    install_bats_dependency "https://github.com/bats-core/bats-assert.git" "bats-assert"

    cd "$ORIGINAL_DIR"
}

setup_remote_server() {
    # enable PasswordAuthentication for ssh
    header "Enabling PasswordAuthentication"
    sudo sed -i 's/#\?PasswordAuthentication .*/PasswordAuthentication yes/g' /etc/ssh/sshd_config
    sudo sed -i 's/#\?PubkeyAuthentication .*/PubkeyAuthentication yes/g' /etc/ssh/sshd_config
    sudo sed -i 's/#\?AuthorizedKeysFile .*/AuthorizedKeysFile .ssh\/authorized_keys/g' /etc/ssh/sshd_config

    # Ensure .ssh directory and authorized_keys file exist with correct permissions
    mkdir -p ~/.ssh
    touch ~/.ssh/authorized_keys
    chmod 700 ~/.ssh
    chmod 600 ~/.ssh/authorized_keys

    # Try restarting SSH service, checking for both common service names
    if ! sudo systemctl restart sshd 2>/dev/null; then
        if ! sudo systemctl restart ssh 2>/dev/null; then
            echo "Failed to restart ssh. Exiting with error."
            exit 1
        fi
    fi
}

setup_node() {
    header "Installing nvm"
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
    . ~/.nvm/nvm.sh

    header "Installing Node (includes npm)"
    nvm install 18.19.0
    nvm alias default v18.19.0

    header "Installing Yarn"
    npm install -g yarn
}

setup_json_parser() {
    header "Installing jq for JSON parsing"
    sudo apt-get install -y jq
}

setup_kubernetes() {
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
}

setup_docker() {
    if command -v docker &>/dev/null; then
        info "Detected: $(docker --version)"
        return 0
    fi

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
}

start_docker() {
    # Try to start Docker (if already running, this should be a no-op)
    sudo service docker start

    # Verify Docker is running by attempting a command
    if ! docker version >/dev/null 2>&1; then
        error "Failed to start Docker or Docker is not running. If you are in Windows Subsystem for Linux (WSL), please start Docker Desktop and try again."
        cd "$ORIGINAL_DIR"
        exit 1
    fi
}

restart_docker() {
    info "Restarting Docker..."
    sudo service docker restart
}

setup_docker_compose() {
    if command -v docker-compose &>/dev/null; then
        info "Detected: $(docker-compose --version)"
        return 0
    fi

    info "Docker Compose is not installed. Installing Docker Compose..."
    sudo curl -L "https://github.com/docker/compose/releases/download/v2.15.1/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod a+rx /usr/local/bin/docker-compose
    # Check if Docker Compose installation failed
    if ! command -v docker-compose &>/dev/null; then
        echo "Error: Docker Compose installation failed."
        cd "$ORIGINAL_DIR"
        exit 1
    fi
}

setup_docker_network() {
    header "Creating nginx-proxy network"
    docker network create nginx-proxy
    # Ignore errors if the network already exists
    if [ $? -ne 0 ]; then
        true
    fi
}

check_docker_internet() {
    header "Checking Docker internet access..."
    if docker run --rm busybox ping -c 1 google.com &>/dev/null; then
        success "Docker internet access: OK"
    else
        error "Docker internet access: FAILED"
        return 1
    fi
}

show_docker_daemon() {
    if [ -f /etc/docker/daemon.json ]; then
        info "Current /etc/docker/daemon.json:"
        cat /etc/docker/daemon.json
    else
        warning "/etc/docker/daemon.json does not exist."
    fi
}

update_docker_daemon() {
    info "Updating /etc/docker/daemon.json to use Google DNS (8.8.8.8)..."

    # Check if /etc/docker/daemon.json exists
    if [ -f /etc/docker/daemon.json ]; then
        # Backup existing file
        sudo cp /etc/docker/daemon.json /etc/docker/daemon.json.backup
        info "Backup created at /etc/docker/daemon.json.backup"
    fi

    # Write new config
    sudo bash -c 'cat > /etc/docker/daemon.json' <<EOF
{
  "dns": ["8.8.8.8"]
}
EOF

    info "/etc/docker/daemon.json updated."
}

setup_docker_internet() {
    if ! check_docker_internet; then
        error "Docker cannot access the internet. This may be a DNS issue."
        show_docker_daemon

        prompt "Would you like to update /etc/docker/daemon.json to use Google DNS (8.8.8.8)? (y/n): " choice
        read -n1 -r choice
        echo
        if is_yes "$choice"; then
            update_docker_daemon
            restart_docker
            info "Docker DNS updated. Retesting Docker internet access..."
            check_docker_internet && success "Docker internet access is now working!" || error "Docker internet access still failing."
        else
            echo "No changes made."
        fi
    else
        echo "Docker already has internet access."
    fi
}

setup_development_environment() {
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
    check_and_add_to_install_list "nodemon" "3.0.2"
    check_and_add_to_install_list "prisma" "6.1.0"
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
    if is_yes "$REINSTALL_MODULES"; then
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
}

setup_production_environment() {
    # Less needs to be done for production environments
    info "Skipping global dependencies installation - production environment detected"
    info "Skipping local dependencies installation - production environment detected"
    info "Skipping type models generation - production environment detected"
    info "Skipping shared.sh - production environment detected"
}

generate_jwt_key_pair() {
    header "Generating JWT key pair for authentication"
    "${HERE}/genJwt.sh"
}

setup_vault() {
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
}

populate_vault() {
    if [ "${ENV_FILES_SET_UP}" = "" ]; then
        echo "${ENVIRONMENT} detected."
        if [ "${ENVIRONMENT}" = "development" ]; then
            prompt "Have you already set up your .env file, and would you like to generate secret files? (Y/n)"
        else
            prompt "Have you already set up your .env-prod file, and would you like to generate secret files? (Y/n)"
        fi
        read -n1 -r ENV_FILES_SET_UP
        echo
    fi
    if is_yes "$ENV_FILES_SET_UP"; then
        if [ "${ENVIRONMENT}" = "development" ]; then
            info "Setting up secrets for development environment..."
            "${HERE}/setSecrets.sh" -e development
        else
            info "Setting up secrets for production environment..."
            "${HERE}/setSecrets.sh" -e production
        fi
    fi
}

main() {
    parse_arguments "$@"

    set_script_permissions

    load_env_file $ENVIRONMENT
    # Determine where this script is running (local or remote)
    export SERVER_LOCATION=$("${HERE}/domainCheck.sh" $SITE_IP $API_URL | tail -n 1)

    fix_system_clock

    check_internet

    run_apt_get_update_and_upgrade

    if [[ "$SERVER_LOCATION" != "local" ]]; then
        setup_remote_server
    else
        setup_local_server
    fi

    setup_node
    setup_json_parser

    if $USE_KUBERNETES; then
        setup_kubernetes
    fi

    # We still need Docker and Docker Compose even if we're using Kubernetes - mostly for local development.
    # May be able to skip some of this for production, but for now just install everything.
    setup_docker
    start_docker
    setup_docker_compose
    setup_docker_network
    setup_docker_internet

    if [ "${ENVIRONMENT}" = "development" ]; then
        setup_development_environment
    else
        setup_production_environment
    fi

    generate_jwt_key_pair

    setup_vault
    populate_vault

    cd "$ORIGINAL_DIR"
    info "Done! You may need to restart your editor for syntax highlighting to work correctly."
}

run_if_executed main "$@"
