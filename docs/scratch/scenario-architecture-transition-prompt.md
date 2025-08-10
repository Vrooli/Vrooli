# Scenario-Based Architecture Transition - Investigation Prompt

## Context: The Architectural Revolution

You are working on Vrooli, a self-improving AI intelligence system that is undergoing a fundamental architectural transformation. The project is transitioning from a **monolithic full-stack application** to a **swarm of tiny, interconnected applications** orchestrated through scenarios.

### The Old Architecture (Being Phased Out)
- Single monolithic application with `docker-compose.yml`
- Fixed UI (packages/ui) and Server (packages/server) containers
- Traditional microservices approach with tight coupling
- Single deployment unit for all functionality

### The New Architecture (Being Implemented)
- Multiple small, focused applications generated from **scenarios**
- Each scenario is a complete business application worth $10K-50K
- Applications share a common pool of local resources
- Scenarios can be bundled together to create custom business solutions

## Understanding Scenarios: The Dual-Purpose Architecture

Scenarios serve a revolutionary dual purpose:
1. **Integration Testing**: Validate that resources work together correctly
2. **Business Templates**: Complete blueprints for deployable SaaS applications

Located in `/scripts/scenarios/core/`, there are 23 production-ready scenarios including:
- `research-assistant`: AI research platform with Windmill UI, n8n workflows, SearXNG
- `campaign-content-studio`: Marketing automation platform
- `secure-document-processing`: Privacy-focused document handling
- `image-generation-pipeline`: AI creative workflows
- And 19 more business-ready applications

## The Resource Sharing Innovation

### Critical Architectural Insight
Resources in `/scripts/resources/` are **shared infrastructure** that multiple applications leverage simultaneously:

```
[Shared Resource Pool]
├── PostgreSQL (shared DB, different schemas per app)
├── n8n (shared automation, different workflow folders)
├── Redis (shared event bus for inter-app communication)
├── Ollama (shared AI models across all apps)
├── MinIO (shared storage with different buckets)
└── 25+ more resources...

     ↓ Shared by ↓

[App Bundle Example]
├── real-estate-lead-generator/
├── credentials-manager/
├── admin-dashboard/
├── generic-lead-tools/
└── real-estate-workflows/
```

### Why This Changes Everything
- **Zero redundancy**: One Ollama instance serves ALL apps
- **Cross-app intelligence**: Apps share learned patterns through resources
- **Modular business solutions**: Compose complex solutions from simple apps
- **Unified management**: Same scripts manage the entire swarm

## The Bundling Vision

When deploying for a client, you can **bundle multiple scenarios** into a custom business operating system:
1. Each app in the bundle shares the same resource pool
2. Apps communicate through shared infrastructure (Redis events, shared DB)
3. All managed with unified commands (`./scripts/main/setup.sh`, `develop.sh`, etc.)
4. Generated apps are managed *exactly* like the main Vrooli app

## Current Implementation Status

### Working Components
- **Scenario-to-App Converter**: `/scripts/scenarios/tools/scenario-to-app.sh` generates standalone apps
- **Resource Management**: Individual resources have self-contained `manage.sh` scripts
- **Service.json Configuration**: Hierarchical configuration system with inheritance
- **Injection System**: `/scripts/scenarios/injection/` handles initialization data

### Legacy Components to Mine
The monolithic codebase contains valuable components that need extraction/adaptation:

1. **Platform Infrastructure** (`platforms/`)
   - CLI with workspace management
   - Electron desktop application
   - Browser extension
   - OG worker for social cards

2. **Deployment Systems**
   - Kubernetes manifests and Helm charts (`k8s/`)
   - Docker configurations
   - CI/CD pipelines

3. **Development Tools**
   - Build scripts (`scripts/main/`)
   - Testing infrastructure
   - Environment management

4. **Documentation** (`docs/`)
   - Contains useful patterns mixed with outdated Vrooli-specific info
   - Deployment guides that could be generalized

5. **Public Assets** (`public/`)
   - Icons, images, and static resources

## The Challenge

The monolithic code contains years of valuable work that shouldn't be deleted but needs careful extraction and generalization. The goal is to:
1. Preserve valuable patterns and infrastructure
2. Extract reusable components for scenario development
3. Create documentation for the new architecture
4. Enable AI agents to generate and deploy new scenarios autonomously

## Required Investigation

Please conduct a deep investigation of the following areas:

### 1. Read Core Documentation
- `/docs/architecture/scenario-conversion-implementation-plan.md`
- `/docs/architecture/unified-initialization-plan.md`
- `/docs/architecture/RECURSIVE_IMPROVEMENT.md`
- `/docs/resources/` (all resource documentation)
- `CLAUDE.md` (project instructions and vision)

### 2. Examine Scenario Infrastructure
- `/scripts/scenarios/core/` (all 23 scenarios)
- `/scripts/scenarios/tools/scenario-to-app.sh` (conversion mechanism)
- `/scripts/scenarios/injection/` (initialization system)
- `.vrooli/service.json` files (configuration system)

### 3. Analyze Resource System
- `/scripts/resources/` (all resource providers)
- Resource contracts in `/scripts/resources/contracts/`
- Resource testing in `/scripts/resources/tests/`
- Port registry and management

### 4. Study Legacy Components
- `/platforms/` (all platform implementations)
- `/packages/server/src/services/` (service architecture)
- `/packages/ui/` (UI components that could be extracted)
- `/scripts/main/` (lifecycle management scripts)

### 5. Review Deployment Infrastructure
- `/k8s/` (Kubernetes configurations)
- Docker configurations
- CI/CD pipelines in `.github/workflows/`
- Environment management scripts

### 6. Understand the Current State
- Recent commits and changes
- Modified files in git status
- Work in progress in `/docs/tasks/`

## Deliverable: 10 Specific Recommendations

After your investigation, provide **10 specific, actionable recommendations** for progressing toward the ideal state. Each recommendation should:

1. **Be concrete and specific** (not vague suggestions)
2. **Include file paths** where changes would occur
3. **Estimate complexity** (simple/medium/complex)
4. **Explain the business value** delivered
5. **Identify dependencies** on other recommendations

### Focus Areas for Recommendations:
- Extracting valuable legacy components for reuse
- Improving scenario generation and bundling
- Standardizing resource sharing patterns
- Creating documentation from legacy code
- Enabling AI agent scenario generation
- Simplifying deployment across targets (Docker, K8s, Desktop, etc.)
- Preserving valuable business logic and patterns
- Building the bundling/composition system
- Migrating existing functionality to scenarios
- Creating development tools for scenario creators

## Investigation Approach

1. **Start with the vision**: Understand the recursive improvement philosophy
2. **Map the current state**: What exists, what works, what's in progress
3. **Identify extraction opportunities**: What legacy code provides immediate value
4. **Find the gaps**: What's missing for the scenario-based future
5. **Prioritize by impact**: What delivers the most value soonest

## Critical Context

Remember:
- Every scenario becomes a **permanent capability** the system can use forever
- Scenarios enable **recursive improvement** where solutions build on solutions
- The goal is **autonomous business generation** by AI agents
- This isn't just refactoring - it's building **self-improving intelligence**

## IMPORTANT INSTRUCTION

**DO NOT change anything yet - I will carefully review the plan and let you know which suggestions to proceed with. Ultra think**

Your investigation should be thorough, your analysis should be deep, and your recommendations should be transformative. This is not incremental improvement - this is architectural revolution.

---

*This prompt captures the full context of the Vrooli scenario-based architecture transition. Use this for all future sessions working on this transformation.*