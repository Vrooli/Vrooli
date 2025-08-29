#!/usr/bin/env bash
################################################################################
# Setup Condition: Resources Population Check
# 
# Checks if resources have been populated with application data
# Verifies resources are not just running but have app-specific content
#
# Input: JSON object with "populated" boolean or "resources" array
# Returns: 0 if setup needed (resources not populated), 1 if populated
################################################################################

set -euo pipefail

# Get check configuration from argument
CHECK_CONFIG="${1:-{}}"
APP_ROOT="${APP_ROOT:-$(pwd)}"

# Check if we're just checking for a population marker file
POPULATED=$(echo "$CHECK_CONFIG" | jq -r '.populated // false' 2>/dev/null)

if [[ "$POPULATED" == "true" ]]; then
    # Simple check - look for population marker
    if [[ -f "$APP_ROOT/data/.resources-populated" ]]; then
        # Resources marked as populated - no setup needed
        exit 1
    else
        echo "[DEBUG] Resources not populated (marker file missing)" >&2
        # Resources not populated - setup needed
        exit 0
    fi
fi

# Check specific resources if listed
RESOURCES=$(echo "$CHECK_CONFIG" | jq -r '.resources[]?' 2>/dev/null || echo "")

if [[ -z "$RESOURCES" ]]; then
    # No specific resources to check - look for generic marker
    if [[ -f "$APP_ROOT/data/.resources-populated" ]]; then
        exit 1
    else
        echo "[DEBUG] No resource population marker found" >&2
        exit 0
    fi
fi

# Check each specified resource
MISSING_COUNT=0
while IFS= read -r resource; do
    if [[ -z "$resource" ]]; then
        continue
    fi
    
    # Check if resource has app-specific data
    # This could be customized per resource type
    case "$resource" in
        postgres|postgresql)
            # Check for database existence (would need actual DB query)
            # For now, check if populate script was run
            if [[ ! -f "$APP_ROOT/data/.postgres-populated" ]]; then
                echo "[DEBUG] PostgreSQL not populated for app" >&2
                ((MISSING_COUNT++))
            fi
            ;;
        redis)
            # Check for Redis keys (would need actual Redis query)
            if [[ ! -f "$APP_ROOT/data/.redis-populated" ]]; then
                echo "[DEBUG] Redis not populated for app" >&2
                ((MISSING_COUNT++))
            fi
            ;;
        windmill)
            # Check for Windmill workspace
            if [[ ! -f "$APP_ROOT/data/.windmill-populated" ]]; then
                echo "[DEBUG] Windmill workspace not created" >&2
                ((MISSING_COUNT++))
            fi
            ;;
        n8n)
            # Check for n8n workflows
            if [[ ! -f "$APP_ROOT/data/.n8n-populated" ]]; then
                echo "[DEBUG] n8n workflows not loaded" >&2
                ((MISSING_COUNT++))
            fi
            ;;
        *)
            # Generic resource check
            if [[ ! -f "$APP_ROOT/data/.${resource}-populated" ]]; then
                echo "[DEBUG] Resource $resource not populated" >&2
                ((MISSING_COUNT++))
            fi
            ;;
    esac
done <<< "$RESOURCES"

# Return 0 if any resources are not populated (setup needed)
[[ $MISSING_COUNT -eq 0 ]] && exit 1 || exit 0