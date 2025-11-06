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

    if [ ! -f "${scenario_dir}/docs/requirements.json" ] && [ ! -d "${scenario_dir}/requirements" ]; then
        log::error "Scenario ${scenario_name} does not define docs/requirements.json or requirements/"
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
    if [ ! -f "${scenario_dir}/docs/requirements.json" ] && [ ! -d "${scenario_dir}/requirements" ]; then
        log::error "Scenario ${scenario_name} does not define docs/requirements.json or requirements/"
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
    local template_name="modular"
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

    local template_dir="${APP_ROOT}/scripts/scenarios/templates/requirements/${template_name}"
    if [ ! -d "$template_dir" ]; then
        log::error "Unknown requirements template '${template_name}'."
        log::info "Available templates: $(ls "${APP_ROOT}/scripts/scenarios/templates/requirements" 2>/dev/null | tr '\n' ' ')"
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

    log::success "Initialized requirements/ registry for ${scenario_name} using '${template_name}' template"
    log::info "Edit files under ${target_dir} to map PRD items to technical requirements."
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

    if [ ! -f "${scenario_dir}/docs/requirements.json" ] && [ ! -d "${scenario_dir}/requirements" ]; then
        printf '%s\n' '{"status":"missing","message":"No requirements registry defined","recommendation":"Add docs/requirements.json or requirements/ to track PRD coverage."}'
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
            schema_message=$(printf '%s' "$validator_output" | tail -n 5 | tr '\n' ' ' | sed 's/  */ /g')
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
        '{status:$status,message:$message,recommendation:$recommendation,summary:$summary,schema:{status:$schema_status,message:$schema_message}}'
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
                echo "  • ${schema_message}"
            fi
            echo "  • Run: node scripts/requirements/validate.js --scenario ${scenario_name}"
            ;;
        unavailable)
            if [ -n "$schema_message" ]; then
                log::info "Requirements: ℹ️  ${schema_message}"
            fi
            ;;
    esac
}
