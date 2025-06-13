# CLAUDE.md

This file provides essential guidance to Claude Code (claude.ai/code) when working with this repository.

## ⚡ Critical Rules - READ FIRST
1. **Imports**: 
   - Always use `.js` extensions in TypeScript imports (e.g., `import { foo } from "./bar.js"`)
   - For monorepo packages, use only the package name: `@vrooli/shared`, `@vrooli/server`, etc.
   - NEVER use deep imports like `@vrooli/shared/id/snowflake.js` - import from the package root
2. **Testing**: Use testcontainers for Redis/PostgreSQL - NEVER mock core infrastructure
3. **Emergent Features**: Don't code security/optimization/learning features - these emerge from agents
4. **Files**: Always prefer editing existing files over creating new ones
5. **Dependencies**: Never install packages without explicit permission
6. **Documentation**: Read `/docs/` files at session start for context

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

# Run tests
pnpm test                                            # Run all tests (shell, unit, run)
pnpm test:shell                                      # Test shell scripts only
pnpm test:unit                                       # Unit tests in all packages
cd packages/server && pnpm test                      # Server tests
cd packages/server && pnpm test-watch                # Watch mode
cd packages/server && pnpm test-coverage             # With coverage
cd packages/jobs && pnpm test                        # Jobs tests

# Type checking (recommended: check smallest number of files possible due to large project size)
cd packages/server && pnpm run type-check            # Server package only  
cd packages/ui && pnpm run type-check                # UI package only
cd packages/shared && pnpm run type-check            # Shared package only
cd packages/server && tsc --noEmit src/file.ts       # Single file
cd packages/server && tsc --noEmit src/file1.ts src/file2.ts  # Multiple files
cd packages/server && tsc --noEmit src/folder/**/*.ts         # Folder/pattern

# Linting
pnpm run lint                                        # Run all linters (JS + shell)
pnpm run lint:js                                     # ESLint for TypeScript
pnpm run lint:shell                                  # ShellCheck for scripts
cd packages/server && pnpm run lint                  # Server linting (slow ~30s+)
cd packages/ui && pnpm run lint                      # UI linting

# Database commands
cd packages/server && pnpm prisma generate           # After schema changes
cd packages/server && pnpm prisma studio             # Visual database editor
cd packages/server && pnpm prisma migrate deploy     # Deploy migrations

# Alternative development commands
pnpm run develop                                     # Alternative to develop script
docker compose up --build                            # Direct Docker alternative
```

> **Note**: When writing tests, make sure you're writing them to test against the DESIRED/EXPECTED behavior, not the actual implementation. This is important for the test to be useful and not just a checkmark.

## ❌ Common Pitfalls
- DON'T remove `.js` extensions from imports
- DON'T mock Redis or PostgreSQL in tests
- DON'T implement emergent capabilities as code
- DON'T use `__tests` directory (use `__test`)
- DON'T skip reading memory files at session start

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

## 🔧 Quick Error Reference
| Error | Solution |
|-------|----------|
| `Missing script: type-check` | No root-level type-check - use package-specific commands |
| Tests failing with `jest is not defined` | Tests using vitest, not jest - check test setup |
| `cd: packages/ui: No such file or directory` | Ensure you're in project root `/root/Vrooli` first |


---

**For detailed documentation, development guidelines, and comprehensive examples, see [/docs/README.md](/docs/README.md)**