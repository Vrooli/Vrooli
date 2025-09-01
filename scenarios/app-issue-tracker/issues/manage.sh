#!/usr/bin/env bash
# Issue Tracker Management CLI  
# Quick commands for managing file-based issues

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISSUES_DIR="$SCRIPT_DIR"

# Colors
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
CYAN='\033[1;36m'
BLUE='\033[1;34m'
NC='\033[0m'

# Helper functions
info() { echo -e "${CYAN}ℹ️  $1${NC}"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; exit 1; }

# Show issue status overview
status() {
    echo -e "${CYAN}🐛 Issue Tracker Status${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    open_count=$(ls -1 "$ISSUES_DIR/open"/*.yaml 2>/dev/null | wc -l)
    investigating_count=$(ls -1 "$ISSUES_DIR/investigating"/*.yaml 2>/dev/null | wc -l)
    progress_count=$(ls -1 "$ISSUES_DIR/in-progress"/*.yaml 2>/dev/null | wc -l)
    fixed_count=$(ls -1 "$ISSUES_DIR/fixed"/*.yaml 2>/dev/null | wc -l)
    closed_count=$(ls -1 "$ISSUES_DIR/closed"/*.yaml 2>/dev/null | wc -l)
    failed_count=$(ls -1 "$ISSUES_DIR/failed"/*.yaml 2>/dev/null | wc -l)
    
    # Count by priority
    critical_count=$(find "$ISSUES_DIR" -name "0[0-9][0-9]-*.yaml" 2>/dev/null | wc -l)
    high_count=$(find "$ISSUES_DIR" -name "1[0-9][0-9]-*.yaml" 2>/dev/null | wc -l)
    medium_count=$(find "$ISSUES_DIR" -name "[2-4][0-9][0-9]-*.yaml" 2>/dev/null | wc -l)
    low_count=$(find "$ISSUES_DIR" -name "[5-9][0-9][0-9]-*.yaml" 2>/dev/null | wc -l)
    
    echo -e "${RED}🚨 Open:${NC}          $open_count issues"
    echo -e "${BLUE}🔍 Investigating:${NC} $investigating_count issues"
    echo -e "${YELLOW}⚙️  In Progress:${NC}   $progress_count issues"
    echo -e "${GREEN}✅ Fixed:${NC}         $fixed_count issues"
    echo -e "${GREEN}📝 Closed:${NC}        $closed_count issues"
    echo -e "${RED}❌ Failed:${NC}        $failed_count issues"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "${CYAN}Total:${NC}           $((open_count + investigating_count + progress_count + fixed_count + closed_count + failed_count)) issues"
    echo ""
    echo -e "${CYAN}📊 By Priority:${NC}"
    echo -e "${RED}  Critical:${NC}       $critical_count issues (001-099)"
    echo -e "${YELLOW}  High:${NC}           $high_count issues (100-199)"
    echo -e "${BLUE}  Medium:${NC}         $medium_count issues (200-499)"
    echo -e "${GREEN}  Low:${NC}            $low_count issues (500-999)"
}

# List issues in a folder
list() {
    local folder="${1:-open}"
    echo -e "${CYAN}📋 Issues in $folder:${NC}"
    echo
    
    if [[ ! -d "$ISSUES_DIR/$folder" ]]; then
        error "Folder does not exist: $folder"
    fi
    
    local count=0
    for file in "$ISSUES_DIR/$folder"/*.yaml; do
        if [[ -f "$file" ]]; then
            filename=$(basename "$file")
            
            # Extract key info from YAML
            title=$(grep "^title:" "$file" | sed 's/title: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
            priority=$(grep "^priority:" "$file" | sed 's/priority: *//')
            app_id=$(grep "^app_id:" "$file" | sed 's/app_id: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
            type=$(grep "^type:" "$file" | sed 's/type: *//')
            created=$(grep "created_at:" "$file" | sed 's/.*created_at: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
            
            # Color code by priority
            case "$priority" in
                critical) priority_color="${RED}" ;;
                high) priority_color="${YELLOW}" ;;  
                medium) priority_color="${BLUE}" ;;
                *) priority_color="${GREEN}" ;;
            esac
            
            # Color code by type
            case "$type" in
                bug) type_icon="🐛" ;;
                feature) type_icon="✨" ;;
                performance) type_icon="⚡" ;;
                security) type_icon="🔒" ;;
                *) type_icon="📝" ;;
            esac
            
            echo -e "${priority_color}●${NC} $filename"
            echo -e "  $type_icon ${title}"
            echo -e "  App: ${app_id} | Priority: ${priority_color}$priority${NC} | Created: ${created}"
            echo
            ((count++))
        fi
    done
    
    if [[ $count -eq 0 ]]; then
        info "No issues found in $folder/"
    fi
}

# Add new issue to tracker
add() {
    echo -e "${CYAN}➕ Add New Issue${NC}"
    
    # Find next available number for priority level
    local priority_input
    read -p "Priority (critical/high/medium/low) [medium]: " priority_input
    priority="${priority_input:-medium}"
    
    local start_num end_num
    case "$priority" in
        critical) start_num=1; end_num=99 ;;
        high) start_num=100; end_num=199 ;;
        medium) start_num=200; end_num=499 ;;
        low) start_num=500; end_num=999 ;;
        *) error "Invalid priority: $priority" ;;
    esac
    
    # Find next available number in range
    local next_num=$start_num
    for file in "$ISSUES_DIR/open"/*.yaml "$ISSUES_DIR/investigating"/*.yaml "$ISSUES_DIR/in-progress"/*.yaml; do
        if [[ -f "$file" ]]; then
            filename=$(basename "$file")
            if [[ "$filename" =~ ^([0-9]{3})- ]]; then
                num="${BASH_REMATCH[1]}"
                num=$((10#$num))
                if ((num >= start_num && num <= end_num && num >= next_num)); then
                    next_num=$((num + 1))
                fi
            fi
        fi
    done
    
    if ((next_num > end_num)); then
        warn "Priority range full, using highest available number"
        next_num=$end_num
    fi
    
    # Get issue details
    read -p "Issue title: " title
    [[ -z "$title" ]] && error "Title is required"
    
    read -p "Brief description: " description
    [[ -z "$description" ]] && description="$title"
    
    read -p "Issue type (bug/feature/improvement/performance/security) [bug]: " type
    type="${type:-bug}"
    
    read -p "App/scenario name [vrooli-core]: " app_id
    app_id="${app_id:-vrooli-core}"
    
    read -p "Reporter name: " reporter_name
    read -p "Reporter email: " reporter_email
    
    # Optional error context for bugs
    error_message=""
    stack_trace=""
    if [[ "$type" == "bug" ]]; then
        read -p "Error message (optional): " error_message
        echo "Stack trace (optional, press Ctrl+D when done):"
        stack_trace=$(cat)
    fi
    
    # Create safe filename
    safe_title=$(echo "$title" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g' | sed 's/--*/-/g' | sed 's/^-//' | sed 's/-$//')
    filename=$(printf "%03d-%s.yaml" "$next_num" "$safe_title")
    filepath="$ISSUES_DIR/open/$filename"
    
    # Generate unique ID
    issue_id="issue-$(date +%Y%m%d-%H%M%S)-$(echo "$safe_title" | head -c 20)"
    
    # Select appropriate template
    local template_file
    case "$type" in
        bug) template_file="$ISSUES_DIR/templates/bug-template.yaml" ;;
        feature) template_file="$ISSUES_DIR/templates/feature-template.yaml" ;;
        performance) template_file="$ISSUES_DIR/templates/performance-template.yaml" ;;
        *) template_file="$ISSUES_DIR/templates/bug-template.yaml" ;;  # Default to bug
    esac
    
    if [[ ! -f "$template_file" ]]; then
        error "Template not found: $template_file"
    fi
    
    # Create issue file from template
    cp "$template_file" "$filepath"
    
    # Update with actual values
    sed -i "s/unique-.*-id/$issue_id/g" "$filepath"
    sed -i "s/Brief descriptive title.*\"/$title\"/g" "$filepath"
    sed -i "s/Detailed description.*\"/$description\"/g" "$filepath"
    sed -i "s/priority: medium/priority: $priority/g" "$filepath"
    sed -i "s/app_id: \"app-name\"/app_id: \"$app_id\"/g" "$filepath"
    sed -i "s/type: bug/type: $type/g" "$filepath"
    
    # Update reporter info
    if [[ -n "$reporter_name" ]]; then
        sed -i "s/Reporter Name/$reporter_name/g" "$filepath"
    fi
    if [[ -n "$reporter_email" ]]; then
        sed -i "s/reporter@domain.com/$reporter_email/g" "$filepath"
    fi
    
    # Update timestamps
    timestamp=$(date -Iseconds)
    sed -i "s/YYYY-MM-DDTHH:MM:SSZ/$timestamp/g" "$filepath"
    
    # Add error context if provided
    if [[ -n "$error_message" ]]; then
        sed -i "s/Exact error message from logs/$error_message/g" "$filepath"
    fi
    if [[ -n "$stack_trace" ]]; then
        # Escape and insert stack trace
        escaped_trace=$(echo "$stack_trace" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g')
        sed -i "/stack_trace: |/,/affected_files:/{ /stack_trace: |/{ r /dev/stdin
        } ; /affected_files:/!d ; }" "$filepath" <<< "    $escaped_trace"
    fi
    
    success "Issue created: $filename"
    echo "Location: $filepath"
    echo "Edit for additional details: $filepath"
}

# Move issue between folders
move() {
    if [[ $# -lt 2 ]]; then
        error "Usage: $0 move <filename> <to-folder>"
    fi
    
    local filename="$1"
    local to_folder="$2"
    
    # Validate destination folder
    if [[ ! -d "$ISSUES_DIR/$to_folder" ]]; then
        error "Invalid destination folder: $to_folder"
    fi
    
    # Find the file
    local found=""
    local source_path=""
    for folder in open investigating in-progress fixed closed failed; do
        if [[ -f "$ISSUES_DIR/$folder/$filename" ]]; then
            found="$folder"
            source_path="$ISSUES_DIR/$folder/$filename"
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
    
    # Update status in YAML file
    sed -i "s/status: $found/status: $to_folder/g" "$source_path"
    sed -i "s/updated_at: \".*\"/updated_at: \"$(date -Iseconds)\"/g" "$source_path"
    
    # Add timestamp for state transitions
    case "$to_folder" in
        investigating)
            if ! grep -q "investigation:" "$source_path"; then
                sed -i "/investigation:/a\\  started_at: \"$(date -Iseconds)\"" "$source_path"
            fi
            ;;
        in-progress)
            if ! grep -q "implementation:" "$source_path"; then
                sed -i "/implementation:/a\\  started_at: \"$(date -Iseconds)\"" "$source_path"
            fi
            ;;
        fixed)
            sed -i "s/resolved_at: \"\"/resolved_at: \"$(date -Iseconds)\"/g" "$source_path"
            ;;
    esac
    
    # Move the file
    mv "$source_path" "$ISSUES_DIR/$to_folder/$filename"
    success "Moved $filename from $found to $to_folder"
}

# Change issue priority (rename file)
priority() {
    if [[ $# -lt 2 ]]; then
        error "Usage: $0 priority <filename> <new-priority>"
    fi
    
    local filename="$1"
    local new_priority="$2"
    
    # Find the file
    local found=""
    local source_path=""
    for folder in open investigating in-progress fixed closed failed; do
        if [[ -f "$ISSUES_DIR/$folder/$filename" ]]; then
            found="$folder"
            source_path="$ISSUES_DIR/$folder/$filename"
            break
        fi
    done
    
    if [[ -z "$found" ]]; then
        error "File not found: $filename"
    fi
    
    # Determine new number range
    local new_num
    case "$new_priority" in
        critical) new_num=001 ;;
        high) new_num=100 ;;
        medium) new_num=200 ;;
        low) new_num=500 ;;
        *) error "Invalid priority: $new_priority" ;;
    esac
    
    # Extract current name part (after first hyphen and number)
    local name_part=$(echo "$filename" | sed 's/^[0-9]\{3\}-//')
    local new_filename="${new_num}-${name_part}"
    
    # Check if new filename already exists
    if [[ -f "$ISSUES_DIR/$found/$new_filename" ]]; then
        # Find next available number in range
        local start_num end_num
        case "$new_priority" in
            critical) start_num=1; end_num=99 ;;
            high) start_num=100; end_num=199 ;;
            medium) start_num=200; end_num=499 ;;
            low) start_num=500; end_num=999 ;;
        esac
        
        for ((i=start_num; i<=end_num; i++)); do
            local test_filename=$(printf "%03d-%s" "$i" "$name_part")
            if [[ ! -f "$ISSUES_DIR/$found/$test_filename" ]]; then
                new_filename="$test_filename"
                break
            fi
        done
    fi
    
    # Update priority in YAML file
    sed -i "s/priority: .*/priority: $new_priority/g" "$source_path"
    sed -i "s/updated_at: \".*\"/updated_at: \"$(date -Iseconds)\"/g" "$source_path"
    
    # Rename file
    mv "$source_path" "$ISSUES_DIR/$found/$new_filename"
    success "Changed priority: $filename → $new_filename (priority: $new_priority)"
}

# Search issues across all folders
search() {
    local query="$*"
    if [[ -z "$query" ]]; then
        error "Search query required"
    fi
    
    echo -e "${CYAN}🔍 Searching for: $query${NC}"
    echo
    
    local results=0
    for file in "$ISSUES_DIR"/*/*.yaml; do
        if [[ -f "$file" ]] && grep -qi "$query" "$file"; then
            local folder=$(basename "$(dirname "$file")")
            local filename=$(basename "$file")
            local title=$(grep "^title:" "$file" | sed 's/title: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
            
            # Color code by folder
            case "$folder" in
                open) folder_color="${RED}" ;;
                investigating) folder_color="${BLUE}" ;;
                in-progress) folder_color="${YELLOW}" ;;
                fixed) folder_color="${GREEN}" ;;
                *) folder_color="${NC}" ;;
            esac
            
            echo -e "${folder_color}●${NC} $filename (${folder})"
            echo -e "  ${title}"
            
            # Show matching lines
            local matches=$(grep -i "$query" "$file" | head -3)
            while IFS= read -r match; do
                echo -e "  ${CYAN}→${NC} $match"
            done <<< "$matches"
            echo
            ((results++))
        fi
    done
    
    if [[ $results -eq 0 ]]; then
        info "No issues found matching: $query"
    else
        success "Found $results matching issues"
    fi
}

# Show issue details
show() {
    local filename="$1"
    if [[ -z "$filename" ]]; then
        error "Usage: $0 show <filename>"
    fi
    
    # Find the file
    local found=""
    local file_path=""
    for folder in open investigating in-progress fixed closed failed; do
        if [[ -f "$ISSUES_DIR/$folder/$filename" ]]; then
            found="$folder"
            file_path="$ISSUES_DIR/$folder/$filename"
            break
        fi
    done
    
    if [[ -z "$found" ]]; then
        error "File not found: $filename"
    fi
    
    echo -e "${CYAN}📄 Issue Details: $filename${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Show formatted content
    if command -v bat &> /dev/null; then
        bat "$file_path" --style=header,grid --language=yaml
    elif command -v pygmentize &> /dev/null; then
        pygmentize -l yaml "$file_path"
    else
        cat "$file_path"
    fi
}

# Delete issue
delete() {
    local filename="$1"
    if [[ -z "$filename" ]]; then
        error "Usage: $0 delete <filename>"
    fi
    
    # Find and confirm deletion
    for folder in open investigating in-progress fixed closed failed; do
        local filepath="$ISSUES_DIR/$folder/$filename"
        if [[ -f "$filepath" ]]; then
            local title=$(grep "^title:" "$filepath" | sed 's/title: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
            echo -e "${YELLOW}⚠️  Delete issue:${NC} $title"
            echo -e "${YELLOW}   File:${NC} $filename"
            echo -e "${YELLOW}   Status:${NC} $folder"
            echo
            read -p "Are you sure? (y/N) " confirm
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

# Archive old issues
archive() {
    local days="${1:-30}"
    echo -e "${CYAN}📦 Archiving issues older than $days days${NC}"
    
    mkdir -p "$ISSUES_DIR/archive"
    local count=0
    
    for folder in fixed closed; do
        while IFS= read -r -d '' file; do
            local filename=$(basename "$file")
            mv "$file" "$ISSUES_DIR/archive/"
            info "Archived: $filename"
            ((count++))
        done < <(find "$ISSUES_DIR/$folder" -name "*.yaml" -mtime +$days -print0 2>/dev/null)
    done
    
    if [[ $count -eq 0 ]]; then
        info "No issues to archive"
    else
        success "Archived $count issues"
    fi
}

# Show help
show_help() {
    cat << EOF
${CYAN}🐛 App Issue Tracker Management CLI${NC}

Usage: $0 <command> [arguments]

Commands:
  ${GREEN}status${NC}              Show issue statistics overview
  ${GREEN}list [folder]${NC}       List issues in folder (default: open)
  ${GREEN}add${NC}                 Add new issue interactively
  ${GREEN}move <file> <to>${NC}    Move issue between status folders
  ${GREEN}priority <file> <p>${NC} Change issue priority (critical|high|medium|low)
  ${GREEN}search <query>${NC}      Search across all issues
  ${GREEN}show <file>${NC}         Display full issue details
  ${GREEN}delete <file>${NC}       Delete issue permanently
  ${GREEN}archive [days]${NC}      Archive fixed/closed issues older than N days (default: 30)
  ${GREEN}help${NC}                Show this help message

Folders:
  open               New issues waiting for investigation
  investigating      Issues currently being analyzed by AI
  in-progress        Issues with fixes being developed
  fixed              Successfully resolved issues
  closed             Closed without fix (duplicate, wontfix, etc)
  failed             Failed investigation/fix attempts

Priority Ranges:
  critical           001-099 (immediate attention)
  high               100-199 (next sprint)
  medium             200-499 (planned work)
  low                500-999 (backlog)

Examples:
  $0 status                                  # Show overview
  $0 list investigating                      # List issues being analyzed
  $0 add                                     # Interactive issue creation
  $0 move 001-auth-bug.yaml investigating   # Start investigation
  $0 priority 500-minor-bug.yaml critical   # Escalate to critical
  $0 search "authentication timeout"        # Find related issues
  $0 show 001-auth-bug.yaml                 # View full details
  $0 archive 60                             # Archive old issues

File Management:
  # Direct file operations also work:
  cp issues/templates/bug-template.yaml issues/open/123-new-bug.yaml
  mv issues/open/*.yaml issues/investigating/
  grep -r "database" issues/

EOF
}

# Main command dispatcher
case "${1:-help}" in
    status)
        status
        ;;
    list)
        list "${2:-open}"
        ;;
    add)
        add
        ;;
    move)
        move "$2" "$3"
        ;;
    priority)
        priority "$2" "$3"
        ;;
    search)
        shift
        search "$@"
        ;;
    show)
        show "$2"
        ;;
    delete)
        delete "$2"
        ;;
    archive)
        archive "${2:-30}"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        error "Unknown command: $1. Use '$0 help' for usage."
        ;;
esac