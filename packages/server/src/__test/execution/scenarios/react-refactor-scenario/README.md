# React Component Refactor Scenario

## Overview

This scenario demonstrates **leader delegation during complex refactoring tasks** in a React codebase. It tests the framework's ability to hand off leadership to domain specialists, maintain context isolation between subtasks, and coordinate multiple parallel refactoring efforts while preserving application functionality.

### Key Features

- **Sequential Leader Delegation**: Main coordinator hands leadership to specialists in sequence
- **Context Isolation**: Each specialist works independently without affecting others
- **Handback Protocol**: Leadership returns to coordinator after subtask completion
- **State Preservation**: Overall refactoring progress maintained across delegations
- **Parallel Work Capability**: Multiple specialists can work on isolated components

## Agent Architecture

```mermaid
graph TB
    subgraph RefactorSwarm[React Refactor Swarm]
        RC[Refactor Coordinator]
        SMS[State Management Specialist]
        TS[Testing Specialist]
        PS[Performance Specialist]
        SS[Styling Specialist]
        BB[(Blackboard)]
        
        RC -->|delegates to| SMS
        RC -->|delegates to| TS
        RC -->|delegates to| PS
        RC -->|delegates to| SS
        
        SMS -->|updates components| BB
        TS -->|updates tests| BB
        PS -->|optimizes| BB
        SS -->|migrates styles| BB
        
        RC -->|monitors progress| BB
    end
    
    subgraph AgentRoles[Agent Roles]
        RC_Role[Refactor Coordinator<br/>- Analyzes components<br/>- Plans refactor strategy<br/>- Delegates to specialists<br/>- Verifies integration]
        SMS_Role[State Management<br/>- Redux to Context API<br/>- Class state to hooks<br/>- Custom hook creation<br/>- State logic preservation]
        TS_Role[Testing Specialist<br/>- Enzyme to RTL<br/>- Test migration<br/>- Coverage maintenance<br/>- Behavior testing]
        PS_Role[Performance<br/>- React.memo<br/>- useMemo/useCallback<br/>- Render optimization<br/>- Performance metrics]
        SS_Role[Styling Specialist<br/>- CSS modules to styled<br/>- Theme system<br/>- Design tokens<br/>- Visual consistency]
    end
    
    RC_Role -.->|implements| RC
    SMS_Role -.->|implements| SMS
    TS_Role -.->|implements| TS
    PS_Role -.->|implements| PS
    SS_Role -.->|implements| SS
```

## Leader Delegation Flow

```mermaid
graph LR
    subgraph DelegationSequence[Leadership Delegation Sequence]
        Start[Coordinator<br/>Analyzes] --> D1[Delegate to<br/>State Mgmt]
        D1 --> W1[State Work<br/>Isolated Context]
        W1 --> H1[Handback to<br/>Coordinator]
        
        H1 --> D2[Delegate to<br/>Testing]
        D2 --> W2[Test Work<br/>Isolated Context]
        W2 --> H2[Handback to<br/>Coordinator]
        
        H2 --> D3[Delegate to<br/>Performance]
        D3 --> W3[Perf Work<br/>Isolated Context]
        W3 --> H3[Handback to<br/>Coordinator]
        
        H3 --> D4[Delegate to<br/>Styling]
        D4 --> W4[Style Work<br/>Isolated Context]
        W4 --> H4[Handback to<br/>Coordinator]
        
        H4 --> Verify[Integration<br/>Verification]
    end
    
    style Start fill:#e8f5e8
    style Verify fill:#e8f5e8
    style W1 fill:#e1f5fe
    style W2 fill:#f3e5f5
    style W3 fill:#fff3e0
    style W4 fill:#fce4ec
```

## Complete Event Flow

```mermaid
sequenceDiagram
    participant START as Swarm Start
    participant RC as Refactor Coordinator
    participant SMS as State Mgmt Specialist
    participant TS as Testing Specialist
    participant PS as Performance Specialist
    participant SS as Styling Specialist
    participant BB as Blackboard
    participant ES as Event System
    
    Note over START,ES: Initial Analysis Phase
    START->>RC: swarm/started
    RC->>RC: Execute component-analysis-routine
    RC->>BB: Store refactor_strategy, current_leader=RC
    
    Note over RC,SMS: State Management Delegation
    RC->>ES: Emit refactor/delegation_requested<br/>(type=state_management, leader=SMS)
    RC->>BB: Set active_subtask_leader=SMS
    
    ES->>SMS: refactor/delegation_requested
    SMS->>ES: Emit refactor/subtask_started
    SMS->>BB: Set current_leader=SMS
    SMS->>SMS: Execute state-refactor-routine
    SMS->>BB: Store refactored components, custom hooks
    SMS->>ES: Emit refactor/subtask_completed
    SMS->>BB: Append 'state_management' to completed_subtasks
    
    Note over RC,TS: Testing Delegation
    ES->>RC: refactor/subtask_completed (state_management)
    RC->>BB: Set current_leader=RC
    RC->>ES: Emit refactor/delegation_requested<br/>(type=testing, leader=TS)
    RC->>BB: Set active_subtask_leader=TS
    
    ES->>TS: refactor/delegation_requested
    TS->>BB: Set current_leader=TS
    TS->>TS: Execute test-migration-routine
    TS->>BB: Store migrated tests, coverage report
    TS->>ES: Emit refactor/subtask_completed
    
    Note over RC,PS: Performance Delegation
    ES->>RC: refactor/subtask_completed (testing)
    RC->>ES: Emit refactor/delegation_requested<br/>(type=performance, leader=PS)
    
    ES->>PS: refactor/delegation_requested
    PS->>BB: Set current_leader=PS
    PS->>PS: Execute performance-optimization-routine
    PS->>BB: Store optimizations, metrics
    PS->>ES: Emit refactor/subtask_completed
    
    Note over RC,SS: Styling Delegation
    ES->>RC: refactor/subtask_completed (performance)
    RC->>ES: Emit refactor/delegation_requested<br/>(type=styling, leader=SS)
    
    ES->>SS: refactor/delegation_requested
    SS->>BB: Set current_leader=SS
    SS->>SS: Execute styling-migration-routine
    SS->>BB: Store styled components, theme
    SS->>ES: Emit refactor/subtask_completed
    
    Note over RC,ES: Integration Verification
    ES->>RC: refactor/subtask_completed (styling)
    RC->>BB: Set current_leader=RC
    RC->>RC: Execute integration-verification-routine
    RC->>BB: Store final_verification
    RC->>ES: Emit refactor/all_complete
    RC->>BB: Set refactor_complete=true
```

## Context Isolation Pattern

```mermaid
graph TD
    subgraph ContextIsolation[Isolated Work Contexts]
        Global[Global Context<br/>- Component list<br/>- Dependencies<br/>- Overall strategy]
        
        StateCtx[State Context<br/>- Redux patterns<br/>- Lifecycle methods<br/>- State logic<br/>✅ Isolated]
        
        TestCtx[Test Context<br/>- Enzyme tests<br/>- Coverage data<br/>- Test patterns<br/>✅ Isolated]
        
        PerfCtx[Performance Context<br/>- Render patterns<br/>- Bottlenecks<br/>- Metrics<br/>✅ Isolated]
        
        StyleCtx[Styling Context<br/>- CSS modules<br/>- Visual styles<br/>- Theme needs<br/>✅ Isolated]
    end
    
    Global -->|read-only| StateCtx
    Global -->|read-only| TestCtx
    Global -->|read-only| PerfCtx
    Global -->|read-only| StyleCtx
    
    StateCtx -.->|no access| TestCtx
    TestCtx -.->|no access| PerfCtx
    PerfCtx -.->|no access| StyleCtx
    
    style StateCtx fill:#e1f5fe
    style TestCtx fill:#f3e5f5
    style PerfCtx fill:#fff3e0
    style StyleCtx fill:#fce4ec
```

## Blackboard State Evolution

```mermaid
graph LR
    subgraph StateEvolution[Blackboard State Through Delegations]
        Init[Initial State<br/>- target_components[15]<br/>- current_leader: RC<br/>- completed_subtasks: []]
        
        AfterState[After State Refactor<br/>+ state_refactor_result<br/>+ created_hooks[3]<br/>+ refactored_components<br/>+ completed: [state]]
        
        AfterTest[After Testing<br/>+ test_migration_result<br/>+ test_coverage: 82%<br/>+ refactored_tests<br/>+ completed: [state, test]]
        
        AfterPerf[After Performance<br/>+ performance_result<br/>+ render_improvements: 47%<br/>+ performance_optimized<br/>+ completed: [state, test, perf]]
        
        AfterStyle[After Styling<br/>+ styling_result<br/>+ design_tokens<br/>+ migrated_styles<br/>+ completed: [state, test, perf, style]]
        
        Final[Final State<br/>+ final_verification<br/>+ refactor_complete: true<br/>+ current_leader: RC]
    end
    
    Init --> AfterState
    AfterState --> AfterTest
    AfterTest --> AfterPerf
    AfterPerf --> AfterStyle
    AfterStyle --> Final
    
    style Init fill:#e1f5fe
    style Final fill:#e8f5e8
```

### Leadership Transition Tracking

| Step | Current Leader | Active Subtask Leader | Activity |
|------|---------------|----------------------|----------|
| 1 | Refactor Coordinator | - | Initial analysis |
| 2 | Refactor Coordinator | State Mgmt Specialist | Delegation |
| 3 | State Mgmt Specialist | State Mgmt Specialist | State refactoring |
| 4 | Refactor Coordinator | - | Handback after state |
| 5 | Refactor Coordinator | Testing Specialist | Delegation |
| 6 | Testing Specialist | Testing Specialist | Test migration |
| 7 | Refactor Coordinator | - | Handback after testing |
| 8 | Refactor Coordinator | Performance Specialist | Delegation |
| 9 | Performance Specialist | Performance Specialist | Optimization |
| 10 | Refactor Coordinator | - | Handback after perf |
| 11 | Refactor Coordinator | Styling Specialist | Delegation |
| 12 | Styling Specialist | Styling Specialist | Style migration |
| 13 | Refactor Coordinator | - | Final verification |

## Refactoring Strategy Breakdown

```mermaid
graph TD
    subgraph RefactorPhases[Refactoring Phases]
        Phase1[Phase 1: State Management<br/>- Redux → Context API<br/>- Class state → Hooks<br/>- Lifecycle → useEffect<br/>- Custom hooks creation]
        
        Phase2[Phase 2: Testing<br/>- Enzyme → RTL<br/>- Shallow → Full rendering<br/>- API queries → User queries<br/>- Coverage maintenance]
        
        Phase3[Phase 3: Performance<br/>- Identify re-renders<br/>- Add React.memo<br/>- useMemo for calculations<br/>- useCallback for functions]
        
        Phase4[Phase 4: Styling<br/>- CSS modules → styled<br/>- Create theme system<br/>- Design tokens<br/>- Scoped styles]
    end
    
    Phase1 --> Phase2
    Phase2 --> Phase3
    Phase3 --> Phase4
    
    style Phase1 fill:#e1f5fe
    style Phase2 fill:#f3e5f5
    style Phase3 fill:#fff3e0
    style Phase4 fill:#fce4ec
```

## Component Transformation Example

```mermaid
graph LR
    subgraph BeforeRefactor[Before: UserProfile Component]
        ClassComp[Class Component<br/>- Redux connect()<br/>- componentDidMount<br/>- Local state<br/>- CSS modules]
    end
    
    subgraph AfterRefactor[After: UserProfile Component]
        FuncComp[Functional Component<br/>- useSelector/useDispatch<br/>- useEffect<br/>- useState/useAuth<br/>- Styled components]
    end
    
    subgraph Improvements[Improvements]
        Imp1[✅ 30% smaller bundle]
        Imp2[✅ 50% fewer renders]
        Imp3[✅ Better test coverage]
        Imp4[✅ Type-safe styles]
    end
    
    ClassComp --> FuncComp
    FuncComp --> Improvements
    
    style ClassComp fill:#ffebee
    style FuncComp fill:#e8f5e8
```

## Expected Scenario Outcomes

### Success Path
1. **Coordinator Analysis**: Identifies 15 components needing refactoring
2. **State Delegation**: SMS converts Redux/class state to hooks (creates 3 custom hooks)
3. **Testing Delegation**: TS migrates 45 tests, improves coverage to 82%
4. **Performance Delegation**: PS reduces renders by 47% with optimizations
5. **Styling Delegation**: SS creates unified theme system, migrates all styles
6. **Integration Verification**: RC confirms all components work together

### Success Criteria

```json
{
  "requiredEvents": [
    "refactor/delegation_requested",
    "refactor/subtask_started",
    "refactor/subtask_completed",
    "refactor/all_complete"
  ],
  "blackboardState": {
    "refactor_complete": "true",
    "completed_subtasks": ["state_management", "testing", "performance", "styling"],
    "current_leader": "refactor-coordinator",
    "created_hooks": ["useAuth", "useCart", "useProductData"]
  },
  "leadershipTransitions": {
    "minimumHandoffs": 4,
    "correctSequence": true,
    "contextIsolation": true,
    "handbackProtocol": "verified"
  }
}
```

## Running the Scenario

### Prerequisites
- Execution test framework with leader delegation support
- SwarmContextManager configured for leadership transitions
- Mock routine responses for refactoring operations

### Execution Steps

1. **Initialize Scenario**
   ```typescript
   const scenario = new ScenarioFactory("react-refactor-scenario");
   await scenario.setupScenario();
   ```

2. **Configure Components**
   ```typescript
   blackboard.set("target_components", [
     "UserProfile", "ProductList", "ShoppingCart", // ... 15 total
   ]);
   ```

3. **Start Refactoring**
   ```typescript
   await scenario.emitEvent("swarm/started", {
     task: "refactor-react-components"
   });
   ```

4. **Monitor Delegations**
   - Track `current_leader` changes
   - Verify `active_subtask_leader` assignments
   - Monitor `completed_subtasks` accumulation
   - Check specialist work isolation

### Debug Information

Key monitoring points:
- `current_leader` - Active leadership holder
- `active_subtask_leader` - Delegated specialist
- `completed_subtasks` - Progress tracking
- `refactor_strategy` - Overall plan
- Component-specific results (hooks, tests, optimizations, styles)

## Technical Implementation Details

### Leader Delegation Protocol
```typescript
interface LeadershipTransition {
  from: string;              // Current leader
  to: string;                // New leader
  context: "isolated";       // Context handling
  handbackTrigger: string;   // Event for return
}
```

### Resource Configuration
- **Max Credits**: 1.5B micro-dollars (complex refactoring)
- **Max Duration**: 10 minutes (sequential specialists)
- **Resource Quota**: 30% GPU, 16GB RAM (code analysis heavy)

### Context Isolation Rules
1. Specialists cannot access other specialists' work
2. Global read-only access to component list
3. Work results stored in isolated blackboard sections
4. No cross-contamination between refactoring phases

## Real-World Applications

### Common Refactoring Patterns
1. **Framework Migration**: Angular to React, Vue to React
2. **Version Upgrades**: React 16 to 18 with concurrent features
3. **Architecture Changes**: MVC to component-based
4. **Performance Optimization**: Addressing React DevTools warnings
5. **Testing Strategy**: Shifting from implementation to behavior testing

### Benefits of Leader Delegation
- **Parallel Work**: Multiple specialists can prepare while waiting
- **Domain Expertise**: Each specialist focuses on their strength
- **Clean Boundaries**: No conflicts between different refactoring aspects
- **Progress Tracking**: Clear handoff points for monitoring

This scenario demonstrates how complex software engineering tasks can be decomposed and delegated to specialists while maintaining coordination and ensuring successful integration - a critical pattern for large-scale refactoring projects.