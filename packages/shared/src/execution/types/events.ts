/**
 * Core type definitions for the event-driven architecture
 * These types define the communication protocol between tiers
 */

/**
 * Base event interface for all system events
 */
export interface BaseEvent {
    id: string;
    type: string;
    timestamp: Date;
    source: EventSource;
    correlationId: string;
    causationId?: string;
    metadata: EventMetadata;
}

/**
 * Event source identification
 */
export interface EventSource {
    tier: 1 | 2 | 3 | "cross-cutting";
    component: string;
    instanceId: string;
}

/**
 * Event metadata
 */
export interface EventMetadata {
    userId?: string;
    sessionId?: string;
    requestId?: string;
    version: string;
    tags: string[];
    priority: EventPriority;
    ttl?: number; // Time to live in seconds
}

/**
 * Event priority levels
 */
export enum EventPriority {
    LOW = "LOW",
    NORMAL = "NORMAL",
    HIGH = "HIGH",
    CRITICAL = "CRITICAL"
}

/**
 * Event categories for routing
 */
export enum EventCategory {
    // Lifecycle events
    LIFECYCLE = "LIFECYCLE",
    
    // Coordination events
    COORDINATION = "COORDINATION",
    TEAM_MANAGEMENT = "TEAM_MANAGEMENT",
    GOAL_MANAGEMENT = "GOAL_MANAGEMENT",
    
    // Process events
    PROCESS = "PROCESS",
    NAVIGATION = "NAVIGATION",
    OPTIMIZATION = "OPTIMIZATION",
    
    // Execution events
    EXECUTION = "EXECUTION",
    STRATEGY = "STRATEGY",
    TOOL = "TOOL",
    
    // Cross-cutting events
    SECURITY = "SECURITY",
    MONITORING = "MONITORING",
    RESOURCE = "RESOURCE",
    ERROR = "ERROR"
}

/**
 * Event subscription configuration
 */
export interface EventSubscription {
    id: string;
    subscriber: EventSource;
    filters: EventFilter[];
    handler: string; // Handler function name
    config: SubscriptionConfig;
}

/**
 * Event filter
 */
export interface EventFilter {
    field: string;
    operator: "equals" | "contains" | "startsWith" | "endsWith" | "in" | "regex";
    value: unknown;
}

/**
 * Subscription configuration
 */
export interface SubscriptionConfig {
    maxRetries: number;
    timeout: number;
    deadLetterQueue?: string;
    batchSize?: number;
    batchTimeout?: number;
}

/**
 * Event bus interface
 */
export interface EventBus {
    // Publishing
    publish(event: BaseEvent): Promise<void>;
    publishBatch(events: BaseEvent[]): Promise<void>;
    
    // Subscribing
    subscribe(subscription: EventSubscription): Promise<void>;
    unsubscribe(subscriptionId: string): Promise<void>;
    
    // Query
    getEvents(query: EventQuery): Promise<BaseEvent[]>;
    getEventStream(query: EventQuery): AsyncIterator<BaseEvent>;
}

/**
 * Event query
 */
export interface EventQuery {
    filters: EventFilter[];
    timeRange?: TimeRange;
    limit?: number;
    offset?: number;
    orderBy?: string;
    orderDirection?: "asc" | "desc";
}

/**
 * Time range for queries
 */
export interface TimeRange {
    start: Date;
    end: Date;
}

/**
 * Tier 1 Coordination Events
 */
export interface CoordinationEvent extends BaseEvent {
    source: EventSource & { tier: 1 };
    payload: CoordinationEventPayload;
}

export type CoordinationEventPayload = 
    | SwarmLifecyclePayload
    | TeamManagementPayload
    | GoalManagementPayload
    | ResourceAllocationPayload
    | CommunicationPayload;

export interface SwarmLifecyclePayload {
    type: "swarm_lifecycle";
    swarmId: string;
    state: string;
    previousState?: string;
    reason?: string;
}

export interface TeamManagementPayload {
    type: "team_management";
    teamId: string;
    action: "formed" | "disbanded" | "modified";
    agents: string[];
    goal?: string;
}

export interface GoalManagementPayload {
    type: "goal_management";
    goalId: string;
    action: "assigned" | "completed" | "failed" | "modified";
    assignedTo?: string;
    result?: unknown;
}

export interface ResourceAllocationPayload {
    type: "resource_allocation";
    resourceId: string;
    action: "allocated" | "released" | "exhausted";
    consumer: string;
    amount?: number;
}

export interface CommunicationPayload {
    type: "communication";
    action: "message" | "consensus" | "conflict";
    participants: string[];
    content: unknown;
}

/**
 * Tier 2 Process Events
 */
export interface ProcessEvent extends BaseEvent {
    source: EventSource & { tier: 2 };
    payload: ProcessEventPayload;
}

export type ProcessEventPayload = 
    | RunLifecyclePayload
    | StepExecutionPayload
    | NavigationPayload
    | OptimizationPayload
    | ContextUpdatePayload;

export interface RunLifecyclePayload {
    type: "run_lifecycle";
    runId: string;
    routineId: string;
    state: string;
    previousState?: string;
    error?: string;
}

export interface StepExecutionPayload {
    type: "step_execution";
    stepId: string;
    runId: string;
    action: "started" | "completed" | "failed" | "skipped";
    result?: unknown;
    error?: string;
}

export interface NavigationPayload {
    type: "navigation";
    runId: string;
    from: string;
    to: string[];
    reason: string;
}

export interface OptimizationPayload {
    type: "optimization";
    runId: string;
    optimizationType: string;
    target: string[];
    expectedImprovement: number;
    applied: boolean;
}

export interface ContextUpdatePayload {
    type: "context_update";
    runId: string;
    updates: Record<string, unknown>;
    scope: string;
}

/**
 * Tier 3 Execution Events
 */
export interface ExecutionEvent extends BaseEvent {
    source: EventSource & { tier: 3 };
    payload: ExecutionEventPayload;
}

export type ExecutionEventPayload = 
    | StrategyExecutionPayload
    | ToolExecutionPayload
    | AdaptationPayload
    | LearningPayload;

export interface StrategyExecutionPayload {
    type: "strategy_execution";
    stepId: string;
    strategy: string;
    result: "success" | "failure" | "partial";
    confidence: number;
    resourceUsage: Record<string, number>;
}

export interface ToolExecutionPayload {
    type: "tool_execution";
    toolName: string;
    stepId: string;
    parameters: Record<string, unknown>;
    result?: unknown;
    error?: string;
    duration: number;
}

export interface AdaptationPayload {
    type: "adaptation";
    stepId: string;
    adaptationType: string;
    before: unknown;
    after: unknown;
    reason: string;
}

export interface LearningPayload {
    type: "learning";
    pattern: string;
    confidence: number;
    impact: number;
    applicableTo: string[];
}

/**
 * Cross-cutting Events
 */
export interface SystemEvent extends BaseEvent {
    source: EventSource & { tier: "cross-cutting" };
    payload: SystemEventPayload;
}

export type SystemEventPayload = 
    | SecurityEventPayload
    | MonitoringEventPayload
    | ResourceEventPayload
    | ErrorEventPayload;

export interface SecurityEventPayload {
    type: "security";
    action: "violation" | "authentication" | "authorization" | "audit";
    severity: "low" | "medium" | "high" | "critical";
    details: Record<string, unknown>;
}

export interface MonitoringEventPayload {
    type: "monitoring";
    metric: string;
    value: number;
    threshold?: number;
    status: "normal" | "warning" | "critical";
}

export interface ResourceEventPayload {
    type: "resource";
    resourceType: string;
    action: "allocated" | "released" | "exhausted" | "limited";
    current: number;
    limit?: number;
}

export interface ErrorEventPayload {
    type: "error";
    errorType: string;
    message: string;
    stack?: string;
    context: Record<string, unknown>;
    severity: "low" | "medium" | "high" | "critical";
}

/**
 * Event store interface for persistence
 */
export interface EventStore {
    save(event: BaseEvent): Promise<void>;
    saveBatch(events: BaseEvent[]): Promise<void>;
    get(eventId: string): Promise<BaseEvent | null>;
    query(query: EventQuery): Promise<BaseEvent[]>;
    stream(query: EventQuery): AsyncIterator<BaseEvent>;
}

/**
 * Event replay configuration
 */
export interface EventReplayConfig {
    fromTimestamp: Date;
    toTimestamp?: Date;
    eventTypes?: string[];
    speed?: number; // 1 = real-time, 2 = 2x speed, etc.
    filters?: EventFilter[];
}
