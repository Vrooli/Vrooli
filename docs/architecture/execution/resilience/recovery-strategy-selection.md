# 🧠 Recovery Strategy Selection Algorithm

> **TL;DR**: Systematic algorithm for choosing appropriate recovery strategies based on error classification and system context. Use this after classifying an error to determine the best recovery approach.

---

## 🔄 Recovery Strategy Selection Flowchart

This flowchart shows how to choose a recovery strategy once an error has been classified:

```mermaid
flowchart TD
    Start[Error Classified:<br>Type, Severity, Category, Recoverability] --> IsFatal{Severity == FATAL?}
    IsFatal -- Yes --> EmergencyStop[EMERGENCY_STOP<br>🛑 Halt system operations<br>Alert administrators]
    IsFatal -- No --> IsCritical{Severity == CRITICAL?}

    IsCritical -- Yes --> CriticalPath{Data Loss or<br>Security Breach?}
    CriticalPath -- Yes --> HumanEscalation[ESCALATE_TO_HUMAN<br>👤 Immediate human intervention<br>Attempt safe state if possible]
    CriticalPath -- No --> TryRobustRecovery{Attempt Full Recovery with<br>Fallback & Resource Adjustment?}
    TryRobustRecovery -- Yes --> FallbackStrategy[FALLBACK_STRATEGY / FALLBACK_MODEL<br>🔄 Switch to simpler/safer strategy/model<br>💰 Adjust resources, then RETRY_MODIFIED]
    TryRobustRecovery -- No --> DegradeGracefully[GRACEFUL_DEGRADATION<br>📉 Partial success acceptable<br>Log and monitor]
    FallbackStrategy -->|Success| Resolved[Resolved]
    FallbackStrategy -->|Failure| DegradeOrEscalateCritical{Degrade or Escalate?}
    DegradeOrEscalateCritical -- Degrade --> DegradeGracefully
    DegradeOrEscalateCritical -- Escalate --> HumanEscalation

    IsCritical -- No --> IsError{Severity == ERROR?}
    IsError -- Yes --> ErrorPath{Recoverable Error Type?<br>e.g. Transient, Resource}
    ErrorPath -- Yes & Transient --> RetrySame[RETRY_SAME / WAIT_AND_RETRY<br>🔁 Retry with backoff/jitter<br>Monitor transient conditions]
    ErrorPath -- Yes & Resource --> ReduceScope[REDUCE_SCOPE / RETRY_MODIFIED<br>⚙️ Reduce resource demand<br>Attempt modified retry]
    ErrorPath -- No: Logic/Config/Unknown --> FallbackOrEscalateError{Attempt Fallback or Escalate?}
    RetrySame -->|Success| Resolved
    RetrySame -->|Max Attempts| FallbackOrEscalateError
    ReduceScope -->|Success| Resolved
    ReduceScope -->|Failure| FallbackOrEscalateError
    FallbackOrEscalateError -- Fallback --> FallbackStrategyError[FALLBACK_STRATEGY<br>🧩 Attempt alternative logic]
    FallbackOrEscalateError -- Escalate --> EscalateToParentError[ESCALATE_TO_PARENT<br>⬆️ Notify higher tier/agent]
    FallbackStrategyError -->|Success| Resolved
    FallbackStrategyError -->|Failure| EscalateToParentError
    EscalateToParentError --> ResolvedOrHumanEscalate{Parent Handles or Escalates Further}

    IsError -- No --> IsWarning{Severity == WARNING?}
    IsWarning -- Yes --> LogWarning[LOG_WARNING<br>📄 Log for analysis<br>Monitor for patterns]
    IsWarning -- No --> LogInfo[LOG_INFO<br>ℹ️ Informational event]

    EmergencyStop --> End1([*])
    HumanEscalation --> End2([*])
    Resolved --> End3([*])
    DegradeGracefully --> End4([*])
    ResolvedOrHumanEscalate --> End5([*])
    LogWarning --> End6([*])
    LogInfo --> End7([*])

    classDef emergency fill:#ffebee,stroke:#c62828,color:#c62828,stroke-width:2px
    classDef critical fill:#fff3e0,stroke:#f57c00,color:#f57c00,stroke-width:2px
    classDef error fill:#fffde7,stroke:#fbc02d,color:#fbc02d,stroke-width:2px
    classDef warning fill:#e3f2fd,stroke:#1565c0,color:#1565c0,stroke-width:2px
    classDef info fill:#e8f5e9,stroke:#2e7d32,color:#2e7d32,stroke-width:2px
    classDef decision fill:#eceff1,stroke:#37474f,color:#37474f,stroke-width:1px

    class IsFatal,IsCritical,CriticalPath,TryRobustRecovery,DegradeOrEscalateCritical,IsError,ErrorPath,FallbackOrEscalateError,IsWarning,ResolvedOrHumanEscalate decision
    class EmergencyStop emergency
    class HumanEscalation critical
    class FallbackStrategy,ReduceScope,FallbackStrategyError,RetrySame error
    class DegradeGracefully,EscalateToParentError warning
    class LogWarning warning
    class LogInfo info
    class Start,Resolved default
```

---

## 🎯 Key Recovery Strategies

| Strategy | When Used | Purpose | Example |
|----------|-----------|---------|---------|
| **EMERGENCY_STOP** | 🔴 FATAL errors | Halt system operations immediately | Memory overflow, security breach |
| **ESCALATE_TO_HUMAN** | 🟠 CRITICAL errors with data/security risk | Require human intervention | Data corruption, authentication failure |
| **FALLBACK_STRATEGY** | 🟠 CRITICAL or 🟡 ERROR | Switch to simpler/safer approach | Use cached data instead of API |
| **REDUCE_SCOPE** | 🟡 ERROR (resource issues) | Modify operation to use fewer resources | Analyze fewer data points |
| **GRACEFUL_DEGRADATION** | 🟠 CRITICAL (non-data-threatening) | Accept partial success | Serve limited results |
| **RETRY_SAME** | 🟡 ERROR (transient) | Retry exact same operation with backoff | Network timeout, temporary service outage |
| **WAIT_AND_RETRY** | 🟡 ERROR (resource constraint) | Wait for conditions to improve | Rate limit reset, resource availability |
| **RETRY_MODIFIED** | 🟡 ERROR | Retry with adjusted parameters | Different model, reduced scope |
| **ESCALATE_TO_PARENT** | 🟡 ERROR (unresolvable) | Let higher tier handle the error | Complex coordination required |
| **LOG_WARNING/INFO** | 🔵 WARNING / ⚪ INFO | Monitor and log for pattern analysis | Minor configuration issue |

---

## 🤖 Contextual Factors

The algorithm considers these factors when selecting strategies:

### **Error Context**
- **Recoverability**: Automatic, partial, or requires manual intervention
- **Error Category**: Transient, resource, logic, configuration, unknown
- **Attempt Count**: Number of previous recovery attempts
- **Circuit State**: Whether circuit breakers are open

### **System Context**
- **Resource Availability**: Current system load and available resources
- **Data Sensitivity**: Importance of preventing data loss
- **Performance Requirements**: Acceptable degradation levels
- **Security Constraints**: Privilege escalation risks

---

> 💡 **Usage**: Apply this algorithm after using [Error Classification](error-classification-severity.md) to determine error severity. The selected strategy guides the specific recovery implementation.

> 📚 **Implementation**: See [Quick Reference](quick-reference.md) for code patterns implementing these recovery strategies. 