# Vrooli Bridge

The Vrooli Bridge enables external projects to leverage Vrooli's intelligence by managing documentation injection and integration tracking.

## 🎯 Intelligence Amplification

This capability creates a bidirectional learning loop where:
- Agents learn from diverse codebases and project structures outside Vrooli
- External project patterns inform better scenario design
- Cross-project intelligence accumulates to identify common development needs
- Vrooli's capabilities become universally applicable, not just internal

## 🔁 Recursive Value

New scenarios enabled after this exists:
1. **cross-project-refactor** - Safely refactor code patterns across multiple Vrooli-enabled projects
2. **dependency-orchestrator** - Manage dependencies and breaking changes across integrated projects
3. **project-health-monitor** - Track and improve health metrics across all Vrooli-enabled codebases
4. **pattern-harvester** - Extract successful patterns from external projects to improve Vrooli scenarios
5. **multi-project-test-runner** - Coordinate testing across interdependent Vrooli-enabled projects

## 📊 Performance Targets

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Scan Time | < 5s for 100 projects | CLI timing |
| Documentation Generation | < 500ms per project | API monitoring |
| UI Response Time | < 200ms for list operations | Frontend monitoring |
| Database Queries | < 50ms for project lookups | PostgreSQL monitoring |

## Features
- Project registry and metadata management
- Automated documentation injection (CLAUDE_ADDITIONS.md, VROOLI_INTEGRATION.md)
- Integration tracking across Vrooli ecosystem
- REST API for programmatic access
- Simple dashboard for project management

## Quick Start
```bash
make start  # Start API and UI (recommended)
# Or: vrooli scenario start vrooli-bridge
# Access UI at http://localhost:${UI_PORT}
# API at http://localhost:${API_PORT}
```

## Development
- API: Go-based, see api/main.go
- UI: Vanilla JS, see ui/app.js
- Storage: PostgreSQL with schema in initialization/storage/postgres/

## Testing
```bash
make test  # Run full test suite
```

For detailed operational requirements, see PRD.md and requirements/index.json.