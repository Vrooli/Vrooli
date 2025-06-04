# Tier Interface Contracts

This document is the **authoritative source** for defining the conceptual contracts and responsibilities of interfaces between Vrooli's three execution tiers. It outlines how tiers interact, what data they exchange, and the expected behaviors. The actual TypeScript interface definitions are centralized in [types/core-types.ts](types/core-types.ts).

**Prerequisites**: 
- Read [README.md](README.md) for overall architectural context and navigation.
- Understand the [Communication Patterns](communication-patterns.md) that these interfaces facilitate.
- Familiarize yourself with the [Centralized Type System](types/core-types.ts) where all data structures and type definitions (e.g., `RoutineExecutionRequest`, `StepExecutionResult`, `SecurityContext`) are specified.

## Interface Philosophy

- **Clear Separation of Concerns**: Each tier has distinct responsibilities, and interfaces reflect this separation.
- **Contract-Based Interaction**: Interactions are based on well-defined contracts (the interfaces), not implementation details.
- **Standardized Data Structures**: All data exchanged between tiers (requests, responses, contexts) uses structures defined in [types/core-types.ts](types/core-types.ts).
- **Security Context Propagation**: The `SecurityContext` is a mandatory part of inter-tier communication, ensuring security is maintained. See [Security Boundaries](security-boundaries.md) for details.
- **Error Handling**: Errors are propagated according to the [Error Propagation and Recovery Framework](error-propagation.md), using standardized error types from [types/core-types.ts](types/core-types.ts).

## Tier 1 (Coordination) → Tier 2 (Process) Interface

**Primary Communication Pattern**: [MCP Tool Communication](communication-patterns.md#1-mcp-tool-communication-tier-1--tier-2) via `CompositeToolRunner`, often invoking built-in tools like `run_routine`.

**Key Interactions & Data Structures (defined in [types/core-types.ts](types/core-types.ts))**:

1.  **Execute Routine (`run_routine` tool)**
    *   **Request from T1 to T2**: Tier 1 (e.g., `SwarmStateMachine`) initiates routine execution.
        *   Data: `RoutineExecutionRequest` (includes `routineId`, `inputs`, `resourceLimits`, `securityContext`, `priority`, `approvalConfig`).
        *   Mechanism: Typically via an MCP tool call like `run_routine` (see `RunRoutineMcpTool` in types).
    *   **Response from T2 to T1**: Tier 2 (`RunStateMachine`) returns the outcome.
        *   Data: `RoutineExecutionResult` (includes `runId`, `status`, `outputs`, `resourceUsage`, `errors`). For asynchronous calls, T2 might initially return a `runId` and status, with the full result delivered later via an event or another mechanism.
        *   Mechanism: MCP tool response (`McpToolResponse`).

2.  **Other Coordination Actions (Examples via MCP Tools)**:
    *   `send_message`: Tier 1 instructs Tier 2 (or another agent via T2) to send a message.
    *   `spawn_swarm`: Tier 1 requests the creation of a new swarm.
    *   `update_swarm_shared_state`: Tier 1 updates the shared blackboard accessible by agents in a swarm, managed by Tier 2.
    *   `resource_manage`: Tier 1 queries or adjusts resource allocations or definitions, potentially impacting Tier 2/3 operations.

**Responsibilities**:
*   **Tier 1**: Assembles `RoutineExecutionRequest`, ensures `SecurityContext` is appropriate, handles high-level outcomes and errors.
*   **Tier 2**: Validates request and `SecurityContext`, manages `RunContext` lifecycle, orchestrates routine steps (delegating to Tier 3), aggregates results, and manages resources within its allocated budget.

## Tier 2 (Process) → Tier 3 (Execution) Interface

**Primary Communication Pattern**: [Direct Service Interface](communication-patterns.md#2-direct-service-interface-tier-2--tier-3).

**Key Interactions & Data Structures (defined in [types/core-types.ts](types/core-types.ts))**:

1.  **Execute Step**
    *   **Request from T2 to T3**: Tier 2 (`RunStateMachine`) requests execution of a single routine step.
        *   Data: `StepExecutionRequest` (includes `stepId`, `runId`, `context` (`RunContext`), `strategy`, `inputData`, `securityContext`, `availableCredits`, `timeoutMs`).
    *   **Response from T3 to T2**: Tier 3 (`UnifiedExecutor`) returns the result of the step execution.
        *   Data: `StepExecutionResult` (includes `status`, `output`, `contextUpdates`, `resourceUsage`, `qualityScore`, `recoverableErrors`).

**Responsibilities**:
*   **Tier 2**: Prepares `StepExecutionRequest`, manages overall routine flow, handles step outcomes (including errors escalated by T3), updates `RunContext` based on `contextUpdates` from T3, and decides on the next step.
*   **Tier 3**: Validates request and `SecurityContext`, executes the step according to the specified `strategy`, invokes tools/models, tracks fine-grained `resourceUsage`, performs output validation, and handles step-level errors (attempting local recovery or escalating).

## Cross-Tier Eventing Interface

**Primary Communication Pattern**: [Event-Driven Messaging](communication-patterns.md#3-event-driven-messaging-all-tiers).
- **Mechanism**: All tiers can publish events to and subscribe to events from the [Event Bus Protocol](event-bus-protocol.md).
- **Data**: Various `EventPayload` types defined in [types/core-types.ts](types/core-types.ts) (e.g., `RunStatusEvent`, `StepCompletionEvent`, `ResourceUsageEvent`, `SecurityAlertEvent`).
- **Purpose**: Asynchronous notifications, monitoring, logging, and coordination that doesn't fit the request/response model (e.g., alerting a `SecurityAgent` about a potential threat, notifying Tier 1 of routine progress).

## State Synchronization Interface

**Primary Communication Pattern**: [State Synchronization](communication-patterns.md#4-state-synchronization-all-tiers).
- **Mechanism**: Primarily through the management and propagation of `RunContext` and cached configurations. Details in [State Synchronization](state-synchronization.md).
- **Data**: `RunContext`, `ContextVariable`, `CacheInvalidationEvent` (via Event Bus).
- **Purpose**: Ensuring consistent view of state where necessary, optimizing performance through caching, and managing context inheritance.

## Error Handling Across Interfaces

All inter-tier communication must adhere to the [Error Propagation and Recovery Framework](error-propagation.md).
- Errors are represented by the `ExecutionError` structure.
- Severity and type are classified using the defined decision trees.
- Recovery strategies are selected and executed as per the framework.

## Implementation Validation

Validation of these interface contracts in action is covered by the [Integration Map and Validation Document](integration-map.md). This includes ensuring:
- Correct data structures (`RoutineExecutionRequest`, `StepExecutionResult`, etc.) are used.
- `SecurityContext` is correctly propagated and validated at each tier.
- Resource limits are respected and usage is reported accurately.
- Errors are propagated and handled as per the [Error Propagation and Recovery Framework](error-propagation.md).
- Communication patterns are correctly implemented for each interface.

## Related Documentation
- **[README.md](README.md)**: Overall navigation for the communication architecture.
- **[Centralized Type System](types/core-types.ts)**: The single source of truth for all TypeScript interface definitions and data types.
- **[Communication Patterns](communication-patterns.md)**: Describes the underlying patterns these interfaces implement.
- **Authoritative Documents for Cross-Cutting Concerns**:
    - [Security Boundaries](security-boundaries.md)
    - [Error Propagation and Recovery Framework](error-propagation.md)
    - [State Synchronization](state-synchronization.md)
    - [Resource Coordination](resource-coordination.md)
- **[Integration Map and Validation Document](integration-map.md)**: For testing and validating inter-tier communication.

This document clarifies the expected interactions and responsibilities at each tier boundary, relying on the [Centralized Type System](types/core-types.ts) for precise data structure definitions. 