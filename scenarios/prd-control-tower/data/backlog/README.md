# Backlog Storage

This directory contains git-backed backlog entries for the PRD Control Tower.

## Structure

Each backlog entry is stored as a markdown file with frontmatter metadata:

```markdown
---
id: <uuid>
entity_type: scenario|resource
suggested_name: <slug>
status: pending|converted|archived
created_at: <RFC3339 timestamp>
updated_at: <RFC3339 timestamp>
converted_draft_id: <uuid>  # Optional, present if status=converted
---

<idea text content>
```

## Persistence Strategy

- **Source of Truth**: Filesystem (git-backed)
- **Database**: PostgreSQL for fast querying (synchronized from filesystem on startup)
- **Sync**: Automatic on API `/api/v1/backlog` list endpoint
- **Safety**: All backlog entries are git-tracked to prevent data loss

## Files

- `{uuid}.md`: Individual backlog entry
- `.gitkeep`: Ensures directory is tracked by git

## Workflow

1. **Create**: User pastes freeform text → API creates markdown file + DB entry
2. **Convert**: Backlog entry → Draft PRD (status changes to "converted")
3. **Delete**: Removes both filesystem file and DB entry
4. **Sync**: On API startup, filesystem files are loaded into database

This ensures backlog ideas are never lost, even if the database is reset.
