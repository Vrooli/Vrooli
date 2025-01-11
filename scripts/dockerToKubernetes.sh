#!/bin/bash
# Converts a Docker Compose file to Kubernetes YAML files using Kompose.
# Installs Kompose if it is not already installed.
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

KOMPOSE_VERSION="v1.28.0"
SECRET_NAME="vrooli-secrets"

# Check if Kompose is installed (setup script should already ensure Docker, Kubernetes, and Minikube are installed)
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

# Find docker-compose*.yml files
COMPOSE_FILES=$(find ${HERE}/.. -maxdepth 1 -name 'docker-compose*.yml')
COMPOSE_FILES_ARRAY=($COMPOSE_FILES)
COMPOSE_FILE=""
if [ ${#COMPOSE_FILES_ARRAY[@]} -eq 0 ]; then
    prompt "No docker-compose*.yml files found in the directory above. Please enter the path to your docker-compose file."
    read -r COMPOSE_FILE
else
    prompt "Select the number of the docker-compose file you want to use:"
    for ((i = 0; i < ${#COMPOSE_FILES_ARRAY[@]}; i++)); do
        echo "$((i + 1)). ${COMPOSE_FILES_ARRAY[i]}"
    done
    read -r INPUT
    COMPOSE_FILE_NUMBER="${INPUT//[^0-9]/}"
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
    prompt "Select the number of the .env file you want to use:"
    for ((i = 0; i < ${#ENV_FILES_ARRAY[@]}; i++)); do
        echo "$((i + 1)). ${ENV_FILES_ARRAY[i]}"
    done
    read -r INPUT
    ENV_FILE_NUMBER="${INPUT//[^0-9]/}"
    ENV_FILE=${ENV_FILES_ARRAY[$((ENV_FILE_NUMBER - 1))]}
fi
info "Using .env file: ${ENV_FILE}"

# Set Kubernetes secrets using .env file
# NOTE: To check the secrets, run `kubectl get secrets` and `kubectl describe secret <SECRET_NAME>`
info "Setting Kubernetes secrets using .env file: ${ENV_FILE}"

# Define the array of environment variables to include (i.e. data that is not included in the vault)
# NOTE: This should not include passwords or other sensitive data. This is because
# we need to deploy the Docker images to Docker Hub, which means these are environment variables in
# docker-compose, which means they can be found in the image. The best way to keep them secret
# is using a vault, which we'll do later.
INCLUDE_ENV=("GOOGLE_ADSENSE_PUBLISHER_ID" "GOOGLE_TRACKING_ID" "LETSENCRYPT_EMAIL" "SERVER_URL" "SITE_EMAIL_ALIAS" "SITE_EMAIL_FROM" "SITE_EMAIL_USERNAME" "SITE_IP" "STRIPE_PUBLISHABLE_KEY" "VAPID_PUBLIC_KEY" "VIRTUAL_HOST" "VIRTUAL_HOST_DOCS")

# Convert the array to an associative array (i.e., a hash map) for efficient lookups
declare -A INCLUDE_ENV_MAP
for KEY in "${INCLUDE_ENV[@]}"; do
    INCLUDE_ENV_MAP["$KEY"]=1
done

SECRET_DATA=""
while IFS= read -r line || [ -n "$line" ]; do
    # Ignore lines that start with '#' or are blank
    if echo "$line" | grep -q -v '^#' && [ -n "$line" ]; then
        KEY=$(echo "$line" | cut -d '=' -f 1)
        VALUE=$(echo "$line" | cut -d '=' -f 2-)
        # Only add the key-value pair to the secret data if the key is in the include array
        if [[ -n "${INCLUDE_ENV_MAP[$KEY]}" ]]; then
            SECRET_DATA+=" --from-literal=${KEY}=${VALUE}"
        fi
    fi
done <"$ENV_FILE"

# Start Minikube if it is not already running
if ! minikube status >/dev/null 2>&1; then
    info "Starting Minikube..."
    # NOTE: If this is failing, try running `minikube delete` and then running this script again.
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

if [ -n "$SECRET_DATA" ]; then
    info "Setting secrets..."
    kubectl create secret generic "${SECRET_NAME}" $SECRET_DATA --dry-run=client -o yaml | kubectl apply -f -
    if [ $? -ne 0 ]; then
        error "Failed to set Kubernetes secrets"
        exit 1
    fi
else
    error "No secrets to set"
fi
success "Kubernetes secrets set successfully"

# Specify non-sensitive environment variables, which will be replaced with their values in the Docker Compose file.
# DO NOT INCLUDE VARIABLES FOR PASSWORDS OR OTHER SENSITIVE INFORMATION!!!
# NOTE: Variables used in ports must be included here, because Kompose doesn't support environment variables in ports.
NON_SENSITIVE_VARS=("CREATE_MOCK_DATA" "DB_PULL" "DB_USER" "GENERATE_SOURCEMAP" "PORT_DB" "PORT_DOCS" "PORT_REDIS" "PORT_SERVER" "PORT_TRANSLATE" "PORT_UI" "PROJECT_DIR" "SERVER_LOCATION")

# Load environment variables from .env file
set -a
. ${ENV_FILE}
set +a

# Create a copy of the docker-compose file, which we will prepare for Kompose
cp ${COMPOSE_FILE} ${COMPOSE_FILE}.edit
trap "rm ${COMPOSE_FILE}.edit" EXIT

# Replace non-sensitive environment variables with their values
for VAR_NAME in ${NON_SENSITIVE_VARS[@]}; do
    VAR_VALUE=${!VAR_NAME}
    info "Replacing ${VAR_NAME} with ${VAR_VALUE}"
    sed -i -E 's|\$\{'"${VAR_NAME}"'(:-[^}]*)?\}|'"${VAR_VALUE}"'|g' ${COMPOSE_FILE}.edit
done

# Replace remaining environment variables with their names, wrapped in angle brackets
# For example, ${DB_USER} becomes <DB_USER>, or ${DB_USER:-default} becomes <DB_USER>
# These will be replaced with Kubernetes secrets later.
sed -i -E 's/\$\{([^:}]*).*\}/<\1>/g' ${COMPOSE_FILE}.edit

# Generate name for output file based on selected docker-compose file and environment
COMPOSE_BASENAME=$(basename "$COMPOSE_FILE" .yml)
ENV_BASENAME=$(basename "${HERE}/../.env" .env)
ENV_BASENAME=${ENV_BASENAME#.} # Remove the leading dot
OUTPUT_FILE="k8s-${COMPOSE_BASENAME}-${ENV_BASENAME}.yml"

# Convert the docker-compose file
kompose convert -f ${COMPOSE_FILE}.edit -o ${OUTPUT_FILE}
if [ $? -ne 0 ]; then
    error "Failed to convert the Docker Compose file"
    exit 1
else
    success "Docker Compose file converted successfully"
fi

# Replace angle brackets surrounded by whitespaces with Kubernetes secrets
sed -i -E 's|value: <([^>]+)>|valueFrom:\n                secretKeyRef:\n                  name: '"${SECRET_NAME}"'\n                  key: \1|g' ${OUTPUT_FILE}
if [ $? -ne 0 ]; then
    error "Failed to replace angle brackets with Kubernetes secrets"
    exit 1
else
    success "Angle brackets replaced with Kubernetes secrets successfully"
fi
# Replace angle brackets within strings with Kubernetes variable references
# NOTE: This assumes that the referenced variables are defined earlier in the same env section.
# This is because Kubernetes does not allow secret references in string interpolations.
sed -i -E 's|<([^>]+)>|$(\1)|g' ${OUTPUT_FILE}
if [ $? -ne 0 ]; then
    error "Failed to replace angle brackets within strings with Kubernetes variable references"
    exit 1
else
    success "Angle brackets within strings replaced with Kubernetes variable references successfully"
fi

# Set up the Vault for sensitive environment variables, as well as jwt keys
# TODO

# Dry run the generated Kubernetes YAML file to check for errors
kubectl apply -f ${OUTPUT_FILE} --dry-run=client
if [ $? -ne 0 ]; then
    error "Failed to dry run the generated Kubernetes YAML file"
    exit 1
else
    success "Generated Kubernetes YAML file validated successfully"
    prompt "Would you like to apply the generated Kubernetes YAML file now? (y/n)"
    read -n1 -r APPLY
    echo
    if is_yes "$APPLY"; then
        kubectl apply -f ${OUTPUT_FILE}
        if [ $? -ne 0 ]; then
            error "Failed to apply the generated Kubernetes YAML file"
            exit 1
        else
            success "Generated Kubernetes YAML file applied successfully"
        fi
    else
        success "Generated Kubernetes YAML file not applied"
    fi
fi

# Ask if Minikube should be stopped
prompt "Would you like to stop Minikube? (y/n)"
read -n1 -r STOP_MINIKUBE
echo
if is_yes "$STOP_MINIKUBE"; then
    info "Stopping Minikube..."
    minikube stop
    if [ $? -ne 0 ]; then
        error "Failed to stop Minikube"
        exit 1
    else
        success "Minikube stopped successfully"
    fi
fi

success "Success! Generated Kubernetes config file: ${OUTPUT_FILE}"
warning "NOTE: Please refer to the Kubernetes testing guide for instructions on how to test the generated config file."
warning "It most likely WILL NOT WORK without some modifications."
