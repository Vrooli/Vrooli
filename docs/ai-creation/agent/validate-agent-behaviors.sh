#!/bin/bash

# Agent Behavior Validation Script
# Validates agent specifications and routine connections

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_DIR="$SCRIPT_DIR"
ROUTINE_DIR="$SCRIPT_DIR/../routine"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
    cat << EOF
Agent Behavior Validation

USAGE:
    $0 [OPTIONS] [FILES...]

OPTIONS:
    --routine-check      Validate routine label references (default)
    --event-check        Validate event subscription format
    --inputmap-check     Validate inputMap field paths
    --all-checks         Run all validation checks
    --verbose           Show detailed output
    --directory DIR     Validate all agents in directory
    --help              Show this help message

EXAMPLES:
    # Validate specific agent file
    $0 staged/coordinator/workflow-coordinator.json

    # Validate all agents in directory
    $0 --directory staged/

    # Run all validation checks
    $0 --all-checks staged/coordinator/*.json
EOF
}

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $*" >&2
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

# Valid events from socketEvents.ts
VALID_EVENTS=(
    "swarm/started"
    "swarm/state/changed"
    "swarm/resource/updated"
    "swarm/config/updated"
    "swarm/team/updated"
    "swarm/goal/created"
    "swarm/goal/updated"
    "swarm/goal/completed"
    "swarm/goal/failed"
    "run/started"
    "run/completed"
    "run/failed"
    "run/task/ready"
    "run/decision/requested"
    "step/started"
    "step/completed"
    "step/failed"
    "safety/pre_action"
    "safety/post_action"
)

# Event payload mappings (what fields are available in event.data)
declare -A EVENT_PAYLOADS
EVENT_PAYLOADS["swarm/goal/created"]="chatId,swarmId,goalId,description,priority"
EVENT_PAYLOADS["run/completed"]="runId,outputs,duration,message"
EVENT_PAYLOADS["run/failed"]="runId,error,duration,retryable"
EVENT_PAYLOADS["step/completed"]="runId,routineId,stepId,outputs,success,metrics,error,context"
EVENT_PAYLOADS["safety/pre_action"]="actionType,actorId,actorType,contextId,contextType,actionDetails,actionId"
EVENT_PAYLOADS["safety/post_action"]="actionId,actionType,actorId,actorType,contextId,contextType,actionName,result,success,error,duration,creditsUsed"

validate_event_subscriptions() {
    local file="$1"
    local errors=0
    
    log "Validating event subscriptions in $file"
    
    # Extract subscriptions from JSON
    local subscriptions
    subscriptions=$(jq -r '.subscriptions[]? // empty' "$file" 2>/dev/null || echo "")
    
    if [[ -z "$subscriptions" ]]; then
        error "No subscriptions found in $file"
        return 1
    fi
    
    while IFS= read -r event; do
        [[ -z "$event" ]] && continue
        
        local valid=false
        for valid_event in "${VALID_EVENTS[@]}"; do
            if [[ "$event" == "$valid_event" ]]; then
                valid=true
                break
            fi
        done
        
        if [[ "$valid" == false ]]; then
            error "Invalid event subscription: '$event' in $file"
            error "Must be one of: ${VALID_EVENTS[*]}"
            ((errors++))
        fi
    done <<< "$subscriptions"
    
    return $errors
}

validate_inputmap_paths() {
    local file="$1"
    local errors=0
    
    log "Validating inputMap field paths in $file"
    
    # Extract all inputMap entries
    local inputmaps
    inputmaps=$(jq -r '.behaviours[]?.action.inputMap? // empty | to_entries[] | "\(.key)=\(.value)"' "$file" 2>/dev/null || echo "")
    
    while IFS= read -r mapping; do
        [[ -z "$mapping" ]] && continue
        
        local key="${mapping%%=*}"
        local path="${mapping#*=}"
        
        # Check for common mistakes
        if [[ "$path" =~ event\.payload\. ]]; then
            error "Invalid field path '$path' in $file"
            error "Should use 'event.data.*' not 'event.payload.*'"
            ((errors++))
        fi
        
        # Validate path format
        if [[ ! "$path" =~ ^event\.data\. ]] && [[ ! "$path" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
            warn "Unusual field path '$path' in $file - should typically be 'event.data.fieldName'"
        fi
        
    done <<< "$inputmaps"
    
    return $errors
}

validate_routine_references() {
    local file="$1"
    local errors=0
    
    log "Validating routine references in $file"
    
    # Extract routine names from behaviors
    local routine_names
    routine_names=$(jq -r '.behaviours[]? | select(.action.type == "routine") | .action.label' "$file" 2>/dev/null || echo "")
    
    if [[ -z "$routine_names" ]]; then
        log "No routine references found in $file"
        return 0
    fi
    
    # Check if routine reference file exists
    local routine_ref="$ROUTINE_DIR/routine-reference.json"
    if [[ ! -f "$routine_ref" ]]; then
        warn "Routine reference file not found: $routine_ref"
        warn "Cannot validate routine names - please run routine reference generator"
        return 0
    fi
    
    # Check if each routine name exists
    while IFS= read -r routine_name; do
        [[ -z "$routine_name" ]] && continue
        
        local found
        found=$(jq -r --arg name "$routine_name" '.routines[]? | select(.name == $name) | .id' "$routine_ref" 2>/dev/null || echo "")
        
        if [[ -z "$found" ]]; then
            error "Routine name '$routine_name' not found in routine reference"
            error "Available routines: $(jq -r '.routines[]?.name' "$routine_ref" 2>/dev/null | head -5 | tr '\n' ' ')..."
            ((errors++))
        else
            success "Routine name '$routine_name' -> ID $found"
        fi
    done <<< "$routine_names"
    
    return $errors
}

validate_agent_structure() {
    local file="$1"
    local errors=0
    
    log "Validating agent structure in $file"
    
    # Check if file is valid JSON
    if ! jq . "$file" >/dev/null 2>&1; then
        error "Invalid JSON in $file"
        return 1
    fi
    
    # Check required fields
    local required_fields=("identity.name" "goal" "role" "subscriptions" "behaviours")
    for field in "${required_fields[@]}"; do
        if ! jq -e ".$field" "$file" >/dev/null 2>&1; then
            error "Missing required field: $field in $file"
            ((errors++))
        fi
    done
    
    # Check behavior structure
    local behavior_count
    behavior_count=$(jq '.behaviours | length' "$file" 2>/dev/null || echo "0")
    
    if [[ "$behavior_count" -eq 0 ]]; then
        error "Agent must have at least one behavior in $file"
        ((errors++))
    elif [[ "$behavior_count" -gt 3 ]]; then
        warn "Agent has $behavior_count behaviors (recommended max: 3) in $file"
    fi
    
    return $errors
}

validate_file() {
    local file="$1"
    local checks="$2"
    local total_errors=0
    
    echo ""
    log "Validating agent file: $file"
    
    if [[ ! -f "$file" ]]; then
        error "File not found: $file"
        return 1
    fi
    
    # Run structure validation first
    if ! validate_agent_structure "$file"; then
        error "Structure validation failed for $file"
        return 1
    fi
    
    # Run requested checks
    if [[ "$checks" == *"routine"* ]] || [[ "$checks" == "all" ]]; then
        validate_routine_references "$file"
        ((total_errors += $?))
    fi
    
    if [[ "$checks" == *"event"* ]] || [[ "$checks" == "all" ]]; then
        validate_event_subscriptions "$file"
        ((total_errors += $?))
    fi
    
    if [[ "$checks" == *"inputmap"* ]] || [[ "$checks" == "all" ]]; then
        validate_inputmap_paths "$file"
        ((total_errors += $?))
    fi
    
    if [[ $total_errors -eq 0 ]]; then
        success "All validations passed for $file"
    else
        error "$total_errors validation errors found in $file"
    fi
    
    return $total_errors
}

# Parse command line arguments
CHECKS="routine"
VERBOSE=false
DIRECTORY=""
FILES=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --routine-check)
            CHECKS="routine"
            shift
            ;;
        --event-check)
            CHECKS="event"
            shift
            ;;
        --inputmap-check)
            CHECKS="inputmap"
            shift
            ;;
        --all-checks)
            CHECKS="all"
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --directory)
            DIRECTORY="$2"
            shift 2
            ;;
        --help)
            usage
            exit 0
            ;;
        -*)
            error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            FILES+=("$1")
            shift
            ;;
    esac
done

# Collect files to validate
if [[ -n "$DIRECTORY" ]]; then
    if [[ ! -d "$DIRECTORY" ]]; then
        error "Directory not found: $DIRECTORY"
        exit 1
    fi
    mapfile -t FILES < <(find "$DIRECTORY" -name "*.json" -type f)
fi

if [[ ${#FILES[@]} -eq 0 ]]; then
    error "No files specified for validation"
    usage
    exit 1
fi

# Main validation loop
total_files=0
failed_files=0

for file in "${FILES[@]}"; do
    ((total_files++))
    if ! validate_file "$file" "$CHECKS"; then
        ((failed_files++))
    fi
done

echo ""
echo "======================================"
echo "Validation Summary:"
echo "  Total files: $total_files"
echo "  Failed files: $failed_files"
echo "  Success rate: $(( (total_files - failed_files) * 100 / total_files ))%"
echo "======================================"

if [[ $failed_files -gt 0 ]]; then
    error "Validation failed for $failed_files out of $total_files files"
    exit 1
else
    success "All validations passed!"
    exit 0
fi