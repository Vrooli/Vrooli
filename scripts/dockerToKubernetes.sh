#!/bin/bash
# Converts a Docker Compose file to Kubernetes YAML files using Kompose.
# Installs Kompose if it is not already installed.
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

KOMPOSE_VERSION="v1.28.0"
SECRET_NAME="vrooli-secrets"

# Check if Kubernetes is installed
if ! [ -x "$(command -v kubectl)" ]; then
    info "Kubernetes not found. Installing Kubernetes..."

    # Install Kubernetes
    curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x ./kubectl
    sudo mv ./kubectl /usr/local/bin/kubectl

    if ! [ -x "$(command -v kubectl)" ]; then
        error "Failed to install Kubernetes"
        exit 1
    else
        success "Kubernetes installed successfully"
    fi
else
    success "Kubernetes is already installed"
fi

# Check if Kompose is installed
if ! [ -x "$(command -v kompose)" ]; then
    info "Kompose not found. Installing Kompose..."

    # Download and install Kompose
    curl -L https://github.com/kubernetes/kompose/releases/download/${KOMPOSE_VERSION}/kompose-linux-amd64 -o kompose

    # Make the binary executable
    chmod +x kompose

    # Move the binary to your PATH
    sudo mv ./kompose /usr/local/bin/kompose

    if ! [ -x "$(command -v kompose)" ]; then
        error "Failed to install Kompose"
        exit 1
    else
        success "Kompose installed successfully"
    fi
else
    success "Kompose is already installed"
fi

# Check if Minikube is installed
if ! [ -x "$(command -v minikube)" ]; then
    info "Minikube not found. Installing Minikube..."

    # Install Minikube
    curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
    sudo install minikube-linux-amd64 /usr/local/bin/minikube

    if ! [ -x "$(command -v minikube)" ]; then
        error "Failed to install Minikube"
        exit 1
    else
        success "Minikube installed successfully"
    fi
else
    success "Minikube is already installed"
fi

# Start Minikube if it's not running
if ! minikube status >/dev/null 2>&1; then
    info "Starting Minikube..."
    minikube start --driver=docker --force # TODO get rid of --force by running Docker in rootless mode

    if [ $? -ne 0 ]; then
        error "Failed to start Minikube"
        exit 1
    else
        success "Minikube started successfully"
    fi
else
    success "Minikube is already running"
fi

# Find docker-compose*.yml files
COMPOSE_FILES=$(find ${HERE}/.. -maxdepth 1 -name 'docker-compose*.yml')
COMPOSE_FILES_ARRAY=($COMPOSE_FILES)
COMPOSE_FILE=""
if [ ${#COMPOSE_FILES_ARRAY[@]} -eq 0 ]; then
    prompt "No docker-compose*.yml files found in the directory above. Please enter the path to your docker-compose file."
    read -r COMPOSE_FILE
else
    echo "Select the number of the docker-compose file you want to use:"
    for ((i = 0; i < ${#COMPOSE_FILES_ARRAY[@]}; i++)); do
        echo "$((i + 1)). ${COMPOSE_FILES_ARRAY[i]}"
    done
    read -r COMPOSE_FILE_NUMBER
    COMPOSE_FILE=${COMPOSE_FILES_ARRAY[$((COMPOSE_FILE_NUMBER - 1))]}
fi
info "Using docker-compose file: ${COMPOSE_FILE}"

# Find .env* files
ENV_FILES=$(find ${HERE}/.. -maxdepth 1 -name '.env*')
ENV_FILES_ARRAY=($ENV_FILES)
ENV_FILE=""
if [ ${#ENV_FILES_ARRAY[@]} -eq 0 ]; then
    prompt "No .env* files found in the directory above. Please enter the path to your .env file."
    read -r ENV_FILE
else
    echo "Select the number of the .env file you want to use:"
    for ((i = 0; i < ${#ENV_FILES_ARRAY[@]}; i++)); do
        echo "$((i + 1)). ${ENV_FILES_ARRAY[i]}"
    done
    read -r ENV_FILE_NUMBER
    ENV_FILE=${ENV_FILES_ARRAY[$((ENV_FILE_NUMBER - 1))]}
fi

# Set Kubernetes secrets using .env file
# NOTE: To check the secrets, run `kubectl get secrets` and `kubectl describe secret <SECRET_NAME>`
info "Setting Kubernetes secrets using .env file: ${ENV_FILE}"
SECRET_DATA=""
while IFS= read -r line || [ -n "$line" ]; do
    # Ignore lines that start with '#' or are blank
    if echo "$line" | grep -q -v '^#' && [ -n "$line" ]; then
        KEY=$(echo "$line" | cut -d '=' -f 1)
        VALUE=$(echo "$line" | cut -d '=' -f 2-)
        SECRET_DATA+=" --from-literal=${KEY}=${VALUE}"
    fi
done <"$ENV_FILE"
if [ -n "$SECRET_DATA" ]; then
    echo "Setting secrets..."
    kubectl create secret generic "${SECRET_NAME}" $SECRET_DATA --dry-run=client -o yaml | kubectl apply -f -
    if [ $? -ne 0 ]; then
        error "Failed to set Kubernetes secrets"
        exit 1
    fi
    success "Kubernetes secrets set successfully"
else
    error "No secrets to set"
fi
success "Kubernetes secrets set successfully"

# Specify non-sensitive environment variables, which will be replaced with their values in the Docker Compose file.
# DO NOT INCLUDE VARIABLES FOR PASSWORDS OR OTHER SENSITIVE INFORMATION!!!
NON_SENSITIVE_VARS=("SERVER_LOCATION" "CREATE_MOCK_DATA" "DB_PULL" "PORT_UI" "PORT_SERVER" "PORT_DB" "PORT_REDIS" "PORT_DOCS" "PORT_TRANSLATE" "PROJECT_DIR" "GENERATE_SOURCEMAP")

# Load environment variables from .env file
set -a
source ${ENV_FILE}
set +a

# Create a copy of the docker-compose file, which we will prepare for Kompose
cp ${COMPOSE_FILE} ${COMPOSE_FILE}.edit

# Replace non-sensitive environment variables with their values
for VAR_NAME in ${NON_SENSITIVE_VARS[@]}; do
    VAR_VALUE=${!VAR_NAME}
    echo "Replacing ${VAR_NAME} with ${VAR_VALUE}"
    sed -i -E 's|\$\{'"${VAR_NAME}"'\}|'"${VAR_VALUE}"'|g' ${COMPOSE_FILE}.edit
done

# Replace remaining environment variables with their names, wrapped in angle brackets
# For example, ${DB_USER} becomes <DB_USER>.
# These will be replaced with Kubernetes secrets later.
sed -i -E 's/\$\{([^}]+)\}/<\1>/g' ${COMPOSE_FILE}.edit

# Convert the docker-compose file
kompose convert -f ${COMPOSE_FILE}.edit -o k8s.yml
if [ $? -ne 0 ]; then
    error "Failed to convert the Docker Compose file"
    exit 1
else
    success "Docker Compose file converted successfully"
fi

# Remove the edited docker-compose file
rm ${COMPOSE_FILE}.edit

# Replace angle brackets surrounded by whitespaces with Kubernetes secrets
sed -i -E 's|(\s)<([^>]+)>|\n              valueFrom:\n              secretKeyRef:\n                name: '"${SECRET_NAME}"'\n                key: \1|g' k8s.yml
if [ $? -ne 0 ]; then
    error "Failed to replace angle brackets with Kubernetes secrets"
    exit 1
else
    success "Angle brackets replaced with Kubernetes secrets successfully"
fi
# Replace angle brackets within strings with Kubernetes variable references
# NOTE: This assumes that the referenced variables are defined earlier in the same env section.
# This is because Kubernetes does not allow secret references in string interpolations.
sed -i -E 's|<([^>]+)>|$(\1)|g' k8s.yml
if [ $? -ne 0 ]; then
    error "Failed to replace angle brackets within strings with Kubernetes variable references"
    exit 1
else
    success "Angle brackets within strings replaced with Kubernetes variable references successfully"
fi

# Dry run the generated Kubernetes YAML file to check for errors
kubectl apply -f k8s.yml --dry-run=client
if [ $? -ne 0 ]; then
    error "Failed to dry run the generated Kubernetes YAML file"
    exit 1
else
    success "Generated Kubernetes YAML file validated successfully"
fi
