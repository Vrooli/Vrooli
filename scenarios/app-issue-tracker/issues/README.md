# File-Based Issue Storage System

## Overview

This folder-based issue system allows you to manage issues through simple YAML files. You can add, edit, and prioritize issues manually or through the API/CLI.

## Directory Structure

```
issues/
├── open/           # New issues waiting for investigation
├── investigating/  # Issues currently being analyzed by AI
├── in-progress/    # Issues with fixes being developed
├── fixed/          # Successfully resolved issues  
├── closed/         # Closed (duplicate, wontfix, invalid)
├── failed/         # Investigation/fix attempts that failed
├── templates/      # Template files for creating new issues
└── README.md       # This file
```

## How to Use

### Adding a New Issue

1. **Manual Method:**
   - Copy `templates/issue-template.yaml` to `open/`
   - Rename with format: `XXX-descriptive-name.yaml` (XXX = priority number)
   - Edit the file with your issue details
   - Save the file - it will appear in the UI automatically

2. **CLI Method:**
   ```bash
   app-issue-tracker create --title "Bug in auth service" --type bug --priority high
   ```

3. **API Method:**
   ```bash
   curl -X POST http://localhost:8090/api/issues -H "Content-Type: application/json" \
     -d '{"title": "Bug in auth service", "type": "bug", "priority": "high"}'
   ```

### Priority System

Files are processed based on their filename prefix:
- `001-` through `099-` = CRITICAL/HIGH priority
- `100-` through `199-` = MEDIUM priority  
- `200-` through `999-` = LOW priority

To change priority, simply rename the file with a different number prefix.

### Issue Workflow States

#### open/
New issues waiting for investigation. Add your YAML files here.

#### investigating/  
Issues currently being analyzed by Claude Code AI agents. Files are automatically moved here when investigation starts.

#### in-progress/
Issues where fixes are being developed or applied. Manual or automated fix generation in progress.

#### fixed/
Successfully resolved issues. Files include fix details, verification results, and closure information.

#### closed/
Issues that were closed without fixes (duplicates, won't fix, invalid, etc.).

#### failed/
Issues where investigation or fix attempts failed. Files include error information and can be moved back to open/ after addressing blockers.

## YAML File Format

Each issue is a YAML file with this structure:

```yaml
# Brief title - Priority: LEVEL | Reporter: email@domain.com  

id: unique-issue-identifier
title: "Descriptive issue title"
description: "Detailed description of the issue"

type: bug                    # bug|feature|improvement|performance|security
priority: critical           # critical|high|medium|low  
app_id: "app-name"
status: investigating        # open|investigating|in-progress|fixed|closed|failed

reporter:
  name: "Reporter Name"
  email: "reporter@email.com" 
  timestamp: "2024-12-15T09:00:00Z"

error_context:
  error_message: "Error details"
  stack_trace: |
    Stack trace here if available
  affected_files:
    - "path/to/file1.js"
    - "path/to/file2.py"

investigation:
  agent_id: "investigator-agent-name"
  report: "Investigation findings"
  root_cause: "Identified root cause"
  confidence_score: 8

fix:
  suggested_fix: "Description of fix"
  applied: true
  verification_status: "tests-passing"

metadata:
  created_at: "2024-12-15T09:00:00Z"
  updated_at: "2024-12-15T09:30:00Z"
  tags: ["tag1", "tag2"]
```

## File Operations

### Manual Operations
```bash
# Change priority (rename with new prefix)
mv issues/open/500-low-bug.yaml issues/open/001-critical-bug.yaml

# Move to different status
mv issues/open/001-auth-bug.yaml issues/investigating/

# Bulk status change
mv issues/investigating/*.yaml issues/failed/

# Search issues
grep -r "authentication" issues/
grep -l "timeout" issues/*/*.yaml
```

### CLI Operations
```bash
# List issues by status
app-issue-tracker list open
app-issue-tracker list --status investigating

# Move issue between statuses  
app-issue-tracker move issue-123 investigating

# Change priority
app-issue-tracker priority issue-123 critical

# Search semantically
app-issue-tracker search "authentication error"
```

## Integration with AI

### Investigation Process
1. Issue created in `open/`
2. AI agent analyzes → moves to `investigating/`
3. Claude Code generates investigation report
4. Report written back to YAML file
5. Issue moved to `in-progress/` or `failed/` based on results

### Vector Search
- Qdrant scans all YAML files for embeddings
- Semantic search works across all issue statuses
- Similar issue detection uses file content analysis

## Best Practices

1. **Use Clear Naming:** File names should be descriptive: `001-auth-timeout-bug.yaml` not `001-thing.yaml`

2. **Detailed Error Context:** Include stack traces, affected files, environment info

3. **Investigation Notes:** Add manual observations to the notes field

4. **Git Integration:** Commit issue changes to track investigation progress

5. **Backup Important Issues:** Copy critical issues to multiple folders if needed

## Monitoring

Check issue status:
```bash
# Count issues in each state  
echo "Open: $(ls issues/open/*.yaml 2>/dev/null | wc -l)"
echo "Investigating: $(ls issues/investigating/*.yaml 2>/dev/null | wc -l)" 
echo "In Progress: $(ls issues/in-progress/*.yaml 2>/dev/null | wc -l)"
echo "Fixed: $(ls issues/fixed/*.yaml 2>/dev/null | wc -l)"
echo "Closed: $(ls issues/closed/*.yaml 2>/dev/null | wc -l)"
echo "Failed: $(ls issues/failed/*.yaml 2>/dev/null | wc -l)"
```

## Troubleshooting

### Issue Not Appearing in UI
- Check YAML syntax: `yamllint issues/open/your-file.yaml`
- Ensure file has `.yaml` extension
- Verify file permissions are readable

### Investigation Stuck
- Check `investigating/` folder for stuck items
- Move back to `open/` to retry: `mv issues/investigating/stuck.yaml issues/open/`
- Check API logs for errors

### Priority Not Working  
- Ensure filename starts with 3-digit number: `001-`, not `1-` or `01-`
- Lower numbers = higher priority
- Rename file to change priority

---

For more information, see the main [App Issue Tracker documentation](../IMPLEMENTATION_PLAN.md)