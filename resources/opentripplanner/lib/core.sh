#!/usr/bin/env bash
# OpenTripPlanner Core Functions

set -euo pipefail

# Resource configuration
readonly OPENTRIPPLANNER_IMAGE="opentripplanner/opentripplanner:latest-jvm17"
readonly OPENTRIPPLANNER_CONTAINER="vrooli-opentripplanner"
readonly OPENTRIPPLANNER_NETWORK="vrooli-network"

# Get configuration from environment or defaults
export OTP_PORT="${OTP_PORT:-8080}"
export OTP_HEAP_SIZE="${OTP_HEAP_SIZE:-2G}"
export OTP_BUILD_TIMEOUT="${OTP_BUILD_TIMEOUT:-300}"
export OTP_DATA_DIR="${OTP_DATA_DIR:-${HOME}/.vrooli/opentripplanner/data}"
export OTP_CACHE_DIR="${OTP_CACHE_DIR:-${HOME}/.vrooli/opentripplanner/cache}"

# Lifecycle Management Functions

opentripplanner::install() {
    local force="${1:-false}"
    
    if [[ "$force" != "--force" ]] && opentripplanner::is_installed; then
        echo "OpenTripPlanner is already installed"
        return 2
    fi
    
    echo "Installing OpenTripPlanner..."
    
    # Create directories
    mkdir -p "${OTP_DATA_DIR}" "${OTP_CACHE_DIR}"
    
    # Pull Docker image
    docker pull "${OPENTRIPPLANNER_IMAGE}" || {
        echo "Failed to pull OpenTripPlanner image"
        return 1
    }
    
    # Create Docker network if it doesn't exist
    docker network create "${OPENTRIPPLANNER_NETWORK}" &>/dev/null || true
    
    # Download sample data (Portland)
    opentripplanner::download_sample_data || {
        echo "Warning: Failed to download sample data"
    }
    
    echo "OpenTripPlanner installed successfully"
    return 0
}

opentripplanner::uninstall() {
    local keep_data="${1:-false}"
    
    echo "Uninstalling OpenTripPlanner..."
    
    # Stop service if running
    opentripplanner::stop --force &>/dev/null || true
    
    # Remove container
    docker rm -f "${OPENTRIPPLANNER_CONTAINER}" &>/dev/null || true
    
    # Remove image
    docker rmi "${OPENTRIPPLANNER_IMAGE}" &>/dev/null || true
    
    # Remove data if requested
    if [[ "$keep_data" != "--keep-data" ]]; then
        rm -rf "${OTP_DATA_DIR}" "${OTP_CACHE_DIR}"
    fi
    
    echo "OpenTripPlanner uninstalled successfully"
    return 0
}

opentripplanner::start() {
    local wait_flag="${1:-}"
    local timeout="${2:-60}"
    
    if opentripplanner::is_running; then
        echo "OpenTripPlanner is already running"
        return 2
    fi
    
    echo "Starting OpenTripPlanner..."
    
    # Ensure data directory exists
    mkdir -p "${OTP_DATA_DIR}" "${OTP_CACHE_DIR}"
    
    # Start container
    docker run -d \
        --name "${OPENTRIPPLANNER_CONTAINER}" \
        --network "${OPENTRIPPLANNER_NETWORK}" \
        -p "${OTP_PORT}:8080" \
        -v "${OTP_DATA_DIR}:/var/opentripplanner" \
        -v "${OTP_CACHE_DIR}:/var/cache/otp" \
        -e JAVA_OPTS="-Xmx${OTP_HEAP_SIZE}" \
        "${OPENTRIPPLANNER_IMAGE}" \
        --load --serve || {
        echo "Failed to start OpenTripPlanner"
        return 1
    }
    
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "Waiting for OpenTripPlanner to be ready..."
        if ! opentripplanner::wait_for_ready "$timeout"; then
            echo "OpenTripPlanner failed to become ready within ${timeout} seconds"
            return 1
        fi
    fi
    
    echo "OpenTripPlanner started successfully on port ${OTP_PORT}"
    return 0
}

opentripplanner::stop() {
    local force="${1:-}"
    local timeout="${2:-30}"
    
    if ! opentripplanner::is_running; then
        echo "OpenTripPlanner is not running"
        return 2
    fi
    
    echo "Stopping OpenTripPlanner..."
    
    if [[ "$force" == "--force" ]]; then
        docker kill "${OPENTRIPPLANNER_CONTAINER}" &>/dev/null || true
    else
        docker stop -t "$timeout" "${OPENTRIPPLANNER_CONTAINER}" &>/dev/null || true
    fi
    
    docker rm "${OPENTRIPPLANNER_CONTAINER}" &>/dev/null || true
    
    echo "OpenTripPlanner stopped successfully"
    return 0
}

opentripplanner::restart() {
    local force="${1:-}"
    
    opentripplanner::stop "$force" || true
    opentripplanner::start --wait
}

# Status Functions

opentripplanner::status() {
    local format="${1:-text}"
    local verbose="${2:-}"
    
    local running="false"
    local healthy="false"
    local port="${OTP_PORT}"
    local container_status="stopped"
    
    if opentripplanner::is_running; then
        running="true"
        container_status="running"
        
        if opentripplanner::is_healthy; then
            healthy="true"
        fi
    fi
    
    if [[ "$format" == "--json" || "$format" == "json" ]]; then
        cat <<EOF
{
  "service": "opentripplanner",
  "running": ${running},
  "healthy": ${healthy},
  "port": ${port},
  "container_status": "${container_status}",
  "data_dir": "${OTP_DATA_DIR}",
  "cache_dir": "${OTP_CACHE_DIR}"
}
EOF
    else
        echo "OpenTripPlanner Status:"
        echo "  Running: ${running}"
        echo "  Healthy: ${healthy}"
        echo "  Port: ${port}"
        echo "  Container: ${container_status}"
        
        if [[ "$verbose" == "--verbose" ]]; then
            echo "  Data Directory: ${OTP_DATA_DIR}"
            echo "  Cache Directory: ${OTP_CACHE_DIR}"
            echo "  Heap Size: ${OTP_HEAP_SIZE}"
        fi
    fi
    
    [[ "$healthy" == "true" ]] && return 0 || return 1
}

opentripplanner::logs() {
    local tail="${1:-50}"
    local follow="${2:-}"
    
    if [[ "$follow" == "--follow" ]]; then
        docker logs -f --tail "$tail" "${OPENTRIPPLANNER_CONTAINER}"
    else
        docker logs --tail "$tail" "${OPENTRIPPLANNER_CONTAINER}"
    fi
}

# Content Management Functions

opentripplanner::content::add() {
    local file=""
    local type=""
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file) file="$2"; shift 2 ;;
            --type) type="$2"; shift 2 ;;
            --name) name="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$file" || -z "$type" || -z "$name" ]]; then
        echo "Usage: content add --file <path> --type <gtfs|osm> --name <identifier>"
        return 1
    fi
    
    local dest_dir="${OTP_DATA_DIR}"
    
    case "$type" in
        gtfs)
            cp "$file" "${dest_dir}/${name}.gtfs.zip"
            echo "Added GTFS data: ${name}"
            ;;
        osm)
            cp "$file" "${dest_dir}/${name}.osm.pbf"
            echo "Added OSM data: ${name}"
            ;;
        *)
            echo "Unknown type: $type (use gtfs or osm)"
            return 1
            ;;
    esac
    
    return 0
}

opentripplanner::content::list() {
    local format="${1:-text}"
    
    echo "Available data files:"
    echo ""
    echo "GTFS Files:"
    ls -la "${OTP_DATA_DIR}"/*.gtfs.zip 2>/dev/null || echo "  No GTFS files"
    echo ""
    echo "OSM Files:"
    ls -la "${OTP_DATA_DIR}"/*.osm.pbf 2>/dev/null || echo "  No OSM files"
    echo ""
    echo "Graph Cache:"
    ls -la "${OTP_CACHE_DIR}"/graph*.obj 2>/dev/null || echo "  No cached graphs"
}

opentripplanner::content::get() {
    local name=""
    local output=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) name="$2"; shift 2 ;;
            --output) output="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Usage: content get --name <identifier> [--output <path>]"
        return 1
    fi
    
    local source_file=""
    
    # Try to find the file
    for ext in "gtfs.zip" "osm.pbf"; do
        if [[ -f "${OTP_DATA_DIR}/${name}.${ext}" ]]; then
            source_file="${OTP_DATA_DIR}/${name}.${ext}"
            break
        fi
    done
    
    if [[ -z "$source_file" ]]; then
        echo "Content not found: ${name}"
        return 2
    fi
    
    if [[ -n "$output" ]]; then
        cp "$source_file" "$output"
        echo "Retrieved content to: $output"
    else
        echo "Content found at: $source_file"
    fi
    
    return 0
}

opentripplanner::content::remove() {
    local name=""
    local force=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) name="$2"; shift 2 ;;
            --force) force="true"; shift ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Usage: content remove --name <identifier> [--force]"
        return 1
    fi
    
    local found=false
    
    # Remove matching files
    for ext in "gtfs.zip" "osm.pbf"; do
        if [[ -f "${OTP_DATA_DIR}/${name}.${ext}" ]]; then
            rm -f "${OTP_DATA_DIR}/${name}.${ext}"
            echo "Removed: ${name}.${ext}"
            found=true
        fi
    done
    
    if [[ "$found" == "false" ]]; then
        echo "Content not found: ${name}"
        return 2
    fi
    
    return 0
}

opentripplanner::content::execute() {
    local action=""
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --action) action="$2"; shift 2 ;;
            --name) name="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    case "$action" in
        build-graph)
            echo "Building routing graph..."
            # Execute graph build in container
            docker exec "${OPENTRIPPLANNER_CONTAINER}" \
                java -Xmx"${OTP_HEAP_SIZE}" -jar /opt/opentripplanner/otp.jar \
                --build /var/opentripplanner || {
                echo "Failed to build graph"
                return 1
            }
            echo "Graph built successfully"
            ;;
        *)
            echo "Unknown action: $action"
            return 1
            ;;
    esac
    
    return 0
}

# Helper Functions

opentripplanner::is_installed() {
    docker image inspect "${OPENTRIPPLANNER_IMAGE}" &>/dev/null
}

opentripplanner::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${OPENTRIPPLANNER_CONTAINER}$"
}

opentripplanner::is_healthy() {
    local health_url="http://localhost:${OTP_PORT}/otp"
    timeout 5 curl -sf "${health_url}" &>/dev/null
}

opentripplanner::wait_for_ready() {
    local timeout="${1:-60}"
    local elapsed=0
    
    while [[ $elapsed -lt $timeout ]]; do
        if opentripplanner::is_healthy; then
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    echo ""
    
    return 1
}

opentripplanner::download_sample_data() {
    echo "Downloading sample GTFS and OSM data for Portland..."
    
    # Download Portland GTFS data
    local gtfs_url="https://developer.trimet.org/schedule/gtfs.zip"
    local gtfs_file="${OTP_DATA_DIR}/portland.gtfs.zip"
    
    if [[ ! -f "$gtfs_file" ]]; then
        curl -L -o "$gtfs_file" "$gtfs_url" || {
            echo "Failed to download GTFS data"
            return 1
        }
        echo "Downloaded Portland GTFS data"
    fi
    
    # Download Portland OSM extract
    local osm_url="https://download.geofabrik.de/north-america/us/oregon-latest.osm.pbf"
    local osm_file="${OTP_DATA_DIR}/oregon.osm.pbf"
    
    if [[ ! -f "$osm_file" ]]; then
        echo "Downloading Oregon OSM data (includes Portland)..."
        curl -L -o "$osm_file" "$osm_url" || {
            echo "Failed to download OSM data"
            return 1
        }
        echo "Downloaded Oregon OSM data"
    fi
    
    return 0
}