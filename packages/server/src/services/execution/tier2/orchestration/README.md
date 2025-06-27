# Tier 2 Orchestration Components

This directory contains the core orchestration logic for Tier 2 Process Intelligence.

## Architecture Overview

```
┌─────────────────────────┐
│   TierTwoOrchestrator   │  ← Public API (../tierTwoOrchestrator.ts)
│  (Main Entry Point)     │
└───────────┬─────────────┘
            │ creates & delegates to
            ▼
┌─────────────────────────┐
│ UnifiedRunStateMachine  │  ← Core Implementation
│  (Complete Tier 2 Logic)│
└───────────┬─────────────┘
            │ provides interface via
            ▼
    getRunOrchestrator()     ← Returns compatible object
    { createRun, startRun,      (NOT RunOrchestrator instance)
      updateProgress, ... }
```

## Component Descriptions

### UnifiedRunStateMachine ✅ (Active)
- **File**: `unifiedRunStateMachine.ts`
- **Purpose**: Complete implementation of Tier 2 process intelligence
- **Status**: Production - This is the core of Tier 2
- **Features**:
  - Comprehensive state machine (NAVIGATOR_SELECTION → PLANNING → EXECUTING → etc.)
  - Universal navigator support (Native, BPMN, Langchain, Temporal)
  - MOISE+ organizational validation
  - Parallel branch coordination
  - Swarm context inheritance
  - Resource management & checkpointing

### RunOrchestrator ⚠️ (Legacy)
- **File**: `runOrchestrator.ts`
- **Purpose**: Simple CRUD operations for run management
- **Status**: Not used in production - kept for reference
- **Note**: UnifiedRunStateMachine implements this functionality internally

### Deprecated Components 🗑️ (Removed)
The following components have been consolidated into UnifiedRunStateMachine:
- ~~TierTwoRunStateMachine~~ - Basic state machine (removed)
- ~~BranchCoordinator~~ - Parallel execution (removed)
- ~~StepExecutor~~ - Step execution logic (removed)
- ~~ContextManager~~ - Context management (removed)
- ~~CheckpointManager~~ - Persistence (removed)

## Key Design Decisions

### Why UnifiedRunStateMachine?
1. **Consolidation**: Merges 5+ fragmented components into one cohesive implementation
2. **Completeness**: Implements the full documented tier 2 architecture
3. **Maintainability**: Single source of truth for all tier 2 logic
4. **Extensibility**: Clean state machine pattern allows easy additions

### Why Keep RunOrchestrator?
1. **Reference**: Shows the interface contract for run management
2. **Testing**: Can be used in isolation for unit tests
3. **Documentation**: Helps understand the separation of concerns

### Why getRunOrchestrator() Pattern?
1. **Backward Compatibility**: Existing code expecting RunOrchestrator continues to work
2. **Single Implementation**: Avoids duplicating logic between classes
3. **Flexibility**: Can change internal implementation without breaking external contracts

## Usage Examples

### Creating and Starting a Run
```typescript
// Via TierTwoOrchestrator (recommended)
const tier2 = new TierTwoOrchestrator(logger, eventBus, tier3);
const result = await tier2.execute({
    routineId: "routine123",
    userId: "user456",
    inputs: { data: "test" }
});

// Via UnifiedRunStateMachine directly (internal use)
const orchestrator = unifiedStateMachine.getRunOrchestrator();
const run = await orchestrator.createRun({
    routineId: "routine123",
    userId: "user456", 
    inputs: { data: "test" }
});
```

## Future Considerations

1. **RunOrchestrator Removal**: Once we're confident in the unified architecture, 
   RunOrchestrator.ts can be removed entirely.

2. **Interface Extraction**: The run management interface could be extracted to a 
   separate TypeScript interface for better type safety.

3. **Further Consolidation**: Some navigation and validation logic could potentially
   be moved into the state machine for even tighter integration.