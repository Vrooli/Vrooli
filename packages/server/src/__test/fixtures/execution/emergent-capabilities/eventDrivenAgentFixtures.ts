/**
 * Event-Driven Agent Test Fixtures
 * 
 * Comprehensive test fixtures for EventDrivenAgent components, providing
 * validated configurations for testing emergent agent behaviors, event
 * processing patterns, and goal-driven intelligence.
 */

import type {
    BaseConfigObject,
    BotConfigObject,
    ChatConfigObject,
} from "@vrooli/shared";
import {
    botConfigFixtures,
    chatConfigFixtures,
} from "@vrooli/shared";
import type {
    ExecutionFixture,
    EmergenceDefinition,
    IntegrationDefinition,
} from "../types.js";
import { BaseExecutionFixtureFactory } from "../executionFactories.js";

/**
 * Agent goal categories
 */
export type AgentGoalCategory = "security" | "performance" | "quality" | "cost" | "compliance" | "user-experience";

/**
 * Agent learning strategies
 */
export type LearningStrategy = "reinforcement" | "supervised" | "unsupervised" | "transfer" | "meta";

/**
 * Event-driven agent specific fixture interface
 */
export interface EventDrivenAgentFixture<TConfig extends BaseConfigObject = BotConfigObject> extends ExecutionFixture<TConfig> {
    agent: {
        /** Agent identity and purpose */
        identity: {
            id: string;
            name: string;
            goal: string;
            goalCategory: AgentGoalCategory;
            description: string;
            version: string;
        };
        /** Event patterns the agent monitors */
        eventPatterns: Array<{
            pattern: string;
            relevance: number;
            actions: string[];
            learningEnabled: boolean;
        }>;
        /** Learning configuration */
        learning: {
            strategy: LearningStrategy;
            enabled: boolean;
            modelType?: string;
            updateFrequency: number;
            minDataPoints: number;
            confidenceThreshold: number;
        };
        /** Routine improvement capabilities */
        routineImprovement: {
            enabled: boolean;
            targetRoutines: string[];
            improvementTypes: Array<"optimization" | "quality" | "security" | "feature">;
            proposalThreshold: number;
            maxProposalsPerDay: number;
        };
        /** Collaboration with other agents */
        collaboration: {
            collaborators: string[];
            sharedGoals: string[];
            communicationProtocol: "direct" | "broadcast" | "pub-sub";
            consensusRequired: boolean;
        };
        /** Performance expectations */
        performance: {
            eventProcessingRate: number;
            averageResponseTime: number;
            learningRate: number;
            improvementSuccess: number;
        };
    };
}

/**
 * Factory for creating event-driven agent test fixtures
 */
export class EventDrivenAgentFixtureFactory extends BaseExecutionFixtureFactory<BotConfigObject> {
    protected ConfigClass = botConfigFixtures.BotConfig;
    protected tier = "tier1" as const;

    createMinimal(overrides?: Partial<EventDrivenAgentFixture>): EventDrivenAgentFixture {
        return {
            config: botConfigFixtures.minimal,
            emergence: {
                capabilities: ["event_processing", "basic_learning"],
                evolutionPath: "reactive → adaptive → intelligent",
                ...overrides?.emergence,
            },
            integration: {
                tier: "tier1",
                producedEvents: ["agent.insight", "agent.proposal"],
                consumedEvents: ["system.*", "tier*.*"],
                ...overrides?.integration,
            },
            agent: {
                identity: {
                    id: "agent_minimal",
                    name: "Minimal Agent",
                    goal: "Monitor system events",
                    goalCategory: "quality",
                    description: "Basic event monitoring agent",
                    version: "1.0.0",
                },
                eventPatterns: [{
                    pattern: "system.error",
                    relevance: 1.0,
                    actions: ["log", "alert"],
                    learningEnabled: false,
                }],
                learning: {
                    strategy: "supervised",
                    enabled: false,
                    updateFrequency: 86400000, // Daily
                    minDataPoints: 100,
                    confidenceThreshold: 0.7,
                },
                routineImprovement: {
                    enabled: false,
                    targetRoutines: [],
                    improvementTypes: [],
                    proposalThreshold: 0.8,
                    maxProposalsPerDay: 1,
                },
                collaboration: {
                    collaborators: [],
                    sharedGoals: [],
                    communicationProtocol: "direct",
                    consensusRequired: false,
                },
                performance: {
                    eventProcessingRate: 10,
                    averageResponseTime: 1000,
                    learningRate: 0.01,
                    improvementSuccess: 0.5,
                },
                ...overrides?.agent,
            },
        };
    }

    createComplete(overrides?: Partial<EventDrivenAgentFixture>): EventDrivenAgentFixture {
        return {
            config: botConfigFixtures.complete,
            emergence: {
                capabilities: [
                    "pattern_recognition",
                    "predictive_analysis",
                    "autonomous_improvement",
                    "collaborative_learning",
                    "goal_optimization",
                    "self_evolution",
                ],
                eventPatterns: [
                    "tier*.error",
                    "performance.*",
                    "security.*",
                    "user.*",
                    "system.*",
                ],
                evolutionPath: "reactive → adaptive → predictive → autonomous → transcendent",
                emergenceConditions: {
                    minEvents: 10000,
                    requiredResources: ["event_history", "ml_models", "collaboration_network"],
                    timeframe: 604800000, // 7 days
                },
                learningMetrics: {
                    performanceImprovement: "90% accuracy in pattern detection",
                    adaptationTime: "24 hours to new event patterns",
                    innovationRate: "10 novel insights per week",
                },
                ...overrides?.emergence,
            },
            integration: {
                tier: "tier1",
                producedEvents: [
                    "agent.pattern.detected",
                    "agent.insight.generated",
                    "agent.proposal.created",
                    "agent.improvement.suggested",
                    "agent.collaboration.initiated",
                    "agent.evolution.achieved",
                ],
                consumedEvents: [
                    "tier1.swarm.*",
                    "tier2.routine.*",
                    "tier3.execution.*",
                    "system.performance.*",
                    "security.threat.*",
                    "user.feedback.*",
                ],
                sharedResources: [
                    "event_stream",
                    "knowledge_base",
                    "ml_models",
                    "collaboration_space",
                ],
                crossTierDependencies: {
                    dependsOn: ["event_bus", "ml_infrastructure", "knowledge_graph"],
                    provides: ["intelligent_monitoring", "autonomous_improvement", "predictive_insights"],
                },
                mcpTools: ["pattern_analyzer", "insight_generator", "proposal_creator"],
                ...overrides?.integration,
            },
            agent: {
                identity: {
                    id: "agent_advanced_security",
                    name: "Advanced Security Guardian",
                    goal: "Predict and prevent security threats through intelligent event analysis",
                    goalCategory: "security",
                    description: "Autonomous security agent with predictive capabilities",
                    version: "3.0.0",
                },
                eventPatterns: [
                    {
                        pattern: "security.threat.*",
                        relevance: 1.0,
                        actions: ["analyze", "predict", "prevent", "alert"],
                        learningEnabled: true,
                    },
                    {
                        pattern: "auth.failed",
                        relevance: 0.8,
                        actions: ["analyze", "correlate", "detect_pattern"],
                        learningEnabled: true,
                    },
                    {
                        pattern: "system.anomaly",
                        relevance: 0.9,
                        actions: ["investigate", "classify", "respond"],
                        learningEnabled: true,
                    },
                    {
                        pattern: "network.suspicious",
                        relevance: 0.95,
                        actions: ["quarantine", "analyze", "report"],
                        learningEnabled: true,
                    },
                    {
                        pattern: "data.access.unusual",
                        relevance: 0.85,
                        actions: ["monitor", "flag", "investigate"],
                        learningEnabled: true,
                    },
                ],
                learning: {
                    strategy: "reinforcement",
                    enabled: true,
                    modelType: "deep_q_network",
                    updateFrequency: 3600000, // Hourly
                    minDataPoints: 1000,
                    confidenceThreshold: 0.85,
                },
                routineImprovement: {
                    enabled: true,
                    targetRoutines: [
                        "security_check",
                        "threat_detection",
                        "access_control",
                        "anomaly_detection",
                    ],
                    improvementTypes: ["security", "optimization", "quality"],
                    proposalThreshold: 0.9,
                    maxProposalsPerDay: 5,
                },
                collaboration: {
                    collaborators: [
                        "performance_optimizer",
                        "quality_guardian",
                        "compliance_monitor",
                    ],
                    sharedGoals: [
                        "system_protection",
                        "threat_prevention",
                        "incident_response",
                    ],
                    communicationProtocol: "pub-sub",
                    consensusRequired: true,
                },
                performance: {
                    eventProcessingRate: 1000,
                    averageResponseTime: 50,
                    learningRate: 0.1,
                    improvementSuccess: 0.85,
                },
                ...overrides?.agent,
            },
            validation: {
                emergenceTests: [
                    "pattern_detection_accuracy",
                    "predictive_capability",
                    "autonomous_improvement",
                ],
                integrationTests: [
                    "event_processing_pipeline",
                    "cross_agent_collaboration",
                    "proposal_generation",
                ],
                evolutionTests: [
                    "learning_rate_improvement",
                    "goal_achievement_progress",
                    "capability_emergence",
                ],
                ...overrides?.validation,
            },
            metadata: {
                domain: "security",
                complexity: "complex",
                maintainer: "ai-team",
                lastUpdated: new Date().toISOString(),
                ...overrides?.metadata,
            },
        };
    }

    createWithDefaults(overrides?: Partial<EventDrivenAgentFixture>): EventDrivenAgentFixture {
        return this.createMinimal({
            agent: {
                identity: {
                    id: "agent_default",
                    name: "Default Monitor",
                    goal: "Monitor and learn from system events",
                    goalCategory: "quality",
                    description: "Standard event monitoring agent",
                    version: "1.0.0",
                },
                eventPatterns: [
                    {
                        pattern: "system.*",
                        relevance: 0.5,
                        actions: ["monitor", "log"],
                        learningEnabled: true,
                    },
                    {
                        pattern: "error.*",
                        relevance: 0.8,
                        actions: ["alert", "analyze"],
                        learningEnabled: true,
                    },
                ],
                learning: {
                    strategy: "supervised",
                    enabled: true,
                    updateFrequency: 86400000, // Daily
                    minDataPoints: 500,
                    confidenceThreshold: 0.75,
                },
                routineImprovement: {
                    enabled: true,
                    targetRoutines: ["error_handling"],
                    improvementTypes: ["quality"],
                    proposalThreshold: 0.85,
                    maxProposalsPerDay: 2,
                },
                collaboration: {
                    collaborators: [],
                    sharedGoals: ["system_stability"],
                    communicationProtocol: "direct",
                    consensusRequired: false,
                },
                performance: {
                    eventProcessingRate: 100,
                    averageResponseTime: 500,
                    learningRate: 0.05,
                    improvementSuccess: 0.7,
                },
            },
            ...overrides,
        });
    }

    createVariant(variant: string, overrides?: Partial<EventDrivenAgentFixture>): EventDrivenAgentFixture {
        const variants: Record<string, () => EventDrivenAgentFixture> = {
            performanceOptimizer: () => this.createComplete({
                agent: {
                    identity: {
                        id: "agent_performance",
                        name: "Performance Optimizer",
                        goal: "Optimize system performance through intelligent analysis",
                        goalCategory: "performance",
                        description: "Autonomous performance optimization agent",
                        version: "2.0.0",
                    },
                    eventPatterns: [
                        {
                            pattern: "performance.metric.*",
                            relevance: 1.0,
                            actions: ["analyze", "optimize", "predict"],
                            learningEnabled: true,
                        },
                        {
                            pattern: "resource.usage.*",
                            relevance: 0.9,
                            actions: ["monitor", "forecast", "recommend"],
                            learningEnabled: true,
                        },
                    ],
                    learning: {
                        strategy: "reinforcement",
                        enabled: true,
                        modelType: "gradient_boosting",
                        updateFrequency: 1800000, // 30 minutes
                        minDataPoints: 2000,
                        confidenceThreshold: 0.8,
                    },
                    routineImprovement: {
                        enabled: true,
                        targetRoutines: ["resource_allocation", "query_optimization"],
                        improvementTypes: ["optimization", "feature"],
                        proposalThreshold: 0.85,
                        maxProposalsPerDay: 10,
                    },
                    collaboration: {
                        collaborators: ["resource_manager", "cost_optimizer"],
                        sharedGoals: ["system_efficiency", "cost_reduction"],
                        communicationProtocol: "pub-sub",
                        consensusRequired: false,
                    },
                    performance: {
                        eventProcessingRate: 5000,
                        averageResponseTime: 20,
                        learningRate: 0.15,
                        improvementSuccess: 0.9,
                    },
                },
                ...overrides,
            }),

            qualityGuardian: () => this.createComplete({
                agent: {
                    identity: {
                        id: "agent_quality",
                        name: "Quality Guardian",
                        goal: "Ensure and improve system quality through continuous monitoring",
                        goalCategory: "quality",
                        description: "Autonomous quality assurance agent",
                        version: "2.5.0",
                    },
                    eventPatterns: [
                        {
                            pattern: "quality.metric.*",
                            relevance: 1.0,
                            actions: ["validate", "measure", "improve"],
                            learningEnabled: true,
                        },
                        {
                            pattern: "test.result.*",
                            relevance: 0.9,
                            actions: ["analyze", "correlate", "suggest"],
                            learningEnabled: true,
                        },
                        {
                            pattern: "user.feedback.*",
                            relevance: 0.95,
                            actions: ["process", "categorize", "prioritize"],
                            learningEnabled: true,
                        },
                    ],
                    learning: {
                        strategy: "supervised",
                        enabled: true,
                        modelType: "neural_network",
                        updateFrequency: 3600000, // Hourly
                        minDataPoints: 1500,
                        confidenceThreshold: 0.9,
                    },
                    routineImprovement: {
                        enabled: true,
                        targetRoutines: ["validation", "testing", "monitoring"],
                        improvementTypes: ["quality", "feature"],
                        proposalThreshold: 0.9,
                        maxProposalsPerDay: 7,
                    },
                    collaboration: {
                        collaborators: ["test_automation", "user_experience"],
                        sharedGoals: ["quality_assurance", "user_satisfaction"],
                        communicationProtocol: "broadcast",
                        consensusRequired: true,
                    },
                    performance: {
                        eventProcessingRate: 2000,
                        averageResponseTime: 100,
                        learningRate: 0.12,
                        improvementSuccess: 0.88,
                    },
                },
                ...overrides,
            }),

            costOptimizer: () => this.createComplete({
                agent: {
                    identity: {
                        id: "agent_cost",
                        name: "Cost Optimizer",
                        goal: "Minimize operational costs while maintaining service quality",
                        goalCategory: "cost",
                        description: "Intelligent cost optimization agent",
                        version: "1.5.0",
                    },
                    eventPatterns: [
                        {
                            pattern: "billing.usage.*",
                            relevance: 1.0,
                            actions: ["track", "analyze", "optimize"],
                            learningEnabled: true,
                        },
                        {
                            pattern: "resource.allocated.*",
                            relevance: 0.85,
                            actions: ["monitor", "predict", "recommend"],
                            learningEnabled: true,
                        },
                    ],
                    learning: {
                        strategy: "reinforcement",
                        enabled: true,
                        modelType: "policy_gradient",
                        updateFrequency: 7200000, // 2 hours
                        minDataPoints: 3000,
                        confidenceThreshold: 0.85,
                    },
                    routineImprovement: {
                        enabled: true,
                        targetRoutines: ["resource_provisioning", "workload_scheduling"],
                        improvementTypes: ["optimization", "cost"],
                        proposalThreshold: 0.8,
                        maxProposalsPerDay: 8,
                    },
                    collaboration: {
                        collaborators: ["resource_manager", "performance_optimizer"],
                        sharedGoals: ["cost_efficiency", "resource_optimization"],
                        communicationProtocol: "pub-sub",
                        consensusRequired: false,
                    },
                    performance: {
                        eventProcessingRate: 3000,
                        averageResponseTime: 75,
                        learningRate: 0.1,
                        improvementSuccess: 0.82,
                    },
                },
                ...overrides,
            }),

            metaLearner: () => this.createComplete({
                emergence: {
                    capabilities: [
                        "cross_domain_learning",
                        "transfer_learning",
                        "meta_optimization",
                        "agent_synthesis",
                    ],
                },
                agent: {
                    identity: {
                        id: "agent_meta",
                        name: "Meta Learner",
                        goal: "Learn from other agents to create new optimization strategies",
                        goalCategory: "performance",
                        description: "Meta-learning agent that improves other agents",
                        version: "4.0.0",
                    },
                    eventPatterns: [
                        {
                            pattern: "agent.*.proposal",
                            relevance: 1.0,
                            actions: ["analyze", "synthesize", "improve"],
                            learningEnabled: true,
                        },
                        {
                            pattern: "agent.*.learning",
                            relevance: 0.95,
                            actions: ["observe", "correlate", "transfer"],
                            learningEnabled: true,
                        },
                    ],
                    learning: {
                        strategy: "meta",
                        enabled: true,
                        modelType: "transformer",
                        updateFrequency: 21600000, // 6 hours
                        minDataPoints: 10000,
                        confidenceThreshold: 0.9,
                    },
                    routineImprovement: {
                        enabled: true,
                        targetRoutines: ["agent_coordination", "learning_optimization"],
                        improvementTypes: ["optimization", "feature", "quality"],
                        proposalThreshold: 0.95,
                        maxProposalsPerDay: 3,
                    },
                    collaboration: {
                        collaborators: ["all_agents"],
                        sharedGoals: ["collective_intelligence", "system_evolution"],
                        communicationProtocol: "pub-sub",
                        consensusRequired: true,
                    },
                    performance: {
                        eventProcessingRate: 10000,
                        averageResponseTime: 200,
                        learningRate: 0.2,
                        improvementSuccess: 0.95,
                    },
                },
                ...overrides,
            }),
        };

        const factory = variants[variant];
        if (!factory) {
            throw new Error(`Unknown agent variant: ${variant}`);
        }

        return factory();
    }

    /**
     * Create agent collaboration scenarios
     */
    createCollaborationScenario(
        scenario: "security_incident" | "performance_crisis" | "quality_issue" | "cost_overrun",
    ): EventDrivenAgentFixture[] {
        const scenarios = {
            security_incident: () => [
                this.createVariant("performanceOptimizer"),
                this.createComplete({
                    agent: {
                        identity: {
                            id: "incident_responder",
                            name: "Incident Response Coordinator",
                            goal: "Coordinate security incident response",
                            goalCategory: "security",
                            description: "Emergency response agent",
                            version: "1.0.0",
                        },
                        collaboration: {
                            collaborators: ["security_guardian", "performance_optimizer"],
                            sharedGoals: ["incident_containment", "service_restoration"],
                            communicationProtocol: "broadcast",
                            consensusRequired: true,
                        },
                    },
                }),
            ],

            performance_crisis: () => [
                this.createVariant("performanceOptimizer"),
                this.createVariant("costOptimizer"),
                this.createVariant("qualityGuardian"),
            ],

            quality_issue: () => [
                this.createVariant("qualityGuardian"),
                this.createComplete({
                    agent: {
                        identity: {
                            id: "test_enhancer",
                            name: "Test Coverage Enhancer",
                            goal: "Improve test coverage and quality",
                            goalCategory: "quality",
                            description: "Test improvement agent",
                            version: "1.0.0",
                        },
                    },
                }),
            ],

            cost_overrun: () => [
                this.createVariant("costOptimizer"),
                this.createVariant("performanceOptimizer"),
            ],
        };

        const factory = scenarios[scenario];
        if (!factory) {
            throw new Error(`Unknown collaboration scenario: ${scenario}`);
        }

        return factory();
    }

    /**
     * Create evolution path showing agent capability growth
     */
    createEvolutionPath(stages = 5): EventDrivenAgentFixture[] {
        const evolutionStages = ["reactive", "adaptive", "predictive", "autonomous", "transcendent"];
        
        return Array.from({ length: Math.min(stages, evolutionStages.length) }, (_, i) => {
            const learningStrategies: LearningStrategy[] = 
                ["supervised", "supervised", "reinforcement", "reinforcement", "meta"];

            return this.createComplete({
                emergence: {
                    capabilities: [
                        "event_processing",
                        ...(i >= 1 ? ["pattern_learning"] : []),
                        ...(i >= 2 ? ["predictive_analysis"] : []),
                        ...(i >= 3 ? ["autonomous_decision"] : []),
                        ...(i >= 4 ? ["emergent_intelligence"] : []),
                    ],
                    evolutionPath: evolutionStages.slice(0, i + 1).join(" → "),
                },
                agent: {
                    identity: {
                        id: `agent_evolution_${i}`,
                        name: `${evolutionStages[i]} Agent`,
                        goal: `Achieve ${evolutionStages[i]} intelligence`,
                        goalCategory: "performance",
                        description: `Agent at ${evolutionStages[i]} stage`,
                        version: `${i + 1}.0.0`,
                    },
                    eventPatterns: Array.from({ length: i + 1 }, (_, j) => ({
                        pattern: `tier${j + 1}.*`,
                        relevance: 0.8 + (0.05 * j),
                        actions: ["analyze", "learn", "improve"],
                        learningEnabled: true,
                    })),
                    learning: {
                        strategy: learningStrategies[i],
                        enabled: true,
                        modelType: i < 2 ? "linear" : i < 4 ? "neural_network" : "transformer",
                        updateFrequency: 86400000 / (i + 1), // Faster updates
                        minDataPoints: 100 * (i + 1),
                        confidenceThreshold: 0.7 + (0.05 * i),
                    },
                    routineImprovement: {
                        enabled: i >= 1,
                        targetRoutines: Array.from({ length: i }, (_, j) => `routine_${j}`),
                        improvementTypes: ["optimization", "quality"],
                        proposalThreshold: 0.8 + (0.02 * i),
                        maxProposalsPerDay: i + 1,
                    },
                    collaboration: {
                        collaborators: i >= 2 ? [`agent_${i - 1}`, `agent_${i - 2}`] : [],
                        sharedGoals: [`level_${i}_goal`],
                        communicationProtocol: i < 3 ? "direct" : "pub-sub",
                        consensusRequired: i >= 3,
                    },
                    performance: {
                        eventProcessingRate: 100 * Math.pow(2, i),
                        averageResponseTime: 1000 / (i + 1),
                        learningRate: 0.01 * (i + 1),
                        improvementSuccess: 0.5 + (0.1 * i),
                    },
                },
            });
        });
    }
}

// Export factory instance
export const eventDrivenAgentFixtureFactory = new EventDrivenAgentFixtureFactory();

// Export pre-built fixtures
export const eventDrivenAgentFixtures = {
    minimal: eventDrivenAgentFixtureFactory.createMinimal(),
    complete: eventDrivenAgentFixtureFactory.createComplete(),
    withDefaults: eventDrivenAgentFixtureFactory.createWithDefaults(),
    variants: {
        performanceOptimizer: eventDrivenAgentFixtureFactory.createVariant("performanceOptimizer"),
        qualityGuardian: eventDrivenAgentFixtureFactory.createVariant("qualityGuardian"),
        costOptimizer: eventDrivenAgentFixtureFactory.createVariant("costOptimizer"),
        metaLearner: eventDrivenAgentFixtureFactory.createVariant("metaLearner"),
    },
    collaborationScenarios: {
        securityIncident: eventDrivenAgentFixtureFactory.createCollaborationScenario("security_incident"),
        performanceCrisis: eventDrivenAgentFixtureFactory.createCollaborationScenario("performance_crisis"),
        qualityIssue: eventDrivenAgentFixtureFactory.createCollaborationScenario("quality_issue"),
        costOverrun: eventDrivenAgentFixtureFactory.createCollaborationScenario("cost_overrun"),
    },
    evolution: {
        reactive: eventDrivenAgentFixtureFactory.createEvolutionPath(1)[0],
        adaptive: eventDrivenAgentFixtureFactory.createEvolutionPath(2)[1],
        predictive: eventDrivenAgentFixtureFactory.createEvolutionPath(3)[2],
        autonomous: eventDrivenAgentFixtureFactory.createEvolutionPath(4)[3],
        transcendent: eventDrivenAgentFixtureFactory.createEvolutionPath(5)[4],
    },
};
