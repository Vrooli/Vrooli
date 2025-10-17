#!/usr/bin/env bash

# Migration Script: PostgreSQL to File-Based YAML Storage
# Exports existing issues from database to YAML files

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${SCRIPT_DIR}/.."
ISSUES_DIR="${PROJECT_DIR}/data/issues"

echo "[WARN] migrate-to-files.sh targeted the legacy flat YAML layout." >&2
echo "       Folder-based bundles are now the default storage; database migration scripts will be refreshed in a future update." >&2
exit 1
BACKUP_DIR="${PROJECT_DIR}/backup/migration-$(date +%Y%m%d-%H%M%S)"

# Database configuration
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
# Validate POSTGRES_PASSWORD is set
if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
    error "POSTGRES_PASSWORD environment variable must be set"
    echo "Please export POSTGRES_PASSWORD with a secure password before running this script" >&2
    exit 1
fi
POSTGRES_DB="${POSTGRES_DB:-issue_tracker}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }

# Check if PostgreSQL is available
check_postgres() {
    log "Checking PostgreSQL connection..."
    
    if ! command -v psql &> /dev/null; then
        error "psql command not found. Please install PostgreSQL client."
        return 1
    fi
    
    if PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;" &>/dev/null; then
        success "PostgreSQL connection successful"
        return 0
    else
        warn "PostgreSQL connection failed. This might be expected if migrating to file-only mode."
        return 1
    fi
}

# Export issues from PostgreSQL to YAML files
export_issues() {
    log "Exporting issues from PostgreSQL..."
    
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$ISSUES_DIR"/{open,active,completed,failed,archived}
    
    # Export issues query
    local export_query="
    SELECT 
        i.id,
        i.title,
        i.description,
        i.status,
        i.priority,
        i.type,
        i.reporter_name,
        i.reporter_email,
        i.error_message,
        i.stack_trace,
        array_to_string(i.affected_files, ',') as affected_files,
        array_to_string(i.tags, ',') as tags,
        i.investigation_report,
        i.root_cause,
        i.suggested_fix,
        i.confidence_score,
        i.created_at,
        i.updated_at,
        i.resolved_at,
        a.name as app_name
    FROM issues i
    LEFT JOIN apps a ON i.app_id = a.id
    ORDER BY i.created_at DESC;
    "
    
    # Export to CSV first
    local csv_file="${BACKUP_DIR}/issues_export.csv"
    
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
        -c "COPY ($export_query) TO STDOUT WITH CSV HEADER" > "$csv_file"
    
    success "Exported $(wc -l < "$csv_file") issues to CSV: $csv_file"
    
    # Convert CSV to YAML files
    local count=0
    while IFS=',' read -r id title description status priority type reporter_name reporter_email error_message stack_trace affected_files tags investigation_report root_cause suggested_fix confidence_score created_at updated_at resolved_at app_name; do
        # Skip header
        if [[ "$id" == "id" ]]; then
            continue
        fi
        
        ((count++))
        
        # Clean up data (remove quotes from CSV)
        id=$(echo "$id" | sed 's/^"//;s/"$//')
        title=$(echo "$title" | sed 's/^"//;s/"$//' | sed 's/""/"/g')
        description=$(echo "$description" | sed 's/^"//;s/"$//' | sed 's/""/"/g')
        status=$(echo "$status" | sed 's/^"//;s/"$//')
        priority=$(echo "$priority" | sed 's/^"//;s/"$//')
        app_name=$(echo "$app_name" | sed 's/^"//;s/"$//')
        
        # Map database status to file folders
        local target_folder="$status"
        case "$status" in
            "open") target_folder="open" ;;
            "investigating"|"in_progress") target_folder="active" ;;
            "fixed"|"closed") target_folder="completed" ;;
            "failed") target_folder="failed" ;;
            "archived") target_folder="archived" ;;
            *) target_folder="open" ;;
        esac
        
        # Generate filename with priority
        local priority_num
        case "$priority" in
            "critical") priority_num=001 ;;
            "high") priority_num=100 ;;
            "medium") priority_num=200 ;;
            "low") priority_num=500 ;;
            *) priority_num=200 ;;
        esac
        
        # Create safe filename
        local safe_title=$(echo "$title" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g' | sed 's/--*/-/g' | head -c 50)
        local filename="${priority_num}-migrated-${count}-${safe_title}.yaml"
        local filepath="$ISSUES_DIR/$target_folder/$filename"
        
        # Ensure unique filename
        local counter=1
        while [[ -f "$filepath" ]]; do
            filename="${priority_num}-migrated-${count}-${counter}-${safe_title}.yaml"
            filepath="$ISSUES_DIR/$target_folder/$filename"
            ((counter++))
        done
        
        # Create YAML file
        cat > "$filepath" << EOF
# Migrated from PostgreSQL - $(date +"%Y-%m-%d %H:%M:%S")

id: $id
title: "$(echo "$title" | sed 's/"/\\"/g')"
description: "$(echo "$description" | sed 's/"/\\"/g')"

type: $type
priority: $priority
app_id: "$(echo "$app_name" | sed 's/"/\\"/g')"
status: $status

reporter:
  name: "$(echo "$reporter_name" | sed 's/^"//;s/"$//' | sed 's/"/\\"/g')"
  email: "$(echo "$reporter_email" | sed 's/^"//;s/"$//' | sed 's/"/\\"/g')"
  timestamp: "$(echo "$created_at" | sed 's/^"//;s/"$//')"

error_context:
  error_message: "$(echo "$error_message" | sed 's/^"//;s/"$//' | sed 's/"/\\"/g')"
  stack_trace: |$(if [[ -n "$stack_trace" && "$stack_trace" != '""' ]]; then echo "$stack_trace" | sed 's/^"//;s/"$//' | sed 's/\\n/\n    /g' | sed 's/^/\n    /'; fi)
  affected_files:$(if [[ -n "$affected_files" && "$affected_files" != '""' ]]; then echo "$affected_files" | sed 's/^"//;s/"$//' | tr ',' '\n' | sed 's/^/\n    - "/;s/$/"/'; fi)

investigation:
  agent_id: "unified-resolver"
  started_at: ""
  completed_at: "$(echo "$updated_at" | sed 's/^"//;s/"$//')"
  report: "$(echo "$investigation_report" | sed 's/^"//;s/"$//' | sed 's/"/\\"/g')"
  root_cause: "$(echo "$root_cause" | sed 's/^"//;s/"$//' | sed 's/"/\\"/g')"
  suggested_fix: "$(echo "$suggested_fix" | sed 's/^"//;s/"$//' | sed 's/"/\\"/g')"
  confidence_score: $(echo "$confidence_score" | sed 's/^"//;s/"$//' | grep -oE '[0-9]+' | head -1 || echo "null")

fix:
  applied: false
  verification_status: ""

metadata:
  created_at: "$(echo "$created_at" | sed 's/^"//;s/"$//')"
  updated_at: "$(echo "$updated_at" | sed 's/^"//;s/"$//')"
  resolved_at: "$(echo "$resolved_at" | sed 's/^"//;s/"$//')"
  tags:$(if [[ -n "$tags" && "$tags" != '""' ]]; then echo "$tags" | sed 's/^"//;s/"$//' | tr ',' '\n' | sed 's/^/\n    - "/;s/$/"/'; fi)

notes: |
  Migrated from PostgreSQL database on $(date +"%Y-%m-%d %H:%M:%S")
  Original database ID: $id
EOF

    done < "$csv_file"
    
    success "Migrated $count issues to YAML files"
    log "Issues saved in: $ISSUES_DIR"
}

# Export apps data
export_apps() {
    log "Exporting apps configuration..."
    
    local apps_query="
    SELECT 
        name,
        display_name,
        type,
        status,
        scenario_name,
        total_issues,
        open_issues
    FROM apps;
    "
    
    local apps_file="${BACKUP_DIR}/apps.yaml"
    
    # Export apps to YAML
    cat > "$apps_file" << EOF
# Migrated Apps Configuration
# Date: $(date +"%Y-%m-%d %H:%M:%S")

apps:
EOF
    
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
        -t -c "$apps_query" | while IFS='|' read -r name display_name type status scenario_name total_issues open_issues; do
        
        name=$(echo "$name" | xargs)
        display_name=$(echo "$display_name" | xargs)
        type=$(echo "$type" | xargs)
        status=$(echo "$status" | xargs)
        
        if [[ -n "$name" ]]; then
            cat >> "$apps_file" << EOF
  $name:
    display_name: "$display_name"
    type: "$type"
    status: "$status"
    scenario_name: "$(echo "$scenario_name" | xargs)"
    total_issues: $(echo "$total_issues" | xargs | grep -oE '[0-9]+' || echo "0")
    open_issues: $(echo "$open_issues" | xargs | grep -oE '[0-9]+' || echo "0")
EOF
        fi
    done
    
    success "Apps configuration saved to: $apps_file"
}

# Create migration report
create_migration_report() {
    log "Creating migration report..."
    
    local report_file="${BACKUP_DIR}/migration-report.md"
    
    cat > "$report_file" << EOF
# PostgreSQL to File-Based Migration Report

**Migration Date:** $(date +"%Y-%m-%d %H:%M:%S")
**Source:** PostgreSQL Database ($POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB)
**Destination:** File-based YAML storage ($ISSUES_DIR)

## Migration Summary

- **Issues Migrated:** $(find "$ISSUES_DIR" -name "*.yaml" | wc -l) files
- **Backup Location:** $BACKUP_DIR
- **Migration Status:** Completed

## File Structure Created

\`\`\`
issues/
├── open/        # $(ls "$ISSUES_DIR/open"/*.yaml 2>/dev/null | wc -l) files
├── active/      # $(ls "$ISSUES_DIR/active"/*.yaml 2>/dev/null | wc -l) files
├── completed/   # $(ls "$ISSUES_DIR/completed"/*.yaml 2>/dev/null | wc -l) files
├── failed/      # $(ls "$ISSUES_DIR/failed"/*.yaml 2>/dev/null | wc -l) files
├── archived/    # $(ls "$ISSUES_DIR/archived"/*.yaml 2>/dev/null | wc -l) files
└── templates/   # Template files for new issues
\`\`\`

## Next Steps

1. **Test File-Based System:**
   \`\`\`bash
   cd $PROJECT_DIR
   ./issues/manage.sh status
   ./cli/app-issue-tracker-v2.sh list
   \`\`\`

2. **Start File-Based API:**
   \`\`\`bash
   cd api
   ./app-issue-tracker-file-api
   \`\`\`

3. **Update UI:**
   \`\`\`bash
   # Update UI to use app-v2.js
   cp ui/app-v2.js ui/app.js
   \`\`\`

4. **Verify Migration:**
   \`\`\`bash
   # Check issue counts match
   ./issues/manage.sh status
   
   # Test creating new issues
   ./issues/manage.sh add
   
   # Test search functionality
   ./issues/manage.sh search "authentication"
   \`\`\`

## File Operations Quick Reference

\`\`\`bash
# View issues
ls issues/open/*.yaml

# Move issue to different status
mv issues/open/001-bug.yaml issues/active/

# Edit issue manually
vi issues/active/001-bug.yaml

# Search across all issues
grep -r "timeout" issues/

# Get issue counts by status
for dir in issues/*/; do
  echo "\$(basename "\$dir"): \$(ls \$dir/*.yaml 2>/dev/null | wc -l) issues"
done
\`\`\`

## Rollback Plan

If you need to return to PostgreSQL:

1. **Restore Database:**
   \`\`\`bash
   # Import backup data
   PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB < $BACKUP_DIR/database_backup.sql
   \`\`\`

2. **Switch Back to Original API:**
   \`\`\`bash
   cd api
   ./app-issue-tracker-api  # Original database version
   \`\`\`

3. **Revert UI:**
   \`\`\`bash
   git checkout ui/app.js  # Revert to original version
   \`\`\`

## Benefits of File-Based System

✅ **Direct file editing** with any text editor
✅ **Git-friendly** - full version history and diffs
✅ **No database setup** required
✅ **Portable** - copy issues folder anywhere
✅ **Transparent** - all data visible in filesystem
✅ **Scriptable** - easy automation with bash/shell scripts
✅ **Backup-friendly** - simple file copy operations

## Troubleshooting

### Issues Not Appearing
- Check YAML syntax: \`yamllint issues/open/file.yaml\`
- Verify file permissions: \`chmod 644 issues/**/*.yaml\`

### API Not Working
- Start file-based API: \`cd api && ./app-issue-tracker-file-api\`
- Check port availability: \`lsof -i :8090\`

### Performance Issues
- File system performance depends on number of files
- Consider archiving old issues: \`./issues/manage.sh archive 90\`

---

Migration completed successfully! 🎉
EOF

    success "Migration report created: $report_file"
}

# Backup current database (if available)
backup_database() {
    if check_postgres; then
        log "Creating database backup..."
        
        local backup_file="${BACKUP_DIR}/database_backup.sql"
        
        PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" "$POSTGRES_DB" > "$backup_file"
        
        if [[ -f "$backup_file" && -s "$backup_file" ]]; then
            success "Database backup created: $backup_file"
            return 0
        else
            error "Database backup failed or empty"
            return 1
        fi
    else
        log "Skipping database backup (PostgreSQL not available)"
        return 0
    fi
}

# Main migration function
main() {
    log "Starting PostgreSQL to File-Based migration..."
    log "Project directory: $PROJECT_DIR"
    log "Issues directory: $ISSUES_DIR"
    log "Backup directory: $BACKUP_DIR"
    
    mkdir -p "$BACKUP_DIR"
    
    # Step 1: Backup existing database
    backup_database
    
    # Step 2: Export issues (if database is available)
    if check_postgres; then
        export_issues
        export_apps
    else
        warn "No PostgreSQL database found. Creating empty file structure..."
        mkdir -p "$ISSUES_DIR"/{open,active,completed,failed,archived,templates}
        
        # Create sample issues for testing
        log "Creating sample issues for testing..."
        
        cp "$ISSUES_DIR/templates/bug-template.yaml" "$ISSUES_DIR/open/001-sample-critical-bug.yaml"
        sed -i "s/unique-bug-id/sample-critical-auth-bug-001/g" "$ISSUES_DIR/open/001-sample-critical-bug.yaml"
        sed -i "s/Brief descriptive title of the bug/Critical authentication service timeout/g" "$ISSUES_DIR/open/001-sample-critical-bug.yaml"
        sed -i "s/priority: medium/priority: critical/g" "$ISSUES_DIR/open/001-sample-critical-bug.yaml"
        sed -i "s/YYYY-MM-DDTHH:MM:SSZ/$(date -Iseconds)/g" "$ISSUES_DIR/open/001-sample-critical-bug.yaml"
        
        success "Created sample issue: 001-sample-critical-bug.yaml"
    fi
    
    # Step 3: Create migration report
    create_migration_report
    
    # Step 4: Set up file permissions
    find "$ISSUES_DIR" -name "*.yaml" -exec chmod 644 {} \;
    chmod +x "$ISSUES_DIR/manage.sh"
    
    success "Migration completed successfully!"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "${GREEN}🎉 File-Based Issue Tracker Ready!${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "📁 Issues directory: $ISSUES_DIR"
    echo "📊 Management CLI: $ISSUES_DIR/manage.sh"
    echo "📋 Migration report: $BACKUP_DIR/migration-report.md"
    echo ""
    echo "🚀 Next Steps:"
    echo "  1. Test file operations: $ISSUES_DIR/manage.sh status"
    echo "  2. Start file-based API: cd api && ./app-issue-tracker-file-api"
    echo "  3. Update UI: cp ui/app-v2.js ui/app.js"
    echo "  4. Test in browser: http://localhost:8090"
    echo ""
    echo "📖 Full documentation: $ISSUES_DIR/README.md"
}

# Handle command line arguments
case "${1:-migrate}" in
    migrate)
        main
        ;;
    backup-only)
        backup_database
        ;;
    export-only)
        export_issues
        export_apps
        ;;
    test)
        check_postgres
        echo "PostgreSQL available: $?"
        ;;
    *)
        echo "Usage: $0 [migrate|backup-only|export-only|test]"
        exit 1
        ;;
esac
