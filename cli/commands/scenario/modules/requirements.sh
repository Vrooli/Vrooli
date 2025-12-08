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
        json|markdown|trace|summary)
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

    if ! command -v jq >/dev/null 2>&1; then
        log::error "jq is required to generate requirements reports"
        return 1
    fi

    local test_genie_cli
    test_genie_cli=$(scenario::requirements::_test_genie_cli) || return 1

    local base_args=(requirements report --dir "$scenario_dir" --format json)
    if [ "$include_pending" = true ]; then
        base_args+=(--include-pending)
    fi

    local temp_json
    temp_json=$(mktemp)
    (cd "$scenario_dir" && "$test_genie_cli" "${base_args[@]}" --output "$temp_json")

    local critical_gap
    critical_gap=$(jq -r '.summary.critical_gap // 0' "$temp_json" 2>/dev/null || echo "0")

    local final_status=0
    if [ "$format" = "json" ]; then
        if [ -n "$output_path" ]; then
            mkdir -p "$(dirname "$output_path")"
            mv "$temp_json" "$output_path"
        else
            cat "$temp_json"
            rm -f "$temp_json"
        fi
    else
        if [ -n "$output_path" ]; then
            mkdir -p "$(dirname "$output_path")"
            (cd "$scenario_dir" && "$test_genie_cli" requirements report --dir "$scenario_dir" --format "$format" ${include_pending:+--include-pending} --output "$output_path")
            rm -f "$temp_json"
        else
            (cd "$scenario_dir" && "$test_genie_cli" requirements report --dir "$scenario_dir" --format "$format" ${include_pending:+--include-pending})
            rm -f "$temp_json"
        fi
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

  validate <name> [--quiet]         Run schema + semantic validation (test-genie requirements validate)
  sync <name>                       Update requirement status fields via native syncer
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

    local scenario_dir
    scenario_dir=$(scenario::requirements::_ensure_scenario_dir "$scenario_name") || return 1

    local test_genie_cli
    test_genie_cli=$(scenario::requirements::_test_genie_cli) || return 1

    local cmd=("$test_genie_cli" requirements manual-log --dir "$scenario_dir" --scenario "$scenario_name" --requirement "$requirement_id" --status "$status" --validated-by "$validated_by")
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

    local test_genie_cli
    test_genie_cli=$(scenario::requirements::_test_genie_cli) || return 1

    local cmd=("$test_genie_cli" requirements lint-prd --dir "$scenario_dir" --scenario "$scenario_name")
    if [ "$json_output" = true ]; then
        cmd+=(--json)
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

scenario::requirements::_test_genie_cli() {
    local cli_path="${VROOLI_TEST_GENIE_CLI:-${APP_ROOT}/scenarios/test-genie/cli/test-genie}"
    if [[ -x "$cli_path" ]]; then
        echo "$cli_path"
        return 0
    fi

    log::error "test-genie CLI not found at $cli_path"
    log::info "Build test-genie with: cd ${APP_ROOT}/scenarios/test-genie && make build"
    return 1
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

    local test_genie_cli
    test_genie_cli=$(scenario::requirements::_test_genie_cli) || return 1

    local args=(requirements validate --dir "$scenario_dir" --scenario "$scenario_name")
    if [ "$quiet" = true ]; then
        args+=(--quiet)
    fi

    (cd "$scenario_dir" && "$test_genie_cli" "${args[@]}")
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

    local test_genie_cli
    test_genie_cli=$(scenario::requirements::_test_genie_cli) || return 1

    (cd "$scenario_dir" && "$test_genie_cli" requirements sync --dir "$scenario_dir" --scenario "$scenario_name")
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

Runs `test-genie requirements phase --phase <phase>` to list the validations expected for a
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

    local test_genie_cli
    test_genie_cli=$(scenario::requirements::_test_genie_cli) || return 1

    local cmd=("$test_genie_cli" requirements phase --dir "$scenario_dir" --scenario "$scenario_name" --phase "$phase_name")
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

    local test_genie_cli
    test_genie_cli=$(scenario::requirements::_test_genie_cli) || return 1

    local args=(requirements init --dir "$scenario_dir" --scenario "$scenario_name" --template "$template_name")
    if [ "$force" = true ]; then
        args+=(--force)
    fi
    if [ -n "$owner_contact" ]; then
        args+=(--owner "$owner_contact")
    fi

    (cd "$scenario_dir" && "$test_genie_cli" "${args[@]}")
}

scenario::requirements::quick_check() {
    local scenario_name="$1"
    local scenario_dir="${APP_ROOT}/scenarios/${scenario_name}"

    if [ ! -d "$scenario_dir" ]; then
        printf '%s\n' '{"status":"not_found","message":"Scenario directory not found"}'
        return 0
    fi

    if [ ! -d "${scenario_dir}/requirements" ]; then
        printf '%s\n' '{"status":"missing","message":"Requirements/ registry not found","recommendation":"Run `vrooli scenario requirements init <name>` to scaffold requirements/."}'
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        printf '%s\n' '{"status":"unavailable","message":"jq not available","recommendation":"Install jq to evaluate requirement coverage."}'
        return 0
    fi

    local test_genie_cli
    if ! test_genie_cli=$(scenario::requirements::_test_genie_cli); then
        printf '%s\n' '{"status":"unavailable","message":"test-genie CLI not available","recommendation":"Build test-genie (cd scenarios/test-genie && make build)"}'
        return 0
    fi

    local report_json
    if ! report_json=$(cd "$scenario_dir" && "$test_genie_cli" requirements report --dir "$scenario_dir" --scenario "$scenario_name" --format json 2>/dev/null); then
        printf '%s\n' '{"status":"error","message":"Failed to generate requirements summary"}'
        return 0
    fi

    local metrics
    metrics=$(echo "$report_json" | jq '{
        total: (.summary.total // 0),
        complete: (.summary.complete // 0),
        in_progress: (.summary.in_progress // 0),
        pending: (.summary.pending // 0),
        criticality_gap: (.summary.critical_gap // 0)
    } | .coverage_ratio = (if .total == 0 then 0 else (.complete / (.total)) end)
      | .status = (if .total == 0 then "empty" elif .criticality_gap > 0 then "critical_gap" elif .complete == .total then "complete" else "incomplete" end)')

    local drift_json="null"
    if [ -d "${scenario_dir}/requirements" ]; then
        local drift_raw
        if drift_raw=$(cd "$scenario_dir" && "$test_genie_cli" requirements drift --dir "$scenario_dir" --scenario "$scenario_name" --json 2>/dev/null); then
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
    local validator_output=""
    if validator_output=$(cd "$scenario_dir" && "$test_genie_cli" requirements validate --dir "$scenario_dir" --scenario "$scenario_name" --quiet 2>&1); then
        schema_status="valid"
    else
        schema_status="invalid"
        schema_message=$(printf '%s' "$validator_output" | tail -n 12)
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
        local requirements_doc_path="${APP_ROOT}/scenarios/test-genie/docs/phases/business/requirements-sync.md"
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
            echo "  • Run: vrooli scenario requirements validate ${scenario_name}"
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
        case "$drift_status" in
            missing_snapshot)
                log::warning "Requirements: ⚠️  Snapshot missing; run the scenario test suite to refresh requirements metadata"
                ;;
            drift_detected)
                log::error "Requirements: 🔴 Drift detected"
                ;;
        esac

        local drift_messages
        drift_messages=$(echo "$drift_section" | jq -r '.messages[]?' 2>/dev/null || true)
        if [ -n "$drift_messages" ]; then
            while IFS= read -r msg; do
                [ -n "$msg" ] && echo "  • ${msg}"
            done <<< "$drift_messages"
        fi
    fi
}
