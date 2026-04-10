# Data Model

Schema definition: [CODE: initialization/postgres/schema.sql]
Domain model: [CODE: api/domain/reference/model.go#Reference]

## Entity Relationship Diagram

```
references
    │
    │ 1:N
    ▼
skill_connections
    │
    │ 1:N                    1:N
    ├──────────────┐──────────────┐
    ▼              ▼              ▼
structural_    cli_tool_      validation_
expectations   assertions     results
```

## Tables

### references

Schema: [CODE: initialization/postgres/schema.sql:9]
Repository: [CODE: api/infrastructure/postgres/reference_repo.go#ReferenceRepository]

Stores registered reference scenarios.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Unique identifier |
| name | TEXT | UNIQUE, NOT NULL | Scenario name (e.g., "reference-react-vite") |
| template | TEXT | NOT NULL | Template it's based on (e.g., "react-vite") |
| path | TEXT | NOT NULL | Filesystem path to scenario |
| created_at | TIMESTAMPTZ | NOT NULL | When registered |
| updated_at | TIMESTAMPTZ | NOT NULL | Last modified |

### skill_connections

Schema: [CODE: initialization/postgres/schema.sql:25]

Maps skills to references with version pinning.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Unique identifier |
| reference_id | UUID | FK → references(id) | Which reference |
| skill_id | TEXT | NOT NULL | prompt-manager skill ID (e.g., "api-steer") |
| skill_version | INT | NOT NULL | Version number at connection time |
| skill_content_hash | TEXT | NOT NULL | SHA256 hash at connection time |
| connected_at | TIMESTAMPTZ | NOT NULL | When connected |
| | | UNIQUE(reference_id, skill_id) | One connection per skill per reference |

### structural_expectations

Schema: [CODE: initialization/postgres/schema.sql:48]

Filesystem expectations per connection.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Unique identifier |
| connection_id | UUID | FK → skill_connections(id) ON DELETE CASCADE | Parent connection |
| type | TEXT | NOT NULL, CHECK (folder/file/snippet) | Expectation type |
| path | TEXT | NOT NULL | Path or glob pattern |
| required | BOOLEAN | DEFAULT true | Whether absence is a failure |
| snippet_content | TEXT | NULL | For snippets: expected content |
| snippet_location | TEXT | NULL | For snippets: where to search |
| description | TEXT | NULL | Human-readable explanation |
| created_at | TIMESTAMPTZ | NOT NULL | When created |

### cli_tool_assertions

Schema: [CODE: initialization/postgres/schema.sql:73]

CLI command assertions per connection.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Unique identifier |
| connection_id | UUID | FK → skill_connections(id) ON DELETE CASCADE | Parent connection |
| command | TEXT | NOT NULL | CLI command to execute |
| json_path | TEXT | NOT NULL | JSONPath expression |
| operator | TEXT | NOT NULL, CHECK (valid operators) | Comparison operator |
| expected_value | JSONB | NULL | Expected value (type varies) |
| timeout_seconds | INT | DEFAULT 60 | Command timeout |
| description | TEXT | NULL | Human-readable explanation |
| created_at | TIMESTAMPTZ | NOT NULL | When created |

### validation_runs

Schema: [CODE: initialization/postgres/schema.sql:94]

Historical validation results.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Unique identifier |
| reference_id | UUID | FK → references(id) | Which reference |
| run_at | TIMESTAMPTZ | NOT NULL | When validation ran |
| structural_pass | INT | NOT NULL | Count of passing structural checks |
| structural_fail | INT | NOT NULL | Count of failing structural checks |
| cli_pass | INT | NOT NULL | Count of passing CLI assertions |
| cli_fail | INT | NOT NULL | Count of failing CLI assertions |
| overlaps_detected | INT | NOT NULL | Number of overlaps found |
| conflicts_detected | INT | NOT NULL | Number of conflicts found |
| drift_detected | INT | NOT NULL | Number of drifted connections |
| unconfigured_skills | INT | NOT NULL | Number of unconfigured connections |
| report | JSONB | NOT NULL | Full report JSON |

### baseline_runs [P1]

Historical tooling baseline results.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Unique identifier |
| reference_id | UUID | FK → references(id) | Which reference |
| tool | TEXT | NOT NULL | Tool name (auditor/test-genie/completeness) |
| run_at | TIMESTAMPTZ | NOT NULL | When baseline ran |
| passed | BOOLEAN | NOT NULL | Whether baseline expectation was met |
| actual_result | JSONB | NOT NULL | Actual tool output (summary) |
| expected_result | JSONB | NOT NULL | Expected values |
| details | TEXT | NULL | Failure details if applicable |

## Valid Operator Values

The `operator` column in `cli_tool_assertions` accepts:
- `eq` — equals
- `neq` — not equals
- `gt` — greater than
- `gte` — greater than or equal
- `lt` — less than
- `lte` — less than or equal
- `exists` — JSONPath exists (expected_value is null)
- `contains` — string contains
- `matches` — regex match
- `between` — value between [min, max] (expected_value is JSON array)

## Indexes

```sql
CREATE INDEX idx_connections_reference ON skill_connections(reference_id);
CREATE INDEX idx_connections_skill ON skill_connections(skill_id);
CREATE INDEX idx_structural_connection ON structural_expectations(connection_id);
CREATE INDEX idx_cli_connection ON cli_tool_assertions(connection_id);
CREATE INDEX idx_validation_reference ON validation_runs(reference_id, run_at DESC);
CREATE INDEX idx_baseline_reference ON baseline_runs(reference_id, tool, run_at DESC);
```
