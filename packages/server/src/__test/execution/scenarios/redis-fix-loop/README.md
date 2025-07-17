# Redis Fix Loop Scenario

## Overview

This scenario demonstrates a **progressive retry coordination pattern** for infrastructure problem resolution. It models a realistic Redis connection failure where three specialized agents work together to diagnose, fix, and validate the connection through multiple retry attempts with increasing sophistication.

### Key Features

- **Progressive Fix Strategy**: Each attempt applies more sophisticated fixes
- **Retry Coordination**: Automatic retry loop with attempt limiting (max 5)
- **Comprehensive Validation**: Multi-step validation with detailed diagnostics
- **Event-Driven Workflow**: Agents coordinate through custom Redis events
- **Blackboard History**: Complete audit trail of fixes and validations

## Agent Architecture

```mermaid
graph TB
    subgraph RedisSwarm[Redis Fix Loop Swarm]
        FC[Fixer Agent]
        VAL[Validator Agent]
        COORD[Coordinator Agent]
        BB[(Blackboard)]
        
        FC -->|applies fixes| BB
        VAL -->|validates results| BB
        COORD -->|monitors progress| BB
        COORD -->|triggers retries| FC
    end
    
    subgraph AgentRoles[Agent Roles]
        FC_Role[Fixer Specialist<br/>- Progressive fix strategy<br/>- Connection pool to Retry logic to Keepalive<br/>- Escalate complexity per attempt]
        VAL_Role[Validator Monitor<br/>- Comprehensive testing<br/>- Stability and performance checks<br/>- Detailed diagnostic reports]
        COORD_Role[Coordinator<br/>- Retry loop management<br/>- Success failure detection<br/>- Attempt limiting max 5]
    end
    
    FC_Role -.->|implements| FC
    VAL_Role -.->|implements| VAL
    COORD_Role -.->|implements| COORD
```

## Progressive Fix Strategy

The fixer agent applies increasingly sophisticated fixes based on attempt number:

```mermaid
graph LR
    subgraph FixEscalation[Progressive Fix Escalation]
        A1[Attempt 1<br/>Connection Pool<br/>Settings]
        A2[Attempt 2<br/>Retry Logic<br/>+ Backoff]
        A3[Attempt 3<br/>Connection<br/>Keepalive]
        A4[Attempt 4<br/>Advanced<br/>Diagnostics]
        A5[Attempt 5<br/>Comprehensive<br/>Overhaul]
        
        A1 --> A2
        A2 --> A3
        A3 --> A4
        A4 --> A5
    end
    
    style A1 fill:#e1f5fe
    style A2 fill:#f3e5f5
    style A3 fill:#e8f5e8
    style A4 fill:#fff3e0
    style A5 fill:#ffebee
```

## Complete Event Flow

```mermaid
sequenceDiagram
    participant INIT as Initial Trigger
    participant FIXER as Fixer Agent
    participant VAL as Validator Agent
    participant COORD as Coordinator Agent
    participant BB as Blackboard
    participant ES as Event System
    
    Note over INIT,ES: Initial Redis Failure
    INIT->>FIXER: custom/redis/fix_requested (attempt=1)
    
    Note over FIXER,BB: Fix Application
    FIXER->>FIXER: Execute redis-connection-fixer
    FIXER->>BB: Store current_fix, fix_attempt=1
    FIXER->>ES: Emit custom/redis/ready_for_validation
    
    Note over VAL,BB: Validation Testing
    ES->>VAL: custom/redis/ready_for_validation
    VAL->>VAL: Execute redis-validation-workflow
    VAL->>BB: Store validation_status={success: false}
    VAL->>ES: Emit custom/redis/validation_complete
    
    Note over COORD,ES: Retry Coordination
    ES->>COORD: custom/redis/validation_complete
    COORD->>COORD: Check: success=false, attempt < 5
    COORD->>COORD: Execute retry-coordinator
    COORD->>BB: Increment total_retries
    COORD->>ES: Emit custom/redis/fix_requested (attempt=2)
    
    Note over FIXER,BB: Second Fix Attempt
    ES->>FIXER: custom/redis/fix_requested (attempt=2)
    FIXER->>FIXER: Execute redis-connection-fixer (escalated)
    FIXER->>BB: Store current_fix=retry logic, fix_attempt=2
    FIXER->>ES: Emit custom/redis/ready_for_validation
    
    Note over VAL,ES: Second Validation
    ES->>VAL: custom/redis/ready_for_validation
    VAL->>VAL: Execute redis-validation-workflow
    VAL->>BB: Store validation_status={success: false, partial improvement}
    VAL->>ES: Emit custom/redis/validation_complete
    
    Note over COORD,ES: Third Attempt Coordination
    ES->>COORD: custom/redis/validation_complete  
    COORD->>COORD: Check: success=false, attempt < 5
    COORD->>ES: Emit custom/redis/fix_requested (attempt=3)
    
    Note over FIXER,BB: Third Fix Attempt
    ES->>FIXER: custom/redis/fix_requested (attempt=3)
    FIXER->>FIXER: Execute redis-connection-fixer (keepalive)
    FIXER->>BB: Store current_fix=keepalive, fix_attempt=3
    FIXER->>ES: Emit custom/redis/ready_for_validation
    
    Note over VAL,ES: Successful Validation
    ES->>VAL: custom/redis/ready_for_validation
    VAL->>VAL: Execute redis-validation-workflow
    VAL->>BB: Store validation_status={success: true}
    VAL->>ES: Emit custom/redis/validation_complete
    
    Note over COORD,ES: Success Detection
    ES->>COORD: custom/redis/validation_complete
    COORD->>COORD: Check: success=true
    COORD->>ES: Emit custom/redis/fix_successful
    
    Note over ES: Workflow Complete
```

## Blackboard State Evolution

The blackboard accumulates comprehensive state through each retry cycle:

```mermaid
graph TD
    subgraph BlackboardProg[Blackboard Progression]
        Initial[Initial State<br/>last_error_logs]
        
        Attempt1[After Attempt 1<br/>+ current_fix<br/>+ fix_attempt=1<br/>+ fix_history]
        
        Validation1[After Validation 1<br/>+ validation_status<br/>+ validation_history]
        
        Retry1[After Retry 1<br/>+ total_retries=1<br/>+ retry_history]
        
        Attempt2[After Attempt 2<br/>+ current_fix updated<br/>+ fix_attempt=2]
        
        Success[Final Success<br/>+ validation_status.success=true<br/>+ success_timestamp<br/>+ success_event_id]
        
        MaxRetries[Alternative: Max Retries<br/>+ failure_event_id<br/>+ total_attempts=5]
    end
    
    Initial --> Attempt1
    Attempt1 --> Validation1
    Validation1 --> Retry1
    Retry1 --> Attempt2
    Attempt2 --> Success
    Retry1 -.->|if attempts >= 5| MaxRetries
```

### Blackboard Data Structure

| Key | Type | Purpose | Updated By |
|-----|------|---------|------------|
| `current_fix` | string | Latest fix description | Fixer Agent |
| `fix_attempt` | number | Current attempt number | Fixer Agent |
| `fix_history[]` | array | All fix attempts | Fixer Agent |
| `validation_status` | object | Latest validation result | Validator Agent |
| `validation_history[]` | array | All validation results | Validator Agent |
| `total_retries` | number | Retry count | Coordinator Agent |
| `retry_history[]` | array | Retry coordination details | Coordinator Agent |
| `success_timestamp` | string | Success completion time | Coordinator Agent |

## Agent Behavior Patterns

### Fixer Agent (Specialist)

```mermaid
graph TD
    subgraph FixerBehaviors[Fixer Agent Behaviors]
        FR[custom/redis/fix_requested] --> Apply[Execute redis-connection-fixer]
        Apply --> BB1[Blackboard Output:<br/>- current_fix<br/>- fix_attempt<br/>- append fix_history]
        
        FR --> Emit[Emit ready_for_validation]
        Emit --> BB2[Blackboard Output:<br/>- validation_request_id]
    end
    
    subgraph FixLogic[Progressive Fix Logic]
        Attempt1[Attempt 1:<br/>Connection Pool Settings]
        Attempt2[Attempt 2:<br/>Retry Logic + Backoff]
        Attempt3[Attempt 3:<br/>Connection Keepalive]
        Attempt4[Attempt 4+:<br/>Advanced Diagnostics]
    end
    
    Apply -.->|escalates complexity| Attempt1
    Attempt1 --> Attempt2
    Attempt2 --> Attempt3
    Attempt3 --> Attempt4
```

### Validator Agent (Monitor)

```mermaid
graph TD
    subgraph ValidatorBehaviors[Validator Agent Behaviors]
        RFV[custom/redis/ready_for_validation] --> Test[Execute redis-validation-workflow]
        Test --> BB1[Blackboard Output:<br/>- validation_status<br/>- append validation_history]
        
        RFV --> Emit[Emit validation_complete]
        Emit --> BB2[Blackboard Output:<br/>- validation_complete_id<br/>- increment total_validations]
    end
    
    subgraph ValidationTests[Validation Tests]
        Basic[Basic Connection Test]
        Ping[Redis Ping Test]
        Ops[Operation Tests]
        Stress[Stress Testing]
        Edge[Edge Case Testing]
    end
    
    Test -.->|comprehensive testing| Basic
    Basic --> Ping
    Ping --> Ops
    Ops --> Stress
    Stress --> Edge
```

### Coordinator Agent (Coordinator)

```mermaid
graph TD
    subgraph CoordDecision[Coordinator Agent Decision Logic]
        VC[custom/redis/validation_complete] --> Check{Validation Result?}
        
        Check -->|success=false, attempt<5| Retry[Execute retry-coordinator]
        Check -->|success=true| Success[Emit fix_successful]
        Check -->|attempt>=5, success=false| Failure[Emit fix_failed]
        
        Retry --> BB1[Blackboard Output:<br/>- increment total_retries<br/>- append retry_history]
        Retry --> EmitRetry[Emit fix_requested<br/>attempt+1]
        
        Success --> BB2[Blackboard Output:<br/>- success_event_id<br/>- success_timestamp]
        
        Failure --> BB3[Blackboard Output:<br/>- failure_event_id]
    end
```

## Retry Loop Decision Tree

```mermaid
graph TD
    Start[Validation Complete] --> CheckSuccess{Validation Success?}
    
    CheckSuccess -->|Yes| EmitSuccess[Emit fix_successful]
    CheckSuccess -->|No| CheckAttempts{Attempts < 5?}
    
    CheckAttempts -->|Yes| Retry[Execute retry-coordinator]
    CheckAttempts -->|No| EmitFailure[Emit fix_failed]
    
    Retry --> UpdateBlackboard[Update retry counters]
    UpdateBlackboard --> EmitRetry[Emit fix_requested<br/>attempt + 1]
    
    EmitRetry --> WaitForFix[Wait for next fix attempt]
    WaitForFix --> Start
    
    EmitSuccess --> EndSuccess[Success - Workflow Complete]
    EmitFailure --> EndFailure[Failure - Max Retries Exceeded]
    
    style EmitSuccess fill:#e8f5e8
    style EmitFailure fill:#ffebee
```

## Routine Coordination Architecture

```mermaid
graph LR
    subgraph FixerRoutines[Fixer Routines]
        R1[redis-connection-fixer<br/>RoutineGenerate]
    end
    
    subgraph ValidatorRoutines[Validator Routines]
        R2[redis-validation-workflow<br/>RoutineGenerate]
    end
    
    subgraph CoordinatorRoutines[Coordinator Routines]
        R3[retry-coordinator<br/>RoutineGenerate]
    end
    
    subgraph CoordInfra[Coordination Infrastructure]
        BB[(Blackboard<br/>State Management)]
        Events[Event System<br/>Redis Workflow Events]
    end
    
    R1 --> BB
    R2 --> BB
    R3 --> BB
    
    R1 --> Events
    R2 --> Events
    R3 --> Events
    
    Events --> R1
    Events --> R2
    Events --> R3
```

## Expected Scenario Outcomes

### Success Path (3 Attempts)
1. **Attempt 1**: Connection pool settings → Validation fails (timeout)
2. **Attempt 2**: Retry logic + backoff → Validation fails (partial improvement)
3. **Attempt 3**: Connection keepalive → Validation succeeds ✅

### Alternative Paths
- **Early Success**: Fix works on attempt 1 or 2
- **Maximum Retries**: All 5 attempts fail, emit `fix_failed`
- **Partial Success**: Progressive improvement but never fully succeeds

### Success Criteria

```json
{
  "requiredEvents": [
    "custom/redis/fix_requested",
    "custom/redis/ready_for_validation", 
    "custom/redis/validation_complete",
    "custom/redis/fix_successful"
  ],
  "blackboardState": {
    "validation_status": "exists",
    "fix_attempt": ">=1",
    "current_fix": "exists", 
    "total_validations": ">=1"
  },
  "routineCallsMin": 6,
  "agentCoordination": true
}
```

## Mock Response Strategy

The scenario uses **progressive improvement mocks** that demonstrate realistic fix progression:

### Fix Responses (Progressive Complexity)
1. **Attempt 1**: "Updated connection pool settings"
2. **Attempt 2**: "Added retry logic with exponential backoff"  
3. **Attempt 3**: "Implemented connection keepalive"

### Validation Responses (Gradual Improvement)
1. **Attempt 1**: `{success: false, logs: "Connection timeout"}`
2. **Attempt 2**: `{success: false, logs: "Partial improvement"}`
3. **Attempt 3**: `{success: true, logs: "All operations successful"}`

## Running the Scenario

### Prerequisites
- Execution test framework operational
- SwarmContextManager configured
- Mock routine response system active

### Execution Steps

1. **Initialize Scenario**
   ```typescript
   const scenario = new ScenarioFactory("redis-fix-loop");
   await scenario.setupScenario();
   ```

2. **Trigger Initial Fix Request**
   ```typescript
   await scenario.emitEvent("custom/redis/fix_requested", {
     attemptNumber: 1,
     previousResult: null,
     retryReason: "Initial Redis connection failure detected"
   });
   ```

3. **Monitor Progress**
   - Watch blackboard for `fix_attempt` progression
   - Monitor `validation_status` changes
   - Track `total_retries` and `total_validations` counters

4. **Validate Completion**
   - Confirm `custom/redis/fix_successful` event
   - Verify final `validation_status.success = true`
   - Check complete audit trail in blackboard

### Debug Monitoring

Track these blackboard keys for troubleshooting:
- `current_fix` - Latest fix description
- `fix_attempt` - Current attempt number  
- `validation_status` - Latest validation result
- `total_retries` - Retry coordination count
- `fix_history[]` - Complete fix audit trail
- `validation_history[]` - Complete validation audit trail

## Technical Implementation Details

### Event Quality of Service
- **QoS 1**: All coordination events (at-least-once delivery)
- **No QoS 2**: This scenario doesn't require exactly-once semantics

### Resource Constraints
- **Max Credits**: 500M micro-dollars ($500)
- **Max Duration**: 5 minutes  
- **Resource Quota**: 20% GPU, 16GB RAM, 4 CPU cores

### Custom Event Schema
All events use the `custom/redis/*` namespace:
- `custom/redis/fix_requested` - Trigger new fix attempt
- `custom/redis/ready_for_validation` - Fix applied, ready for testing
- `custom/redis/validation_complete` - Validation results available
- `custom/redis/fix_successful` - Successful resolution
- `custom/redis/fix_failed` - Maximum retries exceeded

This scenario demonstrates **progressive problem resolution** with sophisticated agent coordination, providing a comprehensive test of retry logic, escalation strategies, and multi-agent workflow orchestration in the execution framework.