# ⚡ Resilience Quick Reference: Fast Lookup for Developers

> **TL;DR**: Fast lookup patterns for developers who understand resilience basics. Use this for quick error → strategy → code pattern decisions during development.

> **🚨 System broken right now?** → **[Troubleshooting Guide](troubleshooting-guide.md)** for step-by-step diagnostics

---

## 🎯 Error → Strategy → Code Decision Matrix

| Error Type | Severity | Quick Strategy | Code Pattern | Alt Strategy |
|------------|----------|----------------|--------------|--------------|
| **🔧 Tool/API Rate Limit** | 🟡 ERROR | Wait if <50% timeout, else alternative | [Pattern →](#-tool-failure-patterns) | Tool substitution |
| **💰 Resource Exhaustion** | 🟠 CRITICAL | Emergency expansion → scope reduction | [Pattern →](#-resource-limit-patterns) | Cached results |
| **📡 Communication Failure** | 🟠 CRITICAL | Alternative channel → direct interface | [Pattern →](#-communication-failure-patterns) | Manual execution |
| **🗃️ State Corruption** | 🟠 CRITICAL | Previous checkpoint → event reconstruction | [Pattern →](#-state-recovery-patterns) | Restart from inputs |
| **⚡ Circuit Breaker Open** | 🟡 ERROR | Wait for reset → manual override | [Pattern →](#-circuit-breaker-patterns) | Alternative service |

---

## 📝 Copy-Paste Code Patterns

### **🔧 Tool Failure Patterns**

```typescript
// Pattern: Retry with Alternative
try {
  return await primaryTool.execute(params);
} catch (error) {
  if (error instanceof RateLimitError) {
    // Smart wait or immediate alternative
    const waitTime = error.resetAt - Date.now();
    const maxWait = timeout * 0.5;
    
    if (waitTime <= maxWait) {
      await delay(waitTime + 1000);
      return await primaryTool.execute(params);
    }
  }
  
  // Try alternatives in order of quality/cost
  for (const alt of alternatives) {
    try {
      const result = await alt.tool.execute(params);
      // Emit quality degradation warning
      await eventBus.publish('execution/quality_degraded', {
        qualityReduction: 1.0 - alt.quality
      });
      return result;
    } catch (altError) {
      continue; // Try next alternative
    }
  }
  
  throw new AllToolsFailedError(originalError, attemptedAlternatives);
}
```

### **💰 Resource Limit Patterns**

```typescript
// Pattern: Smart Resource Management
async function checkAndReserveResources(request: StepRequest, cost: number) {
  const usage = await getCurrentUsage(request.swarmId);
  const limits = await getLimits(request.swarmId);
  
  if (usage.credits + cost > limits.maxCredits) {
    // Try emergency expansion
    const expansion = await requestEmergencyExpansion({
      swarmId: request.swarmId,
      requestedAmount: cost,
      justification: generateJustification(request),
      timeoutMs: 30000
    });
    
    if (expansion.approved) {
      await updateLimits(request.swarmId, expansion.newLimits);
      return await reserveResources(request.stepId, cost);
    }
    
    // Expansion denied - suggest alternatives
    const alternatives = await suggestCostReduction(request);
    throw new ResourceLimitExceededError({
      limit: limits.maxCredits,
      requested: cost,
      alternatives: alternatives
    });
  }
  
  return await reserveResources(request.stepId, cost);
}
```

### **📡 Communication Failure Patterns**

```typescript
// Pattern: Communication Fallback
class TierCommunication {
  async executeWithFallback<T>(
    primary: () => Promise<T>,
    fallback: () => Promise<T>,
    description: string
  ): Promise<T> {
    try {
      return await primary();
    } catch (error) {
      if (this.isCommunicationError(error)) {
        await this.eventBus.publish('communication/fallback_triggered', {
          primaryMethod: description,
          error: error.message
        });
        
        return await fallback();
      }
      throw error; // Not a communication error - re-throw
    }
  }
  
  // Usage example
  async routineExecution(params: RoutineParams) {
    return await this.executeWithFallback(
      // Primary: Tool routing through MCP
      () => this.mcpTools.call('execute_routine', params),
      // Fallback: Direct interface call
      () => this.runStateMachine.executeRoutine(params),
      'T1→T2 routine execution'
    );
  }
}
```

### **🗃️ State Recovery Patterns**

```typescript
// Pattern: Cascading Recovery Strategies
async function recoverFromStateCorruption(runId: string, corruptedCheckpoint: string) {
  // Strategy 1: Previous checkpoint
  const checkpoints = await getCheckpoints(runId);
  for (const checkpoint of checkpoints.reverse()) {
    if (checkpoint.id === corruptedCheckpoint) continue;
    
    try {
      const validation = await validateCheckpoint(checkpoint);
      if (validation.isValid) {
        await eventBus.publish('routine/state/recovered', {
          method: 'previous_checkpoint',
          dataLoss: calculateDataLoss(checkpoint, corruptedCheckpoint)
        });
        return await restoreFromCheckpoint(checkpoint);
      }
    } catch (error) {
      continue; // Try next checkpoint
    }
  }
  
  // Strategy 2: Reconstruct from events
  try {
    const events = await getExecutionEvents(runId);
    const reconstructed = await reconstructFromEvents(events);
    
    if (isViableState(reconstructed)) {
      await eventBus.publish('routine/state/reconstructed', {
        method: 'execution_history',
        confidence: calculateConfidence(reconstructed)
      });
      return reconstructed;
    }
  } catch (error) {
    // Reconstruction failed
  }
  
  // Strategy 3: Escalate for restart decision
  throw new UnrecoverableStateError({
    runId,
    attemptedRecoveries: ['previous_checkpoints', 'event_reconstruction'],
    recommendedAction: 'restart_from_beginning'
  });
}
```

### **⚡ Circuit Breaker Patterns**

```typescript
// Pattern: Intelligent Circuit Management
class ServiceCircuitBreaker {
  async executeWithCircuit<T>(
    serviceName: string,
    operation: () => Promise<T>
  ): Promise<T> {
    const circuit = await this.getCircuit(serviceName);
    
    if (circuit.isOpen) {
      // Check if we should attempt half-open
      if (circuit.shouldAttemptReset()) {
        circuit.halfOpen();
        try {
          const result = await operation();
          circuit.recordSuccess();
          return result;
        } catch (error) {
          circuit.recordFailure();
          throw new CircuitOpenError(serviceName, error);
        }
      } else {
        throw new CircuitOpenError(serviceName);
      }
    }
    
    try {
      const result = await operation();
      circuit.recordSuccess();
      return result;
    } catch (error) {
      circuit.recordFailure();
      
      if (circuit.shouldOpen()) {
        await this.eventBus.publish('circuit/opened', {
          serviceName,
          reason: 'failure_threshold_exceeded',
          failureCount: circuit.failureCount
        });
      }
      
      throw error;
    }
  }
}
```

---

## 🤖 AI Agent Integration

### **Quick Agent Deployment**
```typescript
// Deploy performance resilience agent
const agent = {
  name: "Timeout Recovery Specialist",
  subscriptions: ["step/failed", "tool/timeout"],
  onEvent: async (event) => {
    if (event.payload.errorType === "timeout") {
      const pattern = await analyzeTimeoutPattern(event.payload);
      if (pattern.shouldAdjustTimeout) {
        await proposeTimeoutOptimization(pattern);
      }
    }
  }
};

await deployAgent(agent, {
  permissions: ["modify_timeouts", "suggest_alternatives"]
});
```

> 📖 **Complete agent examples**: See [Resilience Agents](../emergent-capabilities/routine-examples/) for detailed patterns

---

> 💡 **Pro Tip**: This reference is for developers who understand resilience basics. For step-by-step crisis response, use **[Troubleshooting Guide](troubleshooting-guide.md)**. For complete implementation guidance, use **[Implementation Guide](resilience-implementation-guide.md)**.

> 📚 **Need help choosing the right document?** → **[Document Guide](document-guide.md)** explains when to use each resilience guide. 