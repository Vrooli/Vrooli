import { describe, it, expect, beforeEach, vi } from "vitest";
import { ExecutionRunContext, type UserData, type StepConfig, type UsageHints } from "./runContext.js";
import { generatePK } from "@vrooli/shared";
import { createMockLogger } from "../../../../../__test/globalHelpers.js";
import { type Logger } from "winston";

/**
 * ExecutionRunContext Tests - Context-Aware Intelligence
 * 
 * These tests validate that the run context provides infrastructure for
 * adaptive execution while enabling intelligence to emerge from:
 * 
 * 1. **Context Accumulation**: Memory without prescriptive strategies
 * 2. **Usage Pattern Hints**: Data for adaptive strategy selection
 * 3. **Environment Awareness**: User/system context for decisions
 * 4. **Immutable State**: Consistency during execution
 * 5. **Cross-Tier Communication**: Context flow between tiers
 * 
 * The context provides data infrastructure - intelligence emerges from
 * how agents use this context to make adaptive decisions.
 */

describe("ExecutionRunContext - Context-Aware Execution Infrastructure", () => {
    let logger: Logger;
    let baseUserData: UserData;
    let baseConfig: any;

    beforeEach(() => {
        logger = createMockLogger();
        
        baseUserData = {
            id: generatePK(),
            email: "test@example.com",
            name: "Test User",
            languages: ["en", "es"],
            preferences: {
                theme: "dark",
                notifications: true,
                defaultModel: "claude-3-opus",
            },
        };

        baseConfig = {
            runId: generatePK(),
            routineId: generatePK(),
            routineName: "Test Routine",
            userData: baseUserData,
        };
    });

    describe("Context Infrastructure Creation", () => {
        it("should create immutable context with minimal required data", () => {
            const context = new ExecutionRunContext(baseConfig);

            expect(context.runId).toBe(baseConfig.runId);
            expect(context.routineId).toBe(baseConfig.routineId);
            expect(context.routineName).toBe(baseConfig.routineName);
            expect(context.userData).toEqual(baseUserData);
            
            // Context is immutable during execution
            expect(() => {
                (context as any).runId = "new-id";
            }).toThrow();
        });

        it("should generate runId if not provided", () => {
            const configWithoutRunId = {
                ...baseConfig,
                runId: undefined,
            };
            
            const context = new ExecutionRunContext(configWithoutRunId);
            
            expect(context.runId).toBeDefined();
            expect(typeof context.runId).toBe("string");
            expect(context.runId.length).toBeGreaterThan(0);
        });

        it("should handle hierarchical execution context", () => {
            const parentRunId = generatePK();
            const swarmId = generatePK();
            const currentStepId = generatePK();
            
            const hierarchicalConfig = {
                ...baseConfig,
                parentRunId,
                swarmId,
                currentStepId,
            };
            
            const context = new ExecutionRunContext(hierarchicalConfig);
            
            expect(context.parentRunId).toBe(parentRunId);
            expect(context.swarmId).toBe(swarmId);
            expect(context.currentStepId).toBe(currentStepId);
        });
    });

    describe("Usage Hints for Adaptive Strategy Selection", () => {
        it("should provide hints without prescriptive strategy logic", () => {
            const usageHints: UsageHints = {
                historicalSuccessRate: 0.85,
                executionFrequency: 50, // executions per day
                averageComplexity: 0.6,
                userPreference: "fast",
                domainRestrictions: ["no_external_apis"],
            };
            
            const configWithHints = {
                ...baseConfig,
                usageHints,
            };
            
            const context = new ExecutionRunContext(configWithHints);
            
            expect(context.usageHints).toEqual(usageHints);
            
            // Hints inform strategy selection, don't dictate it
            expect(context.usageHints?.historicalSuccessRate).toBe(0.85);
            expect(context.usageHints?.userPreference).toBe("fast");
        });

        it("should enable context-based strategy evolution", () => {
            // Simulate different usage patterns
            const patterns = [
                {
                    name: "frequent_simple",
                    hints: {
                        historicalSuccessRate: 0.95,
                        executionFrequency: 200,
                        averageComplexity: 0.2,
                    },
                },
                {
                    name: "infrequent_complex",
                    hints: {
                        historicalSuccessRate: 0.70,
                        executionFrequency: 5,
                        averageComplexity: 0.9,
                    },
                },
                {
                    name: "experimental",
                    hints: {
                        historicalSuccessRate: undefined, // No history
                        executionFrequency: 1,
                        averageComplexity: 0.5,
                        userPreference: "thorough",
                    },
                },
            ];
            
            for (const pattern of patterns) {
                const context = new ExecutionRunContext({
                    ...baseConfig,
                    runId: generatePK(),
                    usageHints: pattern.hints,
                });
                
                // Strategy selectors can use these hints for adaptation
                expect(context.usageHints).toEqual(pattern.hints);
                
                // Different patterns enable different strategy choices
                switch (pattern.name) {
                    case "frequent_simple":
                        // High success rate + frequency suggests deterministic strategy
                        expect(context.usageHints?.historicalSuccessRate).toBeGreaterThan(0.9);
                        break;
                    case "infrequent_complex":
                        // Complex tasks may benefit from reasoning strategy
                        expect(context.usageHints?.averageComplexity).toBeGreaterThan(0.8);
                        break;
                    case "experimental":
                        // No history suggests conversational strategy
                        expect(context.usageHints?.historicalSuccessRate).toBeUndefined();
                        break;
                }
            }
        });
    });

    describe("Environment and User Context", () => {
        it("should provide user context for personalized execution", () => {
            const richUserData: UserData = {
                id: generatePK(),
                email: "expert@company.com",
                name: "Domain Expert",
                languages: ["en", "fr", "de"],
                preferences: {
                    executionStyle: "detailed",
                    outputFormat: "structured",
                    timeZone: "Europe/Paris",
                    expertise: ["machine_learning", "data_analysis"],
                },
            };
            
            const context = new ExecutionRunContext({
                ...baseConfig,
                userData: richUserData,
            });
            
            // User context enables personalized execution
            expect(context.userData.languages).toContain("fr");
            expect(context.userData.preferences?.executionStyle).toBe("detailed");
            expect(context.userData.preferences?.expertise).toContain("machine_learning");
        });

        it("should handle environment configuration for context awareness", () => {
            const environment = {
                NODE_ENV: "production",
                EXECUTION_TIER: "cloud",
                AVAILABLE_MEMORY: "8GB",
                MAX_EXECUTION_TIME: "300000",
                COMPLIANCE_MODE: "strict",
            };
            
            const context = new ExecutionRunContext({
                ...baseConfig,
                environment,
            });
            
            expect(context.environment).toEqual(environment);
            
            // Environment enables adaptive resource allocation
            expect(context.environment.AVAILABLE_MEMORY).toBe("8GB");
            expect(context.environment.COMPLIANCE_MODE).toBe("strict");
        });
    });

    describe("Step Configuration Infrastructure", () => {
        it("should provide step configuration without execution logic", () => {
            const stepConfig: StepConfig = {
                requiredInputs: ["data", "schema"],
                outputTransformations: {
                    format: "json",
                    validation: "strict",
                },
                timeoutMs: 30000,
                retryPolicy: {
                    maxRetries: 3,
                    backoffMs: 1000,
                },
            };
            
            const context = new ExecutionRunContext({
                ...baseConfig,
                stepConfig,
            });
            
            expect(context.stepConfig).toEqual(stepConfig);
            
            // Configuration available, execution logic emerges
            expect(context.stepConfig?.requiredInputs).toContain("data");
            expect(context.stepConfig?.retryPolicy?.maxRetries).toBe(3);
        });

        it("should enable step-specific optimization", () => {
            const optimizationConfigs = [
                {
                    type: "data_processing",
                    config: {
                        requiredInputs: ["dataset"],
                        outputTransformations: { aggregation: "sum" },
                        timeoutMs: 60000,
                    },
                },
                {
                    type: "user_interaction",
                    config: {
                        requiredInputs: ["prompt"],
                        outputTransformations: { format: "markdown" },
                        timeoutMs: 5000,
                    },
                },
                {
                    type: "api_call",
                    config: {
                        requiredInputs: ["endpoint", "params"],
                        retryPolicy: {
                            maxRetries: 5,
                            backoffMs: 2000,
                        },
                        timeoutMs: 15000,
                    },
                },
            ];
            
            for (const { type, config } of optimizationConfigs) {
                const context = new ExecutionRunContext({
                    ...baseConfig,
                    currentStepId: `${type}_step`,
                    stepConfig: config,
                });
                
                // Step-specific configuration enables optimization
                expect(context.stepConfig).toEqual(config);
                expect(context.currentStepId).toBe(`${type}_step`);
            }
        });
    });

    describe("Context Metadata and Timing", () => {
        it("should track execution timing for performance analysis", () => {
            const context = new ExecutionRunContext(baseConfig);
            
            // Timing started at creation
            const initialTime = context.getExecutionTime();
            expect(initialTime).toBe(0);
            
            // Mock time passage
            vi.useFakeTimers();
            vi.advanceTimersByTime(1500);
            
            const elapsedTime = context.getExecutionTime();
            expect(elapsedTime).toBe(1500);
            
            vi.useRealTimers();
        });

        it("should manage metadata without prescriptive structure", () => {
            const context = new ExecutionRunContext(baseConfig);
            
            // Set various metadata types
            context.setMetadata("performanceHint", "optimize_for_speed");
            context.setMetadata("dataSource", { type: "database", connection: "primary" });
            context.setMetadata("userFeedback", { rating: 4.5, comments: "Works well" });
            
            expect(context.getMetadata("performanceHint")).toBe("optimize_for_speed");
            expect(context.getMetadata("dataSource")).toEqual({
                type: "database",
                connection: "primary",
            });
            expect(context.getMetadata("userFeedback")).toEqual({
                rating: 4.5,
                comments: "Works well",
            });
            
            // Metadata enables emergent optimizations
            const allMetadata = context.getAllMetadata();
            expect(Object.keys(allMetadata)).toHaveLength(3);
        });
    });

    describe("Context Serialization and Communication", () => {
        it("should serialize context for cross-tier communication", () => {
            const fullContext = new ExecutionRunContext({
                ...baseConfig,
                parentRunId: generatePK(),
                swarmId: generatePK(),
                currentStepId: generatePK(),
                environment: { MODE: "test" },
                usageHints: {
                    historicalSuccessRate: 0.9,
                    executionFrequency: 100,
                },
            });
            
            fullContext.setMetadata("tier2Data", { checkpoints: 3 });
            
            const serialized = fullContext.toJSON();
            
            expect(serialized).toHaveProperty("runId");
            expect(serialized).toHaveProperty("routineId");
            expect(serialized).toHaveProperty("userData");
            expect(serialized).toHaveProperty("environment");
            expect(serialized).toHaveProperty("usageHints");
            expect(serialized).toHaveProperty("metadata");
            
            // Can be reconstructed from serialization
            expect(serialized.runId).toBe(fullContext.runId);
            expect(serialized.metadata.tier2Data).toEqual({ checkpoints: 3 });
        });

        it("should create child contexts for sub-executions", () => {
            const parentContext = new ExecutionRunContext({
                ...baseConfig,
                swarmId: generatePK(),
            });
            
            parentContext.setMetadata("parentData", { strategy: "reasoning" });
            
            const childContext = parentContext.createChildContext({
                currentStepId: "child_step",
                stepConfig: {
                    requiredInputs: ["parent_output"],
                    timeoutMs: 5000,
                },
            });
            
            // Child inherits parent context
            expect(childContext.parentRunId).toBe(parentContext.runId);
            expect(childContext.swarmId).toBe(parentContext.swarmId);
            expect(childContext.userData).toEqual(parentContext.userData);
            
            // Child has its own execution context
            expect(childContext.runId).not.toBe(parentContext.runId);
            expect(childContext.currentStepId).toBe("child_step");
            
            // Parent metadata available but isolated
            expect(childContext.getMetadata("parentData")).toEqual({ strategy: "reasoning" });
        });
    });

    describe("Context-Based Decision Support", () => {
        it("should provide context analysis for strategy selection", () => {
            const analyticalContext = new ExecutionRunContext({
                ...baseConfig,
                userData: {
                    ...baseUserData,
                    preferences: {
                        ...baseUserData.preferences,
                        expertise: ["data_science", "sql"],
                        executionPreference: "thorough",
                    },
                },
                usageHints: {
                    historicalSuccessRate: 0.75,
                    executionFrequency: 25,
                    averageComplexity: 0.8,
                    domainRestrictions: ["no_external_apis"],
                },
                environment: {
                    COMPLIANCE_MODE: "strict",
                    AVAILABLE_CREDITS: "1000",
                },
            });
            
            const analysis = analyticalContext.getContextAnalysis();
            
            // Analysis provides data for intelligent decisions
            expect(analysis).toHaveProperty("userExpertise");
            expect(analysis).toHaveProperty("executionConstraints");
            expect(analysis).toHaveProperty("performanceHints");
            expect(analysis).toHaveProperty("environmentLimitations");
            
            expect(analysis.userExpertise).toContain("data_science");
            expect(analysis.executionConstraints).toContain("no_external_apis");
            expect(analysis.performanceHints.complexity).toBe(0.8);
        });

        it("should enable adaptive resource allocation", () => {
            const resourceContext = new ExecutionRunContext({
                ...baseConfig,
                environment: {
                    AVAILABLE_MEMORY: "4GB",
                    MAX_EXECUTION_TIME: "120000",
                    CONCURRENT_LIMIT: "3",
                },
                usageHints: {
                    averageComplexity: 0.3, // Simple task
                },
            });
            
            const resourceAnalysis = resourceContext.getResourceConstraints();
            
            // Resource analysis enables adaptive allocation
            expect(resourceAnalysis).toHaveProperty("memoryLimit");
            expect(resourceAnalysis).toHaveProperty("timeLimit");
            expect(resourceAnalysis).toHaveProperty("concurrencyLimit");
            expect(resourceAnalysis).toHaveProperty("complexityScore");
            
            expect(resourceAnalysis.memoryLimit).toBe("4GB");
            expect(resourceAnalysis.complexityScore).toBe(0.3);
        });
    });
});
