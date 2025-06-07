import { describe, it, beforeEach, vi, expect } from "vitest";
import { type Logger } from "winston";
import { type ExecutionContext } from "@vrooli/shared";
import { LegacyOutputGenerator } from "../legacyOutputGenerator.js";

describe("LegacyOutputGenerator", () => {
    let generator: LegacyOutputGenerator;
    let mockLogger: Logger;
    let baseContext: ExecutionContext;
    let mockReasoningResult: any;

    beforeEach(() => {
        mockLogger = {
            debug: vi.fn(),
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        } as any;

        baseContext = {
            stepId: "test-step-123",
            stepType: "analyze_data",
            inputs: {
                data: [1, 2, 3, 4, 5],
                threshold: 3,
            },
            config: {
                expectedOutputs: {
                    result: { type: "Boolean" },
                    summary: { type: "Text" },
                    score: { type: "Number" },
                    count: { type: "Integer" },
                },
            },
            resources: {
                models: [],
                tools: [],
                apis: [],
                credits: 5000,
            },
            history: {
                recentSteps: [],
                totalExecutions: 0,
                successRate: 1.0,
            },
            constraints: {},
        };

        mockReasoningResult = {
            conclusion: "Data analysis completed successfully",
            reasoning: [
                "Analyzed data array with 5 elements",
                "Compared each element against threshold of 3",
                "Found 2 elements above threshold",
                "Conclusion: 40% of data exceeds threshold",
            ],
            evidence: [
                {
                    type: "fact",
                    content: "Data contains values [1,2,3,4,5]",
                    confidence: 1.0,
                },
                {
                    type: "inference",
                    content: "Elements 4 and 5 exceed threshold 3",
                    confidence: 0.95,
                },
            ],
            confidence: 0.92,
            assumptions: ["Threshold comparison is the correct method"],
        };

        generator = new LegacyOutputGenerator(mockLogger);
    });

    describe("generateIntelligentOutputs", () => {
        it("should return reasoning result directly when no expected outputs defined", async () => {
            const contextNoOutputs = {
                ...baseContext,
                config: { expectedOutputs: {} },
            };

            const result = await generator.generateIntelligentOutputs(
                contextNoOutputs,
                mockReasoningResult,
                "Test execution plan"
            );

            expect(result).toEqual({
                conclusion: mockReasoningResult.conclusion,
                reasoning: mockReasoningResult.reasoning,
                confidence: mockReasoningResult.confidence,
                evidence: mockReasoningResult.evidence,
                assumptions: mockReasoningResult.assumptions,
                executionPlan: "Test execution plan",
            });
        });

        it("should generate outputs based on expected output configuration", async () => {
            const result = await generator.generateIntelligentOutputs(
                baseContext,
                mockReasoningResult
            );

            expect(result).toHaveProperty("result");
            expect(result).toHaveProperty("summary");
            expect(result).toHaveProperty("score");
            expect(result).toHaveProperty("count");

            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[LegacyOutputGenerator] Generating intelligent outputs",
                {
                    stepId: "test-step-123",
                    expectedOutputCount: 4,
                }
            );
        });

        it("should generate Boolean outputs correctly", async () => {
            const contextBooleanOnly = {
                ...baseContext,
                config: {
                    expectedOutputs: {
                        highConfidence: { type: "Boolean" },
                        lowConfidence: { type: "Boolean" },
                    },
                },
            };

            // Test high confidence scenario
            const highConfidenceResult = {
                ...mockReasoningResult,
                confidence: 0.85,
            };

            const result1 = await generator.generateIntelligentOutputs(
                contextBooleanOnly,
                highConfidenceResult
            );

            expect(result1.highConfidence).toBe(true); // Should be true (high confidence + has inputs)
            expect(result1.lowConfidence).toBe(true);

            // Test low confidence scenario
            const lowConfidenceResult = {
                ...mockReasoningResult,
                confidence: 0.5,
            };

            const result2 = await generator.generateIntelligentOutputs(
                contextBooleanOnly,
                lowConfidenceResult
            );

            expect(result2.highConfidence).toBe(false); // Should be false (low confidence)
            expect(result2.lowConfidence).toBe(false);
        });

        it("should generate numeric outputs from input context", async () => {
            const contextWithNumbers = {
                ...baseContext,
                inputs: {
                    value1: 10,
                    value2: 20,
                    value3: 30,
                    nonNumeric: "text",
                },
                config: {
                    expectedOutputs: {
                        sum: { type: "Number" },
                        sumInteger: { type: "Integer" },
                    },
                },
            };

            const result = await generator.generateIntelligentOutputs(
                contextWithNumbers,
                mockReasoningResult
            );

            // Should sum numeric inputs: 10 + 20 + 30 = 60
            expect(result.sum).toBe(60);
            expect(result.sumInteger).toBe(60);
        });

        it("should use fallback values for numeric outputs when no numeric inputs", async () => {
            const contextNoNumbers = {
                ...baseContext,
                inputs: {
                    text1: "hello",
                    text2: "world",
                },
                config: {
                    expectedOutputs: {
                        number: { type: "Number" },
                        integer: { type: "Integer" },
                    },
                },
            };

            const result = await generator.generateIntelligentOutputs(
                contextNoNumbers,
                mockReasoningResult
            );

            expect(result.number).toBe(42.0);
            expect(result.integer).toBe(42);
        });

        it("should generate text outputs with contextual information", async () => {
            const contextTextOnly = {
                ...baseContext,
                stepType: "data_validation",
                inputs: {
                    input1: "test1",
                    input2: "test2",
                },
                config: {
                    expectedOutputs: {
                        description: { type: "Text" },
                    },
                },
            };

            const result = await generator.generateIntelligentOutputs(
                contextTextOnly,
                mockReasoningResult
            );

            expect(result.description).toBe(
                "Reasoning result for data_validation: processed 2 inputs - Data analysis completed successfully (confidence: 92%)"
            );
        });

        it("should generate array outputs from reasoning steps", async () => {
            const contextArrayOnly = {
                ...baseContext,
                config: {
                    expectedOutputs: {
                        steps: { type: "Array" },
                        items: { type: [] }, // Alternative array syntax
                    },
                },
            };

            const result = await generator.generateIntelligentOutputs(
                contextArrayOnly,
                mockReasoningResult
            );

            expect(Array.isArray(result.steps)).toBe(true);
            expect(result.steps).toEqual(mockReasoningResult.reasoning);
            expect(Array.isArray(result.items)).toBe(true);
            expect(result.items).toEqual(mockReasoningResult.reasoning);
        });

        it("should generate structured object outputs for unknown types", async () => {
            const contextObjectOnly = {
                ...baseContext,
                config: {
                    expectedOutputs: {
                        analysis: { type: "Analysis" },
                        metadata: { type: "Object" },
                    },
                },
            };

            const result = await generator.generateIntelligentOutputs(
                contextObjectOnly,
                mockReasoningResult
            );

            expect(typeof result.analysis).toBe("object");
            expect(result.analysis).toHaveProperty("result", mockReasoningResult.conclusion);
            expect(result.analysis).toHaveProperty("context", "Based on step test-step-123");
            expect(result.analysis).toHaveProperty("confidence", 0.92);
            expect(result.analysis).toHaveProperty("reasoning", mockReasoningResult.reasoning);
            expect(result.analysis).toHaveProperty("evidence", mockReasoningResult.evidence);
            expect(result.analysis).toHaveProperty("metadata");
            expect(result.analysis.metadata).toHaveProperty("stepType", "analyze_data");
            expect(result.analysis.metadata).toHaveProperty("inputCount", 2);
            expect(result.analysis.metadata).toHaveProperty("strategy", "reasoning");
        });

        it("should log individual output generation", async () => {
            await generator.generateIntelligentOutputs(baseContext, mockReasoningResult);

            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[LegacyOutputGenerator] Generating single output",
                expect.objectContaining({
                    outputName: "result",
                    outputType: "Boolean",
                    availableInputs: 2,
                })
            );
        });
    });

    describe("applyIntelligentRefinements", () => {
        const mockOutputs = {
            result: true,
            summary: "Analysis complete",
            metadata: {
                score: 0.85,
                timestamp: "2023-01-01T00:00:00Z",
            },
            primitiveValue: 42,
        };

        const mockRefinements = [
            "Enhanced clarity",
            "Improved accuracy",
            "Added validation",
        ];

        it("should apply refinements to object outputs", () => {
            const result = generator.applyIntelligentRefinements(
                mockOutputs,
                mockRefinements,
                baseContext
            );

            // Object outputs should have refinement metadata added
            expect(result.metadata).toHaveProperty("_aiRefinements", 3);
            expect(result.metadata).toHaveProperty("_confidence", 0.90);
            expect(result.metadata).toHaveProperty("_strategy", "reasoning");
            expect(result.metadata).toHaveProperty("_stepId", "test-step-123");
            expect(result.metadata).toHaveProperty("_refinements", mockRefinements);

            // Original properties should be preserved
            expect(result.metadata).toHaveProperty("score", 0.85);
            expect(result.metadata).toHaveProperty("timestamp", "2023-01-01T00:00:00Z");
        });

        it("should wrap primitive values in objects with metadata", () => {
            const result = generator.applyIntelligentRefinements(
                mockOutputs,
                mockRefinements,
                baseContext
            );

            expect(result.primitiveValue).toEqual({
                value: 42,
                _aiRefinements: 3,
                _confidence: 0.90,
                _strategy: "reasoning",
                _stepId: "test-step-123",
                _refinements: mockRefinements,
            });
        });

        it("should preserve non-object, non-primitive values unchanged", () => {
            const outputsWithUndefined = {
                ...mockOutputs,
                undefinedValue: undefined,
                nullValue: null,
            };

            const result = generator.applyIntelligentRefinements(
                outputsWithUndefined,
                mockRefinements,
                baseContext
            );

            expect(result.undefinedValue).toBeUndefined();
            expect(result.nullValue).toBeNull();
        });

        it("should add refinement summary", () => {
            const result = generator.applyIntelligentRefinements(
                mockOutputs,
                mockRefinements,
                baseContext
            );

            expect(result).toHaveProperty("_refinementSummary");
            expect(result._refinementSummary).toEqual({
                appliedRefinements: mockRefinements,
                refinementCount: 3,
                processedAt: expect.any(String),
                strategy: "reasoning",
                stepId: "test-step-123",
            });
        });

        it("should log refinement application", () => {
            generator.applyIntelligentRefinements(
                mockOutputs,
                mockRefinements,
                baseContext
            );

            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[LegacyOutputGenerator] Applying intelligent refinements",
                {
                    stepId: "test-step-123",
                    refinementCount: 3,
                    outputCount: 4,
                }
            );
        });
    });

    describe("createLegacyCompatibleSuccessResult", () => {
        const mockFinalOutputs = {
            result: true,
            summary: "Analysis completed",
        };

        const mockAnalysisResult = {
            insights: ["Insight 1", "Insight 2"],
        };

        const mockPlanningResult = {
            executionPlan: "Step 1, Step 2, Step 3",
        };

        const mockRefinementResult = {
            refinements: ["Refinement 1", "Refinement 2"],
        };

        it("should create comprehensive success result structure", () => {
            const result = generator.createLegacyCompatibleSuccessResult(
                mockFinalOutputs,
                650, // creditsUsed
                4,   // toolCallsCount
                15000, // executionTimeMs
                mockAnalysisResult,
                mockPlanningResult,
                mockRefinementResult
            );

            expect(result).toEqual({
                success: true,
                outputs: mockFinalOutputs,
                metadata: {
                    strategy: "reasoning",
                    version: "2.0.0-enhanced",
                    executionPhases: {
                        analysis: {
                            insights: ["Insight 1", "Insight 2"],
                            insightCount: 2,
                        },
                        planning: {
                            plan: "Step 1, Step 2, Step 3",
                            planLength: 17,
                        },
                        refinement: {
                            refinements: ["Refinement 1", "Refinement 2"],
                            refinementCount: 2,
                        },
                    },
                    performance: {
                        executionTimeMs: 15000,
                        creditsUsed: 650,
                        toolCallsCount: 4,
                        averagePhaseTime: 3750, // 15000 / 4
                    },
                    timestamp: expect.any(String),
                },
                resourceUsage: {
                    tokens: 4875, // 650 * 7.5
                    apiCalls: 4,
                    computeTime: 15000,
                    cost: 0.0065, // 650 * 0.00001
                },
                confidence: 0.90,
                performanceScore: expect.any(Number),
            });
        });

        it("should handle missing phase results gracefully", () => {
            const result = generator.createLegacyCompatibleSuccessResult(
                mockFinalOutputs,
                100,
                1,
                5000
            );

            expect(result.metadata.executionPhases.analysis).toBeUndefined();
            expect(result.metadata.executionPhases.planning).toBeUndefined();
            expect(result.metadata.executionPhases.refinement).toBeUndefined();
        });

        it("should calculate realistic performance scores", () => {
            const testCases = [
                { credits: 100, time: 5000, tools: 1, expectedRange: [0.8, 1.0] },
                { credits: 500, time: 15000, tools: 3, expectedRange: [0.6, 0.8] },
                { credits: 1000, time: 30000, tools: 5, expectedRange: [0.4, 0.7] },
            ];

            testCases.forEach(({ credits, time, tools, expectedRange }) => {
                const result = generator.createLegacyCompatibleSuccessResult(
                    mockFinalOutputs,
                    credits,
                    tools,
                    time
                );

                expect(result.performanceScore).toBeGreaterThanOrEqual(expectedRange[0]);
                expect(result.performanceScore).toBeLessThanOrEqual(expectedRange[1]);
            });
        });

        it("should log success result creation", () => {
            generator.createLegacyCompatibleSuccessResult(
                mockFinalOutputs,
                650,
                4,
                15000,
                mockAnalysisResult,
                mockPlanningResult,
                mockRefinementResult
            );

            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[LegacyOutputGenerator] Creating legacy-compatible success result",
                {
                    outputCount: 2,
                    creditsUsed: 650,
                    toolCallsCount: 4,
                    executionTimeMs: 15000,
                }
            );
        });
    });

    describe("createLegacyCompatibleErrorResult", () => {
        const mockError = new Error("Test error message");

        it("should create comprehensive error result structure", () => {
            const result = generator.createLegacyCompatibleErrorResult(
                mockError,
                150, // creditsUsed
                2,   // toolCallsCount
                8000 // executionTimeMs
            );

            expect(result).toEqual({
                success: false,
                error: {
                    message: "Test error message",
                    type: "Error",
                    strategy: "reasoning",
                    phase: "unknown",
                },
                outputs: {},
                metadata: {
                    strategy: "reasoning",
                    version: "2.0.0-enhanced",
                    executionFailed: true,
                    performance: {
                        executionTimeMs: 8000,
                        creditsUsed: 150,
                        toolCallsCount: 2,
                    },
                    timestamp: expect.any(String),
                },
                resourceUsage: {
                    tokens: 1125, // 150 * 7.5
                    apiCalls: 2,
                    computeTime: 8000,
                    cost: 0.0015, // 150 * 0.00001
                },
                confidence: 0.0,
                performanceScore: 0.0,
            });
        });

        it("should handle different error types", () => {
            const typeError = new TypeError("Type error");
            const syntaxError = new SyntaxError("Syntax error");

            const typeResult = generator.createLegacyCompatibleErrorResult(typeError, 0, 0, 0);
            const syntaxResult = generator.createLegacyCompatibleErrorResult(syntaxError, 0, 0, 0);

            expect(typeResult.error.type).toBe("TypeError");
            expect(syntaxResult.error.type).toBe("SyntaxError");
        });

        it("should log error result creation", () => {
            generator.createLegacyCompatibleErrorResult(mockError, 150, 2, 8000);

            expect(mockLogger.error).toHaveBeenCalledWith(
                "[LegacyOutputGenerator] Creating legacy-compatible error result",
                {
                    error: "Test error message",
                    creditsUsed: 150,
                    toolCallsCount: 2,
                    executionTimeMs: 8000,
                }
            );
        });
    });

    describe("legacy pattern preservation", () => {
        it("should maintain legacy Boolean generation logic", async () => {
            const testCases = [
                {
                    confidence: 0.9,
                    hasInputs: true,
                    expected: true,
                    description: "high confidence with inputs",
                },
                {
                    confidence: 0.9,
                    hasInputs: false,
                    expected: false,
                    description: "high confidence without inputs",
                },
                {
                    confidence: 0.5,
                    hasInputs: true,
                    expected: false,
                    description: "low confidence with inputs",
                },
                {
                    confidence: 0.5,
                    hasInputs: false,
                    expected: false,
                    description: "low confidence without inputs",
                },
            ];

            for (const testCase of testCases) {
                const context = {
                    ...baseContext,
                    inputs: testCase.hasInputs ? { data: "test" } : {},
                    config: {
                        expectedOutputs: {
                            result: { type: "Boolean" },
                        },
                    },
                };

                const reasoningResult = {
                    ...mockReasoningResult,
                    confidence: testCase.confidence,
                };

                const result = await generator.generateIntelligentOutputs(
                    context,
                    reasoningResult
                );

                expect(result.result).toBe(testCase.expected);
            }
        });

        it("should maintain legacy numeric scaling patterns", async () => {
            const context = {
                ...baseContext,
                inputs: {
                    num1: 10,
                    num2: 20,
                    num3: 15,
                },
                config: {
                    expectedOutputs: {
                        sum: { type: "Number" },
                        sumInt: { type: "Integer" },
                    },
                },
            };

            const result = await generator.generateIntelligentOutputs(
                context,
                mockReasoningResult
            );

            // Should sum: 10 + 20 + 15 = 45, then apply confidence multiplier
            const expectedBase = 45 * mockReasoningResult.confidence; // 45 * 0.92 = 41.4
            expect(result.sum).toBeCloseTo(expectedBase, 2);
            expect(result.sumInt).toBe(Math.floor(expectedBase));
        });

        it("should maintain legacy text generation patterns", async () => {
            const testCases = [
                { stepType: "analyze", inputCount: 2, conclusionText: "success" },
                { stepType: "validate", inputCount: 1, conclusionText: "passed" },
                { stepType: "optimize", inputCount: 3, conclusionText: "improved" },
            ];

            for (const testCase of testCases) {
                const context = {
                    ...baseContext,
                    stepType: testCase.stepType,
                    inputs: Array.from({ length: testCase.inputCount }, (_, i) => [`input${i}`, `value${i}`])
                        .reduce((obj, [key, value]) => ({ ...obj, [key]: value }), {}),
                    config: {
                        expectedOutputs: {
                            description: { type: "Text" },
                        },
                    },
                };

                const reasoningResult = {
                    ...mockReasoningResult,
                    conclusion: testCase.conclusionText,
                };

                const result = await generator.generateIntelligentOutputs(
                    context,
                    reasoningResult
                );

                expect(result.description).toContain(testCase.stepType);
                expect(result.description).toContain(`processed ${testCase.inputCount} inputs`);
                expect(result.description).toContain(testCase.conclusionText);
                expect(result.description).toContain("92%"); // confidence percentage
            }
        });
    });
});