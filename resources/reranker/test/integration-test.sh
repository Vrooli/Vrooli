#!/usr/bin/env bash
# Reranker resource integration test.
#
# Smoke-tests a RUNNING reranker (TEI) container: health, info, and — the one
# that matters — that /rerank actually orders an obviously-relevant passage
# above noise. Self-contained (no shared harness); start the resource first:
#   vrooli resource start reranker && bash resources/reranker/test/integration-test.sh

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"

# shellcheck disable=SC1091
source "${RESOURCE_DIR}/config/messages.sh"
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/config/defaults.sh"
defaults::export_config

BASE_URL="${RERANKER_BASE_URL:-http://localhost:11453}"
TIMEOUT="${RERANKER_API_TIMEOUT:-30}"

PASS=0
FAIL=0

require_tools() {
    local missing=()
    for t in curl jq; do
        command -v "$t" >/dev/null 2>&1 || missing+=("$t")
    done
    if ((${#missing[@]} > 0)); then
        msg::error "missing required tools: ${missing[*]}"
        exit 1
    fi
}

test_health() {
    if curl -fsS --max-time "$TIMEOUT" "${BASE_URL}/health" >/dev/null 2>&1; then
        msg::ok "health: ${BASE_URL}/health returned 200"
        PASS=$((PASS + 1))
    else
        msg::error "health: ${BASE_URL}/health not ready"
        FAIL=$((FAIL + 1))
    fi
}

test_info() {
    local resp model
    resp=$(curl -fsS --max-time "$TIMEOUT" "${BASE_URL}/info" 2>/dev/null || echo "")
    model=$(echo "$resp" | jq -r '.model_id // empty' 2>/dev/null || echo "")
    if [[ -n "$model" ]]; then
        msg::ok "info: serving model_id=${model}"
        PASS=$((PASS + 1))
    else
        msg::error "info: /info did not report a model_id (resp: ${resp:0:120})"
        FAIL=$((FAIL + 1))
    fi
}

# The substantive test: a clearly-relevant passage must outrank noise.
test_rerank_ordering() {
    local query='How do I restart a scenario from the command line?'
    local payload top_index
    payload=$(jq -n --arg q "$query" '{
        query: $q,
        texts: [
            "The mitochondria is the powerhouse of the cell.",
            "Use the CLI to restart a scenario: vrooli scenario restart <name>.",
            "Bananas are a good source of potassium."
        ],
        raw_scores: false,
        return_text: false
    }')

    local resp
    resp=$(curl -fsS --max-time "$TIMEOUT" -X POST "${BASE_URL}/rerank" \
        -H 'Content-Type: application/json' -d "$payload" 2>/dev/null || echo "")

    # TEI returns results sorted by score descending; the first element's index
    # is the most-relevant passage. Index 1 is the scenario-restart sentence.
    top_index=$(echo "$resp" | jq -r '.[0].index // empty' 2>/dev/null || echo "")
    if [[ "$top_index" == "1" ]]; then
        msg::ok "rerank: relevant passage (index 1) ranked first"
        PASS=$((PASS + 1))
    else
        msg::error "rerank: expected top index 1, got '${top_index}' (resp: ${resp:0:160})"
        FAIL=$((FAIL + 1))
    fi
}

test_container_status() {
    if ! command -v docker >/dev/null 2>&1; then
        msg::warn "container: docker not available, skipping"
        return 0
    fi
    if docker ps --format '{{.Names}}' | grep -q "^${RERANKER_CONTAINER_NAME}$"; then
        msg::ok "container: ${RERANKER_CONTAINER_NAME} is running"
        PASS=$((PASS + 1))
    else
        msg::warn "container: ${RERANKER_CONTAINER_NAME} not found (may be running under a different name)"
    fi
}

main() {
    require_tools
    msg::info "reranker integration test → ${BASE_URL}"
    test_health
    test_info
    test_rerank_ordering
    test_container_status
    echo
    if ((FAIL > 0)); then
        msg::error "integration test FAILED (${PASS} passed, ${FAIL} failed)"
        exit 1
    fi
    msg::ok "integration test PASSED (${PASS} checks)"
}

main "$@"
