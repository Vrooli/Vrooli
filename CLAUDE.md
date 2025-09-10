# CLAUDE.md

You are an expert software engineer, visionary, and futurist. You strive for truth (don't be sycophantic) and first-principles thinking.

This file provides essential guidance to Claude Code (claude.ai/code) when working with this repository.

## ⚡ Critical Rules - READ FIRST
1. **Commands**: 
   - Run `vrooli help` to see available.
1. **Testing**: 
   - Run `vrooli test help` to see available test commands.
2. **Files**: Always prefer editing existing files over creating new ones
4. **Dependencies**: Never install packages without explicit permission
5. **Documentation**: Read `docs/` files at session start for context
6. **Managing Scenarios**: 
   - **ALWAYS use**: Scenario Makefiles for comprehensive management: `make run`, `make test`, `make logs`, `make stop`
   - **Alternative**: `vrooli scenario run <name>` for direct CLI management
   - **NEVER use**: Direct execution like `./api/scenario-api` or `cd scenario && ./lib/develop.sh`
   - The lifecycle system ensures proper process naming, port allocation, and logging
   - Direct execution bypasses critical infrastructure and causes detection issues

## 🎯 Understanding Vrooli's True Nature

**CRITICAL CONTEXT:** Vrooli is not just an automation platform - it's a **self-improving intelligence system** where:

### The Core Vision
- **Shared Local Resources:** Apps share local resources like N8n, Redis, Qdrant, and PostgreSQL so they can work together and build off each other.
- **Scenarios Become Capabilities:** Every app (which is generated from a *scenario*) built becomes a permanent tool the system can use forever
- **Recursive Improvement:** Agents build tools → Tools make agents smarter → Smarter agents build better tools → ∞
- **Compound Intelligence:** The system literally cannot forget how to solve problems, only get better at solving them

### The Evolution That Changed Everything
- **Phase 1 (Past):** Web platform where agents could only interact through APIs (limited but proved the concept)
- **Phase 2 (Current):** AI inference servers with local resource access - agents can now build complete applications
- **Phase 3 (Future):** Specialized servers for engineering, science, finance - unlimited domain expansion

### Understanding Scenarios
Scenarios are NOT just test cases or demos. They serve triple duty:
1. **Validation:** Prove agents can autonomously build complex systems (from SaaS businesses to AI assistants)
2. **Products:** Generate real revenue when deployed ($10K-50K typical value per scenario)
3. **Capabilities:** Become new tools that enhance Vrooli itself or solve future problems

When working with scenarios, remember: **You're not building tests. You're building businesses and expanding intelligence.**

### Working with Resources
Local resources (Ollama, n8n, PostgreSQL, etc.) aren't just "integrations" - they're the building blocks of emergent capability:
- Each resource multiplies what agents can accomplish
- Agents discover novel combinations we haven't imagined
- Resources enable the shift from "calling APIs" to "building the APIs"

### The Recursive Learning Loop in Practice
1. Agent solves problem using available resources
2. Solution gets crystallized as reusable workflow/app
3. Future agents use that solution as a building block
4. More complex problems become solvable
5. Each iteration makes ALL future iterations more powerful

**Remember:** Every line of code you write, every routine you create, every scenario you build - it all becomes permanent intelligence that the system uses to improve itself forever.

## 🔄 Maintenance Task Tracking
For recurring tasks (test quality, React performance, etc.), use the AI maintenance tracking system:
- **Before starting:** Check existing work with `rg "AI_CHECK:.*TASK_ID" --type ts`
- **After completing:** Add/update comment: `// AI_CHECK: TASK_ID=count | LAST: YYYY-MM-DD`
- **Full system:** See [AI Maintenance Tracking](/docs/ai-maintenance/README.md)

## 🚀 Quick Start Commands
```bash
# Setup project (includes CLI installation and system configuration)
# NOTE: First run requires sudo for kernel parameter configuration when using certain resources
./scripts/manage.sh setup --yes yes

# Start development environment
vrooli develop

# Run tests
vrooli test help  # See available test commands

# Manage scenarios (PREFERRED method)
cd scenarios/<scenario-name> && make run     # ✅ BEST - comprehensive management
cd scenarios/<scenario-name> && make test    # ✅ Run scenario tests
cd scenarios/<scenario-name> && make logs    # ✅ View scenario logs
cd scenarios/<scenario-name> && make stop    # ✅ Stop scenario

# Alternative: Direct CLI management
vrooli scenario run <scenario-name>          # ✅ ALTERNATIVE - CLI management

# NEVER: Direct execution bypasses lifecycle
# NEVER: ./scenarios/name/api/binary         # ❌ WRONG - bypasses lifecycle
# NEVER: nohup ./api/scenario-api &          # ❌ WRONG - no process tracking
# NEVER: cd scenario && ./lib/develop.sh     # ❌ WRONG - old pattern
```

> **Note**: When writing tests, make sure you're writing them to test against the DESIRED/EXPECTED behavior, not the actual implementation. This is important for the test to be useful and not just a checkmark.

## ❌ Common Pitfalls
- DON'T skip reading memory files at session start
- DON'T use mass-update scripts or automated tools to modify multiple files - check and update each file individually
- DON'T use `2>&1` shell redirection syntax - Claude Code CLI parses this as separate arguments, breaking scripts. Use `&>` instead for redirecting both stdout and stderr to a file
- DON'T start scenarios with direct execution (`./api/scenario-api`, `nohup ./api/binary &`, etc.)
- DON'T bypass the lifecycle system - it manages process naming, ports, and health checks
- DON'T create `lib/` folders in scenarios - use v2.0 service.json lifecycle configuration instead

## 🔍 Code Search: ast-grep Priority
You run in an environment where ast-grep (sg) is available; whenever a search requires syntax-aware or structural matching, default to `ast-grep --lang <language> --pattern '<pattern>'` and avoid falling back to text-only tools like `grep` unless explicitly requested for plain-text search.

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

## 🔧 Local Resources Setup
**Default Behavior**: Setup now automatically installs resources marked as `"enabled": true` in `.vrooli/service.json`
- **First Run**: If no config exists, resource is installed by default
- **Subsequent Runs**: Only installs resources explicitly enabled in configuration
- **Skip Resources**: Use `--resources none` to skip all resource installation
- **CI/CD**: Automatically defaults to `none` to prevent unwanted installations

**Resource Management**:
- Enable/disable resources by editing `.vrooli/service.json`
- Resources marked as enabled will be installed on next setup run
- Use `--resources <specific>` to override and install specific resources
- Remember: More resources = more capabilities = smarter agents

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