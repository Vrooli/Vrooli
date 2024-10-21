#!/bin/bash
# Sets up reverse proxy for a remote server
ORIGINAL_DIR=$(pwd)
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

# Exit codes
ERROR_USAGE=64
ERROR_PROXY_CONTAINER_START_FAILED=65
ERROR_PROXY_LOCATION_NOT_FOUND=66
ERROR_PROXY_CLONE_FAILED=67

DEFAULT_PROXY_LOCATION="/root/NginxSSLReverseProxy"
PROXY_REPO_URL="https://github.com/MattHalloran/NginxSSLReverseProxy.git"

usage() {
    cat <<EOF
Usage: $(basename "$0") [-n PROXY_LOCATION] [-h]

Sets up reverse proxy for a remote server.

Options:
  -n, --nginx-location PROXY_LOCATION  Nginx proxy location (e.g. "/root/NginxSSLReverseProxy")
  -h, --help                           Show this help message

Exit Codes:
  0                                   Success
  $ERROR_USAGE                        Command line usage error
  $ERROR_PROXY_CONTAINER_START_FAILED Proxy containers failed to start
  $ERROR_PROXY_LOCATION_NOT_FOUND     Proxy location not found or is invalid
  $ERROR_PROXY_CLONE_FAILED           Proxy clone and setup failed

EOF
}

PROXY_LOCATION=""
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        key="$1"
        case $key in
        -n | --nginx-location)
            if [ -z "$2" ] || [[ "$2" == -* ]]; then
                error "Option $key requires an argument."
                usage
                exit $ERROR_USAGE
            fi
            echo "Setting PROXY_LOCATION to $2"
            PROXY_LOCATION="$2"
            shift 2
            ;;
        -h | --help)
            usage
            exit 0
            ;;
        esac
    done
}

is_proxy_location_valid() {
    # Check if PROXY_LOCATION is set
    if [ -z "$PROXY_LOCATION" ]; then
        return 1
    fi
    # Check if PROXY_LOCATION is a valid directory
    if [ ! -d "$PROXY_LOCATION" ]; then
        return 1
    fi
    return 0
}

get_proxy_location() {
    # If location is already valid, return early
    if is_proxy_location_valid; then
        return 0
    fi

    # Prompt user for location
    while true; do
        prompt "Enter path to proxy container directory (defaults to $DEFAULT_PROXY_LOCATION):"
        read -r input_location
        PROXY_LOCATION=${input_location:-$DEFAULT_PROXY_LOCATION}

        # If location is valid, return
        if is_proxy_location_valid; then
            return 0
        fi

        # Otherwise, prompt user to try again
        error "Location invalid."
        prompt "Do you want to try again? Say no to clone and set up proxy containers (y/N):"
        read -r try_again
        if [[ "$try_again" =~ ^(no|n)$ ]]; then
            return 0
        fi
    done
}

are_proxy_containers_running() {
    # Look for running Docker containers with names 'nginx-proxy' and 'nginx-proxy-le'
    if [ "$(docker ps -q -f name=nginx-proxy)" ] && [ "$(docker ps -q -f name=nginx-proxy-acme)" ]; then
        return 0
    fi
    return 1
}

start_proxy_containers() {
    # Check if containers are already running
    if are_proxy_containers_running; then
        return 0
    fi

    if [ -f "${PROXY_LOCATION}/docker-compose.yml" ]; then
        info "Starting proxy containers..."
        cd "$PROXY_LOCATION" && docker-compose up -d
    elif [ -f "${PROXY_LOCATION}/docker-compose.yaml" ]; then
        info "Starting proxy containers..."
        cd "$PROXY_LOCATION" && docker-compose up -d
    else
        error "Could not find docker-compose.yml file in $PROXY_LOCATION"
        return 1
    fi
}

is_proxy_cloned() {
    if [ ! -f "${PROXY_LOCATION}/docker-compose.yml" ] && [ ! -f "${PROXY_LOCATION}/docker-compose.yaml" ]; then
        return 1
    fi
    return 0
}

setup_proxy_repo() {
    # If proxy is already cloned, return early
    if is_proxy_cloned; then
        return 0
    fi

    # Clone and set up repo
    info "Cloning and setting up proxy repo in $PROXY_LOCATION..."
    git clone --depth 1 --branch main "$PROXY_REPO_URL" "$PROXY_LOCATION"
    chmod +x "${PROXY_LOCATION}/scripts/"*
    "${PROXY_LOCATION}/scripts/fullSetup.sh"
}

main() {
    parse_arguments "$@"

    # Get proxy location
    get_proxy_location
    if ! is_proxy_location_valid; then
        error "Proxy location not found or is invalid."
        exit $ERROR_PROXY_LOCATION_NOT_FOUND
    fi

    # Clone and set up proxy repo if necessary
    setup_proxy_repo
    if [ $? -ne 0 ]; then
        exit $ERROR_PROXY_CLONE_FAILED
    fi

    # Start proxy containers
    start_proxy_containers
    if [ $? -ne 0 ]; then
        exit $ERROR_PROXY_CONTAINER_START_FAILED
    fi

    # Return to original directory
    cd "$ORIGINAL_DIR"

    success "Reverse proxy set up!"
}

run_if_executed main "$@"
