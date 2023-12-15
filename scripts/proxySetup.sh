#!/bin/bash
# Sets up reverse proxy for a remote server
ORIGINAL_DIR=$(pwd)
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Read arguments
SETUP_ARGS=()
for arg in "$@"; do
    case $arg in
    -n | --nginx-location)
        NGINX_LOCATION="${2}"
        shift
        shift
        ;;
    -h | --help)
        echo "Usage: $0 [-n NGINX_LOCATION] [-h]"
        echo "  -n --nginx-location: Nginx proxy location (e.g. \"/root/NginxSSLReverseProxy\")"
        echo "  -h --help: Show this help message"
        exit 0
        ;;
    *)
        SETUP_ARGS+=("${arg}")
        shift
        ;;
    esac
done

info "Setting up reverse proxy..."

# Check if nginx-proxy and nginx-proxy-le are running
if [ ! "$(docker ps -q -f name=nginx-proxy)" ] || [ ! "$(docker ps -q -f name=nginx-proxy-le)" ]; then
    error "Proxy containers are not running!"
    if [ -z "$NGINX_LOCATION" ]; then
        while true; do
            prompt "Enter path to proxy container directory (defaults to /root/NginxSSLReverseProxy):"
            read -r NGINX_LOCATION
            if [ -z "$NGINX_LOCATION" ]; then
                NGINX_LOCATION="/root/NginxSSLReverseProxy"
            fi

            if [ -d "${NGINX_LOCATION}" ]; then
                break
            else
                error "Not found at that location."
                prompt "Do you want to try again? Say no to clone and set up proxy containers (y/N):"
                read -r TRY_AGAIN
                if [[ "$TRY_AGAIN" =~ ^(no|n)$ ]]; then
                    info "Proceeding with cloning..."
                    break
                fi
            fi
        done
    fi

    # Check if the NginxSSLReverseProxy directory exists
    if [ ! -d "${NGINX_LOCATION}" ]; then
        info "NginxSSLReverseProxy not installed. Cloning and setting up..."
        git clone --depth 1 --branch main https://github.com/MattHalloran/NginxSSLReverseProxy.git "${NGINX_LOCATION}"
        chmod +x "${NGINX_LOCATION}/scripts/*"
        "${NGINX_LOCATION}/scripts/fullSetup.sh"
    fi

    # Check if ${NGINX_LOCATION}/docker-compose.yml exists
    if [ -f "${NGINX_LOCATION}/docker-compose.yml" ]; then
        info "Starting proxy containers..."
        cd "${NGINX_LOCATION}" && docker-compose up -d
    else
        error "Could not find docker-compose.yml file in ${NGINX_LOCATION}"
        cd "$ORIGINAL_DIR"
        exit 1
    fi
fi

cd "$ORIGINAL_DIR"
success "Reverse proxy set up!"
