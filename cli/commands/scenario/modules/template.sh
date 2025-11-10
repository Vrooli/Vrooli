#!/usr/bin/env bash

SCENARIO_TEMPLATE_BASE_DIR="${SCENARIO_TEMPLATE_BASE_DIR:-${APP_ROOT}/scripts/scenarios/templates}"

scenario::template::dispatch() {
    if [[ $# -eq 0 ]]; then
        scenario::template::list
        return
    fi

    local action="$1"; shift || true
    case "$action" in
        list)
            scenario::template::list
            ;;
        show)
            scenario::template::show "$@"
            ;;
        help|--help|-h)
            scenario::template::help
            ;;
        *)
            log::error "Unknown template command: $action"
            scenario::template::help
            return 1
            ;;
    esac
}

scenario::template::help() {
    cat <<'SCENARIO_TEMPLATE_HELP'
Scenario Template Commands:
  vrooli scenario template list
      List available scenario templates.

  vrooli scenario template show <template>
      Show detailed information about a template (variables, hooks, docs).

  vrooli scenario generate <template> [options]
      Copy a template into scenarios/<id>/ with placeholder replacements.
      Options:
        --id <value>             Scenario slug (required)
        --display-name <value>   Human-friendly name (required)
        --description <value>    One sentence summary (required)
        --dest <path>            Custom destination (defaults to {{PROJECT_PATH}}/scenarios/<id>)
        --category <value>       Optional service category override
        --var KEY=VALUE          Provide additional placeholder overrides
        --force                  Overwrite destination if it already exists
        --dry-run                Show what would happen without writing files
        --run-hooks              Execute template post hooks (e.g., pnpm install)
SCENARIO_TEMPLATE_HELP
}

declare -A scenario_template_required_flags=()
declare -A scenario_template_optional_flags=()
declare -A scenario_template_optional_defaults=()
declare -A scenario_template_flag_map=()
declare -A scenario_template_values=()

declare -A scenario_template_user_overrides=()

scenario::template::list() {
    if [[ ! -d "$SCENARIO_TEMPLATE_BASE_DIR" ]]; then
        log::error "Template directory not found: $SCENARIO_TEMPLATE_BASE_DIR"
        return 1
    fi

    printf "%-18s %-28s %-s\n" "NAME" "DISPLAY NAME" "REQUIRED VARS"
    for template_dir in "$SCENARIO_TEMPLATE_BASE_DIR"/*; do
        [[ -d "$template_dir" ]] || continue
        local name display required manifest
        name="$(basename "$template_dir")"
        manifest="$template_dir/template.json"
        if [[ -f "$manifest" ]]; then
            display=$(jq -r '.displayName // .name // "(missing displayName)"' "$manifest")
            required=$(jq -r '.requiredVars // {} | to_entries | map(.key + (if (.value.flag // "") != "" then " (--" + .value.flag + ")" else "" end)) | join(", ") // ""' "$manifest")
            [[ -n "$required" ]] || required="-"
        else
            display="(template.json missing)"
            required="?"
        fi
        printf "%-18s %-28s %-s\n" "$name" "$display" "$required"
    done
    echo ""
    echo "Tip: Run 'vrooli scenario template show <name>' for details about a template."
}

scenario::template::show() {
    local template_name="$1"
    if [[ -z "$template_name" ]]; then
        log::error "Template name is required"
        return 1
    fi

    local template_dir="$SCENARIO_TEMPLATE_BASE_DIR/$template_name"
    if [[ ! -d "$template_dir" ]]; then
        log::error "Template not found: $template_name"
        return 1
    fi

    local manifest="$template_dir/template.json"
    if [[ ! -f "$manifest" ]]; then
        log::warning "template.json missing for $template_name"
        ls -la "$template_dir"
        return 0
    fi

    local display description stack
    display=$(jq -r '.displayName // .name // "(unnamed template)"' "$manifest")
    description=$(jq -r '.description // ""' "$manifest")
    stack=$(jq -r '.stack // [] | join(", ")' "$manifest")

    log::header "$display ($template_name)"
    if [[ -n "$description" ]]; then
        echo "$description"
    fi
    if [[ -n "$stack" ]]; then
        echo "Stack: $stack"
    fi

    echo ""
    scenario::template::print_var_table "$manifest" "requiredVars" "Required Variables"
    scenario::template::print_var_table "$manifest" "optionalVars" "Optional Variables"

    local hooks_count
    hooks_count=$(jq -r '.postHooks // [] | length' "$manifest")
    if [[ "$hooks_count" -gt 0 ]]; then
        printf "\nPost Hooks:\n"
        jq -r '.postHooks // [] | to_entries[] | "  - " + (.value.description // (.value.cmd))' "$manifest"
    fi

    local docs
    docs=$(jq -r '.docs // {} | to_entries[] | "  - " + (.key) + ": " + (.value)' "$manifest")
    if [[ -n "$docs" ]]; then
        printf "\nDocs:\n"
        printf "%s\n" "$docs"
    fi

    printf "\nFiles:\n"
    (cd "$template_dir" && find . -maxdepth 1 -mindepth 1 | sort)
    printf "\n"
    local required_flags
    required_flags=$(scenario::template::format_required_flags "$manifest")
    printf "Tip: Run 'vrooli scenario generate %s%s' to scaffold this template.\n" "$template_name" "$required_flags"
}

scenario::template::format_required_flags() {
    local manifest="$1"
    local parts
    parts=$(jq -r '.requiredVars // {} | to_entries | map(" --" + (.value.flag // "id") + " <" + (.key | ascii_downcase) + ">") | join("")' "$manifest")
    echo "${parts:- --id <slug>}"
}

scenario::template::print_var_table() {
    local manifest="$1"
    local key="$2"
    local title="$3"
    local rows
    rows=$(jq -r --arg key "$key" '.[$key] // {} | to_entries[] | [.key, (.value.flag // ""), (.value.description // ""), (.value.default // "")] | @tsv' "$manifest")
    if [[ -z "$rows" ]]; then
        return
    fi

    echo "$title:"
    while IFS=$'\t' read -r var flag desc def; do
        local flag_text=""
        [[ -n "$flag" ]] && flag_text=" (--${flag})"
        printf "  - %s%s: %s" "$var" "$flag_text" "$desc"
        if [[ -n "$def" ]]; then
            printf " [default: %s]" "$def"
        fi
        printf "\n"
    done <<< "$rows"
}

scenario::template::generate_usage() {
    cat <<'SCENARIO_TEMPLATE_GENERATE'
Usage: vrooli scenario generate <template> --id <slug> --display-name <name> --description <text> [options]
Options:
  --dest <path>            Destination directory (defaults to {{PROJECT_PATH}}/scenarios/<id>)
  --category <value>       Override scenario category in service.json
  --var KEY=VALUE          Provide additional placeholder overrides (repeatable)
  --force                  Overwrite destination if it already exists
  --dry-run                Show the planned actions without copying files
  --run-hooks              Execute template post hooks after generation
  --help                   Show this message
SCENARIO_TEMPLATE_GENERATE
}

scenario::template::generate() {
    if [[ $# -lt 1 ]]; then
        log::error "Template name is required"
        scenario::template::generate_usage
        return 1
    fi

    local template_name="$1"; shift
    local template_dir="$SCENARIO_TEMPLATE_BASE_DIR/$template_name"
    if [[ ! -d "$template_dir" ]]; then
        log::error "Template not found: $template_name"
        return 1
    fi

    local manifest="$template_dir/template.json"
    if [[ ! -f "$manifest" ]]; then
        log::error "template.json missing for $template_name"
        return 1
    fi

    scenario_template_required_flags=()
    scenario_template_optional_flags=()
    scenario_template_optional_defaults=()
    scenario_template_flag_map=()
    scenario_template_values=()
    scenario_template_user_overrides=()

    local line
    while IFS=$'\t' read -r var flag _; do
        [[ -n "$var" ]] || continue
        scenario_template_required_flags["$var"]="$flag"
        [[ -n "$flag" ]] && scenario_template_flag_map["$flag"]="$var"
    done < <(jq -r '.requiredVars // {} | to_entries[] | [.key, (.value.flag // ""), (.value.description // "")] | @tsv' "$manifest")

    while IFS=$'\t' read -r var flag def _; do
        [[ -n "$var" ]] || continue
        scenario_template_optional_flags["$var"]="$flag"
        scenario_template_optional_defaults["$var"]="$def"
        [[ -n "$flag" ]] && scenario_template_flag_map["$flag"]="$var"
    done < <(jq -r '.optionalVars // {} | to_entries[] | [.key, (.value.flag // ""), (.value.default // ""), (.value.description // "")] | @tsv' "$manifest")

    local dest_path=""
    local force=false
    local dry_run=false
    local run_hooks=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dest)
                dest_path="$2"
                shift 2
                ;;
            --dest=*)
                dest_path="${1#*=}"
                shift
                ;;
            --force)
                force=true
                shift
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            --run-hooks)
                run_hooks=true
                shift
                ;;
            --var)
                if [[ -z "${2:-}" ]]; then
                    log::error "--var requires KEY=VALUE"
                    return 1
                fi
                if ! scenario::template::parse_var_arg "${2}" scenario_template_user_overrides; then
                    return 1
                fi
                shift 2
                ;;
            --var=*)
                if ! scenario::template::parse_var_arg "${1#*=}" scenario_template_user_overrides; then
                    return 1
                fi
                shift
                ;;
            --help|-h)
                scenario::template::generate_usage
                return 0
                ;;
            --*)
                local flag_name flag_value
                if [[ "$1" == *=* ]]; then
                    flag_name="${1%%=*}"
                    flag_value="${1#*=}"
                    flag_name="${flag_name#--}"
                else
                    flag_name="${1#--}"
                    flag_value="${2:-}"
                    shift
                fi
                shift
                if [[ -z "$flag_value" ]]; then
                    log::error "--${flag_name} requires a value"
                    return 1
                fi
                if [[ -n "${scenario_template_flag_map[$flag_name]:-}" ]]; then
                    scenario_template_values["${scenario_template_flag_map[$flag_name]}"]="$flag_value"
                else
                    log::warning "Unknown flag --${flag_name}; use --var KEY=VALUE for arbitrary placeholders"
                fi
                ;;
            *)
                log::error "Unexpected argument: $1"
                scenario::template::generate_usage
                return 1
                ;;
        esac
    done

    local k
    for k in "${!scenario_template_user_overrides[@]}"; do
        if [[ -z "${scenario_template_required_flags[$k]:-}" && -z "${scenario_template_optional_flags[$k]:-}" ]]; then
            log::warning "Template variable '$k' is not defined in manifest; ensure the placeholder exists before relying on it."
        fi
        scenario_template_values["$k"]="${scenario_template_user_overrides[$k]}"
    done

    if [[ -z "${scenario_template_values[SCENARIO_ID]:-}" ]]; then
        log::error "Missing required value: --id / SCENARIO_ID"
        return 1
    fi

    local missing=()
    for k in "${!scenario_template_required_flags[@]}"; do
        if [[ -z "${scenario_template_values[$k]:-}" ]]; then
            missing+=("$k")
        fi
    done
    if [[ ${#missing[@]} -gt 0 ]]; then
        log::error "Missing required values: ${missing[*]}"
        scenario::template::generate_usage
        return 1
    fi

    local current_date="$(date -u +%Y-%m-%d)"
    local random_token="$(scenario::template::random_token)"

    for k in "${!scenario_template_optional_defaults[@]}"; do
        if [[ -z "${scenario_template_values[$k]:-}" ]]; then
            local default="${scenario_template_optional_defaults[$k]}"
            scenario_template_values["$k"]=$(scenario::template::render_default "$default" scenario_template_values "$current_date" "$random_token")
        fi
    done

    scenario_template_values[CURRENT_DATE]="$current_date"
    scenario_template_values[RANDOM_TOKEN]="$random_token"

    if [[ -z "$dest_path" ]]; then
        dest_path="${APP_ROOT}/scenarios/${scenario_template_values[SCENARIO_ID]}"
    fi
    dest_path=$(scenario::template::absolute_path "$dest_path")

    if [[ $dry_run == true ]]; then
        log::info "[DRY-RUN] Would generate template $template_name at $dest_path"
        scenario::template::print_var_summary scenario_template_values
        return 0
    fi

    if [[ -e "$dest_path" ]]; then
        if [[ $force == false ]]; then
            log::error "Destination already exists: $dest_path (use --force to overwrite)"
            return 1
        fi
        rm -rf "$dest_path"
    fi

    mkdir -p "$dest_path"
    rsync -a --exclude '.DS_Store' "$template_dir/" "$dest_path/"
    rm -f "$dest_path/template.json"

    local replacements_json
    replacements_json=$(scenario::template::vars_to_json scenario_template_values)
    scenario::template::replace_placeholders "$dest_path" "$replacements_json"
    scenario::template::verify_no_placeholders "$dest_path"

    log::success "Created ${scenario_template_values[SCENARIO_DISPLAY_NAME]} at $dest_path"
    scenario::template::print_var_summary scenario_template_values
    scenario::template::print_next_steps "$dest_path" "$manifest"

    if [[ $run_hooks == true ]]; then
        scenario::template::run_hooks "$dest_path" "$manifest"
    else
        scenario::template::print_hooks "$manifest"
    fi
}

scenario::template::parse_var_arg() {
    local kv="$1"
    local -n ref="$2"
    if [[ "$kv" != *=* ]]; then
        log::error "Invalid --var format (expected KEY=VALUE): $kv"
        return 1
    fi
    local key="${kv%%=*}"
    local value="${kv#*=}"
    if [[ -z "$key" ]]; then
        log::error "Invalid --var key"
        return 1
    fi
    ref["$key"]="$value"
}

scenario::template::render_default() {
    local template="$1"
    local -n values="$2"
    local current_date="$3"
    local random_token="$4"
    if [[ -z "$template" ]]; then
        echo ""
        return
    fi
    local result="$template"
    for key in "${!values[@]}"; do
        result="${result//\{\{$key\}\}/${values[$key]}}"
    done
    result="${result//\{\{CURRENT_DATE\}\}/$current_date}"
    result="${result//\{\{RANDOM_TOKEN\}\}/$random_token}"
    echo "$result"
}

scenario::template::vars_to_json() {
    local -n values="$1"
    local jq_args=()
    for key in "${!values[@]}"; do
        jq_args+=(--arg "$key" "${values[$key]}")
    done
    if [[ ${#jq_args[@]} -eq 0 ]]; then
        jq -n '{}'
    else
        jq -n "${jq_args[@]}" '$ARGS.named'
    fi
}

scenario::template::replace_placeholders() {
    local dest="$1"
    local replacements="$2"
    if ! command -v python3 >/dev/null 2>&1; then
        log::error "python3 is required to replace template placeholders"
        exit 1
    fi
    python3 - "$dest" "$replacements" <<'SCENARIO_TEMPLATE_PY'
import json, os, sys
root = sys.argv[1]
replacements = json.loads(sys.argv[2])
skip_dirs = {'.git', 'node_modules', '.pnpm'}
for dirpath, dirnames, filenames in os.walk(root):
    dirnames[:] = [d for d in dirnames if d not in skip_dirs]
    for fname in filenames:
        path = os.path.join(dirpath, fname)
        try:
            with open(path, 'r', encoding='utf-8') as fh:
                data = fh.read()
        except UnicodeDecodeError:
            continue
        original = data
        for key, value in replacements.items():
            data = data.replace('{{' + key + '}}', value)
        if data != original:
            with open(path, 'w', encoding='utf-8') as fh:
                fh.write(data)
SCENARIO_TEMPLATE_PY
}

scenario::template::verify_no_placeholders() {
    local dest="$1"
    local temp_file
    temp_file=$(mktemp)
    if rg --no-heading -n '\{\{[A-Z0-9_]+\}\}' "$dest" >"$temp_file" 2>/dev/null; then
        log::error "Unresolved placeholders detected:"
        cat "$temp_file"
        rm -f "$temp_file"
        exit 1
    fi
    rm -f "$temp_file"
}

scenario::template::absolute_path() {
    local path="$1"
    if command -v realpath >/dev/null 2>&1; then
        realpath "$path"
    else
        python3 - "$path" <<'SCENARIO_TEMPLATE_ABS'
import os, sys
print(os.path.abspath(sys.argv[1]))
SCENARIO_TEMPLATE_ABS
    fi
}

scenario::template::random_token() {
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex 16
    else
        python3 - <<'SCENARIO_TEMPLATE_TOKEN'
import secrets
print(secrets.token_hex(16))
SCENARIO_TEMPLATE_TOKEN
    fi
}

scenario::template::print_var_summary() {
    local -n values="$1"
    echo "Applied variables:"
    for key in $(printf '%s\n' "${!values[@]}" | sort); do
        printf "  - %s=%s\n" "$key" "${values[$key]}"
    done
}

scenario::template::print_next_steps() {
    local dest="$1"
    local manifest="$2"
    printf "\nNext steps:\n"
    printf "  1. Draft PRD.md → %s/PRD.md\n" "$dest"
    printf "  2. Seed requirements → %s/requirements/index.json\n" "$dest"
    printf "  3. Update progress log → %s/docs/PROGRESS.md\n" "$dest"
    printf "  4. Run: vrooli scenario status %s\n" "$(basename "$dest")"

    local docs
    docs=$(jq -r '.docs // {} | to_entries[] | "  - " + (.key) + ": " + (.value)' "$manifest")
    if [[ -n "$docs" ]]; then
        printf "\nReference docs:\n"
        printf "%s\n" "$docs"
    fi
}

scenario::template::run_hooks() {
    local dest="$1"
    local manifest="$2"
    local idx=0
    local ran=false
    while IFS= read -r hook; do
        ran=true
        idx=$((idx + 1))
        local desc cmd cwd
        desc=$(jq -r '.description // ""' <<<"$hook")
        cmd=$(jq -r '.cmd' <<<"$hook")
        cwd=$(jq -r '.cwd // "."' <<<"$hook")
        log::info "[Hook $idx] $desc"
        (cd "$dest/$cwd" && eval "$cmd")
    done < <(jq -c '.postHooks // [] | .[]' "$manifest")

    if [[ "$ran" != true ]]; then
        log::info "No post hooks defined for this template"
    fi
}

scenario::template::print_hooks() {
    local manifest="$1"
    local hooks
    hooks=$(jq -r '.postHooks // [] | to_entries[] | "  - " + (.value.description // .value.cmd)' "$manifest")
    if [[ -z "$hooks" ]]; then
        return
    fi
    printf "\nPost hooks (run manually if needed):\n"
    echo "$hooks"
}
