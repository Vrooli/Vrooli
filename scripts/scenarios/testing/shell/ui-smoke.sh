#!/usr/bin/env bash
# Shared UI smoke harness powered by Browserless
set -euo pipefail

SHELL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SHELL_DIR}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${SHELL_DIR}/core.sh"
# shellcheck disable=SC1091
source "${SHELL_DIR}/runtime.sh"

readonly TESTING_UI_SMOKE_EXIT_BROWSERLESS_OFFLINE=50
readonly TESTING_UI_SMOKE_EXIT_BUNDLE_STALE=60

# Run the Browserless-backed UI smoke check
# Options:
#   --scenario <name>
#   --scenario-dir <path>
#   --json (emit JSON summary instead of text)
testing::ui_smoke::run() {
    local scenario_name=""
    local scenario_dir=""
    local output_mode="text"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --scenario)
                scenario_name="$2"
                shift 2
                ;;
            --scenario-dir)
                scenario_dir="$2"
                shift 2
                ;;
            --json)
                output_mode="json"
                shift
                ;;
            *)
                log::error "Unknown flag for ui-smoke: $1"
                return 1
                ;;
        esac
    done

    if [[ -z "$scenario_dir" ]]; then
        scenario_dir="${TESTING_PHASE_SCENARIO_DIR:-$PWD}"
    fi
    scenario_dir="$(cd "$scenario_dir" && pwd)"

    if [[ -z "$scenario_name" ]]; then
        scenario_name="${TESTING_PHASE_SCENARIO_NAME:-$(basename "$scenario_dir")}"
    fi

    local runtime_quiet_prev="${TESTING_RUNTIME_QUIET:-false}"
    trap "TESTING_RUNTIME_QUIET=${runtime_quiet_prev}" RETURN

    local ui_package_json="$scenario_dir/ui/package.json"
    if [[ ! -f "$ui_package_json" ]]; then
        local skip_json
        skip_json=$(testing::ui_smoke::_build_summary_json \
            --scenario "$scenario_name" \
            --status "skipped" \
            --message "UI directory not detected" \
            --scenario-dir "$scenario_dir")
        testing::ui_smoke::_persist_summary "$skip_json" "$scenario_dir" "$scenario_name"
        testing::ui_smoke::_emit_summary "$skip_json" "$output_mode"
        return 0
    fi

    local enabled="true"
    local timeout_ms="90000"
    local handshake_timeout_ms="15000"
    local config_file="$scenario_dir/.vrooli/testing.json"
    if [[ -f "$config_file" ]] && command -v jq >/dev/null 2>&1; then
        enabled=$(jq -r '.structure.ui_smoke.enabled // true' "$config_file" 2>/dev/null || echo "true")
        timeout_ms=$(jq -r '.structure.ui_smoke.timeout_ms // 90000' "$config_file" 2>/dev/null || echo "90000")
        handshake_timeout_ms=$(jq -r '.structure.ui_smoke.handshake_timeout_ms // 15000' "$config_file" 2>/dev/null || echo "15000")
    fi

    if [[ "$enabled" != "true" && "$enabled" != "True" ]]; then
        local disabled_json
        disabled_json=$(testing::ui_smoke::_build_summary_json \
            --scenario "$scenario_name" \
            --status "skipped" \
            --message "UI smoke harness disabled via .vrooli/testing.json" \
            --scenario-dir "$scenario_dir")
        testing::ui_smoke::_persist_summary "$disabled_json" "$scenario_dir" "$scenario_name"
        testing::ui_smoke::_emit_summary "$disabled_json" "$output_mode"
        return 0
    fi

    local bundle_info_json
    bundle_info_json=$(testing::ui_smoke::_check_bundle_freshness "$scenario_dir" 2>/dev/null)
    local bundle_fresh
    bundle_fresh=$(echo "$bundle_info_json" | jq -r '.fresh // true')

    if [[ "$bundle_fresh" != "true" ]]; then
        local bundle_reason
        bundle_reason=$(echo "$bundle_info_json" | jq -r '.reason // "UI bundle missing or outdated"')
        local stale_json
        stale_json=$(testing::ui_smoke::_build_summary_json \
            --scenario "$scenario_name" \
            --status "blocked" \
            --message "$bundle_reason" \
            --scenario-dir "$scenario_dir" \
            --bundle "$bundle_info_json")
        testing::ui_smoke::_persist_summary "$stale_json" "$scenario_dir" "$scenario_name"
        testing::ui_smoke::_emit_summary "$stale_json" "$output_mode"
        return $TESTING_UI_SMOKE_EXIT_BUNDLE_STALE
    fi

    local browserless_json
    if ! browserless_json=$(testing::ui_smoke::_browserless_status); then
        local offline_json
        offline_json=$(testing::ui_smoke::_build_summary_json \
            --scenario "$scenario_name" \
            --status "blocked" \
            --message "Browserless resource is offline" \
            --scenario-dir "$scenario_dir")
        testing::ui_smoke::_persist_summary "$offline_json" "$scenario_dir" "$scenario_name"
        testing::ui_smoke::_emit_summary "$offline_json" "$output_mode"
        return $TESTING_UI_SMOKE_EXIT_BROWSERLESS_OFFLINE
    fi

    local browserless_running
    browserless_running=$(echo "$browserless_json" | jq -r '.running // false')
    local browserless_port
    browserless_port=$(echo "$browserless_json" | jq -r '.configuration.port // 4110')

    if [[ "$browserless_running" != "true" ]]; then
        local offline_json
        offline_json=$(testing::ui_smoke::_build_summary_json \
            --scenario "$scenario_name" \
            --status "blocked" \
            --message "Browserless resource is offline" \
            --scenario-dir "$scenario_dir" \
            --browserless "$browserless_json")
        testing::ui_smoke::_persist_summary "$offline_json" "$scenario_dir" "$scenario_name"
        testing::ui_smoke::_emit_summary "$offline_json" "$output_mode"
        return $TESTING_UI_SMOKE_EXIT_BROWSERLESS_OFFLINE
    fi

    local runtime_managed=false
    if declare -f testing::runtime::configure >/dev/null 2>&1; then
        TESTING_RUNTIME_QUIET=true
        testing::runtime::configure "$scenario_name" true
        runtime_managed=true
        if ! testing::runtime::ensure_available "ui-smoke" true; then
            testing::runtime::cleanup || true
            local runtime_json
            runtime_json=$(testing::ui_smoke::_build_summary_json \
                --scenario "$scenario_name" \
                --status "failed" \
                --message "Failed to auto-start scenario runtime for UI smoke. Run 'vrooli scenario start ${scenario_name}' (or '--setup') to inspect lifecycle logs." \
                --scenario-dir "$scenario_dir")
            testing::ui_smoke::_persist_summary "$runtime_json" "$scenario_dir" "$scenario_name"
            testing::ui_smoke::_emit_summary "$runtime_json" "$output_mode"
            return 1
        fi
        testing::runtime::discover_ports "$scenario_name"
    fi

    local ui_port="${UI_PORT:-}"
    if [[ -z "$ui_port" ]]; then
        ui_port=$(vrooli scenario port "$scenario_name" UI_PORT 2>/dev/null || echo "")
    fi

    if [[ -z "$ui_port" ]]; then
        if [[ "$runtime_managed" = true ]]; then
            testing::runtime::cleanup || true
        fi
        local no_port_json
        no_port_json=$(testing::ui_smoke::_build_summary_json \
            --scenario "$scenario_name" \
            --status "skipped" \
            --message "Scenario does not expose a UI port" \
            --scenario-dir "$scenario_dir")
        testing::ui_smoke::_persist_summary "$no_port_json" "$scenario_dir" "$scenario_name"
        testing::ui_smoke::_emit_summary "$no_port_json" "$output_mode"
        return 0
    fi

    local ui_url="http://localhost:${ui_port}"
    local iframe_json
    iframe_json=$(testing::ui_smoke::_check_iframe_bridge "$scenario_dir")
    local dependency_present
    dependency_present=$(echo "$iframe_json" | jq -r '.dependency_present // false')
    if [[ "$dependency_present" != "true" ]]; then
        if [[ "$runtime_managed" = true ]]; then
            testing::runtime::cleanup || true
        fi
        local bridge_json
        bridge_json=$(testing::ui_smoke::_build_summary_json \
            --scenario "$scenario_name" \
            --status "failed" \
            --message "@vrooli/iframe-bridge dependency missing in ui/package.json" \
            --scenario-dir "$scenario_dir" \
            --iframe "$iframe_json")
        testing::ui_smoke::_persist_summary "$bridge_json" "$scenario_dir" "$scenario_name"
        testing::ui_smoke::_emit_summary "$bridge_json" "$output_mode"
        return 1
    fi

    local js_payload
    js_payload=$(testing::ui_smoke::_build_browserless_script "$ui_url" "$timeout_ms" "$handshake_timeout_ms")

    local start_ms
    start_ms=$(date +%s%3N)
    local raw_result
    raw_result=$(curl -s -X POST \
        -H "Content-Type: application/javascript" \
        --data "$js_payload" \
        "http://localhost:${browserless_port}/chrome/function")
    local curl_status=$?
    local end_ms
    end_ms=$(date +%s%3N)
    local duration_ms=$((end_ms - start_ms))

    if [[ $curl_status -ne 0 ]]; then
        if [[ "$runtime_managed" = true ]]; then
            testing::runtime::cleanup || true
        fi
        local curl_json
        curl_json=$(testing::ui_smoke::_build_summary_json \
            --scenario "$scenario_name" \
            --status "failed" \
            --message "Failed to reach Browserless API" \
            --scenario-dir "$scenario_dir" \
            --browserless "$browserless_json")
        testing::ui_smoke::_persist_summary "$curl_json" "$scenario_dir" "$scenario_name"
        testing::ui_smoke::_emit_summary "$curl_json" "$output_mode"
        return 1
    fi

    if [[ -z "$raw_result" ]] || ! echo "$raw_result" | jq -e . >/dev/null 2>&1; then
        if [[ "$runtime_managed" = true ]]; then
            testing::runtime::cleanup || true
        fi
        local invalid_json
        invalid_json=$(testing::ui_smoke::_build_summary_json \
            --scenario "$scenario_name" \
            --status "failed" \
            --message "Browserless returned invalid response" \
            --scenario-dir "$scenario_dir" \
            --browserless "$browserless_json")
        testing::ui_smoke::_persist_summary "$invalid_json" "$scenario_dir" "$scenario_name"
        testing::ui_smoke::_emit_summary "$invalid_json" "$output_mode"
        return 1
    fi

    local coverage_dir="$scenario_dir/coverage/${scenario_name}/ui-smoke"
    mkdir -p "$coverage_dir"

    local artifact_json
    artifact_json=$(testing::ui_smoke::_write_artifacts "$scenario_dir" "$scenario_name" "$coverage_dir" "$raw_result")

    local raw_without_screenshot
    raw_without_screenshot=$(echo "$raw_result" | jq 'del(.screenshot)' 2>/dev/null || echo '{}')

    local success
    success=$(echo "$raw_result" | jq -r '.success // false')
    local handshake_signaled
    handshake_signaled=$(echo "$raw_result" | jq -r '.handshake.signaled // false')

    local status="passed"
    local message="UI loaded successfully"
    if [[ "$success" != "true" ]]; then
        status="failed"
        message=$(echo "$raw_result" | jq -r '.error // "Browserless execution failed"')
    elif [[ "$handshake_signaled" != "true" ]]; then
        status="failed"
        message="Iframe bridge never signaled ready"
    fi

    local summary_json
    summary_json=$(testing::ui_smoke::_build_summary_json \
        --scenario "$scenario_name" \
        --status "$status" \
        --message "$message" \
        --scenario-dir "$scenario_dir" \
        --browserless "$browserless_json" \
        --bundle "$bundle_info_json" \
        --iframe "$iframe_json" \
        --artifacts "$artifact_json" \
        --raw "$raw_without_screenshot" \
        --ui-url "$ui_url" \
        --duration "$duration_ms")

    testing::ui_smoke::_persist_summary "$summary_json" "$scenario_dir" "$scenario_name"

    if [[ "$runtime_managed" = true ]]; then
        testing::runtime::cleanup || true
    fi

    testing::ui_smoke::_emit_summary "$summary_json" "$output_mode"

    if [[ "$status" = "passed" ]]; then
        return 0
    else
        return 1
    fi
}

# Run bundle freshness check using ui-bundle setup helper
testing::ui_smoke::_check_bundle_freshness() {
    local scenario_dir="$1"
    local service_json="$scenario_dir/.vrooli/service.json"
    local config='{}'
    if [[ -f "$service_json" ]] && command -v jq >/dev/null 2>&1; then
        local candidate
        candidate=$(jq -c '.lifecycle.setup.condition.checks[]? | select(.type == "ui-bundle")' "$service_json" 2>/dev/null | head -n 1)
        if [[ -n "$candidate" ]]; then
            config="$candidate"
        fi
    fi

    local output=""
    local result
    set +e
    output=$(APP_ROOT="$scenario_dir" "$APP_ROOT/scripts/lib/setup-conditions/ui-bundle-check.sh" "$config" 2>&1)
    result=$?
    set -e

    if [[ $result -eq 0 ]]; then
        jq -n --arg reason "${output:-UI bundle missing or outdated}" --argjson config "$config" '{fresh:false, reason:$reason, config:$config}'
    else
        jq -n --argjson config "$config" '{fresh:true, reason:"", config:$config}'
    fi
}

# Fetch Browserless status as JSON
testing::ui_smoke::_browserless_status() {
    if ! command -v resource-browserless >/dev/null 2>&1; then
        return 1
    fi
    resource-browserless status --format json 2>/dev/null
}

# Ensure iframe-bridge dependency exists and capture metadata
testing::ui_smoke::_check_iframe_bridge() {
    local scenario_dir="$1"
    local package_json="$scenario_dir/ui/package.json"
    if [[ ! -f "$package_json" ]] || ! command -v jq >/dev/null 2>&1; then
        jq -n '{dependency_present:false, details:"ui/package.json missing or jq unavailable"}'
        return
    fi

    local version
    version=$(jq -r '.dependencies["@vrooli/iframe-bridge"] // .devDependencies["@vrooli/iframe-bridge"] // empty' "$package_json" 2>/dev/null)
    if [[ -z "$version" || "$version" == "null" ]]; then
        jq -n '{dependency_present:false, details:"@vrooli/iframe-bridge not listed"}'
    else
        jq -n --arg version "$version" '{dependency_present:true, version:$version}'
    fi
}

# Build Browserless JS payload for smoke session
testing::ui_smoke::_build_browserless_script() {
    local url="$1"
    local timeout_ms="$2"
    local handshake_timeout_ms="$3"
    local js_template
    read -r -d '' js_template <<'JSCODE'
export default async ({ page }) => {
    const result = {
        success: false,
        console: [],
        network: [],
        pageErrors: [],
        handshake: { signaled: false, timedOut: false, durationMs: 0 },
        screenshot: null,
        html: '',
        title: '',
        url: '',
        error: null,
        timings: {}
    };

    const handshakeCheck = () => (
        (typeof window !== 'undefined') && (
            window.__vrooliBridgeChildInstalled === true ||
            window.IFRAME_BRIDGE_READY === true ||
            (window.IframeBridge && window.IframeBridge.ready === true) ||
            (window.iframeBridge && window.iframeBridge.ready === true) ||
            (window.IframeBridge && typeof window.IframeBridge.getState === 'function' && window.IframeBridge.getState().ready === true)
        )
    );

    try {
        page.on('console', msg => {
            result.console.push({
                level: msg.type(),
                message: msg.text(),
                timestamp: new Date().toISOString()
            });
        });
        page.on('pageerror', error => {
            result.pageErrors.push({
                message: error.message,
                stack: error.stack || null,
                timestamp: new Date().toISOString()
            });
        });
        page.on('requestfailed', request => {
            result.network.push({
                url: request.url(),
                method: request.method(),
                resourceType: request.resourceType(),
                status: request.response() ? request.response().status() : null,
                errorText: request.failure() ? request.failure().errorText : null,
                timestamp: new Date().toISOString()
            });
        });
        page.on('response', response => {
            if (response.status() >= 400) {
                result.network.push({
                    url: response.url(),
                    method: response.request().method(),
                    resourceType: response.request().resourceType(),
                    status: response.status(),
                    errorText: null,
                    timestamp: new Date().toISOString()
                });
            }
        });

        await page.setViewport({ width: 1280, height: 720 });

        const targetUrl = __URL_JSON__;
        const hostMarkup = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8" />
    <style>
      html, body { margin: 0; padding: 0; background: #050505; height: 100%; }
      #ui-smoke-frame { border: 0; width: 100%; height: 100vh; }
    </style>
  </head>
  <body>
    <iframe id="ui-smoke-frame" src=${targetUrl} allow="clipboard-read; clipboard-write"></iframe>
  </body>
</html>`;

        const startTime = Date.now();
        await page.setContent(hostMarkup, { waitUntil: 'load' });
        const frameElement = await page.waitForSelector('#ui-smoke-frame');
        const frame = await frameElement.contentFrame();
        if (!frame) {
            throw new Error('Failed to attach to scenario frame');
        }

        await frame.waitForSelector('body', { timeout: __TIMEOUT_MS__ });
        const afterGoto = Date.now();

        try {
            const handshakeStart = Date.now();
            await frame.waitForFunction(handshakeCheck, { timeout: __HANDSHAKE_TIMEOUT_MS__ });
            result.handshake.signaled = true;
            result.handshake.durationMs = Date.now() - handshakeStart;
        } catch (handshakeError) {
            result.handshake.timedOut = true;
            result.handshake.error = handshakeError.message;
        }

        await new Promise(resolve => setTimeout(resolve, 1000));
        const screenshotBuffer = await frameElement.screenshot({ type: 'png' });

        result.screenshot = screenshotBuffer.toString('base64');
        result.html = await frame.content();
        result.title = await frame.title();
        result.url = frame.url();
        result.timings = {
            gotoMs: afterGoto - startTime,
            totalMs: Date.now() - startTime
        };
        result.success = true;
        return result;
    } catch (error) {
        result.success = false;
        result.error = error.message;
        result.stack = error.stack || null;
        return result;
    }
};
JSCODE

    local url_json
    url_json=$(python3 - "$url" <<'PY'
import json, sys
print(json.dumps(sys.argv[1]))
PY
)

    js_template=${js_template/__URL_JSON__/$url_json}
    js_template=${js_template/__TIMEOUT_MS__/$timeout_ms}
    js_template=${js_template/__HANDSHAKE_TIMEOUT_MS__/$handshake_timeout_ms}
    printf '%s' "$js_template"
}

# Persist artifacts (screenshot, console, network, html, raw response) and return JSON metadata
testing::ui_smoke::_write_artifacts() {
    local scenario_dir="$1"
    local scenario_name="$2"
    local coverage_dir="$3"
    local raw_result="$4"

    local screenshot_path="$coverage_dir/screenshot.png"
    local console_path="$coverage_dir/console.json"
    local network_path="$coverage_dir/network.json"
    local html_path="$coverage_dir/dom.html"
    local raw_path="$coverage_dir/raw.json"

    local screenshot_b64
    screenshot_b64=$(echo "$raw_result" | jq -r '.screenshot // empty')
    if [[ -n "$screenshot_b64" && "$screenshot_b64" != "null" ]]; then
        printf '%s' "$screenshot_b64" | base64 -d > "$screenshot_path" 2>/dev/null || true
    fi

    echo "$raw_result" | jq '.console // []' > "$console_path"
    echo "$raw_result" | jq '.network // []' > "$network_path"
    echo "$raw_result" | jq -r '.html // ""' > "$html_path"
    echo "$raw_result" | jq 'del(.screenshot)' > "$raw_path"

    local screenshot_rel="${screenshot_path#$APP_ROOT/}"
    local console_rel="${console_path#$APP_ROOT/}"
    local network_rel="${network_path#$APP_ROOT/}"
    local html_rel="${html_path#$APP_ROOT/}"
    local raw_rel="${raw_path#$APP_ROOT/}"

    if [[ ! -s "$screenshot_path" ]]; then
        screenshot_rel=""
    fi

    jq -n \
        --arg screenshot "$screenshot_rel" \
        --arg console "$console_rel" \
        --arg network "$network_rel" \
        --arg html "$html_rel" \
        --arg raw "$raw_rel" \
        '{screenshot:$screenshot, console:$console, network:$network, html:$html, raw:$raw}'
}

testing::ui_smoke::_write_temp_json() {
    local content="$1"
    local scenario_dir="$2"
    local scenario_name="$3"

    local tmp_root="$scenario_dir/coverage/${scenario_name}/ui-smoke/tmp"
    mkdir -p "$tmp_root"
    local tmp_file
    tmp_file=$(mktemp "$tmp_root/tmp-XXXX.json")

    if [[ -z "$content" || "$content" == "null" ]]; then
        content='{}'
    fi

    printf '%s' "$content" > "$tmp_file"
    echo "$tmp_file"
}

# Construct summary JSON with optional attachments
testing::ui_smoke::_build_summary_json() {
    local scenario=""
    local status="failed"
    local message=""
    local scenario_dir=""
    local browserless="{}"
    local bundle="{}"
    local iframe="{}"
    local artifacts="{}"
    local raw="{}"
    local ui_url=""
    local duration_ms=0

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --scenario)
                scenario="$2"
                shift 2
                ;;
            --status)
                status="$2"
                shift 2
                ;;
            --message)
                message="$2"
                shift 2
                ;;
            --scenario-dir)
                scenario_dir="$2"
                shift 2
                ;;
            --browserless)
                browserless="$2"
                shift 2
                ;;
            --bundle)
                bundle="$2"
                shift 2
                ;;
            --iframe)
                iframe="$2"
                shift 2
                ;;
            --artifacts)
                artifacts="$2"
                shift 2
                ;;
            --raw)
                raw="$2"
                shift 2
                ;;
            --ui-url)
                ui_url="$2"
                shift 2
                ;;
            --duration)
                duration_ms="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    local -a _ui_smoke_tmp_files=()

    local browserless_file
    browserless_file=$(testing::ui_smoke::_write_temp_json "$browserless" "$scenario_dir" "$scenario")
    _ui_smoke_tmp_files+=("$browserless_file")

    local bundle_file
    bundle_file=$(testing::ui_smoke::_write_temp_json "$bundle" "$scenario_dir" "$scenario")
    _ui_smoke_tmp_files+=("$bundle_file")

    local iframe_file
    iframe_file=$(testing::ui_smoke::_write_temp_json "$iframe" "$scenario_dir" "$scenario")
    _ui_smoke_tmp_files+=("$iframe_file")

    local artifacts_file
    artifacts_file=$(testing::ui_smoke::_write_temp_json "$artifacts" "$scenario_dir" "$scenario")
    _ui_smoke_tmp_files+=("$artifacts_file")

    local raw_file
    raw_file=$(testing::ui_smoke::_write_temp_json "$raw" "$scenario_dir" "$scenario")
    _ui_smoke_tmp_files+=("$raw_file")

    local summary
    summary=$(jq -n \
        --arg scenario "$scenario" \
        --arg status "$status" \
        --arg message "$message" \
        --arg scenario_dir "$scenario_dir" \
        --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        --arg ui_url "$ui_url" \
        --argjson duration "$duration_ms" \
        --rawfile browserless "$browserless_file" \
        --rawfile bundle "$bundle_file" \
        --rawfile iframe "$iframe_file" \
        --rawfile artifacts "$artifacts_file" \
        --rawfile raw "$raw_file" \
        '($browserless | fromjson) as $browserless_data |
         ($bundle | fromjson) as $bundle_data |
         ($iframe | fromjson) as $iframe_data |
         ($artifacts | fromjson) as $artifacts_data |
         ($raw | fromjson) as $raw_data |
         {scenario:$scenario,status:$status,message:$message,timestamp:$timestamp,ui_url:$ui_url,duration_ms:$duration,browserless:$browserless_data,bundle:$bundle_data,iframe_bridge:$iframe_data,artifacts:$artifacts_data,raw:$raw_data}')

    rm -f "${_ui_smoke_tmp_files[@]}"

    echo "$summary"
}

# Persist latest summary JSON to coverage directory
testing::ui_smoke::_persist_summary() {
    local summary_json="$1"
    local scenario_dir="$2"
    local scenario_name="$3"

    local coverage_dir="$scenario_dir/coverage/${scenario_name}/ui-smoke"
    mkdir -p "$coverage_dir"
    echo "$summary_json" > "$coverage_dir/latest.json"
}

# Emit summary according to requested mode
testing::ui_smoke::_emit_summary() {
    local summary_json="$1"
    local mode="$2"

    if [[ "$mode" = "json" ]]; then
        echo "$summary_json"
        return
    fi

    local status message ui_url duration
    status=$(echo "$summary_json" | jq -r '.status')
    message=$(echo "$summary_json" | jq -r '.message')
    ui_url=$(echo "$summary_json" | jq -r '.ui_url // ""')
    duration=$(echo "$summary_json" | jq -r '.duration_ms // 0')

    case "$status" in
        passed)
            log::success "UI smoke passed (${duration}ms): $ui_url"
            ;;
        skipped)
            log::warning "UI smoke skipped: $message"
            ;;
        blocked)
            log::error "UI smoke blocked: $message"
            ;;
        failed)
            log::error "UI smoke failed: $message"
            ;;
        *)
            log::info "UI smoke status ($status): $message"
            ;;
    esac
}

if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    export -f testing::ui_smoke::run
fi

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    testing::ui_smoke::run "$@"
fi
