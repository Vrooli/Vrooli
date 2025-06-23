/**
 * Execution Architecture Fixture Factories
 * 
 * Provides factory classes for creating execution fixtures, following the proven
 * factory patterns from the shared package config fixtures while extending them
 * for execution-specific requirements.
 * 
 * Integration with Shared Package:
 * - Builds on validated config fixture factories
 * - Provides same factory interface as shared package
 * - Enables composition and reuse of config patterns
 * - Maintains type safety throughout
 */

import type {
    BaseConfigObject,
    ChatConfigObject,
    RoutineConfigObject,
    RunConfigObject
} from "@vrooli/shared";
import { 
    chatConfigFixtures, 
    routineConfigFixtures, 
    runConfigFixtures 
} from "@vrooli/shared/__test/fixtures/config";
import {
    type ExecutionFixture,
    type SwarmFixture,
    type RoutineFixture,
    type ExecutionContextFixture,
    type EmergenceDefinition,
    type IntegrationDefinition,
    type ValidationResult,
    type ConfigType,
    validateConfigAgainstSchema,
    validateEmergence,
    validateIntegration,
    validateEvolutionPathways,
    combineValidationResults,
    createMinimalEmergence,
    createMinimalIntegration
} from "./executionValidationUtils.js";

// ================================================================================================
// Base Factory Interface (Following Shared Package Pattern)
// ================================================================================================

export interface ExecutionFixtureFactory<TConfig extends BaseConfigObject> {
    // Core fixture creation (like config fixtures)
    createMinimal(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    createComplete(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    createWithDefaults(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    
    // Variant collections (like config fixtures)
    createVariant(variant: string, overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    
    // Factory methods (like config fixtures)
    create(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    
    // Validation methods (like shared package validation)
    validateFixture(fixture: ExecutionFixture<TConfig>): Promise<ValidationResult>;
    isValid(fixture: unknown): fixture is ExecutionFixture<TConfig>;
    
    // Composition helpers (like config fixtures)
    merge(base: ExecutionFixture<TConfig>, override: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    applyDefaults(partialFixture: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
}

// ================================================================================================
// Swarm Fixture Factory (Tier 1: Coordination Intelligence)
// ================================================================================================

export class SwarmFixtureFactory implements ExecutionFixtureFactory<ChatConfigObject> {
    private configFactory = chatConfigFixtures;
    private configType: ConfigType = "chat";

    // Core fixture creation
    createMinimal(overrides?: Partial<SwarmFixture>): SwarmFixture {
        const baseFixture: SwarmFixture = {
            config: this.configFactory.minimal,
            emergence: {
                capabilities: ["basic_coordination"],
                evolutionPath: "individual → collaborative"
            },
            integration: {
                tier: "tier1",
                producedEvents: ["tier1.swarm.initialized"],
                consumedEvents: ["tier1.system.ready"]
            },
            metadata: {
                domain: "coordination",
                complexity: "simple",
                maintainer: "execution-team",
                lastUpdated: new Date().toISOString().split('T')[0]
            }
        };

        return this.merge(baseFixture, overrides || {});
    }

    createComplete(overrides?: Partial<SwarmFixture>): SwarmFixture {
        const baseFixture: SwarmFixture = {
            config: this.configFactory.complete,
            emergence: {
                capabilities: [
                    "natural_language_understanding",
                    "pattern_recognition", 
                    "adaptive_response",
                    "collaborative_intelligence"
                ],
                eventPatterns: ["tier2.routine.*", "tier3.execution.*"],
                evolutionPath: "reactive → proactive → predictive",
                emergenceConditions: {
                    minAgents: 2,
                    requiredResources: ["llm_access", "event_bus"],
                    environmentalFactors: ["multi_user_environment"]
                },
                learningMetrics: {
                    performanceImprovement: "task_completion_rate",
                    adaptationTime: "response_time_optimization",
                    innovationRate: "novel_solution_generation"
                }
            },
            integration: {
                tier: "tier1",
                producedEvents: [
                    "tier1.swarm.initialized",
                    "tier1.coordination.started",
                    "tier1.task.assigned",
                    "tier1.resource.allocated"
                ],
                consumedEvents: [
                    "tier2.routine.completed",
                    "tier3.execution.failed",
                    "system.resource.available"
                ],
                sharedResources: ["knowledge_base", "coordination_state"],
                crossTierDependencies: {
                    dependsOn: ["tier2.process_intelligence", "tier3.execution_engine"],
                    provides: ["strategic_coordination", "resource_allocation"]
                },
                mcpTools: ["swarm_coordinator", "resource_manager", "knowledge_synthesizer"]
            },
            swarmMetadata: {
                formation: "dynamic",
                coordinationPattern: "emergence",
                expectedAgentCount: 5,
                minViableAgents: 2,
                roles: [
                    { role: "coordinator", count: 1 },
                    { role: "specialist", count: 3 },
                    { role: "quality_assurer", count: 1 }
                ]
            },
            validation: {
                emergenceTests: ["test_pattern_recognition", "test_adaptive_response"],
                integrationTests: ["test_tier_communication", "test_event_flow"],
                evolutionTests: ["test_performance_improvement", "test_strategy_evolution"]
            },
            metadata: {
                domain: "multi_agent_coordination",
                complexity: "complex",
                maintainer: "swarm-intelligence-team",
                lastUpdated: new Date().toISOString().split('T')[0]
            }
        };

        return this.merge(baseFixture, overrides || {});
    }

    createWithDefaults(overrides?: Partial<SwarmFixture>): SwarmFixture {
        return this.createComplete(overrides);
    }

    // Variant creation
    createVariant(variant: string, overrides?: Partial<SwarmFixture>): SwarmFixture {
        const variants: Record<string, Partial<SwarmFixture>> = {
            customerSupport: {
                config: {
                    ...this.configFactory.variants.supportChat,
                    swarmTask: "Provide comprehensive customer support",
                    swarmSubTasks: [
                        { id: "inquiry_analysis", description: "Analyze customer inquiries" },
                        { id: "solution_generation", description: "Generate appropriate solutions" },
                        { id: "quality_assurance", description: "Ensure response quality" }
                    ]
                },
                emergence: {
                    capabilities: ["customer_satisfaction", "issue_resolution", "empathy_modeling"],
                    evolutionPath: "reactive → proactive → predictive"
                },
                swarmMetadata: {
                    formation: "hierarchical",
                    coordinationPattern: "delegation",
                    expectedAgentCount: 4,
                    minViableAgents: 2
                }
            },

            securityResponse: {
                config: {
                    ...this.configFactory.variants.privateTeamChat,
                    swarmTask: "Monitor and respond to security threats",
                    priority: "high"
                },
                emergence: {
                    capabilities: ["threat_detection", "adaptive_response", "pattern_recognition"],
                    eventPatterns: ["security.*", "system.error.*"],
                    evolutionPath: "reactive → proactive → predictive"
                },
                swarmMetadata: {
                    formation: "flat",
                    coordinationPattern: "consensus",
                    expectedAgentCount: 3,
                    minViableAgents: 1
                }
            },

            researchAnalysis: {
                config: {
                    ...this.configFactory.complete,
                    swarmTask: "Collaborative research and analysis",
                    features: ["document_analysis", "synthesis", "validation"]
                },
                emergence: {
                    capabilities: ["knowledge_synthesis", "quality_assurance", "collaborative_intelligence"],
                    evolutionPath: "individual → collaborative → emergent"
                },
                swarmMetadata: {
                    formation: "dynamic",
                    coordinationPattern: "emergence",
                    expectedAgentCount: 6,
                    minViableAgents: 3
                }
            }
        };

        const variantConfig = variants[variant];
        if (!variantConfig) {
            throw new Error(`Unknown swarm variant: ${variant}`);
        }

        return this.merge(this.createComplete(), { ...variantConfig, ...overrides });
    }

    // Factory methods
    create(overrides?: Partial<SwarmFixture>): SwarmFixture {
        return this.createComplete(overrides);
    }

    // Validation methods
    async validateFixture(fixture: SwarmFixture): Promise<ValidationResult> {
        const results = await Promise.all([
            validateConfigAgainstSchema(fixture.config, this.configType),
            Promise.resolve(validateEmergence(fixture.emergence)),
            Promise.resolve(validateIntegration(fixture.integration)),
            Promise.resolve(validateEvolutionPathways(fixture))
        ]);

        return combineValidationResults(results);
    }

    isValid(fixture: unknown): fixture is SwarmFixture {
        return (
            typeof fixture === "object" &&
            fixture !== null &&
            "config" in fixture &&
            "emergence" in fixture &&
            "integration" in fixture &&
            (fixture as any).integration.tier === "tier1"
        );
    }

    // Composition helpers
    merge(base: SwarmFixture, override: Partial<SwarmFixture>): SwarmFixture {
        return {
            ...base,
            ...override,
            config: { ...base.config, ...override.config },
            emergence: { ...base.emergence, ...override.emergence },
            integration: { ...base.integration, ...override.integration },
            swarmMetadata: { ...base.swarmMetadata, ...override.swarmMetadata },
            metadata: { ...base.metadata, ...override.metadata }
        };
    }

    applyDefaults(partialFixture: Partial<SwarmFixture>): SwarmFixture {
        return this.merge(this.createMinimal(), partialFixture);
    }
}

// ================================================================================================
// Routine Fixture Factory (Tier 2: Process Intelligence)
// ================================================================================================

export class RoutineFixtureFactory implements ExecutionFixtureFactory<RoutineConfigObject> {
    private configFactory = routineConfigFixtures;
    private configType: ConfigType = "routine";

    createMinimal(overrides?: Partial<RoutineFixture>): RoutineFixture {
        const baseFixture: RoutineFixture = {
            config: this.configFactory.minimal,
            emergence: {
                capabilities: ["basic_workflow_execution"],
                evolutionPath: "static → adaptive"
            },
            integration: {
                tier: "tier2",
                producedEvents: ["tier2.routine.started"],
                consumedEvents: ["tier1.task.assigned"]
            },
            evolutionStage: {
                current: "conversational",
                evolutionTriggers: ["performance_threshold"],
                performanceMetrics: {
                    averageExecutionTime: 10000, // 10 seconds
                    successRate: 0.7,
                    costPerExecution: 20
                }
            },
            metadata: {
                domain: "workflow_orchestration",
                complexity: "simple",
                maintainer: "process-team",
                lastUpdated: new Date().toISOString().split('T')[0]
            }
        };

        return this.merge(baseFixture, overrides || {});
    }

    createComplete(overrides?: Partial<RoutineFixture>): RoutineFixture {
        const baseFixture: RoutineFixture = {
            config: this.configFactory.complete,
            emergence: {
                capabilities: [
                    "workflow_automation",
                    "adaptive_response",
                    "error_recovery",
                    "performance_optimization"
                ],
                eventPatterns: ["tier1.task.*", "tier3.execution.*"],
                evolutionPath: "conversational → reasoning → deterministic → routing",
                emergenceConditions: {
                    minAgents: 1,
                    requiredResources: ["execution_engine", "state_manager"],
                    environmentalFactors: ["stable_infrastructure"]
                },
                learningMetrics: {
                    performanceImprovement: "execution_time_reduction",
                    adaptationTime: "strategy_switching_speed",
                    innovationRate: "novel_workflow_patterns"
                }
            },
            integration: {
                tier: "tier2",
                producedEvents: [
                    "tier2.routine.started",
                    "tier2.step.completed",
                    "tier2.routine.completed",
                    "tier2.error.occurred"
                ],
                consumedEvents: [
                    "tier1.task.assigned",
                    "tier1.priority.changed",
                    "tier3.execution.completed",
                    "tier3.tool.failed"
                ],
                sharedResources: ["workflow_state", "execution_context"],
                crossTierDependencies: {
                    dependsOn: ["tier1.coordination", "tier3.execution_engine"],
                    provides: ["workflow_orchestration", "process_intelligence"]
                },
                mcpTools: ["routine_navigator", "state_manager", "strategy_selector"]
            },
            evolutionStage: {
                current: "reasoning",
                nextStage: "deterministic",
                evolutionTriggers: [
                    "success_rate_above_90_percent",
                    "execution_time_consistent",
                    "error_rate_below_5_percent"
                ],
                performanceMetrics: {
                    averageExecutionTime: 3000, // 3 seconds
                    successRate: 0.92,
                    costPerExecution: 8
                }
            },
            validation: {
                emergenceTests: ["test_workflow_adaptation", "test_error_recovery"],
                integrationTests: ["test_state_management", "test_tier_coordination"],
                evolutionTests: ["test_strategy_evolution", "test_performance_improvement"]
            },
            metadata: {
                domain: "process_intelligence",
                complexity: "complex",
                maintainer: "workflow-team",
                lastUpdated: new Date().toISOString().split('T')[0]
            }
        };

        return this.merge(baseFixture, overrides || {});
    }

    createWithDefaults(overrides?: Partial<RoutineFixture>): RoutineFixture {
        return this.createComplete(overrides);
    }

    // Variant creation
    createVariant(variant: string, overrides?: Partial<RoutineFixture>): RoutineFixture {
        const variants: Record<string, Partial<RoutineFixture>> = {
            customerInquiry: {
                config: {
                    ...this.configFactory.variants.action,
                    name: "Customer Inquiry Processing",
                    description: "Analyze and respond to customer inquiries"
                },
                emergence: {
                    capabilities: ["natural_language_understanding", "context_retention"],
                    evolutionPath: "conversational → reasoning → deterministic"
                },
                evolutionStage: {
                    current: "conversational",
                    evolutionTriggers: ["volume_threshold", "accuracy_improvement"],
                    performanceMetrics: {
                        averageExecutionTime: 15000,
                        successRate: 0.75,
                        costPerExecution: 25
                    }
                }
            },

            dataProcessing: {
                config: {
                    ...this.configFactory.variants.informational,
                    name: "Data Processing Pipeline",
                    description: "Process and transform data through multiple stages"
                },
                emergence: {
                    capabilities: ["data_validation", "transformation_optimization"],
                    evolutionPath: "manual → automated → optimized"
                },
                evolutionStage: {
                    current: "deterministic",
                    performanceMetrics: {
                        averageExecutionTime: 2000,
                        successRate: 0.95,
                        costPerExecution: 5
                    }
                }
            },

            securityCheck: {
                config: {
                    ...this.configFactory.complete,
                    name: "Security Validation Routine",
                    priority: "high"
                },
                emergence: {
                    capabilities: ["threat_detection", "compliance_validation"],
                    eventPatterns: ["security.*", "compliance.*"],
                    evolutionPath: "reactive → proactive → predictive"
                },
                evolutionStage: {
                    current: "reasoning",
                    performanceMetrics: {
                        averageExecutionTime: 5000,
                        successRate: 0.98,
                        costPerExecution: 12
                    }
                }
            }
        };

        const variantConfig = variants[variant];
        if (!variantConfig) {
            throw new Error(`Unknown routine variant: ${variant}`);
        }

        return this.merge(this.createComplete(), { ...variantConfig, ...overrides });
    }

    create(overrides?: Partial<RoutineFixture>): RoutineFixture {
        return this.createComplete(overrides);
    }

    async validateFixture(fixture: RoutineFixture): Promise<ValidationResult> {
        const results = await Promise.all([
            validateConfigAgainstSchema(fixture.config, this.configType),
            Promise.resolve(validateEmergence(fixture.emergence)),
            Promise.resolve(validateIntegration(fixture.integration)),
            Promise.resolve(validateEvolutionPathways(fixture))
        ]);

        return combineValidationResults(results);
    }

    isValid(fixture: unknown): fixture is RoutineFixture {
        return (
            typeof fixture === "object" &&
            fixture !== null &&
            "config" in fixture &&
            "emergence" in fixture &&
            "integration" in fixture &&
            (fixture as any).integration.tier === "tier2"
        );
    }

    merge(base: RoutineFixture, override: Partial<RoutineFixture>): RoutineFixture {
        return {
            ...base,
            ...override,
            config: { ...base.config, ...override.config },
            emergence: { ...base.emergence, ...override.emergence },
            integration: { ...base.integration, ...override.integration },
            evolutionStage: { ...base.evolutionStage, ...override.evolutionStage },
            metadata: { ...base.metadata, ...override.metadata }
        };
    }

    applyDefaults(partialFixture: Partial<RoutineFixture>): RoutineFixture {
        return this.merge(this.createMinimal(), partialFixture);
    }
}

// ================================================================================================
// Execution Context Fixture Factory (Tier 3: Execution Intelligence)
// ================================================================================================

export class ExecutionContextFixtureFactory implements ExecutionFixtureFactory<RunConfigObject> {
    private configFactory = runConfigFixtures;
    private configType: ConfigType = "run";

    createMinimal(overrides?: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        const baseFixture: ExecutionContextFixture = {
            config: this.configFactory.minimal,
            emergence: {
                capabilities: ["basic_tool_execution"],
                evolutionPath: "static → adaptive"
            },
            integration: {
                tier: "tier3",
                producedEvents: ["tier3.execution.started"],
                consumedEvents: ["tier2.step.assigned"]
            },
            executionMetadata: {
                supportedStrategies: ["conversational"],
                toolDependencies: ["basic_tools"],
                performanceCharacteristics: {
                    latency: "< 1 second",
                    throughput: "10 ops/sec",
                    resourceUsage: "low"
                }
            },
            metadata: {
                domain: "tool_execution",
                complexity: "simple",
                maintainer: "execution-team",
                lastUpdated: new Date().toISOString().split('T')[0]
            }
        };

        return this.merge(baseFixture, overrides || {});
    }

    createComplete(overrides?: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        const baseFixture: ExecutionContextFixture = {
            config: this.configFactory.complete,
            emergence: {
                capabilities: [
                    "adaptive_tool_selection",
                    "resource_optimization",
                    "error_recovery",
                    "performance_optimization"
                ],
                eventPatterns: ["tier2.step.*", "tool.*", "resource.*"],
                evolutionPath: "reactive → adaptive → predictive",
                emergenceConditions: {
                    minAgents: 1,
                    requiredResources: ["tool_registry", "resource_manager"],
                    environmentalFactors: ["stable_execution_environment"]
                },
                learningMetrics: {
                    performanceImprovement: "tool_selection_accuracy",
                    adaptationTime: "context_switching_speed",
                    innovationRate: "novel_execution_patterns"
                }
            },
            integration: {
                tier: "tier3",
                producedEvents: [
                    "tier3.execution.started",
                    "tier3.tool.invoked",
                    "tier3.execution.completed",
                    "tier3.error.handled"
                ],
                consumedEvents: [
                    "tier2.step.assigned",
                    "tier2.context.updated",
                    "tool.result.ready",
                    "resource.availability.changed"
                ],
                sharedResources: ["tool_registry", "execution_context", "resource_pool"],
                crossTierDependencies: {
                    dependsOn: ["tier2.process_orchestration"],
                    provides: ["tool_execution", "resource_management", "context_adaptation"]
                },
                mcpTools: ["tool_orchestrator", "context_manager", "resource_optimizer"]
            },
            executionMetadata: {
                supportedStrategies: [
                    "conversational",
                    "reasoning", 
                    "deterministic"
                ],
                toolDependencies: [
                    "llm_interface",
                    "file_operations",
                    "api_client",
                    "data_processor"
                ],
                performanceCharacteristics: {
                    latency: "< 500ms",
                    throughput: "100 ops/sec",
                    resourceUsage: "adaptive"
                }
            },
            validation: {
                emergenceTests: ["test_tool_adaptation", "test_resource_optimization"],
                integrationTests: ["test_context_management", "test_error_handling"],
                evolutionTests: ["test_strategy_evolution", "test_performance_improvement"]
            },
            metadata: {
                domain: "execution_intelligence",
                complexity: "complex",
                maintainer: "execution-intelligence-team",
                lastUpdated: new Date().toISOString().split('T')[0]
            }
        };

        return this.merge(baseFixture, overrides || {});
    }

    createWithDefaults(overrides?: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        return this.createComplete(overrides);
    }

    createVariant(variant: string, overrides?: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        const variants: Record<string, Partial<ExecutionContextFixture>> = {
            highPerformance: {
                config: {
                    ...this.configFactory.complete,
                    priority: "high",
                    resourceLimits: { maxMemory: "4GB", maxCPU: "2 cores" }
                },
                emergence: {
                    capabilities: ["performance_optimization", "resource_management"],
                    evolutionPath: "basic → optimized → peak_performance"
                },
                executionMetadata: {
                    supportedStrategies: ["deterministic"],
                    performanceCharacteristics: {
                        latency: "< 100ms",
                        throughput: "500 ops/sec",
                        resourceUsage: "high"
                    }
                }
            },

            secureExecution: {
                config: {
                    ...this.configFactory.complete,
                    securityLevel: "high",
                    auditMode: true
                },
                emergence: {
                    capabilities: ["security_validation", "audit_trail_generation"],
                    eventPatterns: ["security.*", "audit.*"],
                    evolutionPath: "basic → secure → hardened"
                },
                executionMetadata: {
                    supportedStrategies: ["reasoning", "deterministic"],
                    toolDependencies: ["security_validator", "audit_logger"]
                }
            },

            resourceConstrained: {
                config: {
                    ...this.configFactory.minimal,
                    resourceLimits: { maxMemory: "512MB", maxCPU: "0.5 cores" }
                },
                emergence: {
                    capabilities: ["resource_optimization", "efficiency_maximization"],
                    evolutionPath: "basic → efficient → ultra_efficient"
                },
                executionMetadata: {
                    supportedStrategies: ["conversational"],
                    performanceCharacteristics: {
                        latency: "< 2 seconds",
                        throughput: "10 ops/sec",
                        resourceUsage: "minimal"
                    }
                }
            }
        };

        const variantConfig = variants[variant];
        if (!variantConfig) {
            throw new Error(`Unknown execution context variant: ${variant}`);
        }

        return this.merge(this.createComplete(), { ...variantConfig, ...overrides });
    }

    create(overrides?: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        return this.createComplete(overrides);
    }

    async validateFixture(fixture: ExecutionContextFixture): Promise<ValidationResult> {
        const results = await Promise.all([
            validateConfigAgainstSchema(fixture.config, this.configType),
            Promise.resolve(validateEmergence(fixture.emergence)),
            Promise.resolve(validateIntegration(fixture.integration)),
            Promise.resolve(validateEvolutionPathways(fixture))
        ]);

        return combineValidationResults(results);
    }

    isValid(fixture: unknown): fixture is ExecutionContextFixture {
        return (
            typeof fixture === "object" &&
            fixture !== null &&
            "config" in fixture &&
            "emergence" in fixture &&
            "integration" in fixture &&
            (fixture as any).integration.tier === "tier3"
        );
    }

    merge(base: ExecutionContextFixture, override: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        return {
            ...base,
            ...override,
            config: { ...base.config, ...override.config },
            emergence: { ...base.emergence, ...override.emergence },
            integration: { ...base.integration, ...override.integration },
            executionMetadata: { ...base.executionMetadata, ...override.executionMetadata },
            metadata: { ...base.metadata, ...override.metadata }
        };
    }

    applyDefaults(partialFixture: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        return this.merge(this.createMinimal(), partialFixture);
    }
}

// ================================================================================================
// Factory Registry and Utilities
// ================================================================================================

export const executionFixtureFactories = {
    swarm: new SwarmFixtureFactory(),
    routine: new RoutineFixtureFactory(),
    executionContext: new ExecutionContextFixtureFactory()
} as const;

export type FixtureFactoryType = keyof typeof executionFixtureFactories;

/**
 * Get a factory by type with proper typing
 */
export function getExecutionFixtureFactory<T extends FixtureFactoryType>(
    type: T
): typeof executionFixtureFactories[T] {
    return executionFixtureFactories[type];
}

/**
 * Create execution fixture from factory type and variant
 */
export function createExecutionFixture<T extends FixtureFactoryType>(
    type: T,
    variant?: string,
    overrides?: any
): T extends "swarm" ? SwarmFixture : 
   T extends "routine" ? RoutineFixture : 
   T extends "executionContext" ? ExecutionContextFixture : never {
    const factory = getExecutionFixtureFactory(type);
    
    if (variant) {
        return factory.createVariant(variant, overrides) as any;
    }
    
    return factory.create(overrides) as any;
}

/**
 * Validate any execution fixture using appropriate factory
 */
export async function validateExecutionFixture(
    fixture: ExecutionFixture<any>,
    factoryType: FixtureFactoryType
): Promise<ValidationResult> {
    const factory = getExecutionFixtureFactory(factoryType);
    return factory.validateFixture(fixture as any);
}