# Data Model

Database schema and entity relationships for the reference-react-vite scenario.

## Entity Relationship Diagram

```
┌─────────────────────┐
│      projects       │
├─────────────────────┤
│ id (PK)             │
│ name                │
│ description         │
│ color               │
│ status              │
│ created_at          │
│ updated_at          │
└──────────┬──────────┘
           │
           │ 1:N (optional)
           │
           ▼
┌─────────────────────┐
│       tasks         │
├─────────────────────┤
│ id (PK)             │
│ title               │
│ description         │
│ status              │
│ priority            │
│ due_date            │
│ project_id (FK)     │◄─── Optional reference to projects
│ created_at          │
│ updated_at          │
└──────────┬──────────┘
           │
           │ 1:N
           │
           ▼
┌─────────────────────┐
│       notes         │
├─────────────────────┤
│ id (PK)             │
│ content             │
│ author              │
│ task_id (FK)        │◄─── Required reference to tasks
│ created_at          │
│ updated_at          │
└─────────────────────┘
```

## Tables

### projects
[CODE: initialization/postgres/schema.sql:10-22]

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| id | UUID | NO | uuid_generate_v4() | Primary key |
| name | VARCHAR(200) | NO | - | Project name |
| description | TEXT | YES | - | Optional description |
| color | VARCHAR(7) | YES | - | Hex color (#RRGGBB) |
| status | VARCHAR(20) | NO | 'active' | active, archived, completed |
| created_at | TIMESTAMPTZ | NO | CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMPTZ | NO | CURRENT_TIMESTAMP | Last update timestamp |

**Constraints**:
- `chk_projects_status`: status IN ('active', 'archived', 'completed')
- `chk_projects_color`: color ~ '^#[0-9A-Fa-f]{6}$' OR NULL

**Indexes**:
- `idx_projects_status`: status
- `idx_projects_created`: created_at DESC

### tasks
[CODE: initialization/postgres/schema.sql:24-42]

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| id | UUID | NO | uuid_generate_v4() | Primary key |
| title | VARCHAR(500) | NO | - | Task title |
| description | TEXT | YES | - | Optional description |
| status | VARCHAR(20) | NO | 'pending' | Task status |
| priority | INT | NO | 3 | 1 (low) to 5 (high) |
| due_date | TIMESTAMPTZ | YES | - | Optional deadline |
| project_id | UUID | YES | - | Optional parent project |
| created_at | TIMESTAMPTZ | NO | CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMPTZ | NO | CURRENT_TIMESTAMP | Last update timestamp |

**Constraints**:
- `chk_tasks_status`: status IN ('pending', 'in_progress', 'completed', 'blocked')
- `chk_tasks_priority`: priority BETWEEN 1 AND 5
- `fk_tasks_project`: project_id → projects(id) ON DELETE SET NULL

**Indexes**:
- `idx_tasks_project`: project_id
- `idx_tasks_status`: status
- `idx_tasks_priority`: priority DESC
- `idx_tasks_due_date`: due_date
- `idx_tasks_created`: created_at DESC

### notes
[CODE: initialization/postgres/schema.sql:44-56]

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| id | UUID | NO | uuid_generate_v4() | Primary key |
| content | TEXT | NO | - | Note content |
| author | VARCHAR(200) | YES | - | Optional author identifier |
| task_id | UUID | NO | - | Parent task (required) |
| created_at | TIMESTAMPTZ | NO | CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMPTZ | NO | CURRENT_TIMESTAMP | Last update timestamp |

**Constraints**:
- `fk_notes_task`: task_id → tasks(id) ON DELETE CASCADE

**Indexes**:
- `idx_notes_task`: task_id
- `idx_notes_created`: created_at DESC

## Triggers

### Auto-update timestamps
[CODE: initialization/postgres/schema.sql:58-75]

All tables have triggers that automatically update `updated_at` on row modification:

```sql
CREATE TRIGGER projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

## Schema Initialization

The schema file is designed to be idempotent:

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS projects (...);
DROP TRIGGER IF EXISTS projects_updated_at ON projects;
CREATE TRIGGER projects_updated_at ...;
```

This allows the schema to be safely re-run without errors.

## Seed Data
[CODE: initialization/postgres/seed.sql]

Sample data for demonstration:
- 2 projects (Work, Personal)
- 5 tasks across both projects
- 3 notes on various tasks

## Related

- [API Reference](api-endpoints.md) - REST endpoints
- [Configuration](configuration.md) - Database connection settings
- [internal/STORAGE_AUDIT.md](../internal/STORAGE_AUDIT.md) - Storage architecture compliance
