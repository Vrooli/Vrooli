# automation/state

Package `state` provides namespace-aware variable management for workflow execution.

## Overview

This package was extracted from `automation/executor/flow_utils.go` to provide clear responsibility boundaries and enable reuse across the automation stack. It centralizes all state management, variable interpolation, and expression evaluation logic.

## Key Types

### ExecutionState

The primary type for managing workflow execution state. It separates runtime state into three namespaces:

- `@store/` - Mutable runtime state, writable via setVariable/storeResult
- `@params/` - Read-only input parameters from execution request or subflow call
- `@env/` - Read-only environment configuration from project settings

```go
// Create new state
state := state.New(initialStore, initialParams, env)

// Get/set values
state.Set("key", value)
val, ok := state.Get("key")

// Resolve dot notation paths
val, ok := state.Resolve("user.profile.name")

// Namespace-aware resolution
val, ok := state.ResolveNamespaced("params", "userId")
```

### Interpolator

Handles variable substitution in workflow instructions using `${var}` and `{{var}}` syntax.

```go
interp := state.NewInterpolator(execState)

// Interpolate strings
result := interp.InterpolateString("Hello ${@store/name}!")

// Interpolate instructions
interpolatedInstr := interp.InterpolateInstruction(instr)

// Evaluate expressions
result, ok := interp.EvaluateExpression("${count} > 5")
```

### Helper Functions

Value extraction and coercion utilities:

- `StringValue(m, key)` - Extract string from map
- `IntValue(m, key)` - Extract int with coercion
- `FloatValue(m, key)` - Extract float64 with coercion
- `BoolValue(m, key)` - Extract bool with coercion
- `CoerceArray(raw)` - Convert various types to `[]any`

Loop helpers:
- `ExtractLoopItems(params, state)` - Extract loop items from params
- `EvaluateLoopCondition(params, state)` - Evaluate while/until conditions
- `NormalizeVariableValue(value, valueType)` - Normalize variable by declared type

## Migration Guide

### From flowState to ExecutionState

Old code using `flowState`:
```go
state := newFlowState(seed)
val, ok := state.get("key")
state.set("key", value)
```

New code using `state.ExecutionState`:
```go
state := state.NewFromStore(seed)
val, ok := state.Get("key")
state.Set("key", value)
```

### From inline interpolation to Interpolator

Old code:
```go
result := interpolateString(template, flowState)
instr := interpolateInstruction(instr, flowState)
```

New code:
```go
interp := state.NewInterpolator(execState)
result := interp.InterpolateString(template)
instr := interp.InterpolateInstruction(instr)
```

## Thread Safety

`ExecutionState` uses a mutex to protect concurrent access. All public methods are safe to call from multiple goroutines.

## Design Principles

This package follows the architectural principles from:
- `boundary-of-responsibility-enforcement.md` - Clear separation of state management from execution
- `change-axis-and-evolution-resilience-audit.md` - Localized change for namespace additions
- `domain-compression.md` - Single canonical implementation of state management
