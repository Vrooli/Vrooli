# Tier 2: Process Intelligence Fixtures

This directory contains fixtures that test the **Process Intelligence layer** (`/services/execution/tier2/`) - the workflow orchestration tier. These fixtures serve as **building blocks** for routine configurations that are reused by emergent-capabilities and integration-scenarios.

## ⚙️ Overview

Tier 2 handles:
- Universal routine orchestration across workflow formats
- Pluggable navigators for different process standards
- State machine management for execution flows
- Strategy evolution from conversational to deterministic
- Context management and branch coordination

## 🧩 How This Directory Fits

### **Component Testing Focus**
This directory tests individual Tier 2 components:
- `RunStateMachine` orchestration logic
- Navigator implementations (BPMN, native, etc.)
- Routine evolution patterns
- Workflow state management

### **Reused by Higher-Level Testing**
These fixtures provide routine configurations for composition:

```typescript
// emergent-capabilities extends tier2 routines with learning
const evolvingRoutine = {
    baseConfig: customerInquiryRoutine.config,  // From this directory
    evolution: {
        stages: [conversational, reasoning, deterministic],  // From this directory
        learning: { patternRecognition: true }               // Adds emergence
    }
};

// integration-scenarios uses tier2 as the orchestration layer
const customerServiceScenario = {
    tier1: supportSwarmFixture,        // From tier1-coordination/
    tier2: inquiryRoutineFixture,      // From this directory
    tier3: responsiveExecutionContext  // From tier3-execution/
};
```

## 📁 Directory Structure

```
tier2-process/
├── routines/                    # Routine configurations by domain and evolution
│   ├── by-domain/              # Organized by business domain
│   │   ├── medical/            # Healthcare workflows
│   │   ├── security/           # Security processes
│   │   ├── financial/          # Trading and risk management
│   │   ├── customer-service/   # Support workflows
│   │   └── api-bootstrap/      # System initialization
│   └── by-evolution-stage/     # Organized by maturity level
│       ├── conversational/     # Human-like exploration
│       ├── reasoning/          # Structured chain-of-thought
│       ├── deterministic/      # Optimized automation
│       └── routing/            # Multi-routine coordination
├── navigators/                 # Workflow format adapters
│   ├── native/                # Vrooli JSON format
│   ├── bpmn/                  # BPMN 2.0 compliance
│   ├── temporal/              # Temporal workflows
│   └── config-driven/         # Zero-code workflows
└── run-states/                # State machine fixtures
    ├── simple-states/         # Linear workflows
    ├── complex-branches/      # Conditional flows
    └── error-recovery/        # Failure handling
```

## 🔄 Routine Evolution Stages

Routines naturally evolve through four distinct stages, each with different characteristics:

### Stage 1: Conversational
Initial human-like exploration and understanding:

```typescript
const conversationalRoutine: RoutineFixture = {
    config: {
        __version: "1.0.0",
        id: testIdGenerator.next("ROUTINE"),
        name: "Customer Inquiry Analysis",
        description: "Understand and categorize customer requests",
        executionStrategy: "conversational",
        
        // Simple single-node for exploration
        nodes: [
            {
                id: "analyze",
                type: "generate",
                data: {
                    template: "Analyze this customer inquiry: {{input}}",
                    model: "gpt-4",
                    creativity: 0.8  // High creativity for exploration
                }
            }
        ]
    },
    
    evolutionStage: {
        strategy: "conversational",
        version: "1.0.0",
        metrics: {
            avgDuration: 15000,    // 15 seconds
            avgCredits: 20,
            successRate: 0.70,
            errorRate: 0.15,
            satisfaction: 0.65
        }
    },
    
    emergence: {
        capabilities: [
            "natural_language_understanding",
            "context_retention",
            "empathy_modeling"
        ],
        evolutionPath: "conversational → reasoning"
    }
};
```

### Stage 2: Reasoning
Structured approaches with chain-of-thought:

```typescript
const reasoningRoutine: RoutineFixture = {
    config: {
        executionStrategy: "reasoning",
        
        nodes: [
            {
                id: "categorize",
                type: "generate",
                data: {
                    template: `First, let me categorize this inquiry:
                    1. What type of issue is this?
                    2. What's the customer's emotional state?
                    3. What resolution approaches are possible?
                    
                    Inquiry: {{input}}`
                }
            },
            {
                id: "reason",
                type: "generate", 
                data: {
                    template: `Based on the categorization, let me reason through the best response:
                    - Consider customer history: {{customerHistory}}
                    - Apply company policies: {{policies}}
                    - Generate solution: {{solution}}`
                }
            }
        ]
    },
    
    evolutionStage: {
        strategy: "reasoning",
        version: "2.0.0",
        previousVersion: "1.0.0",
        metrics: {
            avgDuration: 8000,     // 8 seconds  
            avgCredits: 15,
            successRate: 0.85,
            errorRate: 0.08,
            satisfaction: 0.82
        },
        improvements: [
            "Added structured reasoning steps",
            "Improved context utilization",
            "Reduced response time by 45%"
        ]
    }
};
```

### Stage 3: Deterministic
Optimized automation for proven patterns:

```typescript
const deterministicRoutine: RoutineFixture = {
    config: {
        executionStrategy: "deterministic",
        
        // Optimized flow with predefined decision points
        nodes: [
            {
                id: "classify",
                type: "api",
                data: {
                    endpoint: "/classify-inquiry",
                    method: "POST",
                    timeout: 1000  // Fast classification
                }
            },
            {
                id: "route",
                type: "branch",
                data: {
                    condition: "{{classification.type}}",
                    branches: {
                        "billing": "billing_flow",
                        "technical": "technical_flow", 
                        "account": "account_flow"
                    }
                }
            }
        ]
    },
    
    evolutionStage: {
        strategy: "deterministic",
        version: "3.0.0",
        previousVersion: "2.0.0",
        metrics: {
            avgDuration: 2000,     // 2 seconds
            avgCredits: 5,
            successRate: 0.95,
            errorRate: 0.02,
            satisfaction: 0.92
        },
        improvements: [
            "Template-based responses",
            "90% latency reduction",
            "Predictable execution paths"
        ]
    }
};
```

### Stage 4: Routing
Multi-routine coordination for complex tasks:

```typescript
const routingRoutine: RoutineFixture = {
    config: {
        executionStrategy: "routing",
        
        // Orchestrates multiple specialized routines
        nodes: [
            {
                id: "orchestrate",
                type: "subroutine",
                data: {
                    routines: [
                        { id: "sentiment_analysis", weight: 0.3 },
                        { id: "intent_classification", weight: 0.4 },
                        { id: "response_generation", weight: 0.3 }
                    ],
                    coordination: "parallel",
                    aggregation: "weighted_consensus"
                }
            }
        ]
    },
    
    evolutionStage: {
        strategy: "routing",
        version: "4.0.0",
        metrics: {
            avgDuration: 1500,     // 1.5 seconds
            avgCredits: 8,
            successRate: 0.98,
            errorRate: 0.01,
            satisfaction: 0.96
        }
    }
};
```

## 🏭 Domain-Specific Routines

### Medical Domain
Healthcare workflows with compliance requirements:

```typescript
const medicalRoutine: RoutineFixture = {
    config: {
        name: "Patient Data Analysis",
        description: "Analyze patient data with HIPAA compliance",
        
        // BPMN navigator for regulatory compliance
        navigator: "bpmn",
        complianceLevel: "HIPAA",
        
        nodes: [
            {
                id: "validate_access",
                type: "security",
                data: {
                    checks: ["user_authorization", "data_encryption"],
                    auditLog: true
                }
            },
            {
                id: "analyze_symptoms",
                type: "generate",
                data: {
                    template: "Medical analysis template",
                    safeguards: ["no_diagnosis", "suggest_consultation"]
                }
            }
        ]
    },
    
    domain: "medical",
    
    emergence: {
        capabilities: [
            "privacy_preservation",
            "clinical_decision_support",
            "audit_trail_generation"
        ]
    }
};
```

### Security Domain
Threat detection and response workflows:

```typescript
const securityRoutine: RoutineFixture = {
    config: {
        name: "Threat Response",
        description: "Automated threat detection and response",
        
        // Deterministic for predictable security responses
        executionStrategy: "deterministic",
        priority: "high",
        
        nodes: [
            {
                id: "threat_assessment",
                type: "api",
                data: {
                    endpoint: "/analyze-threat",
                    timeout: 5000,
                    retries: 3
                }
            },
            {
                id: "response_action",
                type: "branch",
                data: {
                    condition: "{{threat.severity}}",
                    branches: {
                        "critical": "immediate_lockdown",
                        "high": "alert_security_team",
                        "medium": "log_and_monitor"
                    }
                }
            }
        ]
    },
    
    domain: "security",
    
    emergence: {
        capabilities: [
            "pattern_recognition",
            "adaptive_response",
            "threat_prediction"
        ]
    }
};
```

## 🧭 Navigators

Navigators adapt different workflow formats to the unified execution engine:

### Native Navigator
```typescript
const nativeNavigator = {
    format: "vrooli-json",
    features: [
        "ai_friendly_syntax",
        "dynamic_branching",
        "template_substitution",
        "strategy_switching"
    ],
    
    advantages: [
        "Optimized for AI reasoning",
        "Minimal syntax overhead",
        "Natural evolution support"
    ]
};
```

### BPMN Navigator
```typescript
const bpmnNavigator = {
    format: "bpmn-2.0",
    compliance: ["ISO-19510", "OMG-BPMN"],
    
    features: [
        "visual_workflow_design",
        "standards_compliance",
        "enterprise_integration",
        "audit_trails"
    ],
    
    useCase: "Regulatory compliance, enterprise workflows"
};
```

### Config-Driven Navigator
```typescript
const configNavigator = {
    format: "zero-code",
    configuration: {
        triggers: "event-based",
        flow: "declarative",
        actions: "template-driven"
    },
    
    benefits: [
        "No coding required",
        "Business user friendly",
        "Rapid prototyping"
    ]
};
```

## 🔄 State Management

### Run State Fixtures
Test state machine behavior:

```typescript
const runStateFixture: RunStateFixture = {
    initialState: "initialized",
    
    transitions: [
        {
            from: "initialized",
            to: "running", 
            trigger: "start_execution",
            conditions: ["resources_available"],
            actions: ["allocate_context"]
        },
        {
            from: "running",
            to: "completed",
            trigger: "execution_finished",
            actions: ["cleanup_resources", "emit_metrics"]
        },
        {
            from: "running",
            to: "error",
            trigger: "execution_failed",
            actions: ["log_error", "notify_tier1"]
        }
    ],
    
    expectedFinalState: "completed",
    expectedEvents: [
        "tier2.run.started",
        "tier2.step.completed",
        "tier2.run.finished"
    ]
};
```

## 🧪 Testing Tier 2 Fixtures

### Evolution Path Testing
```typescript
describe("Routine Evolution", () => {
    it("should show improvement across evolution stages", async () => {
        const stages = [
            conversationalRoutine,
            reasoningRoutine, 
            deterministicRoutine,
            routingRoutine
        ];
        
        validateEvolutionPath(stages, "customer-inquiry");
        
        // Verify performance improvements
        for (let i = 1; i < stages.length; i++) {
            const prev = stages[i-1].evolutionStage.metrics;
            const curr = stages[i].evolutionStage.metrics;
            
            expect(curr.avgDuration).toBeLessThan(prev.avgDuration);
            expect(curr.successRate).toBeGreaterThanOrEqual(prev.successRate);
        }
    });
});
```

### Navigator Testing
```typescript
describe("Navigator Compatibility", () => {
    it("should execute BPMN workflow correctly", async () => {
        const bpmnRoutine = createBPMNRoutine();
        const result = await executeRoutine(bpmnRoutine);
        
        expect(result.compliance).toBe("BPMN-2.0");
        expect(result.auditTrail).toBeDefined();
    });
});
```

## 🎯 Performance Optimization

### Execution Strategies
```typescript
optimizationStrategies: {
    conversational: {
        optimize_for: "exploration",
        trade_offs: "speed_for_flexibility"
    },
    
    reasoning: {
        optimize_for: "accuracy", 
        trade_offs: "latency_for_quality"
    },
    
    deterministic: {
        optimize_for: "speed",
        trade_offs: "flexibility_for_performance"
    },
    
    routing: {
        optimize_for: "specialization",
        trade_offs: "complexity_for_expertise"
    }
}
```

## 💡 Best Practices

### DO:
- ✅ Define clear evolution paths
- ✅ Include performance metrics
- ✅ Test strategy transitions
- ✅ Document optimization trade-offs
- ✅ Use appropriate navigators

### DON'T:
- ❌ Skip evolution stages
- ❌ Hard-code strategy selection
- ❌ Ignore performance regressions
- ❌ Mix strategy concerns
- ❌ Assume linear improvement

## 🔗 Integration with Other Tiers

### From Tier 1 (Coordination)
```typescript
tier1ToTier2: {
    commands: [
        "execute_routine",
        "modify_workflow",
        "switch_strategy"
    ],
    context: [
        "swarm_goals",
        "resource_allocation",
        "priority_levels"
    ]
}
```

### To Tier 3 (Execution)
```typescript
tier2ToTier3: {
    execution_requests: [
        "tool_invocation",
        "context_update",
        "safety_check"
    ],
    configuration: [
        "execution_strategy",
        "resource_limits",
        "approval_requirements"
    ]
}
```

## 📊 Metrics and Benchmarks

### Process Intelligence KPIs
```typescript
benchmarks: {
    orchestrationLatency: "< 50ms",
    stateTransitionTime: "< 10ms", 
    strategyAdaptation: "< 1 second",
    routineEvolution: "< 7 days",
    errorRecovery: "< 100ms"
}
```

## 🚀 Advanced Patterns

### Adaptive Workflow
```typescript
adaptiveWorkflow: {
    trigger: "performance_degradation",
    adaptation: [
        "add_parallel_paths",
        "adjust_timeouts", 
        "switch_strategy",
        "request_more_resources"
    ],
    learning: "continuous"
}
```

### Multi-Version Execution
```typescript
multiVersionExecution: {
    versions: ["v1_stable", "v2_testing", "v3_experimental"],
    trafficSplit: [70, 20, 10],  // Percentage allocation
    promotion: "metric_based"
}
```

## 📚 References

- [Routine Configuration Guide](/docs/architecture/execution/tier2-process/routines.md)
- [Navigator Implementation](/docs/architecture/execution/tier2-process/navigators.md)
- [State Machine Patterns](/docs/architecture/execution/tier2-process/state-machines.md)