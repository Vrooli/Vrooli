#!/usr/bin/env bash

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

    if [ ! -f "${scenario_dir}/docs/requirements.yaml" ] && [ ! -d "${scenario_dir}/requirements" ]; then
        log::error "Scenario ${scenario_name} does not define docs/requirements.yaml or requirements/"
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
Usage: vrooli scenario requirements <name> [options]

Options:
  --format <json|markdown|trace>   Output format (default: markdown)
  --include-pending               Include pending requirements in the report
  --output <path>                 Write report to file instead of stdout
  --fail-on-critical-gap          Exit with non-zero status if critical P0/P1 requirements are incomplete
__REQ_HELP__
}

scenario::requirements::quick_check() {
    local scenario_name="$1"
    local scenario_dir="${APP_ROOT}/scenarios/${scenario_name}"
    local reporter="${APP_ROOT}/scripts/requirements/report.js"

    if [ ! -d "$scenario_dir" ]; then
        printf '%s\n' '{"status":"not_found","message":"Scenario directory not found"}'
        return 0
    fi

    if [ ! -f "${scenario_dir}/docs/requirements.yaml" ] && [ ! -d "${scenario_dir}/requirements" ]; then
        printf '%s\n' '{"status":"missing","message":"No requirements registry defined","recommendation":"Add docs/requirements.yaml or requirements/ to track PRD coverage."}'
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

    jq -n \
        --arg status "$status" \
        --arg message "$message" \
        --arg recommendation "$recommendation" \
        --argjson summary "$(echo "$metrics" | jq '{total,complete,in_progress,pending,criticality_gap,coverage_ratio}')" \
        '{status:$status,message:$message,recommendation:$recommendation,summary:$summary}'
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

    if [ -n "$recommendation" ]; then
        printf -v SCENARIO_STATUS_EXTRA_RECOMMENDATIONS "%s%s\n" "${SCENARIO_STATUS_EXTRA_RECOMMENDATIONS}" "$recommendation"
    elif [ "$status" = "complete" ]; then
        log::info "  ↳ View details with: vrooli scenario requirements ${scenario_name}"
    fi
}
