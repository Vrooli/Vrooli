/**
 * Cross-cutting Emergent Agent Infrastructure
 * 
 * This module provides the infrastructure for deploying goal-driven agents
 * that learn from events and propose routine improvements. Unlike hard-coded
 * agent types, these agents develop specialized capabilities through experience.
 */

export { EmergentAgent } from "./emergentAgent.js";
export { AgentDeploymentService } from "./agentDeploymentService.js";

export type {
    AgentInsights,
} from "./emergentAgent.js";

export type {
    EmergentAgentConfig,
    EmergentSwarmConfig,
} from "./agentDeploymentService.js";

export type {
    IntelligentEvent,
    AgentResponse,
    AgentSubscription,
    DeliveryGuarantee,
    EventPriority,
} from "../events/eventBus.js";

/**
 * Common event patterns for emergent agents to subscribe to
 */
export const EMERGENT_EVENT_PATTERNS = {
    // Performance monitoring patterns
    ROUTINE_COMPLETED: "routine/completed",
    STEP_COMPLETED: "step/completed", 
    EXECUTION_METRICS: "execution/metrics",
    
    // Quality monitoring patterns
    OUTPUT_GENERATED: "output/generated",
    CONTENT_CREATED: "content/created",
    VALIDATION_REQUIRED: "validation/required",
    
    // Security monitoring patterns
    SECURITY_EVENT: "security/*",
    THREAT_DETECTED: "security/threat/*",
    COMPLIANCE_CHECK: "security/compliance/*",
    
    // Error and warning patterns
    ERROR_OCCURRED: "error/*",
    WARNING_ISSUED: "warning/*",
    ALERT_RAISED: "alert/*",
    
    // Tool execution patterns
    TOOL_CALLED: "tool/called/*",
    TOOL_COMPLETED: "tool/completed/*",
    TOOL_FAILED: "tool/failed/*",
    
    // Tier-specific patterns
    TIER1_EVENTS: "tier1/*",
    TIER2_EVENTS: "tier2/*",
    TIER3_EVENTS: "tier3/*",
} as const;

/**
 * Example agent deployment configurations for common emergent capabilities
 */
export const EMERGENT_AGENT_EXAMPLES = {
    /**
     * Performance monitoring agent that learns to identify optimization opportunities
     */
    PERFORMANCE_MONITOR: {
        goal: "Monitor API response times and identify bottlenecks",
        initialRoutine: "analyze_performance_patterns",
        subscriptions: [
            EMERGENT_EVENT_PATTERNS.ROUTINE_COMPLETED,
            EMERGENT_EVENT_PATTERNS.EXECUTION_METRICS,
            EMERGENT_EVENT_PATTERNS.STEP_COMPLETED,
        ],
        priority: 7,
    },
    
    /**
     * Quality assurance agent that learns to detect output quality issues
     */
    QUALITY_MONITOR: {
        goal: "Monitor output quality and detect accuracy issues",
        initialRoutine: "assess_output_quality",
        subscriptions: [
            EMERGENT_EVENT_PATTERNS.OUTPUT_GENERATED,
            EMERGENT_EVENT_PATTERNS.CONTENT_CREATED,
            EMERGENT_EVENT_PATTERNS.VALIDATION_REQUIRED,
        ],
        priority: 8,
    },
    
    /**
     * Security monitoring agent that learns threat patterns
     */
    SECURITY_MONITOR: {
        goal: "Monitor security events and detect threat patterns",
        initialRoutine: "analyze_security_patterns",
        subscriptions: [
            EMERGENT_EVENT_PATTERNS.SECURITY_EVENT,
            EMERGENT_EVENT_PATTERNS.TOOL_CALLED,
            EMERGENT_EVENT_PATTERNS.ERROR_OCCURRED,
        ],
        priority: 9,
    },
    
    /**
     * Cost optimization agent that learns to reduce operational costs
     */
    COST_OPTIMIZER: {
        goal: "Monitor costs and identify cost reduction opportunities",
        initialRoutine: "analyze_cost_patterns",
        subscriptions: [
            EMERGENT_EVENT_PATTERNS.ROUTINE_COMPLETED,
            EMERGENT_EVENT_PATTERNS.TOOL_COMPLETED,
            EMERGENT_EVENT_PATTERNS.EXECUTION_METRICS,
        ],
        priority: 6,
    },
    
    /**
     * Error pattern recognition agent
     */
    ERROR_ANALYZER: {
        goal: "Analyze error patterns and suggest prevention strategies",
        initialRoutine: "analyze_error_patterns",
        subscriptions: [
            EMERGENT_EVENT_PATTERNS.ERROR_OCCURRED,
            EMERGENT_EVENT_PATTERNS.TOOL_FAILED,
            EMERGENT_EVENT_PATTERNS.WARNING_ISSUED,
        ],
        priority: 8,
    },
} as const;

/**
 * Pre-configured swarm templates for common emergent capabilities
 */
export const EMERGENT_SWARM_TEMPLATES = {
    /**
     * Comprehensive monitoring swarm
     */
    MONITORING_SWARM: {
        name: "Comprehensive Monitoring Swarm",
        description: "Multi-domain monitoring with emergent optimization capabilities",
        agents: [
            {
                ...EMERGENT_AGENT_EXAMPLES.PERFORMANCE_MONITOR,
                agentId: "performance_monitor",
            },
            {
                ...EMERGENT_AGENT_EXAMPLES.QUALITY_MONITOR,
                agentId: "quality_monitor",
            },
            {
                ...EMERGENT_AGENT_EXAMPLES.SECURITY_MONITOR,
                agentId: "security_monitor",
            },
        ],
        coordination: {
            sharedLearning: true,
            collaborativeProposals: true,
            crossAgentInsights: true,
        },
    },
    
    /**
     * Optimization-focused swarm
     */
    OPTIMIZATION_SWARM: {
        name: "Optimization Swarm",
        description: "Performance and cost optimization through emergent analysis",
        agents: [
            {
                ...EMERGENT_AGENT_EXAMPLES.PERFORMANCE_MONITOR,
                agentId: "performance_optimizer",
            },
            {
                ...EMERGENT_AGENT_EXAMPLES.COST_OPTIMIZER,
                agentId: "cost_optimizer",
            },
            {
                ...EMERGENT_AGENT_EXAMPLES.ERROR_ANALYZER,
                agentId: "error_preventer",
            },
        ],
        coordination: {
            sharedLearning: true,
            collaborativeProposals: true,
            crossAgentInsights: true,
        },
    },
    
    /**
     * Security and compliance swarm
     */
    SECURITY_SWARM: {
        name: "Security & Compliance Swarm",
        description: "Adaptive security monitoring and compliance assurance",
        agents: [
            {
                ...EMERGENT_AGENT_EXAMPLES.SECURITY_MONITOR,
                agentId: "security_analyst",
            },
            {
                ...EMERGENT_AGENT_EXAMPLES.QUALITY_MONITOR,
                agentId: "compliance_monitor",
                goal: "Monitor compliance requirements and detect violations",
                initialRoutine: "assess_compliance_status",
            },
        ],
        coordination: {
            sharedLearning: true,
            collaborativeProposals: false, // Security decisions shouldn't be collaborative
            crossAgentInsights: true,
        },
    },
} as const;

/**
 * Utility functions for emergent agent deployment
 */
export class EmergentAgentUtils {
    /**
     * Create agent configuration from example template
     */
    static createAgentConfig(
        agentId: string,
        template: typeof EMERGENT_AGENT_EXAMPLES[keyof typeof EMERGENT_AGENT_EXAMPLES],
        overrides?: Partial<EmergentAgentConfig>,
    ): EmergentAgentConfig {
        return {
            agentId,
            ...template,
            ...overrides,
        };
    }
    
    /**
     * Create swarm configuration from template
     */
    static createSwarmConfig(
        swarmId: string,
        template: typeof EMERGENT_SWARM_TEMPLATES[keyof typeof EMERGENT_SWARM_TEMPLATES],
        overrides?: Partial<EmergentSwarmConfig>,
    ): EmergentSwarmConfig {
        return {
            swarmId,
            ...template,
            agents: template.agents.map(agent => ({
                ...agent,
                agentId: `${swarmId}_${agent.agentId}`,
            })),
            ...overrides,
        };
    }
    
    /**
     * Get recommended event patterns for a goal
     */
    static getRecommendedPatterns(goal: string): string[] {
        const patterns: string[] = [];
        const goalLower = goal.toLowerCase();
        
        // Performance-related goals
        if (goalLower.includes("performance") || goalLower.includes("optimize") || goalLower.includes("speed")) {
            patterns.push(
                EMERGENT_EVENT_PATTERNS.ROUTINE_COMPLETED,
                EMERGENT_EVENT_PATTERNS.EXECUTION_METRICS,
                EMERGENT_EVENT_PATTERNS.STEP_COMPLETED,
            );
        }
        
        // Quality-related goals
        if (goalLower.includes("quality") || goalLower.includes("accuracy") || goalLower.includes("validation")) {
            patterns.push(
                EMERGENT_EVENT_PATTERNS.OUTPUT_GENERATED,
                EMERGENT_EVENT_PATTERNS.CONTENT_CREATED,
                EMERGENT_EVENT_PATTERNS.VALIDATION_REQUIRED,
            );
        }
        
        // Security-related goals
        if (goalLower.includes("security") || goalLower.includes("threat") || goalLower.includes("compliance")) {
            patterns.push(
                EMERGENT_EVENT_PATTERNS.SECURITY_EVENT,
                EMERGENT_EVENT_PATTERNS.TOOL_CALLED,
                EMERGENT_EVENT_PATTERNS.ERROR_OCCURRED,
            );
        }
        
        // Cost-related goals
        if (goalLower.includes("cost") || goalLower.includes("expense") || goalLower.includes("budget")) {
            patterns.push(
                EMERGENT_EVENT_PATTERNS.ROUTINE_COMPLETED,
                EMERGENT_EVENT_PATTERNS.TOOL_COMPLETED,
                EMERGENT_EVENT_PATTERNS.EXECUTION_METRICS,
            );
        }
        
        // Error-related goals
        if (goalLower.includes("error") || goalLower.includes("failure") || goalLower.includes("reliability")) {
            patterns.push(
                EMERGENT_EVENT_PATTERNS.ERROR_OCCURRED,
                EMERGENT_EVENT_PATTERNS.TOOL_FAILED,
                EMERGENT_EVENT_PATTERNS.WARNING_ISSUED,
            );
        }
        
        // General monitoring
        if (goalLower.includes("monitor") || goalLower.includes("observe") || goalLower.includes("track")) {
            patterns.push(
                EMERGENT_EVENT_PATTERNS.ROUTINE_COMPLETED,
                EMERGENT_EVENT_PATTERNS.EXECUTION_METRICS,
                EMERGENT_EVENT_PATTERNS.ERROR_OCCURRED,
            );
        }
        
        return [...new Set(patterns)]; // Remove duplicates
    }
    
    /**
     * Calculate priority based on goal and event type
     */
    static calculatePriority(goal: string, eventType?: string): number {
        let priority = 5; // Base priority
        
        const goalLower = goal.toLowerCase();
        
        // Security goals get higher priority
        if (goalLower.includes("security") || goalLower.includes("threat")) {
            priority += 3;
        }
        
        // Quality goals get medium-high priority
        if (goalLower.includes("quality") || goalLower.includes("compliance")) {
            priority += 2;
        }
        
        // Performance goals get medium priority
        if (goalLower.includes("performance") || goalLower.includes("optimize")) {
            priority += 1;
        }
        
        // Event-specific adjustments
        if (eventType) {
            if (eventType.includes("security") && goalLower.includes("security")) {
                priority += 1;
            }
            if (eventType.includes("error") && goalLower.includes("error")) {
                priority += 1;
            }
        }
        
        return Math.min(10, priority);
    }
    
    /**
     * Validate agent configuration
     */
    static validateAgentConfig(config: EmergentAgentConfig): string[] {
        const errors: string[] = [];
        
        if (!config.agentId) {
            errors.push("Agent ID is required");
        }
        
        if (!config.goal) {
            errors.push("Agent goal is required");
        }
        
        if (!config.initialRoutine) {
            errors.push("Initial routine is required");
        }
        
        if (!config.subscriptions || config.subscriptions.length === 0) {
            errors.push("At least one event subscription is required");
        }
        
        if (config.priority !== undefined && (config.priority < 1 || config.priority > 10)) {
            errors.push("Priority must be between 1 and 10");
        }
        
        return errors;
    }
}

/**
 * Re-export types for convenience
 */
import type { EmergentAgentConfig, EmergentSwarmConfig } from "./agentDeploymentService.js";