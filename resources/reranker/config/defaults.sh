#!/usr/bin/env bash
# Reranker resource configuration defaults.
# Sourced by test/integration-test.sh. The compose-service lifecycle itself is
# declarative (resource.json) — these values mirror the manifest so the smoke
# test can reach the running container without recomputing them.

#######################################
# Export configuration constants. Idempotent — safe to call multiple times.
#######################################
defaults::export_config() {
    if [[ -z "${RERANKER_PORT:-}" ]]; then
        readonly RERANKER_PORT="${RERANKER_CUSTOM_PORT:-11453}"
    fi
    if [[ -z "${RERANKER_BASE_URL:-}" ]]; then
        readonly RERANKER_BASE_URL="http://localhost:${RERANKER_PORT}"
    fi
    if [[ -z "${RERANKER_CONTAINER_NAME:-}" ]]; then
        readonly RERANKER_CONTAINER_NAME="reranker"
    fi
    if [[ -z "${RERANKER_MODEL:-}" ]]; then
        readonly RERANKER_MODEL="BAAI/bge-reranker-v2-m3"
    fi
    if [[ -z "${RERANKER_API_TIMEOUT:-}" ]]; then
        readonly RERANKER_API_TIMEOUT="30"
    fi
}
