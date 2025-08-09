#!/usr/bin/env bash
set -euo pipefail

APP_LIFECYCLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO="Agent Metareasoning Manager"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_PORT_REGISTRY_FILE}"

main() {
    log::header "Initializing '${SCENARIO}'..."

    # Inject workflows, sql, etc.
    "${var_RUNTIME_ENGINE}" "${var_SERVICE_JSON_FILE}"

    log::success "🚀 '${SCENARIO}' is initialized"
    log::info "Dashboard available at http://localhost:${RESOURCE_PORTS[windmill]}
    log::info "n8n workflows at http://localhost:${RESOURCE_PORTS[n8n]}"
}

main "$@"