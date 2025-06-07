/**
 * Monitoring Intelligence Examples - Index
 * 
 * This module demonstrates how monitoring intelligence emerges from swarm 
 * configurations rather than being hardcoded. It showcases the "two-lens" 
 * monitoring philosophy where specialized monitoring capabilities develop 
 * naturally through swarm intelligence and collaboration.
 * 
 * @fileoverview Entry point for emergent monitoring examples
 */

// Swarm Configurations
export {
    createPerformanceMonitorSwarm,
    PERFORMANCE_MONITOR_SWARM_CONFIG,
    PERFORMANCE_MONITOR_ROUTINES,
    type PerformanceMonitorSwarmConfig
} from "./swarms/performanceMonitorSwarm.js";

export {
    createSLOGuardianSwarm,
    SLO_GUARDIAN_SWARM_CONFIG,
    SLO_GUARDIAN_ROUTINES,
    SLO_GUARDIAN_INTEGRATIONS,
    type SLOGuardianSwarmConfig
} from "./swarms/sloGuardianSwarm.js";

export {
    createPatternAnalystSwarm,
    PATTERN_ANALYST_SWARM_CONFIG,
    PATTERN_ANALYST_ROUTINES,
    PATTERN_ANALYST_SPECIALIZATIONS,
    type PatternAnalystSwarmConfig
} from "./swarms/patternAnalystSwarm.js";

export {
    createResourceOptimizerSwarm,
    RESOURCE_OPTIMIZER_SWARM_CONFIG,
    RESOURCE_OPTIMIZER_ROUTINES,
    RESOURCE_OPTIMIZER_INTELLIGENCE_PATTERNS,
    type ResourceOptimizerSwarmConfig
} from "./swarms/resourceOptimizerSwarm.js";

// Integration and Collaboration
export {
    createMonitoringSwarmNetwork,
    CollaborativeMonitoringExamples,
    demonstrateSwarmCollaboration,
    type SwarmCollaborationNetwork,
    type CollaborationPattern,
    type DataExchangePattern,
    type EmergentCapability,
    type CoordinationMechanism
} from "./integration/swarmCollaboration.js";

// Types and Interfaces
import { type Logger } from "winston";
import { type SessionUser } from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";

/**
 * Monitoring Intelligence Configuration
 * 
 * Defines the overall configuration for emergent monitoring intelligence
 */
export interface MonitoringIntelligenceConfig {
    network: {
        scope: "comprehensive" | "focused" | "specialized";
        collaboration_intensity: "minimal" | "moderate" | "intensive";
        intelligence_level: "adaptive" | "predictive" | "autonomous";
    };
    business_context?: {
        industry: string;
        criticality: "low" | "medium" | "high" | "critical";
        compliance_requirements: string[];
        performance_sensitivity: "low" | "medium" | "high";
        cost_sensitivity: "low" | "medium" | "high";
    };
    technical_context?: {
        environment: "development" | "staging" | "production";
        scale: "small" | "medium" | "large" | "enterprise";
        complexity: "simple" | "moderate" | "complex" | "advanced";
        real_time_requirements: boolean;
    };
    learning_preferences?: {
        adaptation_speed: "conservative" | "moderate" | "aggressive";
        confidence_requirements: "low" | "medium" | "high";
        autonomy_level: "guided" | "semi_autonomous" | "fully_autonomous";
        collaboration_preference: "independent" | "cooperative" | "collaborative";
    };
}

/**
 * Emergent Monitoring Intelligence Factory
 * 
 * Creates a comprehensive monitoring intelligence system based on configuration
 */
export class EmergentMonitoringIntelligence {
    private readonly user: SessionUser;
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly config: MonitoringIntelligenceConfig;

    constructor(
        user: SessionUser,
        logger: Logger,
        eventBus: EventBus,
        config: MonitoringIntelligenceConfig
    ) {
        this.user = user;
        this.logger = logger;
        this.eventBus = eventBus;
        this.config = config;
    }

    /**
     * Initialize the complete monitoring intelligence system
     */
    async initialize(): Promise<SwarmCollaborationNetwork> {
        this.logger.info("[EmergentMonitoring] Initializing monitoring intelligence system", {
            config: this.config
        });

        try {
            // Create the collaborative monitoring network
            const network = await createMonitoringSwarmNetwork(
                this.user,
                this.logger,
                this.eventBus,
                this.config.network
            );

            // Apply business context adaptations
            if (this.config.business_context) {
                await this.applyBusinessContextAdaptations(network);
            }

            // Apply technical context optimizations
            if (this.config.technical_context) {
                await this.applyTechnicalContextOptimizations(network);
            }

            // Configure learning preferences
            if (this.config.learning_preferences) {
                await this.applyLearningPreferences(network);
            }

            this.logger.info("[EmergentMonitoring] Monitoring intelligence system initialized successfully", {
                swarms: Object.keys(network.swarms),
                emergentCapabilities: network.emergentCapabilities.length,
                collaborationPatterns: network.collaborationPatterns.length
            });

            return network;
        } catch (error) {
            this.logger.error("[EmergentMonitoring] Failed to initialize monitoring intelligence", error);
            throw error;
        }
    }

    /**
     * Demonstrate the monitoring intelligence capabilities
     */
    async demonstrateCapabilities(network: SwarmCollaborationNetwork): Promise<void> {
        this.logger.info("[EmergentMonitoring] Demonstrating monitoring intelligence capabilities");

        const examples = new CollaborativeMonitoringExamples(network, this.logger);

        try {
            // Demonstrate core emergent capabilities
            await examples.demonstratePredictiveIncidentPrevention();
            await examples.demonstrateAutonomousOptimization();
            await examples.demonstrateContextualAlerting();

            this.logger.info("[EmergentMonitoring] All capability demonstrations completed successfully");
        } catch (error) {
            this.logger.error("[EmergentMonitoring] Capability demonstration failed", error);
            throw error;
        }
    }

    /**
     * Apply business context adaptations to the monitoring network
     */
    private async applyBusinessContextAdaptations(network: SwarmCollaborationNetwork): Promise<void> {
        const context = this.config.business_context!;

        // Adjust SLO targets based on business criticality
        if (context.criticality === "critical") {
            network.swarms.slo.sloDefinitions.availability.target = 0.9999; // 99.99%
            network.swarms.slo.errorBudgetManagement.burn_rate_alerting.fast_burn = 6.0;
        }

        // Industry-specific adaptations
        if (context.industry === "financial") {
            network.swarms.slo.adaptiveGovernance.stakeholder_communication.escalation_matrix.push("compliance_officer");
            network.swarms.slo.quality_assurance.validation.compliance_verification = true;
        }

        // Performance sensitivity adjustments
        if (context.performance_sensitivity === "high") {
            network.swarms.performance.adaptiveThresholds.responseTime.baseline *= 0.5; // Stricter thresholds
            network.swarms.performance.emergentBehaviors.adaptiveAlerting.escalationLevels = [0.5, 1, 2, 5];
        }

        this.logger.info("[EmergentMonitoring] Applied business context adaptations", context);
    }

    /**
     * Apply technical context optimizations to the monitoring network
     */
    private async applyTechnicalContextOptimizations(network: SwarmCollaborationNetwork): Promise<void> {
        const context = this.config.technical_context!;

        // Environment-specific adjustments
        if (context.environment === "production") {
            // Increase monitoring intensity and reduce tolerance
            Object.values(network.swarms).forEach(swarm => {
                swarm.quality_assurance.self_monitoring.review_interval = Math.floor(
                    swarm.quality_assurance.self_monitoring.review_interval / 2
                );
            });
        }

        // Scale-based resource adjustments
        const scaleMultiplier = {
            small: 0.5,
            medium: 1.0,
            large: 2.0,
            enterprise: 4.0
        }[context.scale];

        Object.values(network.swarms).forEach(swarm => {
            swarm.members.forEach(member => {
                member.resources.cpu *= scaleMultiplier;
                member.resources.memory = Math.floor(member.resources.memory * scaleMultiplier);
                member.resources.credits = Math.floor(member.resources.credits * scaleMultiplier);
            });
        });

        // Real-time requirements
        if (context.real_time_requirements) {
            Object.values(network.swarms).forEach(swarm => {
                swarm.members.forEach(member => {
                    member.learning_config.feedback_window = Math.floor(member.learning_config.feedback_window / 2);
                    member.learning_config.adaptation_rate *= 1.5;
                });
            });
        }

        this.logger.info("[EmergentMonitoring] Applied technical context optimizations", context);
    }

    /**
     * Apply learning preferences to the monitoring network
     */
    private async applyLearningPreferences(network: SwarmCollaborationNetwork): Promise<void> {
        const preferences = this.config.learning_preferences!;

        // Adaptation speed adjustments
        const speedMultiplier = {
            conservative: 0.5,
            moderate: 1.0,
            aggressive: 2.0
        }[preferences.adaptation_speed];

        Object.values(network.swarms).forEach(swarm => {
            swarm.members.forEach(member => {
                member.learning_config.adaptation_rate *= speedMultiplier;
            });
        });

        // Confidence requirements
        const confidenceThreshold = {
            low: 0.6,
            medium: 0.8,
            high: 0.95
        }[preferences.confidence_requirements];

        Object.values(network.swarms).forEach(swarm => {
            swarm.members.forEach(member => {
                member.learning_config.confidence_threshold = confidenceThreshold;
            });
        });

        // Autonomy level adjustments
        if (preferences.autonomy_level === "fully_autonomous") {
            Object.values(network.swarms).forEach(swarm => {
                swarm.members.forEach(member => {
                    if (member.autonomy_level === "guided" || member.autonomy_level === "semi_autonomous") {
                        member.autonomy_level = "adaptive";
                    }
                });
            });
        }

        this.logger.info("[EmergentMonitoring] Applied learning preferences", preferences);
    }
}

/**
 * Convenience function to create and initialize monitoring intelligence
 */
export async function createEmergentMonitoringIntelligence(
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus,
    config: MonitoringIntelligenceConfig
): Promise<{
    intelligence: EmergentMonitoringIntelligence;
    network: SwarmCollaborationNetwork;
}> {
    const intelligence = new EmergentMonitoringIntelligence(user, logger, eventBus, config);
    const network = await intelligence.initialize();

    return { intelligence, network };
}

/**
 * Example configurations for different use cases
 */
export const EXAMPLE_CONFIGURATIONS = {
    // Startup configuration - resource-conscious, moderate intelligence
    startup: {
        network: {
            scope: "focused" as const,
            collaboration_intensity: "moderate" as const,
            intelligence_level: "adaptive" as const
        },
        business_context: {
            industry: "technology",
            criticality: "medium" as const,
            compliance_requirements: ["basic"],
            performance_sensitivity: "medium" as const,
            cost_sensitivity: "high" as const
        },
        technical_context: {
            environment: "production" as const,
            scale: "small" as const,
            complexity: "moderate" as const,
            real_time_requirements: false
        },
        learning_preferences: {
            adaptation_speed: "moderate" as const,
            confidence_requirements: "medium" as const,
            autonomy_level: "semi_autonomous" as const,
            collaboration_preference: "cooperative" as const
        }
    },

    // Enterprise configuration - comprehensive monitoring, high intelligence
    enterprise: {
        network: {
            scope: "comprehensive" as const,
            collaboration_intensity: "intensive" as const,
            intelligence_level: "autonomous" as const
        },
        business_context: {
            industry: "financial",
            criticality: "critical" as const,
            compliance_requirements: ["financial", "security", "privacy"],
            performance_sensitivity: "high" as const,
            cost_sensitivity: "medium" as const
        },
        technical_context: {
            environment: "production" as const,
            scale: "enterprise" as const,
            complexity: "advanced" as const,
            real_time_requirements: true
        },
        learning_preferences: {
            adaptation_speed: "conservative" as const,
            confidence_requirements: "high" as const,
            autonomy_level: "fully_autonomous" as const,
            collaboration_preference: "collaborative" as const
        }
    },

    // Development configuration - experimental, learning-focused
    development: {
        network: {
            scope: "specialized" as const,
            collaboration_intensity: "minimal" as const,
            intelligence_level: "predictive" as const
        },
        business_context: {
            industry: "technology",
            criticality: "low" as const,
            compliance_requirements: [],
            performance_sensitivity: "low" as const,
            cost_sensitivity: "high" as const
        },
        technical_context: {
            environment: "development" as const,
            scale: "small" as const,
            complexity: "simple" as const,
            real_time_requirements: false
        },
        learning_preferences: {
            adaptation_speed: "aggressive" as const,
            confidence_requirements: "low" as const,
            autonomy_level: "guided" as const,
            collaboration_preference: "independent" as const
        }
    }
} as const;

/**
 * Quick setup functions for common scenarios
 */
export async function setupStartupMonitoring(
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus
): Promise<SwarmCollaborationNetwork> {
    const { network } = await createEmergentMonitoringIntelligence(
        user,
        logger,
        eventBus,
        EXAMPLE_CONFIGURATIONS.startup
    );
    return network;
}

export async function setupEnterpriseMonitoring(
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus
): Promise<SwarmCollaborationNetwork> {
    const { network } = await createEmergentMonitoringIntelligence(
        user,
        logger,
        eventBus,
        EXAMPLE_CONFIGURATIONS.enterprise
    );
    return network;
}

export async function setupDevelopmentMonitoring(
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus
): Promise<SwarmCollaborationNetwork> {
    const { network } = await createEmergentMonitoringIntelligence(
        user,
        logger,
        eventBus,
        EXAMPLE_CONFIGURATIONS.development
    );
    return network;
}