# Swarm Manager Architecture

## Mental Model

Swarm Manager is the **central command center** for the Vrooli scenario ecosystem. It orchestrates three primary domains:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           SWARM MANAGER                                  │
│ "The place where backlog becomes scenarios and scenarios become products" │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────┐   ┌─────────────┐    ┌──────────────┐    ┌─────────┐ │
│  │  BACKLOG     │──▶│  SCENARIOS  │───▶│RECOMMENDATIONS│◀───│INSIGHTS │ │
│  │ (idea/research│  │  (catalog)  │    │   (engine)   │    │(patterns)│ │
│  │  fix/execute)│   └─────────────┘    └──────────────┘    └─────────┘ │
│  └──────────────┘                                                     │
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
| **Backlog Item** | A tracked unit of work (idea, research, fix, execute) stored as git-tracked folders | backlog → researching → ready → queued → in_progress → completed/archived | [CODE: ui/src/types/domain.ts#BacklogItem] |
| **Scenario** | A deployed application in the Vrooli ecosystem | running, stopped, error, unknown | [CODE: ui/src/types/domain.ts#Scenario] |
| **Recommendation** | A system-generated suggestion for improvement | pending → approved/rejected | [CODE: ui/src/types/domain.ts#Recommendation] |
| **Insight** | Pattern-based observation about system health | N/A (informational) | (not yet implemented) |

### Key Flows

1. **Backlog (Idea) → Scenario Pipeline**
   ```
   Create Idea → Clarify → Suggest → Enhance → Queue for Processing → agent-manager executes → Scenario exists
   ```
   - Clarify writes `clarify/questions.json`, Suggest writes `suggest/suggestions.json`, Enhance writes `enhance/summary.md`.

2. **Research → Convert Flow**
   ```
   Create Research → Run Research Agent → research/summary.md → Convert to Idea/Fix/Execute → Next queue action
   ```

3. **Fix/Execute Processing**
   ```
   Create Fix/Execute → (Optional Research) → Queue for Processing → agent-manager applies changes
   ```

4. **Recommendation Engine Flow**
   ```
   Data Sources (PROBLEMS.md, completeness, tests) → Engine analyzes → Generates Recommendations → User approves/rejects
   ```

5. **Scenario Lifecycle Management**
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
│ BUSINESS LOGIC LAYER                                         │
│ Backlog/scenario/settings/recommendations/queue orchestration│
│ Implemented in internal/* handlers with validation + flows   │
├─────────────────────────────────────────────────────────────┤
│ INTEGRATION LAYER                                            │
│ Discovery-based adapters for agent-manager, ecosystem-manager│
│ Pattern: All agent work via agent-manager (never direct)     │
├─────────────────────────────────────────────────────────────┤
│ PERSISTENCE LAYER                                            │
│ Filesystem (ideas/, research/, fix/, execute/, .vrooli/settings.json, .vrooli/queue.json) │
└─────────────────────────────────────────────────────────────┘
```

### Current Implementation State

| Layer | Status | Notes |
|-------|--------|-------|
| Presentation | Functional | 4 pages wired with services and error handling |
| API Gateway | Implemented | Backlog/scenarios/settings/recommendations/queue endpoints |
| Business Logic | Implemented | CRUD + queue/research + recommendations engine |
| Integration | Implemented | Discovery-based agent-manager + ecosystem-manager clients |
| Persistence | Filesystem-first | Backlog items, settings, queue, recommendations stored on disk; scenario inventory sourced from Vrooli CLI |

## Physical Structure

Key implementation files:
- Domain types: [CODE: ui/src/types/domain.ts]
- API client: [CODE: ui/src/lib/api-client.ts]
- Configuration: [CODE: ui/src/config/index.ts]
- Services: [CODE: ui/src/services/backlog-service.ts], [CODE: ui/src/services/scenarios-service.ts]
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
│   │   │   ├── BacklogPage.tsx
│   │   │   ├── BacklogDetailsPage.tsx
│   │   │   ├── ScenariosPage.tsx
│   │   │   ├── RecommendationsPage.tsx
│   │   │   └── SettingsPage.tsx
│   │   ├── stores/           # Zustand stores for shared list state
│   │   ├── lib/              # Utilities (api client)
│   │   └── consts/           # Constants (selectors)
│   └── server.js             # Production static server
│
├── ideas/                    # Git-tracked backlog (idea)
├── research/                 # Git-tracked backlog (research)
├── fix/                      # Git-tracked backlog (fix)
├── execute/                  # Git-tracked backlog (execute)
│
├── requirements/             # PRD-linked requirement modules
│   ├── index.json
│   └── NN-target-name/module.json
│
├── initialization/           # Scenario initialization assets
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

All inter-scenario calls use `api-core/discovery` to resolve dynamic ports at runtime.

### Required Integrations (P0)

| Integration | Purpose | Pattern |
|-------------|---------|---------|
| **agent-manager** | Spawn agents for backlog processing and research | Discovery-resolved API URL |

### Optional Integrations (P1)

| Integration | Purpose | Data Flow |
|-------------|---------|-----------|
| knowledge-observatory | PROBLEMS.md management | Read problems → Display → Prune |
| visited-tracker | Context cleanup campaigns | View campaigns → Trigger cleanup |
| scenario-completeness-scoring | Completeness metrics | Fetch scores → Display in UI |
| app-issue-tracker | Issue management | Create/view issues → Track resolution |
| test-genie | Test execution | Trigger tests → Display results |
| prompt-manager | Prompt templates | Fetch prompts → Use in recommendations |
| ecosystem-manager | Scenario bootstrapping (future) | Discovery-resolved API URL |

## Natural Boundaries

### UI Boundaries
- Each **page** (Backlog, Scenarios, Recommendations, Settings) owns its domain concerns
- **MainLayout** handles navigation chrome, pages handle content
- **selectors.ts** is the single source of truth for test identifiers

### API Boundaries
- `/health` - Infrastructure health (readiness, dependencies)
- `/api/v1/backlog/*` - Backlog CRUD, research, queue, convert operations
- `/api/v1/scenarios/*` - Scenario catalog operations
- `/api/v1/recommendations/*` - Recommendation engine
- `/api/v1/settings/*` - User preferences

### Data Boundaries
- **Filesystem** (`ideas/`, `research/`, `fix/`, `execute/`, `.vrooli/settings.json`, `.vrooli/queue.json`): Source of truth for local state

## Design Principles

1. **Integration-First**: All complex operations delegate to ecosystem-manager and agent-manager
2. **File-Based Backlog**: Backlog items are stored as files for git tracking and human readability
3. **Three-State Recommendations**: Off (manual), Suggestions (review), YOLO (auto-approve)
4. **Progressive Enhancement**: P0 core → P1 integrations → P2 advanced features
