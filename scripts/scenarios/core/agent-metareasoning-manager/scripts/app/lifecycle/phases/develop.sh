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

# Main deployment flow
main() {
    log::header "Starting '${SCENARIO}'..."

    # Inject workflows, sql, etc.
    "${var_RUNTIME_ENGINE}" "${SCRIPT_DIR}/../.vrooli/service.json"

    log::success "🚀 '${SCENARIO}' is running"
    log::info "Dashboard available at http://localhost:${RESOURCE_PORTS[windmill]}
    log::info "n8n workflows at http://localhost:${RESOURCE_PORTS[n8n]}"
}

main "$@"