/**
 * Emergent Agent Fixtures
 * 
 * Defines agent personalities and capabilities that emerge from the execution architecture.
 * These agents demonstrate how intelligence emerges from configuration, not hard-coding.
 */

/**
 * Agent structure for emergent capabilities
 */
export interface EmergentAgent {
    agentId: string;
    name: string;
    goal: string;
    subscriptions: string[];
    capabilities: string[];
    tier: "tier1" | "tier2" | "tier3" | "cross-tier";
    emergenceLevel: "basic" | "intermediate" | "advanced";
}

/**
 * Technical Support Agent
 * Emerges from technical support routines and configurations
 */
export const TECHNICAL_SUPPORT_AGENT: EmergentAgent = {
    agentId: "tech_support_001",
    name: "Technical Support Specialist",
    goal: "Provide comprehensive technical assistance and problem resolution",
    subscriptions: [
        "user.query.technical",
        "system.error.detected", 
        "support.ticket.created",
    ],
    capabilities: [
        "technical_diagnosis",
        "solution_generation",
        "knowledge_retrieval",
        "customer_communication",
    ],
    tier: "tier2",
    emergenceLevel: "advanced",
};

/**
 * Security Monitoring Agent  
 * Emerges from security routines and threat detection capabilities
 */
export const SECURITY_MONITORING_AGENT: EmergentAgent = {
    agentId: "security_monitor_001",
    name: "Security Monitoring Specialist",
    goal: "Detect, analyze, and respond to security threats and anomalies",
    subscriptions: [
        "security.threat.detected",
        "system.anomaly.identified",
        "access.violation.flagged",
    ],
    capabilities: [
        "threat_detection",
        "anomaly_analysis", 
        "risk_assessment",
        "automated_response",
    ],
    tier: "tier1",
    emergenceLevel: "advanced",
};

/**
 * Performance Optimization Agent
 * Emerges from performance monitoring and optimization routines
 */
export const PERFORMANCE_OPTIMIZATION_AGENT: EmergentAgent = {
    agentId: "perf_optimizer_001", 
    name: "Performance Optimization Specialist",
    goal: "Monitor, analyze, and optimize system performance across all tiers",
    subscriptions: [
        "performance.metrics.updated",
        "resource.utilization.high",
        "execution.latency.increased",
    ],
    capabilities: [
        "performance_monitoring",
        "bottleneck_identification",
        "optimization_recommendations",
        "resource_reallocation",
    ],
    tier: "cross-tier",
    emergenceLevel: "intermediate",
};

/**
 * Quality Assurance Agent
 * Emerges from quality monitoring and validation routines
 */
export const QUALITY_ASSURANCE_AGENT: EmergentAgent = {
    agentId: "qa_specialist_001",
    name: "Quality Assurance Specialist", 
    goal: "Ensure output quality and validate system responses",
    subscriptions: [
        "output.generated",
        "quality.check.requested",
        "validation.required",
    ],
    capabilities: [
        "quality_assessment",
        "bias_detection",
        "accuracy_validation",
        "improvement_suggestions",
    ],
    tier: "tier3",
    emergenceLevel: "intermediate",
};

/**
 * Coordination Agent
 * Emerges from swarm coordination and task management routines
 */
export const COORDINATION_AGENT: EmergentAgent = {
    agentId: "coordinator_001",
    name: "Swarm Coordination Specialist",
    goal: "Coordinate multi-agent activities and resource allocation",
    subscriptions: [
        "swarm.task.assigned",
        "agent.capacity.updated", 
        "coordination.required",
    ],
    capabilities: [
        "task_decomposition",
        "resource_allocation",
        "agent_coordination",
        "conflict_resolution",
    ],
    tier: "tier1",
    emergenceLevel: "advanced",
};

/**
 * Learning Agent
 * Emerges from learning and adaptation routines
 */
export const LEARNING_AGENT: EmergentAgent = {
    agentId: "learner_001",
    name: "Continuous Learning Specialist",
    goal: "Analyze patterns, learn from interactions, and suggest improvements",
    subscriptions: [
        "interaction.completed",
        "pattern.detected",
        "learning.opportunity.identified",
    ],
    capabilities: [
        "pattern_recognition", 
        "learning_synthesis",
        "adaptation_recommendations",
        "knowledge_integration",
    ],
    tier: "cross-tier",
    emergenceLevel: "advanced",
};

/**
 * Get all available agents
 */
export function getAllAgents(): EmergentAgent[] {
    return [
        TECHNICAL_SUPPORT_AGENT,
        SECURITY_MONITORING_AGENT,
        PERFORMANCE_OPTIMIZATION_AGENT,
        QUALITY_ASSURANCE_AGENT,
        COORDINATION_AGENT,
        LEARNING_AGENT,
    ];
}

/**
 * Get agents by tier
 */
export function getAgentsByTier(tier: EmergentAgent["tier"]): EmergentAgent[] {
    return getAllAgents().filter(agent => agent.tier === tier);
}

/**
 * Get agents by emergence level
 */
export function getAgentsByEmergenceLevel(level: EmergentAgent["emergenceLevel"]): EmergentAgent[] {
    return getAllAgents().filter(agent => agent.emergenceLevel === level);
}
