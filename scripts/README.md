# Scripts Directory Structure

## Overview

The `scripts/` directory follows a **strict separation** between Vrooli-specific code and universal/reusable code. This architecture enables:
1. **Clean standalone app generation** - Only universal code gets copied
2. **Clear boundaries** - No ambiguity about what's Vrooli-specific
3. **Unified lifecycle management** - All apps use the same patterns
4. **Recursive improvement** - Apps can build on universal capabilities

## Directory Structure

```
scripts/
├── manage.sh          # Universal entry point for ALL operations
├── app/               # Vrooli-specific code (NEVER copied to standalone)
│   ├── lifecycle/     # Vrooli's lifecycle implementations  
│   │   ├── setup.sh
│   │   ├── develop.sh
│   │   ├── build.sh
│   │   └── deploy.sh
│   ├── utils/         # Vrooli-specific utilities
│   │   ├── env.sh
│   │   ├── proxy.sh
│   │   └── ...
│   ├── package/       # Package-specific helpers
│   └── experimental/  # Experimental Vrooli features
│
├── lib/               # Universal libraries (SAFE for standalone apps)
│   ├── lifecycle/     # Lifecycle engine (declarative JSON execution)
│   │   └── engine.sh
│   ├── utils/         # Core utilities every app needs
│   │   ├── log.sh
│   │   ├── var.sh
│   │   ├── args.sh
│   │   └── flow.sh
│   ├── deps/          # Dependency installers
│   │   ├── bats.sh
│   │   └── vault.sh
│   ├── system/        # System utilities
│   │   ├── clock.sh
│   │   └── permissions.sh
│   ├── network/       # Network utilities
│   └── service/       # Service management
│
├── resources/         # Resource management system
│   ├── ai/           # AI resources (Ollama, Whisper, etc.)
│   ├── automation/   # Automation platforms (n8n, Node-RED, etc.)
│   ├── storage/      # Storage systems (PostgreSQL, Redis, etc.)
│   └── ...
│
├── scenarios/         # Direct scenario execution system
│   └── tools/
│       └── orchestrator/  # Scenario orchestration
│
└── __test/           # Test infrastructure
```

## Core Principles

### 1. Separation of Concerns
- **`app/`** = Vrooli monorepo ONLY. Never copied to standalone apps.
- **`lib/`** = Universal code. Safe for ANY app to use.
- **`resources/`** = External service management. Conditionally used.
- **`scenarios/`** = App generation system. Creates standalone apps.

### 2. Single Entry Point
- **`manage.sh`** is THE way to interact with any app (Vrooli or standalone)
- Replaces the old `scripts/main/` directory with a unified interface
- All behavior controlled by `.vrooli/service.json`

### 3. Declarative Lifecycle
Instead of hardcoded scripts, everything is configured in `.vrooli/service.json`:
```json
{
  "lifecycle": {
    "setup": {
      "description": "Prepare the environment",
      "steps": [
        {"name": "install-deps", "run": "pnpm install"},
        {"name": "vrooli-setup", "run": "./scripts/app/lifecycle/setup.sh"}
      ]
    },
    "develop": {
      "description": "Start development environment",
      "targets": {
        "native-linux": {
          "steps": [
            {"name": "start-services", "run": "./scripts/app/lifecycle/develop.sh"}
          ]
        }
      }
    }
  }
}
```

### 4. Context Awareness
Scripts detect their context automatically:
- **Monorepo context**: Full Vrooli with all capabilities
- **Standalone context**: Minimal app with only what it needs

## Usage Patterns

### For Vrooli Development
```bash
# Unified management interface:
./scripts/manage.sh setup --target native-linux
./scripts/manage.sh develop --detached yes
./scripts/manage.sh build --environment production
./scripts/manage.sh deploy --target k8s
```

### For Standalone Apps
Standalone apps get:
- `scripts/manage.sh` (entry point)
- `scripts/lib/` (universal libraries)
- Their own `.vrooli/service.json` (lifecycle configuration)

They DON'T get:
- `scripts/app/` (Vrooli-specific code)
- `scripts/resources/` (unless explicitly needed)
- `scripts/scenarios/` (app generation tools)

## File Organization Rules

### What Goes in `app/`?
- Vrooli-specific lifecycle scripts
- Monorepo-specific utilities (env management, proxy setup)
- Package-specific helpers (server, UI, jobs)
- Experimental features being developed
- Any code that references Vrooli packages or structure

### What Goes in `lib/`?
- Generic utilities (logging, argument parsing, flow control)
- The lifecycle engine itself
- Common dependency installers
- System utilities that any app might need
- Network and service utilities
- Anything that could work in ANY application

### What Goes in `resources/`?
- Resource-specific management scripts
- Each resource is self-contained in its directory
- Resources are opt-in based on app needs
- Integration tests and configuration

## Migration Path

### Phase 1: File Reorganization (COMPLETED) ✅
- Moved Vrooli-specific helpers to `app/`
- Created universal `lib/` directory
- Clear separation established

### Phase 2: Entry Point Unification (COMPLETED) ✅
- Created `manage.sh` as universal entry point
- Lifecycle engine in `lib/lifecycle/engine.sh`
- Service.json drives all behavior

### Phase 3: Complete Migration (COMPLETED) ✅
- Moved `scripts/main/*.sh` logic to `scripts/app/lifecycle/`
- Updated `.vrooli/service.json` with full lifecycle
- Removed `scripts/main/` directory
- Updated all documentation

### Phase 4: Gradual Rollout (COMPLETED) ✅
- Updated all entry points and CI/CD pipelines
- Created deprecation shims for smooth transition
- Updated standalone app generation
- Comprehensive testing and documentation

### Phase 5: Final Cleanup (COMPLETED) ✅
- Removed deprecated `scripts/main/` directory
- Cleaned up all remaining references
- Migration complete - single universal entry point achieved

## Why This Structure?

### Problem 1: Unclear Boundaries
**Old**: Helpers mixed Vrooli-specific and universal code
**New**: Clear `app/` vs `lib/` separation

### Problem 2: Multiple Entry Points
**Old**: Different scripts for each lifecycle phase
**New**: Single `manage.sh` for everything

### Problem 3: Standalone App Confusion
**Old**: Standalone apps got unnecessary Vrooli code
**New**: Standalone apps only get `lib/` directory

### Problem 4: Hardcoded Behavior
**Old**: Behavior hardcoded in shell scripts
**New**: Declarative JSON configuration

## The Lifecycle Engine

The lifecycle engine (`lib/lifecycle/engine.sh`) is the heart of the system:

1. **Reads service.json** - Gets lifecycle configuration
2. **Resolves targets** - Handles inheritance and overrides
3. **Executes steps** - Runs commands in order
4. **Supports patterns**:
   - Sequential steps
   - Parallel execution
   - Conditional execution
   - Environment-specific behavior
   - Target-specific overrides

Example lifecycle execution:
```bash
./scripts/manage.sh develop --target docker
# 1. Reads lifecycle.develop from service.json
# 2. Merges universal steps with docker-specific steps
# 3. Executes each step in order
# 4. Handles errors and logging
```

## Best Practices

1. **Always use manage.sh** - Don't call lifecycle scripts directly
2. **Keep app/ pure** - No universal code in app/ directory
3. **Keep lib/ generic** - No Vrooli-specific code in lib/
4. **Document dependencies** - If a lib/ script needs something, document it
5. **Test both contexts** - Ensure scripts work in monorepo AND standalone
6. **Use service.json** - Define behavior declaratively, not in scripts

## Troubleshooting

### "No service.json found"
Every app needs `.vrooli/service.json`. Create one or copy from templates.

### "Lifecycle engine not found"
The `scripts/lib/` directory is missing. For Vrooli, run `git restore scripts/lib`.

### "Phase not found in service.json"
Check `.vrooli/service.json` has the lifecycle phase defined.

### Import Errors After Migration
Update import paths:
- Old: `../helpers/utils/log.sh`
- New: `../lib/utils/log.sh` or `../app/utils/log.sh`

### Scripts Still Using Old Structure
Migration is complete! All phases finished:
- ✅ File structure reorganized
- ✅ manage.sh created as universal entry point
- ✅ Updated all lifecycle scripts
- ✅ Removed scripts/main/ directory
- ✅ Updated documentation and CI/CD pipelines

## Additional Resources

- [Lifecycle Engine Documentation](lib/lifecycle/README.md)
- [Service.json Schema](../.vrooli/schemas/service.schema.json)
- [Scenario System Documentation](scenarios/README.md)
- [Resource Management Documentation](resources/README.md)
- [CLAUDE.md - AI Context](../CLAUDE.md)
