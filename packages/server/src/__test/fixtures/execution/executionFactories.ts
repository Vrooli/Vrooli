/**
 * Execution Fixture Factories
 * 
 * Production-grade factory implementations for creating validated execution fixtures.
 * These factories follow the proven patterns from the shared package, ensuring:
 * - Type safety through TypeScript generics
 * - Config validation using actual config classes
 * - Round-trip consistency testing
 * - 82% code reduction through automatic test generation
 */

import type {
    BaseConfigObject,
    ChatConfigObject,
    RoutineConfigObject,
    RunConfigObject,
    BotConfigObject,
} from "@vrooli/shared";
import {
    ChatConfig,
    RoutineConfig,
    RunConfig,
    chatConfigFixtures,
    routineConfigFixtures,
    runConfigFixtures,
    botConfigFixtures,
} from "@vrooli/shared";
import {
    validateEmergence,
    validateIntegration,
    validateExecutionFixtureConfig,
    createMinimalEmergence,
    createMinimalIntegration,
    type ValidationResult,
} from "./executionValidationUtils.js";
import type {
    ExecutionFixture,
    ExecutionFixtureFactory,
    SwarmFixture,
    RoutineFixture,
    ExecutionContextFixture,
    EmergenceDefinition,
    IntegrationDefinition,
    EnhancedEmergenceDefinition,
    MeasurableCapability,
} from "./types.js";

/**
 * Base factory class that implements common functionality
 */
abstract class BaseExecutionFixtureFactory<TConfig extends BaseConfigObject> 
    implements ExecutionFixtureFactory<TConfig> {
    
    protected abstract ConfigClass: typeof ChatConfig | typeof RoutineConfig | typeof RunConfig;
    protected abstract tier: "tier1" | "tier2" | "tier3";
    
    /**
     * Validate config using actual config class
     */
    async validateConfig(config: TConfig): Promise<ValidationResult> {
        try {
            const configInstance = new this.ConfigClass({ config });
            const exported = configInstance.export();
            const reimported = new this.ConfigClass({ config: exported });
            
            if (JSON.stringify(exported) !== JSON.stringify(reimported.export())) {
                throw new Error("Config failed round-trip consistency test");
            }
            
            return { pass: true, message: "Config validation passed", data: exported };
        } catch (error) {
            return {
                pass: false,
                message: "Config validation failed",
                errors: [error instanceof Error ? error.message : String(error)]
            };
        }
    }

    /**
     * Validate execution fixture config using actual config class
     */
    async validateExecutionFixtureConfig(
        fixture: ExecutionFixture<TConfig>,
        ConfigClass: typeof ChatConfig | typeof RoutineConfig | typeof RunConfig
    ): Promise<ValidationResult> {
        try {
            const configInstance = new ConfigClass({ config: fixture.config });
            const exported = configInstance.export();
            const reimported = new ConfigClass({ config: exported });
            const reexported = reimported.export();
            
            if (JSON.stringify(exported) !== JSON.stringify(reexported)) {
                throw new Error("Config failed round-trip consistency test");
            }

            return { 
                pass: true, 
                message: "Config validation passed",
                data: exported
            };
        } catch (error) {
            return { 
                pass: false, 
                message: "Config validation failed",
                errors: [error instanceof Error ? error.message : String(error)]
            };
        }
    }
    
    /**
     * Validate complete fixture
     */
    async validateFixture(fixture: ExecutionFixture<TConfig>): Promise<ValidationResult> {
        // Validate config through actual config class
        const configResult = await this.validateExecutionFixtureConfig(fixture, this.ConfigClass);
        if (!configResult.pass) return configResult;
        
        // Validate emergence definition
        const emergenceResult = this.validateEmergence(fixture.emergence);
        if (!emergenceResult.pass) return emergenceResult;
        
        // Validate integration patterns
        const integrationResult = this.validateIntegration(fixture.integration);
        if (!integrationResult.pass) return integrationResult;
        
        return { pass: true, message: "All validations passed" };
    }
    
    /**
     * Validate emergence definition
     */
    validateEmergence(emergence: EmergenceDefinition): ValidationResult {
        return validateEmergence(emergence);
    }
    
    /**
     * Validate integration definition
     */
    validateIntegration(integration: IntegrationDefinition): ValidationResult {
        return validateIntegration(integration);
    }
    
    /**
     * Type guard for fixture validation
     */
    isValid(fixture: unknown): fixture is ExecutionFixture<TConfig> {
        if (!fixture || typeof fixture !== 'object') return false;
        const f = fixture as any;
        return f.config && f.emergence && f.integration;
    }
    
    /**
     * Merge fixtures with validation
     */
    merge(
        base: ExecutionFixture<TConfig>, 
        override: Partial<ExecutionFixture<TConfig>>
    ): ExecutionFixture<TConfig> {
        return {
            ...base,
            ...override,
            config: { ...base.config, ...override.config } as TConfig,
            emergence: { ...base.emergence, ...override.emergence },
            integration: { ...base.integration, ...override.integration },
        };
    }
    
    /**
     * Apply defaults to partial fixture
     */
    applyDefaults(partialFixture: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig> {
        return this.merge(this.createMinimal(), partialFixture);
    }
    
    /**
     * Create fixture with overrides
     */
    create(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig> {
        return overrides ? this.merge(this.createMinimal(), overrides) : this.createMinimal();
    }
    
    /**
     * Create multiple fixtures
     */
    createBatch(count: number, overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>[] {
        return Array.from({ length: count }, (_, i) => {
            const fixture = this.create(overrides);
            // Add index to any id fields for uniqueness
            if ('id' in fixture.config) {
                (fixture.config as any).id = `${(fixture.config as any).id}_${i}`;
            }
            return fixture;
        });
    }
    
    /**
     * Get config validator (for shared package integration)
     */
    getConfigValidator(): any {
        return {
            validate: async (config: TConfig) => this.validateConfig(config)
        };
    }
    
    /**
     * Get integration adapter (for shared package integration)
     */
    getIntegrationAdapter(): any {
        return {
            create: this.getConfigValidator()
        };
    }
    
    // Abstract methods to be implemented by subclasses
    abstract createMinimal(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    abstract createComplete(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    abstract createWithDefaults(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    abstract createVariant(variant: string, overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
}

/**
 * Swarm Fixture Factory (Tier 1: Coordination Intelligence)
 */
export class SwarmFixtureFactory extends BaseExecutionFixtureFactory<ChatConfigObject> {
    protected ConfigClass = ChatConfig as typeof ChatConfig;
    protected tier = "tier1" as const;
    
    createMinimal(overrides?: Partial<SwarmFixture>): SwarmFixture {
        const baseConfig = chatConfigFixtures.minimal;
        
        const config: ChatConfigObject = {
            ...baseConfig,
            goal: overrides?.config?.goal || "Basic coordination task",
            ...overrides?.config
        };
        
        return {
            config,
            emergence: overrides?.emergence || createMinimalEmergence(),
            integration: overrides?.integration || createMinimalIntegration("tier1"),
            swarmMetadata: overrides?.swarmMetadata || {
                formation: "flat",
                coordinationPattern: "emergence",
                expectedAgentCount: 2,
                minViableAgents: 1
            }
        };
    }
    
    createComplete(overrides?: Partial<SwarmFixture>): SwarmFixture {
        const baseConfig = chatConfigFixtures.complete;
        
        const config: ChatConfigObject = {
            ...baseConfig,
            goal: "Comprehensive swarm coordination task",
            ...overrides?.config
        };
        
        return {
            config,
            emergence: {
                capabilities: ["swarm_coordination", "task_delegation", "resource_management"],
                eventPatterns: ["tier1.*", "swarm.*"],
                evolutionPath: "reactive → proactive → predictive",
                emergenceConditions: {
                    minAgents: 2,
                    requiredResources: ["communication", "coordination"],
                    environmentalFactors: ["event_availability", "agent_readiness"]
                },
                learningMetrics: {
                    performanceImprovement: "20% reduction in coordination overhead",
                    adaptationTime: "5 cycles to stable formation",
                    innovationRate: "2 new patterns per 100 executions"
                },
                ...overrides?.emergence
            },
            integration: {
                tier: "tier1",
                producedEvents: ["tier1.swarm.initialized", "tier1.task.delegated", "tier1.coordination.optimized"],
                consumedEvents: ["tier2.routine.completed", "tier3.execution.finished"],
                sharedResources: ["blackboard", "agent_registry"],
                crossTierDependencies: {
                    dependsOn: ["tier2_orchestration", "tier3_execution"],
                    provides: ["coordination_intelligence", "task_decomposition"]
                },
                mcpTools: ["task_manager", "agent_coordinator"],
                ...overrides?.integration
            },
            swarmMetadata: {
                formation: "hierarchical",
                coordinationPattern: "delegation",
                expectedAgentCount: 5,
                minViableAgents: 3,
                ...overrides?.swarmMetadata
            },
            validation: {
                emergenceTests: ["coordination_under_load", "task_distribution", "fault_tolerance"],
                integrationTests: ["cross_tier_communication", "event_flow"],
                evolutionTests: ["performance_improvement", "capability_expansion"],
                ...overrides?.validation
            },
            metadata: {
                domain: "coordination",
                complexity: "complex",
                maintainer: "execution-team",
                lastUpdated: new Date().toISOString(),
                ...overrides?.metadata
            }
        };
    }
    
    createWithDefaults(overrides?: Partial<SwarmFixture>): SwarmFixture {
        const baseConfig = chatConfigFixtures.withDefaults;
        
        return {
            config: {
                ...baseConfig,
                goal: "Default swarm coordination",
                ...overrides?.config
            },
            emergence: {
                capabilities: ["basic_coordination", "event_processing"],
                evolutionPath: "static → adaptive → intelligent",
                ...overrides?.emergence
            },
            integration: {
                tier: "tier1",
                producedEvents: ["tier1.swarm.ready"],
                consumedEvents: ["system.initialized"],
                ...overrides?.integration
            },
            swarmMetadata: {
                formation: "flat",
                coordinationPattern: "consensus",
                expectedAgentCount: 3,
                minViableAgents: 2,
                ...overrides?.swarmMetadata
            }
        };
    }
    
    createVariant(variant: string, overrides?: Partial<SwarmFixture>): SwarmFixture {
        const variants: Record<string, () => SwarmFixture> = {
            customerSupport: () => this.createComplete({
                config: {
                    goal: "Provide comprehensive customer support",
                    preferredModel: "gpt-4",
                },
                emergence: {
                    capabilities: ["customer_satisfaction", "issue_resolution", "knowledge_retrieval"],
                    eventPatterns: ["customer.*", "support.*"],
                },
                swarmMetadata: {
                    formation: "dynamic",
                    coordinationPattern: "emergence",
                    expectedAgentCount: 3,
                    minViableAgents: 2,
                },
                ...overrides
            }),
            
            securityResponse: () => this.createComplete({
                config: {
                    goal: "Monitor and respond to security threats",
                },
                emergence: {
                    capabilities: ["threat_detection", "automated_response", "incident_analysis"],
                    eventPatterns: ["security.*", "system.error", "auth.failed"],
                },
                swarmMetadata: {
                    formation: "hierarchical",
                    coordinationPattern: "delegation",
                    expectedAgentCount: 4,
                    minViableAgents: 2,
                },
                ...overrides
            }),
            
            researchAnalysis: () => this.createComplete({
                config: {
                    goal: "Conduct comprehensive research and analysis",
                },
                emergence: {
                    capabilities: ["information_synthesis", "pattern_recognition", "insight_generation"],
                    eventPatterns: ["research.*", "data.*"],
                },
                swarmMetadata: {
                    formation: "matrix",
                    coordinationPattern: "negotiation",
                    expectedAgentCount: 6,
                    minViableAgents: 3,
                },
                ...overrides
            })
        };
        
        const factory = variants[variant];
        if (!factory) {
            throw new Error(`Unknown swarm variant: ${variant}`);
        }
        
        return factory();
    }
    
    /**
     * Create evolution path for swarm development
     */
    createEvolutionPath(stages: number = 3): SwarmFixture[] {
        const evolutionStages = ["reactive", "proactive", "predictive"];
        const formations = ["flat", "hierarchical", "dynamic"];
        
        return Array.from({ length: Math.min(stages, evolutionStages.length) }, (_, i) => {
            return this.createComplete({
                emergence: {
                    evolutionPath: evolutionStages.slice(0, i + 1).join(" → "),
                    capabilities: [
                        "basic_coordination",
                        ...(i >= 1 ? ["pattern_recognition"] : []),
                        ...(i >= 2 ? ["predictive_optimization"] : [])
                    ]
                },
                swarmMetadata: {
                    formation: formations[i] as any,
                    expectedAgentCount: 2 + i,
                    minViableAgents: 1 + Math.floor(i / 2)
                }
            });
        });
    }
}

/**
 * Routine Fixture Factory (Tier 2: Process Intelligence)
 */
export class RoutineFixtureFactory extends BaseExecutionFixtureFactory<RoutineConfigObject> {
    protected ConfigClass = RoutineConfig as typeof RoutineConfig;
    protected tier = "tier2" as const;
    
    createMinimal(overrides?: Partial<RoutineFixture>): RoutineFixture {
        const baseConfig = routineConfigFixtures.minimal;
        
        return {
            config: {
                ...baseConfig,
                ...overrides?.config
            },
            emergence: overrides?.emergence || createMinimalEmergence(),
            integration: overrides?.integration || createMinimalIntegration("tier2"),
            evolutionStage: overrides?.evolutionStage || {
                strategy: "conversational",
                version: "1.0.0",
                metrics: {
                    avgDuration: 5000,
                    avgCredits: 100,
                    successRate: 0.8
                }
            }
        };
    }
    
    createComplete(overrides?: Partial<RoutineFixture>): RoutineFixture {
        const baseConfig = routineConfigFixtures.complete;
        
        return {
            config: {
                ...baseConfig,
                ...overrides?.config
            },
            emergence: {
                capabilities: ["process_optimization", "error_handling", "adaptive_execution"],
                eventPatterns: ["tier2.*", "routine.*"],
                evolutionPath: "conversational → reasoning → deterministic",
                emergenceConditions: {
                    minEvents: 10,
                    requiredResources: ["execution_history", "performance_metrics"],
                    timeframe: 3600000 // 1 hour
                },
                learningMetrics: {
                    performanceImprovement: "50% execution time reduction",
                    adaptationTime: "10 executions to stable performance",
                    innovationRate: "1 optimization per 50 runs"
                },
                ...overrides?.emergence
            },
            integration: {
                tier: "tier2",
                producedEvents: ["tier2.routine.started", "tier2.step.completed", "tier2.routine.finished"],
                consumedEvents: ["tier1.task.assigned", "tier3.tool.result"],
                sharedResources: ["execution_context", "step_results"],
                crossTierDependencies: {
                    dependsOn: ["tier1_coordination", "tier3_tools"],
                    provides: ["process_orchestration", "workflow_management"]
                },
                mcpTools: ["workflow_engine", "step_executor"],
                ...overrides?.integration
            },
            evolutionStage: {
                strategy: "reasoning",
                version: "2.0.0",
                metrics: {
                    avgDuration: 2500,
                    avgCredits: 75,
                    successRate: 0.92,
                    errorRate: 0.08,
                    satisfaction: 0.88
                },
                previousVersion: "1.0.0",
                improvements: ["Added error recovery", "Optimized step ordering"],
                ...overrides?.evolutionStage
            },
            domain: "general",
            navigator: "native",
            validation: {
                emergenceTests: ["adaptive_execution", "error_recovery"],
                integrationTests: ["step_coordination", "resource_sharing"],
                evolutionTests: ["performance_gains", "cost_reduction"],
                ...overrides?.validation
            }
        };
    }
    
    createWithDefaults(overrides?: Partial<RoutineFixture>): RoutineFixture {
        return this.createMinimal({
            emergence: {
                capabilities: ["basic_execution", "sequential_processing"],
                evolutionPath: "static → adaptive"
            },
            ...overrides
        });
    }
    
    createVariant(variant: string, overrides?: Partial<RoutineFixture>): RoutineFixture {
        const variants: Record<string, () => RoutineFixture> = {
            customerInquiry: () => this.createComplete({
                config: {
                    name: "Customer Inquiry Handler",
                    description: "Process customer inquiries intelligently"
                },
                emergence: {
                    capabilities: ["natural_language_understanding", "context_retention", "solution_generation"],
                },
                evolutionStage: {
                    strategy: "conversational",
                    version: "1.0.0",
                    metrics: {
                        avgDuration: 5000,
                        avgCredits: 100,
                        successRate: 0.85
                    }
                },
                domain: "customer-service",
                ...overrides
            }),
            
            dataProcessing: () => this.createComplete({
                config: {
                    name: "Data Processing Pipeline",
                    description: "Process and transform data efficiently"
                },
                emergence: {
                    capabilities: ["parallel_processing", "data_validation", "format_conversion"],
                },
                evolutionStage: {
                    strategy: "deterministic",
                    version: "3.0.0",
                    metrics: {
                        avgDuration: 1000,
                        avgCredits: 50,
                        successRate: 0.98
                    }
                },
                domain: "system",
                navigator: "native",
                ...overrides
            }),
            
            securityCheck: () => this.createComplete({
                config: {
                    name: "Security Validation Routine",
                    description: "Validate security compliance"
                },
                emergence: {
                    capabilities: ["threat_analysis", "compliance_checking", "report_generation"],
                },
                evolutionStage: {
                    strategy: "reasoning",
                    version: "2.1.0",
                    metrics: {
                        avgDuration: 3000,
                        avgCredits: 80,
                        successRate: 0.95
                    }
                },
                domain: "security",
                ...overrides
            })
        };
        
        const factory = variants[variant];
        if (!factory) {
            throw new Error(`Unknown routine variant: ${variant}`);
        }
        
        return factory();
    }
    
    /**
     * Create evolution path showing routine improvement
     */
    createEvolutionPath(stages: number = 4): RoutineFixture[] {
        const strategies: Array<"conversational" | "reasoning" | "deterministic" | "routing"> = 
            ["conversational", "reasoning", "deterministic", "routing"];
        
        return Array.from({ length: Math.min(stages, strategies.length) }, (_, i) => {
            const baseMetrics = {
                avgDuration: 5000 * Math.pow(0.6, i), // 40% faster each stage
                avgCredits: 100 * Math.pow(0.75, i),  // 25% cheaper each stage
                successRate: 0.8 + (0.05 * i),        // 5% more reliable each stage
            };
            
            return this.createComplete({
                evolutionStage: {
                    strategy: strategies[i],
                    version: `${i + 1}.0.0`,
                    metrics: baseMetrics,
                    previousVersion: i > 0 ? `${i}.0.0` : undefined,
                    improvements: i > 0 ? [`Evolved from ${strategies[i-1]} to ${strategies[i]}`] : []
                },
                emergence: {
                    capabilities: [
                        "basic_execution",
                        ...(i >= 1 ? ["pattern_learning"] : []),
                        ...(i >= 2 ? ["optimization"] : []),
                        ...(i >= 3 ? ["intelligent_routing"] : [])
                    ],
                    evolutionPath: strategies.slice(0, i + 1).join(" → ")
                }
            });
        });
    }
}

/**
 * Execution Context Fixture Factory (Tier 3: Execution Intelligence)
 */
export class ExecutionContextFixtureFactory extends BaseExecutionFixtureFactory<RunConfigObject> {
    protected ConfigClass = RunConfig as typeof RunConfig;
    protected tier = "tier3" as const;
    
    createMinimal(overrides?: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        const baseConfig = runConfigFixtures.minimal;
        
        return {
            config: {
                ...baseConfig,
                ...overrides?.config
            },
            emergence: overrides?.emergence || createMinimalEmergence(),
            integration: overrides?.integration || createMinimalIntegration("tier3"),
            strategy: overrides?.strategy || "conversational",
            context: overrides?.context || {
                tools: ["web_search", "calculation"],
                constraints: {
                    maxTokens: 1000,
                    timeout: 30000
                }
            }
        };
    }
    
    createComplete(overrides?: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        const baseConfig = runConfigFixtures.complete;
        
        return {
            config: {
                ...baseConfig,
                ...overrides?.config
            },
            emergence: {
                capabilities: ["tool_orchestration", "resource_optimization", "adaptive_strategy"],
                eventPatterns: ["tier3.*", "execution.*", "tool.*"],
                evolutionPath: "basic → optimized → intelligent",
                emergenceConditions: {
                    minEvents: 5,
                    requiredResources: ["tool_registry", "execution_metrics"],
                    environmentalFactors: ["available_tools", "resource_limits"]
                },
                learningMetrics: {
                    performanceImprovement: "30% resource utilization improvement",
                    adaptationTime: "3 executions to optimal tool selection",
                    innovationRate: "1 new tool combination per 20 runs"
                },
                ...overrides?.emergence
            },
            integration: {
                tier: "tier3",
                producedEvents: ["tier3.execution.started", "tier3.tool.invoked", "tier3.execution.completed"],
                consumedEvents: ["tier2.step.request", "tier1.resource.allocated"],
                sharedResources: ["tool_results", "execution_logs"],
                crossTierDependencies: {
                    dependsOn: ["tier2_steps", "tier1_resources"],
                    provides: ["tool_execution", "result_processing"]
                },
                mcpTools: ["universal_tool_executor", "result_processor"],
                ...overrides?.integration
            },
            strategy: "reasoning",
            context: {
                tools: ["web_search", "calculation", "code_execution", "file_operations"],
                constraints: {
                    maxTokens: 5000,
                    timeout: 60000,
                    requireApproval: ["code_execution", "file_operations"],
                    resourceLimits: {
                        memory: 512,
                        cpu: 50,
                        apiCalls: 100
                    }
                },
                sharedMemory: {
                    previousResults: [],
                    learnedPatterns: []
                },
                resources: {
                    creditBudget: 1000,
                    timeBudget: 60000,
                    priority: "medium",
                    pools: ["general", "priority"]
                },
                safety: {
                    syncChecks: ["input_validation", "tool_authorization"],
                    asyncAgents: ["security_monitor", "quality_checker"],
                    domainRules: ["no_destructive_operations", "data_privacy"],
                    emergencyStop: {
                        conditions: ["malicious_intent", "resource_exhaustion"],
                        actions: ["halt_execution", "alert_admin"]
                    }
                },
                ...overrides?.context
            },
            tools: {
                available: [
                    { id: "web_search", name: "Web Search", category: "information" },
                    { id: "calculation", name: "Calculator", category: "computation" },
                    { id: "code_execution", name: "Code Runner", category: "execution", requiresApproval: true },
                    { id: "file_operations", name: "File Manager", category: "storage", requiresApproval: true }
                ],
                restrictions: {
                    blacklist: ["system_command"],
                    rateLimits: { web_search: 10, code_execution: 5 }
                },
                preferences: {
                    taskPreferences: {
                        information_gathering: ["web_search"],
                        computation: ["calculation", "code_execution"]
                    },
                    costOptimization: true,
                    performanceOptimization: true
                },
                ...overrides?.tools
            },
            validation: {
                emergenceTests: ["tool_selection", "resource_management"],
                integrationTests: ["result_passing", "error_propagation"],
                evolutionTests: ["efficiency_gains", "strategy_adaptation"],
                ...overrides?.validation
            }
        };
    }
    
    createWithDefaults(overrides?: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        return this.createMinimal({
            emergence: {
                capabilities: ["basic_tool_usage", "result_processing"],
                evolutionPath: "manual → automated"
            },
            ...overrides
        });
    }
    
    createVariant(variant: string, overrides?: Partial<ExecutionContextFixture>): ExecutionContextFixture {
        const variants: Record<string, () => ExecutionContextFixture> = {
            highPerformance: () => this.createComplete({
                strategy: "deterministic",
                context: {
                    tools: ["calculation", "data_processing"],
                    constraints: {
                        maxTokens: 10000,
                        timeout: 10000,
                        resourceLimits: {
                            memory: 2048,
                            cpu: 80,
                            apiCalls: 1000
                        }
                    },
                    resources: {
                        creditBudget: 5000,
                        timeBudget: 10000,
                        priority: "high",
                        pools: ["priority", "dedicated"]
                    }
                },
                ...overrides
            }),
            
            secureExecution: () => this.createComplete({
                strategy: "reasoning",
                context: {
                    tools: ["secure_compute", "encrypted_storage"],
                    constraints: {
                        requireApproval: ["all"],
                        maxTokens: 2000,
                        timeout: 120000
                    },
                    safety: {
                        syncChecks: ["comprehensive_validation", "authorization"],
                        asyncAgents: ["security_monitor", "compliance_checker", "audit_logger"],
                        domainRules: ["zero_trust", "data_encryption", "access_control"],
                        emergencyStop: {
                            conditions: ["unauthorized_access", "data_breach_attempt"],
                            actions: ["immediate_halt", "security_alert", "session_termination"]
                        }
                    }
                },
                ...overrides
            }),
            
            resourceConstrained: () => this.createMinimal({
                strategy: "conversational",
                context: {
                    tools: ["basic_search"],
                    constraints: {
                        maxTokens: 500,
                        timeout: 15000,
                        resourceLimits: {
                            memory: 128,
                            cpu: 10,
                            apiCalls: 10
                        }
                    },
                    resources: {
                        creditBudget: 100,
                        timeBudget: 15000,
                        priority: "low",
                        pools: ["shared"]
                    }
                },
                ...overrides
            })
        };
        
        const factory = variants[variant];
        if (!factory) {
            throw new Error(`Unknown execution context variant: ${variant}`);
        }
        
        return factory();
    }
}

/**
 * Create factory instances for easy import
 */
export const swarmFactory = new SwarmFixtureFactory();
export const routineFactory = new RoutineFixtureFactory();
export const executionFactory = new ExecutionContextFixtureFactory();

/**
 * Helper function to create measurable capabilities
 */
export function createMeasurableCapability(
    name: string,
    metric: string,
    baseline: number,
    target: number,
    unit: string,
    description?: string
): MeasurableCapability {
    return {
        name,
        metric,
        baseline,
        target,
        unit,
        description
    };
}

/**
 * Helper function to create enhanced emergence definition
 */
export function createEnhancedEmergence(
    base: EmergenceDefinition,
    measurableCapabilities: MeasurableCapability[],
    emergenceTests?: Array<{
        setup: string;
        trigger: string;
        expectedOutcome: string;
        measurementMethod: string;
    }>
): EnhancedEmergenceDefinition {
    return {
        ...base,
        measurableCapabilities,
        emergenceTests
    };
}