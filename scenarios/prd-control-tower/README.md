# PRD Control Tower

> Centralized management, validation, and publishing of Product Requirements Documents across all Vrooli scenarios and resources.

## 🎯 Purpose

The PRD Control Tower provides a comprehensive command center for managing PRDs throughout their lifecycle:
- **Browse** published PRDs across all scenarios and resources
- **Create** new drafts from templates with AI assistance
- **Edit** drafts with markdown editor and live preview
- **Validate** PRD structure against standards (via scenario-auditor)
- **Publish** drafts to repository with diff preview
- **Integrate** with ecosystem-manager for new scenario creation

## 🚀 Quick Start

### Prerequisites
- PostgreSQL (for draft metadata)
- scenario-auditor (for PRD validation)
- resource-openrouter (optional, for AI assistance)

### Setup and Start
```bash
cd scenarios/prd-control-tower

# First time setup (initialize database)
make setup

# Start service (via Makefile - recommended)
make start

# OR start via vrooli CLI
vrooli scenario start prd-control-tower
```

### Access UI
```bash
# Open in browser
open http://localhost:36300
```

### CLI Usage
```bash
# Install CLI
cd cli && ./install.sh

# Check status
prd-control-tower status

# List drafts
prd-control-tower list-drafts

# Validate PRD
prd-control-tower validate <scenario-name>

# Create draft (interactive)
prd-control-tower create-draft

# Publish draft
prd-control-tower publish <draft-id>
```

## 📋 Features

### Catalog View
- List all scenarios and resources with PRD status
- Status indicators: ✅ Has PRD, ⚠️ Violations, ❌ Missing, 📝 Draft Pending
- Filter by type (scenario/resource), search by name
- Click to view published PRD or open draft

### Draft Management
- Create new draft from template or existing PRD
- Edit with markdown editor and live preview
- Autosave (local + explicit save)
- Validate structure against standards
- AI-assisted section generation
- Diff viewer comparing draft vs published
- Publish to PRD.md with atomic file write

### Requirements & Target Coverage
- Parse modular `requirements/` registries (JSON + imports) for each scenario/resource
- Extract operational targets from PRD checklists and link them to requirements via `prd_ref`
- Surface missing template sections, unlinked targets, and unmatched requirements directly in the draft workspace
- Expose `/catalog/{type}/{name}/requirements` and `/catalog/{type}/{name}/targets` APIs for UI + automation consumers

### AI Assistance
- Generate missing sections with natural language prompts
- Rewrite sections for template compliance
- Context-aware suggestions (includes existing PRD, violations)
- Staged with diff approval before applying

### Integration
- scenario-auditor: Real-time PRD structure validation
- resource-openrouter: AI section generation and rewriting
- ecosystem-manager: Create new scenarios with draft PRD

## 🏗️ Architecture

### Components
1. **API** (Go): Catalog service, draft CRUD, validation, AI integration, publishing
2. **UI** (React/TypeScript): Catalog view, markdown editor, draft manager, diff viewer
3. **CLI** (Bash): Status, list drafts, validate, create, publish commands
4. **Storage**: Filesystem (`data/prd-drafts/`) + PostgreSQL (metadata)

### Endpoints
- `GET /api/v1/catalog` - List all entities with PRD status
- `GET /api/v1/drafts` - List drafts
- `POST /api/v1/drafts` - Create draft
- `PUT /api/v1/drafts/{id}` - Update draft
- `POST /api/v1/drafts/{id}/publish` - Publish to PRD.md
- `POST /api/v1/drafts/{id}/ai/generate-section` - AI assistance

### Ports
- **API**: 18600
- **UI**: 36300

## 🧪 Testing

```bash
# Run all tests
make test

# Individual test phases
./test/phases/test-structure.sh      # Directory layout validation
./test/phases/test-dependencies.sh   # Dependency checks
./test/phases/test-integration.sh    # API integration tests
./test/phases/test-unit.sh           # Go unit tests

# CLI tests (bats)
bats cli/prd-control-tower.bats      # 18 CLI command tests

# Go unit tests
cd api && go test -v -cover ./...    # Unit tests with coverage
```

## 📖 Development

### Prerequisites
- Go 1.21+
- Node.js 18+
- PostgreSQL 15+

### Setup
```bash
# Initialize database and dependencies
make setup

# Start development servers
make develop

# View logs
make logs

# Stop services
make stop
```

### Project Structure
```
prd-control-tower/
├── api/                    # Go API service
│   ├── main.go            # Entry point
│   ├── catalog.go         # Catalog enumeration
│   ├── drafts.go          # Draft CRUD operations
│   ├── validation.go      # scenario-auditor integration
│   ├── ai.go              # resource-openrouter integration
│   └── publish.go         # Publishing logic
├── ui/                    # React TypeScript UI
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Catalog.tsx      # Entity catalog view
│   │   │   ├── DraftEditor.tsx  # Markdown editor
│   │   │   └── DraftIndex.tsx   # Draft list
│   │   └── components/
│   │       ├── MarkdownEditor.tsx  # TipTap editor
│   │       ├── DiffViewer.tsx      # Monaco diff
│   │       └── StatusChip.tsx      # Status indicators
│   └── package.json
├── cli/                   # Bash CLI wrapper
│   └── prd-control-tower
├── data/                  # Draft storage
│   └── prd-drafts/
│       ├── scenario/      # Scenario drafts
│       └── resource/      # Resource drafts
├── initialization/        # Setup scripts
│   └── postgres/
│       └── schema.sql     # Database schema
├── test/                  # Test suite
│   └── phases/
├── Makefile              # Build and lifecycle
├── PRD.md                # This document
└── README.md             # Usage guide
```

## 🔧 Configuration

### Environment Variables
```bash
# API Configuration
PRD_CONTROL_TOWER_API_PORT=18600
PRD_CONTROL_TOWER_DB_HOST=localhost
PRD_CONTROL_TOWER_DB_PORT=5432
PRD_CONTROL_TOWER_DB_NAME=vrooli

# UI Configuration
PRD_CONTROL_TOWER_UI_PORT=36300
PRD_CONTROL_TOWER_API_URL=http://localhost:18600

# AI Integration (Required for AI features)
OPENROUTER_API_KEY=sk-or-v1-...  # Get from https://openrouter.ai
RESOURCE_OPENROUTER_URL=https://openrouter.ai/api/v1  # Optional, defaults to OpenRouter public API

# Integration URLs
SCENARIO_AUDITOR_URL=http://localhost:18507
ECOSYSTEM_MANAGER_URL=http://localhost:18200
```

### Database Setup
```sql
-- Drafts table
CREATE TABLE drafts (
    id UUID PRIMARY KEY,
    entity_type VARCHAR(20) NOT NULL,  -- 'scenario' or 'resource'
    entity_name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    owner VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'draft'  -- 'draft', 'validating', 'ready'
);

-- Audit results cache
CREATE TABLE audit_results (
    draft_id UUID REFERENCES drafts(id) ON DELETE CASCADE,
    violations JSONB NOT NULL,
    cached_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (draft_id)
);
```

## 📚 Resources

### Internal
- [PRD Template](/scripts/scenarios/templates/react-vite/PRD.md)
- [Scenario Auditor](/scenarios/scenario-auditor/README.md)
- [Ecosystem Manager](/scenarios/ecosystem-manager/README.md)

### External
- [TipTap Documentation](https://tiptap.dev/)
- [shadcn/UI Components](https://ui.shadcn.com/)
- [CommonMark Spec](https://spec.commonmark.org/)

## 🤝 Contributing

Follow the standard Vrooli scenario development workflow:
1. Review PRD.md for requirements
2. Implement features following existing patterns
3. Add tests for new functionality
4. Update documentation
5. Run full test suite before committing

## 📄 License

Part of the Vrooli ecosystem. See root LICENSE file.
