#!/usr/bin/env bash
# Scenario Insights Module
# Gathers stack/resource/package/lifecycle metadata for scenario status views

set -euo pipefail

# Locate a scenario's service.json file
scenario::insights::find_service_json() {
    local scenario_path="$1"

    if [[ -f "${scenario_path}/.vrooli/service.json" ]]; then
        echo "${scenario_path}/.vrooli/service.json"
    elif [[ -f "${scenario_path}/service.json" ]]; then
        echo "${scenario_path}/service.json"
    else
        echo ""
    fi
}

# Collect aggregated insights for a scenario as JSON
scenario::insights::collect_data() {
    local scenario_name="$1"
    local scenario_path="${APP_ROOT}/scenarios/${scenario_name}"

    if [[ ! -d "$scenario_path" ]]; then
        echo '{"stack":{},"resources":{},"packages":{},"lifecycle":{}}'
        return 0
    fi

    local service_json
    service_json=$(scenario::insights::find_service_json "$scenario_path")

    local stack_json resources_json packages_json lifecycle_json metadata_json documentation_json health_config_json
    stack_json=$(scenario::insights::collect_stack_data "$scenario_path" "$service_json")
    resources_json=$(scenario::insights::collect_resource_data "$scenario_name" "$service_json")
    packages_json=$(scenario::insights::collect_workspace_packages "$scenario_path")
    lifecycle_json=$(scenario::insights::collect_lifecycle_data "$service_json")
    metadata_json=$(scenario::insights::collect_metadata "$scenario_name" "$service_json")
    documentation_json=$(scenario::insights::collect_documentation "$scenario_name")
    health_config_json=$(scenario::insights::collect_health_config "$service_json")

    jq -n \
        --argjson stack "$stack_json" \
        --argjson resources "$resources_json" \
        --argjson packages "$packages_json" \
        --argjson lifecycle "$lifecycle_json" \
        --argjson metadata "$metadata_json" \
        --argjson documentation "$documentation_json" \
        --argjson health_config "$health_config_json" \
        '{stack: $stack, resources: $resources, packages: $packages, lifecycle: $lifecycle, metadata: $metadata, documentation: $documentation, health_config: $health_config}'
}

# Collect high-level metadata (display name, description, tags)
scenario::insights::collect_metadata() {
    local scenario_name="$1"
    local service_json="$2"

    local display_name=""
    local description=""
    local tags_json='[]'

    if [[ -n "$service_json" ]] && [[ -f "$service_json" ]]; then
        display_name=$(jq -r '.service.displayName // .service.display_name // ""' "$service_json" 2>/dev/null || echo "")
        description=$(jq -r '.service.description // ""' "$service_json" 2>/dev/null || echo "")
        tags_json=$(jq -c '.service.tags // []' "$service_json" 2>/dev/null || echo '[]')
    fi

    # Fallback to scenario id if no display name defined
    if [[ -z "$display_name" ]]; then
        display_name="$scenario_name"
    fi

    jq -n \
        --arg display_name "$display_name" \
        --arg description "$description" \
        --argjson tags "$tags_json" \
        '{display_name: $display_name, description: (if $description == "" then null else $description end), tags: $tags}'
}

# Collect documentation and runbook references
scenario::insights::collect_documentation() {
    local scenario_name="$1"
    local scenario_rel_path="scenarios/${scenario_name}"

    local docs='[]'
    local base_dir
    base_dir="${APP_ROOT:-$(pwd -P)}"
    base_dir="${base_dir%/}"

    local readme_path="${scenario_rel_path}/README.md"
    local readme_abs="${base_dir}/${readme_path}"
    if [[ -f "${base_dir}/${readme_path}" ]]; then
        docs=$(echo "$docs" | jq --arg label "README" --arg path "$readme_path" --arg abs "$readme_abs" '. += [{label: $label, path: $path, absolute_path: $abs, exists: true}]')
    else
        docs=$(echo "$docs" | jq --arg label "README" --arg path "$readme_path" --arg abs "$readme_abs" '. += [{label: $label, path: $path, absolute_path: $abs, exists: false, suggestion: "Create a project README summarizing business value and setup."}]')
    fi

    local prd_path="${scenario_rel_path}/PRD.md"
    local prd_abs="${base_dir}/${prd_path}"
    if [[ -f "${base_dir}/${prd_path}" ]]; then
        docs=$(echo "$docs" | jq --arg label "PRD" --arg path "$prd_path" --arg abs "$prd_abs" '. += [{label: $label, path: $path, absolute_path: $abs, exists: true}]')
    else
        docs=$(echo "$docs" | jq --arg label "PRD" --arg path "$prd_path" --arg abs "$prd_abs" '. += [{label: $label, path: $path, absolute_path: $abs, exists: false, suggestion: "Document product requirements in PRD.md to clarify acceptance criteria."}]')
    fi

    local playbook_path="${scenario_rel_path}/RUNBOOK.md"
    local playbook_abs="${base_dir}/${playbook_path}"
    if [[ -f "${base_dir}/${playbook_path}" ]]; then
        docs=$(echo "$docs" | jq --arg label "Runbook" --arg path "$playbook_path" --arg abs "$playbook_abs" '. += [{label: $label, path: $path, absolute_path: $abs, exists: true}]')
    fi

    jq -n --argjson docs "$docs" '{items: $docs}'
}

# Summarize health configuration coverage from service.json
scenario::insights::collect_health_config() {
    local service_json="$1"

    if [[ -z "$service_json" ]] || [[ ! -f "$service_json" ]]; then
        echo '{"present":false,"description":null,"endpoints":{},"missing_core_endpoints":["api","ui"],"checks":{"count":0,"critical":0,"items":[]}}'
        return 0
    fi

    local config_raw
    config_raw=$(jq -c '(.lifecycle.health // .health // null)' "$service_json" 2>/dev/null || echo "null")

    if [[ -z "$config_raw" ]] || [[ "$config_raw" == "null" ]]; then
        echo '{"present":false,"description":null,"endpoints":{},"missing_core_endpoints":["api","ui"],"checks":{"count":0,"critical":0,"items":[]}}'
        return 0
    fi

    echo "$config_raw" | jq -c '
        def safe_checks: (.checks // []);
        def endpoint_keys: ((.endpoints // {}) | keys);
        {
            present: true,
            description: (.description // null),
            endpoints: (.endpoints // {}),
            missing_core_endpoints: (["api","ui"] - endpoint_keys),
            checks: {
                count: (safe_checks | length),
                critical: (safe_checks | map(select(.critical == true)) | length),
                items: (safe_checks | map({name: (.name // null), type: (.type // null), target: (.target // null), critical: (.critical // false)}))
            }
        }
    '
}

# Build tech stack metadata
scenario::insights::collect_stack_data() {
    local scenario_path="$1"
    local service_json="$2"

    local tags_json="[]"
    if [[ -n "$service_json" ]] && [[ -f "$service_json" ]]; then
        tags_json=$(jq -c '.service.tags // []' "$service_json" 2>/dev/null || echo '[]')
    fi

    local detection='{}'
    local components=()

    if [[ -f "${scenario_path}/api/go.mod" ]]; then
        components+=("Go API")
        detection=$(echo "$detection" | jq '.api = {language: "Go", path: "api"}')
    elif [[ -f "${scenario_path}/api/pyproject.toml" ]]; then
        components+=("Python API")
        detection=$(echo "$detection" | jq '.api = {language: "Python", path: "api"}')
    elif [[ -f "${scenario_path}/api/package.json" ]]; then
        components+=("Node.js API")
        detection=$(echo "$detection" | jq '.api = {language: "Node.js", path: "api"}')
    fi

    local ui_dir=""
    for candidate in ui frontend web app; do
        if [[ -f "${scenario_path}/${candidate}/package.json" ]]; then
            ui_dir="$candidate"
            break
        fi
    done

    if [[ -n "$ui_dir" ]]; then
        local pkg_file="${scenario_path}/${ui_dir}/package.json"
        local framework=""
        local renderer=""
        local scripts_json
        scripts_json=$(jq -c '.scripts // {}' "$pkg_file" 2>/dev/null || echo '{}')

        if jq -e '((.dependencies // {}) + (.devDependencies // {}) + (.peerDependencies // {})) | has("next")' "$pkg_file" >/dev/null 2>&1; then
            framework="Next.js"
        elif jq -e '((.dependencies // {}) + (.devDependencies // {}) + (.peerDependencies // {})) | has("@angular/core")' "$pkg_file" >/dev/null 2>&1; then
            framework="Angular"
        elif jq -e '((.dependencies // {}) + (.devDependencies // {}) + (.peerDependencies // {})) | has("vue")' "$pkg_file" >/dev/null 2>&1; then
            framework="Vue"
        elif jq -e '((.dependencies // {}) + (.devDependencies // {}) + (.peerDependencies // {})) | has("svelte")' "$pkg_file" >/dev/null 2>&1; then
            framework="Svelte"
        elif jq -e '((.dependencies // {}) + (.devDependencies // {}) + (.peerDependencies // {})) | has("react")' "$pkg_file" >/dev/null 2>&1; then
            framework="React"
        fi

        if jq -e '((.dependencies // {}) + (.devDependencies // {}) + (.peerDependencies // {})) | has("vite")' "$pkg_file" >/dev/null 2>&1; then
            renderer="Vite"
        elif echo "$scripts_json" | jq -e '.dev // "" | test("webpack")' >/dev/null 2>&1; then
            renderer="Webpack"
        elif echo "$scripts_json" | jq -e '.dev // "" | test("astro")' >/dev/null 2>&1; then
            renderer="Astro"
        fi

        local ui_component=""
        if [[ -n "$framework" ]]; then
            if [[ -n "$renderer" ]] && [[ "$framework" != "Next.js" ]]; then
                ui_component="$framework + $renderer UI"
            else
                ui_component="$framework UI"
            fi
        else
            ui_component="Web UI (${ui_dir})"
        fi
        components+=("$ui_component")
        detection=$(echo "$detection" | jq --arg path "$ui_dir" --arg framework "$framework" --arg renderer "$renderer" '.ui = {path: $path, framework: ($framework // null), renderer: ($renderer // null)}')
    fi

    # Map well-known tags to components (avoid duplicates)
    if [[ "$tags_json" != "[]" ]]; then
        local tag_components=()
        if echo "$tags_json" | jq -e '.[] | select(. == "postgres-storage")' >/dev/null 2>&1; then
            tag_components+=("PostgreSQL")
        fi
        if echo "$tags_json" | jq -e '.[] | select(. == "redis-cache")' >/dev/null 2>&1; then
            tag_components+=("Redis")
        fi
        if echo "$tags_json" | jq -e '.[] | select(. == "browser-automation")' >/dev/null 2>&1; then
            tag_components+=("Browser Automation")
        fi
        if echo "$tags_json" | jq -e '.[] | select(. == "ai-debugging")' >/dev/null 2>&1; then
            tag_components+=("AI Tooling")
        fi
        if [[ ${#tag_components[@]} -gt 0 ]]; then
            components+=("${tag_components[@]}")
        fi
    fi

    local components_json
    if [[ ${#components[@]} -gt 0 ]]; then
        components_json=$(printf '%s
' "${components[@]}" | jq -R -s 'split("\n") | map(select(length > 0))')
    else
        components_json='[]'
    fi

    jq -n \
        --argjson components "$components_json" \
        --argjson tags "$tags_json" \
        --argjson detection "$detection" \
        '{components: $components, tags: $tags, detection: $detection}'
}

# Collect resource readiness data
scenario::insights::collect_resource_data() {
    local scenario_name="$1"
    local service_json="$2"

    if [[ -z "$service_json" ]] || [[ ! -f "$service_json" ]]; then
        echo '{"items":[],"summary":{"total":0,"required":0,"required_running":0,"issues":0}}'
        return 0
    fi

    local resources
    resources=$(jq -c '.resources // {} | to_entries[] | {name: .key, type: (.value.type // "custom"), required: (.value.required // false), enabled: (.value.enabled // false), description: (.value.description // "")}' "$service_json" 2>/dev/null || echo '')

    if [[ -z "$resources" ]]; then
        echo '{"items":[],"summary":{"total":0,"required":0,"required_running":0,"issues":0}}'
        return 0
    fi

    local -a items=()
    while IFS= read -r resource; do
        [[ -z "$resource" ]] && continue
        local name type required enabled description
        name=$(echo "$resource" | jq -r '.name')
        type=$(echo "$resource" | jq -r '.type')
        required=$(echo "$resource" | jq -r '.required')
        enabled=$(echo "$resource" | jq -r '.enabled')
        description=$(echo "$resource" | jq -r '.description')

        local status="unknown"
        local running=false
        local healthy=null
        local installed=false
        local note=""
        local resource_json=""

        if command -v vrooli >/dev/null 2>&1; then
            if resource_json=$(vrooli resource status "$name" --json 2>/dev/null); then
                running=$(echo "$resource_json" | jq -r '.running // false')
                healthy=$(echo "$resource_json" | jq -r '.healthy // null')
                installed=$(echo "$resource_json" | jq -r '.installed // false')
                status=$(echo "$resource_json" | jq -r '.status // (if .running then "Running" else "" end)')
                if [[ "$installed" != "true" ]]; then
                    status="not_installed"
                    note=""
                elif [[ "$running" == "true" ]]; then
                    if [[ "$healthy" == "true" ]]; then
                        status="running"
                    else
                        status="running_with_issues"
                        note="Reported unhealthy"
                    fi
                else
                    status="stopped"
                fi
            else
                status="unavailable"
                note="Unable to query resource status"
            fi
        else
            status="unavailable"
            note="vrooli CLI not available"
        fi

        local item
        item=$(jq -n \
            --arg name "$name" \
            --arg type "$type" \
            --arg description "$description" \
            --arg status "$status" \
            --arg note "$note" \
            --argjson required "$required" \
            --argjson enabled "$enabled" \
            --argjson running "$running" \
            --argjson healthy "$healthy" \
            --argjson installed "$installed" \
            '{name: $name, type: $type, description: $description, required: $required, enabled: $enabled, status: $status, running: $running, healthy: $healthy, installed: $installed, note: (if $note == "" then null else $note end)}')
        items+=("$item")
    done <<< "$resources"

    local items_json
    if [[ ${#items[@]} -gt 0 ]]; then
        items_json=$(printf '%s
' "${items[@]}" | jq -s '.')
    else
        items_json='[]'
    fi

    local summary
    summary=$(echo "$items_json" | jq '{
        total: length,
        required: ([.[] | select(.required == true)] | length),
        required_running: ([.[] | select(.required == true and .running == true)] | length),
        issues: ([.[] | select(.required == true and (.running != true or .status == "running_with_issues" or .status == "not_installed"))] | length)
    }')

    jq -n --argjson items "$items_json" --argjson summary "$summary" '{items: $items, summary: $summary}'
}

# Collect workspace package alignment data
scenario::insights::collect_workspace_packages() {
    local scenario_path="$1"

    if [[ ! -d "$scenario_path" ]]; then
        echo '{"items":[],"summary":{"total":0,"aligned":0,"issues":0}}'
        return 0
    fi

    local -a package_files=()
    while IFS= read -r file; do
        package_files+=("$file")
    done < <(find "$scenario_path" -maxdepth 2 -name package.json -not -path "*/node_modules/*" -print 2>/dev/null | sort)

    local -a items=()

    for pkg_file in "${package_files[@]}"; do
        local pkg_dir
        pkg_dir=$(dirname "$pkg_file")
        local deps
        deps=$(jq -c '[((.dependencies // {}) + (.devDependencies // {}) + (.peerDependencies // {}))
            | to_entries[]
            | select((.value | type == "string") and (.value | test("^(file|workspace|link):")))
            | {name: .key, spec: .value}
        ]' "$pkg_file" 2>/dev/null || echo '[]')

        if [[ "$deps" == "[]" ]]; then
            continue
        fi

        local dep
        while IFS= read -r dep; do
            [[ -z "$dep" || "$dep" == "null" ]] && continue
            local name spec
            name=$(echo "$dep" | jq -r '.name')
            spec=$(echo "$dep" | jq -r '.spec')

            local rel_path=""
            if [[ "$spec" == file:* ]]; then
                rel_path="${spec#file:}"
            elif [[ "$spec" == link:* ]]; then
                rel_path="${spec#link:}"
            elif [[ "$spec" == workspace:* ]]; then
                rel_path="packages/${name##*/}"
            fi

            local abs_path=""
            if [[ -n "$rel_path" ]]; then
                abs_path=$(builtin cd "$pkg_dir" >/dev/null 2>&1 && realpath "$rel_path" 2>/dev/null || echo "")
            fi

            local root_version=""
            if [[ -n "$abs_path" ]] && [[ -f "$abs_path/package.json" ]]; then
                root_version=$(jq -r '.version // ""' "$abs_path/package.json" 2>/dev/null)
            fi

            local node_module_path="$pkg_dir/node_modules/${name}"
            local installed_version=""
            if [[ -f "$node_module_path/package.json" ]]; then
                installed_version=$(jq -r '.version // ""' "$node_module_path/package.json" 2>/dev/null)
            fi

            local status="aligned"
            local note=""
            if [[ -z "$installed_version" ]]; then
                status="missing"
                note="Dependency not installed"
            elif [[ -n "$root_version" ]] && [[ "$installed_version" != "$root_version" ]]; then
                status="drift"
                note="Root ${root_version:-unknown} vs installed ${installed_version:-unknown}"
            fi

            local pkg_item
            pkg_item=$(jq -n \
                --arg name "$name" \
                --arg spec "$spec" \
                --arg package_json "${pkg_file#${APP_ROOT}/}" \
                --arg root_version "$root_version" \
                --arg installed_version "$installed_version" \
                --arg status "$status" \
                --arg note "$note" \
                '{name: $name, spec: $spec, package_json: $package_json, root_version: (if $root_version == "" then null else $root_version end), installed_version: (if $installed_version == "" then null else $installed_version end), status: $status, note: (if $note == "" then null else $note end)}')
            items+=("$pkg_item")
        done <<< "$(echo "$deps" | jq -c '.[]')"
    done

    local items_json
    if [[ ${#items[@]} -gt 0 ]]; then
        items_json=$(printf '%s
' "${items[@]}" | jq -s '.')
    else
        items_json='[]'
    fi

    local summary
    summary=$(echo "$items_json" | jq '{
        total: length,
        aligned: ([.[] | select(.status == "aligned")] | length),
        issues: ([.[] | select(.status != "aligned")] | length)
    }')

    jq -n --argjson items "$items_json" --argjson summary "$summary" '{items: $items, summary: $summary}'
}

# Summarize lifecycle coverage
scenario::insights::collect_lifecycle_data() {
    local service_json="$1"

    if [[ -z "$service_json" ]] || [[ ! -f "$service_json" ]]; then
        echo '{"phases":[],"summary":{"defined":[],"missing":[]}}'
        return 0
    fi

    local service_data
    service_data=$(cat "$service_json")

    jq -n --argjson svc "$service_data" '
      ( ["setup","develop","test","production","stop"] | map(
          ($svc.lifecycle[.]? // null) as $phase
          | {
              name: .,
              defined: ($phase != null),
              steps: (if $phase != null then ($phase.steps // []) | length else 0 end),
              description: (if $phase != null then ($phase.description // "") else null end),
              status: (if $phase == null then "missing" elif (($phase.steps // []) | length) > 0 then "defined" else "empty" end)
            }
        )) as $phases
        | {
            phases: $phases,
            summary: {
              defined: [$phases[] | select(.defined == true) | .name],
              missing: [$phases[] | select(.defined != true) | .name]
            }
          }
    '
}

# Display helpers ------------------------------------------------------------

scenario::insights::display_metadata() {
    local insights_json="$1"
    local scenario_name="$2"

    local metadata_json
    metadata_json=$(echo "$insights_json" | jq -c '.metadata // {}' 2>/dev/null || echo '{}')

    if [[ "$metadata_json" == "{}" ]]; then
        return 0
    fi

    local display_name description tags joined_tags
    display_name=$(echo "$metadata_json" | jq -r '.display_name // ""')
    description=$(echo "$metadata_json" | jq -r '.description // ""')
    joined_tags=$(echo "$metadata_json" | jq -r 'if (.tags // [] | length) > 0 then (.tags | join(", ")) else "" end')

    # Avoid repeating scenario id if nothing custom is provided
    if [[ "$display_name" == "$scenario_name" ]] && [[ -z "$description" ]] && [[ -n "$joined_tags" ]]; then
        :
    fi

    if [[ "$display_name" == "$scenario_name" ]] && [[ -z "$description" ]] && [[ -z "$joined_tags" ]]; then
        return 0
    fi

    echo "Profile:"
    if [[ -n "$display_name" ]] && [[ "$display_name" != "$scenario_name" ]]; then
        printf '  Name: %s\n' "$display_name"
    fi
    if [[ -n "$description" ]]; then
        echo "  Description:"
        echo "$description" | fold -s -w 72 | sed 's/^/    /'
    fi
    if [[ -n "$joined_tags" ]]; then
        printf '  Tags: %s\n' "$joined_tags"
    fi
    echo ""
}

scenario::insights::display_documentation() {
    local insights_json="$1"
    local docs_json
    docs_json=$(echo "$insights_json" | jq -c '.documentation.items // []' 2>/dev/null || echo '[]')

    local doc_count
    doc_count=$(echo "$docs_json" | jq 'length' 2>/dev/null || echo 0)
    if [[ "$doc_count" -eq 0 ]]; then
        return 0
    fi

    echo "Docs & Runbooks:"
    echo "$docs_json" | jq -c '.[]' | while read -r doc; do
        local label path absolute exists suggestion
        label=$(echo "$doc" | jq -r '.label')
        path=$(echo "$doc" | jq -r '.path')
        absolute=$(echo "$doc" | jq -r '.absolute_path // ""')
        exists=$(echo "$doc" | jq -r '.exists')
        suggestion=$(echo "$doc" | jq -r '.suggestion // ""')

        local path_display
        if [[ -n "$absolute" ]]; then
            path_display="$absolute"
        else
            local base_dir="${APP_ROOT:-$(pwd)}"
            base_dir="${base_dir%/}"
            path_display="${base_dir}/${path}"
        fi

        if [[ "$exists" == "true" ]]; then
            printf '  ✅ %s: %s\n' "$label" "$path_display"
        else
            printf '  ⚠️  %s missing — expected at %s\n' "$label" "$path_display"
            if [[ -n "$suggestion" ]]; then
                echo "     • $suggestion"
            fi
        fi
    done
    echo ""
}

scenario::insights::display_health_config() {
    local insights_json="$1"
    local coverage_json
    coverage_json=$(echo "$insights_json" | jq -c '.health_config // {}' 2>/dev/null || echo '{}')

    if [[ "$coverage_json" == "{}" ]]; then
        return 0
    fi

    local present
    present=$(echo "$coverage_json" | jq -r '.present // false')

    echo "Health Config:"
    if [[ "$present" != "true" ]]; then
        echo "  ⚠️  No health configuration defined in .vrooli/service.json"
        echo "     • Add lifecycle.health endpoints so diagnostics stay reliable"
        echo ""
        return 0
    fi

    local check_count critical_count
    check_count=$(echo "$coverage_json" | jq -r '.checks.count // 0')
    critical_count=$(echo "$coverage_json" | jq -r '.checks.critical // 0')
    printf '  Coverage: ✅ defined (%s checks, %s critical)\n' "$check_count" "$critical_count"

    local description
    description=$(echo "$coverage_json" | jq -r '.description // ""')
    if [[ -n "$description" ]]; then
        echo "  Purpose:"
        echo "$description" | fold -s -w 72 | sed 's/^/    /'
    fi

    local endpoints_json
    endpoints_json=$(echo "$coverage_json" | jq -c '.endpoints // {}')
    if [[ "$endpoints_json" != "{}" ]]; then
        echo "  Endpoints:"
        echo "$endpoints_json" | jq -r 'to_entries[] | "    • " + .key + " → " + (.value // "")'
    fi

    local missing_core
    missing_core=$(echo "$coverage_json" | jq -r '(.missing_core_endpoints // []) | join(", ")')
    if [[ -n "$missing_core" ]]; then
        echo "  ⚠️  Missing core endpoints: $missing_core"
    fi

    echo ""
}

scenario::insights::display_stack() {
    local insights_json="$1"
    local components
    components=$(echo "$insights_json" | jq -r '.stack.components[]?' 2>/dev/null || echo "")

    if [[ -z "$components" ]]; then
        return 0
    fi

    echo "Stack:"
    echo "$components" | while IFS= read -r component; do
        [[ -z "$component" ]] && continue
        printf '  • %s\n' "$component"
    done
    echo ""
}

scenario::insights::display_resources() {
    local insights_json="$1"
    local items_json
    items_json=$(echo "$insights_json" | jq -c '.resources.items' 2>/dev/null || echo '[]')

    if [[ "$items_json" == "[]" ]]; then
        return 0
    fi

    echo "Dependencies:"
    echo "$items_json" | jq -c '.[]' | while read -r item; do
        local name required running status note
        name=$(echo "$item" | jq -r '.name')
        required=$(echo "$item" | jq -r '.required')
        running=$(echo "$item" | jq -r '.running')
        status=$(echo "$item" | jq -r '.status')
        note=$(echo "$item" | jq -r '.note // empty')

        local icon message
        if [[ "$status" == "running" ]]; then
            icon="✅"
            message="${name}"
        elif [[ "$status" == "running_with_issues" ]]; then
            icon="🟡"
            message="${name} (running, unhealthy)"
        elif [[ "$status" == "not_installed" ]]; then
            icon="❌"
            message="${name} not installed"
        elif [[ "$status" == "stopped" ]]; then
            icon=$([[ "$required" == "true" ]] && echo "❌" || echo "⚠️")
            message=$([[ "$required" == "true" ]] && echo "${name} stopped" || echo "${name} optional, stopped")
        else
            icon=$([[ "$required" == "true" ]] && echo "⚠️" || echo "ℹ️")
            message="${name} status unknown"
        fi

        printf '  %s %s' "$icon" "$message"
        if [[ -n "$note" ]]; then
            printf ' — %s' "$note"
        elif [[ "$status" == "stopped" && "$required" == "true" ]]; then
            printf ' — start with: vrooli resource start %s' "$name"
        elif [[ "$status" == "not_installed" ]]; then
            printf ' — install with: vrooli resource install %s' "$name"
        fi
        printf '\n'
    done
    echo ""
}

scenario::insights::display_workspace_packages() {
    local insights_json="$1"
    local items_json
    items_json=$(echo "$insights_json" | jq -c '.packages.items' 2>/dev/null || echo '[]')

    if [[ "$items_json" == "[]" ]]; then
        return 0
    fi

    echo "Workspace Packages:"
    echo "$items_json" | jq -c '.[]' | while read -r item; do
        local name status root_version installed_version note
        name=$(echo "$item" | jq -r '.name')
        status=$(echo "$item" | jq -r '.status')
        root_version=$(echo "$item" | jq -r '.root_version // empty')
        installed_version=$(echo "$item" | jq -r '.installed_version // empty')
        note=$(echo "$item" | jq -r '.note // empty')

        local icon detail
        case "$status" in
            aligned)
                icon="✅"
                detail="$name"
                if [[ -n "$root_version" ]]; then
                    detail+=" (${root_version})"
                fi
                ;;
            drift)
                icon="❌"
                detail="$name"
                if [[ -n "$note" ]]; then
                    detail+=" — $note"
                fi
                ;;
            missing)
                icon="⚠️"
                detail="$name — install dependencies"
                ;;
            *)
                icon="ℹ️"
                detail="$name"
                ;;
        esac

        printf '  %s %s\n' "$icon" "$detail"
    done
    echo ""
}

scenario::insights::display_lifecycle() {
    local insights_json="$1"
    local phases_json
    phases_json=$(echo "$insights_json" | jq -c '.lifecycle.phases' 2>/dev/null || echo '[]')

    if [[ "$phases_json" == "[]" ]]; then
        return 0
    fi

    echo "Lifecycle:"
    echo "$phases_json" | jq -c '.[]' | while read -r phase; do
        local name status steps
        name=$(echo "$phase" | jq -r '.name')
        status=$(echo "$phase" | jq -r '.status')
        steps=$(echo "$phase" | jq -r '.steps')

        local icon detail
        case "$status" in
            defined)
                icon="✅"
                ;;
            empty)
                icon="⚠️"
                ;;
            missing)
                icon="⚠️"
                ;;
            *)
                icon="ℹ️"
                ;;
        esac
        local plural="s"
        if [[ "$steps" == "1" ]]; then
            plural=""
        fi
        if [[ "$status" == "defined" ]]; then
            detail="$name (${steps} step${plural})"
        elif [[ "$status" == "empty" ]]; then
            detail="$name defined but no steps"
        elif [[ "$status" == "missing" ]]; then
            detail="$name missing"
        else
            detail="$name"
        fi
        printf '  %s %s\n' "$icon" "$detail"
    done
    echo ""
}
