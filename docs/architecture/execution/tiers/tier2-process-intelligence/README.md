# Tier 2: Process Intelligence - RunStateMachine

**Purpose**: Navigator-agnostic routine execution with parallel coordination and state management

## 📋 Table of Contents

- [🏗️ Architecture Overview](./architecture.md) - Universal automation ecosystem and plug-and-play design
- [🧭 Navigator System](./navigators.md) - Universal interface and platform support
- [⚙️ Routine Types](./routine-types.md) - Single-step vs multi-step execution patterns
- [🎯 Core Responsibilities](./responsibilities.md) - Key capabilities and functions
- [📚 Routine Examples](./routine-examples/README.md) - Comprehensive collection of multi-step routine examples

## 🎯 Overview

The `RunStateMachine` is at the heart of Vrooli's ability to execute diverse automation routines. It represents Vrooli's core innovation: a **universal routine execution engine** that's completely agnostic to the underlying automation platform.

This creates an unprecedented **universal automation ecosystem** where workflows from different platforms can share and execute each other's routines, enabling the best automation workflows to be used anywhere, regardless of their original platform.

## 🔄 State Machine Lifecycle

The following diagram visualizes the RunStateMachine's lifecycle and the various states it transitions through while managing routine execution:

```mermaid
stateDiagram-v2
    [*] --> Idle

    %% ───────────────────────────────────  Initialisation  ──────────────────────────────────
    Idle --> Initializing            : initNewRun() / initExistingRun()
    Initializing --> LoadingRoutine  : Load routine definition
    LoadingRoutine --> ValidatingConfiguration : Validate run config
    ValidatingConfiguration --> SelectingNavigator : Choose appropriate navigator
    SelectingNavigator --> InitializingContext    : Setup execution context
    InitializingContext --> Initialized           : All systems ready

    %% ───────────────────────────────────  Core execution  ──────────────────────────────────
    Initialized --> Running           : runUntilDone() / runOneIteration()
    Running --> ExecutingBranches     : Process active branches
    ExecutingBranches --> EvaluatingConditions  : Check branch conditions
    EvaluatingConditions --> HandlingEvents      : Process boundary events

    %% NEW — deontic permission gate
    HandlingEvents --> MoiseDeonticGate          : checkDeontic()
    MoiseDeonticGate --> UpdatingProgress        : ✓ permitted
    MoiseDeonticGate --> EscalatingError         : ✗ forbidden (PermissionError)

    UpdatingProgress --> CheckingLimits          : Validate resource limits

    %% ───────────────────────────────────  Decision points  ─────────────────────────────────
    CheckingLimits --> LimitsExceeded            : Limits reached
    CheckingLimits --> BranchesCompleted         : All branches done
    CheckingLimits --> HasActiveBranches         : Branches still active
    CheckingLimits --> AllBranchesWaiting        : All branches waiting

    HasActiveBranches --> ExecutingBranches      : Continue execution
    AllBranchesWaiting --> WaitingForEvents      : Enter waiting state

    %% ───────────────────────────────  Event handling while waiting  ────────────────────────
    WaitingForEvents --> ProcessingTimeEvent     : Timer triggers
    WaitingForEvents --> ProcessingMessageEvent  : Message received
    WaitingForEvents --> ProcessingSignalEvent   : Signal received
    WaitingForEvents --> ProcessingErrorEvent    : Error boundary triggered

    ProcessingTimeEvent --> ReactivatingBranches
    ProcessingMessageEvent --> ReactivatingBranches
    ProcessingSignalEvent --> ReactivatingBranches
    ProcessingErrorEvent --> ErrorRecovery

    ReactivatingBranches --> ExecutingBranches

    %% ──────────────────────────────────  Error / recovery  ─────────────────────────────────
    ErrorRecovery --> RetryingExecution          : Retry strategy
    ErrorRecovery --> FallbackStrategy           : Use fallback
    ErrorRecovery --> EscalatingError            : Cannot recover

    RetryingExecution --> ExecutingBranches      : Retry ok
    RetryingExecution --> EscalatingError        : Retry failed
    FallbackStrategy  --> ExecutingBranches      : Fallback ok
    FallbackStrategy  --> EscalatingError        : Fallback failed

    %% ────────────────────────────────────  Terminals  ──────────────────────────────────────
    BranchesCompleted --> Completed
    LimitsExceeded    --> LimitsFailed
    EscalatingError   --> Failed

    %% ───────────────────────────────── Pause / resume  ─────────────────────────────────────
    Running           --> Pausing    : stopRun(PAUSED)
    ExecutingBranches --> Pausing
    WaitingForEvents  --> Pausing
    Pausing --> Paused
    Paused --> Resuming
    Resuming --> Running

    %% ───────────────────────────────── Cancellation  ───────────────────────────────────────
    Running           --> Cancelling : stopRun(CANCELLED)
    ExecutingBranches --> Cancelling
    WaitingForEvents  --> Cancelling
    Paused            --> Cancelling
    Cancelling --> Cancelled

    %% ─────────────────────────────── Navigator selection  ─────────────────────────────────
    SelectingNavigator --> BpmnNavigator
    SelectingNavigator --> LangchainNavigator
    SelectingNavigator --> TemporalNavigator
    SelectingNavigator --> CustomNavigator
    BpmnNavigator      --> InitializingContext
    LangchainNavigator --> InitializingContext
    TemporalNavigator  --> InitializingContext
    CustomNavigator    --> InitializingContext

    %% ───────────────────────────────────  Finals  ──────────────────────────────────────────
    Completed   --> [*]
    Failed      --> [*]
    LimitsFailed--> [*]
    Cancelled   --> [*]

    %% ─────────────────────────────────  Sub-state blocks  ──────────────────────────────────
    state WaitingForEvents {
        [*] --> EventListener
        EventListener --> TimerCheck       : Check timers
        EventListener --> MessageQueue     : Check messages
        EventListener --> SignalMonitor    : Check signals
        TimerCheck     --> EventListener
        MessageQueue   --> EventListener
        SignalMonitor  --> EventListener
    }

    state ExecutingBranches {
        [*] --> DeterminingConcurrency
        DeterminingConcurrency --> SequentialExecution : Resource constrained
        DeterminingConcurrency --> ParallelExecution   : Parallel allowed
        SequentialExecution --> BranchComplete
        ParallelExecution   --> BranchComplete
        BranchComplete      --> [*]
    }
```

## 🚀 Next Steps

Explore the detailed documentation in the sections above to understand:

- How the universal architecture enables cross-platform automation
- The navigator interface that makes any workflow platform compatible
- The different types of routines and their execution patterns
- The comprehensive responsibilities handled by the RunStateMachine

This modular design makes Vrooli the **universal execution layer** for automation - like how Kubernetes became the universal orchestration layer for containers, Vrooli becomes the universal orchestration layer for intelligent workflows. 