# Core-Debugger Integration with Swarm Manager

## Overview
This document describes the integration between `core-debugger` and `swarm-manager`, enabling intelligent health-aware task orchestration with automatic core issue prioritization.

## What Was Changed

### 1. Scenario Registry (`config/scenario-registry.yaml`)
- ✅ Added `core-debugger` scenario with capabilities
- ✅ Set `priority_multiplier: 10` for core issues
- ✅ Added selection rules for core infrastructure keywords
- ✅ Marked as `zero_dependencies: true` (only needs claude-code)

### 2. Settings Configuration (`config/settings.yaml`)
- ✅ Added `core-debugger` to available scenarios
- ✅ Mapped task types: `core-infrastructure`, `core-fix`, `cli-issue`, etc.
- ✅ Added priority modifiers (10x for core issues, 0.5x if workaround exists)
- ✅ Configured health check before dispatch settings

### 3. Problem Analyzer (`prompts/problem-analyzer.md`)
- ✅ Added core infrastructure routing (HIGHEST PRIORITY)
- ✅ Listed core issue patterns (CLI failures, orchestrator problems)
- ✅ Integrated workaround database awareness

### 4. Backlog Generator (`prompts/backlog-generator.md`)
- ✅ Updated problem routing hierarchy
- ✅ Core infrastructure failures now route to core-debugger first

### 5. Task Executor (`prompts/task-executor.md`)
- ✅ Added pre-execution health check requirement
- ✅ Included core-debugger CLI commands
- ✅ Marked core commands as HIGHEST PRIORITY

### 6. Health Check Gate (`prompts/health-check-gate.md`)
- ✅ Created comprehensive health check workflow
- ✅ Defined status interpretations (healthy/degraded/critical)
- ✅ Workaround application logic
- ✅ Priority override rules for core issues

### 7. Health Check Dispatcher (`scripts/health-check-dispatcher.sh`)
- ✅ Created executable dispatcher with health checks
- ✅ Automatic workaround application
- ✅ Emergency task creation for critical issues
- ✅ Component-specific health verification

## How It Works

### Normal Flow
1. Task arrives in swarm-manager backlog
2. Dispatcher checks core health via `core-debugger status`
3. If healthy → proceed with task
4. If degraded → check for workarounds, apply if available
5. If critical → block task, create emergency core-fix task

### Core Issue Flow
1. Core issue detected (e.g., "vrooli: command not found")
2. Automatically routed to `core-debugger` (not `app-debugger`)
3. Gets 10x priority boost (e.g., 100 → 1000)
4. Checks workaround database for known fixes
5. Applies fix or escalates to Claude for analysis
6. Once fixed, normal work resumes

## Priority Calculation

```
Base Priority = Impact × Urgency × Success_Probability / Resource_Cost

Core Issue Priority = Base Priority × 10.0 (if core infrastructure)
                   × 0.5 (if workaround exists)
                   × 2.0 (if user reported)
                   × 5.0 (if critical severity)
```

## Key Benefits

1. **Prevents Cascading Failures**: Won't dispatch work if core is broken
2. **Automatic Prioritization**: Core issues get immediate attention
3. **Workaround Intelligence**: Known fixes applied automatically
4. **Zero Dependencies**: Core-debugger only needs claude-code
5. **Clear Separation**: Core vs app issues handled appropriately

## Configuration Examples

### Task Type Routing
```yaml
# In settings.yaml
task_type_scenarios:
  core-infrastructure: core-debugger  # Core Vrooli issues
  app-debug: app-debugger             # Application issues
  system-performance: system-monitor   # Performance issues
```

### Health Check Config
```yaml
health_check_before_dispatch:
  enabled: true
  block_on_critical: true
  components_to_check:
    - cli
    - orchestrator
    - resource-manager
    - setup
```

## Testing the Integration

Run the integration test:
```bash
cd scenarios/swarm-manager/tests
./test-core-debugger-integration.sh
```

Simulate a core issue:
```bash
# Create a core issue task
cat > tasks/backlog/manual/test-core.yaml << EOF
title: "Fix CLI command not found"
type: core-infrastructure
description: "vrooli: command not found when running scenarios"
EOF

# Run dispatcher with health check
./scripts/health-check-dispatcher.sh
```

## Monitoring

Check integration status:
```bash
# Verify core-debugger in registry
grep 'core-debugger' config/scenario-registry.yaml

# Check priority modifiers
grep 'core_infrastructure_issue' config/settings.yaml

# View health check logs
tail -f logs/dispatcher.log
```

## Future Enhancements

1. **Auto-healing**: Automatically apply fixes without human approval
2. **Pattern Learning**: Track which workarounds succeed most often
3. **Predictive Detection**: Identify issues before they become critical
4. **Cross-instance Sharing**: Share workarounds across Vrooli instances
5. **Performance Metrics**: Track mean time to resolution (MTTR)

## Troubleshooting

### If health checks aren't running:
- Ensure `core-debugger` CLI is in PATH
- Check `health_check_before_dispatch.enabled` is true
- Verify dispatcher script has execute permissions

### If core issues aren't getting priority:
- Check `priority_multiplier: 10` in scenario-registry
- Verify task type is `core-infrastructure`
- Ensure routing rules match issue keywords

### If workarounds aren't applied:
- Check workaround database exists: `data/workarounds/common.json`
- Verify error pattern matches workaround pattern
- Test manually: `core-debugger get-workaround "<error>"`

---

**Integration Status**: ✅ COMPLETE
**Date**: 2025-01-03
**Scenarios Integrated**: swarm-manager ↔ core-debugger