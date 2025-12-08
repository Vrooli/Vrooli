# AGENTS.md

You are an expert software engineer, visionary, and futurist. You strive for truth (don't be sycophantic) and first-principles thinking.

This file provides essential guidance to Claude Code (claude.ai/code) when working with this repository.

## ⚡ Critical Rules - READ FIRST
1. **Commands**: 
   - Run `vrooli help` to see available.
1. **Testing**: 
   - Use `vrooli scenario test <name>` (or test-genie) to run scenario tests.
2. **Files**: Always prefer editing existing files over creating new ones
4. **Dependencies**: Never install packages without explicit permission
5. **Documentation**: Run `vrooli info` at session start for the canonical project briefing
6. **Managing Scenarios**:
   - **ALWAYS use**: Scenario Makefiles for comprehensive management: `make start`, `make test`, `make logs`, `make stop`
   - **Alternative**: `vrooli scenario start <name>` for direct CLI management
   - **NEVER use**: Direct execution like `./api/scenario-api` or `cd scenario && ./lib/develop.sh`
   - The lifecycle system ensures proper process naming, port allocation, and logging
   - Direct execution bypasses critical infrastructure and causes detection issues

## 🎯 Understanding Vrooli's True Nature

### Key Definitions
- **Resources**: Core local services (AI/ML like claude-code, ollama; storage like postgres, redis, qdrant; development helpers like judge0, browserless, vault) that scenarios can compose.
- **Scenarios**: Full applications or microservices - with APIs, CLIs, and UIs - that combine resources and other scenarios to deliver reusable business capabilities.

**CRITICAL CONTEXT:** Vrooli is not just an automation platform - it's a **self-improving intelligence system** where:

### The Core Vision
- **Shared Local Resources:** Apps share local resources like Ollama, Redis, Qdrant, and PostgreSQL so they can work together and build off each other.
- **Scenarios Become Capabilities:** Every app (which is generated from a scenario) built becomes a permanent tool the system can use forever
- **Recursive Improvement:** Agents build tools → Tools make agents smarter → Smarter agents build better tools → ∞
- **Compound Intelligence:** The system literally cannot forget how to solve problems, only get better at solving them
- **Scenario-Based Business Model**: Scenarios target measurable value; deliverables can deploy directly, ship as SaaS, serve enterprise installs, or simply act as internal tools or microservices for other scenarios to leverage. Each scenario we complete should increase Vrooli's capabilities and/or be a new monetizable service

### The Evolution That Changed Everything
- **Phase 1 (Past):** Web platform where agents could only interact through APIs (limited but proved the concept)
- **Phase 2 (Current):** Physical server with local resource access - agents can now build complete applications by building off of existing resources and scenarios
- **Phase 3 (Future):** Specialized servers for engineering, science, finance. Hardware line where businesses and households can run their own specialized Vrooli server

### Understanding Scenarios
Scenarios are NOT just test cases or demos. They serve triple duty:
1. **Products:** Generate real revenue when deployed
2. **Validation:** Serve as implementation references for building future scenarios

3. **Capabilities:** Become new tools that enhance Vrooli itself or solve future problems

When working with scenarios, remember: **You're building businesses and expanding intelligence.**

### Deployment Vision
- Current deployments run via the Tier 1 local stack (full Vrooli installation + app-monitor Cloudflare tunnel).
- Future tiers (desktop, mobile, SaaS, enterprise) are documented in the [Deployment Hub](docs/deployment/README.md); consult it whenever considering packaging or delivery tasks.

### Working with Resources
Local resources (Ollama, PostgreSQL, etc.) aren't just "integrations" - they're the building blocks of emergent capability:
- Each resource multiplies what agents can accomplish
- Agents discover novel combinations we haven't imagined
- Resources enable the shift from "calling APIs" to "building the APIs"

### The Recursive Learning Loop in Practice
1. Agent solves problem using available resources
2. Solution gets crystallized as reusable scenario
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
vrooli scenario test <name>  # Run scenario test suite

# Manage scenarios (PREFERRED method)
cd scenarios/<scenario-name> && make start   # ✅ BEST - comprehensive management
cd scenarios/<scenario-name> && make test    # ✅ Run scenario tests
cd scenarios/<scenario-name> && make logs    # ✅ View scenario logs
cd scenarios/<scenario-name> && make stop    # ✅ Stop scenario

# Alternative: Direct CLI management
vrooli scenario start <scenario-name>        # ✅ ALTERNATIVE - CLI management

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

## 🔍 Available Tools
- **ast-grep (sg)**: For syntax-aware code search - default to `ast-grep --lang <language> --pattern '<pattern>'` over `grep` for structural matching
- **jq/yq**: For JSON/YAML processing
- **gofumpt**: Stricter Go formatting (superset of gofmt) - use `gofumpt -w .` to format Go code
- **golangci-lint**: Comprehensive Go linting - use `golangci-lint run` to check Go code quality and catch issues

## 📚 Session Start Checklist
1. [ ] Run `vrooli info` for the consolidated project overview

## 🔧 Local Resources Setup
**Default Behavior**: Setup now automatically installs resources marked as `"enabled": true` in `.vrooli/service.json`
- **First Run**: If no config exists, resource is installed by default
- **Subsequent Runs**: Only installs resources explicitly enabled in configuration
- **Skip Resources**: Use `--resources none` to skip all resource installation
- **CI/CD**: Automatically defaults to `none` to prevent unwanted installations

**Resource Management**:
- Enable/disable resources by editing `.vrooli/service.json`
- Resources marked as enabled will be installed on next setup run

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
