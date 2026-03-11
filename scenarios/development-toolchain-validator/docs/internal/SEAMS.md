# Integration Boundaries (SEAMS)

## External Seams

### prompt-manager API (HTTP Client)
- **Boundary**: `api/pkg/promptmanager/client.go`
- **What crosses**: Skill metadata, version history, content hashes
- **Contract**: REST API at `PROMPT_MANAGER_API_URL`
- **Testability**: Interface-based client, mockable in unit tests
- **Failure mode**: If prompt-manager is down, drift detection and skill fetching fail. Connections can still be managed with cached version data.

### Scenario CLIs (Subprocess Execution)
- **Boundary**: `api/pkg/validation/cli_executor.go`
- **What crosses**: CLI commands, JSON stdout
- **Contract**: Commands produce valid JSON on stdout, exit code 0 on success
- **Testability**: Executor interface, mockable in unit tests. Integration tests run real commands.
- **Failure modes**: Command not found, timeout, non-JSON output, non-zero exit code

### Reference Scenario Filesystem (os.Stat / filepath.Glob)
- **Boundary**: `api/pkg/validation/structural_checker.go`
- **What crosses**: File/folder existence checks, file content reads
- **Contract**: Standard POSIX filesystem operations
- **Testability**: Use temp directories in tests with known structures
- **Failure modes**: Permission errors, race conditions with concurrent modifications

### PostgreSQL (Database)
- **Boundary**: `api/pkg/db/` (connection management), domain repositories
- **What crosses**: CRUD for all entities, validation history
- **Contract**: PostgreSQL 15+ with schema from `initialization/postgres/`
- **Testability**: Testcontainers with postgres:15-alpine
- **Failure modes**: Connection refused, schema mismatch, constraint violations

## Internal Seams

### Registry → Validation Engine
- Registry provides connection and expectation data
- Validation engine operates on this data without modifying it
- Clear separation: registry is CRUD, validation is read-only execution

### Validation Engine → Report Generator
- Validation produces per-expectation results
- Report generator aggregates into overlaps, conflicts, summaries
- Overlap/conflict detection is a separate analysis pass, not inline with validation

### API Handlers → Domain Services
- Handlers parse HTTP, call domain services, serialize responses
- Domain services contain all business logic
- Handlers do not access the database directly
