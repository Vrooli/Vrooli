# Swarm Manager Architecture

## Mental Model

Swarm Manager is the **central command center** for the Vrooli scenario ecosystem. It orchestrates three primary domains:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           SWARM MANAGER                                  │
│  "The place where ideas become scenarios and scenarios become products" │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐    ┌─────────────┐    ┌──────────────┐    ┌─────────┐ │
│  │   IDEAS     │───▶│  SCENARIOS  │───▶│RECOMMENDATIONS│◀───│INSIGHTS │ │
│  │  (backlog)  │    │  (catalog)  │    │   (engine)   │    │(patterns)│ │
│  └─────────────┘    └─────────────┘    └──────────────┘    └─────────┘ │
│       │                   │                   │                  │      │
│       ▼                   ▼                   ▼                  ▼      │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    EXTERNAL INTEGRATIONS                         │   │
│  │  agent-manager │ ecosystem-manager │ test-genie │ knowledge-obs │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Domain Concepts

| Concept | Description | Lifecycle States | Implementation |
|---------|-------------|------------------|----------------|
| **Idea** | A proposal for a new scenario, stored as git-tracked folders | backlog → researching → ready → queued → in_progress → completed/archived | [CODE: ui/src/types/domain.ts#Idea] |
| **Scenario** | A deployed application in the Vrooli ecosystem | running, stopped, error, unknown | [CODE: ui/src/types/domain.ts#Scenario] |
| **Recommendation** | A system-generated suggestion for improvement | pending → approved/rejected | [CODE: ui/src/types/domain.ts#Recommendation] |
| **Insight** | Pattern-based observation about system health | N/A (informational) | (not yet implemented) |

### Key Flows

1. **Idea-to-Scenario Pipeline**
   ```
   Create Idea → Research → Queue for Processing → ecosystem-manager initializes → Scenario exists
   ```

2. **Recommendation Engine Flow**
   ```
   Data Sources (PROBLEMS.md, completeness, tests) → Engine analyzes → Generates Recommendations → User approves/rejects
   ```

3. **Scenario Lifecycle Management**
   ```
   View Scenarios → Configure (greenfield/brownfield, enable/disable) → Delete with safeguards
   ```

## Logical Architecture

### Layer Responsibilities

```
┌─────────────────────────────────────────────────────────────┐
│ PRESENTATION LAYER (UI)                                      │
│ React components, pages, routing                             │
│ Surfaces: Desktop tabs, Mobile bottom-nav                    │
├─────────────────────────────────────────────────────────────┤
│ API GATEWAY LAYER (API)                                      │
│ HTTP endpoints, request validation, response formatting      │
│ Technology: Go + Gorilla Mux                                 │
├─────────────────────────────────────────────────────────────┤
│ BUSINESS LOGIC LAYER (to be implemented)                     │
│ Idea management, scenario operations, recommendation engine  │
│ Currently: Placeholder, minimal implementation               │
├─────────────────────────────────────────────────────────────┤
│ INTEGRATION LAYER (to be implemented)                        │
│ Adapters for agent-manager, ecosystem-manager, etc.          │
│ Pattern: All agent work via agent-manager (never direct)     │
├─────────────────────────────────────────────────────────────┤
│ PERSISTENCE LAYER                                            │
│ PostgreSQL (metadata), Redis (queues), Filesystem (ideas/)   │
└─────────────────────────────────────────────────────────────┘
```

### Current Implementation State

| Layer | Status | Notes |
|-------|--------|-------|
| Presentation | Scaffold complete | 4 pages with proper navigation |
| API Gateway | Health endpoint only | No business endpoints implemented |
| Business Logic | Not implemented | Empty placeholder |
| Integration | Not implemented | Dependencies declared, not connected |
| Persistence | Schema placeholder | No tables defined |

## Physical Structure

Key implementation files:
- Domain types: [CODE: ui/src/types/domain.ts]
- API client: [CODE: ui/src/lib/api-client.ts]
- Configuration: [CODE: ui/src/config/index.ts]
- Services: [CODE: ui/src/services/ideas-service.ts], [CODE: ui/src/services/scenarios-service.ts]
- API server: [CODE: api/main.go]
- CLI: [CODE: cli/app.go]

```
swarm-manager/
├── api/                      # Go API server
│   ├── main.go               # Server entry point, routes, health
│   └── go.mod                # Dependencies
│
├── cli/                      # Go CLI application
│   ├── main.go               # CLI entry point
│   ├── app.go                # Command definitions
│   └── install.sh            # Installation script
│
├── ui/                       # React frontend
│   ├── src/
│   │   ├── App.tsx           # Router and route definitions
│   │   ├── main.tsx          # Entry point
│   │   ├── components/
│   │   │   ├── layout/       # MainLayout (tabs, nav)
│   │   │   └── ui/           # shadcn components (button, tabs)
│   │   ├── pages/            # Page components per tab
│   │   │   ├── IdeasPage.tsx
│   │   │   ├── ScenariosPage.tsx
│   │   │   ├── RecommendationsPage.tsx
│   │   │   └── SettingsPage.tsx
│   │   ├── lib/              # Utilities (api client)
│   │   └── consts/           # Constants (selectors)
│   └── server.js             # Production static server
│
├── ideas/                    # Git-tracked idea backlog (empty)
│
├── requirements/             # PRD-linked requirement modules
│   ├── index.json
│   └── NN-target-name/module.json
│
├── initialization/           # Database setup
│   └── postgres/schema.sql   # (placeholder)
│
├── docs/                     # Documentation
│   ├── concepts/             # Mental models, architecture
│   ├── internal/             # Seams, implementation details
│   ├── PROGRESS.md           # Development log
│   ├── PROBLEMS.md           # Known issues
│   └── RESEARCH.md           # Research notes
│
├── bas/                      # Behavior-driven test cases
│   ├── cases/                # Per-target test cases
│   ├── flows/                # User journey validations
│   └── registry.json         # Test registry
│
├── .vrooli/                  # Vrooli configuration
│   ├── service.json          # Lifecycle, ports, dependencies
│   ├── endpoints.json        # API endpoint definitions
│   └── testing.json          # Test configuration
│
├── PRD.md                    # Product Requirements Document
├── README.md                 # Quick start guide
└── Makefile                  # Build/run commands
```

## Integration Strategy

### Required Integrations (P0)

| Integration | Purpose | Pattern |
|-------------|---------|---------|
| **agent-manager** | Spawn agents for automated work | All agent operations via this API |
| **ecosystem-manager** | Initialize/improve scenarios from ideas | Queue ideas → EM processes → Scenario exists |

### Optional Integrations (P1)

| Integration | Purpose | Data Flow |
|-------------|---------|-----------|
| knowledge-observatory | PROBLEMS.md management | Read problems → Display → Prune |
| visited-tracker | Context cleanup campaigns | View campaigns → Trigger cleanup |
| scenario-completeness-scoring | Completeness metrics | Fetch scores → Display in UI |
| app-issue-tracker | Issue management | Create/view issues → Track resolution |
| test-genie | Test execution | Trigger tests → Display results |
| prompt-manager | Prompt templates | Fetch prompts → Use in recommendations |

## Natural Boundaries

### UI Boundaries
- Each **page** (Ideas, Scenarios, Recommendations, Settings) owns its domain concerns
- **MainLayout** handles navigation chrome, pages handle content
- **selectors.ts** is the single source of truth for test identifiers

### API Boundaries
- `/health` - Infrastructure health (readiness, dependencies)
- `/api/v1/ideas/*` - Idea CRUD operations
- `/api/v1/scenarios/*` - Scenario catalog operations
- `/api/v1/recommendations/*` - Recommendation engine
- `/api/v1/settings/*` - User preferences

### Data Boundaries
- **Filesystem** (`ideas/`): Git-tracked idea specifications (source of truth)
- **PostgreSQL**: Metadata, settings, cached scenario state
- **Redis**: Real-time updates, queue state, session management

## Design Principles

1. **Integration-First**: All complex operations delegate to ecosystem-manager and agent-manager
2. **File-Based Ideas**: Ideas are stored as files for git tracking and human readability
3. **Three-State Recommendations**: Off (manual), Suggestions (review), YOLO (auto-approve)
4. **Progressive Enhancement**: P0 core → P1 integrations → P2 advanced features
