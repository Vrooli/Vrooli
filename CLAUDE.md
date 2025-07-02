# CLAUDE.md

You are an expert software engineer, visionary, and futurist. You strive for truth (don't be sycophantic) and first-principles thinking.

This file provides essential guidance to Claude Code (claude.ai/code) when working with this repository.

## ⚡ Critical Rules - READ FIRST
1. **Imports**: 
   - Always use `.js` extensions in TypeScript imports (e.g., `import { foo } from "./bar.js"`)
   - For monorepo packages, use only the package name: `@vrooli/shared`, `@vrooli/server`, etc.
   - NEVER use deep imports like `@vrooli/shared/id/snowflake.js` - import from the package root
2. **Testing**: 
   - Use testcontainers for Redis/PostgreSQL - NEVER mock core infrastructure
   - For UI components: Test behavior, not visuals (no CSS class checks) - use Storybook for visual testing
3. **Emergent Features**: Don't code security/optimization/learning features - these emerge from agents
4. **Files**: Always prefer editing existing files over creating new ones
5. **Linting**: After editing TypeScript files, run linting in the specific package (e.g., `cd packages/server && pnpm run lint --fix`)
6. **Dependencies**: Never install packages without explicit permission
7. **Documentation**: Read `/docs/` files at session start for context

## 🔄 Maintenance Task Tracking
For recurring tasks (test quality, React performance, etc.), use the AI maintenance tracking system:
- **Before starting:** Check existing work with `rg "AI_CHECK:.*TASK_ID" --type ts`
- **After completing:** Add/update comment: `// AI_CHECK: TASK_ID=count | LAST: YYYY-MM-DD`
- **Full system:** See [AI Maintenance Tracking](/docs/ai-maintenance/README.md)

## 🏗️ Architecture Overview
**Three-Tier AI System:**
- **Tier 1**: Coordination Intelligence (strategic planning, resource allocation)
- **Tier 2**: Process Intelligence (task orchestration, routine navigation)  
- **Tier 3**: Execution Intelligence (direct task execution, tool integration)
- **Event Bus**: Redis-based communication between tiers

## 📁 Key Locations
```
/packages/
  /ui/            # React frontend (PWA)
  /server/        # Express backend with AI tiers
  /shared/        # Shared types and utilities
  /jobs/          # Background job processing
  
Key Files:
- Types: packages/shared/src/api/types.ts
- Endpoints: packages/server/src/endpoints/logic/
- Tests: __test directories (NOT __tests)
- AI Tiers: packages/server/src/services/execution/tier[1-3]/
```

## 🚀 Quick Start Commands
```bash
# Start development environment
./scripts/main/develop.sh --target docker --detached yes

# Build specific artifacts
./scripts/main/build.sh --environment development --test no --lint no --bundles zip --artifacts docker
./scripts/main/build.sh --environment development --test no --lint no --bundles zip --artifacts k8s --version 2.0.0

# Deploy to Kubernetes
# IMPORTANT: Deployment uses the currently active kubectl context
# For production: Use KUBECONFIG=/root/Vrooli/k8s/kubeconfig-vrooli-prod.yaml 
# For development: The develop.sh script sets up minikube automatically
./scripts/main/deploy.sh --source k8s --environment dev --version 2.0.0    # Development deployment
./scripts/main/deploy.sh --source k8s --environment prod --version 2.0.0   # Production deployment

# Run tests
# ⚠️ IMPORTANT: Tests can take 3-5+ minutes. Always use extended timeouts for test commands
pnpm test                                            # Run all tests (shell, unit, run) - needs 5+ min timeout
pnpm test:shell                                      # Test shell scripts only
pnpm test:unit                                       # Unit tests in all packages - needs 5+ min timeout
cd packages/server && pnpm test                      # Server tests - needs 5+ min timeout
cd packages/server && pnpm test-watch                # Watch mode
cd packages/server && pnpm test-coverage             # With coverage - needs 5+ min timeout
cd packages/jobs && pnpm test                        # Jobs tests - needs 3+ min timeout

# Type checking (recommended: check smallest number of files possible due to large project size)
# ⚠️ IMPORTANT: Full package type-check can take 2-4+ minutes. Use extended timeouts
cd packages/server && pnpm run type-check            # Server package only - needs 4+ min timeout
cd packages/ui && pnpm run type-check                # UI package only - needs 3+ min timeout
cd packages/shared && pnpm run type-check            # Shared package only
cd packages/server && tsc --noEmit src/file.ts       # Single file
cd packages/server && tsc --noEmit src/file1.ts src/file2.ts  # Multiple files
cd packages/server && tsc --noEmit src/folder/**/*.ts         # Folder/pattern

# Linting
pnpm run lint                                        # Run all linters (JS + shell)
pnpm run lint:js                                     # ESLint for TypeScript
pnpm run lint:shell                                  # ShellCheck for scripts
cd packages/server && pnpm run lint --fix            # Server linting with auto-fix
cd packages/ui && pnpm run lint --fix                # UI linting with auto-fix
cd packages/shared && pnpm run lint --fix            # Shared linting with auto-fix
cd packages/jobs && pnpm run lint --fix              # Jobs linting with auto-fix

# Database commands
cd packages/server && pnpm prisma generate           # After schema changes
cd packages/server && pnpm prisma studio             # Visual database editor
cd packages/server && pnpm prisma migrate deploy     # Deploy migrations

# Kubernetes development commands (for local development only)
./scripts/main/develop.sh --target k8s-cluster       # Start local k8s dev environment - auto-installs kubectl, Helm, Minikube
./scripts/main/setup.sh --target k8s-cluster         # Setup local k8s cluster only - auto-installs kubectl, Helm, Minikube

# Alternative development commands
pnpm run develop                                     # Alternative to develop script
docker compose up --build                            # Direct Docker alternative - needs 5+ min timeout
```

> **Note**: When writing tests, make sure you're writing them to test against the DESIRED/EXPECTED behavior, not the actual implementation. This is important for the test to be useful and not just a checkmark.
> 
> **UI Testing Note**: For UI components, focus on testing user interactions, accessibility, and component behavior. Avoid testing CSS classes or visual styling - these are implementation details that should be verified through Storybook instead.

## ❌ Common Pitfalls
- DON'T remove `.js` extensions from imports
- DON'T mock Redis or PostgreSQL in tests
- DON'T implement emergent capabilities as code
- DON'T use `__tests` directory (use `__test`)
- DON'T skip reading memory files at session start
- DON'T use mass-update scripts or automated tools to modify multiple files - check and update each file individually

## 🎯 Task Management Commands
These commands can be invoked by using the keywords listed for each:

### **Organize Next Task**
**Keywords:** _"organize next task," "structure task," "clarify next task," "organize backlog," "prepare next task"_
- Process the first unstructured task in `/docs/tasks/backlog.md`
- Explore codebase, clarify requirements, split complex tasks
- Follow task template and move to staged tasks when ready

### **Start Next Task**
**Keywords:** _"start next task," "pick task," "begin task," "go," "work next," "start working"_
- Select highest-priority task from backlog
- Draft implementation plan and wait for confirmation
- Set status to IN_PROGRESS, mark DONE only after explicit confirmation

### **Update Task Status**
**Keywords:** _"update task statuses," "refresh tasks," "task progress update," "update backlog"_
- Review and update task statuses (TODO/IN_PROGRESS/BLOCKED/DONE)
- Update progress indicators and identify blockers

### **Research**
**Keywords:** _"research," "investigate," "explore topic," "find info on," "deep dive"_
- Perform in-depth research on specified topics
- Document findings in `/docs/scratch/` for reference
- Present summarized results with key links

### **Suggest New Tasks**
**Keywords:** _"suggest new tasks," "find new tasks," "discover tasks," "analyze for tasks," "task discovery"_
- Analyze `/README.md`, `/docs/tasks/backlog.md`, and `/docs/tasks/active.md`
- Identify gaps, improvements, and opportunities
- Suggest 10 potential tasks with brief descriptions
- Wait for user selection before adding to backlog

## 📚 Session Start Checklist
1. [ ] Read `/docs/context.md` for project overview
2. [ ] Read `/docs/tasks/active.md` for current work
3. [ ] Check git status for uncommitted changes
4. [ ] Review `/docs/scratch/` for previous session notes

## 🚢 Kubernetes Deployment Notes

### Local Development
**Auto-Installation**: The project automatically installs kubectl, Helm, and Minikube when targeting k8s-cluster
- `kubectl` - Latest stable version from Google's Kubernetes release API
- `Helm` - Package manager for Kubernetes
- `Minikube` - Local Kubernetes cluster for development

### Production Deployment
**Prerequisites**: Production cluster setup in DigitalOcean with required operators already installed
- Uses existing kubeconfig at `/root/Vrooli/k8s/kubeconfig-vrooli-prod.yaml`
- **IMPORTANT**: Deploy script uses whatever kubectl context is currently active
- Set `KUBECONFIG=/root/Vrooli/k8s/kubeconfig-vrooli-prod.yaml` for production deployments

**Required Operators** (auto-installed in dev, manual setup for prod):
- CrunchyData PostgreSQL Operator (PGO) v5.8.2
- Spotahome Redis Operator v1.2.4  
- Vault Secrets Operator (VSO)

**Deployment Readiness**: Use `./scripts/helpers/deploy/k8s-prerequisites.sh --check-only` to verify cluster setup

## 🔧 Quick Error Reference
| Error | Solution |
|-------|----------|
| `Missing script: type-check` | No root-level type-check - use package-specific commands |
| `cd: packages/ui: No such file or directory` | Ensure you're in project root `/root/Vrooli` first |
| `Command timed out after 2m 0.0s` | Increase timeout for long-running commands |
| `kubectl: command not found` | Run `./scripts/main/setup.sh --target k8s-cluster` to auto-install |

## ⏱️ Timeout Guidelines for Long-Running Commands
**Remember to set appropriate timeouts when running:**
- Test suites: Can take 15+ minutes in worst case scenarios (better to be safe than sorry)
- Type checking full packages: Can take 15+ minutes
- Building/compiling: Can take 10+ minutes (UI build alone takes 5-10 minutes due to 4444+ modules)
- Database migrations: Can take 3+ minutes
- Docker builds: Can take 20+ minutes
- UI build performance issue: vite build processes 4400+ modules, causing 5-10 minute build times

The default timeout is 2 minutes, which is often insufficient for these operations.


---

**For detailed documentation, development guidelines, and comprehensive examples, see [/docs/README.md](/docs/README.md)**