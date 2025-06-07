import { generatePK, generatePublicId } from "@vrooli/shared";
import {
    type BaseEvent,
    type EventSource,
    type EventMetadata,
    type EventSubscription,
    type EventFilter,
    type CoordinationEvent,
    type ProcessEvent,
    type ExecutionEvent,
    type SystemEvent,
    type SwarmLifecyclePayload,
    type TeamManagementPayload,
    type GoalManagementPayload,
    type RunLifecyclePayload,
    type StepExecutionPayload,
    type StrategyExecutionPayload,
    type ToolExecutionPayload,
    type SecurityEventPayload,
    type MonitoringEventPayload,
    type ErrorEventPayload,
    EventPriority,
    EventCategory,
} from "@vrooli/shared";

/**
 * Database fixtures for Event Bus - used for seeding test data
 * These support the event-driven communication between tiers
 */

// Consistent IDs for testing
export const eventDbIds = {
    event1: generatePK(),
    event2: generatePK(),
    event3: generatePK(),
    subscription1: generatePK(),
    subscription2: generatePK(),
    correlation1: generatePublicId(),
    correlation2: generatePublicId(),
    causation1: generatePublicId(),
};

/**
 * Default event metadata
 */
export const defaultEventMetadata: EventMetadata = {
    userId: generatePublicId(),
    sessionId: generatePublicId(),
    requestId: generatePublicId(),
    version: "1.0.0",
    tags: ["test"],
    priority: EventPriority.NORMAL,
};

/**
 * Event sources for each tier
 */
export const eventSources = {
    tier1: {
        tier: 1 as const,
        component: "SwarmStateMachine",
        instanceId: generatePK(),
    },
    tier2: {
        tier: 2 as const,
        component: "RunStateMachine",
        instanceId: generatePK(),
    },
    tier3: {
        tier: 3 as const,
        component: "UnifiedExecutor",
        instanceId: generatePK(),
    },
    crossCutting: {
        tier: "cross-cutting" as const,
        component: "EventBus",
        instanceId: generatePK(),
    },
};

/**
 * Minimal base event
 */
export const minimalEvent: BaseEvent = {
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
export const swarmLifecycleEvent: CoordinationEvent = {
    id: eventDbIds.event2,
    type: "swarm.lifecycle.started",
    timestamp: new Date(),
    source: eventSources.tier1,
    correlationId: eventDbIds.correlation1,
    metadata: {
        ...defaultEventMetadata,
        priority: EventPriority.HIGH,
        tags: ["swarm", "lifecycle"],
    },
    payload: {
        type: "swarm_lifecycle",
        swarmId: generatePK(),
        state: "EXECUTING",
        previousState: "PLANNING",
        reason: "Planning phase completed successfully",
    } as SwarmLifecyclePayload,
};

/**
 * Coordination event - Team management
 */
export const teamManagementEvent: CoordinationEvent = {
    id: generatePK(),
    type: "team.management.formed",
    timestamp: new Date(),
    source: eventSources.tier1,
    correlationId: eventDbIds.correlation1,
    metadata: defaultEventMetadata,
    payload: {
        type: "team_management",
        teamId: generatePK(),
        action: "formed",
        agents: [generatePK(), generatePK(), generatePK()],
        goal: "Process customer data analysis",
    } as TeamManagementPayload,
};

/**
 * Process event - Run lifecycle
 */
export const runLifecycleEvent: ProcessEvent = {
    id: generatePK(),
    type: "run.lifecycle.started",
    timestamp: new Date(),
    source: eventSources.tier2,
    correlationId: eventDbIds.correlation1,
    metadata: defaultEventMetadata,
    payload: {
        type: "run_lifecycle",
        runId: generatePK(),
        routineId: generatePK(),
        state: "RUNNING",
        previousState: "READY",
    } as RunLifecyclePayload,
};

/**
 * Process event - Step execution
 */
export const stepExecutionEvent: ProcessEvent = {
    id: generatePK(),
    type: "step.execution.completed",
    timestamp: new Date(),
    source: eventSources.tier2,
    correlationId: eventDbIds.correlation1,
    causationId: eventDbIds.causation1,
    metadata: {
        ...defaultEventMetadata,
        tags: ["step", "execution", "success"],
    },
    payload: {
        type: "step_execution",
        stepId: generatePK(),
        runId: generatePK(),
        action: "completed",
        result: { processed: 100, duration: 5000 },
    } as StepExecutionPayload,
};

/**
 * Execution event - Strategy execution
 */
export const strategyExecutionEvent: ExecutionEvent = {
    id: generatePK(),
    type: "strategy.execution.success",
    timestamp: new Date(),
    source: eventSources.tier3,
    correlationId: eventDbIds.correlation1,
    metadata: defaultEventMetadata,
    payload: {
        type: "strategy_execution",
        stepId: generatePK(),
        strategy: "reasoning",
        result: "success",
        confidence: 0.92,
        resourceUsage: {
            tokens: 1500,
            time: 3500,
            cost: 0.05,
        },
    } as StrategyExecutionPayload,
};

/**
 * Execution event - Tool execution
 */
export const toolExecutionEvent: ExecutionEvent = {
    id: generatePK(),
    type: "tool.execution.completed",
    timestamp: new Date(),
    source: eventSources.tier3,
    correlationId: eventDbIds.correlation1,
    metadata: {
        ...defaultEventMetadata,
        priority: EventPriority.LOW,
    },
    payload: {
        type: "tool_execution",
        toolName: "dataAnalyzer",
        stepId: generatePK(),
        parameters: {
            dataset: "customers",
            metric: "retention",
        },
        result: { retention: 0.85, trend: "increasing" },
        duration: 2500,
    } as ToolExecutionPayload,
};

/**
 * System event - Security
 */
export const securityEvent: SystemEvent = {
    id: generatePK(),
    type: "security.violation.detected",
    timestamp: new Date(),
    source: eventSources.crossCutting,
    correlationId: eventDbIds.correlation2,
    metadata: {
        ...defaultEventMetadata,
        priority: EventPriority.CRITICAL,
        tags: ["security", "alert"],
    },
    payload: {
        type: "security",
        action: "violation",
        severity: "high",
        details: {
            violationType: "unauthorized_access",
            resource: "sensitive_data",
            actor: "unknown_agent",
            blocked: true,
        },
    } as SecurityEventPayload,
};

/**
 * System event - Monitoring
 */
export const monitoringEvent: SystemEvent = {
    id: generatePK(),
    type: "monitoring.metric.threshold",
    timestamp: new Date(),
    source: eventSources.crossCutting,
    correlationId: eventDbIds.correlation2,
    metadata: {
        ...defaultEventMetadata,
        priority: EventPriority.HIGH,
        ttl: 3600, // 1 hour
    },
    payload: {
        type: "monitoring",
        metric: "memory_usage",
        value: 0.92,
        threshold: 0.8,
        status: "critical",
    } as MonitoringEventPayload,
};

/**
 * System event - Error
 */
export const errorEvent: SystemEvent = {
    id: generatePK(),
    type: "error.execution.failed",
    timestamp: new Date(),
    source: eventSources.crossCutting,
    correlationId: eventDbIds.correlation2,
    metadata: {
        ...defaultEventMetadata,
        priority: EventPriority.HIGH,
        tags: ["error", "execution"],
    },
    payload: {
        type: "error",
        errorType: "ExecutionError",
        message: "Failed to execute step: timeout exceeded",
        stack: "Error: Failed to execute step\n  at executor.js:123",
        context: {
            stepId: generatePK(),
            duration: 300000,
            attempts: 3,
        },
        severity: "high",
    } as ErrorEventPayload,
};

/**
 * Event subscription fixtures
 */
export const basicSubscription: EventSubscription = {
    id: eventDbIds.subscription1,
    subscriber: eventSources.tier2,
    filters: [
        {
            field: "type",
            operator: "startsWith",
            value: "swarm.",
        },
    ],
    handler: "handleSwarmEvents",
    config: {
        maxRetries: 3,
        timeout: 30000,
    },
};

export const advancedSubscription: EventSubscription = {
    id: eventDbIds.subscription2,
    subscriber: eventSources.tier1,
    filters: [
        {
            field: "metadata.priority",
            operator: "in",
            value: [EventPriority.HIGH, EventPriority.CRITICAL],
        },
        {
            field: "source.tier",
            operator: "equals",
            value: 3,
        },
    ],
    handler: "handleHighPriorityExecutionEvents",
    config: {
        maxRetries: 5,
        timeout: 60000,
        deadLetterQueue: "dlq.high-priority",
        batchSize: 10,
        batchTimeout: 5000,
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
            id: generatePK(),
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
        overrides?: Partial<CoordinationEvent>
    ): CoordinationEvent {
        const payloads = {
            swarm: {
                type: "swarm_lifecycle",
                swarmId: generatePK(),
                state: "EXECUTING",
            } as SwarmLifecyclePayload,
            team: {
                type: "team_management",
                teamId: generatePK(),
                action: "formed",
                agents: [generatePK()],
            } as TeamManagementPayload,
            goal: {
                type: "goal_management",
                goalId: generatePK(),
                action: "assigned",
                assignedTo: generatePK(),
            } as GoalManagementPayload,
            resource: {
                type: "resource_allocation",
                resourceId: generatePK(),
                action: "allocated",
                consumer: generatePK(),
                amount: 100,
            },
        };

        return {
            id: generatePK(),
            type: `${eventType}.event`,
            timestamp: new Date(),
            source: eventSources.tier1,
            correlationId: generatePublicId(),
            metadata: defaultEventMetadata,
            payload: payloads[eventType] as any,
            ...overrides,
        };
    }

    /**
     * Create process event
     */
    static createProcessEvent(
        eventType: "run" | "step" | "navigation" | "optimization",
        overrides?: Partial<ProcessEvent>
    ): ProcessEvent {
        const payloads = {
            run: {
                type: "run_lifecycle",
                runId: generatePK(),
                routineId: generatePK(),
                state: "RUNNING",
            } as RunLifecyclePayload,
            step: {
                type: "step_execution",
                stepId: generatePK(),
                runId: generatePK(),
                action: "started",
            } as StepExecutionPayload,
            navigation: {
                type: "navigation",
                runId: generatePK(),
                from: "step1",
                to: ["step2", "step3"],
                reason: "Conditional branch",
            },
            optimization: {
                type: "optimization",
                runId: generatePK(),
                optimizationType: "parallelize",
                target: ["step2", "step3"],
                expectedImprovement: 0.3,
                applied: true,
            },
        };

        return {
            id: generatePK(),
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
                stepId: generatePK(),
                strategy: "reasoning",
                result: "success",
                confidence: 0.85,
                resourceUsage: { tokens: 1000 },
            } as StrategyExecutionPayload,
            tool: {
                type: "tool_execution",
                toolName: "calculator",
                stepId: generatePK(),
                parameters: { operation: "sum", values: [1, 2, 3] },
                result: 6,
                duration: 100,
            } as ToolExecutionPayload,
            adaptation: {
                type: "adaptation",
                stepId: generatePK(),
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
            id: generatePK(),
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
        overrides?: Partial<SystemEvent>
    ): SystemEvent {
        const payloads = {
            security: {
                type: "security",
                action: "audit",
                severity,
                details: { action: "data_access", user: generatePublicId() },
            } as SecurityEventPayload,
            monitoring: {
                type: "monitoring",
                metric: "cpu_usage",
                value: 0.75,
                threshold: 0.8,
                status: severity === "critical" ? "critical" : "warning",
            } as MonitoringEventPayload,
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
            } as ErrorEventPayload,
        };

        return {
            id: generatePK(),
            type: `${eventType}.event`,
            timestamp: new Date(),
            source: eventSources.crossCutting,
            correlationId: generatePublicId(),
            metadata: {
                ...defaultEventMetadata,
                priority: severity === "critical" ? EventPriority.CRITICAL
                    : severity === "high" ? EventPriority.HIGH
                    : EventPriority.NORMAL,
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
        filters: EventFilter[],
        overrides?: Partial<EventSubscription>
    ): EventSubscription {
        return {
            id: generatePK(),
            subscriber: eventSources.tier2,
            filters,
            handler: "handleEvents",
            config: {
                maxRetries: 3,
                timeout: 30000,
            },
            ...overrides,
        };
    }
}

/**
 * Helper to create event filter
 */
export function createEventFilter(
    field: string,
    operator: EventFilter["operator"],
    value: unknown
): EventFilter {
    return { field, operator, value };
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
            EventFactory.createSubscription([
                createEventFilter("type", "startsWith", "swarm."),
            ]),
            EventFactory.createSubscription([
                createEventFilter("metadata.priority", "equals", EventPriority.HIGH),
            ]),
            EventFactory.createSubscription([
                createEventFilter("source.tier", "in", [1, 2]),
            ]),
        ];

        for (const subscription of subscriptions) {
            await eventBus.subscribe(subscription);
        }
    }

    return events;
}