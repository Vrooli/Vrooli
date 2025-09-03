#!/usr/bin/env bash
# Queue-based selection system for distributed scenarios
# Selects highest priority item from pending queue

set -euo pipefail

QUEUE_DIR="${1:-queue}"
SCENARIO_TYPE="${2:-improvement}"

if [[ ! -d "$QUEUE_DIR/pending" ]]; then
    echo "ERROR: Queue directory not found: $QUEUE_DIR/pending" >&2
    exit 1
fi

# Function to calculate priority score from YAML
calculate_priority() {
    local file="$1"
    local score=0
    
    # Extract priority estimates if present
    if command -v yq >/dev/null 2>&1; then
        local impact=$(yq '.priority_estimates.impact // 5' "$file")
        local urgency=$(yq '.priority_estimates.urgency' "$file")
        local success_prob=$(yq '.priority_estimates.success_prob // 0.5' "$file")
        local resource_cost=$(yq '.priority_estimates.resource_cost' "$file")
        
        # Convert urgency to numeric
        case "$urgency" in
            critical) urgency=10 ;;
            high) urgency=7 ;;
            medium) urgency=5 ;;
            low) urgency=2 ;;
            *) urgency=5 ;;
        esac
        
        # Convert resource_cost to numeric
        case "$resource_cost" in
            minimal) resource_cost=1 ;;
            moderate) resource_cost=2 ;;
            heavy) resource_cost=4 ;;
            *) resource_cost=2 ;;
        esac
        
        # Calculate score: (impact * 2 + urgency * 1.5) * success_prob / resource_cost
        score=$(echo "scale=2; ($impact * 2 + $urgency * 1.5) * $success_prob / $resource_cost" | bc 2>/dev/null || echo "10")
    else
        # Fallback: use filename priority prefix
        local prefix=$(basename "$file" | cut -d- -f1)
        score=$((1000 - ${prefix:-999}))
    fi
    
    echo "$score"
}

# Function to check cooldown
check_cooldown() {
    local file="$1"
    local now=$(date +%s)
    
    if command -v yq >/dev/null 2>&1; then
        local cooldown_until=$(yq '.metadata.cooldown_until // ""' "$file")
        if [[ -n "$cooldown_until" ]]; then
            local cooldown_ts=$(date -d "$cooldown_until" +%s 2>/dev/null || echo "0")
            if [[ $cooldown_ts -gt $now ]]; then
                return 1  # Still in cooldown
            fi
        fi
    fi
    
    return 0  # Not in cooldown
}

# Find highest priority item not in cooldown
SELECTED=""
HIGHEST_SCORE=0

for file in "$QUEUE_DIR"/pending/*.yaml; do
    [[ -f "$file" ]] || continue
    
    # Skip if in cooldown
    if ! check_cooldown "$file"; then
        continue
    fi
    
    # Calculate priority score
    score=$(calculate_priority "$file")
    
    # Track highest scoring item
    if (( $(echo "$score > $HIGHEST_SCORE" | bc -l) )); then
        HIGHEST_SCORE=$score
        SELECTED="$file"
    fi
done

if [[ -z "$SELECTED" ]]; then
    echo "No items available for processing (all in cooldown or queue empty)" >&2
    exit 1
fi

# Move selected item to in-progress
IN_PROGRESS_DIR="$QUEUE_DIR/in-progress"
mkdir -p "$IN_PROGRESS_DIR"

# Check if in-progress already has an item
if [[ -n "$(ls -A "$IN_PROGRESS_DIR" 2>/dev/null)" ]]; then
    echo "ERROR: Item already in progress. Complete or clear it first." >&2
    exit 1
fi

# Move the selected item
BASENAME=$(basename "$SELECTED")
mv "$SELECTED" "$IN_PROGRESS_DIR/$BASENAME"

# Record selection event
EVENTS_FILE="${QUEUE_DIR}/events.ndjson"
TS=$(date -Is)
if command -v jq >/dev/null 2>&1; then
    EVENT=$(jq -c -n \
        --arg ts "$TS" \
        --arg item "$BASENAME" \
        --arg score "$HIGHEST_SCORE" \
        --arg type "$SCENARIO_TYPE" \
        '{type:"queue_selection", ts:$ts, item:$item, score:$score, scenario_type:$type}')
else
    EVENT=$(printf '{"type":"queue_selection","ts":"%s","item":"%s","score":%s}' "$TS" "$BASENAME" "$HIGHEST_SCORE")
fi

echo "$EVENT" >> "$EVENTS_FILE"

# Output selected item
echo "Selected: $BASENAME (score: $HIGHEST_SCORE)"
echo "Location: $IN_PROGRESS_DIR/$BASENAME"