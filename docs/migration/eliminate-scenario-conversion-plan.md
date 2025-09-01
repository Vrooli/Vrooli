# Detailed Plan: Eliminating Scenario-to-App Conversion

## Executive Summary
Remove the unnecessary scenario-to-app conversion layer, allowing scenarios to run directly from their source locations within Vrooli. This eliminates ~2000 lines of complex conversion code, removes gigabytes of duplication, and accelerates development cycles by 2-5 seconds per scenario operation.

**KEY INSIGHT**: Scenarios are ALREADY correctly configured with proper paths. The conversion process unnecessarily MODIFIES these paths. This plan simply REMOVES the modification step, not adds new complexity.

## 📊 Priority & Effort Matrix

| Phase | Priority | Effort | Risk | Notes |
|-------|----------|--------|------|-------|
| **Phase 1: Migration Tracking** | Medium | Low | Low | Simple setup, helps track progress |
| **Phase 2: Core Infrastructure** | **CRITICAL** | High | Medium | Foundation for everything else |
| **Phase 2.5: Orchestrator Refactor** | **CRITICAL** | **Very High** | **High** | Biggest blocker, most complex task |
| **Phase 3: Testing Framework** | High | Medium | Low | Must validate changes work |
| **Phase 4: Documentation** | **CRITICAL** | Medium | Low | User-facing docs must be accurate |
| **Phase 5: CLI & Script Removal** | High | Low | Medium | Clean break from old system |
| **Phase 6: Cleanup** | Low | Low | Low | Can be done gradually |
| **Phase 7: Deployment** | Medium | Medium | Medium | Important for production use |

### 🎯 Critical Path (Must Do First)
1. **Phase 2.5**: Orchestrator refactor - blocks everything else
2. **Phase 2**: Core infrastructure - enables direct execution
3. **Phase 4**: Documentation - users need accurate guides

### ⚡ Quick Wins (Easy & High Impact)
- populate.sh already works with direct paths!
- manage.sh changes are straightforward
- Documentation updates are clear and defined

## Key Benefits
- **Zero conversion time**: Instant scenario execution
- **No duplication**: Save gigabytes of disk space
- **Simpler architecture**: Remove complex path rewriting logic
- **Faster iteration**: 2-5 seconds saved per test/deploy cycle
- **Single source of truth**: No file synchronization issues

---

## Phase 1: Migration Tracking (Day 1)
*Goal: Track progress and maintain visibility*

### 1.1 Create Migration Status File
```bash
# Create migration status file
cat > docs/migration/scenario-direct-execution-status.md << 'EOF'
# Scenario Direct Execution Migration

## Status: In Progress
Started: [DATE]

## Scenarios Migrated
- [ ] simple-test
- [ ] make-it-vegan
- [ ] swarm-manager
[... list all scenarios ...]

## Systems Updated
- [ ] Testing framework
- [ ] CLI commands
- [ ] CI/CD pipelines
- [ ] Documentation
EOF
```

---

## Phase 2: Core Infrastructure Updates (Day 2-3)
*Goal: Enable direct scenario execution*

### 2.1 Create Scenario Runner Script
```bash
# New file: scripts/lib/scenario/runner.sh
#!/usr/bin/env bash
set -euo pipefail

scenario::run() {
    local scenario_name="$1"
    shift
    
    local scenario_path="${var_ROOT_DIR}/scenarios/${scenario_name}"
    
    if [[ ! -d "$scenario_path" ]]; then
        log::error "Scenario not found: $scenario_name"
        return 1
    fi
    
    # Set context for scenario execution
    export SCENARIO_NAME="$scenario_name"
    export SCENARIO_PATH="$scenario_path"
    # APP_ROOT remains as-is in scenarios (already correct)
    
    # Change to scenario directory
    cd "$scenario_path" || return 1
    
    # Execute manage.sh which will handle the scenario context
    "${var_ROOT_DIR}/scripts/manage.sh" "$@"
}

scenario::list() {
    log::info "Available scenarios:"
    for scenario in "${var_ROOT_DIR}"/scenarios/*/; do
        if [[ -d "$scenario" ]]; then
            local name="${scenario%/}"
            name="${name##*/}"
            local service_json="${scenario}/.vrooli/service.json"
            if [[ -f "$service_json" ]]; then
                local description=$(jq -r '.service.description // ""' "$service_json")
                echo "  • $name - $description"
            else
                echo "  • $name"
            fi
        fi
    done
}

scenario::test() {
    local scenario_name="$1"
    shift
    scenario::run "$scenario_name" test "$@"
}
```

### 2.2 Update manage.sh to Support Scenarios
```bash
# Add to scripts/manage.sh after argument parsing:

# Robust scenario directory detection
if [[ -f "${PWD}/.vrooli/service.json" ]]; then
    # Check if we're in a scenario directory by looking at parent paths
    if [[ "${PWD}" == */scenarios/* ]]; then
        SCENARIO_NAME="$(basename "${PWD}")"
        export SCENARIO_NAME
        export SCENARIO_MODE=true
        export SCENARIO_PATH="${PWD}"
        log::info "Running in scenario mode: $SCENARIO_NAME"
    fi
fi

# Adjust service.json path for scenarios
if [[ "${SCENARIO_MODE:-false}" == "true" ]]; then
    SERVICE_JSON_PATH="${PWD}/.vrooli/service.json"
    
    # Scenario-specific process manager paths
    export PM_HOME="${HOME}/.vrooli/processes/scenarios/${SCENARIO_NAME}"
    export PM_LOG_DIR="${HOME}/.vrooli/logs/scenarios/${SCENARIO_NAME}"
    
    # Source port registry for resource ports
    source "${var_ROOT_DIR}/scripts/resources/port_registry.sh" 2>/dev/null || true
else
    SERVICE_JSON_PATH="${APP_ROOT}/.vrooli/service.json"
fi

# Scenarios already handle their own isolation and paths correctly
# No additional path or port adjustments needed
```

### 2.3 Replace Existing Vrooli CLI Scenario Commands
```bash
# File: cli/commands/scenario-commands.sh
# This COMPLETELY REPLACES the existing API-based scenario commands
# 
# Current API-based commands to REMOVE:
#   All API calls to localhost:8092/scenarios
#   scenario convert    - Removes entirely
#   scenario validate   - Removes entirely
#   scenario enable/disable - Removes entirely
#   scenario convert-all - Removes entirely
#
# New direct execution commands:
#   scenario run        - Run scenario directly
#   scenario test       - Test scenario directly  
#   scenario list       - List scenarios (direct filesystem)

# REPLACE ENTIRE FILE CONTENTS:

#!/usr/bin/env bash
# Vrooli CLI - Scenario Management Commands (Direct Execution)
set -euo pipefail

# Get CLI directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${APP_ROOT}/scripts/lib/scenario/runner.sh"

# Show help for scenario commands
show_scenario_help() {
    cat << EOF
🚀 Vrooli Scenario Commands

USAGE:
    vrooli scenario <subcommand> [options]

SUBCOMMANDS:
    run <name>              Run a scenario directly
    test <name>             Test a scenario
    list                    List available scenarios

EXAMPLES:
    vrooli scenario run make-it-vegan
    vrooli scenario test swarm-manager
    vrooli scenario list
EOF
}

# Main handler
main() {
    if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_scenario_help
        return 0
    fi
    
    local subcommand="$1"; shift
    case "$subcommand" in
        run)
            local scenario_name="${1:-}"
            [[ -z "$scenario_name" ]] && { 
                log::error "Scenario name required"
                log::info "Usage: vrooli scenario run <name>"
                return 1
            }
            shift
            scenario::run "$scenario_name" develop "$@"
            ;;
        test)
            local scenario_name="${1:-}"
            [[ -z "$scenario_name" ]] && { 
                log::error "Scenario name required"
                log::info "Usage: vrooli scenario test <name>"
                return 1
            }
            shift
            scenario::run "$scenario_name" test "$@"
            ;;
        list)
            scenario::list
            ;;
        # Removed: convert, convert-all, validate, enable, disable
        *)
            log::error "Unknown scenario command: $subcommand"
            echo ""
            show_scenario_help
            return 1
            ;;
    esac
}

main "$@"

# Update the help text in cli/vrooli main script:
# FIND the "scenario" case in the help function
# REPLACE with:

"scenario")
    cat << EOF
🎯 SCENARIO MANAGEMENT:
    scenario run <name>      Run a scenario directly
    scenario test <name>     Test a scenario
    scenario list            List available scenarios
    
Examples:
    vrooli scenario run make-it-vegan
    vrooli scenario test swarm-manager
    vrooli scenario list
EOF
    ;;
```

---

## Phase 3: Testing Framework Migration (Day 3-4)
*Goal: Update all tests to use direct execution*

### 3.1 Update Test Runners
```bash
# Update scripts/scenarios/validation/run-all-scenarios.sh

# OLD CODE:
# ./scenario-to-app.sh "$scenario_name"
# cd ~/generated-apps/"$scenario_name"
# ./scripts/manage.sh test

# NEW CODE:
run_scenario_test() {
    local scenario_name="$1"
    local scenario_path="${APP_ROOT}/scenarios/${scenario_name}"
    
    if [[ ! -d "$scenario_path" ]]; then
        log::error "Scenario not found: $scenario_name"
        return 1
    fi
    
    # Run test directly from scenario directory
    (
        cd "$scenario_path" || exit 1
        "${APP_ROOT}/scripts/manage.sh" test --quick
    )
}
```

### 3.2 Update Integration Tests
```bash
# Update scripts/scenarios/tools/run-integration-test.sh
# Remove all references to:
# - scenario-to-app.sh
# - generated-apps directory
# - App generation steps

# Add direct execution logic:
run_integration_test() {
    local scenario="$1"
    
    log::info "Running integration test for scenario: $scenario"
    
    cd "${APP_ROOT}/scenarios/${scenario}" || {
        log::error "Failed to change to scenario directory"
        return 1
    }
    
    # Run the scenario's test suite
    "${APP_ROOT}/scripts/manage.sh" test
}
```

### 2.4 Update setup::is_needed to Accept Path Parameter
```bash
# In scripts/lib/utils/setup.sh, modify setup::is_needed:
setup::is_needed() {
    # Accept optional path parameter for service.json location
    local check_path="${1:-$APP_ROOT}"
    
    # Reset global array for setup reasons
    SETUP_REASONS=()
    
    # Get service.json path
    local service_json="${check_path}/.vrooli/service.json"
    
    if [[ ! -f "$service_json" ]]; then
        log::debug "No service.json found at $check_path, assuming setup not needed"
        return 1
    fi
    
    # Rest of function remains the same...
}
```

### 2.5 Refactor Orchestrator for Direct Scenario Execution
```bash
# CRITICAL: The orchestrator is deeply integrated with generated-apps
# This full refactor updates it to work with scenarios directly

# 1. Backup existing orchestrator before changes
cp -r scripts/scenarios/tools/orchestrator scripts/scenarios/tools/orchestrator.backup

# 2. Update enhanced_orchestrator.py
# File: scripts/scenarios/tools/orchestrator/enhanced_orchestrator.py

# CHANGES NEEDED:
# Line 100: Change from:
self.generated_apps_dir = Path.home() / "generated-apps"
# To:
self.scenarios_dir = self.vrooli_root / "scenarios"

# Line 358: Change from:
app_path = self.generated_apps_dir / app_name
# To:
app_path = self.scenarios_dir / app_name

# Lines 222-224: Change process detection from:
if (f"generated-apps/{app_name}" in cmdline or 
    f"generated-apps/{app_name}" in cwd or
    (app_name in cwd and "generated-apps" in cwd)):
# To:
if (f"scenarios/{app_name}" in cmdline or 
    f"scenarios/{app_name}" in cwd or
    (app_name in cwd and "scenarios" in cwd)):

# Update app discovery method to scan scenarios:
def discover_apps(self):
    """Discover available scenarios"""
    self.apps = {}
    
    if not self.scenarios_dir.exists():
        self.logger.warning(f"Scenarios directory not found: {self.scenarios_dir}")
        return
    
    # Scan for scenarios with service.json
    for scenario_dir in self.scenarios_dir.iterdir():
        if scenario_dir.is_dir():
            service_json = scenario_dir / ".vrooli" / "service.json"
            if service_json.exists():
                try:
                    config = json.loads(service_json.read_text())
                    app = EnhancedApp(
                        name=scenario_dir.name,
                        path=str(scenario_dir),
                        enabled=config.get('enabled', True),
                        description=config.get('service', {}).get('description', '')
                    )
                    self.apps[scenario_dir.name] = app
                except Exception as e:
                    self.logger.error(f"Failed to load scenario {scenario_dir.name}: {e}")

# 3. Update preflight-check.sh
# File: scripts/scenarios/tools/orchestrator/preflight-check.sh

# Lines 73-85: Change from counting generated apps to scenarios:
SCENARIOS_DIR="${APP_ROOT}/scenarios"
if [[ -d "$SCENARIOS_DIR" ]]; then
    SCENARIO_COUNT=$(find "$SCENARIOS_DIR" -maxdepth 1 -type d ! -name ".*" ! -name "scenarios" | wc -l)
    echo "Scenarios found: $SCENARIO_COUNT"
    
    if [[ $SCENARIO_COUNT -gt $MAX_APPS_TO_START ]]; then
        echo -e "${YELLOW}WARNING: Many scenarios found ($SCENARIO_COUNT)${NC}"
        echo "Only the first $MAX_APPS_TO_START scenarios will be started to prevent overload."
        
        # Create a file to limit scenarios
        echo "$MAX_APPS_TO_START" > /tmp/vrooli-max-scenarios-limit
    fi
fi

# 4. Update start-apps-safely.sh
# No changes needed - it just calls the orchestrator

# 5. Update run.sh
# No changes needed - it's just a wrapper
```

## Phase 4: Documentation Updates (Day 4)
*Goal: Update all documentation to reflect direct execution model*

### 4.1 Critical User-Facing Documentation
These files are what users see first and must be updated immediately:

```bash
# 1. Complete rewrite of DEPLOYMENT.md
cat > docs/scenarios/DEPLOYMENT.md << 'EOF'
# Direct Scenario Deployment Guide

## 🚀 Running Scenarios Directly

Scenarios run directly from their source location without conversion:

\`\`\`bash
# Run a scenario
cd scenarios/research-assistant
../../scripts/manage.sh develop

# Or use the CLI
vrooli scenario run research-assistant
\`\`\`

## How Direct Execution Works

1. **No Conversion Needed**: Scenarios run directly from `scenarios/` folder
2. **Process Isolation**: Each scenario gets its own PM_HOME and PM_LOG_DIR
3. **Resource Sharing**: Scenarios use Vrooli's scripts and libraries
4. **Instant Updates**: Changes take effect immediately

[Rest of updated content...]
EOF

# 2. Update TESTING_GUIDE.md
# Remove all references to scenario-to-app and generated apps
# Update testing commands to use direct execution

# 3. Update VALIDATION.md
# Remove conversion validation steps
# Focus on runtime validation
```

### 4.2 Internal Documentation Updates
```bash
# Update automation prompts and internal docs
find auto/ -name "*.md" -exec sed -i 's/vrooli scenario convert/vrooli scenario run/g' {} \;
# Note: Requires manual review for context-specific changes
```

### 4.3 Create Migration Guide
```bash
cat > docs/migration/direct-execution-guide.md << 'EOF'
# Migration Guide: Direct Scenario Execution

## What's Changed
- **Before**: Scenarios converted to apps in ~/generated-apps
- **Now**: Scenarios run directly from scenarios/ folder

## Command Translation
| Old Command | New Command |
|------------|-------------|
| `vrooli scenario convert <name>` | `vrooli scenario run <name>` |
| `vrooli app start <name>` | `vrooli scenario run <name>` |
| `cd ~/generated-apps/<name>` | `cd scenarios/<name>` |

[Rest of migration guide...]
EOF
```

## Phase 5: CLI & Script Removal (Day 5)
*Goal: Remove conversion infrastructure and update user commands*

### 5.1 Remove All Conversion Scripts and Auto-Converters
```bash
# DELETE these files entirely (no backwards compatibility):
rm -f scripts/scenarios/tools/scenario-to-app.sh
rm -f scripts/scenarios/tools/app-structure.json
rm -f scripts/scenarios/tools/start-app-safe.sh
rm -f scripts/scenarios/tools/auto-converter.sh

# Stop and remove auto-converter loops
pkill -f "manage-scenario-loop.sh" 2>/dev/null || true
rm -f scripts/auto/manage-scenario-loop.sh
rm -f scripts/auto/manage-resource-loop.sh

# Remove any systemd services or cron jobs for auto-conversion
systemctl --user stop vrooli-scenario-converter 2>/dev/null || true
systemctl --user disable vrooli-scenario-converter 2>/dev/null || true
rm -f ~/.config/systemd/user/vrooli-scenario-converter.service
crontab -l | grep -v "scenario-to-app\|auto-converter" | crontab - 2>/dev/null || true

# Remove the generated-apps directory
rm -rf ~/generated-apps
```

### 5.2 Update CLI Commands
- Replace scenario-commands.sh with direct execution version
- Remove all API-based scenario commands
- Add new direct filesystem commands

---

## Phase 6: Cleanup & Optimization (Day 6)
*Goal: Remove unnecessary code and optimize*

### 6.1 Files to Remove (After Full Verification)
```bash
# Scripts that become obsolete:
- scripts/scenarios/tools/scenario-to-app.sh
- scripts/scenarios/tools/app-structure.json
- scripts/scenarios/tools/start-app-safe.sh
- scripts/scenarios/tools/auto-converter.sh

# Functions to remove from existing files:
- Path adjustment logic in scenario-to-app.sh
- APP_ROOT depth adjustment functions
- Generated app specific code
```

### 6.2 Remove Generated Apps References
```bash
# No resource management changes needed - scenarios already handle ports correctly
# Simply remove all references to generated-apps paths in existing code
```

---

## Phase 7: Deployment Updates (Day 7)
*Goal: Ensure deployment scenarios work with new model*

### 7.1 Create Deployment Packager for New Model
```bash
#!/usr/bin/env bash
# scripts/deployment/package-scenario-deployment.sh
set -euo pipefail

# Package Vrooli with specific scenarios for deployment
package::deployment() {
    local deployment_name="$1"
    local output_dir="$2"
    shift 2
    local scenarios=("$@")
    
    log::info "Creating deployment package: $deployment_name"
    log::info "Including scenarios: ${scenarios[*]}"
    
    # Create deployment structure
    mkdir -p "$output_dir"
    
    # Copy core Vrooli files
    log::info "Copying Vrooli framework..."
    cp -r scripts "$output_dir/"
    cp -r .vrooli "$output_dir/"
    cp -r cli "$output_dir/"
    
    # Copy only specified scenarios
    mkdir -p "$output_dir/scenarios"
    for scenario in "${scenarios[@]}"; do
        if [[ -d "scenarios/$scenario" ]]; then
            log::info "Adding scenario: $scenario"
            cp -r "scenarios/$scenario" "$output_dir/scenarios/"
        else
            log::warning "Scenario not found: $scenario"
        fi
    done
    
    # Create deployment manifest
    cat > "$output_dir/deployment.json" << EOF
{
    "name": "$deployment_name",
    "created": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "scenarios": $(printf '%s\n' "${scenarios[@]}" | jq -R . | jq -s .),
    "vrooli_version": "$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"
}
EOF
    
    log::success "Deployment package created: $output_dir"
}

# Example usage:
# package::deployment "saas-platform" ~/deployments/saas-platform \
#     make-it-vegan invoice-generator mind-maps
```

### 7.2 Update CI/CD Pipelines
```yaml
# Update .github/workflows/test-scenarios.yml or similar

jobs:
  test-scenarios:
    steps:
      - name: Test Scenarios Directly
        run: |
          # Use absolute paths for robustness
          SCENARIOS_DIR="${GITHUB_WORKSPACE}/scenarios"
          SCRIPTS_DIR="${GITHUB_WORKSPACE}/scripts"
          
          # No longer need to generate apps
          for scenario_path in "${SCENARIOS_DIR}"/*/; do
            if [[ ! -d "$scenario_path" ]]; then
              continue
            fi
            
            scenario_name="$(basename "${scenario_path}")"
            echo "Testing scenario: ${scenario_name}"
            
            # Run directly from scenario directory with absolute paths
            (
              cd "${scenario_path}"
              "${SCRIPTS_DIR}/manage.sh" test --ci
            ) || {
              echo "Failed to test scenario: ${scenario_name}"
              exit 1
            }
          done
```

---

## Verification Checklist

### Pre-Migration Tests
- [ ] Document current test pass rate
- [ ] Measure current scenario startup time
- [ ] Record disk usage of generated-apps
- [ ] List all places that reference generated-apps

### During Migration
- [ ] Each phase completes successfully
- [ ] No regression in test pass rate
- [ ] Scenarios run without conversion
- [ ] CLI commands work as expected
- [ ] Documentation is clear

### Post-Migration Tests
- [ ] All scenarios run with `vrooli scenario run <name>`
- [ ] All scenarios pass tests with direct execution
- [ ] Startup time improved by 2-5 seconds
- [ ] No generated-apps directory needed
- [ ] Resource allocation works correctly
- [ ] Port management handles multiple scenarios
- [ ] PID tracking works for scenarios
- [ ] Deployment packaging works

### Performance Metrics to Track
- Scenario startup time (before/after)
- Test execution time (before/after)
- Disk usage (before/after)
- Memory usage during scenario execution
- Time to run all scenario tests

---

## Rollback Plan

If critical issues arise: STOP and alert the user. DO NOT attempt to rollback. The user will decide how to proceed.

---

## Success Metrics

### Immediate (Week 1)
- ✓ All scenarios run directly without conversion
- ✓ Test suite maintains 100% pass rate
- ✓ 2-5 second improvement in scenario operations
- ✓ Zero generated-apps needed for development

### Short-term (Week 2-4)
- ✓ No regression reports from team
- ✓ Documentation fully updated
- ✓ CI/CD pipelines optimized
- ✓ Disk space recovered (document GB saved)

### Long-term (Month 1-3)
- ✓ Deployment scenarios using new model
- ✓ Team productivity increased
- ✓ Maintenance burden reduced
- ✓ System complexity decreased

---

## Risk Mitigation

### Risk: Scenarios interfere with each other
**Mitigation:** 
- Scenario-specific PM_HOME and PM_LOG_DIR paths
- Process manager isolation per scenario
- Resource ports handled by port_registry.sh
- Each scenario runs in its own working directory

### Risk: Orchestrator refactor breaks
**Mitigation:**
- Full backup before changes: `cp -r orchestrator orchestrator.backup`
- Test orchestrator changes incrementally
- Can temporarily symlink generated-apps to scenarios if needed
- Consider orchestrator deprecation if not needed

### Risk: Performance regression
**Mitigation:**
- Benchmark before and after each phase
- Profile scenario execution paths
- Actually FASTER without conversion overhead
- Direct execution saves 2-5 seconds per operation

### Risk: Hidden dependencies on generated-apps
**Mitigation:**
- Comprehensive grep for "generated-apps" references
- Test all user-facing commands
- Monitor error logs for path-related issues
- Orchestrator is the main dependency (addressed in Phase 2.5)

---

## 📚 Documentation Priority List

### URGENT - Blocks User Understanding
1. **docs/scenarios/DEPLOYMENT.md** - Complete rewrite needed (currently all about conversion)
2. **docs/scenarios/TESTING_GUIDE.md** - Major updates to remove conversion references
3. **docs/scenarios/VALIDATION.md** - Significant updates for direct execution

### HIGH - User-Facing Commands
4. **CLI help text** - Update all `vrooli scenario` command examples
5. **Error messages** - Update to reference direct execution, not conversion
6. **Quick start guides** - Ensure new users learn the right approach

### MEDIUM - Internal Documentation
7. **Auto-task prompts** - Update AI prompts to use direct execution
8. **Developer guides** - Update contribution docs
9. **Architecture docs** - Explain new direct execution model

### LOW - Nice to Have
10. **Resource READMEs** - Minor updates if they reference scenarios
11. **Template docs** - Update scenario templates
12. **Historical docs** - Archive old conversion documentation

## Implementation Timeline

### Week 1: Core Migration
- **Day 1:** Migration tracking setup, orchestrator backup
- **Day 2:** Core infrastructure updates (manage.sh, runner.sh)
- **Day 3:** Orchestrator refactor (most complex task)
- **Day 4:** Documentation overhaul (critical for users)
- **Day 5:** CLI & script removal
- **Day 6:** Cleanup & optimization
- **Day 7:** Deployment updates & verification

### Week 2: Stabilization
- Monitor for edge cases
- Fix any reported issues
- Update remaining documentation
- Optimize performance bottlenecks

### Week 3-4: Finalization
- Remove deprecated code
- Final performance tuning
- Team training sessions
- Close migration tracking

---

## Complete File Modification Checklist

### NEW FILES TO CREATE
```bash
□ scripts/lib/scenario/runner.sh
  - Main scenario execution functions
  - scenario::run(), scenario::list(), scenario::test()

□ docs/migration/scenario-direct-execution-status.md
  - Migration tracking document
  - Checklist of completed items

□ docs/migration/scenario-direct-execution-guide.md
  - User guide for new execution model
  - Migration instructions for users
```

### FILES TO DELETE COMPLETELY
```bash
# Core conversion scripts (DELETE these files entirely)
□ scripts/scenarios/tools/scenario-to-app.sh
□ scripts/scenarios/tools/app-structure.json
□ scripts/scenarios/tools/start-app-safe.sh
□ scripts/scenarios/tools/auto-converter.sh

# Auto-converter loops (DELETE these files)
□ scripts/auto/manage-scenario-loop.sh
□ scripts/auto/manage-resource-loop.sh

# Generated apps directory (after migration complete)
□ ~/generated-apps/ (entire directory)

# Systemd services and cron jobs
□ ~/.config/systemd/user/vrooli-scenario-converter.service
□ Any cron entries containing "scenario-to-app" or "auto-converter"
```

### FILES TO MODIFY - Remove Conversion References
```bash
# CLI Commands
□ cli/commands/scenario-commands.sh
  - REPLACE ENTIRE FILE with new direct execution implementation
  - Remove ALL API calls to localhost:8092
  - Remove: convert, validate, enable, disable, convert-all subcommands
  - Add: Direct filesystem-based run, test, list commands

□ cli/commands/app-commands.sh
  - May not need changes if it works with running apps
  - Only update if it references generated-apps paths

□ cli/commands/stop-commands.sh
  - Remove: Generated app stop logic
  - Update: To stop scenarios directly

# Testing Scripts
□ scripts/scenarios/tools/run-integration-test.sh
  - Remove: scenario-to-app.sh calls
  - Remove: cd ~/generated-apps logic
  - Update: To run tests from scenarios/

□ scripts/scenarios/validation/run-all-scenarios.sh
  - Remove: App generation steps
  - Remove: Generated apps references
  - Update: To test scenarios directly

□ __test/phases/test-integration.sh
  - Remove: Generated apps testing
  - Update: To test scenarios directly

□ __test/phases/test-structure.sh
  - Remove: Generated apps structure validation
  - Update: Scenario structure validation only

# Orchestration Scripts (Major Refactor Required)
□ scripts/scenarios/tools/orchestrator/enhanced_orchestrator.py
  - REPLACE: self.generated_apps_dir with self.scenarios_dir  
  - UPDATE: All path references from generated-apps to scenarios
  - REWRITE: discover_apps() method to scan scenarios/
  - MODIFY: Process detection logic for scenarios paths

□ scripts/scenarios/tools/orchestrator/preflight-check.sh
  - Remove: Generated apps checks
  - Update: To check and count scenarios directly
  - Change: GENERATED_APPS_DIR to SCENARIOS_DIR

□ scripts/scenarios/tools/cleanup-processes.sh
  - Remove: Generated apps cleanup
  - Update: To clean scenario processes

# Lifecycle Management
□ scripts/lib/lifecycle/stop-manager.sh
  - Remove: Generated apps stopping logic
  - Update: Scenario stopping logic

□ scripts/lib/utils/setup.sh
  - Update: setup::is_needed() to accept path parameter
  - Allow checking different locations for service.json

# CI/CD Workflows (if they reference scenarios)
□ .github/workflows/test.yml
  - Remove: Generated apps testing
  - Update: Direct scenario testing

□ .github/workflows/dev.yml
  - Remove: Generated apps references
  - Update: Scenario deployment if needed

□ .github/workflows/master.yml
  - Remove: Generated apps build steps
  - Update: Scenario validation if needed
```

### CODE/FUNCTIONS TO REMOVE FROM FILES
```bash
# In scenario-related files, REMOVE these functions/patterns:
□ scenario_to_app::* (all functions with this prefix)
□ adjust_app_root_depth()
□ process_template_variables() (if only for conversion)
□ copy_from_manifest()
□ bulk_copy_remaining_files()
□ Path adjustment logic (sed commands modifying APP_ROOT depth)
□ /scenarios/<name>/ path stripping logic
□ Generated app validation logic
□ Conversion-specific JSON schema validation

# Environment Variables and Constants to UPDATE/REMOVE:
□ GENERATED_APPS_DIR references
□ GENERATED_APPS_PATH references  
□ Any paths containing "generated-apps"
□ Conversion-specific environment variables
□ APP_ROOT depth adjustment code (remove it all)
```

### DOCUMENTATION TO UPDATE - CRITICAL PHASE
```bash
# HIGH PRIORITY - User-facing documentation that heavily references conversion

□ docs/scenarios/DEPLOYMENT.md (MAJOR REWRITE NEEDED)
  - CURRENT: Entire file is about scenario-to-app conversion
  - REPLACE WITH: Direct scenario deployment guide
  - Key sections to rewrite:
    * "Quick Start" - Remove all `vrooli scenario convert` commands
    * "How It Works" - Explain direct execution instead of conversion
    * "Generated App" section - Remove entirely
    * "Running Your Generated App" - Replace with "Running Your Scenario"
    * Update all command examples to use direct execution
  - NEW CONTENT:
    * How scenarios run directly from scenarios/ folder
    * Process isolation and PM_HOME/PM_LOG_DIR paths
    * Resource sharing model
    * Direct deployment without conversion

□ docs/scenarios/TESTING_GUIDE.md (MAJOR UPDATES)
  - Lines 19-24: Remove scenario-to-app.sh references
  - Lines 28-44: Update "Post-Generation" to "Direct Execution"
  - Lines 51-63: Fix file path validation complexity section
  - Lines 145-151: Remove manual testing with generated apps
  - Update all testing commands to use direct execution
  - Add section on testing scenarios in-place

□ docs/scenarios/VALIDATION.md (SIGNIFICANT UPDATES)
  - Line 354: Remove "./tools/scenario-to-app.sh" dry-run reference
  - Lines 358-366: Update deployment simulation section
  - Remove all references to app generation
  - Update validation to work with direct execution
  - Emphasize runtime validation over conversion validation

□ docs/scenarios/CONCEPTS.md (MODERATE UPDATES)
  - Line 141: Remove `vrooli scenario convert` command
  - Update any references to generated apps
  - Explain new direct execution model

□ docs/scenarios/getting-started.md (MINOR UPDATES)
  - Good news: This file already uses direct execution!
  - Only minor touchups needed if any

# MEDIUM PRIORITY - Internal documentation and READMEs

□ scripts/scenarios/README.md (MODERATE UPDATES)
  - Lines 80-89: Remove all `vrooli scenario convert` examples
  - Line 107: Remove conversion reference
  - Update workflow descriptions
  - Add direct execution examples

□ auto/tasks/scenario-improvement/prompts/scenario-improvement-loop.md
  - Line 90: Remove `vrooli scenario convert` command
  - Line 128: Update NOTE about workflow injection
  - Line 190: Remove conversion step
  - Update entire workflow to use direct execution

□ auto/docs/PROMPT_ENGINEERING.md
  - Line 361: Remove `vrooli scenario convert` reference
  - Update any app generation references

# LOW PRIORITY - Resource documentation

□ resources/litellm/README.md
  - Line 155: Update `vrooli scenario generate` if needed
  - Check for any generated-apps references

□ resources/openrouter/README.md
  - Lines 80, 83: Update scenario generation examples if needed
  - Check for conversion references

# ADDITIONAL DOCUMENTATION NEEDS

□ CREATE: docs/migration/direct-execution-guide.md
  - User migration guide from old to new model
  - Command translation table
  - Troubleshooting common issues
  - Benefits of direct execution

□ UPDATE: Main README.md
  - Already clean! No scenario-to-app references found
  - May need minor updates to reinforce direct execution

□ UPDATE: CLAUDE.md
  - Line references to `vrooli scenario` commands
  - Ensure all examples use new direct execution syntax

□ UPDATE: All scenario template READMEs
  - Remove any conversion instructions
  - Update to explain direct execution
```

### GREP VERIFICATION COMMANDS
```bash
# After changes, these should return ZERO results:
□ grep -r "scenario-to-app" . --exclude-dir=.git
□ grep -r "generated-apps" . --exclude-dir=.git
□ grep -r "generate_app" . --exclude-dir=.git
□ grep -r "scenario_to_app" . --exclude-dir=.git
□ grep -r "app-structure.json" . --exclude-dir=.git
□ grep -r "adjust_app_root" . --exclude-dir=.git
□ grep -r "copy_from_manifest" . --exclude-dir=.git

# These should only show the new implementations:
□ grep -r "scenario::run" . --exclude-dir=.git
□ grep -r "scenario::test" . --exclude-dir=.git
□ grep -r "SCENARIO_MODE" . --exclude-dir=.git
```

### FINAL CLEANUP CHECKLIST
```bash
# After all code changes are complete:
□ Remove all backup files (.backup extensions)
□ Delete migration-specific temporary scripts
□ Remove deprecated wrapper functions
□ Clean up any TODO/FIXME comments added during migration
□ Update all copyright dates if needed
□ Run shellcheck on all modified bash scripts
□ Ensure all scripts are executable (chmod +x)
```

### VERIFICATION TESTS
```bash
# These must ALL pass before migration is complete:
□ vrooli scenario list (shows all scenarios)
□ vrooli scenario run simple-test (runs successfully)
□ vrooli scenario test simple-test (passes tests)
□ cd scenarios/make-it-vegan && ../../scripts/manage.sh develop (works)
□ No references to ~/generated-apps in codebase
□ No orphaned processes after stopping scenarios
□ Port allocation works for multiple scenarios
□ All CI/CD pipelines pass
```

---

## Recommended Testing Order

Based on complexity and dependencies, follow this testing sequence:

### 1. **Test populate.sh First** (Already Compatible!)
```bash
# Good news: populate.sh already supports direct paths!
cd scenarios/simple-test
../../scripts/resources/populate/populate.sh .
# This already works with populate::add_from_path()
```

### 2. **Test manage.sh Scenario Detection**
```bash
cd scenarios/simple-test
../../scripts/manage.sh develop --dry-run
# Verify SCENARIO_MODE is set and paths are correct
```

### 3. **Test New CLI Commands**
```bash
vrooli scenario list
vrooli scenario run simple-test --dry-run
vrooli scenario test simple-test --dry-run
```

### 4. **Test Process Manager Integration**
```bash
# Verify scenario-specific PM paths are created
cd scenarios/simple-test
../../scripts/manage.sh develop
# Check ~/.vrooli/processes/scenarios/simple-test/
```

### 5. **Tackle Orchestrator Refactor** (Most Complex)
```bash
# After backing up and refactoring
python3 scripts/scenarios/tools/orchestrator/enhanced_orchestrator.py
# Verify it discovers scenarios instead of generated-apps
```

---

## Important Notes

### populate.sh Compatibility
- **Already supports direct execution!** The `populate::add_from_path()` function exists and works
- Expects `.vrooli/service.json` which scenarios already have
- Sets `SCENARIO_PATH` and `SCENARIO_NAME` environment variables automatically
- Minimal changes needed - mostly documentation updates

### Process Manager Isolation
- Each scenario gets its own PM_HOME and PM_LOG_DIR
- Prevents process conflicts between scenarios
- Logs are organized per scenario for easier debugging

### Orchestrator Considerations
- The orchestrator might not even be needed with direct execution
- Scenarios can self-manage via pm::start/stop
- Consider deprecating orchestrator in Phase 7 if not needed

---

## Next Immediate Steps

1. **Backup orchestrator before changes**
   ```bash
   cp -r scripts/scenarios/tools/orchestrator scripts/scenarios/tools/orchestrator.backup
   ```

2. **Start with simple-test scenario**
   - Smallest scenario for proof of concept
   - Verify direct execution works
   - Measure performance improvement

3. **Create migration tracking issue**
   ```markdown
   Title: Eliminate Scenario-to-App Conversion
   
   ## Objective
   Remove unnecessary conversion layer, allow direct scenario execution
   
   ## Tracking
   - [ ] Phase 1: Foundation
   - [ ] Phase 2: Infrastructure
   - [ ] Phase 3: Testing
   - [ ] Phase 4: CLI
   - [ ] Phase 5: Cleanup
   - [ ] Phase 6: Deployment
   
   ## Metrics
   - Before: X seconds to run scenario
   - After: Y seconds (goal: X-3 seconds)
   ```

4. **Run proof of concept**
   ```bash
   cd scenarios/simple-test
   ../../scripts/manage.sh test
   # Verify this works before proceeding
   ```

---

## Appendix: Technical Details

### Path Resolution - NO CHANGES NEEDED
- Scenarios ALREADY use correct paths: `APP_ROOT="../../.."`
- Generated apps unnecessarily modified these to: `APP_ROOT=".."`
- This plan REMOVES the modification step entirely
- Scenarios work as-is without any path changes

### Port Management - NO CHANGES NEEDED  
- Scenarios ALREADY handle port allocation correctly
- No port conflict resolution needed
- Remove any port management complexity from the plan

### Resource Sharing Model
- Vrooli provides all scripts and libraries
- Scenarios provide app-specific code and configuration
- Runtime data stored in ~/.vrooli/scenario-data/
- No duplication of framework code

---

This plan eliminates unnecessary complexity while maintaining system reliability. The key insight: scenarios are already configured correctly for direct execution - we're removing an unnecessary translation layer that adds complexity without value.