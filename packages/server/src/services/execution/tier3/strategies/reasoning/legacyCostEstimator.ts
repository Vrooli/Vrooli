import { type ExecutionContext, type ResourceUsage } from "@vrooli/shared";

/**
 * Legacy cost estimation patterns extracted from legacy ReasoningStrategy
 * 
 * This module handles cost estimation using proven legacy patterns converted
 * from the original BigInt cost model to the new ResourceUsage format.
 */
export class LegacyCostEstimator {
    // EXTRACTED FROM LEGACY: Cost constants converted from BigInt
    private static readonly LEGACY_COST_MAPPING = {
        BASE_REASONING_COST: 50,
        TOKEN_COST_MULTIPLIER: 1,
        TOOL_REASONING_COST: 20,
        COMPLEXITY_FACTOR: 3,
        DEFAULT_TOKEN_ESTIMATE: 750,
    };

    /**
     * EXTRACTED FROM LEGACY: estimateCost converted to ResourceUsage
     * Estimates resources using legacy cost calculation patterns
     */
    estimateResources(context: ExecutionContext): ResourceUsage {
        // LEGACY PATTERN: Cost calculation adapted from legacy estimateCost method
        const baseCost = LegacyCostEstimator.LEGACY_COST_MAPPING.BASE_REASONING_COST;
        const inputCount = Object.keys(context.inputs).length;
        const outputCount = Object.keys(context.config.expectedOutputs || {}).length;

        // LEGACY LOGIC: Estimate tokens based on routine complexity
        const estimatedTokens = this.estimateTokenUsage(context);
        const tokenCost = estimatedTokens * LegacyCostEstimator.LEGACY_COST_MAPPING.TOKEN_COST_MULTIPLIER;

        // LEGACY LOGIC: Additional cost for complex reasoning
        const complexityCost = (inputCount + outputCount) * LegacyCostEstimator.LEGACY_COST_MAPPING.COMPLEXITY_FACTOR;

        // LEGACY PATTERN: Tool usage cost estimation
        const toolCost = this.estimateToolCost(context);

        const totalCost = baseCost + tokenCost + complexityCost + toolCost;

        // Convert legacy BigInt cost model to new ResourceUsage
        return {
            tokens: estimatedTokens,
            apiCalls: this.estimateApiCalls(context), // LEGACY PATTERN: 4-phase execution
            computeTime: this.estimateComputeTime(context), // Based on complexity
            cost: totalCost * 0.00001, // Convert to currency units
        };
    }

    /**
     * EXTRACTED FROM LEGACY: estimateTokenUsage adapted for new context
     * Estimates token usage based on input/output complexity and routine type
     */
    private estimateTokenUsage(context: ExecutionContext): number {
        // LEGACY LOGIC: Simple estimation adapted for new context structure
        const baseTokens = 500; // LEGACY: Base tokens for reasoning
        const inputCount = Object.keys(context.inputs).length;
        const expectedOutputCount = Object.keys(context.config.expectedOutputs || {}).length;
        
        // LEGACY PATTERN: Tokens per I/O field
        const ioTokens = (inputCount + expectedOutputCount) * 50;
        
        // LEGACY PATTERN: Additional tokens for routine processing
        const routineComplexity = this.calculateRoutineComplexity(context);
        const complexityTokens = routineComplexity * 100;
        
        // LEGACY PATTERN: Step type specific adjustments
        const stepTypeMultiplier = this.getStepTypeTokenMultiplier(context.stepType);
        
        const totalTokens = (baseTokens + ioTokens + complexityTokens) * stepTypeMultiplier;
        
        // LEGACY FALLBACK: Ensure minimum token estimate
        return Math.max(totalTokens, LegacyCostEstimator.LEGACY_COST_MAPPING.DEFAULT_TOKEN_ESTIMATE);
    }

    /**
     * LEGACY PATTERN: Calculate routine complexity based on various factors
     */
    private calculateRoutineComplexity(context: ExecutionContext): number {
        let complexity = 1;

        // LEGACY LOGIC: More inputs = more complex
        const inputCount = Object.keys(context.inputs).length;
        complexity += inputCount * 0.2;

        // LEGACY LOGIC: More expected outputs = more complex
        const outputCount = Object.keys(context.config.expectedOutputs || {}).length;
        complexity += outputCount * 0.3;

        // LEGACY LOGIC: Historical context adds complexity
        const historyWeight = 0.1;
        const maxHistoryComplexity = 0.5;
        complexity += Math.min(context.history.recentSteps.length * historyWeight, maxHistoryComplexity);

        // LEGACY LOGIC: Constraints add complexity
        if (context.constraints.requiredConfidence) {
            complexity += 0.5;
        }
        if (context.constraints.maxExecutionTime) {
            complexity += 0.3;
        }

        return Math.min(complexity, 3); // LEGACY PATTERN: Cap at 3x
    }

    /**
     * LEGACY PATTERN: Get token multiplier based on step type
     */
    private getStepTypeTokenMultiplier(stepType: string): number {
        const normalizedType = stepType.toLowerCase();
        
        // LEGACY LOGIC: More complex reasoning types require more tokens
        if (normalizedType.includes("analyze") || normalizedType.includes("evaluate")) {
            return 1.5; // Analysis requires more detailed reasoning
        }
        
        if (normalizedType.includes("decide") || normalizedType.includes("choose")) {
            return 1.3; // Decision making requires structured comparison
        }
        
        if (normalizedType.includes("validate") || normalizedType.includes("verify")) {
            return 1.2; // Validation requires checking logic
        }
        
        if (normalizedType.includes("reason") || normalizedType.includes("deduce")) {
            return 1.4; // Pure reasoning requires extensive logic
        }
        
        return 1.0; // Default multiplier
    }

    /**
     * LEGACY PATTERN: Estimate tool usage cost
     */
    private estimateToolCost(context: ExecutionContext): number {
        const availableTools = context.resources?.tools || [];
        const toolCount = availableTools.length;
        
        if (toolCount === 0) {
            return 0; // No tools, no tool cost
        }
        
        // LEGACY LOGIC: Base tool cost plus per-tool cost
        const baseToolCost = LegacyCostEstimator.LEGACY_COST_MAPPING.TOOL_REASONING_COST;
        const perToolCost = toolCount * 5; // Additional cost per available tool
        
        // LEGACY PATTERN: Some step types are more likely to use tools
        const toolUsageProbability = this.getToolUsageProbability(context.stepType);
        
        return (baseToolCost + perToolCost) * toolUsageProbability;
    }

    /**
     * LEGACY PATTERN: Estimate tool usage probability based on step type
     */
    private getToolUsageProbability(stepType: string): number {
        const normalizedType = stepType.toLowerCase();
        
        // LEGACY LOGIC: Some reasoning types are more likely to use tools
        if (normalizedType.includes("calculate") || normalizedType.includes("compute")) {
            return 0.8; // High probability for calculation tasks
        }
        
        if (normalizedType.includes("analyze") || normalizedType.includes("evaluate")) {
            return 0.6; // Medium-high probability for analysis
        }
        
        if (normalizedType.includes("validate") || normalizedType.includes("verify")) {
            return 0.7; // High probability for validation tasks
        }
        
        return 0.4; // Default probability
    }

    /**
     * LEGACY PATTERN: Estimate API calls based on 4-phase execution
     */
    private estimateApiCalls(context: ExecutionContext): number {
        // LEGACY PATTERN: 4-phase execution typically requires multiple API calls
        let apiCalls = 4; // Analysis + Planning + Execution + Refinement
        
        // LEGACY LOGIC: Complex tasks might require additional calls
        const complexity = this.calculateRoutineComplexity(context);
        if (complexity > 2) {
            apiCalls += 1; // Additional call for complex tasks
        }
        
        // LEGACY LOGIC: Tool usage might require additional calls
        const toolCount = (context.resources?.tools || []).length;
        if (toolCount > 0) {
            apiCalls += Math.min(toolCount, 2); // Up to 2 additional calls for tools
        }
        
        return apiCalls;
    }

    /**
     * LEGACY PATTERN: Estimate compute time based on complexity
     */
    private estimateComputeTime(context: ExecutionContext): number {
        const baseTime = 15000; // 15 seconds base time
        const complexity = this.calculateRoutineComplexity(context);
        
        // LEGACY LOGIC: More complex tasks take longer
        const complexityTime = complexity * 5000; // 5 seconds per complexity point
        
        // LEGACY LOGIC: More I/O requires more processing time
        const inputCount = Object.keys(context.inputs).length;
        const outputCount = Object.keys(context.config.expectedOutputs || {}).length;
        const ioTime = (inputCount + outputCount) * 1000; // 1 second per I/O field
        
        const totalTime = baseTime + complexityTime + ioTime;
        
        // LEGACY PATTERN: Cap at reasonable maximum
        return Math.min(totalTime, 60000); // Max 60 seconds
    }

    /**
     * Creates estimated resource usage for tracking purposes
     */
    createResourceUsageSnapshot(
        actualTokens: number,
        actualApiCalls: number,
        actualComputeTime: number,
        actualCreditsUsed: number
    ): ResourceUsage {
        return {
            tokens: actualTokens,
            apiCalls: actualApiCalls,
            computeTime: actualComputeTime,
            cost: actualCreditsUsed * 0.00001, // Convert credits to cost
        };
    }
}