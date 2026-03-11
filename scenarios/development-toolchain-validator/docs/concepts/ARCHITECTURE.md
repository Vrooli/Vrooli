# Architecture

## System Overview

Development Toolchain Validator (DTV) is a Go API + React UI + CLI scenario that validates the Vrooli development ecosystem against known-good reference scenarios. It has four core subsystems:

```
┌─────────────────────────────────────────────────────────────────┐
│                   Development Toolchain Validator                │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │  Registry     │  │  Config      │  │  Validation Engine     │ │
│  │              │  │              │  │                        │ │
│  │ • References  │  │ • Structural │  │ • Structural checker   │ │
│  │ • Skill       │  │   expectations│ │ • CLI tool executor    │ │
│  │   connections │  │ • CLI tool   │  │ • Overlap detector     │ │
│  │ • Version     │  │   assertions │  │ • Conflict analyzer    │ │
│  │   tracking    │  │              │  │ • Report generator     │ │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬─────────────┘ │
│         │                 │                      │               │
│         └─────────────────┼──────────────────────┘               │
│                           │                                      │
│                    ┌──────┴───────┐                              │
│                    │  PostgreSQL   │                              │
│                    └──────────────┘                              │
└─────────────────────────────────────────────────────────────────┘
         │                                    │
         │ API calls                          │ CLI subprocess
         ▼                                    ▼
  ┌──────────────┐                  ┌──────────────────────┐
  │ prompt-manager│                 │ scenario-auditor      │
  │ API           │                 │ test-genie            │
  │ (read skills) │                 │ completeness-scoring  │
  └──────────────┘                  └──────────────────────┘
```

## Domain Modules

### 1. Registry (`api/pkg/registry/`)

Manages the core entities:

- **Reference Scenarios**: Which scenarios are registered as references, what template they're based on.
- **Skill Connections**: Which prompt-manager skills are connected to which references, with version pinning.
- **Drift Detection**: Compares stored version/hash against current skill state in prompt-manager.

### 2. Configuration (`api/pkg/config/`)

Stores the declarative expectations for each skill-reference connection:

- **Structural Expectations**: Folders, files (glob patterns), content snippets expected at specific locations.
- **CLI Tool Assertions**: Commands to run, JSONPath expressions to evaluate, operators, expected values.

### 3. Validation Engine (`api/pkg/validation/`)

Executes expectations against reference scenarios:

- **Structural Checker**: Walks the reference scenario's filesystem, checks folder/file existence, matches glob patterns, verifies snippets.
- **CLI Tool Executor**: Runs configured commands as subprocesses, parses JSON output, evaluates JSONPath assertions.
- **Overlap Detector**: Finds structural expectations from different skills that target the same files/folders.
- **Conflict Analyzer**: Identifies semantically incompatible expectations (mutually exclusive structures).
- **Report Generator**: Aggregates all results into a comprehensive validation report.

### 4. Tooling Baselines (`api/pkg/baselines/`) [P1]

Runs external development tools against references and validates expected results:

- **scenario-auditor**: Expects zero violations on a healthy reference.
- **test-genie**: Expects all 11 phases to pass.
- **scenario-completeness-scoring**: Expects score >= 96 (Production Ready).

## Data Flow

### Registration Flow

```
1. User registers reference: POST /api/v1/references
   └─ Store: reference name, template, path in PostgreSQL

2. User connects skill: POST /api/v1/connections
   └─ Fetch skill from prompt-manager API (version, content hash)
   └─ Store: skill ID, reference ID, version, hash in PostgreSQL

3. User adds expectations: POST /api/v1/connections/{id}/expectations
   └─ Store: structural or CLI tool expectations in PostgreSQL
```

### Validation Flow

```
1. Trigger: POST /api/v1/validate/{reference}

2. For each skill connection:
   a. Check drift (compare stored version vs current in prompt-manager)
   b. Run structural expectations:
      - Check folder existence
      - Match file glob patterns
      - Verify content snippets
   c. Run CLI tool assertions:
      - Execute command as subprocess with timeout
      - Parse JSON output
      - Evaluate JSONPath + operator + value
   d. Collect results per expectation

3. Cross-connection analysis:
   a. Overlap detection: find expectations targeting same paths
   b. Conflict detection: find mutually exclusive expectations

4. Generate report:
   - Per-connection results (pass/fail/drift)
   - Overlap map
   - Conflict list
   - Unconfigured skills (connected but no expectations)
   - Maturity scores

5. Persist results in PostgreSQL for history
```

### Tooling Baseline Flow [P1]

```
1. Trigger: POST /api/v1/baselines/{reference}

2. Run each tool:
   a. scenario-auditor audit {reference} --json → parse violations
   b. test-genie execute {reference} --preset comprehensive --json → parse phase results
   c. scenario-completeness-scoring score {reference} --json → parse score

3. Compare against expected baselines:
   - Auditor: zero violations (or configured allowlist)
   - Test-genie: all phases pass
   - Completeness: score >= 96

4. Report deviations as tooling issues (not reference issues)
```

## External Integrations

### prompt-manager API

**Purpose**: Read skill content, versions, and metadata. DTV is a consumer only.

**Endpoints used**:
- `GET /api/v1/skills/{id}` — Fetch skill metadata
- `GET /api/v1/skills/{id}/versions` — Fetch version history
- `GET /api/v1/skills/sync` — Bulk fetch with content hash for change detection

**Integration pattern**: HTTP client with retry and timeout. Skill content is NOT duplicated locally — always fetched from prompt-manager when needed for display or drift checking.

### Scenario CLIs (subprocess execution)

**Purpose**: Run read-only validation tools against reference scenarios.

**Tools invoked**:
- `scenario-auditor audit {reference} --json --timeout 240`
- `test-genie execute {reference} --preset comprehensive --json`
- `scenario-completeness-scoring score {reference} --json`

**Integration pattern**: Subprocess execution with configurable timeout. JSON output parsed and evaluated against configured assertions. Commands are read-only — they do not modify the reference.

## Database Schema (Conceptual)

```sql
-- Reference scenarios registered for validation
references (
  id UUID PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,        -- e.g., "reference-react-vite"
  template TEXT NOT NULL,           -- e.g., "react-vite"
  path TEXT NOT NULL,               -- filesystem path
  created_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ
)

-- Skills connected to references with version pinning
skill_connections (
  id UUID PRIMARY KEY,
  reference_id UUID REFERENCES references(id),
  skill_id TEXT NOT NULL,           -- prompt-manager skill ID
  skill_version INT NOT NULL,       -- version at connection time
  skill_content_hash TEXT NOT NULL,  -- SHA256 hash at connection time
  connected_at TIMESTAMPTZ,
  UNIQUE(reference_id, skill_id)
)

-- Structural expectations per connection
structural_expectations (
  id UUID PRIMARY KEY,
  connection_id UUID REFERENCES skill_connections(id),
  type TEXT NOT NULL,               -- 'folder', 'file', 'snippet'
  path TEXT NOT NULL,               -- glob pattern or file path
  required BOOLEAN DEFAULT true,
  snippet_content TEXT,             -- for type='snippet': expected content
  snippet_location TEXT,            -- for type='snippet': where in file
  description TEXT,
  created_at TIMESTAMPTZ
)

-- CLI tool assertions per connection
cli_tool_assertions (
  id UUID PRIMARY KEY,
  connection_id UUID REFERENCES skill_connections(id),
  command TEXT NOT NULL,            -- CLI command to execute
  json_path TEXT NOT NULL,          -- JSONPath expression
  operator TEXT NOT NULL,           -- eq, neq, gt, gte, lt, lte, exists, contains, matches, between
  expected_value JSONB,            -- expected value (type depends on operator)
  description TEXT,
  created_at TIMESTAMPTZ
)

-- Validation results (history)
validation_runs (
  id UUID PRIMARY KEY,
  reference_id UUID REFERENCES references(id),
  run_at TIMESTAMPTZ,
  structural_pass INT,
  structural_fail INT,
  cli_pass INT,
  cli_fail INT,
  overlaps_detected INT,
  conflicts_detected INT,
  drift_detected INT,
  unconfigured_skills INT,
  report JSONB                     -- full report stored as JSON
)
```

## Technology Choices

| Component | Choice | Rationale |
|-----------|--------|-----------|
| API | Go + standard library HTTP | Consistent with Vrooli patterns |
| Database | PostgreSQL | Consistent with Vrooli; structured data with relationships |
| UI | React + TypeScript + Vite | Standard react-vite template |
| CLI | Go + cli-core | Standard Vrooli CLI pattern |
| JSONPath | Go library (e.g., `ohler55/ojg`) | For evaluating assertions against JSON output |
| Subprocess | `os/exec` | For running CLI tools against references |
