/**
 * Emergent Capabilities Fixtures
 * 
 * Cross-tier emergent behaviors where capabilities emerge from specialized agents
 * rather than being built-in features.
 * 
 * Enhanced with comprehensive validation utilities and factory-based creation.
 */

// Core validation utilities
export * from "./emergentValidationUtils.js";

// Factory implementations
export * from "./EmergentFixtureFactory.js";

// Agent type configurations
export * from "./agent-types/emergentAgentFixtures.js";
export * from "./agent-types/advancedAgentPatterns.js";
export * from "./agent-types/eventDrivenAgentExamples.js";

// Event-driven agent fixtures
export * from "./eventDrivenAgentFixtures.js";

// Evolution examples
export * from "./evolution-examples/emergentBehaviorExamples.js";

// Self-improvement patterns
export * from "./self-improvement/eventPatternLearning.js";

/**
 * Key Emergent Capabilities:
 * 
 * Agent Types:
 * - Security Agents: Domain-specific threat detection and compliance
 * - Quality Agents: Output validation and bias detection
 * - Optimization Agents: Performance enhancement and cost reduction
 * - Monitoring Agents: Intelligent observability and predictive analytics
 * 
 * Evolution Process:
 * 1. Pattern Recognition: Agents monitor routine execution events
 * 2. Optimization Identification: Agents analyze patterns and identify improvements
 * 3. Proposal Generation: Agents create improved routine versions
 * 4. Collaborative Review: Multiple agents review proposals
 * 5. Deployment: Approved optimizations are deployed and monitored
 * 
 * Compound Intelligence Effect:
 * - Every improvement compounds system capabilities
 * - Agents learn from each optimization outcome
 * - Successful patterns spread across teams
 * - Domain-specific expertise develops naturally
 */

// New example fixtures demonstrating true emergence
export * from "./researchSwarmExample.js";
export * from "./evolvingRoutineExample.js";
export * from "./crossTierEmergenceExample.js";

// Example evolution metrics
export const evolutionMetrics = {
    customerSupport: {
        v1_conversational: { duration: "45s", cost: "$0.12", accuracy: "92%" },
        v2_reasoning: { duration: "15s", cost: "$0.08", accuracy: "95%" },
        v3_deterministic: { duration: "2s", cost: "$0.02", accuracy: "99%" },
        improvement: {
            speedGain: "22.5x",
            costReduction: "83%",
            qualityIncrease: "8%",
        },
    },
    securityScanning: {
        v1_conversational: { duration: "120s", cost: "$0.45", coverage: "85%" },
        v2_reasoning: { duration: "30s", cost: "$0.15", coverage: "92%" },
        v3_deterministic: { duration: "5s", cost: "$0.03", coverage: "98%" },
        improvement: {
            speedGain: "24x",
            costReduction: "93%",
            coverageIncrease: "15%",
        },
    },
};
