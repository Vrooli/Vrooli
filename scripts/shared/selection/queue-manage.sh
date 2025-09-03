#!/usr/bin/env bash
# Queue management utilities for distributed scenarios

set -euo pipefail

COMMAND="${1:-help}"
QUEUE_DIR="${2:-queue}"
ITEM="${3:-}"

# Ensure queue directories exist
init_queue() {
    mkdir -p "$QUEUE_DIR"/{pending,in-progress,completed,failed,templates}
    echo "Initialized queue at: $QUEUE_DIR"
}

# List queue status
status() {
    echo "=== Queue Status: $QUEUE_DIR ==="
    echo "Pending:     $(ls -1 "$QUEUE_DIR/pending" 2>/dev/null | wc -l) items"
    echo "In Progress: $(ls -1 "$QUEUE_DIR/in-progress" 2>/dev/null | wc -l) items"
    echo "Completed:   $(ls -1 "$QUEUE_DIR/completed" 2>/dev/null | wc -l) items"
    echo "Failed:      $(ls -1 "$QUEUE_DIR/failed" 2>/dev/null | wc -l) items"
    
    # Show in-progress item if exists
    local in_progress=$(ls -1 "$QUEUE_DIR/in-progress" 2>/dev/null | head -1)
    if [[ -n "$in_progress" ]]; then
        echo ""
        echo "Currently processing: $in_progress"
    fi
    
    # Show top pending items
    echo ""
    echo "=== Top Pending Items ==="
    ls -1 "$QUEUE_DIR/pending" 2>/dev/null | head -5 || echo "No pending items"
}

# Complete current item
complete() {
    local in_progress=$(ls -1 "$QUEUE_DIR/in-progress" 2>/dev/null | head -1)
    if [[ -z "$in_progress" ]]; then
        echo "ERROR: No item in progress" >&2
        exit 1
    fi
    
    # Move to completed
    mv "$QUEUE_DIR/in-progress/$in_progress" "$QUEUE_DIR/completed/"
    
    # Record event
    record_event "queue_complete" "$in_progress" "success"
    
    echo "Completed: $in_progress"
}

# Fail current item
fail() {
    local in_progress=$(ls -1 "$QUEUE_DIR/in-progress" 2>/dev/null | head -1)
    if [[ -z "$in_progress" ]]; then
        echo "ERROR: No item in progress" >&2
        exit 1
    fi
    
    local reason="${ITEM:-Unknown reason}"
    
    # Add failure reason to file
    if command -v yq >/dev/null 2>&1; then
        yq -i ".metadata.failure_reasons += [\"$(date -Is): $reason\"]" "$QUEUE_DIR/in-progress/$in_progress"
        yq -i ".metadata.last_attempt = \"$(date -Is)\"" "$QUEUE_DIR/in-progress/$in_progress"
        yq -i ".metadata.attempt_count += 1" "$QUEUE_DIR/in-progress/$in_progress"
    fi
    
    # Move to failed
    mv "$QUEUE_DIR/in-progress/$in_progress" "$QUEUE_DIR/failed/"
    
    # Record event
    record_event "queue_fail" "$in_progress" "$reason"
    
    echo "Failed: $in_progress"
    echo "Reason: $reason"
}

# Retry a failed item
retry() {
    if [[ -z "$ITEM" ]]; then
        echo "ERROR: Specify item to retry" >&2
        echo "Usage: $0 retry <queue-dir> <item-name>" >&2
        exit 1
    fi
    
    if [[ ! -f "$QUEUE_DIR/failed/$ITEM" ]]; then
        echo "ERROR: Item not found in failed queue: $ITEM" >&2
        exit 1
    fi
    
    # Update cooldown
    if command -v yq >/dev/null 2>&1; then
        local new_cooldown=$(date -d "+1 hour" -Is)
        yq -i ".metadata.cooldown_until = \"$new_cooldown\"" "$QUEUE_DIR/failed/$ITEM"
    fi
    
    # Move back to pending
    mv "$QUEUE_DIR/failed/$ITEM" "$QUEUE_DIR/pending/"
    
    # Record event
    record_event "queue_retry" "$ITEM" "manual"
    
    echo "Retrying: $ITEM"
    echo "Moved to pending queue with 1 hour cooldown"
}

# Clear stuck in-progress items
clear() {
    local in_progress=$(ls -1 "$QUEUE_DIR/in-progress" 2>/dev/null | head -1)
    if [[ -z "$in_progress" ]]; then
        echo "No items in progress"
        return 0
    fi
    
    # Check age of in-progress item
    local file_age=$(( $(date +%s) - $(stat -c %Y "$QUEUE_DIR/in-progress/$in_progress" 2>/dev/null || echo 0) ))
    local two_hours=7200
    
    if [[ $file_age -gt $two_hours ]]; then
        echo "Item stuck for $(( file_age / 3600 )) hours: $in_progress"
        echo -n "Move back to pending? [y/N] "
        read -r confirm
        if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
            mv "$QUEUE_DIR/in-progress/$in_progress" "$QUEUE_DIR/pending/"
            record_event "queue_clear" "$in_progress" "stuck"
            echo "Moved back to pending: $in_progress"
        fi
    else
        echo "Item in progress for $(( file_age / 60 )) minutes: $in_progress"
        echo "Not stuck yet (threshold: 2 hours)"
    fi
}

# Add new item to queue
add() {
    if [[ -z "$ITEM" ]]; then
        echo "ERROR: Specify template name" >&2
        echo "Available templates:"
        ls -1 "$QUEUE_DIR/templates" 2>/dev/null || echo "No templates found"
        exit 1
    fi
    
    local template="$QUEUE_DIR/templates/$ITEM"
    if [[ ! -f "$template" ]]; then
        # Try with .yaml extension
        template="$QUEUE_DIR/templates/${ITEM%.yaml}.yaml"
    fi
    
    if [[ ! -f "$template" ]]; then
        echo "ERROR: Template not found: $ITEM" >&2
        exit 1
    fi
    
    # Generate unique filename
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local priority="100"  # Default medium priority
    local basename="${priority}-${ITEM%.yaml}-${timestamp}.yaml"
    
    # Copy template to pending
    cp "$template" "$QUEUE_DIR/pending/$basename"
    
    # Update metadata
    if command -v yq >/dev/null 2>&1; then
        yq -i ".metadata.created_at = \"$(date -Is)\"" "$QUEUE_DIR/pending/$basename"
        yq -i ".metadata.created_by = \"human\"" "$QUEUE_DIR/pending/$basename"
        yq -i ".metadata.cooldown_until = \"$(date -Is)\"" "$QUEUE_DIR/pending/$basename"
    fi
    
    record_event "queue_add" "$basename" "manual"
    
    echo "Added to queue: $basename"
    echo "Edit the file to customize: $QUEUE_DIR/pending/$basename"
}

# Record events for tracking
record_event() {
    local event_type="$1"
    local item="$2"
    local detail="${3:-}"
    
    local events_file="$QUEUE_DIR/events.ndjson"
    local ts=$(date -Is)
    
    if command -v jq >/dev/null 2>&1; then
        jq -c -n \
            --arg type "$event_type" \
            --arg ts "$ts" \
            --arg item "$item" \
            --arg detail "$detail" \
            '{type:$type, ts:$ts, item:$item, detail:$detail}' >> "$events_file"
    else
        printf '{"type":"%s","ts":"%s","item":"%s","detail":"%s"}\n' \
            "$event_type" "$ts" "$item" "$detail" >> "$events_file"
    fi
}

# Show help
help() {
    cat <<EOF
Queue Management Utilities

Usage: $0 <command> [queue-dir] [item]

Commands:
  init              Initialize queue directories
  status            Show queue status
  complete          Mark in-progress item as completed
  fail [reason]     Mark in-progress item as failed
  retry <item>      Retry a failed item
  clear             Clear stuck in-progress items
  add <template>    Add new item from template
  help              Show this help

Queue Directory Structure:
  pending/          Items waiting to be processed
  in-progress/      Currently being worked on (max 1)
  completed/        Successfully completed items
  failed/           Failed items with logs
  templates/        Templates for new items

Examples:
  $0 status queue                    # Show queue status
  $0 complete queue                  # Complete current item
  $0 fail queue "API error"          # Fail with reason
  $0 retry queue "001-fix.yaml"      # Retry failed item
  $0 add queue improvement           # Add from template

EOF
}

# Execute command
case "$COMMAND" in
    init) init_queue ;;
    status) status ;;
    complete) complete ;;
    fail) fail ;;
    retry) retry ;;
    clear) clear ;;
    add) add ;;
    help) help ;;
    *)
        echo "Unknown command: $COMMAND" >&2
        help
        exit 1
        ;;
esac