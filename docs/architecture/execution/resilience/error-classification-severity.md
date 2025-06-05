# 📋 Error Classification and Severity Assessment

> **TL;DR**: Systematic decision algorithm for classifying errors by severity and type. Use this to ensure consistent error handling across all tiers.

---

## 🔄 Severity Decision Tree

This flowchart helps determine the severity of a detected error based on its impact on the system and data.

```mermaid
flowchart TD
    Start[Error Detected] --> SystemCheck{System Still<br/>Functional?}
    
    SystemCheck -->|No| Fatal[FATAL<br/>🔴 System failure<br/>Initiate emergency stop<br/>Alert administrators immediately]
    SystemCheck -->|Yes| ImpactCheck{Multiple Components<br/>or Runs Affected?}
    
    ImpactCheck -->|Yes| CriticalCheck{Data Loss, Corruption,<br/>or Security Risk?}
    ImpactCheck -->|No| SingleOperationCheck{Single Operation<br/>or Step Failed?}
    
    CriticalCheck -->|Yes| Critical[CRITICAL<br/>🟠 Significant impact<br/>Immediate attention required<br/>Attempt automated recovery<br/>Escalate if recovery fails]
    CriticalCheck -->|No| HighError[ERROR<br/>🟡 Multiple non-critical failures<br/>Attempt automated recovery<br/>Monitor closely, escalate on pattern]
    
    SingleOperationCheck -->|Yes| RecoverableCheck{Error Type<br/>Known & Recoverable?}
    SingleOperationCheck -->|No| UnclearImpactError[ERROR<br/>🟡 Uncertain impact<br/>Investigate and classify further<br/>Attempt safe recovery if possible]

    RecoverableCheck -->|Yes| StandardError[ERROR<br/>🟢 Standard recoverable error<br/>Attempt automated recovery<br/>Log for monitoring]
    RecoverableCheck -->|No| Warning[WARNING<br/>🔵 Minor, unexpected issue<br/>Log for analysis<br/>Monitor for recurrence]

    classDef fatal fill:#ffebee,stroke:#c62828,color:#c62828,stroke-width:2px;
    classDef critical fill:#fff3e0,stroke:#f57c00,color:#f57c00,stroke-width:2px;
    classDef error fill:#fffde7,stroke:#fbc02d,color:#fbc02d,stroke-width:2px;
    classDef warning fill:#e3f2fd,stroke:#1565c0,color:#1565c0,stroke-width:2px;

    class Start,SystemCheck,ImpactCheck,CriticalCheck,SingleOperationCheck,RecoverableCheck,UnclearImpactError,HighError,StandardError,Warning default
    class Fatal fatal
    class Critical critical
    class HighError error
    class UnclearImpactError error
    class StandardError error
    class Warning warning
```

---

## 📊 Severity Levels Defined

| Level | Symbol | Description | Response |
|-------|--------|-------------|----------|
| **FATAL** | 🔴 | Complete system failure or imminent catastrophic risk | Immediate system-wide halt and manual intervention |
| **CRITICAL** | 🟠 | Severe error with significant impact (data loss, security breach, major component failure) | Immediate attention, automated recovery, escalation if needed |
| **ERROR** | 🟡 | Functionality impacted but not critical (single operation failure, multiple minor failures) | Automated recovery as primary response |
| **WARNING** | 🔵 | Minor issue that doesn't immediately impact functionality but could lead to problems | Log for monitoring, alert on patterns |
| **INFO** | ⚪ | Informational messages about normal system operation | Standard logging |

---

## 🎯 Classification Examples

### **🔧 Tool/API Errors**
- **Rate Limit**: 🟡 ERROR → Retry with delay or alternative
- **Authentication Failure**: 🟠 CRITICAL → Security implications
- **Service Unavailable**: 🟡 ERROR → Alternative service or cached results

### **💰 Resource Errors**
- **Credit Exhaustion**: 🟠 CRITICAL → Scope reduction or emergency expansion
- **Memory Overflow**: 🔴 FATAL → System protection required
- **Timeout**: 🟡 ERROR → Retry or alternative approach

### **📡 Communication Errors**
- **Network Timeout**: 🟡 ERROR → Alternative communication path
- **MCP Server Down**: 🟠 CRITICAL → Fallback to direct interface
- **Event Bus Failure**: 🟠 CRITICAL → Local buffering and direct calls

### **🗃️ State Errors**
- **Checkpoint Corruption**: 🟠 CRITICAL → Previous checkpoint or reconstruction
- **Context Loss**: 🟡 ERROR → Rebuild from swarm state
- **Concurrency Conflict**: 🔵 WARNING → Retry with backoff

---

> 💡 **Usage**: This classification algorithm drives recovery strategy selection and escalation procedures. The output severity level determines the appropriate response pattern.

> 📚 **Next Steps**: Use [Recovery Strategy Selection](recovery-strategy-selection.md) to choose the appropriate recovery approach based on the classification. 