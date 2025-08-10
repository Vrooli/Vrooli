#!/bin/bash

# Agent Generation Prompt Builder
# Builds prompts for generating agents using the Swarm-Architect GPT system

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source trash module for safe cleanup
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
AGENT_DIR="$SCRIPT_DIR"
ROUTINE_DIR="$SCRIPT_DIR/../routine"

# Configuration
PROMPT_FILE="$AGENT_DIR/prompts/agent-generation-prompt.md"
BACKLOG_FILE="$AGENT_DIR/backlog.md"
ROUTINE_REFERENCE="$ROUTINE_DIR/routine-reference.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
    cat << EOF
Agent Generation Prompt Builder

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --agent-name NAME     Generate prompt for specific agent (default: first in backlog)
    --output FILE         Save prompt to file instead of displaying
    --include-templates   Include template examples in prompt
    --routine-limit N     Limit routine table to N entries (default: 20)
    --help               Show this help message

EXAMPLES:
    # Generate prompt for first agent in backlog
    $0

    # Generate prompt for specific agent
    $0 --agent-name "Task Orchestration Coordinator"

    # Save prompt to file with templates
    $0 --output agent-prompt.md --include-templates

    # Generate with limited routine table
    $0 --routine-limit 10
EOF
}

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $*" >&2
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
    exit 1
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

# Parse command line arguments
AGENT_NAME=""
OUTPUT_FILE=""
INCLUDE_TEMPLATES=false
ROUTINE_LIMIT=20

while [[ $# -gt 0 ]]; do
    case $1 in
        --agent-name)
            AGENT_NAME="$2"
            shift 2
            ;;
        --output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --include-templates)
            INCLUDE_TEMPLATES=true
            shift
            ;;
        --routine-limit)
            ROUTINE_LIMIT="$2"
            shift 2
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Validate required files
[[ -f "$PROMPT_FILE" ]] || error "Prompt template not found: $PROMPT_FILE"
[[ -f "$BACKLOG_FILE" ]] || error "Backlog file not found: $BACKLOG_FILE"

extract_agent_from_backlog() {
    local name="$1"
    
    if [[ -z "$name" ]]; then
        # Get first unprocessed agent
        awk '
        /^### / && !/\[PROCESSED\]/ { 
            agent = substr($0, 5); 
            gsub(/^[ \t]+|[ \t]+$/, "", agent);
            while(getline && $0 !~ /^### / && $0 !~ /^---/) {
                if($0 ~ /- \*\*Goal\*\*:/) {
                    goal = substr($0, index($0, ":") + 2);
                    gsub(/^[ \t]+|[ \t]+$/, "", goal);
                }
                else if($0 ~ /- \*\*Role\*\*:/) {
                    role = substr($0, index($0, ":") + 2);
                    gsub(/^[ \t]+|[ \t]+$/, "", role);
                }
                else if($0 ~ /- \*\*Subscriptions\*\*:/) {
                    subs = substr($0, index($0, ":") + 2);
                    gsub(/^[ \t]+|[ \t]+$/, "", subs);
                }
                else if($0 ~ /- \*\*Resources\*\*:/) {
                    resources = substr($0, index($0, ":") + 2);
                    gsub(/^[ \t]+|[ \t]+$/, "", resources);
                }
            }
            printf "AGENT_NAME=\"%s\"\n", agent;
            printf "GOAL=\"%s\"\n", goal;
            printf "ROLE=\"%s\"\n", role;
            printf "SUBSCRIPTIONS=\"%s\"\n", subs;
            printf "RESOURCES=\"%s\"\n", resources;
            exit
        }' "$BACKLOG_FILE"
    else
        # Get specific agent
        awk -v target="$name" '
        /^### / { 
            current = substr($0, 5); 
            gsub(/^[ \t]+|[ \t]+$/, "", current);
            if(current == target) {
                while(getline && $0 !~ /^### / && $0 !~ /^---/) {
                    if($0 ~ /- \*\*Goal\*\*:/) {
                        goal = substr($0, index($0, ":") + 2);
                        gsub(/^[ \t]+|[ \t]+$/, "", goal);
                    }
                    else if($0 ~ /- \*\*Role\*\*:/) {
                        role = substr($0, index($0, ":") + 2);
                        gsub(/^[ \t]+|[ \t]+$/, "", role);
                    }
                    else if($0 ~ /- \*\*Subscriptions\*\*:/) {
                        subs = substr($0, index($0, ":") + 2);
                        gsub(/^[ \t]+|[ \t]+$/, "", subs);
                    }
                    else if($0 ~ /- \*\*Resources\*\*:/) {
                        resources = substr($0, index($0, ":") + 2);
                        gsub(/^[ \t]+|[ \t]+$/, "", resources);
                    }
                }
                printf "AGENT_NAME=\"%s\"\n", current;
                printf "GOAL=\"%s\"\n", goal;
                printf "ROLE=\"%s\"\n", role;
                printf "SUBSCRIPTIONS=\"%s\"\n", subs;
                printf "RESOURCES=\"%s\"\n", resources;
                exit
            }
        }' "$BACKLOG_FILE"
    fi
}

build_routine_table() {
    local limit="$1"
    
    if [[ -f "$ROUTINE_REFERENCE" ]]; then
        jq -r --argjson limit "$limit" '
        .routines[0:$limit] | .[] | 
        "| \(.name) | \(.translations[0].description // .name) |"
        ' "$ROUTINE_REFERENCE" 2>/dev/null || {
            warn "Could not read routine reference, using placeholder table"
            echo "| Task Decomposer | Break complex tasks into manageable subtasks |"
            echo "| Data Analyzer | Comprehensive data analysis and insights |"
            echo "| Error Recovery Planner | Handle and recover from system errors |"
        }
    else
        warn "Routine reference not found, using placeholder table"
        echo "| Task Decomposer | Break complex tasks into manageable subtasks |"
        echo "| Data Analyzer | Comprehensive data analysis and insights |"  
        echo "| Error Recovery Planner | Handle and recover from system errors |"
    fi
}

build_template_examples() {
    if [[ "$INCLUDE_TEMPLATES" == "true" ]]; then
        cat << 'EOF'

## Template Examples

### Coordinator Agent Example:
```json
{
  "identity": { "name": "workflow-coordinator" },
  "goal": "Orchestrate multi-step workflows across specialist agents",
  "role": "coordinator",
  "subscriptions": ["workflow.started", "task.completed"],
  "behaviors": [
    {
      "trigger": { "topic": "workflow.started" },
      "action": { "type": "routine", "label": "task-decomposer" }
    }
  ]
}
```

### Specialist Agent Example:
```json
{
  "identity": { "name": "data-analyst" },
  "goal": "Perform deep data analysis and generate insights",
  "role": "specialist", 
  "subscriptions": ["data.analysis.requested"],
  "behaviors": [
    {
      "trigger": { "topic": "data.analysis.requested" },
      "action": { "type": "routine", "label": "comprehensive-data-analyzer" }
    }
  ]
}
```

EOF
    fi
}

generate_prompt() {
    # Extract agent information
    local agent_info
    agent_info=$(extract_agent_from_backlog "$AGENT_NAME")
    
    if [[ -z "$agent_info" ]]; then
        if [[ -n "$AGENT_NAME" ]]; then
            error "Agent '$AGENT_NAME' not found in backlog"
        else
            error "No unprocessed agents found in backlog"
        fi
    fi
    
    # Parse agent information
    eval "$agent_info"
    
    log "Generating prompt for agent: $AGENT_NAME"
    log "Goal: $GOAL"
    log "Role: $ROLE"
    
    # Build routine table
    local routine_table
    routine_table=$(build_routine_table "$ROUTINE_LIMIT")
    
    # Create temporary file for safe substitution
    local temp_prompt
    temp_prompt=$(mktemp)
    
    # Manual substitution line by line to avoid sed issues
    while IFS= read -r line; do
        line="${line//\{\{SUBGOAL_TEXT\}\}/$GOAL}"
        line="${line//\{\{ROLE_NAME\}\}/$ROLE}"
        line="${line//\{\{TOPIC_LIST\}\}/$SUBSCRIPTIONS}"
        line="${line//\{\{RESOURCE_HINTS\}\}/$RESOURCES}"
        echo "$line"
    done < "$PROMPT_FILE" > "$temp_prompt"
    
    # Output the final prompt
    {
        cat "$temp_prompt"
        echo ""
        echo "$routine_table"
        build_template_examples
    }
    
    # Clean up
    trash::safe_remove "$temp_prompt" --temp
}

main() {
    log "Building agent generation prompt..."
    
    local prompt
    prompt=$(generate_prompt)
    
    if [[ -n "$OUTPUT_FILE" ]]; then
        echo "$prompt" > "$OUTPUT_FILE"
        success "Prompt saved to: $OUTPUT_FILE"
        
        echo ""
        echo "You can now:"
        echo "1. Copy the prompt from $OUTPUT_FILE"
        echo "2. Paste it into Claude (web interface or Claude Code)"
        echo "3. Review and save the generated agent JSON"
        
    else
        echo "$prompt"
        
        echo ""
        echo "===================================================="
        echo "COPY THE ABOVE PROMPT TO CLAUDE FOR AGENT GENERATION"
        echo "===================================================="
    fi
}

main "$@"