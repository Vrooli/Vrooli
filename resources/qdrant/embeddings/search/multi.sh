#!/usr/bin/env bash
# Multi-App Search for Qdrant Embeddings
# Enables semantic search across all apps in the system

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Define paths from APP_ROOT
QDRANT_DIR="${APP_ROOT}/resources/qdrant"
EMBEDDINGS_DIR="${QDRANT_DIR}/embeddings"
SEARCH_DIR="${EMBEDDINGS_DIR}/search"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source search components
source "${SEARCH_DIR}/single.sh"
source "${EMBEDDINGS_DIR}/indexers/identity.sh"

# Default settings
DEFAULT_LIMIT=20
DEFAULT_MIN_SCORE=0.5

#######################################
# Search across all apps
# Arguments:
#   $1 - Query text
#   $2 - Type filter (optional: all|workflows|scenarios|knowledge|code|resources)
#   $3 - Limit per app (optional, default: 5)
#   $4 - Min score (optional, default: 0.5)
# Returns: Aggregated JSON results
#######################################
qdrant::search::all_apps() {
    local query="$1"
    local type="${2:-all}"
    local limit_per_app="${3:-5}"
    local min_score="${4:-$DEFAULT_MIN_SCORE}"
    
    if [[ -z "$query" ]]; then
        log::error "Query is required"
        return 1
    fi
    
    log::info "Searching all apps for: $query"
    local search_start=$(date +%s%3N)
    
    # Find all app identities
    local app_ids=()
    while IFS= read -r identity_file; do
        if [[ -f "$identity_file" ]]; then
            local app_id=$(jq -r '.app_id // empty' "$identity_file" 2>/dev/null)
            if [[ -n "$app_id" ]]; then
                app_ids+=("$app_id")
            fi
        fi
    done < <(find "${HOME}" -name "app-identity.json" -type f 2>/dev/null | grep -E "/.vrooli/app-identity.json$")
    
    # Fallback: search for collections by pattern
    if [[ ${#app_ids[@]} -eq 0 ]]; then
        log::debug "No app identities found, searching by collection pattern"
        app_ids=($(qdrant::search::discover_apps))
    fi
    
    if [[ ${#app_ids[@]} -eq 0 ]]; then
        log::warn "No apps found to search"
        echo '{
            "query": "'"$query"'",
            "type": "'"$type"'",
            "results": [],
            "apps_searched": 0,
            "total_results": 0,
            "search_time_ms": 0
        }'
        return 0
    fi
    
    log::debug "Found ${#app_ids[@]} apps to search"
    
    # Search each app in parallel (if possible)
    local all_results="[]"
    local apps_with_results=0
    
    for app_id in "${app_ids[@]}"; do
        log::debug "Searching app: $app_id"
        
        # Search single app
        local app_results=$(qdrant::search::single_app \
            "$query" \
            "$app_id" \
            "$type" \
            "$limit_per_app" \
            "$min_score" 2>/dev/null)
        
        if [[ -n "$app_results" ]]; then
            local result_count=$(echo "$app_results" | jq -r '.result_count // 0')
            if [[ "$result_count" -gt 0 ]]; then
                ((apps_with_results++))
                
                # Extract just the results array
                local results_array=$(echo "$app_results" | jq -c '.results // []')
                
                # Merge with all results
                all_results=$(echo "$all_results $results_array" | jq -s 'add')
            fi
        fi
    done
    
    # Sort all results by score and apply global limit
    local total_limit=$((limit_per_app * ${#app_ids[@]}))
    if [[ $total_limit -gt $DEFAULT_LIMIT ]]; then
        total_limit=$DEFAULT_LIMIT
    fi
    
    all_results=$(echo "$all_results" | jq --argjson limit "$total_limit" \
        'sort_by(-.score) | .[0:$limit]')
    
    # Calculate search time
    local search_end=$(date +%s%3N)
    local search_time=$((search_end - search_start))
    
    # Build final response
    jq -n \
        --arg query "$query" \
        --arg type "$type" \
        --argjson results "$all_results" \
        --arg apps_searched "${#app_ids[@]}" \
        --arg apps_with_results "$apps_with_results" \
        --arg total_results "$(echo "$all_results" | jq 'length')" \
        --arg search_time "$search_time" \
        '{
            query: $query,
            type: $type,
            results: $results,
            apps_searched: ($apps_searched | tonumber),
            apps_with_results: ($apps_with_results | tonumber),
            total_results: ($total_results | tonumber),
            search_time_ms: ($search_time | tonumber),
            timestamp: now | strftime("%Y-%m-%dT%H:%M:%SZ")
        }'
}

#######################################
# Discover apps from Qdrant collections
# Returns: Array of app IDs
#######################################
qdrant::search::discover_apps() {
    local collections=$(curl -s -X GET \
        "${QDRANT_URL:-http://localhost:6333}/collections" 2>/dev/null | \
        jq -r '.result.collections[].name // empty' 2>/dev/null)
    
    if [[ -z "$collections" ]]; then
        return
    fi
    
    # Extract app IDs from collection names (format: app-id-type)
    local app_ids=()
    while IFS= read -r collection; do
        # Remove the suffix to get app ID
        local app_id="${collection%-workflows}"
        app_id="${app_id%-scenarios}"
        app_id="${app_id%-knowledge}"
        app_id="${app_id%-code}"
        app_id="${app_id%-resources}"
        
        # Add if not already in array
        if [[ ! " ${app_ids[@]} " =~ " ${app_id} " ]]; then
            app_ids+=("$app_id")
        fi
    done <<< "$collections"
    
    printf '%s\n' "${app_ids[@]}"
}

#######################################
# Cross-app pattern discovery
# Arguments:
#   $1 - Query text
#   $2 - Min similarity threshold (optional, default: 0.7)
# Returns: Pattern analysis
#######################################
qdrant::search::discover_patterns() {
    local query="$1"
    local threshold="${2:-0.7}"
    
    log::info "Discovering patterns for: $query"
    
    # Search all apps
    local results=$(qdrant::search::all_apps "$query" "all" "10" "$threshold")
    
    if [[ -z "$results" ]] || [[ "$(echo "$results" | jq -r '.total_results')" -eq 0 ]]; then
        echo "No patterns found matching: $query"
        return 0
    fi
    
    # Analyze patterns
    echo "=== Pattern Discovery ==="
    echo "Query: \"$query\""
    echo
    
    # Group by type
    echo "By Type:"
    echo "$results" | jq -r '.results | group_by(.type) | .[] | 
        "\(.[]|.type // "unknown" | select(. == .)):" + 
        " \(length) occurrence(s)"' | sort -u
    echo
    
    # Group by app
    echo "By App:"
    echo "$results" | jq -r '.results | group_by(.app_id) | .[] | 
        "\(.[]|.app_id // "unknown" | select(. == .)):" + 
        " \(length) match(es)"' | sort -u
    echo
    
    # Show top patterns
    echo "Top Patterns (score > $threshold):"
    echo "$results" | jq -r --arg t "$threshold" '.results[] | 
        select(.score > ($t|tonumber)) |
        "[\(.score|tostring[0:4])] \(.type) in \(.app_id):\n  \(.content | split("\n")[0] | .[0:100])\n"'
}

#######################################
# Find reusable solutions
# Arguments:
#   $1 - Problem description
#   $2 - Preferred type (optional)
# Returns: Reusable solutions
#######################################
qdrant::search::find_solutions() {
    local problem="$1"
    local preferred_type="${2:-workflows}"
    
    log::info "Finding solutions for: $problem"
    
    # Search with focus on preferred type
    local primary_results=$(qdrant::search::all_apps "$problem" "$preferred_type" "5" "0.6")
    local secondary_results=$(qdrant::search::all_apps "$problem" "all" "3" "0.7")
    
    # Combine and deduplicate
    local combined=$(echo "$primary_results $secondary_results" | jq -s '
        .[0].results + .[1].results | 
        unique_by(.id) | 
        sort_by(-.score)')
    
    # Format as solutions
    echo "=== Reusable Solutions ==="
    echo "Problem: \"$problem\""
    echo
    
    local count=0
    echo "$combined" | jq -r '.[] | 
        "Solution #\(input_line_number):\n" +
        "  Type: \(.type)\n" +
        "  App: \(.app_id)\n" +
        "  Score: \(.score|tostring[0:4])\n" +
        "  Description: \(.content | split("\n")[0:2] | join(" "))\n" +
        (if .metadata.source_file then "  File: \(.metadata.source_file)\n" else "" end)' | 
    while IFS= read -r line; do
        echo "$line"
        ((count++))
        if [[ $count -ge 5 ]]; then
            break
        fi
    done
}

#######################################
# Compare solutions across apps
# Arguments:
#   $1 - Query text
#   $2 - App ID 1
#   $3 - App ID 2
# Returns: Comparison analysis
#######################################
qdrant::search::compare_apps() {
    local query="$1"
    local app1="$2"
    local app2="$3"
    
    if [[ -z "$query" ]] || [[ -z "$app1" ]] || [[ -z "$app2" ]]; then
        log::error "Query and two app IDs required"
        return 1
    fi
    
    echo "=== App Comparison ==="
    echo "Query: \"$query\""
    echo "Comparing: $app1 vs $app2"
    echo
    
    # Search both apps
    local results1=$(qdrant::search::single_app "$query" "$app1" "all" "5")
    local results2=$(qdrant::search::single_app "$query" "$app2" "all" "5")
    
    local count1=$(echo "$results1" | jq -r '.result_count // 0')
    local count2=$(echo "$results2" | jq -r '.result_count // 0')
    
    echo "$app1:"
    echo "  Results: $count1"
    if [[ "$count1" -gt 0 ]]; then
        echo "  Best Score: $(echo "$results1" | jq -r '.results[0].score // 0' | cut -c1-4)"
        echo "  Types: $(echo "$results1" | jq -r '[.results[].type] | unique | join(", ")')"
    fi
    echo
    
    echo "$app2:"
    echo "  Results: $count2"
    if [[ "$count2" -gt 0 ]]; then
        echo "  Best Score: $(echo "$results2" | jq -r '.results[0].score // 0' | cut -c1-4)"
        echo "  Types: $(echo "$results2" | jq -r '[.results[].type] | unique | join(", ")')"
    fi
    echo
    
    # Determine winner
    if [[ "$count1" -gt "$count2" ]]; then
        echo "Recommendation: $app1 has more relevant content"
    elif [[ "$count2" -gt "$count1" ]]; then
        echo "Recommendation: $app2 has more relevant content"
    else
        echo "Recommendation: Both apps have similar content"
    fi
}

#######################################
# Track knowledge evolution
# Arguments:
#   $1 - Concept to track
# Returns: Evolution timeline
#######################################
qdrant::search::track_evolution() {
    local concept="$1"
    
    log::info "Tracking evolution of: $concept"
    
    # Search all apps for the concept
    local results=$(qdrant::search::all_apps "$concept" "all" "20" "0.5")
    
    echo "=== Knowledge Evolution: $concept ==="
    echo
    
    # Extract dates and sort chronologically
    echo "$results" | jq -r '.results[] | 
        select(.metadata.modified or .metadata.created) |
        {
            date: (.metadata.modified // .metadata.created // "unknown"),
            app: .app_id,
            type: .type,
            content: (.content | split("\n")[0] | .[0:80])
        }' | jq -s 'sort_by(.date)' | jq -r '.[] |
        "[\(.date | split("T")[0])] \(.app)/\(.type): \(.content)"'
}

#######################################
# Find knowledge gaps
# Arguments:
#   $1 - Topic to analyze
# Returns: Gap analysis
#######################################
qdrant::search::find_gaps() {
    local topic="$1"
    
    log::info "Analyzing knowledge gaps for: $topic"
    
    # Search for the topic
    local results=$(qdrant::search::all_apps "$topic" "all" "10" "0.3")
    local total=$(echo "$results" | jq -r '.total_results // 0')
    
    echo "=== Knowledge Gap Analysis: $topic ==="
    echo
    
    if [[ "$total" -eq 0 ]]; then
        echo "❌ No knowledge found on this topic"
        echo "   Recommendation: Create new documentation or workflows"
        return 0
    fi
    
    # Analyze coverage by type
    local types=$(echo "$results" | jq -r '[.results[].type] | unique | sort')
    
    echo "Current Coverage:"
    echo "$results" | jq -r '.results | group_by(.type) | .[] | 
        "  • \(.[0].type): \(length) item(s)"'
    echo
    
    # Identify missing types
    echo "Potential Gaps:"
    local all_types='["workflows", "scenarios", "knowledge", "code", "resources"]'
    local missing=$(echo "$all_types $types" | jq -s '.[0] - .[1]')
    
    if [[ "$(echo "$missing" | jq 'length')" -gt 0 ]]; then
        echo "$missing" | jq -r '.[] | "  ❌ No \(.) found"'
    else
        echo "  ✅ All content types have coverage"
    fi
    echo
    
    # Analyze app coverage
    local app_count=$(echo "$results" | jq -r '.apps_with_results // 0')
    local total_apps=$(echo "$results" | jq -r '.apps_searched // 0')
    
    echo "App Coverage: $app_count/$total_apps apps have relevant content"
    if [[ "$app_count" -lt "$total_apps" ]]; then
        echo "  Consider adding $topic content to more apps"
    fi
}

#######################################
# Generate search report
# Arguments:
#   $1 - Query text
#   $2 - Output format (text|json|csv)
# Returns: Formatted report
#######################################
qdrant::search::report() {
    local query="$1"
    local format="${2:-text}"
    
    local results=$(qdrant::search::all_apps "$query" "all" "20" "0.4")
    
    case "$format" in
        json)
            echo "$results" | jq '.'
            ;;
        csv)
            echo "Score,Type,App,Content"
            echo "$results" | jq -r '.results[] | 
                [.score, .type, .app_id, (.content | gsub(","; ";") | .[0:100])] | 
                @csv'
            ;;
        text|*)
            echo "=== Search Report ==="
            echo "Query: \"$query\""
            echo "Timestamp: $(date -Iseconds)"
            echo
            echo "Summary:"
            echo "  Total Results: $(echo "$results" | jq -r '.total_results')"
            echo "  Apps Searched: $(echo "$results" | jq -r '.apps_searched')"
            echo "  Apps with Results: $(echo "$results" | jq -r '.apps_with_results')"
            echo "  Search Time: $(echo "$results" | jq -r '.search_time_ms')ms"
            echo
            echo "Top Results:"
            echo "$results" | jq -r '.results[0:5][] | 
                "\n#\(input_line_number). [\(.score|tostring[0:4])] \(.type) from \(.app_id)" +
                "\n   \(.content | split("\n")[0] | .[0:150])" +
                (if .metadata.source_file then "\n   File: \(.metadata.source_file)" else "" end)'
            ;;
    esac
}

#######################################
# Interactive search explorer
# Arguments: None (interactive)
# Returns: Interactive search session
#######################################
qdrant::search::explore() {
    echo "=== Qdrant Search Explorer ==="
    echo "Commands: search, patterns, solutions, gaps, compare, quit"
    echo
    
    while true; do
        read -p "search> " cmd args
        
        case "$cmd" in
            search)
                read -p "Query: " query
                read -p "Type (all/workflows/scenarios/knowledge/code/resources): " type
                type="${type:-all}"
                qdrant::search::all_apps "$query" "$type" "5" | jq -r '
                    "Found \(.total_results) results across \(.apps_with_results) apps\n" +
                    (.results[0:3][] | "\n[\(.score|tostring[0:4])] \(.type) in \(.app_id):\n  \(.content | split("\n")[0] | .[0:100])")'
                ;;
            patterns)
                read -p "Query: " query
                qdrant::search::discover_patterns "$query"
                ;;
            solutions)
                read -p "Problem: " problem
                qdrant::search::find_solutions "$problem"
                ;;
            gaps)
                read -p "Topic: " topic
                qdrant::search::find_gaps "$topic"
                ;;
            compare)
                read -p "Query: " query
                read -p "App 1: " app1
                read -p "App 2: " app2
                qdrant::search::compare_apps "$query" "$app1" "$app2"
                ;;
            quit|exit)
                echo "Goodbye!"
                break
                ;;
            help|?)
                echo "Commands:"
                echo "  search   - Search across all apps"
                echo "  patterns - Discover patterns"
                echo "  solutions - Find reusable solutions"
                echo "  gaps     - Analyze knowledge gaps"
                echo "  compare  - Compare two apps"
                echo "  quit     - Exit explorer"
                ;;
            *)
                echo "Unknown command: $cmd (type 'help' for commands)"
                ;;
        esac
        echo
    done
}