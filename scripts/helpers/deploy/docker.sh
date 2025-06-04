#!/usr/bin/env bash
set -euo pipefail

DEPLOY_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${DEPLOY_DIR}/../utils/docker.sh"
# shellcheck disable=SC1091
source "${DEPLOY_DIR}/../utils/env.sh"
# shellcheck disable=SC1091
source "${DEPLOY_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${DEPLOY_DIR}/../utils/var.sh"

# Placeholder for Docker deployment logic
deploy::deploy_docker() {
  local artifact_dir="$1"

  # Load Docker images
  if [[ -f "$artifact_dir/docker-images.tar" ]]; then
    docker::load_images_from_tar "$artifact_dir/docker-images.tar"
  fi

  # Select compose file based on target environment
  local compose_file="$(docker::get_compose_file)"

  # Navigate to project root
  pushd "$var_ROOT_DIR" >/dev/null || {
    log::error "Failed to change directory to project root: $var_ROOT_DIR"
    return 1
  }

  docker-compose -f "$compose_file" down
  docker::kill_all
  log::info "Starting Docker containers in detached mode"
  docker-compose -f "$compose_file" up -d --remove-orphans

  # Return to original directory
  popd >/dev/null

  log::success "✅ Docker deployment completed for environment: $ENVIRONMENT"
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    deploy::deploy_docker "$@"
fi
