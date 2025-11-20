#!/usr/bin/env bash

scenario::requirements::dispatch() {
    if [[ $# -eq 0 ]]; then
        scenario::requirements::help
        return 0
    fi

    local subcommand="$1"

    case "$subcommand" in
        help|--help|-h)
            scenario::requirements::help
            return 0
            ;;
        init)
            shift
            scenario::requirements::init "$@"
            return
            ;;
        report)
            shift
            scenario::requirements::report "$@"
            return
            ;;
        validate)
            shift
            scenario::requirements::validate "$@"
            return
            ;;
        sync)
            shift
            scenario::requirements::sync "$@"
            return
            ;;
        manual-log)
            shift
            scenario::requirements::manual_log "$@"
            return
            ;;
        snapshot)
            shift
            scenario::requirements::snapshot "$@"
            return
            ;;
        lint-prd)
            shift
            scenario::requirements::lint_prd "$@"
            return
            ;;
        phase|phase-inspect)
            shift
            scenario::requirements::phase_inspect "$@"
            return
            ;;
    esac

    # Backward compatibility: treat first argument as scenario name or option for report
    scenario::requirements::report "$@"
}

scenario::requirements::report() {
    local scenario_name=""
    local format="markdown"
    local include_pending=false
    local output_path=""
    local fail_on_gap=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="${2:-markdown}"
                shift 2
                ;;
            --format=*)
                format="${1#*=}"
                shift
                ;;
            --include-pending)
                include_pending=true
                shift
                ;;
            --output)
                output_path="$2"
                shift 2
                ;;
            --output=*)
                output_path="${1#*=}"
                shift
                ;;
            --fail-on-critical-gap)
                fail_on_gap=true
                shift
                ;;
            --help|-h)
                scenario::requirements::help
                return 0
                ;;
            *)
                if [ -z "$scenario_name" ]; then
                    scenario_name="$1"
                    shift
                else
                    log::error "Unexpected argument: $1"
                    return 1
                fi
                ;;
        esac
    done

    if [ -z "$scenario_name" ]; then
        log::error "Scenario name is required"
        scenario::requirements::help
        return 1
    fi

    case "$format" in
        json|markdown|trace)
            ;;
        *)
            log::error "Unsupported format: $format"
            return 1
            ;;
    esac

    local scenario_dir="${APP_ROOT}/scenarios/${scenario_name}"
    if [ ! -d "$scenario_dir" ]; then
        log::error "Scenario directory not found: $scenario_dir"
        return 1
    fi

    if [ ! -d "${scenario_dir}/requirements" ]; then
        log::error "Scenario ${scenario_name} does not define requirements/"
        log::info "Initialize it with: vrooli scenario requirements init ${scenario_name}"
        return 1
    fi

    if ! command -v node >/dev/null 2>&1; then
        log::error "Node.js is required to run the requirements reporter"
        return 1
    fi

    local reporter="${APP_ROOT}/scripts/requirements/report.js"
    if [ ! -f "$reporter" ]; then
        log::error "Requirements reporter not found at $reporter"
        return 1
    fi

    local include_flag=""
    if [ "$include_pending" = true ]; then
        include_flag="--include-pending"
    fi

    local temp_json
    temp_json=$(mktemp)
    (cd "$scenario_dir" && node "$reporter" --scenario "$scenario_name" --format json $include_flag --output "$temp_json")

    local critical_gap
    critical_gap=$(node -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(process.argv[1],'utf8')); process.stdout.write(String(data?.summary?.criticalityGap ?? 0));" "$temp_json")

    local final_status=0
    if [ "$format" = "json" ]; then
        if [ -n "$output_path" ]; then
            mkdir -p "$(dirname "$output_path")"
            mv "$temp_json" "$output_path"
        else
            cat "$temp_json"
        fi
    else
        if [ -n "$output_path" ]; then
            mkdir -p "$(dirname "$output_path")"
            (cd "$scenario_dir" && node "$reporter" --scenario "$scenario_name" --format "$format" $include_flag --output "$output_path")
        else
            (cd "$scenario_dir" && node "$reporter" --scenario "$scenario_name" --format "$format" $include_flag)
        fi
        rm -f "$temp_json"
    fi

    if [ "$fail_on_gap" = true ] && [ "${critical_gap}" != "0" ]; then
        log::error "Criticality gap detected (${critical_gap}) for scenario ${scenario_name}"
        final_status=1
    fi

    return $final_status
}

scenario::requirements::help() {
    cat <<'__REQ_HELP__'
Usage: vrooli scenario requirements <subcommand> [options]

Subcommands:
  help                               Show this message
  report <name> [options]            Generate coverage summary (default when no subcommand provided)
    --format <json|markdown|trace>   Output format (default: markdown)
    --include-pending               Include pending requirements in the report
    --output <path>                 Write report to file instead of stdout
    --fail-on-critical-gap          Exit with non-zero status if critical P0/P1 requirements are incomplete

  validate <name> [--quiet]         Run schema + semantic validation (`scripts/requirements/validate.js`)
  sync <name>                       Update requirement status fields via `report.js --mode sync`
  phase <name> --phase <phase> [--output path]
                                    Inspect expected validations for a single phase (phase-inspect mode)

  manual-log <name> <requirement> [options]
                                    Record manual validation evidence (writes coverage/manual-validations/log.jsonl)
  snapshot <name>                    Print the latest requirements snapshot summary (coverage/requirements-sync/latest.json)
  lint-prd <name> [--json]           Ensure every PRD operational target maps to at least one requirement (and vice versa)

  init <name> [--force] [--template modular] [--owner contact@example.com]
                                    Scaffold the modular requirements/ registry template

Examples:
  vrooli scenario requirements report browser-automation-studio --format json
  vrooli scenario requirements validate browser-automation-studio
  vrooli scenario requirements sync browser-automation-studio
  vrooli scenario requirements phase browser-automation-studio --phase integration | jq
  vrooli scenario requirements init browser-automation-studio --force
__REQ_HELP__
}

scenario::requirements::manual_log() {
    local scenario_name=""
    local requirement_id=""
    local status="passed"
    local notes=""
    local artifact=""
    local validated_by="${VROOLI_AGENT_ID:-${USER:-manual-validator}}"
    local validated_at=""
    local expires_in_days="30"
    local expires_at=""
    local manifest_override=""
    local dry_run=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --status)
                status="${2:-passed}"
                shift 2
                ;;
            --notes)
                notes="${2:-}"
                shift 2
                ;;
            --artifact)
                artifact="${2:-}"
                shift 2
                ;;
            --validated-by)
                validated_by="${2:-$validated_by}"
                shift 2
                ;;
            --validated-at)
                validated_at="${2:-}"
                shift 2
                ;;
            --expires-in)
                expires_in_days="${2:-30}"
                shift 2
                ;;
            --expires-at)
                expires_at="${2:-}"
                shift 2
                ;;
            --manifest)
                manifest_override="${2:-}"
                shift 2
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            --requirement)
                requirement_id="${2:-}"
                shift 2
                ;;
            --help|-h)
                log::info "Usage: vrooli scenario requirements manual-log <scenario> <requirement> [options]"
                return 0
                ;;
            --*)
                log::error "Unknown option: $1"
                return 1
                ;;
            *)
                if [[ -z "$scenario_name" ]]; then
                    scenario_name="$1"
                elif [[ -z "$requirement_id" ]]; then
                    requirement_id="$1"
                else
                    log::error "Unexpected argument: $1"
                    return 1
                fi
                shift
                ;;
        esac
    done

    if [[ -z "$scenario_name" ]] || [[ -z "$requirement_id" ]]; then
        log::error "Usage: vrooli scenario requirements manual-log <scenario> <requirement> [options]"
        return 1
    fi

    if ! command -v node >/dev/null 2>&1; then
        log::error "Node.js is required to log manual validations"
        return 1
    fi

    local scenario_dir="${APP_ROOT}/scenarios/${scenario_name}"
    if [[ ! -d "$scenario_dir" ]]; then
        log::error "Scenario directory not found: $scenario_dir"
        return 1
    fi

    local manual_logger="${APP_ROOT}/scripts/requirements/manual-log.js"
    if [[ ! -f "$manual_logger" ]]; then
        log::error "Manual validation logger not found at $manual_logger"
        return 1
    fi

    local cmd=(node "$manual_logger" --scenario "$scenario_name" --requirement "$requirement_id" --status "$status" --validated-by "$validated_by")
    if [[ -n "$notes" ]]; then
        cmd+=(--notes "$notes")
    fi
    if [[ -n "$artifact" ]]; then
        cmd+=(--artifact "$artifact")
    fi
    if [[ -n "$validated_at" ]]; then
        cmd+=(--validated-at "$validated_at")
    fi
    if [[ -n "$expires_at" ]]; then
        cmd+=(--expires-at "$expires_at")
    else
        cmd+=(--expires-in "$expires_in_days")
    fi
    if [[ -n "$manifest_override" ]]; then
        cmd+=(--manifest "$manifest_override")
    fi
    if [[ "$dry_run" == true ]]; then
        cmd+=(--dry-run)
    fi

    (cd "$scenario_dir" && "${cmd[@]}")
}

scenario::requirements::snapshot() {
    local scenario_name=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --help|-h)
                echo "Usage: vrooli scenario requirements snapshot <name>"
                return 0
                ;;
            *)
                if [ -z "$scenario_name" ]; then
                    scenario_name="$1"
                    shift
                else
                    log::error "Unexpected argument: $1"
                    return 1
                fi
                ;;
        esac
    done

    if [ -z "$scenario_name" ]; then
        log::error "Scenario name is required"
        return 1
    fi

    local scenario_dir
    scenario_dir=$(scenario::requirements::_ensure_scenario_dir "$scenario_name") || return 1

    local snapshot_path="$scenario_dir/coverage/requirements-sync/latest.json"
    if [ ! -f "$snapshot_path" ]; then
        log::error "Snapshot not found: ${snapshot_path#$scenario_dir/}"
        log::info "Run the scenario test suite to regenerate requirements metadata."
        return 1
    fi

    if ! command -v jq >/dev/null 2>&1; then
        log::info "Snapshot path: ${snapshot_path}"
        log::warning "Install jq to view a structured snapshot summary"
        return 0
    fi

    local snapshot_rel="${snapshot_path#$scenario_dir/}"
    local synced_at
    synced_at=$(jq -r '.synced_at // "unknown"' "$snapshot_path")
    local manifest_log
    manifest_log=$(jq -r '.manifest_log // empty' "$snapshot_path")

    log::info "Requirements snapshot (${scenario_name})"
    echo "  File: ${snapshot_rel}"
    echo "  Synced at: ${synced_at}"
    if [ -n "$manifest_log" ]; then
        echo "  Run log: ${manifest_log}"
    fi

    mapfile -t snapshot_commands < <(jq -r '.tests_run[]? | select(length > 0)' "$snapshot_path")
    if [ ${#snapshot_commands[@]} -gt 0 ]; then
        echo "  Tests run:"
        for cmd in "${snapshot_commands[@]}"; do
            echo "    • $cmd"
        done
    fi

    local total_targets
    total_targets=$(jq -r '(.operational_targets // []) | length' "$snapshot_path")
    if [ "$total_targets" -gt 0 ]; then
        local complete_targets
        complete_targets=$(jq -r '(.operational_targets // []) | map(select(((.status // "") | ascii_downcase) == "complete")) | length' "$snapshot_path")
        echo "  Operational targets: ${complete_targets}/${total_targets} complete"
        mapfile -t incomplete_targets < <(jq -r '(.operational_targets // []) | map(select(((.status // "") | ascii_downcase) != "complete")) | map((.target_id // .folder_hint // .key // "unmapped") + " – " + (.status // "unknown")) | .[]?' "$snapshot_path")
        if [ ${#incomplete_targets[@]} -gt 0 ]; then
            echo "    Remaining targets:"
            for entry in "${incomplete_targets[@]}"; do
                echo "      - $entry"
            done
        fi
    fi

    local manual_total
    manual_total=$(jq -r '.manual_validations.total // 0' "$snapshot_path")
    if [ "$manual_total" -gt 0 ]; then
        local manual_manifest
        manual_manifest=$(jq -r '.manual_validations.manifest_path // empty' "$snapshot_path")
        echo "  Manual validations: ${manual_total}${manual_manifest:+ (manifest: ${manual_manifest})}"
        local now_epoch
        now_epoch=$(date -u +%s)
        local manual_expired
        manual_expired=$(jq -r --argjson now "$now_epoch" '(.manual_validations.entries // []) | map(select(.expires_at and ((try (.expires_at | fromdateiso8601) catch 0) < $now))) | length' "$snapshot_path" 2>/dev/null || echo "0")
        if [ "$manual_expired" -gt 0 ]; then
            local expired_list
            expired_list=$(jq -r --argjson now "$now_epoch" '(.manual_validations.entries // [])
                | map(select(.expires_at and ((try (.expires_at | fromdateiso8601) catch 0) < $now)))
                | map(.requirement_id)
                | join(", ")' "$snapshot_path")
            echo "    ↳ Expired entries (${manual_expired}): ${expired_list}"
        fi
    fi
}

scenario::requirements::lint_prd() {
    local scenario_name=""
    local json_output=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: vrooli scenario requirements lint-prd <name> [--json]"
                return 0
                ;;
            *)
                if [ -z "$scenario_name" ]; then
                    scenario_name="$1"
                    shift
                else
                    log::error "Unexpected argument: $1"
                    return 1
                fi
                ;;
        esac
    done

    if [ -z "$scenario_name" ]; then
        log::error "Scenario name is required"
        return 1
    fi

    local scenario_dir
    scenario_dir=$(scenario::requirements::_ensure_scenario_dir "$scenario_name") || return 1

    if ! command -v node >/dev/null 2>&1; then
        log::error "Node.js is required for linting PRD mappings"
        return 1
    fi

    local lint_script="${APP_ROOT}/scripts/requirements/lint-prd.js"
    if [ ! -f "$lint_script" ]; then
        log::error "Lint script not found at $lint_script"
        return 1
    fi

    local cmd=(node "$lint_script" --scenario "$scenario_name")
    if [ "$json_output" = true ]; then
        cmd+=(--format json)
    fi

    (cd "$scenario_dir" && "${cmd[@]}")
}

scenario::requirements::_ensure_scenario_dir() {
    local scenario_name="$1"
    local scenario_dir="${APP_ROOT}/scenarios/${scenario_name}"
    if [ -z "$scenario_name" ]; then
        log::error "Scenario name is required"
        return 1
    fi
    if [ ! -d "$scenario_dir" ]; then
        log::error "Scenario directory not found: $scenario_dir"
        return 1
    fi
    if [ ! -d "${scenario_dir}/requirements" ]; then
        log::error "Scenario ${scenario_name} does not define requirements/"
        log::info "Initialize it with: vrooli scenario requirements init ${scenario_name}"
        return 1
    fi
    echo "$scenario_dir"
    return 0
}

scenario::requirements::validate() {
    local scenario_name=""
    local quiet=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --quiet)
                quiet=true
                shift
                ;;
            --help|-h)
                echo "Usage: vrooli scenario requirements validate <name> [--quiet]"
                return 0
                ;;
            *)
                if [ -z "$scenario_name" ]; then
                    scenario_name="$1"
                    shift
                else
                    log::error "Unexpected argument: $1"
                    return 1
                fi
                ;;
        esac
    done

    local scenario_dir
    scenario_dir=$(scenario::requirements::_ensure_scenario_dir "$scenario_name") || return 1

    if ! command -v node >/dev/null 2>&1; then
        log::error "Node.js is required to run requirements validation"
        return 1
    fi

    local validator="${APP_ROOT}/scripts/requirements/validate.js"
    if [ ! -f "$validator" ]; then
        log::error "Validator not found at $validator"
        return 1
    fi

    local args=(--scenario "$scenario_name")
    if [ "$quiet" = true ]; then
        args+=(--quiet)
    fi

    (cd "$scenario_dir" && node "$validator" "${args[@]}")
}

scenario::requirements::sync() {
    local scenario_name=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --help|-h)
                echo "Usage: vrooli scenario requirements sync <name>"
                return 0
                ;;
            *)
                if [ -z "$scenario_name" ]; then
                    scenario_name="$1"
                    shift
                else
                    log::error "Unexpected argument: $1"
                    return 1
                fi
                ;;
        esac
    done

    local scenario_dir
    scenario_dir=$(scenario::requirements::_ensure_scenario_dir "$scenario_name") || return 1

    if ! command -v node >/dev/null 2>&1; then
        log::error "Node.js is required to run requirements sync"
        return 1
    fi

    local reporter="${APP_ROOT}/scripts/requirements/report.js"
    if [ ! -f "$reporter" ]; then
        log::error "Requirements reporter not found at $reporter"
        return 1
    fi

    (cd "$scenario_dir" && node "$reporter" --scenario "$scenario_name" --mode sync)
}

scenario::requirements::phase_inspect() {
    local scenario_name=""
    local phase_name=""
    local output_path=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --phase)
                phase_name="$2"
                shift 2
                ;;
            --output)
                output_path="$2"
                shift 2
                ;;
            --help|-h)
                cat <<'__REQ_PHASE_HELP__'
Usage: vrooli scenario requirements phase <name> --phase <phase> [--output path]

Runs `scripts/requirements/report.js --mode phase-inspect` to list the validations expected for a
specific phase. Handy when wiring automation assets to requirement entries.
__REQ_PHASE_HELP__
                return 0
                ;;
            *)
                if [ -z "$scenario_name" ]; then
                    scenario_name="$1"
                    shift
                else
                    log::error "Unexpected argument: $1"
                    return 1
                fi
                ;;
        esac
    done

    if [ -z "$phase_name" ]; then
        log::error "--phase is required"
        return 1
    fi

    local scenario_dir
    scenario_dir=$(scenario::requirements::_ensure_scenario_dir "$scenario_name") || return 1

    if ! command -v node >/dev/null 2>&1; then
        log::error "Node.js is required to inspect phase validations"
        return 1
    fi

    local reporter="${APP_ROOT}/scripts/requirements/report.js"
    if [ ! -f "$reporter" ]; then
        log::error "Requirements reporter not found at $reporter"
        return 1
    fi

    local cmd=(node "$reporter" --scenario "$scenario_name" --mode phase-inspect --phase "$phase_name")
    if [ -n "$output_path" ]; then
        mkdir -p "$(dirname "$output_path")"
        (cd "$scenario_dir" && "${cmd[@]}") > "$output_path"
        log::info "Phase inspect output written to $output_path"
    else
        (cd "$scenario_dir" && "${cmd[@]}")
    fi
}

scenario::requirements::init() {
    local scenario_name=""
    local force=false
    local template_name="react-vite"
    local owner_contact=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force=true
                shift
                ;;
            --template)
                template_name="$2"
                shift 2
                ;;
            --owner)
                owner_contact="$2"
                shift 2
                ;;
            --help|-h)
                cat <<'__REQ_INIT_HELP__'
Usage: vrooli scenario requirements init <name> [--force] [--template modular] [--owner contact@example.com]

Copies the shared requirements/ template into the scenario directory and
updates metadata placeholders (scenario slug, generated timestamp, owner).
__REQ_INIT_HELP__
                return 0
                ;;
            *)
                if [ -z "$scenario_name" ]; then
                    scenario_name="$1"
                    shift
                else
                    log::error "Unexpected argument: $1"
                    return 1
                fi
                ;;
        esac
    done

    if [ -z "$scenario_name" ]; then
        log::error "Scenario name is required for requirements init"
        return 1
    fi

    local scenario_dir="${APP_ROOT}/scenarios/${scenario_name}"
    if [ ! -d "$scenario_dir" ]; then
        log::error "Scenario directory not found: $scenario_dir"
        return 1
    fi

    local template_dir="${APP_ROOT}/scripts/scenarios/templates/react-vite/requirements"
    if [ ! -d "$template_dir" ]; then
        log::error "Cannot locate requirements template at $template_dir"
        log::info "Ensure the repository includes scripts/scenarios/templates/react-vite/requirements"
        return 1
    fi

    local target_dir="${scenario_dir}/requirements"
    if [ -e "$target_dir" ] && [ "$force" = false ]; then
        log::error "${target_dir} already exists. Re-run with --force to overwrite."
        return 1
    fi

    rm -rf "$target_dir"
    mkdir -p "$target_dir"
    (cd "$template_dir" && tar -cf - .) | (cd "$target_dir" && tar -xf -)

    local generated_date
    generated_date=$(date -I)
    if [ -z "$owner_contact" ]; then
        owner_contact="engineering@${scenario_name}.local"
    fi

    local replace_script
    replace_script=$(cat <<'PY'
import os
from pathlib import Path

scenario = os.environ["REQ_SCENARIO_NAME"]
generated = os.environ["REQ_GENERATED_DATE"]
contact = os.environ["REQ_CONTACT"]
file_path = Path(os.environ["REQ_FILE"])
text = file_path.read_text()
text = text.replace("__SCENARIO_NAME__", scenario)
text = text.replace("__GENERATED_DATE__", generated)
text = text.replace("__CONTACT__", contact)
file_path.write_text(text)
PY
)

    while IFS= read -r -d '' file; do
        REQ_FILE="$file" \
        REQ_SCENARIO_NAME="$scenario_name" \
        REQ_GENERATED_DATE="$generated_date" \
        REQ_CONTACT="$owner_contact" \
        python3 - <<PY
$replace_script
PY
    done < <(find "$target_dir" -type f -name '*.yaml' -print0)

    log::success "Initialized requirements/ registry for ${scenario_name} using ${template_name} template"
    log::info "Edit files under ${target_dir} to map PRD items to technical requirements."
    log::info "Next steps:"$'\n'
    log::info "  • Read docs/testing/guides/requirement-tracking-quick-start.md for how to structure requirements and tag tests"
    log::info "  • Review docs/testing/architecture/REQUIREMENT_FLOW.md for the full PRD → requirement → validation flow"
    log::info "  • After tagging tests with [REQ:ID], run 'node scripts/requirements/report.js --scenario ${scenario_name} --mode sync' to propagate results"
}

scenario::requirements::quick_check() {
    local scenario_name="$1"
    local scenario_dir="${APP_ROOT}/scenarios/${scenario_name}"
    local reporter="${APP_ROOT}/scripts/requirements/report.js"
    local validator="${APP_ROOT}/scripts/requirements/validate.js"

    if [ ! -d "$scenario_dir" ]; then
        printf '%s\n' '{"status":"not_found","message":"Scenario directory not found"}'
        return 0
    fi

    if [ ! -d "${scenario_dir}/requirements" ]; then
        printf '%s\n' '{"status":"missing","message":"Requirements/ registry not found","recommendation":"Run `vrooli scenario requirements init <name>` to scaffold requirements/."}'
        return 0
    fi

    if ! command -v node >/dev/null 2>&1; then
        printf '%s\n' '{"status":"unavailable","message":"Node.js not available","recommendation":"Install Node.js to evaluate requirement coverage."}'
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        printf '%s\n' '{"status":"unavailable","message":"jq not available","recommendation":"Install jq to evaluate requirement coverage."}'
        return 0
    fi

    if [ ! -f "$reporter" ]; then
        printf '%s\n' '{"status":"unavailable","message":"requirements/report.js not found"}'
        return 0
    fi

    local report_json
    if ! report_json=$(cd "$scenario_dir" && node "$reporter" --scenario "$scenario_name" --format json 2>/dev/null); then
        printf '%s\n' '{"status":"error","message":"Failed to generate requirements summary"}'
        return 0
    fi

    local metrics
    metrics=$(echo "$report_json" | jq '{
        total: (.summary.total // 0),
        complete: (.summary.byStatus.complete // 0),
        in_progress: (.summary.byStatus.in_progress // 0),
        pending: (.summary.byStatus.pending // 0),
        criticality_gap: (.summary.criticalityGap // 0)
    } | .coverage_ratio = (if .total == 0 then 0 else (.complete / (.total)) end)
      | .status = (if .total == 0 then "empty" elif .criticality_gap > 0 then "critical_gap" elif .complete == .total then "complete" else "incomplete" end)')

    local drift_json="null"
    if [ -d "${scenario_dir}/requirements" ] && command -v node >/dev/null 2>&1; then
        local drift_raw
        if drift_raw=$(cd "$scenario_dir" && node "$APP_ROOT/scripts/requirements/drift-check.js" --scenario "$scenario_name" 2>/dev/null); then
            if echo "$drift_raw" | jq -e . >/dev/null 2>&1; then
                drift_json="$drift_raw"
            fi
        fi
    fi

    local status
    status=$(echo "$metrics" | jq -r '.status')

    local message=""
    local recommendation=""
    case "$status" in
        complete)
            message="All documented requirements satisfied"
            ;;
        critical_gap)
            message="Critical P0/P1 requirements missing validation"
            recommendation="Run 'vrooli scenario requirements ${scenario_name} --fail-on-critical-gap' to inspect gaps."
            ;;
        incomplete)
            message="Requirements exist but implementation coverage is incomplete"
            recommendation="Expand validations or BA workflows for remaining requirements."
            ;;
        empty)
            message="Requirement registry found but contains no entries"
            recommendation="Document PRD-derived requirements inside requirements/."
            ;;
        *)
            message="Requirement status unavailable"
            ;;
    esac

    local schema_status="unavailable"
    local schema_message=""
    if [ -f "$validator" ]; then
        local validator_output=""
        if validator_output=$(cd "$scenario_dir" && node "$validator" --scenario "$scenario_name" --quiet 2>&1); then
            schema_status="valid"
        else
            schema_status="invalid"
            schema_message=$(printf '%s' "$validator_output" | tail -n 12)
        fi
    else
        schema_message="Schema validator not installed"
    fi

    jq -n \
        --arg status "$status" \
        --arg message "$message" \
        --arg recommendation "$recommendation" \
        --arg schema_status "$schema_status" \
        --arg schema_message "$schema_message" \
        --argjson summary "$(echo "$metrics" | jq '{total,complete,in_progress,pending,criticality_gap,coverage_ratio}')" \
        --argjson drift "$drift_json" \
        '{status:$status,message:$message,recommendation:$recommendation,summary:$summary,schema:{status:$schema_status,message:$schema_message},drift:$drift}'
}

scenario::requirements::display_summary() {
    local summary_json="$1"
    local scenario_name="${2:-}"

    if ! command -v jq >/dev/null 2>&1; then
        log::warning "Requirements: ⚠️ jq not available; skipped requirement summary"
        return 0
    fi

    if [ -z "$summary_json" ] || ! echo "$summary_json" | jq -e . >/dev/null 2>&1; then
        log::warning "Requirements: Unable to parse requirement summary"
        return 0
    fi

    local status
    status=$(echo "$summary_json" | jq -r '.status // "unknown"')

    if [[ "$status" == "missing" ]]; then
        local requirements_doc_path="${APP_ROOT}/docs/testing/guides/requirement-tracking.md"
        if [[ -f "$requirements_doc_path" ]]; then
            local already_listed="false"
            if [[ -n "${SCENARIO_STATUS_EXTRA_DOC_LINKS:-}" ]]; then
                if printf '%s\n' "${SCENARIO_STATUS_EXTRA_DOC_LINKS}" | grep -Fxq "$requirements_doc_path"; then
                    already_listed="true"
                fi
            fi

            if [[ "$already_listed" == "false" ]]; then
                printf -v SCENARIO_STATUS_EXTRA_DOC_LINKS "%s%s\n" "${SCENARIO_STATUS_EXTRA_DOC_LINKS:-}" "$requirements_doc_path"
            fi
        fi
    fi

    local coverage
    coverage=$(echo "$summary_json" | jq -r '.summary.coverage_ratio // 0')

    local total
    total=$(echo "$summary_json" | jq -r '.summary.total // 0')

    local complete
    complete=$(echo "$summary_json" | jq -r '.summary.complete // 0')

    local critical_gap
    critical_gap=$(echo "$summary_json" | jq -r '.summary.criticality_gap // 0')

    local message_text
    message_text=$(echo "$summary_json" | jq -r '.message // empty')

    local recommendation
    recommendation=$(echo "$summary_json" | jq -r '.recommendation // empty')

    local coverage_percent
    coverage_percent=$(awk -v c="$coverage" 'BEGIN { printf "%.0f", (c * 100) }')

    case "$status" in
        complete)
            log::success "Requirements: ✅ ${total} documented requirements satisfied"
            ;;
        critical_gap)
            log::error "Requirements: 🔴 ${critical_gap} critical requirement(s) missing validation"
            [ -n "$message_text" ] && log::info "  ↳ ${message_text}"
            ;;
        incomplete)
            printf 'Requirements: 🟡 %s%% complete (%s/%s implemented)\n' "$coverage_percent" "$complete" "$total"
            [ -n "$message_text" ] && log::info "  ↳ ${message_text}"
            ;;
        empty)
            log::warning "Requirements: ⚠️ Registry exists but has no entries"
            ;;
        missing)
            log::warning "Requirements: ⚠️ No requirements registry found"
            ;;
        unavailable)
            if [ -n "$message_text" ]; then
                log::warning "Requirements: ⚠️ ${message_text}"
            else
                log::warning "Requirements: ⚠️ ${scenario_name:-Scenario} requirements could not be evaluated"
            fi
            ;;
        not_found)
            log::warning "Requirements: ⚠️ Scenario directory not found"
            ;;
        *)
            log::warning "Requirements: ⚠️ Status ${status}"
            ;;
    esac

    if [ "$status" != "complete" ] && [ "$status" != "missing" ] && [ "$status" != "not_found" ] && [ "$status" != "empty" ]; then
        echo "  • Coverage: ${coverage_percent}% (${complete}/${total} complete)"
        [ "$critical_gap" != "0" ] && echo "  • Critical gap (open P0/P1): ${critical_gap}"
    fi

    if [ -n "$recommendation" ]; then
        printf -v SCENARIO_STATUS_EXTRA_RECOMMENDATIONS "%s%s\n" "${SCENARIO_STATUS_EXTRA_RECOMMENDATIONS}" "$recommendation"
    elif [ "$status" = "complete" ]; then
        log::info "  ↳ View details with: vrooli scenario requirements ${scenario_name}"
    fi

    local schema_status
    schema_status=$(echo "$summary_json" | jq -r '.schema.status // "unavailable"')
    local schema_message
    schema_message=$(echo "$summary_json" | jq -r '.schema.message // empty')

    case "$schema_status" in
        valid)
            :
            ;;
        invalid)
            log::warning "Requirements: ⚠️  Schema validation failed"
            if [ -n "$schema_message" ]; then
                echo "  • Validator output:"
                printf '%s\n' "$schema_message" | sed 's/\r$//' | sed 's/^/      /'
            fi
            echo "  • Run: node scripts/requirements/validate.js --scenario ${scenario_name}"
            ;;
        unavailable)
            if [ -n "$schema_message" ]; then
                log::info "Requirements: ℹ️  ${schema_message}"
            fi
            ;;
    esac

    local drift_section
    drift_section=$(echo "$summary_json" | jq -c '.drift // null' 2>/dev/null || echo 'null')
    if [ "$drift_section" != "null" ]; then
        local drift_status
        drift_status=$(echo "$drift_section" | jq -r '.status // empty')
        local drift_issues
        drift_issues=$(echo "$drift_section" | jq -r '.issue_count // 0')
        case "$drift_status" in
            missing_snapshot)
                log::warning "Requirements: ⚠️  Drift snapshot missing; run test/run-tests.sh to regenerate requirements metadata"
                ;;
            drift_detected)
                log::error "Requirements: 🔴 Drift detected (${drift_issues} issue(s))"
                ;;
        esac

        local file_drift
        file_drift=$(echo "$drift_section" | jq -r '.file_drift.has_drift // false')
        if [ "$file_drift" = "true" ]; then
            local mismatch_count new_count missing_count
            mismatch_count=$(echo "$drift_section" | jq -r '.file_drift.mismatched | length')
            new_count=$(echo "$drift_section" | jq -r '.file_drift.new_files | length')
            missing_count=$(echo "$drift_section" | jq -r '.file_drift.missing_from_disk | length')
            echo "  • Requirement files changed outside auto-sync (mismatched: ${mismatch_count}, new: ${new_count}, missing: ${missing_count})"
        fi

        local artifact_stale
        artifact_stale=$(echo "$drift_section" | jq -r '.artifact_stale // false')
        if [ "$artifact_stale" = "true" ]; then
            local latest_artifact
            latest_artifact=$(echo "$drift_section" | jq -r '.latest_artifact_at // empty')
            echo "  • Newer coverage artifacts detected (${latest_artifact}); rerun auto-sync"
        fi

        local prd_has_drift
        prd_has_drift=$(echo "$drift_section" | jq -r '.prd.has_drift // false')
        if [ "$prd_has_drift" = "true" ]; then
            local prd_mismatches
            prd_mismatches=$(echo "$drift_section" | jq -r '.prd.mismatches | length')
            local prd_missing
            prd_missing=$(echo "$drift_section" | jq -r '.prd.missing_in_snapshot | length')
            echo "  • PRD mismatch (${prd_mismatches} status differences, ${prd_missing} missing targets)"
        fi

        local manual_info
        manual_info=$(echo "$drift_section" | jq -c '.manual // null' 2>/dev/null || echo 'null')
        if [ "$manual_info" != "null" ]; then
            local manual_total manual_issue_count
            manual_total=$(echo "$manual_info" | jq -r '.total // 0')
            manual_issue_count=$(echo "$manual_info" | jq -r '.issue_count // 0')
            if [ "$manual_total" != "0" ]; then
                echo "  • Manual validations: ${manual_total} tracked"
            fi
            local manual_manifest_missing
            manual_manifest_missing=$(echo "$manual_info" | jq -r '.manifest_missing // false')
            if [ "$manual_manifest_missing" = "true" ]; then
                echo "    ↳ Manifest missing; run manual validations via 'vrooli scenario requirements manual-log'"
            fi
            local manual_expired_count
            manual_expired_count=$(echo "$manual_info" | jq -r '.expired | length')
            if [ "$manual_expired_count" != "0" ]; then
                local expired_list
                expired_list=$(echo "$manual_info" | jq -r '.expired | join(", ")')
                echo "    ↳ Expired entries (${manual_expired_count}): ${expired_list}"
            fi
            local manual_missing_meta
            manual_missing_meta=$(echo "$manual_info" | jq -r '.missing_metadata | length')
            if [ "$manual_missing_meta" != "0" ]; then
                local missing_list
                missing_list=$(echo "$manual_info" | jq -r '.missing_metadata | join(", ")')
                echo "    ↳ Manual validations missing metadata: ${missing_list}"
            fi
            local manual_unsynced
            manual_unsynced=$(echo "$manual_info" | jq -r '.unsynced | length')
            if [ "$manual_unsynced" != "0" ]; then
                local unsynced_list
                unsynced_list=$(echo "$manual_info" | jq -r '.unsynced | join(", ")')
                echo "    ↳ Manual logs newer than requirements: ${unsynced_list} (rerun tests + sync)"
            fi
            local manual_missing_entries
            manual_missing_entries=$(echo "$manual_info" | jq -r '.manifest_missing_entries | length')
            if [ "$manual_missing_entries" != "0" ]; then
                local missing_entries_list
                missing_entries_list=$(echo "$manual_info" | jq -r '.manifest_missing_entries | join(", ")')
                echo "    ↳ Manifest missing entries for: ${missing_entries_list}"
            fi
            if [ "$manual_issue_count" != "0" ]; then
                printf -v SCENARIO_STATUS_EXTRA_RECOMMENDATIONS "%sRevalidate manual requirements or replace them with automated workflows (use 'vrooli scenario requirements manual-log').\n" "${SCENARIO_STATUS_EXTRA_RECOMMENDATIONS}"
            fi
        fi

        local drift_recommendation
        drift_recommendation=$(echo "$drift_section" | jq -r '.recommendation // empty')
        if [ -n "$drift_recommendation" ]; then
            printf -v SCENARIO_STATUS_EXTRA_RECOMMENDATIONS "%s%s\n" "${SCENARIO_STATUS_EXTRA_RECOMMENDATIONS}" "$drift_recommendation"
        fi
    fi
}
