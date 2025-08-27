# Vrooli Development Documentation

Comprehensive development guide for the Vrooli platform. For quick reference, see [/CLAUDE.md](/CLAUDE.md).

## Table of Contents
- [Project Overview](#project-overview)
- [Technology Stack](#technology-stack)
- [Architecture Details](#architecture-details)
- [Development Guidelines](#development-guidelines)
- [Task Management System](#task-management-system)
- [Memory Management](#memory-management)
- [Testing Guide](#testing-guide)
- [Common Tasks](#common-tasks)
- [Emergent Capabilities](#emergent-capabilities)
- [Documentation Structure](#documentation-structure)

## Project Overview

Vrooli is a resource orchestration platform for generating complete business applications from customer requirements. It features dual-purpose scenarios that serve as both integration tests AND $10K-50K revenue applications, powered by a three-tier AI architecture.

### Key Features
- Scenario-based business application generation
- Local resource orchestration (30+ services)
- Dual-purpose architecture (test + revenue)
- Meta-scenario self-improvement
- Privacy-first local execution

## Technology Stack

### Frontend
- **Framework**: React + TypeScript
- **UI Library**: Material-UI (MUI)
- **Build Tool**: Vite
- **Features**: PWA-enabled, responsive design
- **State Management**: Custom stores with localStorage persistence

### Backend
- **Runtime**: Node.js + TypeScript
- **Framework**: Express
- **Database**: PostgreSQL with pgvector extension
- **ORM**: Prisma
- **Cache**: Redis
- **Queue**: Bull (Redis-based)

### AI Integration
- **Providers**: OpenAI, Anthropic, Mistral, Google
- **Protocol**: MCP (Model Context Protocol)
- **Architecture**: Three-tier execution system
- **Features**: Multi-provider support, rate limiting, cost tracking

### Infrastructure
- **Containers**: Docker, Docker Compose
- **Orchestration**: Kubernetes with Helm charts
- **Secrets**: HashiCorp Vault
- **CI/CD**: GitHub Actions
- **Package Manager**: pnpm with workspaces

## Architecture Details

### Three-Tier AI Architecture

```mermaid
graph TD
    T1[Tier 1: Coordination Intelligence<br/>🧠 Strategic Planning<br/>📊 Resource Allocation<br/>👥 Swarm Management]
    T2[Tier 2: Process Intelligence<br/>🔄 Task Decomposition<br/>🗺️ Routine Navigation<br/>📈 Execution Monitoring]
    T3[Tier 3: Execution Intelligence<br/>⚡ Direct Task Execution<br/>🔧 Tool Integration<br/>📝 Context Management]
    
    T1 -->|Orchestrates| T2
    T2 -->|Executes via| T3
    T3 -->|Events| EB[Event Bus<br/>📡 Redis-based]
    EB -->|Feedback| T1
    
    classDef tier1 fill:#e3f2fd,stroke:#1565c0,stroke-width:2px
    classDef tier2 fill:#e8f5e8,stroke:#2e7d32,stroke-width:2px
    classDef tier3 fill:#fff3e0,stroke:#e65100,stroke-width:2px
    classDef eventbus fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    
    class T1 tier1
    class T2 tier2
    class T3 tier3
    class EB eventbus
```

### Key Architectural Patterns
- **Event-Driven Communication**: Redis-based event bus for inter-service communication
- **Resource Management**: Coordinated resource allocation across tiers
- **Error Handling**: Structured error propagation with circuit breakers
- **State Management**: Distributed state with Redis caching
- **Security Boundaries**: Isolated execution environments for each tier

### Package Structure
```
/packages/
  /ui/            # React frontend application (PWA-enabled)
  /server/        # Express backend API with AI execution tiers
  /shared/        # Shared types and utilities
  /jobs/          # Background job processing (cron schedules)
  /extension/     # Browser extension
  /og-worker/     # OpenGraph worker (Cloudflare)
  /postgres/      # PostgreSQL container configuration
/scripts/         # Bash automation scripts with comprehensive tooling
/docs/           # Comprehensive documentation including architecture
/k8s/            # Kubernetes/Helm configurations for deployment
```

## Development Guidelines

### Database Operations
- Use Prisma migrations for schema changes: `cd packages/server && pnpm prisma migrate dev`
- Access Prisma Studio: `cd packages/server && pnpm prisma studio`
- Generate Prisma client after schema changes: `cd packages/server && pnpm prisma generate`

### Environment Variables
- Development: `.env` files in package directories
- Production: Managed via HashiCorp Vault
- Never commit sensitive data; use Vault for secrets

### Import Requirements
- **IMPORTANT**: All TypeScript imports MUST include the `.js` extension (e.g., `import { foo } from "./bar.js"`)
- This is a hard requirement for the testing framework to work correctly
- Do NOT remove `.js` extensions from imports, even if TypeScript complains
- The build system is configured to handle `.js` extensions in TypeScript files

### Error Handling
- Use structured error types from `@vrooli/shared`
- Implement proper error boundaries in React components
- Log errors with appropriate severity levels
- Use circuit breakers for external service calls

### Performance Considerations
- Implement pagination for list endpoints
- Use Redis caching for frequently accessed data
- Optimize database queries with proper indexes
- Use React.memo and useMemo for expensive computations

## Task Management System

### Task Management Commands

#### **Organize Next Task**
**Keywords:** _"organize next task," "structure task," "clarify next task," "organize backlog," "prepare next task"_

Process the **first unstructured task** in the [backlog.md](tasks/backlog.md):
1. **Explore the Codebase** - Search relevant files and analyze existing implementations
2. **Clarify and Research** - Ask targeted questions if unclear
3. **Decide on Splitting Tasks** - Split complex tasks into focused subtasks
4. **Refine and Document** - Follow established task template with clear descriptions
5. **Finalize** - Remove original entry and flag any tasks needing more information

#### **Start Next Task**
**Keywords:** _"start next task," "pick task," "begin task," "go," "work next," "start working"_

- Select highest-priority task from backlog
- Explore codebase to determine implementation strategies
- Draft and present brief implementation plan for confirmation
- Set task status to **IN_PROGRESS** upon confirmation
- Wait for explicit confirmation before marking **DONE**

#### **Update Task Status**
**Keywords:** _"update task statuses," "refresh tasks," "task progress update," "update backlog"_

Review and update:
- Status (**TODO/IN_PROGRESS/BLOCKED/DONE**)
- Progress indicators
- Newly identified blockers or dependencies

#### **Research**
**Keywords:** _"research," "investigate," "explore topic," "find info on," "deep dive"_

Perform in-depth research on specified topics:
1. **Define Scope** - Clarify research goals in relation to project architecture
2. **Information Gathering** - Use web search for comprehensive information
3. **Analyze and Synthesize** - Extract key concepts and integration points
4. **Document Findings** - Store detailed findings in `/docs/scratch/` for reference
5. **Present Results** - Summarize findings with key links and next steps

## Memory Management

You have no persistent memory between sessions. **After every memory reset, rely solely on files in the `/docs` folder** as your long-term memory. Reading **ALL** relevant `/docs` files at the start of every task is mandatory.

### Core Documentation Files (Always Required):
- **[context.md](context.md)** - Project purpose, goals, and business rationale
- **[decisions.md](decisions.md)** - Major project decisions and justifications
- **[risks.md](risks.md)** - Technical, strategic, operational risks and mitigation
- **[roadmap.md](roadmap.md)** - Project milestones and future vision
- **[tools.md](tools.md)** - Available commands and tools

### Task Management (`/docs/tasks/` folder):
- **[active.md](tasks/active.md)** - Tasks currently underway
- **[backlog.md](tasks/backlog.md)** - Unstructured notes and ideas awaiting research
- **[staged.md](tasks/staged.md)** - Clarified, researched tasks ready to start
- **[completed.md](tasks/completed.md)** - Finished tasks with outcomes
- **[failed.md](tasks/failed.md)** - Abandoned tasks with reasons and lessons

### Temporary Working Files (`/docs/scratch/` folder):
- Use for temporary notes, research results, or drafts during current task
- Transfer important information to permanent documentation before completing tasks

### Documentation Guidelines

#### Visual Enhancement
- Use **Mermaid diagrams** for workflows, architectures, and decision trees
- Include **emojis** in headings for visual cues (🚀, 💡, ⚠️, 🎯, ⚙️, 📚)
- Always specify language for code blocks (```typescript, ```bash, ```sql)
- Use **bold** for important concepts, *italics* for emphasis, `backticks` for code

#### Content Standards
- Start with Table of Contents for documents >100 lines
- Include practical examples and real-world context
- Cross-reference related documentation using relative paths
- Document error scenarios and troubleshooting steps
- Keep information up-to-date and actionable

## Testing Guide

### Testing Approach
- Write tests alongside code in `__test` directories
- Use descriptive test names following pattern: `should [expected behavior] when [condition]`
- **IMPORTANT**: Use testcontainers for Redis and PostgreSQL - DO NOT mock these databases
  - More computationally expensive but MUCH more reliable than mocks
  - See `packages/server/src/__test/setup.ts` for the testcontainer setup
  - Integration tests should use real database connections via testcontainers
- Mock external APIs and services (LLM providers, Stripe, etc.) but not core infrastructure
- Aim for >80% code coverage
- Testing framework: Vitest

## Common Tasks

### Adding a New API Endpoint
1. Define types in `packages/shared/src/api/`
2. Add validation schema in `packages/shared/src/validation/`
3. Implement endpoint in `packages/server/src/endpoints/`
4. Add tests in corresponding `__test` directory
5. Update API documentation if needed

### Working with AI Services
1. AI providers configured in `packages/server/src/services/llm/`
2. Use the LLM service abstraction for provider-agnostic calls
3. Implement rate limiting and cost tracking
4. Handle provider-specific errors gracefully

### Debugging
- Server logs: Check Docker container logs or terminal output
- Frontend debugging: Use React Developer Tools
- Database queries: Enable Prisma query logging with `DEBUG=prisma:query`
- Network issues: Check Redis connection and event bus

### All Available Commands

```bash
# Development Environment
./scripts/manage.sh setup --target docker          # Initial setup
vrooli develop --target docker --detached yes      # Start detached
vrooli develop --target docker --detached no       # Start interactive
vrooli develop --target docker --services "server ui redis postgresql"  # Specific services

# Testing
cd packages/server && pnpm test                   # Run all tests
cd packages/server && pnpm test -- path/to/test.test.ts  # Specific test
pnpm run test:coverage                           # All tests with coverage

# Building and Type Checking
pnpm run build                                   # Build all packages
pnpm run type-check                              # Check types across all packages
pnpm run lint                                   # Run linter
pnpm run lint -- --fix                         # Lint with auto-fix

# Database Operations
cd packages/server && pnpm prisma migrate dev   # Run migrations
cd packages/server && pnpm prisma studio       # Visual database editor
cd packages/server && pnpm prisma generate     # Generate client after schema changes
cd packages/server && pnpm prisma db seed      # Seed database

# Package-specific commands
cd packages/[package-name] && pnpm dev          # Start dev server
cd packages/[package-name] && pnpm build        # Build for production
cd packages/[package-name] && pnpm test         # Run tests
cd packages/[package-name] && pnpm lint         # Run linter
cd packages/[package-name] && pnpm type-check   # Run TypeScript type checking
```

## Emergent Capabilities

**IMPORTANT**: Many advanced capabilities in Vrooli are **emergent** - they arise from resource orchestration and scenario deployment, NOT from built-in code. **Do not attempt to build code for these capabilities** as they are designed to emerge through scenario combinations and meta-scenario intelligence.

### What Are Emergent Capabilities?
- **Resource-Orchestrated**: Capabilities emerge from novel combinations of local resources (databases, AI models, automation platforms)
- **Scenario-Driven**: Complex applications emerge from combining and extending existing scenario templates
- **Self-Improving**: Meta-scenarios continuously enhance the platform's ability to generate better scenarios
- **Business-Focused**: Capabilities target real revenue generation ($10K-50K applications) rather than just technical demos
- **Deployment-Validated**: Every capability is proven through actual business deployments and customer usage

### Examples of Emergent (NOT Built-In) Capabilities:
- **Complex Business Applications**: Multi-resource scenarios combining AI models, databases, automation, and UIs
- **Self-Improvement Systems**: Meta-scenarios that generate new scenarios, debug deployments, optimize resources
- **Industry-Specific Solutions**: Domain scenarios that leverage resource combinations for specialized markets
- **Resource Discovery**: Intelligent detection and integration of new local services into the ecosystem
- **Business Model Optimization**: Scenario analysis that identifies higher-value application opportunities
- **Deployment Intelligence**: Automated scaling, monitoring, and maintenance through orchestrated resources

### Key Principle:
Instead of building these features as code, the system provides:
1. **Scenario deployment capabilities** - Deploy complete business applications from validated templates
2. **Resource orchestration architecture** - Combine local services to create emergent business capabilities
3. **Meta-scenario mechanisms** - Self-improving scenarios that enhance platform capabilities
4. **Dual-purpose validation** - Every scenario proves both technical integration AND business viability

### When NOT to Code:
- Business application logic (deploy scenarios that orchestrate resources instead)
- Industry-specific workflows (create scenarios that combine existing resources)
- Self-improvement features (use meta-scenarios like Scenario Generator)
- Resource integration handlers (use resource orchestration patterns)
- Monitoring and optimization systems (deploy System Monitor and App Issue Tracker scenarios)

See [architecture/execution/emergent-capabilities/README.md](architecture/execution/emergent-capabilities/README.md) for comprehensive details.

## Documentation Structure

### 📚 **Core Documentation**
- **[Contributing Guidelines](CONTRIBUTING.md)** - How to contribute to Vrooli
- **[Context](context.md)** - Project vision and core functionality
- **[Decisions](decisions.md)** - Major architectural decisions
- **[Risks](risks.md)** - Risk assessment and mitigation
- **[Roadmap](roadmap.md)** - Future development plans
- **[Tools](tools.md)** - Development tools and commands

### 🏗️ **Architecture Documentation** (`/docs/architecture/`)
- **[AI Implementation Guide](architecture/ai-implementation-guide.md)** - Practical three-tier architecture implementation
- System design and architectural decisions
- Subdirectories: `execution/`, `external-integrations/`, `data/`, `core-services/`, `api-gateway/`, `client/`
- Key docs: Three-tier architecture, event-driven patterns, emergent capabilities

### 🔒 **Security Documentation** (`/docs/security/`)
- **[Security Overview](security/README.md)** - Comprehensive security guide
- **[Threat Model](security/threat-model.md)** - Risk analysis and attack scenarios
- **[Best Practices](security/best-practices.md)** - Secure coding and operational guidelines
- **[Incident Response](security/incident-response.md)** - Security incident handling procedures

### 🌐 **API Documentation** (`/docs/api/`)
- **[API Overview](api/README.md)** - Complete API reference with webhooks and enterprise patterns
- **[Authentication](api/authentication.md)** - Auth flows and security
- **[Endpoints](api/endpoints/)** - Detailed endpoint documentation
- **[Examples](api/examples/)** - Request/response examples and use cases

### 🗄️ **Data Model Documentation** (`/docs/data-model/`)
- **[Data Architecture](data-model/README.md)** - Database design and entity relationships
- **[Entities](data-model/entities/)** - Individual entity documentation
- **[Data Dictionary](data-model/data-dictionary.md)** - Field definitions and constraints
- **[Performance](data-model/performance.md)** - Query optimization and indexing

### 🧪 **Testing Documentation** (`/docs/testing/`)
- `test-strategy.md` - Overall testing approach
- `test-plan.md` - Detailed test planning
- `test-execution.md` - How to run tests
- `writing-tests.md` - Best practices for test writing
- `defect-reporting.md` - Bug reporting process

### 🛠️ **DevOps Documentation** (`/docs/devops/`)
- `server-deployment.md` - Deployment procedures
- `kubernetes.md` - K8s configuration
- `ci-cd.md` - CI/CD pipeline details
- `environment-management.md` - Environment configuration
- `troubleshooting.md` - Common issues and solutions

### ⚙️ **Setup Documentation** (`/docs/setup/`)
- `prerequisites.md` - System requirements
- `repo_setup.md` - Repository setup
- `working_with_docker.md` - Docker usage
- Various service setup guides (OAuth, Stripe, S3, etc.)

### 🎨 **UI Documentation** (`/docs/ui/`)
- Frontend architecture and guidelines
- Performance optimization
- PWA configuration
- Design system documentation

### 📡 **Server Documentation** (`/docs/server/`)
- Backend architecture
- API comprehensive guide
- Database migration procedures

### 🏭 **Operations Documentation** (`/docs/operations/`)
- **[Production Guide](operations/production-guide.md)** - Complete deployment, monitoring, and operations guide

### 📚 **User Guide** (`/docs/user-guide/`)
- **[Interactive Tutorial](user-guide/tutorial/)** - Step-by-step platform learning
- **[Video Scripts](user-guide/video-scripts/)** - Landing page video content
- **[Legacy Documentation](user-guide/old/)** - Preserved reference materials

### 📋 **Task Management** (`/docs/tasks/`)
- **[Active Tasks](tasks/active.md)** - Currently in progress
- **[Backlog](tasks/backlog.md)** - Unstructured tasks awaiting organization
- **[Staged Tasks](tasks/staged.md)** - Ready-to-start tasks
- **[Completed Tasks](tasks/completed.md)** - Finished work
- **[Failed Tasks](tasks/failed.md)** - Abandoned tasks with lessons learned

## Security Guidelines

- All external URLs must be validated before use
- Implement proper authentication/authorization checks
- Sanitize user inputs, especially for database queries
- Use prepared statements/parameterized queries
- Follow OWASP guidelines for web security

## Performance Optimization

- Database queries should use appropriate indexes
- Implement request caching where appropriate
- Use connection pooling for database connections
- Optimize bundle sizes with code splitting
- Monitor memory usage in background jobs

## Script Utilities

The `/scripts/` directory contains comprehensive bash scripts:
- Use `--help` flag with any script for documentation
- Scripts support various targets (docker, k8s, local)
- Environment detection and validation built-in
- Automatic dependency installation when needed

## Common Errors & Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| `Module not found: Error: Can't resolve './file'` | Missing .js extension | Add `.js` to import: `'./file.js'` |
| `Cannot find container vrooli_postgresql_1` | Docker not running | Run `./scripts/manage.sh setup --target docker` |
| `Invalid prisma.user invocation` | Schema out of sync | Run `cd packages/server && pnpm prisma generate` |
| `ECONNREFUSED 127.0.0.1:6379` | Redis not running | Start dev environment with scripts |
| `Type error in test file` | Wrong import path | Use relative imports with `.js` extension |
| `Test timeout` | Database container slow | Increase timeout or use `--runInBand` |
| `Cannot read properties of undefined` | Missing await | Ensure async operations are awaited |
| `ER_BAD_FIELD_ERROR` | Database migration needed | Run `pnpm prisma migrate dev` |

## Agent Workflows

### Initialization Workflow
Every task session begins by reading **ALL** Memory Bank files:
1. Read ALL relevant `/docs` files
2. Verify completeness and context clarity
3. Decide if ready to proceed or need clarifications

### Task Execution Workflow
1. Check & Read Memory Bank
2. Update docs if needed
3. Perform task
4. Update task status & outcomes
5. Summarize & document results

### Key Rules
- **Always** start by reading **every** required memory file
- Maintain file integrity and organization rigorously
- Document decisions, changes, and insights clearly
- Update documentation regularly and proactively
- Use `/docs/scratch/` for temporary working information