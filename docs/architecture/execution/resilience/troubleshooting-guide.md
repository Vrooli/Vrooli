# 🔧 Resilience Troubleshooting Guide

> **TL;DR**: Step-by-step diagnostics for immediate system issues. Use this when something is broken RIGHT NOW and you need immediate fixes.

> 📚 **Need a different document?** → **[Document Guide](document-guide.md)** explains which resilience document to use for your specific situation.

---

## 🚨 Quick Diagnostic Checklist

### **Step 1: Identify the Error Type**
- [ ] **Tool/API Error** → [Tool Failures](#-tool-and-api-failures)
- [ ] **Resource Limit** → [Resource Issues](#-resource-exhaustion)
- [ ] **Communication Error** → [Communication Problems](#-communication-failures)
- [ ] **State Corruption** → [State Issues](#-state-and-context-issues)
- [ ] **Circuit Breaker Open** → [Circuit Breaker Problems](#-circuit-breaker-issues)

### **Step 2: Check Error Severity**
- [ ] **🔴 CRITICAL** → Immediate escalation required
- [ ] **🟠 HIGH** → Automatic recovery attempted
- [ ] **🟡 MEDIUM** → Retry with fallback
- [ ] **🟢 LOW** → Log and continue

### **Step 3: Apply Recovery Strategy**
- [ ] Check [Error Classification](error-classification-severity.md) for severity assessment
- [ ] Apply [Recovery Strategy Selection](recovery-strategy-selection.md) algorithm
- [ ] Implement solution from relevant section below

---

## 🔧 Tool and API Failures

### **Rate Limit Exceeded**
```typescript
// Symptoms: RateLimitError, 429 HTTP responses
// Location: Tier 3 tool execution

// Quick Fix:
1. Check error.resetAt timestamp
2. If wait time < 50% of timeout → wait and retry
3. Otherwise → try alternative data source
4. Log rate limit event for optimization agents
```

**Immediate Actions:**
- Switch to backup API if available
- Reduce request scope (fewer symbols, shorter timeframe)
- Use cached results if acceptable
- Schedule for later execution

### **Tool Authentication Failure**
```typescript
// Symptoms: 401/403 errors, invalid credentials
// Location: Tool authentication layer

// Quick Fix:
1. Check API key validity and expiration
2. Verify permissions for requested operation
3. Refresh tokens if using OAuth
4. Fall back to alternative authentication method
```

**Recovery Steps:**
1. **Immediate**: Use backup credentials if available
2. **Short-term**: Refresh/renew authentication tokens
3. **Long-term**: Implement credential rotation automation

---

## 💰 Resource Exhaustion

### **Credit Limit Exceeded**
```typescript
// Symptoms: ResourceLimitExceededError
// Location: Resource validation before tool execution

// Decision Tree:
if (emergency_budget_available) {
    // Request temporary limit increase
    await requestEmergencyExpansion(justification);
} else if (cached_results_available) {
    // Use cached results with quality warning
    return useCachedResult(quality_impact: -0.1);
} else {
    // Reduce scope or defer execution
    return reduceScope() || scheduleForLater();
}
```

**Cost Reduction Strategies:**
- **Cached Results**: Use recent similar executions (95% cost savings)
- **Reduced Scope**: Exclude premium features (60% cost savings)
- **Delayed Execution**: Wait for limit reset (no additional cost)
- **Alternative Tools**: Use cheaper data sources (40-80% cost savings)

### **Memory/CPU Exhaustion**
```typescript
// Symptoms: Out of memory errors, timeouts
// Location: Long-running routines, large data processing

// Recovery Actions:
1. Check for memory leaks in routine execution
2. Implement data streaming for large datasets
3. Break large operations into smaller chunks
4. Use external processing for heavy computations
```

---

## 📡 Communication Failures

### **Tool Routing Failures**
```typescript
// Symptoms: CompositeToolRunner dispatch errors
// Location: T1 → T2 communication

// Diagnostic Steps:
1. Check MCP server availability
2. Verify tool registration and schemas
3. Test OpenAI tool fallback
4. Validate request format and parameters
```

**Recovery Options:**
- **Direct RunStateMachine call**: Bypass tool routing temporarily
- **Alternative MCP server**: Use backup MCP implementation
- **Simplified tool call**: Reduce parameter complexity
- **Manual execution**: Human intervention for critical operations

### **Event Bus Failures**
```typescript
// Symptoms: Events not delivered, subscription timeouts
// Location: Cross-tier coordination

// Immediate Actions:
1. Check event bus service health
2. Verify subscription configurations
3. Test direct tier communication as fallback
4. Enable local event buffering
```

---

## 🗃️ State and Context Issues

### **Checkpoint Corruption**
```typescript
// Symptoms: StateCorruptionError during recovery
// Location: RunStateMachine checkpoint restoration

// Recovery Sequence:
1. Try previous checkpoint (most recent valid)
2. Reconstruct from execution history events
3. Restart from beginning with preserved inputs
4. Manual state reconstruction if critical
```

**Data Recovery Priority:**
1. **Critical State**: User inputs, progress markers
2. **Valuable State**: Intermediate results, cached data
3. **Disposable State**: Temporary variables, debug info

### **Context Inheritance Failures**
```typescript
// Symptoms: Missing context data in sub-routines
// Location: T2 context building

// Fix Steps:
1. Check ChatConfigObject integrity
2. Verify context mapping configurations
3. Validate inherited resource limits
4. Rebuild context from swarm state
```

---

## ⚡ Circuit Breaker Issues

### **Circuit Breaker Stuck Open**
```typescript
// Symptoms: All requests failing fast, no recovery attempts
// Location: Service protection layer

// Diagnostic Questions:
- Is the protected service actually healthy?
- Are success thresholds too high?
- Is the reset timeout too long?
- Are health checks failing?
```

**Resolution Steps:**
1. **Manual Override**: Force circuit closed if service is healthy
2. **Threshold Adjustment**: Lower success requirements temporarily
3. **Health Check Fix**: Repair or bypass failing health checks
4. **Alternative Service**: Route to backup service

### **Cascading Circuit Failures**
```typescript
// Symptoms: Multiple circuits opening simultaneously
// Location: Multiple service dependencies

// Emergency Protocol:
1. Identify root cause service
2. Implement service mesh failover
3. Enable degraded mode operation
4. Activate emergency resource allocation
```

---

## 🤖 AI Agent Diagnostics

### **Resilience Agent Not Responding**
```typescript
// Symptoms: No pattern learning, static thresholds
// Location: Event-driven intelligence layer

// Troubleshooting:
1. Check agent subscription to relevant events
2. Verify agent has sufficient permissions
3. Test agent reasoning with sample patterns
4. Restart agent with fresh context
```

### **Strategy Adaptation Failures**
```typescript
// Symptoms: Recovery strategies not improving
// Location: Strategy selection and adaptation

// Investigation:
1. Review success/failure pattern data
2. Check confidence thresholds for adaptations
3. Verify strategy evolution algorithms
4. Test with known-good patterns
```

---

## 🎯 Common Resolution Patterns

### **Graceful Degradation**
1. **Quality Reduction**: Use lower-quality alternatives
2. **Feature Removal**: Disable non-essential functionality
3. **Simplified Processing**: Use faster, simpler algorithms
4. **Cached Results**: Serve stale but valid data

### **Escalation Procedures**
1. **Local Recovery**: Component-level fixes
2. **Tier Coordination**: Cross-tier resource allocation
3. **Human Intervention**: Manual override and assistance
4. **Emergency Protocols**: System-wide protection measures

---

> 💡 **Pro Tip**: For immediate crisis response, follow the diagnostic checklist above. For understanding WHY errors happen and detailed debugging, check the **[Error Scenarios Guide](error-scenarios-guide.md)**. For building resilient code, use the **[Implementation Guide](resilience-implementation-guide.md)**. 

> 📚 **Confused about which doc to use?** → **[Document Guide](document-guide.md)** provides navigation help. 