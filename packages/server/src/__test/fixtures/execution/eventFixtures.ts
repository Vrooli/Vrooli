import { generatePublicId, generatePK } from "@vrooli/shared";
import {
    type BaseEvent,
    type ExecutionEvent,
    type EventSubscription,
    type SwarmEventType,
    type StepEventType,
    type SystemEventType,
} from "@vrooli/shared";
import { TEST_IDS, TestIdFactory } from "./testIdGenerator.js";

/**
 * Database fixtures for Event Bus - used for seeding test data
 * These support the event-driven communication between tiers
 */

// Consistent IDs for testing
export const eventDbIds = {
    event1: TEST_IDS.EVENT_1,
    event2: TEST_IDS.EVENT_2,
    event3: TEST_IDS.EVENT_3,
    subscription1: TestIdFactory.event(1001),
    subscription2: TestIdFactory.event(1002),
    correlation1: generatePublicId(),
    correlation2: generatePublicId(),
    causation1: generatePublicId(),
};

/**
 * Default event metadata
 */
export const defaultEventMetadata: Record<string, unknown> = {
    userId: generatePublicId(),
    sessionId: generatePublicId(),
    requestId: generatePublicId(),
    version: "1.0.0",
    tags: ["test"],
    priority: "NORMAL",
};

/**
 * Event sources for each tier
 */
export const eventSources = {
    tier1: {
        tier: "tier1.swarm",
        component: "SwarmStateMachine",
        instanceId: generatePK().toString(),
    },
    tier2: {
        tier: "tier2.run",
        component: "RunStateMachine",
        instanceId: generatePK().toString().toString(),
    },
    tier3: {
        tier: "tier3.step",
        component: "UnifiedExecutor",
        instanceId: generatePK().toString().toString(),
    },
    crossCutting: {
        tier: "cross-cutting",
        component: "EventBus",
        instanceId: generatePK().toString().toString(),
    },
};

/**
 * Minimal base event
 */
export const minimalEvent: ExecutionEvent = {
    id: eventDbIds.event1,
    type: "test.event",
    timestamp: new Date(),
    source: eventSources.tier1,
    correlationId: eventDbIds.correlation1,
    metadata: defaultEventMetadata,
};

/**
 * Coordination event - Swarm lifecycle
 */
export const swarmLifecycleEvent: ExecutionEvent = {
    id: eventDbIds.event2,
    type: "swarm.lifecycle.started",
    timestamp: new Date(),
    source: eventSources.tier1,
    correlationId: eventDbIds.correlation1,
    metadata: {
        ...defaultEventMetadata,
        tags: ["swarm", "lifecycle"],
    },
    data: {
        type: "swarm_lifecycle",
        swarmId: generatePK().toString().toString(),
        state: "EXECUTING",
        previousState: "PLANNING",
        reason: "Planning phase completed successfully",
    },
};

/**
 * Coordination event - Team management
 */
export const teamManagementEvent: ExecutionEvent = {
    id: generatePK().toString().toString(),
    type: "team.management.formed",
    timestamp: new Date(),
    source: eventSources.tier1,
    correlationId: eventDbIds.correlation1,
    metadata: defaultEventMetadata,
    data: {
        type: "team_management",
        teamId: generatePK().toString().toString(),
        action: "formed",
        agents: [generatePK().toString().toString(), generatePK().toString().toString(), generatePK().toString().toString()],
        goal: "Process customer data analysis",
    },
};

/**
 * Process event - Run lifecycle
 */
export const runLifecycleEvent: ExecutionEvent = {
    id: generatePK().toString().toString(),
    type: "run.lifecycle.started",
    timestamp: new Date(),
    source: eventSources.tier2,
    correlationId: eventDbIds.correlation1,
    metadata: defaultEventMetadata,
    data: {
        type: "run_lifecycle",
        runId: generatePK().toString().toString(),
        routineId: generatePK().toString().toString(),
        state: "RUNNING",
        previousState: "READY",
    },
};

/**
 * Process event - Step execution
 */
export const stepExecutionEvent: ExecutionEvent = {
    id: generatePK().toString().toString(),
    type: "step.execution.completed",
    timestamp: new Date(),
    source: eventSources.tier2,
    correlationId: eventDbIds.correlation1,
    causationId: eventDbIds.causation1,
    metadata: {
        ...defaultEventMetadata,
        tags: ["step", "execution", "success"],
    },
    data: {
        type: "step_execution",
        stepId: generatePK().toString().toString(),
        runId: generatePK().toString().toString(),
        action: "completed",
        result: { processed: 100, duration: 5000 },
    },
};

/**
 * Execution event - Strategy execution
 */
export const strategyExecutionEvent: ExecutionEvent = {
    id: generatePK().toString(),
    type: "strategy.execution.success",
    timestamp: new Date(),
    source: eventSources.tier3,
    correlationId: eventDbIds.correlation1,
    metadata: defaultEventMetadata,
    data: {
        type: "strategy_execution",
        stepId: generatePK().toString(),
        strategy: "reasoning",
        result: "success",
        confidence: 0.92,
        resourceUsage: {
            tokens: 1500,
            time: 3500,
            cost: 0.05,
        },
    } ,
};

/**
 * Execution event - Tool execution
 */
export const toolExecutionEvent: ExecutionEvent = {
    id: generatePK().toString(),
    type: "tool.execution.completed",
    timestamp: new Date(),
    source: eventSources.tier3,
    correlationId: eventDbIds.correlation1,
    metadata: {
        ...defaultEventMetadata,
        priority: "LOW",
    },
    data: {
        type: "tool_execution",
        toolName: "dataAnalyzer",
        stepId: generatePK().toString(),
        parameters: {
            dataset: "customers",
            metric: "retention",
        },
        result: { retention: 0.85, trend: "increasing" },
        duration: 2500,
    } ,
};

/**
 * System event - Security
 */
export const securityEvent: ExecutionEvent = {
    id: generatePK().toString(),
    type: "security.violation.detected",
    timestamp: new Date(),
    source: eventSources.crossCutting,
    correlationId: eventDbIds.correlation2,
    metadata: {
        ...defaultEventMetadata,
        priority: "CRITICAL",
        tags: ["security", "alert"],
    },
    data: {
        type: "security",
        action: "violation",
        severity: "high",
        details: {
            violationType: "unauthorized_access",
            resource: "sensitive_data",
            actor: "unknown_agent",
            blocked: true,
        },
    } ,
};

/**
 * System event - Monitoring
 */
export const monitoringEvent: ExecutionEvent = {
    id: generatePK().toString(),
    type: "monitoring.metric.threshold",
    timestamp: new Date(),
    source: eventSources.crossCutting,
    correlationId: eventDbIds.correlation2,
    metadata: {
        ...defaultEventMetadata,
        priority: "HIGH",
        ttl: 3600, // 1 hour
    },
    data: {
        type: "monitoring",
        metric: "memory_usage",
        value: 0.92,
        threshold: 0.8,
        status: "critical",
    } ,
};

/**
 * System event - Error
 */
export const errorEvent: ExecutionEvent = {
    id: generatePK().toString(),
    type: "error.execution.failed",
    timestamp: new Date(),
    source: eventSources.crossCutting,
    correlationId: eventDbIds.correlation2,
    metadata: {
        ...defaultEventMetadata,
        priority: "HIGH",
        tags: ["error", "execution"],
    },
    data: {
        type: "error",
        errorType: "ExecutionError",
        message: "Failed to execute step: timeout exceeded",
        stack: "Error: Failed to execute step\n  at executor.js:123",
        context: {
            stepId: generatePK().toString(),
            duration: 300000,
            attempts: 3,
        },
        severity: "high",
    } ,
};

/**
 * Event subscription fixtures
 */
export const basicSubscription: EventSubscription = {
    pattern: "swarm.*",
    handler: (event: ExecutionEvent) => {
        console.log("Handling swarm event:", event);
    },
};

export const advancedSubscription: EventSubscription = {
    pattern: "*.execution.*",
    handler: (event: ExecutionEvent) => {
        console.log("Handling execution event:", event);
    },
    filter: (event: ExecutionEvent) => {
        const priority = event.metadata?.priority;
        return priority === "HIGH" || priority === "CRITICAL";
    },
};

/**
 * Factory for creating event fixtures with overrides
 */
export class EventFactory {
    /**
     * Create basic event
     */
    static createEvent(
        type: string,
        source: EventSource,
        overrides?: Partial<BaseEvent>
    ): BaseEvent {
        return {
            id: generatePK().toString(),
            type,
            timestamp: new Date(),
            source,
            correlationId: generatePublicId(),
            metadata: defaultEventMetadata,
            ...overrides,
        };
    }

    /**
     * Create coordination event
     */
    static createCoordinationEvent(
        eventType: "swarm" | "team" | "goal" | "resource",
        overrides?: Partial<ExecutionEvent>
    ): ExecutionEvent {
        const payloads = {
            swarm: {
                type: "swarm_lifecycle",
                swarmId: generatePK().toString(),
                state: "EXECUTING",
            } ,
            team: {
                type: "team_management",
                teamId: generatePK().toString(),
                action: "formed",
                agents: [generatePK().toString()],
            } ,
            goal: {
                type: "goal_management",
                goalId: generatePK().toString(),
                action: "assigned",
                assignedTo: generatePK().toString(),
            } ,
            resource: {
                type: "resource_allocation",
                resourceId: generatePK().toString(),
                action: "allocated",
                consumer: generatePK().toString(),
                amount: 100,
            },
        };

        return {
            id: generatePK().toString(),
            type: `${eventType}.event`,
            timestamp: new Date(),
            source: eventSources.tier1,
            correlationId: generatePublicId(),
            metadata: defaultEventMetadata,
            data: payloads[eventType] as any,
            ...overrides,
        };
    }

    /**
     * Create process event
     */
    static createProcessEvent(
        eventType: "run" | "step" | "navigation" | "optimization",
        overrides?: Partial<ExecutionEvent>
    ): ExecutionEvent {
        const payloads = {
            run: {
                type: "run_lifecycle",
                runId: generatePK().toString(),
                routineId: generatePK().toString(),
                state: "RUNNING",
            } ,
            step: {
                type: "step_execution",
                stepId: generatePK().toString(),
                runId: generatePK().toString(),
                action: "started",
            } ,
            navigation: {
                type: "navigation",
                runId: generatePK().toString(),
                from: "step1",
                to: ["step2", "step3"],
                reason: "Conditional branch",
            },
            optimization: {
                type: "optimization",
                runId: generatePK().toString(),
                optimizationType: "parallelize",
                target: ["step2", "step3"],
                expectedImprovement: 0.3,
                applied: true,
            },
        };

        return {
            id: generatePK().toString(),
            type: `${eventType}.event`,
            timestamp: new Date(),
            source: eventSources.tier2,
            correlationId: generatePublicId(),
            metadata: defaultEventMetadata,
            payload: payloads[eventType] as any,
            ...overrides,
        };
    }

    /**
     * Create execution event
     */
    static createExecutionEvent(
        eventType: "strategy" | "tool" | "adaptation" | "learning",
        overrides?: Partial<ExecutionEvent>
    ): ExecutionEvent {
        const payloads = {
            strategy: {
                type: "strategy_execution",
                stepId: generatePK().toString(),
                strategy: "reasoning",
                result: "success",
                confidence: 0.85,
                resourceUsage: { tokens: 1000 },
            } ,
            tool: {
                type: "tool_execution",
                toolName: "calculator",
                stepId: generatePK().toString(),
                parameters: { operation: "sum", values: [1, 2, 3] },
                result: 6,
                duration: 100,
            } ,
            adaptation: {
                type: "adaptation",
                stepId: generatePK().toString(),
                adaptationType: "strategy",
                before: "deterministic",
                after: "reasoning",
                reason: "Complex input detected",
            },
            learning: {
                type: "learning",
                pattern: "tool_sequence_optimization",
                confidence: 0.9,
                impact: 0.25,
                applicableTo: ["similar_workflows"],
            },
        };

        return {
            id: generatePK().toString(),
            type: `${eventType}.event`,
            timestamp: new Date(),
            source: eventSources.tier3,
            correlationId: generatePublicId(),
            metadata: defaultEventMetadata,
            payload: payloads[eventType] as any,
            ...overrides,
        };
    }

    /**
     * Create system event
     */
    static createSystemEvent(
        eventType: "security" | "monitoring" | "resource" | "error",
        severity: "low" | "medium" | "high" | "critical" = "medium",
        overrides?: Partial<ExecutionEvent>
    ): ExecutionEvent {
        const payloads = {
            security: {
                type: "security",
                action: "audit",
                severity,
                details: { action: "data_access", user: generatePublicId() },
            } ,
            monitoring: {
                type: "monitoring",
                metric: "cpu_usage",
                value: 0.75,
                threshold: 0.8,
                status: severity === "critical" ? "critical" : "warning",
            } ,
            resource: {
                type: "resource",
                resourceType: "memory",
                action: "limited",
                current: 900,
                limit: 1000,
            },
            error: {
                type: "error",
                errorType: "SystemError",
                message: "Test error",
                severity,
                context: {},
            } ,
        };

        return {
            id: generatePK().toString(),
            type: `${eventType}.event`,
            timestamp: new Date(),
            source: eventSources.crossCutting,
            correlationId: generatePublicId(),
            metadata: {
                ...defaultEventMetadata,
                priority: severity === "critical" ? "CRITICAL"
                    : severity === "high" ? "HIGH"
                    : "NORMAL",
            },
            payload: payloads[eventType] as any,
            ...overrides,
        };
    }

    /**
     * Create event chain (related events)
     */
    static createEventChain(
        count: number = 3,
        correlationId?: string
    ): BaseEvent[] {
        const events: BaseEvent[] = [];
        const correlation = correlationId || generatePublicId();
        let causationId: string | undefined;

        for (let i = 0; i < count; i++) {
            const event = this.createEvent(
                `chain.event.${i}`,
                i % 3 === 0 ? eventSources.tier1 
                    : i % 3 === 1 ? eventSources.tier2 
                    : eventSources.tier3,
                {
                    correlationId: correlation,
                    causationId,
                    metadata: {
                        ...defaultEventMetadata,
                        tags: ["chain", `step-${i}`],
                    },
                }
            );
            
            events.push(event);
            causationId = event.id; // Next event caused by this one
        }

        return events;
    }

    /**
     * Create subscription
     */
    static createSubscription(
        pattern: string,
        overrides?: Partial<EventSubscription>
    ): EventSubscription {
        return {
            pattern,
            handler: (event: ExecutionEvent) => {
                console.log(`Handling event ${event.type}:`, event);
            },
            ...overrides,
        };
    }
}

/**
 * Simple pattern matcher for event subscriptions
 */
export function createEventPattern(eventType: string): string {
    return `${eventType}.*`;
}

/**
 * Helper to create batch of events
 */
export function createEventBatch(
    options: {
        count?: number;
        types?: Array<"coordination" | "process" | "execution" | "system">;
        correlationId?: string;
        timeSpan?: number; // milliseconds
    } = {}
): BaseEvent[] {
    const count = options.count || 10;
    const types = options.types || ["coordination", "process", "execution", "system"];
    const correlationId = options.correlationId || generatePublicId();
    const timeSpan = options.timeSpan || 60000; // 1 minute
    const events: BaseEvent[] = [];
    const startTime = Date.now() - timeSpan;

    for (let i = 0; i < count; i++) {
        const eventType = types[i % types.length];
        let event: BaseEvent;

        switch (eventType) {
            case "coordination":
                event = EventFactory.createCoordinationEvent(
                    ["swarm", "team", "goal"][i % 3] as any,
                    {
                        correlationId,
                        timestamp: new Date(startTime + (i * timeSpan / count)),
                    }
                );
                break;
            case "process":
                event = EventFactory.createProcessEvent(
                    ["run", "step", "navigation"][i % 3] as any,
                    {
                        correlationId,
                        timestamp: new Date(startTime + (i * timeSpan / count)),
                    }
                );
                break;
            case "execution":
                event = EventFactory.createExecutionEvent(
                    ["strategy", "tool", "adaptation"][i % 3] as any,
                    {
                        correlationId,
                        timestamp: new Date(startTime + (i * timeSpan / count)),
                    }
                );
                break;
            case "system":
                event = EventFactory.createSystemEvent(
                    ["security", "monitoring", "error"][i % 3] as any,
                    ["low", "medium", "high"][i % 3] as any,
                    {
                        correlationId,
                        timestamp: new Date(startTime + (i * timeSpan / count)),
                    }
                );
                break;
        }

        events.push(event);
    }

    return events;
}

/**
 * Helper to seed test events
 */
export async function seedTestEvents(
    eventBus: any,
    count: number = 10,
    options?: {
        correlationId?: string;
        includeSubscriptions?: boolean;
    }
) {
    const events = createEventBatch({ count, correlationId: options?.correlationId });
    
    // Publish events
    for (const event of events) {
        await eventBus.publish(event);
    }

    // Create subscriptions if requested
    if (options?.includeSubscriptions) {
        const subscriptions = [
            EventFactory.createSubscription("swarm.*"),
            EventFactory.createSubscription("*.execution.*"),
            EventFactory.createSubscription("run.*"),
        ];

        for (const subscription of subscriptions) {
            await eventBus.subscribe(subscription);
        }
    }

    return events;
}

/**
 * Mock events for testing
 */
export const mockEvents = {
    swarmLifecycle: swarmLifecycleEvent,
    teamManagement: teamManagementEvent,
    runLifecycle: runLifecycleEvent,
    stepExecution: stepExecutionEvent,
    strategyExecution: strategyExecutionEvent,
    toolExecution: toolExecutionEvent,
    security: securityEvent,
    monitoring: monitoringEvent,
    error: errorEvent,
};