# Vrooli Bridge

The Vrooli Bridge enables external projects to leverage Vrooli's intelligence by managing documentation injection and integration tracking.

## Features
- Project registry and metadata management
- Automated documentation injection (CLAUDE_ADDITIONS.md, VROOLI_INTEGRATION.md)
- Integration tracking across Vrooli ecosystem
- REST API for programmatic access
- Simple dashboard for project management

## Quick Start
```bash
make run  # Start API and UI
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

For detailed documentation, see PRD.md.