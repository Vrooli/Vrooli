#!/usr/bin/env bash
# Speaker Verification - Docker Management

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source docker utilities if available
if [[ -f "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh" ]]; then
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"
fi

# Source config
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/speaker-verification/config/defaults.sh"
speaker_verification::export_config 2>/dev/null || true

readonly SV_NETWORK_NAME="${SPEAKER_VERIFICATION_NETWORK_NAME}"

#######################################
# Build the Docker image from local Dockerfile
# Returns: 0 on success, 1 on failure
#######################################
speaker_verification::docker::build_image() {
    local resource_dir="${APP_ROOT}/resources/speaker-verification"

    log::info "Building speaker-verification Docker image..."
    if docker build -t "${SPEAKER_VERIFICATION_IMAGE}" -f "${resource_dir}/docker/Dockerfile" "${resource_dir}"; then
        log::success "Image built successfully"
        return 0
    else
        log::error "Failed to build Docker image"
        return 1
    fi
}
export -f speaker_verification::docker::build_image

#######################################
# Start the speaker verification container
# Returns: 0 on success, 1 on failure
#######################################
speaker_verification::docker::start_container() {
    local image
    image=$(speaker_verification::get_docker_image)

    # Create network if it doesn't exist
    docker network create "$SV_NETWORK_NAME" &>/dev/null || true

    # Prepare environment variables
    local env_vars=(
        "SPEAKER_VERIFICATION_DEVICE=${SPEAKER_VERIFICATION_DEVICE}"
        "SPEAKER_VERIFICATION_MODEL=${SPEAKER_VERIFICATION_MODEL}"
        "SPEAKER_VERIFICATION_DEFAULT_THRESHOLD=${SPEAKER_VERIFICATION_DEFAULT_THRESHOLD}"
        "SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS=${SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS}"
        "SPEAKER_VERIFICATION_VERIFY_MIN_SECONDS=${SPEAKER_VERIFICATION_VERIFY_MIN_SECONDS}"
        "SPEAKER_VERIFICATION_SAMPLE_RATE=${SPEAKER_VERIFICATION_SAMPLE_RATE}"
        "SPEAKER_VERIFICATION_MAX_UPLOAD_MB=${SPEAKER_VERIFICATION_MAX_UPLOAD_MB}"
    )

    local docker_args=(
        "run" "-d"
        "--name" "${SPEAKER_VERIFICATION_CONTAINER_NAME}"
        "--hostname" "speaker-verification"
        "--restart" "unless-stopped"
        "--network" "$SV_NETWORK_NAME"
        "-p" "127.0.0.1:${SPEAKER_VERIFICATION_PORT}:8891"
        "-v" "${SPEAKER_VERIFICATION_PROFILES_DIR}:/data/profiles"
        "-v" "${SPEAKER_VERIFICATION_CACHE_DIR}:/data/cache"
        "-v" "${SPEAKER_VERIFICATION_LOG_DIR}:/data/logs"
    )

    # Add environment variables
    for env_var in "${env_vars[@]}"; do
        docker_args+=("-e" "$env_var")
    done

    # Add GPU support if available
    if [[ "${SPEAKER_VERIFICATION_GPU_ENABLED}" == "true" ]]; then
        docker_args+=("--gpus" "all")
        docker_args+=("-e" "NVIDIA_VISIBLE_DEVICES=all")
    fi

    # Add health check
    docker_args+=(
        "--health-cmd" "curl -f http://localhost:8891/health || exit 1"
        "--health-interval" "30s"
        "--health-timeout" "10s"
        "--health-retries" "5"
        "--health-start-period" "120s"
    )

    # Add resource limits
    docker_args+=(
        "--memory" "4g"
        "--log-driver" "json-file"
        "--log-opt" "max-size=10m"
        "--log-opt" "max-file=3"
    )

    docker_args+=("$image")

    log::info "Starting speaker-verification container..."
    if docker "${docker_args[@]}"; then
        log::info "Container started, waiting for initialization..."
        sleep "${SPEAKER_VERIFICATION_INITIALIZATION_WAIT:-10}"
        return 0
    else
        log::error "Failed to start container"
        return 1
    fi
}
export -f speaker_verification::docker::start_container

#######################################
# Stop the container
# Returns: 0 on success
#######################################
speaker_verification::docker::stop_container() {
    if common::is_running; then
        log::info "Stopping speaker-verification container..."
        docker stop "$SPEAKER_VERIFICATION_CONTAINER_NAME" &>/dev/null || true
    fi
    return 0
}
export -f speaker_verification::docker::stop_container

#######################################
# Show container logs
#######################################
speaker_verification::docker::show_logs() {
    if common::container_exists; then
        docker logs --tail 100 "$SPEAKER_VERIFICATION_CONTAINER_NAME" 2>&1
    else
        log::error "Container does not exist"
        return 1
    fi
}
export -f speaker_verification::docker::show_logs

#######################################
# Show container stats
#######################################
speaker_verification::docker::show_stats() {
    if common::is_running; then
        docker stats --no-stream "$SPEAKER_VERIFICATION_CONTAINER_NAME" 2>/dev/null
    else
        log::error "$MSG_CONTAINER_NOT_RUNNING"
        return 1
    fi
}
export -f speaker_verification::docker::show_stats
