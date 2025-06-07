/**
 * Event type registry
 * Central registry of all event types in the system
 */

import { 
    SwarmEventType,
    RunEventType,
    EventCategory, 
} from "../types/events.js";

/**
 * Event type metadata
 */
export interface EventTypeMetadata {
    type: string;
    category: EventCategory;
    tier: 1 | 2 | 3 | "cross-cutting";
    description: string;
    schema?: Record<string, unknown>;
    examples?: unknown[];
}

/**
 * Event type registry
 */
export class EventTypeRegistry {
    private static instance: EventTypeRegistry;
    private registry = new Map<string, EventTypeMetadata>();
    
    private constructor() {
        this.registerDefaultTypes();
    }
    
    static getInstance(): EventTypeRegistry {
        if (!this.instance) {
            this.instance = new EventTypeRegistry();
        }
        return this.instance;
    }
    
    register(metadata: EventTypeMetadata): void {
        this.registry.set(metadata.type, metadata);
    }
    
    get(type: string): EventTypeMetadata | undefined {
        return this.registry.get(type);
    }
    
    getByCategory(category: EventCategory): EventTypeMetadata[] {
        return Array.from(this.registry.values()).filter(
            meta => meta.category === category,
        );
    }
    
    getByTier(tier: 1 | 2 | 3 | "cross-cutting"): EventTypeMetadata[] {
        return Array.from(this.registry.values()).filter(
            meta => meta.tier === tier,
        );
    }
    
    getAllTypes(): string[] {
        return Array.from(this.registry.keys());
    }
    
    private registerDefaultTypes(): void {
        // Tier 1: Coordination events
        Object.values(SwarmEventType).forEach(type => {
            this.register({
                type,
                category: this.getCategoryForSwarmEvent(type),
                tier: 1,
                description: this.getDescriptionForSwarmEvent(type),
            });
        });
        
        // Tier 2: Process events
        Object.values(RunEventType).forEach(type => {
            this.register({
                type,
                category: this.getCategoryForRunEvent(type),
                tier: 2,
                description: this.getDescriptionForRunEvent(type),
            });
        });
        
        // Cross-cutting events
        this.register({
            type: "SECURITY_VIOLATION",
            category: EventCategory.SECURITY,
            tier: "cross-cutting",
            description: "Security violation detected",
        });
        
        this.register({
            type: "RESOURCE_EXHAUSTED",
            category: EventCategory.RESOURCE,
            tier: "cross-cutting",
            description: "Resource limit exceeded",
        });
        
        this.register({
            type: "SYSTEM_ERROR",
            category: EventCategory.ERROR,
            tier: "cross-cutting",
            description: "System error occurred",
        });
        
        this.register({
            type: "METRIC_THRESHOLD",
            category: EventCategory.MONITORING,
            tier: "cross-cutting",
            description: "Metric threshold crossed",
        });
    }
    
    private getCategoryForSwarmEvent(type: SwarmEventType): EventCategory {
        if (type.startsWith("SWARM_")) {
            return EventCategory.LIFECYCLE;
        }
        if (type.startsWith("TEAM_")) {
            return EventCategory.TEAM_MANAGEMENT;
        }
        if (type.startsWith("GOAL_")) {
            return EventCategory.GOAL_MANAGEMENT;
        }
        if (type.startsWith("RESOURCE_")) {
            return EventCategory.RESOURCE;
        }
        return EventCategory.COORDINATION;
    }
    
    private getCategoryForRunEvent(type: RunEventType): EventCategory {
        if (type.startsWith("RUN_")) {
            return EventCategory.LIFECYCLE;
        }
        if (type.startsWith("STEP_")) {
            return EventCategory.EXECUTION;
        }
        if (type.includes("OPTIMIZATION")) {
            return EventCategory.OPTIMIZATION;
        }
        return EventCategory.PROCESS;
    }
    
    private getDescriptionForSwarmEvent(type: SwarmEventType): string {
        const descriptions: Record<SwarmEventType, string> = {
            [SwarmEventType.SWARM_STARTED]: "Swarm has been started",
            [SwarmEventType.SWARM_STOPPED]: "Swarm has been stopped",
            [SwarmEventType.SWARM_FAILED]: "Swarm has failed",
            [SwarmEventType.SWARM_COMPLETED]: "Swarm has completed its goal",
            [SwarmEventType.TEAM_FORMED]: "New team has been formed",
            [SwarmEventType.TEAM_DISBANDED]: "Team has been disbanded",
            [SwarmEventType.AGENT_JOINED]: "Agent joined the swarm",
            [SwarmEventType.AGENT_LEFT]: "Agent left the swarm",
            [SwarmEventType.GOAL_ASSIGNED]: "Goal assigned to agent/team",
            [SwarmEventType.GOAL_COMPLETED]: "Goal has been completed",
            [SwarmEventType.GOAL_FAILED]: "Goal execution failed",
            [SwarmEventType.SUBTASK_CREATED]: "New subtask created",
            [SwarmEventType.RESOURCE_ALLOCATED]: "Resource allocated to consumer",
            [SwarmEventType.RESOURCE_RELEASED]: "Resource released by consumer",
            [SwarmEventType.RESOURCE_EXHAUSTED]: "Resource has been exhausted",
            [SwarmEventType.MESSAGE_SENT]: "Message sent between agents",
            [SwarmEventType.CONSENSUS_REACHED]: "Consensus reached by agents",
            [SwarmEventType.CONFLICT_DETECTED]: "Conflict detected between agents",
        };
        return descriptions[type] || "Unknown event";
    }
    
    private getDescriptionForRunEvent(type: RunEventType): string {
        const descriptions: Record<RunEventType, string> = {
            [RunEventType.RUN_STARTED]: "Run has been started",
            [RunEventType.RUN_PAUSED]: "Run has been paused",
            [RunEventType.RUN_RESUMED]: "Run has been resumed",
            [RunEventType.RUN_COMPLETED]: "Run completed successfully",
            [RunEventType.RUN_FAILED]: "Run failed with error",
            [RunEventType.RUN_CANCELLED]: "Run was cancelled",
            [RunEventType.STEP_STARTED]: "Step execution started",
            [RunEventType.STEP_COMPLETED]: "Step completed successfully",
            [RunEventType.STEP_FAILED]: "Step failed with error",
            [RunEventType.STEP_SKIPPED]: "Step was skipped",
            [RunEventType.BRANCH_CREATED]: "New execution branch created",
            [RunEventType.BRANCH_COMPLETED]: "Branch completed execution",
            [RunEventType.BRANCH_FAILED]: "Branch failed with error",
            [RunEventType.CONTEXT_UPDATED]: "Execution context updated",
            [RunEventType.VARIABLE_SET]: "Variable set in context",
            [RunEventType.CHECKPOINT_CREATED]: "Checkpoint created for recovery",
            [RunEventType.BOTTLENECK_DETECTED]: "Performance bottleneck detected",
            [RunEventType.OPTIMIZATION_APPLIED]: "Optimization applied to execution",
        };
        return descriptions[type] || "Unknown event";
    }
}

/**
 * Event type validation
 */
export class EventTypeValidator {
    private registry = EventTypeRegistry.getInstance();
    
    isValidType(type: string): boolean {
        return this.registry.get(type) !== undefined;
    }
    
    validateEvent(event: { type: string; source?: { tier: any } }): string[] {
        const errors: string[] = [];
        
        if (!event.type) {
            errors.push("Event type is required");
            return errors;
        }
        
        const metadata = this.registry.get(event.type);
        if (!metadata) {
            errors.push(`Unknown event type: ${event.type}`);
            return errors;
        }
        
        // Validate tier consistency
        if (event.source?.tier && event.source.tier !== metadata.tier) {
            errors.push(
                `Event type ${event.type} belongs to tier ${metadata.tier}, ` +
                `but source tier is ${event.source.tier}`,
            );
        }
        
        return errors;
    }
}
