# AI Mock Integration Guide for Emergent Capabilities Testing

This guide demonstrates how to use AI mock fixtures to test emergent capabilities in Vrooli's three-tier execution architecture. The mocks enable predictable testing of agent learning, capability evolution, and swarm coordination without requiring real AI models.

## 🎯 Overview

AI mocks provide controlled, predictable responses that allow you to:
- Test agent learning patterns with deterministic outcomes
- Validate capability evolution through defined progression paths
- Verify swarm coordination with synchronized mock behaviors
- Ensure resilience testing with simulated failures and recovery

## 🧠 Core Concepts

### Mock Registry Architecture

The `MockRegistry` is a singleton service that manages all AI mock behaviors:

```typescript
import { getMockRegistry, registerMockBehavior } from "../ai-mocks/integration/mockRegistry.js";

// Register a behavior for testing agent learning
registerMockBehavior("learning_agent", {
    pattern: /analyze.*patterns/i,
    response: {
        content: "I've identified 3 optimization opportunities",
        reasoning: "Based on historical data analysis...",
        confidence: 0.85,
        metadata: {
            learningIteration: 1,
            patternsIdentified: ["memory_leak", "slow_query", "cache_miss"]
        }
    },
    priority: 10
});
```

### Behavior Patterns

Mock behaviors can match requests using:
- **RegExp patterns**: Match against message content
- **Function patterns**: Custom logic for complex matching
- **Priority ordering**: Higher priority behaviors match first

## 📚 Examples

### 1. Testing Agent Learning with Mocked AI Responses

This example shows how to test an agent that learns from performance patterns:

```typescript
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { getMockRegistry, clearAllMockBehaviors } from "../ai-mocks/integration/mockRegistry.js";
import { createTestEvent } from "../emergent-capabilities/agent-types/emergentAgentFixtures.js";
import { AgentDeploymentService } from "../../../../services/execution/cross-cutting/agents/agentDeploymentService.js";

describe("Agent Learning with AI Mocks", () => {
    let mockRegistry;
    let deploymentService: AgentDeploymentService;
    
    beforeEach(() => {
        mockRegistry = getMockRegistry();
        mockRegistry.setDebugMode(true);
        
        // Register progressive learning behaviors
        registerLearningBehaviors();
        
        deploymentService = new AgentDeploymentService(logger);
    });
    
    afterEach(() => {
        clearAllMockBehaviors();
    });
    
    function registerLearningBehaviors() {
        // Initial pattern recognition
        mockRegistry.registerBehavior("pattern_recognition_v1", {
            pattern: /analyze performance.*first time/i,
            response: {
                content: "Initial analysis complete. Found basic patterns.",
                reasoning: "First exposure to data shows common performance issues",
                confidence: 0.6,
                metadata: {
                    iteration: 1,
                    patternsFound: 2,
                    learningStage: "novice"
                }
            },
            maxUses: 1  // Only use once to simulate learning progression
        });
        
        // Improved pattern recognition after learning
        mockRegistry.registerBehavior("pattern_recognition_v2", {
            pattern: /analyze performance.*learned/i,
            response: {
                content: "Advanced analysis reveals deeper insights.",
                reasoning: "With accumulated knowledge, I can identify complex patterns",
                confidence: 0.9,
                metadata: {
                    iteration: 5,
                    patternsFound: 7,
                    learningStage: "expert",
                    newCapabilities: ["predictive_analysis", "root_cause_detection"]
                }
            }
        });
    }
    
    it("should demonstrate learning progression through multiple iterations", async () => {
        // Deploy performance analysis agent
        const agentId = await deploymentService.deployAgent({
            agentId: "perf_learner_001",
            goal: "Learn and improve performance analysis capabilities",
            initialRoutine: "analyze_performance_patterns",
            subscriptions: ["performance/*", "learning/*"]
        });
        
        // First analysis - novice level
        const event1 = createTestEvent("performance.degradation", {
            context: "analyze performance for first time"
        });
        
        const result1 = await deploymentService.processEvent(agentId, event1);
        expect(result1.metadata.learningStage).toBe("novice");
        expect(result1.confidence).toBe(0.6);
        
        // Simulate learning events
        for (let i = 0; i < 3; i++) {
            await deploymentService.processEvent(agentId, 
                createTestEvent("learning.iteration", { iteration: i + 2 })
            );
        }
        
        // Later analysis - expert level
        const event2 = createTestEvent("performance.degradation", {
            context: "analyze performance with learned knowledge"
        });
        
        const result2 = await deploymentService.processEvent(agentId, event2);
        expect(result2.metadata.learningStage).toBe("expert");
        expect(result2.confidence).toBe(0.9);
        expect(result2.metadata.newCapabilities).toContain("predictive_analysis");
        
        // Verify learning metrics
        const stats = mockRegistry.getStats();
        expect(stats.behaviorHits["pattern_recognition_v1"]).toBe(1);
        expect(stats.behaviorHits["pattern_recognition_v2"]).toBeGreaterThan(0);
    });
});
```

### 2. Testing Capability Evolution Using Predictable AI Behaviors

This example demonstrates testing how capabilities evolve from reactive to proactive:

```typescript
describe("Capability Evolution with Predictable AI", () => {
    let evolutionTracker;
    
    beforeEach(() => {
        evolutionTracker = new Map();
        registerEvolutionBehaviors();
    });
    
    function registerEvolutionBehaviors() {
        // Stage 1: Reactive capability
        mockRegistry.registerBehavior("reactive_stage", {
            pattern: /security threat detected/i,
            response: {
                content: "Threat detected. Alerting administrators.",
                reasoning: "Following predefined alert protocol",
                confidence: 0.7,
                metadata: {
                    evolutionStage: "reactive",
                    responseTime: 500,
                    capabilities: ["detect", "alert"]
                }
            }
        });
        
        // Stage 2: Proactive capability
        mockRegistry.registerBehavior("proactive_stage", {
            pattern: /analyze threat patterns/i,
            response: {
                content: "Identified threat pattern. Implementing preventive measures.",
                reasoning: "Historical analysis shows this pattern precedes attacks",
                confidence: 0.85,
                metadata: {
                    evolutionStage: "proactive",
                    responseTime: 200,
                    capabilities: ["detect", "alert", "prevent", "analyze_patterns"]
                }
            }
        });
        
        // Stage 3: Predictive capability
        mockRegistry.registerBehavior("predictive_stage", {
            pattern: /predict future threats/i,
            response: {
                content: "ML model predicts 87% chance of DDoS attempt in next 2 hours.",
                reasoning: "Multiple indicators correlate with historical attack patterns",
                confidence: 0.92,
                metadata: {
                    evolutionStage: "predictive",
                    responseTime: 100,
                    capabilities: ["detect", "alert", "prevent", "analyze_patterns", 
                                 "predict", "preemptive_mitigation"],
                    predictions: [
                        { threat: "ddos", probability: 0.87, timeframe: "2h" },
                        { threat: "sql_injection", probability: 0.23, timeframe: "24h" }
                    ]
                }
            }
        });
    }
    
    it("should evolve from reactive to predictive capabilities", async () => {
        const securityAgent = await deploymentService.deployAgent({
            agentId: "security_evolution_001",
            goal: "Evolve security capabilities through experience",
            initialRoutine: "security_monitoring",
            subscriptions: ["security/*", "threat/*"]
        });
        
        // Test reactive stage
        const reactiveEvent = createTestEvent("threat.detected", {
            message: "Security threat detected in API endpoint"
        });
        const reactiveResult = await deploymentService.processEvent(
            securityAgent, 
            reactiveEvent
        );
        
        expect(reactiveResult.metadata.evolutionStage).toBe("reactive");
        expect(reactiveResult.metadata.capabilities).toHaveLength(2);
        evolutionTracker.set("reactive", reactiveResult);
        
        // Simulate learning through multiple threat encounters
        for (let i = 0; i < 10; i++) {
            await deploymentService.processEvent(
                securityAgent,
                createTestEvent("threat.analyzed", { 
                    threatId: `threat_${i}`,
                    pattern: "sql_injection"
                })
            );
        }
        
        // Test proactive stage
        const proactiveEvent = createTestEvent("system.analysis", {
            message: "Analyze threat patterns for prevention"
        });
        const proactiveResult = await deploymentService.processEvent(
            securityAgent,
            proactiveEvent
        );
        
        expect(proactiveResult.metadata.evolutionStage).toBe("proactive");
        expect(proactiveResult.metadata.capabilities).toContain("prevent");
        expect(proactiveResult.metadata.responseTime).toBeLessThan(
            reactiveResult.metadata.responseTime
        );
        evolutionTracker.set("proactive", proactiveResult);
        
        // Further learning with pattern analysis
        await deploymentService.updateAgentCapabilities(securityAgent, {
            patternRecognition: true,
            historicalAnalysis: true,
            mlModelTrained: true
        });
        
        // Test predictive stage
        const predictiveEvent = createTestEvent("system.forecast", {
            message: "Predict future threats based on current indicators"
        });
        const predictiveResult = await deploymentService.processEvent(
            securityAgent,
            predictiveEvent
        );
        
        expect(predictiveResult.metadata.evolutionStage).toBe("predictive");
        expect(predictiveResult.metadata.capabilities).toContain("predict");
        expect(predictiveResult.metadata.predictions).toBeDefined();
        expect(predictiveResult.confidence).toBeGreaterThan(0.9);
        
        // Verify evolution progression
        verifyEvolutionProgression(evolutionTracker);
    });
    
    function verifyEvolutionProgression(tracker: Map<string, any>) {
        const stages = ["reactive", "proactive", "predictive"];
        let previousCapabilityCount = 0;
        let previousResponseTime = Infinity;
        let previousConfidence = 0;
        
        for (const stage of stages) {
            const result = tracker.get(stage);
            expect(result).toBeDefined();
            
            // Capabilities should increase
            expect(result.metadata.capabilities.length).toBeGreaterThan(
                previousCapabilityCount
            );
            previousCapabilityCount = result.metadata.capabilities.length;
            
            // Response time should decrease
            expect(result.metadata.responseTime).toBeLessThan(previousResponseTime);
            previousResponseTime = result.metadata.responseTime;
            
            // Confidence should increase
            expect(result.confidence).toBeGreaterThan(previousConfidence);
            previousConfidence = result.confidence;
        }
    }
});
```

### 3. Testing Swarm Coordination with Mocked AI

This example shows how to test multi-agent swarms with synchronized mock behaviors:

```typescript
describe("Swarm Coordination with AI Mocks", () => {
    let swarmCoordinator;
    
    beforeEach(() => {
        swarmCoordinator = new SwarmCoordinator(logger);
        registerSwarmBehaviors();
    });
    
    function registerSwarmBehaviors() {
        // Leader agent behavior
        mockRegistry.registerBehavior("swarm_leader", {
            pattern: /coordinate.*agents.*task/i,
            response: {
                content: "Task decomposed into 3 subtasks. Assigning to specialized agents.",
                toolCalls: [
                    {
                        name: "assign_task",
                        arguments: {
                            agentId: "analyzer_001",
                            task: "analyze_data_patterns"
                        }
                    },
                    {
                        name: "assign_task",
                        arguments: {
                            agentId: "optimizer_001",
                            task: "optimize_performance"
                        }
                    },
                    {
                        name: "assign_task",
                        arguments: {
                            agentId: "validator_001",
                            task: "validate_results"
                        }
                    }
                ],
                metadata: {
                    role: "coordinator",
                    decomposition: {
                        originalTask: "complex_analysis",
                        subtasks: 3,
                        estimatedTime: "15m"
                    }
                }
            }
        });
        
        // Specialized agent behaviors
        mockRegistry.registerBehavior("analyzer_agent", {
            pattern: /analyze_data_patterns/i,
            response: {
                content: "Pattern analysis complete. Found 5 significant correlations.",
                metadata: {
                    role: "analyzer",
                    findings: {
                        correlations: 5,
                        anomalies: 2,
                        confidence: 0.88
                    }
                },
                delay: 2000  // Simulate processing time
            }
        });
        
        mockRegistry.registerBehavior("optimizer_agent", {
            pattern: /optimize_performance/i,
            response: {
                content: "Optimization complete. Performance improved by 34%.",
                metadata: {
                    role: "optimizer",
                    improvements: {
                        performanceGain: 0.34,
                        memoryReduction: 0.21,
                        latencyReduction: 0.45
                    }
                },
                delay: 3000
            }
        });
        
        mockRegistry.registerBehavior("validator_agent", {
            pattern: /validate_results/i,
            response: {
                content: "Validation successful. All results meet quality thresholds.",
                metadata: {
                    role: "validator",
                    validation: {
                        passed: true,
                        accuracyScore: 0.92,
                        completenessScore: 0.95
                    }
                },
                delay: 1000
            }
        });
        
        // Consensus behavior
        mockRegistry.registerBehavior("swarm_consensus", {
            pattern: /reach consensus.*results/i,
            response: {
                content: "Swarm consensus achieved. Recommended action: Deploy optimization.",
                reasoning: "All agents agree on the optimization strategy with high confidence",
                metadata: {
                    consensusType: "unanimous",
                    participatingAgents: 4,
                    confidenceScores: [0.88, 0.91, 0.92, 0.89],
                    decision: "deploy_optimization"
                }
            }
        });
    }
    
    it("should coordinate multiple agents to solve complex task", async () => {
        // Create customer support swarm
        const swarmId = await swarmCoordinator.createSwarm({
            swarmId: "support_swarm_001",
            goal: "Provide comprehensive customer support",
            agents: [
                { agentId: "leader_001", role: "coordinator" },
                { agentId: "analyzer_001", role: "analyzer" },
                { agentId: "optimizer_001", role: "optimizer" },
                { agentId: "validator_001", role: "validator" }
            ],
            coordinationPattern: "hierarchical"
        });
        
        // Submit complex task to swarm
        const complexTask = createTestEvent("task.submitted", {
            message: "Coordinate agents to solve task: analyze customer complaints and optimize response strategy",
            taskComplexity: "high",
            requiredCapabilities: ["analysis", "optimization", "validation"]
        });
        
        const coordinationResult = await swarmCoordinator.processTask(
            swarmId,
            complexTask
        );
        
        // Verify task decomposition
        expect(coordinationResult.subtasks).toHaveLength(3);
        expect(coordinationResult.metadata.role).toBe("coordinator");
        
        // Wait for parallel agent execution
        const agentResults = await swarmCoordinator.waitForAgentResults(swarmId);
        
        // Verify each agent completed their task
        const analyzerResult = agentResults.find(r => r.metadata.role === "analyzer");
        expect(analyzerResult.metadata.findings.correlations).toBe(5);
        
        const optimizerResult = agentResults.find(r => r.metadata.role === "optimizer");
        expect(optimizerResult.metadata.improvements.performanceGain).toBe(0.34);
        
        const validatorResult = agentResults.find(r => r.metadata.role === "validator");
        expect(validatorResult.metadata.validation.passed).toBe(true);
        
        // Test consensus mechanism
        const consensusEvent = createTestEvent("swarm.consensus", {
            message: "Reach consensus on results from all agents",
            results: agentResults
        });
        
        const consensusResult = await swarmCoordinator.reachConsensus(
            swarmId,
            consensusEvent
        );
        
        expect(consensusResult.metadata.consensusType).toBe("unanimous");
        expect(consensusResult.metadata.decision).toBe("deploy_optimization");
        
        // Verify emergent capability
        const emergentCapabilities = await swarmCoordinator.assessEmergentCapabilities(
            swarmId
        );
        
        expect(emergentCapabilities).toContain("complex_problem_solving");
        expect(emergentCapabilities).toContain("collaborative_decision_making");
    });
    
    it("should demonstrate emergent swarm intelligence", async () => {
        // Create research swarm with learning capabilities
        const researchSwarmId = await swarmCoordinator.createSwarm({
            swarmId: "research_swarm_001",
            goal: "Discover new optimization strategies through collaboration",
            agents: [
                { agentId: "researcher_001", capabilities: ["hypothesis_generation"] },
                { agentId: "researcher_002", capabilities: ["experiment_design"] },
                { agentId: "researcher_003", capabilities: ["data_analysis"] },
                { agentId: "researcher_004", capabilities: ["theory_synthesis"] }
            ],
            coordinationPattern: "flat",  // No hierarchy, peer collaboration
            emergenceEnabled: true
        });
        
        // Register behaviors for emergent discovery
        mockRegistry.registerBehavior("emergent_discovery", {
            pattern: /collaborate.*discover.*optimization/i,
            response: (request) => {
                // Simulate emergent behavior based on swarm state
                const swarmState = extractSwarmState(request);
                const emergentInsight = generateEmergentInsight(swarmState);
                
                return {
                    content: `Swarm discovered novel approach: ${emergentInsight.name}`,
                    reasoning: "Through collaborative exploration, agents identified pattern not visible to individuals",
                    metadata: {
                        emergentDiscovery: true,
                        noveltyScore: 0.94,
                        insight: emergentInsight,
                        contributingAgents: swarmState.activeAgents,
                        synergyFactor: 2.3  // Swarm performance vs individual
                    }
                };
            },
            priority: 20
        });
        
        // Run collaborative discovery task
        const discoveryTask = createTestEvent("research.task", {
            message: "Collaborate to discover new optimization strategies",
            explorationBudget: 100,
            diversityTarget: 0.8
        });
        
        const discoveryResult = await swarmCoordinator.processTask(
            researchSwarmId,
            discoveryTask
        );
        
        // Verify emergent discovery
        expect(discoveryResult.metadata.emergentDiscovery).toBe(true);
        expect(discoveryResult.metadata.noveltyScore).toBeGreaterThan(0.9);
        expect(discoveryResult.metadata.synergyFactor).toBeGreaterThan(2.0);
        
        // Track swarm learning over time
        const learningMetrics = await swarmCoordinator.trackSwarmLearning(
            researchSwarmId,
            5  // iterations
        );
        
        expect(learningMetrics.knowledgeGrowth).toBeGreaterThan(1.5);
        expect(learningMetrics.collaborationEfficiency).toBeGreaterThan(0.8);
    });
});

// Helper functions for emergent behavior simulation
function extractSwarmState(request: any) {
    return {
        activeAgents: ["researcher_001", "researcher_002", "researcher_003", "researcher_004"],
        sharedKnowledge: 42,
        iterationCount: 5
    };
}

function generateEmergentInsight(swarmState: any) {
    return {
        name: "Adaptive Resource Pooling",
        description: "Dynamic allocation based on predictive load modeling",
        confidence: 0.94
    };
}
```

### 4. Best Practices for Using AI Mocks in Emergent Capability Tests

```typescript
describe("Best Practices for AI Mock Testing", () => {
    it("should use progressive mock behaviors for learning tests", async () => {
        // DO: Create a sequence of behaviors that show progression
        const learningSequence = [
            {
                id: "learning_stage_1",
                pattern: /iteration: 1/,
                response: { confidence: 0.5, capabilities: ["basic"] },
                maxUses: 1
            },
            {
                id: "learning_stage_2", 
                pattern: /iteration: [2-5]/,
                response: { confidence: 0.7, capabilities: ["basic", "intermediate"] },
                maxUses: 4
            },
            {
                id: "learning_stage_3",
                pattern: /iteration: [6-9]/,
                response: { confidence: 0.9, capabilities: ["basic", "intermediate", "advanced"] }
            }
        ];
        
        learningSequence.forEach(behavior => 
            mockRegistry.registerBehavior(behavior.id, behavior)
        );
        
        // Test shows clear progression
        for (let i = 1; i <= 9; i++) {
            const result = await processLearningIteration(i);
            
            if (i === 1) expect(result.confidence).toBe(0.5);
            else if (i <= 5) expect(result.confidence).toBe(0.7);
            else expect(result.confidence).toBe(0.9);
        }
    });
    
    it("should use deterministic delays for coordination tests", async () => {
        // DO: Use consistent delays to test coordination timing
        mockRegistry.registerBehavior("coordinator", {
            pattern: /coordinate/,
            response: {
                content: "Coordinating agents...",
                delay: 100  // Predictable delay
            }
        });
        
        mockRegistry.registerBehavior("worker", {
            pattern: /execute task/,
            response: {
                content: "Task complete",
                delay: 500  // Longer predictable delay
            }
        });
        
        const start = Date.now();
        const results = await Promise.all([
            coordinatorAction(),
            workerAction()
        ]);
        const duration = Date.now() - start;
        
        // Verify predictable timing
        expect(duration).toBeGreaterThanOrEqual(500);
        expect(duration).toBeLessThan(600);
    });
    
    it("should capture interactions for debugging", async () => {
        // DO: Enable debug mode and capture interactions
        mockRegistry.setDebugMode(true);
        mockRegistry.clearCapturedInteractions();
        
        // Run test scenario
        await runComplexScenario();
        
        // Analyze captured interactions
        const interactions = mockRegistry.getCapturedInteractions();
        
        // Verify expected interaction pattern
        expect(interactions).toHaveLength(5);
        expect(interactions[0].request.messages[0].content).toContain("initialize");
        expect(interactions[4].response.metadata.phase).toBe("complete");
        
        // Use for debugging test failures
        if (testFailed) {
            console.log("Interaction trace:", interactions.map(i => ({
                request: i.request.messages[0].content.substring(0, 50),
                response: i.response.response.content.substring(0, 50),
                metadata: i.response.metadata
            })));
        }
    });
    
    it("should validate mock responses match expected schema", async () => {
        // DO: Validate mock configurations before use
        const mockConfig = {
            content: "Test response",
            toolCalls: [
                {
                    name: "invalid_tool",  // This will fail validation
                    arguments: {}
                }
            ]
        };
        
        // Validation should catch schema issues
        expect(() => {
            mockRegistry.registerBehavior("invalid", {
                response: mockConfig
            });
        }).toThrow(/Invalid mock config/);
    });
    
    it("should use behavior priorities for complex scenarios", async () => {
        // DO: Use priorities to control which mock responds
        mockRegistry.registerBehavior("general_response", {
            pattern: /.*/,  // Matches everything
            response: { content: "General response" },
            priority: 1  // Low priority
        });
        
        mockRegistry.registerBehavior("specific_response", {
            pattern: /urgent/,
            response: { content: "Urgent response" },
            priority: 10  // High priority
        });
        
        const normalResult = await processRequest("normal task");
        expect(normalResult.content).toBe("General response");
        
        const urgentResult = await processRequest("urgent task");
        expect(urgentResult.content).toBe("Urgent response");
    });
});
```

## 🛡️ Error Handling and Resilience Testing

```typescript
describe("Resilience Testing with AI Mocks", () => {
    it("should test graceful degradation with AI failures", async () => {
        // Register failure behavior
        mockRegistry.registerBehavior("ai_failure", {
            pattern: /critical analysis/,
            response: {
                error: "Model overloaded",
                errorType: "capacity_exceeded",
                retryAfter: 5000
            }
        });
        
        // Register fallback behavior
        mockRegistry.registerBehavior("fallback_analysis", {
            pattern: /fallback.*analysis/,
            response: {
                content: "Basic analysis completed with reduced accuracy",
                confidence: 0.6,
                metadata: {
                    fallbackMode: true,
                    degradedCapabilities: ["advanced_reasoning", "prediction"]
                }
            }
        });
        
        const agent = await deployAgent({
            agentId: "resilient_001",
            capabilities: ["auto_fallback", "graceful_degradation"]
        });
        
        // Attempt critical analysis (will fail)
        const criticalResult = await agent.process("Perform critical analysis");
        expect(criticalResult.error).toBe("Model overloaded");
        
        // Agent should automatically fallback
        const fallbackResult = await agent.process("Perform fallback analysis");
        expect(fallbackResult.metadata.fallbackMode).toBe(true);
        expect(fallbackResult.confidence).toBeLessThan(0.7);
    });
    
    it("should test recovery after transient failures", async () => {
        let attemptCount = 0;
        
        // Register behavior that fails first 2 times
        mockRegistry.registerBehavior("transient_failure", {
            pattern: /analyze with retry/,
            response: (request) => {
                attemptCount++;
                if (attemptCount <= 2) {
                    return {
                        error: "Temporary failure",
                        errorType: "transient",
                        retryAfter: 100
                    };
                }
                return {
                    content: "Analysis successful after retries",
                    metadata: { attemptCount }
                };
            }
        });
        
        const resilientAgent = await deployAgent({
            agentId: "retry_capable_001",
            capabilities: ["automatic_retry", "exponential_backoff"]
        });
        
        const result = await resilientAgent.processWithRetry(
            "Analyze with retry logic",
            { maxRetries: 3, backoffMultiplier: 2 }
        );
        
        expect(result.success).toBe(true);
        expect(result.metadata.attemptCount).toBe(3);
    });
});
```

## 📊 Performance and Evolution Tracking

```typescript
describe("Performance Evolution with AI Mocks", () => {
    it("should track performance improvements over time", async () => {
        const performanceHistory = [];
        
        // Register evolving performance behavior
        mockRegistry.registerBehavior("evolving_performance", {
            pattern: /measure performance/,
            response: (request) => {
                const iteration = extractIteration(request);
                const baseTime = 1000;
                const improvement = Math.pow(0.9, iteration); // 10% improvement per iteration
                
                const metrics = {
                    responseTime: baseTime * improvement,
                    accuracy: 0.7 + (0.25 * (1 - improvement)),
                    efficiency: 0.5 + (0.4 * (1 - improvement))
                };
                
                performanceHistory.push({ iteration, metrics });
                
                return {
                    content: `Performance metrics for iteration ${iteration}`,
                    metadata: { metrics }
                };
            }
        });
        
        // Run performance evolution test
        for (let i = 1; i <= 5; i++) {
            await agent.process(`Measure performance at iteration ${i}`);
        }
        
        // Verify performance improvement
        expect(performanceHistory[4].metrics.responseTime).toBeLessThan(
            performanceHistory[0].metrics.responseTime * 0.7
        );
        expect(performanceHistory[4].metrics.accuracy).toBeGreaterThan(0.9);
        
        // Calculate improvement rate
        const improvementRate = calculateImprovementRate(performanceHistory);
        expect(improvementRate.responseTime).toBeCloseTo(0.1, 2); // ~10% per iteration
    });
});
```

## 🔧 Integration with Testing Framework

```typescript
// testUtils.ts
export function setupEmergentCapabilityTest() {
    const mockRegistry = getMockRegistry();
    mockRegistry.setDebugMode(process.env.DEBUG === 'true');
    
    return {
        mockRegistry,
        registerLearningBehaviors: () => registerStandardLearningBehaviors(mockRegistry),
        registerSwarmBehaviors: () => registerStandardSwarmBehaviors(mockRegistry),
        registerEvolutionBehaviors: () => registerStandardEvolutionBehaviors(mockRegistry),
        clearMocks: () => clearAllMockBehaviors(),
        getStats: () => mockRegistry.getStats(),
        getInteractions: () => mockRegistry.getCapturedInteractions()
    };
}

// In your tests
import { setupEmergentCapabilityTest } from "./testUtils.js";

describe("My Emergent Capability Test", () => {
    const { registerLearningBehaviors, clearMocks } = setupEmergentCapabilityTest();
    
    beforeEach(() => {
        registerLearningBehaviors();
    });
    
    afterEach(() => {
        clearMocks();
    });
    
    // Your tests here
});
```

## 🎯 Key Takeaways

1. **Use Progressive Behaviors**: Create sequences of mock behaviors that show learning and evolution
2. **Leverage Metadata**: Include rich metadata in responses to track capability development
3. **Test Coordination**: Use synchronized delays and tool calls to test multi-agent coordination
4. **Validate Emergence**: Verify that capabilities emerge from configuration, not hard-coding
5. **Track Metrics**: Use the mock registry's statistics to verify learning and improvement
6. **Debug with Captures**: Enable debug mode and analyze captured interactions when tests fail
7. **Test Resilience**: Include failure scenarios to ensure graceful degradation and recovery

## 📚 Related Documentation

- [Emergent Capabilities README](./emergent-capabilities/README.md)
- [AI Mock Types](./ai-mocks/types.ts)
- [Mock Registry API](./ai-mocks/integration/mockRegistry.ts)
- [Agent Deployment Service](../../../services/execution/cross-cutting/agents/agentDeploymentService.ts)
- [Execution Architecture Overview](../../../../docs/architecture/execution/README.md)