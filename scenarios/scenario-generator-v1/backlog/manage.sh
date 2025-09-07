#!/usr/bin/env bash
# Backlog Management CLI
# Quick commands for managing the scenario backlog

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKLOG_DIR="$SCRIPT_DIR"

# Colors
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
CYAN='\033[1;36m'
NC='\033[0m'

# Helper functions
info() { echo -e "${CYAN}ℹ️  $1${NC}"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; exit 1; }

# Show backlog status
status() {
    echo -e "${CYAN}📊 Backlog Status${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    pending_count=$(ls -1 "$BACKLOG_DIR/pending"/*.yaml 2>/dev/null | wc -l)
    progress_count=$(ls -1 "$BACKLOG_DIR/in-progress"/*.yaml 2>/dev/null | wc -l)
    completed_count=$(ls -1 "$BACKLOG_DIR/completed"/*.yaml 2>/dev/null | wc -l)
    failed_count=$(ls -1 "$BACKLOG_DIR/failed"/*.yaml 2>/dev/null | wc -l)
    
    echo -e "${YELLOW}⏰ Pending:${NC}     $pending_count items"
    echo -e "${CYAN}🔄 In Progress:${NC} $progress_count items"
    echo -e "${GREEN}✅ Completed:${NC}   $completed_count items"
    echo -e "${RED}❌ Failed:${NC}      $failed_count items"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "${CYAN}Total:${NC}         $((pending_count + progress_count + completed_count + failed_count)) items"
}

# List items in a folder
list() {
    folder="${1:-pending}"
    echo -e "${CYAN}📋 Items in $folder:${NC}"
    echo
    
    for file in "$BACKLOG_DIR/$folder"/*.yaml; do
        if [[ -f "$file" ]]; then
            filename=$(basename "$file")
            # Extract name from YAML
            name=$(grep "^name:" "$file" | sed 's/name: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
            priority=$(grep "^priority:" "$file" | sed 's/priority: *//')
            revenue=$(grep "^estimated_revenue:" "$file" | sed 's/estimated_revenue: *//')
            
            # Color code by priority
            if [[ "$priority" == "high" ]]; then
                priority_color="${RED}"
            elif [[ "$priority" == "medium" ]]; then
                priority_color="${YELLOW}"
            else
                priority_color="${GREEN}"
            fi
            
            echo -e "${priority_color}●${NC} $filename"
            echo -e "  Name: $name"
            echo -e "  Priority: ${priority_color}$priority${NC}"
            echo -e "  Revenue: \$${revenue}"
            echo
        fi
    done
}

# Add new item to backlog
add() {
    echo -e "${CYAN}➕ Add New Scenario to Backlog${NC}"
    
    # Find next available number
    max_num=0
    for file in "$BACKLOG_DIR/pending"/*.yaml; do
        if [[ -f "$file" ]]; then
            filename=$(basename "$file")
            if [[ "$filename" =~ ^([0-9]{3})- ]]; then
                num="${BASH_REMATCH[1]}"
                num=$((10#$num))  # Force base-10
                if ((num > max_num)); then
                    max_num=$num
                fi
            fi
        fi
    done
    next_num=$((max_num + 1))
    
    # Get input from user
    read -p "Scenario name: " name
    read -p "Brief description: " description
    read -p "Priority (high/medium/low) [medium]: " priority
    priority="${priority:-medium}"
    read -p "Estimated revenue [25000]: " revenue
    revenue="${revenue:-25000}"
    read -p "Category [business-tool]: " category
    category="${category:-business-tool}"
    
    # Determine priority number
    if [[ "$priority" == "high" ]]; then
        file_num=$(printf "%03d" $((next_num < 100 ? next_num : 99)))
    elif [[ "$priority" == "low" ]]; then
        file_num=$(printf "%03d" $((next_num + 200)))
    else
        file_num=$(printf "%03d" $((next_num + 100)))
    fi
    
    # Create safe filename
    safe_name=$(echo "$name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g' | sed 's/--*/-/g')
    filename="${file_num}-${safe_name}.yaml"
    filepath="$BACKLOG_DIR/pending/$filename"
    
    # Create YAML file
    cat > "$filepath" << EOF
# $name
# Priority: ${priority^^}
# Estimated Value: \$$revenue

id: $safe_name
name: "$name"
description: "$description"
prompt: |
  [Add detailed prompt here]
complexity: intermediate
category: $category
priority: $priority
estimated_revenue: $revenue
tags: []
metadata:
  requested_by: "CLI User"
  requested_date: "$(date +%Y-%m-%d)"
resources_required:
  - postgres
  - redis
  - claude-code
validation_criteria:
  - "Core functionality working"
  - "Tests passing"
notes: |
  Added via CLI on $(date)
EOF
    
    success "Added to backlog: $filename"
    echo "Edit the file to add more details: $filepath"
}

# Move item between folders
move() {
    if [[ $# -lt 2 ]]; then
        error "Usage: $0 move <filename> <to-folder>"
    fi
    
    filename="$1"
    to_folder="$2"
    
    # Find the file
    found=""
    for folder in pending in-progress completed failed; do
        if [[ -f "$BACKLOG_DIR/$folder/$filename" ]]; then
            found="$folder"
            break
        fi
    done
    
    if [[ -z "$found" ]]; then
        error "File not found: $filename"
    fi
    
    if [[ "$found" == "$to_folder" ]]; then
        warn "File is already in $to_folder"
        return
    fi
    
    # Move the file
    mv "$BACKLOG_DIR/$found/$filename" "$BACKLOG_DIR/$to_folder/$filename"
    success "Moved $filename from $found to $to_folder"
}

# Delete item from backlog
delete() {
    if [[ $# -lt 1 ]]; then
        error "Usage: $0 delete <filename>"
    fi
    
    filename="$1"
    
    # Find and delete
    for folder in pending in-progress completed failed; do
        filepath="$BACKLOG_DIR/$folder/$filename"
        if [[ -f "$filepath" ]]; then
            read -p "Delete $filename? (y/N) " confirm
            if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
                rm "$filepath"
                success "Deleted: $filename"
            else
                info "Cancelled"
            fi
            return
        fi
    done
    
    error "File not found: $filename"
}

# Show help
show_help() {
    cat << EOF
${CYAN}📋 Scenario Generator Backlog Manager${NC}

Usage: $0 <command> [arguments]

Commands:
  ${GREEN}status${NC}              Show backlog statistics
  ${GREEN}list [folder]${NC}       List items in folder (default: pending)
  ${GREEN}add${NC}                 Add new scenario to backlog
  ${GREEN}move <file> <to>${NC}    Move item between folders
  ${GREEN}delete <file>${NC}       Delete item from backlog
  ${GREEN}help${NC}                Show this help message

Folders:
  pending             New scenarios waiting to be processed
  in-progress         Currently being generated
  completed           Successfully generated scenarios
  failed              Failed generation attempts

Examples:
  $0 status                           # Show backlog overview
  $0 list pending                     # List pending items
  $0 add                              # Interactive add to backlog
  $0 move 001-invoice.yaml in-progress # Move to in-progress
  $0 delete 003-old-idea.yaml         # Delete an item

EOF
}

# Main command dispatcher
case "${1:-help}" in
    status)
        status
        ;;
    list)
        list "${2:-pending}"
        ;;
    add)
        add
        ;;
    move)
        move "$2" "$3"
        ;;
    delete)
        delete "$2"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        error "Unknown command: $1. Use '$0 help' for usage."
        ;;
esac