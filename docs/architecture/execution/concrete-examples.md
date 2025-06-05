# 🛠️ Concrete Implementation Examples

> **TL;DR**: This document provides practical, concrete examples showing how Vrooli's three-tier execution architecture works in practice. Use these examples as templates for understanding the flow from high-level swarm coordination down to individual tool execution.

---

## 📚 Table of Contents

- [🚀 Complete Execution Flow Example](#-complete-execution-flow-example)
- [🧠 Strategy Evolution in Action](#-strategy-evolution-in-action)
- [🔧 Tool Approval & Scheduling](#-tool-approval--scheduling)
- [⚠️ Error Handling Across Tiers](#️-error-handling-across-tiers)
- [📊 Resource Management Example](#-resource-management-example)
- [🔄 Event-Driven Coordination](#-event-driven-coordination)
- [🏗️ Minimal Infrastructure Setup](#️-minimal-infrastructure-setup)

---

## 🚀 Complete Execution Flow Example

Let's trace a complete execution from **Tier 1 coordination** through **Tier 2 process intelligence** to **Tier 3 execution**:

### **Scenario**: Customer Feedback Collection & Analysis

A user requests: *"Analyze customer feedback from the past month and provide insights on common issues"*

#### **Tier 1: Coordination Intelligence** 🧠

```typescript
// Tier 1 receives the user request and breaks it down
const coordinationPlan: CoordinationPlan = {
    requestId: "req_feedback_analysis_001",
    userId: "user_12345",
    task: "Analyze customer feedback from the past month",
    
    // Tier 1 identifies this requires multiple coordinated routines
    coordinationStrategy: {
        approach: "sequential_with_aggregation",
        estimatedComplexity: "medium",
        requiredCapabilities: ["data_collection", "analysis", "reporting"]
    },
    
    // Tier 1 spawns multiple routines to handle different aspects
    subroutines: [
        {
            routineId: "feedback_collection_v2",
            purpose: "Collect feedback data from multiple sources",
            priority: "high",
            dependencies: []
        },
        {
            routineId: "sentiment_analysis_v1", 
            purpose: "Analyze sentiment and extract key themes",
            priority: "high",
            dependencies: ["feedback_collection_v2"]
        },
        {
            routineId: "insight_generation_v3",
            purpose: "Generate actionable insights and recommendations", 
            priority: "medium",
            dependencies: ["sentiment_analysis_v1"]
        }
    ]
};
```

#### **Tier 2: Process Intelligence** ⚙️

**Why Native Vrooli Format (CUSTOM)?**
For this workflow, we chose `NavigatorType.CUSTOM` because:
- ✅ **Sequential flow**: Simple data collection → analysis → reporting pattern
- ✅ **AI-friendly**: Easy for agents to modify based on available data sources
- ✅ **Lightweight**: No complex branching or event handling needed
- ✅ **Rapid iteration**: Can be modified as feedback sources change

> **When to use BPMN instead**: If this workflow required complex approval processes, regulatory compliance, parallel data collection with synchronization barriers, or sophisticated error handling with compensation flows, we would choose `NavigatorType.BPMN`.

```typescript
// Tier 2 receives the first routine and processes it using Native Navigator
const feedbackRoutine: RunRequest = {
    routineVersionId: "feedback_collection_v2",
    name: "Customer Feedback Collection",
    description: "Collect and consolidate customer feedback from multiple sources",
    
    // Using CUSTOM navigator for clean, sequential execution
    config: {
        navigatorType: NavigatorType.CUSTOM,
        
        // Native Vrooli workflow format - clean and AI-friendly
        steps: [
            {
                stepId: "step_1_query_support",
                stepType: StepType.PROCESS,
                name: "Query Support System",
                description: "Collect support tickets from the past month",
                
                // Simple, clear data flow mappings
                inputMappings: { 
                    "timeframe": "inputs.timeframe" 
                },
                outputMappings: { 
                    "support_data": "step_1_output" 
                },
                
                // Step-specific configuration
                configuration: {
                    tool: "database_query",
                    query_template: "SELECT * FROM support_tickets WHERE created_date >= {timeframe_start}",
                    max_results: 1000
                },
                required: true,
                timeout: 120000
            },
            {
                stepId: "step_2_query_reviews",
                stepType: StepType.PROCESS,
                name: "Query Review Platform",
                description: "Collect reviews from external platforms",
                inputMappings: { 
                    "timeframe": "inputs.timeframe" 
                },
                outputMappings: { 
                    "review_data": "step_2_output" 
                },
                configuration: {
                    tool: "api_client",
                    endpoints: ["trustpilot", "google_reviews", "app_store"],
                    auth_required: true
                },
                required: true,
                timeout: 180000
            },
            {
                stepId: "step_3_consolidate",
                stepType: StepType.PROCESS,
                name: "Consolidate Data",
                description: "Merge and normalize feedback from all sources",
                inputMappings: { 
                    "support": "step_1_output.support_data",
                    "reviews": "step_2_output.review_data",
                    "timeframe": "inputs.timeframe"
                },
                outputMappings: { 
                    "consolidated_feedback": "final_output" 
                },
                configuration: {
                    tool: "data_processing",
                    format: "structured_json",
                    deduplication: true,
                    normalization_rules: "feedback_v2"
                },
                required: true,
                timeout: 60000
            }
        ]
    },
    
    // Resource requirements for the entire routine
    resourceRequirements: {
        minCredits: 100,
        estimatedCredits: 500,
        maxCredits: 1000,
        estimatedDurationMs: 300000,  // 5 minutes
        maxDurationMs: 600000,        // 10 minutes max
        memoryMB: 256,
        concurrencyLevel: 2,
        toolsRequired: ["database_query", "api_client", "data_processing"]
    },
    
    inputs: [
        {
            name: "timeframe", 
            value: "past_month",
            type: "string"
        }
    ]
};
```

#### **Tier 3: Execution Engine** 🔧

```typescript
// Tier 3 executes each step, managing resources and tool approvals
const executionFlow = {
    
    // Step 1: Query Support System
    step_1_execution: {
        toolRequest: {
            tool: "database_query", 
            parameters: {
                query: "SELECT * FROM support_tickets WHERE created_date >= '2024-01-01'",
                max_results: 1000
            }
        },
        
        // Tier 3 handles resource allocation and tool approval
        resourceAllocation: {
            creditsRequired: 50,
            memoryMB: 64,
            timeoutMs: 120000
        },
        
        toolApproval: {
            status: "auto_approved",  // Database queries are pre-approved
            reason: "standard_data_collection"
        },
        
        executionResult: {
            status: "completed",
            output: {
                support_data: [
                    {
                        ticket_id: "T12345",
                        subject: "Login issues",
                        description: "Cannot login to account",
                        sentiment: "negative",
                        created_date: "2024-01-15"
                    },
                    // ... more tickets
                ]
            },
            resourcesUsed: {
                credits: 45,
                durationMs: 85000,
                memoryMB: 52
            }
        }
    },
    
    // Step 2: Query Review Platform (parallel execution possible)
    step_2_execution: {
        toolRequest: {
            tool: "api_client",
            parameters: {
                endpoints: ["trustpilot", "google_reviews"],
                timeframe: "past_month",
                auth_tokens: ["encrypted_token_1", "encrypted_token_2"]
            }
        },
        
        resourceAllocation: {
            creditsRequired: 100,
            memoryMB: 128,
            timeoutMs: 180000
        },
        
        toolApproval: {
            status: "approved_with_monitoring",  // External APIs require monitoring
            reason: "external_data_source",
            monitoringLevel: "standard"
        },
        
        executionResult: {
            status: "completed",
            output: {
                review_data: [
                    {
                        platform: "trustpilot",
                        rating: 2,
                        review: "App crashes frequently",
                        date: "2024-01-20"
                    },
                    // ... more reviews
                ]
            },
            resourcesUsed: {
                credits: 95,
                durationMs: 165000,
                memoryMB: 110
            }
        }
    },
    
    // Step 3: Consolidate Data
    step_3_execution: {
        toolRequest: {
            tool: "data_processing",
            parameters: {
                inputs: {
                    support: "step_1_output.support_data",
                    reviews: "step_2_output.review_data"
                },
                processing_rules: "feedback_consolidation_v2",
                output_format: "structured_json"
            }
        },
        
        executionResult: {
            status: "completed", 
            output: {
                consolidated_feedback: {
                    total_items: 450,
                    sources: ["support_tickets", "trustpilot", "google_reviews"],
                    common_themes: ["login_issues", "app_crashes", "slow_performance"],
                    sentiment_distribution: {
                        positive: 0.25,
                        neutral: 0.35, 
                        negative: 0.40
                    },
                    processed_at: "2024-01-25T10:30:00Z"
                }
            }
        }
    }
};
```

### **Step 4: Results Flow Back Through Tiers**

```typescript
// Tier 3 → Tier 2: Step result returned
// Tier 2 processes remaining steps, then consolidates results

// Tier 2 → Tier 1: Routine completion via MCP tool response
const routineResult: RoutineExecutionResult = {
    runId: routineRequest.requestId,
    requestId: routineRequest.requestId,
    status: RunStatus.COMPLETED,
    
    outputs: {
        consolidated_feedback: {
            support_tickets: 847,
            app_reviews: 293, 
            survey_responses: 156,
            total_feedback_items: 1296,
            data_quality_score: 0.91,
            collection_timestamp: new Date().toISOString()
        }
    },
    
    exports: [
        {
            key: "customer_feedback_data",
            destination: ExportDestination.BLACKBOARD,
            sensitivity: DataSensitivity.CONFIDENTIAL,
            transformation: "structured_json"
        }
    ],
    
    errors: [],
    
    resourceUsage: {
        creditsUsed: 387,
        timeElapsedMs: 340000,
        memoryUsedMB: 512,
        toolCallsMade: 8,
        concurrentBranchesActive: 2,
        peakMemoryMB: 710,
        networkBytesTransferred: 1250000,
        diskBytesUsed: 5600000
    },
    
    executionMetrics: {
        startTime: new Date(Date.now() - 340000),
        endTime: new Date(),
        duration: 340000,
        stepCount: 4,
        successRate: 1.0,
        averageStepDuration: 85000
    },
    
    strategyEvolution: {
        originalStrategy: StrategyType.REASONING,
        finalStrategy: StrategyType.REASONING, // No evolution this time
        evolutionSteps: [],
        qualityImprovement: 0.0
    },
    
    finalState: {
        runId: routineRequest.requestId,
        currentStep: "completed",
        completedSteps: ["step_1_query_support", "step_2_query_reviews", "step_3_survey_data", "step_4_consolidate"],
        context: { /* final context state */ },
        timestamp: new Date()
    },
    
    checkpoints: [
        {
            checkpointId: "cp_after_data_collection",
            runId: routineRequest.requestId,
            stepId: "step_4_consolidate",
            state: { /* checkpoint state */ },
            timestamp: new Date(),
            recoverable: true
        }
    ]
};

// Tier 1: Leader agent receives completion and updates swarm state
await mcp.updateSwarmSharedState({
    subtasks: [
        // Update T1 status to completed
        {
            id: "T1_data_collection",
            description: "Collect customer feedback from support tickets, reviews, and surveys from the past month",
            status: "done", // ✅ Status updated
            assignee_bot_id: "data_specialist_bot_789",
            priority: "high",
            created_at: previousTask.created_at
        }
        // ... other subtasks remain
    ],
    
    // Add results to blackboard for other agents
    blackboard: [
        {
            id: "customer_feedback_data",
            value: routineResult.outputs.consolidated_feedback,
            created_at: new Date().toISOString()
        }
    ],
    
    // Record this execution
    records: [
        {
            id: "record_T1_" + nanoid(),
            routine_id: "data_collection_routine_v2",
            routine_name: "Customer Feedback Data Collection",
            params: routineRequest.inputs,
            output_resource_ids: ["customer_feedback_data"],
            caller_bot_id: "data_specialist_bot_789",
            created_at: new Date().toISOString()
        }
    ]
});
```

---

## 🧠 Strategy Evolution in Action

Let's see how a routine evolves from conversational to deterministic over multiple executions:

### **First Execution: Conversational Strategy**

```typescript
// Week 1: New routine, starts conversational
const initialStrategy: ExecutionStrategy = {
    type: StrategyType.CONVERSATIONAL,
    configuration: {
        model: "gpt-4o",
        temperature: 0.7,
        systemPrompt: "You are a data analyst. Help collect and analyze customer feedback."
    },
    costMultiplier: 1.0,
    qualityTarget: 0.7
};

// Results: Good quality but high cost
const week1Results = {
    qualityScore: 0.81,
    creditsUsed: 850,
    timeElapsed: 420000, // 7 minutes
    variability: 0.23 // High variability in outputs
};
```

### **Second Execution: Pattern Recognition**

```typescript
// Week 2: Optimization agent detects patterns
const optimizationInsight = {
    pattern: "structured_data_collection",
    confidence: 0.75,
    recommendation: "Consider structured reasoning approach for data collection tasks",
    evidenceCount: 2,
    costSavingsPotential: 0.4
};

// Strategy evolves to reasoning
const evolvedStrategy: ExecutionStrategy = {
    type: StrategyType.REASONING,
    configuration: {
        model: "gpt-4o",
        temperature: 0.3, // Lower temperature for structured tasks
        reasoningFramework: "data_collection_systematic",
        stepValidation: true
    },
    fallbackStrategy: initialStrategy,
    costMultiplier: 0.7,
    qualityTarget: 0.85
};
```

### **Fifth Execution: Deterministic Evolution**

```typescript
// Week 5: High confidence in pattern, evolve to deterministic
const deterministicStrategy: ExecutionStrategy = {
    type: StrategyType.DETERMINISTIC,
    configuration: {
        cacheEnabled: true,
        queryTemplates: {
            support_tickets: "SELECT * FROM support_tickets WHERE created_date >= ?",
            app_reviews: "SELECT * FROM reviews WHERE date >= ? AND platform IN (?)",
            surveys: "SELECT * FROM survey_responses WHERE submitted_date >= ?"
        },
        outputSchema: {
            type: "object",
            properties: {
                /* structured schema */
            }
        },
        validationRules: ["max_results", "data_quality", "timeout_limits"]
    },
    fallbackStrategy: evolvedStrategy,
    costMultiplier: 0.15, // 85% cost reduction
    qualityTarget: 0.95
};

// Results: Consistent high quality, very low cost
const week5Results = {
    qualityScore: 0.96,
    creditsUsed: 127, // 85% reduction
    timeElapsed: 95000, // 77% faster
    variability: 0.03 // Very consistent
};
```

---

## 🔧 Tool Approval & Scheduling

Here's how the tool approval system works for sensitive operations:

### **Tool Requiring Approval**

```typescript
// Agent attempts to execute a sensitive tool
const sensitiveToolCall = {
    toolName: "database_modify",
    arguments: {
        table: "customer_data",
        operation: "delete",
        condition: "inactive_since < '2022-01-01'"
    }
};

// System creates pending tool call entry
const pendingEntry: PendingToolCallEntry = {
    pendingId: "pending_" + nanoid(),
    toolCallId: "tool_call_123",
    toolName: "database_modify",
    toolArguments: JSON.stringify(sensitiveToolCall.arguments),
    callerBotId: "data_cleanup_bot_456",
    conversationId: "conv_789",
    requestedAt: Date.now(),
    status: PendingToolCallStatus.PENDING_APPROVAL,
    statusReason: "Requires approval for data deletion operations",
    scheduledExecutionTime: undefined,
    approvalTimeoutAt: Date.now() + 300000, // 5 minutes
    userIdToApprove: "admin_user_123",
    executionAttempts: 0
};

// User receives approval request through UI
const approvalRequest = {
    pendingId: pendingEntry.pendingId,
    toolName: "database_modify",
    description: "Bot wants to delete inactive customer records older than 2 years",
    risk: "HIGH",
    affectedData: "customer_data table",
    requestingBot: "data_cleanup_bot_456",
    estimatedImpact: "~50,000 records"
};
```

### **User Approval Flow**

```typescript
// User approves the tool call
const approvalResponse: RespondToToolApprovalInput = {
    conversationId: "conv_789",
    pendingId: pendingEntry.pendingId,
    approved: true,
    reason: "Cleanup approved after review of retention policy"
};

// System updates pending entry
const approvedEntry: PendingToolCallEntry = {
    ...pendingEntry,
    status: PendingToolCallStatus.APPROVED_READY_FOR_EXECUTION,
    statusReason: "Approved by admin_user_123",
    approvedOrRejectedByUserId: "admin_user_123",
    decisionTime: Date.now(),
    scheduledExecutionTime: Date.now() + 5000 // Execute in 5 seconds
};
```

### **Scheduled Execution**

```typescript
// Tool executes after approval
const executionResult = {
    pendingId: approvedEntry.pendingId,
    status: PendingToolCallStatus.COMPLETED_SUCCESS,
    result: JSON.stringify({
        recordsDeleted: 47832,
        tablesAffected: ["customer_data", "customer_preferences"],
        executionTime: 8.3,
        rollbackAvailable: true,
        rollbackId: "rollback_" + nanoid()
    }),
    cost: "25", // Low cost for database operation
    executionAttempts: 1,
    lastAttemptTime: Date.now()
};
```

---

## ⚠️ Error Handling Across Tiers

Here's how errors propagate and get handled across the three tiers:

### **Tier 3 Error (Tool Execution Failure)**

```typescript
// Tier 3: API call fails
const tier3Error: ExecutionError = {
    id: "error_" + nanoid(),
    type: ExecutionErrorType.TOOL_EXECUTION_FAILED,
    message: "External API rate limit exceeded",
    details: {
        apiEndpoint: "https://api.customer-feedback.com/reviews",
        statusCode: 429,
        retryAfter: 60,
        requestId: "req_123"
    },
    timestamp: new Date(),
    context: {
        tier: Tier.TIER_3,
        componentId: "unified_executor",
        operationType: "api_call",
        requestId: "req_123",
        correlationId: "corr_456",
        sessionId: "session_789",
        agentId: "data_specialist_bot_789",
        additionalContext: {
            stepId: "step_2_query_reviews",
            routineId: "data_collection_routine_v2"
        }
    }
};

// Tier 3 applies recovery strategy
const recoveryAction = {
    strategy: "exponential_backoff_retry",
    waitTime: 60000, // Wait 60 seconds as suggested by API
    maxRetries: 3,
    fallbackAction: "use_cached_data"
};
```

### **Tier 2 Error Handling**

```typescript
// Tier 2: Receives error from Tier 3, decides on response
const tier2Response = {
    action: "partial_execution_with_degraded_data",
    reasoning: "Reviews API unavailable, but support tickets and surveys succeeded",
    modifications: {
        skipStep: "step_2_query_reviews",
        useAlternative: "cached_reviews_from_yesterday",
        adjustQualityTarget: 0.8 // Lower from 0.9 due to missing data
    }
};

// Continue execution with degraded data
const modifiedResult: RoutineExecutionResult = {
    // ... standard fields
    status: RunStatus.COMPLETED,
    outputs: {
        consolidated_feedback: {
            support_tickets: 847,
            app_reviews: 0, // Failed to collect
            survey_responses: 156,
            cached_reviews_used: 184, // Used cached data instead
            data_quality_score: 0.78, // Reduced due to missing fresh reviews
            warnings: ["Fresh app reviews unavailable, used cached data from previous day"]
        }
    },
    errors: [tier3Error], // Non-fatal error included in response
    // ... other fields
};
```

### **Tier 1 Error Escalation**

```typescript
// Tier 1: Leader agent receives partial success, adapts plan
await mcp.updateSwarmSharedState({
    subtasks: [
        {
            id: "T1_data_collection",
            status: "done", // Completed with degraded data
            // ... other fields
        },
        {
            id: "T2_sentiment_analysis",
            status: "in_progress",
            description: "Perform sentiment analysis (adjusted for missing review data)",
            // ... other fields
        },
        // Add new subtask to address the data gap
        {
            id: "T1b_review_recovery",
            description: "Retry collection of app reviews when API becomes available",
            status: "todo",
            priority: "low",
            depends_on: [],
            created_at: new Date().toISOString()
        }
    ],
    
    // Log the issue on blackboard for team awareness
    blackboard: [
        {
            id: "api_availability_issue",
            value: {
                issue: "App reviews API temporarily unavailable",
                impact: "Using cached data from previous day",
                recoveryPlan: "Retry collection in subtask T1b_review_recovery",
                qualityImpact: "Reduced from 0.9 to 0.78"
            },
            created_at: new Date().toISOString()
        }
    ]
});
```

---

## 📊 Resource Management Example

Here's how resource limits are enforced and optimized:

### **Resource Limit Enforcement**

```typescript
// Agent starts with resource allocation
const agentLimits: ResourceLimits = {
    maxCredits: 5000,
    maxDurationMs: 1800000, // 30 minutes
    maxConcurrentBranches: 3,
    maxMemoryMB: 2048,
    maxToolCalls: 50,
    priorityBonus: 1000, // High priority agent
    emergencyReserve: 500
};

// During execution, limits are checked
const currentUsage: ResourceUsage = {
    creditsUsed: 3800, // Approaching limit
    timeElapsedMs: 1200000, // 20 minutes used
    memoryUsedMB: 1500,
    toolCallsMade: 35,
    concurrentBranchesActive: 2,
    peakMemoryMB: 1800,
    networkBytesTransferred: 50000000,
    diskBytesUsed: 150000000
};

// System calculates remaining resources
const remainingResources = {
    creditsRemaining: agentLimits.maxCredits - currentUsage.creditsUsed, // 1200
    timeRemaining: agentLimits.maxDurationMs - currentUsage.timeElapsedMs, // 600000 (10 min)
    percentComplete: currentUsage.timeElapsedMs / agentLimits.maxDurationMs, // 66%
    riskLevel: "medium" // Approaching credit limit
};
```

### **Dynamic Resource Optimization**

```typescript
// Optimization agent detects inefficiency
const optimizationSuggestion = {
    type: "model_optimization",
    finding: "Using expensive model for simple data extraction tasks",
    recommendation: "Switch to gpt-3.5-turbo for steps 1-3, keep gpt-4o for analysis",
    expectedSavings: {
        creditsReduction: "60%",
        qualityImpact: "minimal (<5%)",
        timeImpact: "15% faster"
    },
    confidence: 0.89
};

// Agent applies optimization
const optimizedStrategy: ExecutionStrategy = {
    type: StrategyType.REASONING,
    configuration: {
        model: "gpt-3.5-turbo", // Cheaper model
        temperature: 0.3,
        maxTokens: 1500,
        fallbackModel: "gpt-4o" // Upgrade if quality insufficient
    },
    costMultiplier: 0.4, // 60% cost reduction
    qualityTarget: 0.85
};
```

---

## 🔄 Event-Driven Coordination

Here's how specialized agents provide capabilities through event processing:

### **Security Agent Example**

```typescript
// Security agent monitors for sensitive data exposure
const securityAgent = {
    subscriptions: [
        "data/access/*",
        "ai/generation/*", 
        "tool/execution/database_query"
    ],
    
    onEvent: async (event) => {
        if (event.type === "tool/execution/database_query") {
            const query = event.payload.query;
            const sensitivity = await analyzeSensitivity(query);
            
            if (sensitivity.level === "HIGH") {
                // Emit security concern
                await publishEvent({
                    type: "security/sensitive_data_access",
                    payload: {
                        query: query,
                        sensitivity: sensitivity,
                        agent: event.payload.agentId,
                        timestamp: new Date()
                    }
                });
                
                // If critical, stop execution
                if (sensitivity.risk === "CRITICAL") {
                    await publishEvent({
                        type: "security/emergency_stop",
                        payload: {
                            reason: "Critical data sensitivity violation",
                            affectedExecution: event.payload.runId
                        }
                    });
                }
            }
        }
    }
};
```

### **Quality Agent Example**

```typescript
// Quality agent monitors AI output quality
const qualityAgent = {
    subscriptions: [
        "ai/completion/*",
        "step/completed/*"
    ],
    
    onEvent: async (event) => {
        if (event.type === "ai/completion/generated") {
            const output = event.payload.output;
            const qualityAnalysis = await analyzeQuality(output);
            
            if (qualityAnalysis.score < 0.7) {
                // Suggest quality improvement
                await publishEvent({
                    type: "quality/improvement_suggested",
                    payload: {
                        originalOutput: output,
                        qualityScore: qualityAnalysis.score,
                        issues: qualityAnalysis.issues,
                        suggestions: qualityAnalysis.improvements,
                        agentId: event.payload.agentId
                    }
                });
            }
            
            // Track quality trends
            await updateQualityMetrics({
                agentId: event.payload.agentId,
                score: qualityAnalysis.score,
                timestamp: new Date()
            });
        }
    }
};
```

---

## 🏗️ Minimal Infrastructure Setup

Here's what the minimal, hard-coded infrastructure looks like:

### **Core Event Bus (Minimal)**

```typescript
// Simple, reliable event bus - minimal hard-coded infrastructure
class EventBus {
    private subscribers: Map<string, Function[]> = new Map();
    
    async publish(event: Event): Promise<void> {
        const handlers = this.subscribers.get(event.type) || [];
        await Promise.all(handlers.map(handler => handler(event)));
    }
    
    subscribe(eventType: string, handler: Function): void {
        if (!this.subscribers.has(eventType)) {
            this.subscribers.set(eventType, []);
        }
        this.subscribers.get(eventType)!.push(handler);
    }
    
    // Barrier synchronization for safety-critical events
    async publishWithBarrier(event: SafetyEvent, timeout: number = 2000): Promise<boolean> {
        const responses = new Map<string, boolean>();
        const barrierPromise = new Promise<boolean>((resolve) => {
            // Wait for all safety agents to respond
            const checkComplete = () => {
                const allResponded = Array.from(responses.values()).length >= this.safetyAgentCount;
                const anyAlarm = Array.from(responses.values()).some(r => r === false);
                
                if (anyAlarm) {
                    resolve(false); // ALARM - stop execution
                } else if (allResponded) {
                    resolve(true); // All OK - proceed
                }
            };
            
            // Set timeout
            setTimeout(() => resolve(false), timeout);
            
            // Listen for responses
            this.subscribe(`safety/response/${event.id}`, (response) => {
                responses.set(response.agentId, response.status === 'OK');
                checkComplete();
            });
        });
        
        await this.publish(event);
        return barrierPromise;
    }
}
```

### **State Machines (Minimal)**

```typescript
// Three core state machines - minimal hard-coded infrastructure
class SwarmStateMachine {
    async processGoal(goal: string): Promise<void> {
        // Simple goal processing - delegates complex reasoning to agents
        const swarmConfig = await this.initializeSwarm(goal);
        await this.delegateToLeader(swarmConfig);
    }
}

class RunStateMachine {
    async executeRoutine(request: RoutineExecutionRequest): Promise<RoutineExecutionResult> {
        // Simple workflow execution - delegates complex logic to navigators
        const navigator = this.getNavigator(request.navigatorType);
        return await navigator.execute(request);
    }
}

class UnifiedExecutor {
    async executeStep(request: StepExecutionRequest): Promise<StepExecutionResult> {
        // Simple step execution - delegates strategy to strategy engines
        const strategy = this.getStrategy(request.strategy.type);
        return await strategy.execute(request);
    }
}
```

**Everything Else is Agent-Driven:**
- Security monitoring → Security agent swarms
- Performance optimization → Optimization agent swarms  
- Quality assurance → Quality agent swarms
- Resource optimization → Resource management agents
- Error analysis → Error handling agents
- Strategy evolution → Evolution tracking agents

---

## 🎯 Key Takeaways

1. **Simple Infrastructure**: Only event bus and state machines are hard-coded
2. **Agent-Driven Capabilities**: All intelligence comes from deployable agent swarms
3. **Strategy Evolution**: Routines automatically improve through usage patterns
4. **Type Safety**: Everything is strongly typed through the centralized type system
5. **Event-Driven Coordination**: Asynchronous intelligence through event processing
6. **Resource Optimization**: Dynamic resource allocation with automatic optimization
7. **Error Resilience**: Multi-tier error handling with graceful degradation

This architecture creates a **self-improving system** where every execution makes the platform smarter, more efficient, and more capable. 🚀 