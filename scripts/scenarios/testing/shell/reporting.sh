#!/usr/bin/env bash
# Reporting helpers for scenario test runners
set -euo pipefail

# Generate a JUnit XML report from execution results
# Arguments:
#   $1 - suite name
#   $2 - output path
#   $3 - total duration
#   $4 - name of associative array with statuses (item -> status)
#   $5 - name of associative array with durations (item -> seconds)
#   $6 - name of associative array with log paths (item -> path)
#   $7 - name of indexed array with execution order (items in logical order)
#   $8 - name of associative array with display names (item -> junit testcase name)
#   $9 - name of associative array with item types (item -> phase|test)
testing::reporting::generate_junit() {
    local suite_name="$1"
    local output_path="$2"
    local total_duration="$3"
    local -n _statuses_ref="$4"
    local -n _durations_ref="$5"
    local -n _logs_ref="$6"
    local -n _order_ref="$7"
    local -n _display_ref="$8"
    local -n _types_ref="$9"

    local timestamp
    timestamp=$(date -Iseconds)

    local tests=0
    local failures=0
    local skipped=0

    for item in "${_order_ref[@]}"; do
        ((tests++)) || true
        case "${_statuses_ref[$item]}" in
            failed|timed_out|not_executable|missing)
                ((failures++)) || true
                ;;
            skipped)
                ((skipped++)) || true
                ;;
        esac
    done

    {
        printf '<?xml version="1.0" encoding="UTF-8"?>\n'
        printf '<testsuite name="%s" tests="%d" failures="%d" skipped="%d" time="%s" timestamp="%s">\n' \
            "$suite_name" "$tests" "$failures" "$skipped" "$total_duration" "$timestamp"

        for item in "${_order_ref[@]}"; do
            local testcase_name="${_display_ref[$item]}"
            local duration="${_durations_ref[$item]:-0}"
            local status="${_statuses_ref[$item]}"
            local log_path="${_logs_ref[$item]:-}"

            printf '    <testcase classname="%s" name="%s" time="%s">\n' \
                "${suite_name}" "$testcase_name" "$duration"

            case "$status" in
                skipped)
                    printf '        <skipped message="%s"/>\n' "${_types_ref[$item]} ${testcase_name} skipped"
                    ;;
                failed|timed_out|not_executable|missing)
                    local failure_message="${_types_ref[$item]} ${testcase_name}"
                    printf '        <failure message="%s">' "$failure_message"
                    if [ -n "$log_path" ] && [ -f "$log_path" ]; then
                        printf '<![CDATA['
                        sed 's/]]>/]]]]><![CDATA[>/' "$log_path"
                        printf ']]>'
                    fi
                    printf '</failure>\n'
                    ;;
            esac

            printf '    </testcase>\n'
        done

        printf '</testsuite>\n'
    } > "$output_path"

    return 0
}

export -f testing::reporting::generate_junit
