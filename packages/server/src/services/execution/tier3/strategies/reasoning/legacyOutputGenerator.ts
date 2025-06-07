import { type Logger } from "winston";
import { type ExecutionContext } from "@vrooli/shared";

/**
 * Result interfaces for reasoning
 */
interface ReasoningResult {
    conclusion: unknown;
    reasoning: string[];
    evidence: Evidence[];
    confidence: number;
    assumptions: string[];
}

interface Evidence {
    type: "fact" | "inference" | "assumption";
    content: string;
    source?: string;
    confidence: number;
}

/**
 * Legacy intelligent output generation patterns extracted from legacy ReasoningStrategy
 * 
 * This module handles output generation using proven legacy patterns that create
 * contextually appropriate outputs based on input context and reasoning results.
 */
export class LegacyOutputGenerator {
    private readonly logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * EXTRACTED FROM LEGACY: generateIntelligentOutput adapted for new context
     * Generates outputs based on reasoning results and expected output configuration
     */
    async generateIntelligentOutputs(
        context: ExecutionContext,
        reasoningResult: ReasoningResult,
        executionPlan?: string
    ): Promise<Record<string, unknown>> {
        const outputs: Record<string, unknown> = {};
        const expectedOutputs = context.config.expectedOutputs as Record<string, any> || {};
        
        this.logger.debug("[LegacyOutputGenerator] Generating intelligent outputs", {
            stepId: context.stepId,
            expectedOutputCount: Object.keys(expectedOutputs).length,
        });
        
        // If no expected outputs specified, use reasoning result directly
        if (Object.keys(expectedOutputs).length === 0) {
            return {
                conclusion: reasoningResult.conclusion,
                reasoning: reasoningResult.reasoning,
                confidence: reasoningResult.confidence,
                evidence: reasoningResult.evidence,
                assumptions: reasoningResult.assumptions,
                executionPlan,
            };
        }

        // LEGACY PATTERN: Generate outputs based on expected output configuration
        for (const [outputName, outputConfig] of Object.entries(expectedOutputs)) {
            outputs[outputName] = await this.generateIntelligentOutput(
                outputName,
                outputConfig,
                context,
                reasoningResult
            );
        }

        return outputs;
    }

    /**
     * EXTRACTED FROM LEGACY: generateIntelligentOutput adapted
     * Generates a single intelligent output based on configuration and context
     */
    private async generateIntelligentOutput(
        outputName: string,
        outputConfig: any,
        context: ExecutionContext,
        reasoningResult: ReasoningResult
    ): Promise<unknown> {
        // LEGACY PATTERN: Use input context to generate relevant output
        const inputValues = Object.entries(context.inputs);

        this.logger.debug("[LegacyOutputGenerator] Generating single output", {
            outputName,
            outputType: outputConfig.type,
            availableInputs: inputValues.length,
        });

        if (outputConfig.type === "Boolean") {
            // LEGACY LOGIC: Boolean based on confidence and available inputs
            if (reasoningResult.confidence > 0.7) {
                return inputValues.length > 0; // True if we have inputs and high confidence
            }
            return false; // Low confidence results in false
        }

        if (outputConfig.type === "Integer" || outputConfig.type === "Number") {
            // LEGACY PATTERN: Generate number based on input context and reasoning
            const numericInputs = inputValues.filter(([, value]) => typeof value === "number");
            if (numericInputs.length > 0) {
                const sum = numericInputs.reduce((acc, [, value]) => acc + (value as number), 0);
                const confidenceMultiplier = reasoningResult.confidence;
                const result = sum * confidenceMultiplier;
                return outputConfig.type === "Integer" ? Math.floor(result) : result;
            }
            
            // LEGACY FALLBACK: Generate based on confidence
            const confidenceValue = reasoningResult.confidence * 100;
            return outputConfig.type === "Integer" ? Math.floor(confidenceValue) : confidenceValue;
        }

        if (outputConfig.type === "Text" || typeof outputConfig.type === "string") {
            // LEGACY PATTERN: Generate text based on context and reasoning
            const stepType = context.stepType || "reasoning";
            const inputSummary = inputValues.length > 0
                ? `processed ${inputValues.length} inputs`
                : "no inputs provided";

            // Enhanced text generation with reasoning details
            const conclusionText = typeof reasoningResult.conclusion === "string" 
                ? reasoningResult.conclusion 
                : JSON.stringify(reasoningResult.conclusion);

            return `Reasoning result for ${stepType}: ${inputSummary} - ${conclusionText} (confidence: ${Math.round(reasoningResult.confidence * 100)}%)`;
        }

        if (outputConfig.type === "Array" || Array.isArray(outputConfig.type)) {
            // LEGACY PATTERN: Generate array based on reasoning steps
            return reasoningResult.reasoning.length > 0 
                ? reasoningResult.reasoning 
                : [reasoningResult.conclusion];
        }

        // LEGACY PATTERN: Default structured output with comprehensive reasoning data
        return {
            result: reasoningResult.conclusion,
            context: `Based on step ${context.stepId}`,
            timestamp: new Date().toISOString(),
            confidence: reasoningResult.confidence,
            reasoning: reasoningResult.reasoning,
            evidence: reasoningResult.evidence,
            assumptions: reasoningResult.assumptions,
            metadata: {
                stepType: context.stepType,
                inputCount: inputValues.length,
                strategy: "reasoning",
            },
        };
    }

    /**
     * EXTRACTED FROM LEGACY: applyIntelligentRefinements adapted
     * Applies context-aware improvements to outputs using legacy patterns
     */
    applyIntelligentRefinements(
        outputs: Record<string, unknown>,
        refinements: string[],
        context: ExecutionContext
    ): Record<string, unknown> {
        const refinedOutputs = { ...outputs };

        this.logger.debug("[LegacyOutputGenerator] Applying intelligent refinements", {
            stepId: context.stepId,
            refinementCount: refinements.length,
            outputCount: Object.keys(outputs).length,
        });

        // LEGACY PATTERN: Apply context-aware improvements to outputs
        for (const [outputName, outputValue] of Object.entries(outputs)) {
            if (outputValue !== undefined && typeof outputValue === "object" && outputValue !== null) {
                // LEGACY PATTERN: Add metadata to indicate AI reasoning was applied
                refinedOutputs[outputName] = {
                    ...outputValue,
                    _aiRefinements: refinements.length,
                    _confidence: 0.90,
                    _strategy: "reasoning",
                    _stepId: context.stepId,
                    _refinements: refinements,
                };
            } else if (outputValue !== undefined) {
                // For primitive values, wrap in object with metadata
                refinedOutputs[outputName] = {
                    value: outputValue,
                    _aiRefinements: refinements.length,
                    _confidence: 0.90,
                    _strategy: "reasoning",
                    _stepId: context.stepId,
                    _refinements: refinements,
                };
            }
        }

        // LEGACY PATTERN: Add overall refinement summary
        refinedOutputs._refinementSummary = {
            appliedRefinements: refinements,
            refinementCount: refinements.length,
            processedAt: new Date().toISOString(),
            strategy: "reasoning",
            stepId: context.stepId,
        };

        return refinedOutputs;
    }

    /**
     * Creates a legacy-compatible success result with comprehensive metadata
     */
    createLegacyCompatibleSuccessResult(
        finalOutputs: Record<string, unknown>,
        creditsUsed: number,
        toolCallsCount: number,
        executionTimeMs: number,
        analysisResult?: { insights: string[] },
        planningResult?: { executionPlan: string },
        refinementResult?: { refinements: string[] }
    ): any {
        this.logger.debug("[LegacyOutputGenerator] Creating legacy-compatible success result", {
            outputCount: Object.keys(finalOutputs).length,
            creditsUsed,
            toolCallsCount,
            executionTimeMs,
        });

        return {
            success: true,
            outputs: finalOutputs,
            metadata: {
                strategy: "reasoning",
                version: "2.0.0-enhanced",
                executionPhases: {
                    analysis: analysisResult ? {
                        insights: analysisResult.insights,
                        insightCount: analysisResult.insights.length,
                    } : undefined,
                    planning: planningResult ? {
                        plan: planningResult.executionPlan,
                        planLength: planningResult.executionPlan.length,
                    } : undefined,
                    refinement: refinementResult ? {
                        refinements: refinementResult.refinements,
                        refinementCount: refinementResult.refinements.length,
                    } : undefined,
                },
                performance: {
                    executionTimeMs,
                    creditsUsed,
                    toolCallsCount,
                    averagePhaseTime: executionTimeMs / 4, // 4 phases
                },
                timestamp: new Date().toISOString(),
            },
            resourceUsage: {
                tokens: Math.floor(creditsUsed * 7.5), // Estimate tokens from credits
                apiCalls: toolCallsCount,
                computeTime: executionTimeMs,
                cost: creditsUsed * 0.00001, // Convert credits to cost estimate
            },
            confidence: 0.90, // High confidence for reasoning strategy
            performanceScore: this.calculatePerformanceScore(creditsUsed, executionTimeMs, toolCallsCount),
        };
    }

    /**
     * Creates a legacy-compatible error result
     */
    createLegacyCompatibleErrorResult(
        error: Error,
        creditsUsed: number,
        toolCallsCount: number,
        executionTimeMs: number
    ): any {
        this.logger.error("[LegacyOutputGenerator] Creating legacy-compatible error result", {
            error: error.message,
            creditsUsed,
            toolCallsCount,
            executionTimeMs,
        });

        return {
            success: false,
            error: {
                message: error.message,
                type: error.constructor.name,
                strategy: "reasoning",
                phase: "unknown", // Could be enhanced to track which phase failed
            },
            outputs: {}, // Empty outputs for error case
            metadata: {
                strategy: "reasoning",
                version: "2.0.0-enhanced",
                executionFailed: true,
                performance: {
                    executionTimeMs,
                    creditsUsed,
                    toolCallsCount,
                },
                timestamp: new Date().toISOString(),
            },
            resourceUsage: {
                tokens: Math.floor(creditsUsed * 7.5),
                apiCalls: toolCallsCount,
                computeTime: executionTimeMs,
                cost: creditsUsed * 0.00001,
            },
            confidence: 0.0, // Zero confidence for failed execution
            performanceScore: 0.0, // Zero performance for errors
        };
    }

    /**
     * Calculates performance score based on resource usage
     */
    private calculatePerformanceScore(creditsUsed: number, executionTimeMs: number, toolCallsCount: number): number {
        const BASE_SCORE = 0.6;
        const EFFICIENCY_WEIGHT = 0.2;
        const SPEED_WEIGHT = 0.2;
        
        // Efficiency score (lower credits = better)
        const efficiencyScore = Math.max(0, 1 - (creditsUsed / 1000)); // Normalize by 1000 credits
        
        // Speed score (lower time = better) 
        const speedScore = Math.max(0, 1 - (executionTimeMs / 30000)); // Normalize by 30 seconds
        
        const finalScore = BASE_SCORE + (efficiencyScore * EFFICIENCY_WEIGHT) + (speedScore * SPEED_WEIGHT);
        return Math.min(1, finalScore);
    }
}