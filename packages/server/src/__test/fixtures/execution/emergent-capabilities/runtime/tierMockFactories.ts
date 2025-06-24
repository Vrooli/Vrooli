/**
 * Tier-Specific Mock Factories
 * 
 * Creates AI mock behaviors tailored to each tier of the execution architecture.
 * These mocks simulate real AI responses while maintaining deterministic testing.
 */

import { type AIMockConfig } from "../../ai-mocks/types.js";

/**
 * Factory methods for creating tier-specific mock behaviors
 */
export const TierMockFactories = {
    /**
     * Tier 1: Swarm Coordination Mocks
     * Simulates multi-agent coordination and collective intelligence
     */
    tier1: {
        /**
         * Create mocks for swarm coordination behaviors
         */
        createSwarmCoordinationMocks(swarmType: string): Map<string, AIMockConfig> {
            const mocks = new Map<string, AIMockConfig>();
            
            // Task delegation mock
            mocks.set("task_delegation", {
                pattern: /delegate.*task|assign.*work|distribute.*workload/i,
                content: "Task optimally assigned based on agent capabilities",
                toolCalls: [{
                    name: "assignTask",
                    arguments: {
                        agentId: "agent_specialized_123",
                        taskId: "task_456",
                        reason: "Best match for required skills: data analysis, pattern recognition",
                        confidence: 0.92
                    }
                }],
                metadata: {
                    tier: "tier1",
                    capability: "intelligent_delegation",
                    confidence: 0.9,
                    agentsInvolved: 3,
                    decisionFactors: ["skill_match", "availability", "past_performance"]
                }
            });
            
            // Collective intelligence mock
            mocks.set("collective_intelligence", {
                pattern: /synthesize.*insights|combine.*knowledge|aggregate.*findings/i,
                content: "Synthesized insights from multiple agent perspectives",
                reasoning: `Agent1 identified Pattern A: recurring customer issues with checkout
Agent2 identified Pattern B: peak failure times correlate with high traffic
Agent3 identified Correlation: database connection pool exhaustion during peaks
Synthesis: Implement dynamic connection pooling to prevent checkout failures`,
                metadata: {
                    tier: "tier1",
                    capability: "knowledge_synthesis",
                    contributingAgents: 3,
                    confidenceLevel: 0.88,
                    emergentInsight: true
                }
            });
            
            // Swarm formation mock
            mocks.set("swarm_formation", {
                pattern: /form.*swarm|create.*team|assemble.*agents/i,
                content: "Dynamic swarm formed based on task requirements",
                toolCalls: [{
                    name: "formSwarm",
                    arguments: {
                        swarmId: `swarm_${swarmType}_${Date.now()}`,
                        agents: [
                            { role: "coordinator", agentId: "coord_001" },
                            { role: "analyzer", agentId: "analyst_002" },
                            { role: "executor", agentId: "exec_003" }
                        ],
                        formation: "hierarchical",
                        objective: swarmType
                    }
                }],
                metadata: {
                    tier: "tier1",
                    capability: "dynamic_team_formation",
                    formationStrategy: "skill_based",
                    expectedDuration: "15-30 minutes"
                }
            });
            
            // Consensus building mock
            mocks.set("consensus_building", {
                pattern: /reach.*consensus|agree.*on|vote.*for/i,
                content: "Consensus reached through weighted voting",
                reasoning: "3 out of 4 agents agree on approach, with confidence weights applied",
                metadata: {
                    tier: "tier1",
                    capability: "democratic_decision_making",
                    votingMethod: "weighted_confidence",
                    consensusStrength: 0.75
                }
            });
            
            // Load balancing mock
            mocks.set("load_balancing", {
                pattern: /balance.*load|distribute.*work|optimize.*allocation/i,
                content: "Workload optimally distributed across available agents",
                toolCalls: [{
                    name: "redistributeWork",
                    arguments: {
                        currentLoad: { agent1: 0.8, agent2: 0.3, agent3: 0.9 },
                        newDistribution: { agent1: 0.6, agent2: 0.6, agent3: 0.6 },
                        strategy: "equal_distribution"
                    }
                }],
                metadata: {
                    tier: "tier1",
                    capability: "adaptive_load_balancing",
                    efficiency: 0.94
                }
            });
            
            return mocks;
        },
        
        /**
         * Create domain-specific swarm mocks
         */
        createDomainSwarmMocks(domain: string): Map<string, AIMockConfig> {
            const mocks = new Map<string, AIMockConfig>();
            
            switch (domain) {
                case "customerSupport":
                    mocks.set("customer_empathy", {
                        pattern: /understand.*customer|empathize|frustrated/i,
                        content: "I understand your frustration. Let me help resolve this immediately.",
                        metadata: {
                            tier: "tier1",
                            capability: "customer_satisfaction",
                            sentimentDetected: "negative",
                            responseStrategy: "empathetic_solution"
                        }
                    });
                    
                    mocks.set("issue_resolution", {
                        pattern: /resolve.*issue|fix.*problem|solution/i,
                        content: "I've identified the root cause and have a solution ready.",
                        toolCalls: [{
                            name: "applyFix",
                            arguments: { fixType: "automatic", confidence: 0.95 }
                        }],
                        metadata: {
                            tier: "tier1",
                            capability: "issue_resolution",
                            resolutionTime: 120
                        }
                    });
                    break;
                    
                case "security":
                    mocks.set("threat_detection", {
                        pattern: /detect.*threat|identify.*anomaly|security.*breach/i,
                        content: "Anomalous activity detected - potential security threat identified",
                        toolCalls: [{
                            name: "raiseAlert",
                            arguments: {
                                threatLevel: "medium",
                                threatType: "unauthorized_access_attempt",
                                confidence: 0.87
                            }
                        }],
                        metadata: {
                            tier: "tier1",
                            capability: "threat_detection",
                            responseTime: 50
                        }
                    });
                    
                    mocks.set("incident_response", {
                        pattern: /respond.*incident|contain.*threat|mitigate/i,
                        content: "Initiating incident response protocol",
                        reasoning: "Threat isolated to subnet 192.168.1.x, implementing containment",
                        metadata: {
                            tier: "tier1",
                            capability: "automated_response",
                            containmentStrategy: "network_isolation"
                        }
                    });
                    break;
                    
                case "research":
                    mocks.set("pattern_discovery", {
                        pattern: /discover.*pattern|find.*correlation|analyze.*data/i,
                        content: "Discovered significant pattern in dataset",
                        reasoning: "Statistical analysis reveals p<0.01 correlation between variables X and Y",
                        metadata: {
                            tier: "tier1",
                            capability: "pattern_discovery",
                            statisticalSignificance: 0.99,
                            dataPointsAnalyzed: 10000
                        }
                    });
                    break;
            }
            
            return mocks;
        }
    },
    
    /**
     * Tier 2: Routine Evolution Mocks
     * Simulates the evolution from conversational to deterministic execution
     */
    tier2: {
        /**
         * Create mocks for routine evolution stages
         */
        createRoutineEvolutionMocks(routineType: string): Map<string, AIMockConfig> {
            const mocks = new Map<string, AIMockConfig>();
            
            // Conversational stage - flexible but slower
            mocks.set("routine_conversational", {
                pattern: new RegExp(`${routineType}.*conversational|flexible.*approach`, "i"),
                content: "Let me help you with that. I'll need to understand your specific needs first...",
                reasoning: "Using conversational approach for maximum flexibility and context awareness",
                metadata: {
                    tier: "tier2",
                    stage: "conversational",
                    executionTime: 8000,
                    accuracy: 0.85,
                    cost: 0.05,
                    characteristics: ["flexible", "context_aware", "exploratory"]
                },
                streamConfig: {
                    enabled: true,
                    chunkDelayMs: 100
                }
            });
            
            // Reasoning stage - structured analysis
            mocks.set("routine_reasoning", {
                pattern: new RegExp(`${routineType}.*reasoning|structured.*analysis`, "i"),
                content: "Analyzing your request using structured reasoning...",
                reasoning: `Step 1: Identify key requirements
Step 2: Evaluate available options
Step 3: Select optimal approach based on constraints
Step 4: Execute with confidence`,
                metadata: {
                    tier: "tier2",
                    stage: "reasoning",
                    executionTime: 4000,
                    accuracy: 0.92,
                    cost: 0.03,
                    characteristics: ["analytical", "structured", "systematic"]
                }
            });
            
            // Deterministic stage - optimized execution
            mocks.set("routine_deterministic", {
                pattern: new RegExp(`${routineType}.*deterministic|optimized.*execution`, "i"),
                content: "Executing optimized solution",
                metadata: {
                    tier: "tier2",
                    stage: "deterministic",
                    executionTime: 500,
                    accuracy: 0.98,
                    cost: 0.01,
                    characteristics: ["fast", "predictable", "cacheable", "efficient"]
                },
                cached: true
            });
            
            // Routing stage - intelligent path selection
            mocks.set("routine_routing", {
                pattern: /route.*request|select.*path|choose.*strategy/i,
                content: "Optimal execution path selected",
                toolCalls: [{
                    name: "selectRoute",
                    arguments: {
                        selectedPath: "optimized_path_v2",
                        reason: "Historical success rate: 97%",
                        alternativePaths: ["standard_path", "fallback_path"]
                    }
                }],
                metadata: {
                    tier: "tier2",
                    capability: "intelligent_routing",
                    decisionTime: 100
                }
            });
            
            return mocks;
        },
        
        /**
         * Create workflow-specific mocks
         */
        createWorkflowMocks(workflowType: string): Map<string, AIMockConfig> {
            const mocks = new Map<string, AIMockConfig>();
            
            // State management
            mocks.set("state_management", {
                pattern: /manage.*state|track.*progress|checkpoint/i,
                content: "Workflow state updated and persisted",
                metadata: {
                    tier: "tier2",
                    capability: "stateful_execution",
                    stateTransition: "processing → completed",
                    checkpoint: true
                }
            });
            
            // Error recovery
            mocks.set("error_recovery", {
                pattern: /recover.*error|handle.*failure|retry/i,
                content: "Error detected and recovered using fallback strategy",
                reasoning: "Primary path failed, successfully executed fallback",
                metadata: {
                    tier: "tier2",
                    capability: "adaptive_recovery",
                    recoveryStrategy: "exponential_backoff",
                    attemptNumber: 2
                }
            });
            
            // Parallel execution
            mocks.set("parallel_execution", {
                pattern: /parallel.*execution|concurrent.*processing|batch/i,
                content: "Executing tasks in parallel for optimal performance",
                toolCalls: [
                    { name: "executeParallel", arguments: { taskCount: 5, strategy: "fork_join" } }
                ],
                metadata: {
                    tier: "tier2",
                    capability: "parallel_processing",
                    speedup: 3.5
                }
            });
            
            return mocks;
        }
    },
    
    /**
     * Tier 3: Execution Strategy Mocks
     * Simulates low-level execution and tool orchestration
     */
    tier3: {
        /**
         * Create execution strategy mocks
         */
        createExecutionStrategyMocks(): Map<string, AIMockConfig> {
            const mocks = new Map<string, AIMockConfig>();
            
            // Tool orchestration
            mocks.set("tool_orchestration", {
                pattern: /execute.*tools|orchestrate.*operations|coordinate.*execution/i,
                content: "Orchestrating multi-tool execution pipeline",
                toolCalls: [
                    { 
                        name: "validateInput", 
                        arguments: { schema: "input_schema_v2", strict: true } 
                    },
                    { 
                        name: "transformData", 
                        arguments: { format: "normalized", parallel: true } 
                    },
                    { 
                        name: "generateOutput", 
                        arguments: { format: "json", compress: true } 
                    }
                ],
                metadata: {
                    tier: "tier3",
                    capability: "intelligent_orchestration",
                    toolsUsed: 3,
                    executionPlan: "pipeline",
                    optimizations: ["parallel_transform", "output_compression"]
                }
            });
            
            // Strategy selection
            mocks.set("strategy_selection", {
                pattern: /select.*strategy|choose.*approach|determine.*method/i,
                content: "Optimal execution strategy selected based on context",
                reasoning: "Input size < 1MB and complexity is low, using fast-path strategy",
                metadata: {
                    tier: "tier3",
                    capability: "adaptive_strategy_selection",
                    selectedStrategy: "fast_path",
                    selectionCriteria: ["input_size", "complexity", "resource_availability"]
                }
            });
            
            // Resource optimization
            mocks.set("resource_optimization", {
                pattern: /optimize.*resources|manage.*memory|efficient.*execution/i,
                content: "Resources optimized for efficient execution",
                metadata: {
                    tier: "tier3",
                    capability: "resource_management",
                    memoryUsage: "45MB",
                    cpuUtilization: "23%",
                    optimizations: ["lazy_loading", "stream_processing"]
                }
            });
            
            // Caching behavior
            mocks.set("intelligent_caching", {
                pattern: /cache.*result|store.*output|reuse.*computation/i,
                content: "Result cached for future reuse",
                metadata: {
                    tier: "tier3",
                    capability: "intelligent_caching",
                    cacheKey: "execution_hash_abc123",
                    ttl: 3600,
                    hitRate: 0.78
                }
            });
            
            // Error handling
            mocks.set("error_handling", {
                pattern: /handle.*error|catch.*exception|recover.*failure/i,
                content: "Error gracefully handled with fallback execution",
                reasoning: "Primary execution failed due to timeout, switching to degraded mode",
                metadata: {
                    tier: "tier3",
                    capability: "resilient_execution",
                    errorType: "timeout",
                    fallbackStrategy: "degraded_mode",
                    recoveryTime: 150
                }
            });
            
            return mocks;
        },
        
        /**
         * Create performance-specific mocks
         */
        createPerformanceMocks(performanceProfile: string): Map<string, AIMockConfig> {
            const mocks = new Map<string, AIMockConfig>();
            
            switch (performanceProfile) {
                case "highPerformance":
                    mocks.set("ultra_fast_execution", {
                        pattern: /execute.*fast|high.*performance|optimize.*speed/i,
                        content: "Executed with maximum performance optimizations",
                        metadata: {
                            tier: "tier3",
                            profile: "high_performance",
                            executionTime: 50,
                            optimizations: ["simd", "cache_aligned", "zero_copy"]
                        }
                    });
                    break;
                    
                case "resourceConstrained":
                    mocks.set("memory_efficient", {
                        pattern: /conserve.*memory|limited.*resources|efficient/i,
                        content: "Executed within resource constraints",
                        metadata: {
                            tier: "tier3",
                            profile: "resource_constrained",
                            memoryUsage: "10MB",
                            techniques: ["streaming", "incremental_processing"]
                        }
                    });
                    break;
                    
                case "balanced":
                    mocks.set("balanced_execution", {
                        pattern: /balanced|moderate|standard/i,
                        content: "Executed with balanced resource usage",
                        metadata: {
                            tier: "tier3",
                            profile: "balanced",
                            tradeoff: "performance_vs_resources"
                        }
                    });
                    break;
            }
            
            return mocks;
        }
    },
    
    /**
     * Cross-Tier Integration Mocks
     * Simulates communication and coordination across all tiers
     */
    crossTier: {
        /**
         * Create cross-tier coordination mocks
         */
        createIntegrationMocks(integrationType: string): Map<string, AIMockConfig> {
            const mocks = new Map<string, AIMockConfig>();
            
            // End-to-end coordination
            mocks.set("end_to_end_coordination", {
                pattern: /coordinate.*across.*tiers|end.*to.*end|full.*stack/i,
                content: "Coordinating execution across all three tiers",
                reasoning: `Tier 1: Task identified and delegated to specialized agents
Tier 2: Workflow selected and optimized for execution
Tier 3: Tools orchestrated for efficient processing`,
                metadata: {
                    tiers: ["tier1", "tier2", "tier3"],
                    capability: "cross_tier_coordination",
                    latency: {
                        tier1: 100,
                        tier2: 200,
                        tier3: 150,
                        total: 450
                    }
                }
            });
            
            // Event propagation
            mocks.set("event_propagation", {
                pattern: /propagate.*event|broadcast.*update|notify.*tiers/i,
                content: "Event propagated across tier boundaries",
                toolCalls: [{
                    name: "broadcastEvent",
                    arguments: {
                        event: "capability_emerged",
                        source: "tier2",
                        targets: ["tier1", "tier3"],
                        payload: { capability: "new_optimization", confidence: 0.9 }
                    }
                }],
                metadata: {
                    capability: "event_driven_architecture",
                    propagationTime: 25
                }
            });
            
            // Feedback loops
            mocks.set("feedback_integration", {
                pattern: /feedback.*loop|learning.*propagation|improvement.*sharing/i,
                content: "Feedback integrated across tiers for continuous improvement",
                reasoning: "Tier 3 performance metrics fed back to Tier 1 for better task allocation",
                metadata: {
                    capability: "continuous_learning",
                    feedbackType: "performance_metrics",
                    improvementRate: 0.15
                }
            });
            
            // Emergent capability detection
            mocks.set("emergent_capability_detection", {
                pattern: /detect.*emergence|new.*capability|evolved.*behavior/i,
                content: "New emergent capability detected through tier interaction",
                reasoning: `Unexpected synergy detected:
- Tier 1 delegation patterns + 
- Tier 2 workflow optimization + 
- Tier 3 caching strategies = 
- New capability: predictive_resource_allocation`,
                metadata: {
                    capability: "emergence_detection",
                    emergentCapability: "predictive_resource_allocation",
                    confidenceScore: 0.82,
                    evidenceSources: ["pattern_analysis", "performance_metrics", "behavior_correlation"]
                }
            });
            
            return mocks;
        }
    },
    
    /**
     * Utility function to create progressive learning mocks
     */
    createProgressiveLearningMocks(
        capability: string,
        stages: Array<{ iteration: number; confidence: number; improvements: string[] }>
    ): Map<string, AIMockConfig> {
        const mocks = new Map<string, AIMockConfig>();
        
        stages.forEach((stage, index) => {
            mocks.set(`${capability}_stage_${index}`, {
                pattern: new RegExp(`${capability}.*iteration.*${stage.iteration}`, "i"),
                content: `Capability ${capability} at iteration ${stage.iteration}`,
                reasoning: `Learning progress: ${stage.improvements.join(", ")}`,
                metadata: {
                    capability,
                    iteration: stage.iteration,
                    confidence: stage.confidence,
                    improvements: stage.improvements,
                    learningStage: index === 0 ? "novice" : index === stages.length - 1 ? "expert" : "intermediate"
                },
                confidence: stage.confidence,
                maxUses: 1 // Ensure progression
            });
        });
        
        return mocks;
    },
    
    /**
     * Create error simulation mocks
     */
    createErrorSimulationMocks(): Map<string, AIMockConfig> {
        const mocks = new Map<string, AIMockConfig>();
        
        // Network failure
        mocks.set("network_failure", {
            pattern: /simulate.*network.*failure|test.*connectivity/i,
            error: {
                type: "network_error",
                message: "Connection timeout",
                code: "ETIMEDOUT",
                retryable: true
            }
        });
        
        // Resource exhaustion
        mocks.set("resource_exhaustion", {
            pattern: /simulate.*resource.*exhaustion|out.*of.*memory/i,
            error: {
                type: "resource_error",
                message: "Insufficient memory",
                code: "ENOMEM",
                retryable: false
            }
        });
        
        // Rate limiting
        mocks.set("rate_limit", {
            pattern: /simulate.*rate.*limit|too.*many.*requests/i,
            error: {
                type: "rate_limit",
                message: "Rate limit exceeded",
                code: 429,
                retryAfter: 60
            }
        });
        
        return mocks;
    }
};