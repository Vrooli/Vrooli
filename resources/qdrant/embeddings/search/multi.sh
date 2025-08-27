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
    
    # Silent operation for JSON output
    local search_start=$(date +%s%3N)
    
    # Find all app identities (with debug output)
    local app_ids=()
    local seen_apps=()  # Track seen app_ids to prevent duplicates
    
    # Start search from current working directory, not HOME (too broad)
    local search_base="${SEARCH_BASE:-$(pwd)}"
    # Debug: Looking for app identities
    
    # Find app-identity.json files only in .vrooli directories
    # Exclude test fixtures and backup directories
    local identity_files=$(find "$search_base" -path "*/.vrooli/app-identity.json" -type f 2>/dev/null \
        | grep -v "/test/" \
        | grep -v "/backup/" \
        | grep -v "/.git/")
    # Debug: Found identity files
    
    while IFS= read -r identity_file; do
        if [[ -f "$identity_file" ]]; then
            # Debug: Processing identity file
            local app_id=$(jq -r '.app_id // empty' "$identity_file" 2>/dev/null)
            # Debug: Extracted app_id
            
            # Check if we've already seen this app_id (deduplication)
            local already_seen=false
            for seen_id in "${seen_apps[@]}"; do
                if [[ "$seen_id" == "$app_id" ]]; then
                    already_seen=true
                    # Debug: Skipping duplicate app_id
                    break
                fi
            done
            
            if [[ -n "$app_id" && "$already_seen" == "false" ]]; then
                app_ids+=("$app_id")
                seen_apps+=("$app_id")
            fi
        fi
    done <<< "$identity_files"
    
    # Fallback: search for collections by pattern
    if [[ ${#app_ids[@]} -eq 0 ]]; then
        # Debug: No app identities found, trying fallback
        log::debug "No app identities found, searching by collection pattern"
        local discovered_apps=$(qdrant::search::discover_apps)
        # Debug: Fallback discovered apps
        app_ids=($(echo "$discovered_apps"))
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
    
    # Debug: Found apps to search
    log::debug "Found ${#app_ids[@]} apps to search"
    
    # Search each app in parallel (if possible)
    local all_results="[]"
    local apps_with_results=0
    
    for app_id in "${app_ids[@]}"; do
        # Debug: Starting search for app
        log::debug "Searching app: $app_id"
        
        # Search single app
        # Debug: Calling single_app search
        local app_results=$(qdrant::search::single_app \
            "$query" \
            "$app_id" \
            "$type" \
            "$limit_per_app" \
            "$min_score" 2>/dev/null)
        
        # Debug: single_app returned results
        
        if [[ -n "$app_results" ]]; then
            # Debug: Processing app results
            local result_count=$(echo "$app_results" | jq -r '.result_count // 0' 2>/dev/null)
            # Debug: Extracted result count
            if [[ "$result_count" -gt 0 ]]; then
                # Debug: Result count > 0
                ((apps_with_results++))
                
                # Extract just the results array
                local results_array=$(echo "$app_results" | jq -c '.results // []' 2>/dev/null)
                # Debug: Extracted results array
                
                # Merge with all results
                all_results=$(echo "$all_results $results_array" | jq -s 'add' 2>/dev/null)
                # Debug: After merge
            else
                # Debug: Result count is 0 or invalid
                :
            fi
        else
            # Debug: app_results is empty
            :
        fi
    done
    
    # Sort all results by score and apply global limit
    local total_limit=$((limit_per_app * ${#app_ids[@]}))
    if [[ $total_limit -gt $DEFAULT_LIMIT ]]; then
        total_limit=$DEFAULT_LIMIT
    fi
    
    # Debug: Before sorting
    # Debug: Applying limit
    
    all_results=$(echo "$all_results" | jq --argjson limit "$total_limit" \
        'sort_by(-.score) | .[0:$limit]' 2>/dev/null)
    
    # Debug: After sorting
    
    # Calculate search time
    local search_end=$(date +%s%3N)
    local search_time=$((search_end - search_start))
    
    # Build final response (with debug)
    # Debug: Building final response
    # Debug: all_results length
    # Debug: apps_with_results
    # Debug: app_ids count
    
    jq -n \
        --arg query "$query" \
        --arg type "$type" \
        --argjson results "$all_results" \
        --arg apps_searched "${#app_ids[@]}" \
        --arg apps_with_results "$apps_with_results" \
        --arg total_results "$(echo "$all_results" | jq 'length' 2>/dev/null || echo '0')" \
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
        }' 2>/dev/null
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
        local already_exists=false
        for existing_id in "${app_ids[@]}"; do
            if [[ "$existing_id" == "$app_id" ]]; then
                already_exists=true
                break
            fi
        done
        if [[ "$already_exists" == "false" ]]; then
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
    
    # Search all apps
    local results=$(qdrant::search::all_apps "$query" "all" "10" "$threshold")
    
    if [[ -z "$results" ]] || [[ "$(echo "$results" | jq -r '.total_results // 0' 2>/dev/null || echo "0")" -eq 0 ]]; then
        echo '{"query": "'"$query"'", "patterns": [], "message": "No patterns found"}'
        return 0
    fi
    
    # Analyze patterns and return JSON
    echo "$results" | jq --arg query "$query" --arg threshold "$threshold" '{
        query: $query,
        threshold: ($threshold | tonumber),
        patterns: {
            by_type: (.results | group_by(.type) | map({
                type: .[0].type,
                count: length,
                examples: [.[:3] | .[] | {app_id, score, content: (.content | split("\n")[0] | .[0:100])}]
            })),
            by_app: (.results | group_by(.app_id) | map({
                app_id: .[0].app_id,
                count: length,
                types: ([.[].type] | unique)
            })),
            top_matches: (.results | map(select(.score > ($threshold|tonumber))) | .[:5])
        },
        total_patterns: .total_results,
        timestamp: now | strftime("%Y-%m-%dT%H:%M:%SZ")
    }'
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
    
    # Search with focus on preferred type
    local primary_results=$(qdrant::search::all_apps "$problem" "$preferred_type" "5" "0.6")
    local secondary_results=$(qdrant::search::all_apps "$problem" "all" "3" "0.7")
    
    # Combine, deduplicate and format as JSON
    printf '%s\n%s\n' "$primary_results" "$secondary_results" | jq -s --arg problem "$problem" --arg pref_type "$preferred_type" '{
        problem: $problem,
        preferred_type: $pref_type,
        solutions: (
            .[0].results + .[1].results | 
            unique_by(.id) | 
            sort_by(-.score) | 
            .[:10] | 
            map({
                type: .type,
                app_id: .app_id,
                score: .score,
                content: .content,
                metadata: .metadata,
                relevance: (
                    if .score >= 0.9 then "high"
                    elif .score >= 0.7 then "medium"
                    else "low" end
                )
            })
        ),
        solution_count: (.[0].results + .[1].results | unique_by(.id) | length),
        timestamp: now | strftime("%Y-%m-%dT%H:%M:%SZ")
    }'
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
        echo '{"error": "Query and two app IDs required"}'
        return 1
    fi
    
    # Search both apps
    local results1=$(qdrant::search::single_app "$query" "$app1" "all" "5")
    local results2=$(qdrant::search::single_app "$query" "$app2" "all" "5")
    
    # Combine results and create comparison JSON
    jq -n --arg query "$query" --arg app1 "$app1" --arg app2 "$app2" \
        --argjson r1 "$results1" --argjson r2 "$results2" '{
        query: $query,
        comparison: {
            app1: {
                id: $app1,
                result_count: $r1.result_count,
                best_score: ($r1.results[0].score // 0),
                types: [$r1.results[].type] | unique,
                top_results: $r1.results[:3]
            },
            app2: {
                id: $app2,
                result_count: $r2.result_count,
                best_score: ($r2.results[0].score // 0),
                types: [$r2.results[].type] | unique,
                top_results: $r2.results[:3]
            }
        },
        analysis: {
            better_coverage: (if $r1.result_count > $r2.result_count then $app1 
                             elif $r2.result_count > $r1.result_count then $app2 
                             else "equal" end),
            higher_relevance: (if ($r1.results[0].score // 0) > ($r2.results[0].score // 0) then $app1 
                              elif ($r2.results[0].score // 0) > ($r1.results[0].score // 0) then $app2 
                              else "equal" end),
            unique_in_app1: ([$r1.results[].type] | unique) - ([$r2.results[].type] | unique),
            unique_in_app2: ([$r2.results[].type] | unique) - ([$r1.results[].type] | unique),
            common_types: ([$r1.results[].type] | unique) - (([$r1.results[].type] | unique) - ([$r2.results[].type] | unique))
        },
        timestamp: now | strftime("%Y-%m-%dT%H:%M:%SZ")
    }'
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
    
    # Search for the topic
    local results=$(qdrant::search::all_apps "$topic" "all" "10" "0.3")
    local total=$(echo "$results" | jq -r '.total_results // 0')
    
    # Analyze coverage and return JSON
    echo "$results" | jq --arg topic "$topic" '{
        topic: $topic,
        total_results: .total_results,
        coverage: {
            by_type: (.results | group_by(.type) | map({
                type: .[0].type,
                count: length,
                items: [.[:3] | .[] | {app_id, score, content: (.content | split("\n")[0] | .[0:100])}]
            })),
            by_app: {
                apps_with_content: .apps_with_results,
                total_apps_searched: .apps_searched,
                coverage_percentage: (if .apps_searched > 0 then (.apps_with_results / .apps_searched * 100) else 0 end)
            }
        },
        gaps: {
            missing_types: (["workflows", "scenarios", "knowledge", "code", "resources"] - ([.results[].type] | unique)),
            has_all_types: (([.results[].type] | unique | length) == 5),
            recommendations: (
                if .total_results == 0 then
                    ["No knowledge found - create initial documentation", "Add workflows for this topic", "Create code examples"]
                elif ([.results[].type] | unique | length) < 3 then
                    ["Limited coverage - expand to more content types", "Add more examples and patterns"]
                else
                    ["Good coverage - consider adding advanced patterns", "Document edge cases"]
                end
            )
        },
        timestamp: now | strftime("%Y-%m-%dT%H:%M:%SZ")
    }'
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
    
    # Silent operation for JSON output
    local results=$(qdrant::search::all_apps "$query" "all" "20" "0.4")
    # Debug: Report received results
    
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
            # Safely extract values with fallbacks
            local total_results=$(echo "$results" | jq -r '.total_results // "0"' 2>/dev/null || echo "0")
            local apps_searched=$(echo "$results" | jq -r '.apps_searched // "0"' 2>/dev/null || echo "0")
            local apps_with_results=$(echo "$results" | jq -r '.apps_with_results // "0"' 2>/dev/null || echo "0")
            local search_time=$(echo "$results" | jq -r '.search_time_ms // "0"' 2>/dev/null || echo "0")
            
            echo "  Total Results: $total_results"
            echo "  Apps Searched: $apps_searched"
            echo "  Apps with Results: $apps_with_results"
            echo "  Search Time: ${search_time}ms"
            echo
            echo "Top Results:"
            
            # Check if we have valid results before trying to display them
            if [[ -n "$results" ]] && echo "$results" | jq -e '.results' >/dev/null 2>&1; then
                local result_count=$(echo "$results" | jq -r '.results | length' 2>/dev/null || echo "0")
                if [[ "$result_count" -gt 0 ]]; then
                    echo "$results" | jq -r '.results[0:5][] | 
                        "\n#\(1 + (input_line_number - 1)). [\(.score|tostring[0:4])] \(.type) from \(.app_id)" +
                        "\n   \(.content | split("\n")[0] | .[0:150])" +
                        (if .metadata.source_file then "\n   File: \(.metadata.source_file)" else "" end)' 2>/dev/null || echo "  (No results to display)"
                else
                    echo "  No results found"
                fi
            else
                echo "  No results found"
            fi
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