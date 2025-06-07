import { type Logger } from "winston";
import {
    type ExecutionContext,
    type ExecutionStrategy,
    type StrategyExecutionResult,
    type ResourceUsage,
    type StrategyFeedback,
    type StrategyPerformance,
    StrategyType,
} from "@vrooli/shared";
import { LLMIntegrationService } from "../../integration/llmIntegrationService.js";
import { LegacyInputHandler } from "./reasoning/legacyInputHandler.js";
import { FourPhaseExecutor } from "./reasoning/fourPhaseExecutor.js";
import { LegacyOutputGenerator } from "./reasoning/legacyOutputGenerator.js";
import { LegacyCostEstimator } from "./reasoning/legacyCostEstimator.js";

/**
 * Reasoning framework types
 */
interface ReasoningFramework {
    type: "logical" | "analytical" | "decision_tree" | "evidence_based";
    steps: ReasoningStep[];
}

interface ReasoningStep {
    id: string;
    type: string;
    description: string;
    inputs: string[];
    outputs: string[];
    validation?: ValidationRule[];
}

interface ValidationRule {
    type: "consistency" | "completeness" | "logic" | "bias";
    check: (data: unknown) => boolean;
    message: string;
}

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
 * ReasoningStrategy - Structured analytical execution for complex decision-making
 * 
 * This strategy applies structured, logic-driven frameworks to tasks once basic
 * conversational patterns are known. It's ideal for multi-step decision trees,
 * data-driven analysis, and evidence-based conclusions.
 * 
 * Key characteristics:
 * - Emphasizes consistency, traceability, and justifiable outputs
 * - Combines deterministic sub-routines with LLM assistance
 * - Balances human interpretability with partial automation
 * 
 * Illustrative capabilities:
 * - Analytical Frameworks (logical structures, data analysis, decision trees)
 * - Knowledge Integration (fact retrieval, concept mapping, evidence synthesis)
 * - Quality Assurance (logic validation, bias detection, confidence scoring)
 */
export class ReasoningStrategy implements ExecutionStrategy {
    readonly type = StrategyType.REASONING;
    readonly name = "ReasoningStrategy";
    readonly version = "2.0.0";

    private readonly logger: Logger;
    private readonly llmService: LLMIntegrationService;
    
    // LEGACY INTEGRATION: Modular components for legacy patterns
    private readonly inputHandler: LegacyInputHandler;
    private readonly phaseExecutor: FourPhaseExecutor;
    private readonly outputGenerator: LegacyOutputGenerator;
    private readonly costEstimator: LegacyCostEstimator;
    
    // Configuration constants
    private readonly COMPLEXITY_MULTIPLIER = 500;
    private readonly CONFIDENCE_THRESHOLD = 0.7;
    private readonly SCORING_FACTOR = 0.2;
    private readonly PENALTY_FACTOR = 0.1;
    private readonly CONSTRAINT_COMPLEXITY = 0.5;
    private readonly BASE_SCORE = 0.6;

    constructor(logger: Logger) {
        this.logger = logger;
        this.llmService = new LLMIntegrationService(logger);
        
        // LEGACY INTEGRATION: Initialize modular components
        this.inputHandler = new LegacyInputHandler(logger);
        this.phaseExecutor = new FourPhaseExecutor(logger);
        this.outputGenerator = new LegacyOutputGenerator(logger);
        this.costEstimator = new LegacyCostEstimator();
    }

    /**
     * LEGACY INTEGRATION: Enhanced execution using 4-phase pattern from legacy ReasoningStrategy
     * 
     * Phases extracted from legacy implementation:
     * 1. Analyze (analyzeRoutine) - Understand task requirements and complexity
     * 2. Plan (planExecution) - Create structured execution approach
     * 3. Execute (executeWithReasoning) - Perform reasoning with AI assistance
     * 4. Refine (refineResults) - Validate and enhance outputs
     */
    async execute(context: ExecutionContext): Promise<StrategyExecutionResult> {
        const startTime = Date.now();
        const stepId = context.stepId;
        let creditsUsed = 0;
        let toolCallsCount = 0;

        this.logger.info("[ReasoningStrategy] Starting 4-phase reasoning execution", {
            stepId,
            stepType: context.stepType,
        });

        try {
            // LEGACY INTEGRATION: Check for missing inputs and generate intelligently
            const processedContext = await this.inputHandler.handleMissingInputs(context);
            
            // LEGACY PHASE 1: Analyze task (extracted from analyzeRoutine)
            const analysisResult = await this.phaseExecutor.analyzeTask(processedContext);
            creditsUsed += analysisResult.creditsUsed;
            toolCallsCount += analysisResult.toolCallsCount;

            // LEGACY PHASE 2: Plan reasoning approach (extracted from planExecution)
            const planningResult = await this.phaseExecutor.planReasoningApproach(
                processedContext, 
                analysisResult.insights
            );
            creditsUsed += planningResult.creditsUsed;
            toolCallsCount += planningResult.toolCallsCount;

            // LEGACY PHASE 3: Execute reasoning logic (extracted from executeWithReasoning)
            const executionResult = await this.phaseExecutor.executeReasoningLogic(
                processedContext,
                planningResult.executionPlan
            );
            creditsUsed += executionResult.creditsUsed;
            toolCallsCount += executionResult.toolCallsCount;

            // LEGACY PHASE 4: Refine and validate (extracted from refineResults)
            const refinementResult = await this.phaseExecutor.refineAndValidate(
                processedContext,
                executionResult,
                planningResult.executionPlan
            );
            creditsUsed += refinementResult.creditsUsed;
            toolCallsCount += refinementResult.toolCallsCount;

            // LEGACY INTEGRATION: Create success result with legacy metadata patterns
            return this.outputGenerator.createLegacyCompatibleSuccessResult(
                refinementResult.outputs,
                creditsUsed,
                toolCallsCount,
                Date.now() - startTime,
                analysisResult,
                planningResult,
                refinementResult
            );

        } catch (error) {
            this.logger.error("[ReasoningStrategy] 4-phase execution failed", {
                stepId,
                error: error instanceof Error ? error.message : String(error),
            });

            // LEGACY INTEGRATION: Enhanced error handling
            return this.outputGenerator.createLegacyCompatibleErrorResult(
                error as Error,
                creditsUsed,
                toolCallsCount,
                Date.now() - startTime
            );
        }
    }

    /**
     * Checks if this strategy can handle the given step type
     */
    canHandle(stepType: string, config?: Record<string, unknown>): boolean {
        // Check explicit strategy request
        if (config?.strategy === "reasoning") {
            return true;
        }

        // Check for reasoning keywords in step type
        const reasoningKeywords = [
            "analyze", "reason", "evaluate", "assess",
            "decide", "compare", "synthesize", "deduce",
            "infer", "conclude", "calculate", "optimize",
            "validate", "verify", "justify", "explain",
        ];

        const normalizedType = stepType.toLowerCase();
        return reasoningKeywords.some(keyword => normalizedType.includes(keyword));
    }

    /**
     * LEGACY INTEGRATION: Resource estimation using legacy cost model converted to ResourceUsage
     */
    estimateResources(context: ExecutionContext): ResourceUsage {
        return this.costEstimator.estimateResources(context);
    }

    /**
     * Learning method
     */
    learn(feedback: import("@vrooli/shared").StrategyFeedback): void {
        this.logger.info("[ReasoningStrategy] Learning from feedback", {
            outcome: feedback.outcome,
            performance: feedback.performanceScore,
        });
        // TODO: Implement actual learning mechanism
    }

    /**
     * Returns performance metrics
     */
    getPerformanceMetrics(): import("@vrooli/shared").StrategyPerformance {
        // TODO: Implement actual metrics tracking
        return {
            totalExecutions: 0,
            successCount: 0,
            failureCount: 0,
            averageExecutionTime: 0,
            averageResourceUsage: {},
            averageConfidence: 0,
            evolutionScore: 0.5, // Medium evolution
        };
    }

    /**
     * LEGACY INTEGRATION: Reasoning framework selection (preserved for new architecture compatibility)
     */
    private selectReasoningFramework(context: ExecutionContext): ReasoningFramework {
        const stepType = context.stepType.toLowerCase();

        // Decision tree for choice-based tasks
        if (stepType.includes("decide") || stepType.includes("choose")) {
            return this.createDecisionTreeFramework(context);
        }

        // Analytical framework for analysis tasks
        if (stepType.includes("analyze") || stepType.includes("evaluate")) {
            return this.createAnalyticalFramework(context);
        }

        // Evidence-based for validation/verification
        if (stepType.includes("validate") || stepType.includes("verify")) {
            return this.createEvidenceBasedFramework(context);
        }

        // Default to logical framework
        return this.createLogicalFramework(context);
    }

    private createLogicalFramework(_context: ExecutionContext): ReasoningFramework {
        return {
            type: "logical",
            steps: [
                {
                    id: "premises",
                    type: "identify_premises",
                    description: "Identify key premises and assumptions",
                    inputs: ["context", "inputs"],
                    outputs: ["premises", "assumptions"],
                },
                {
                    id: "rules",
                    type: "apply_rules",
                    description: "Apply logical rules and inference",
                    inputs: ["premises"],
                    outputs: ["inferences"],
                },
                {
                    id: "conclusion",
                    type: "draw_conclusion",
                    description: "Draw logical conclusion",
                    inputs: ["inferences", "premises"],
                    outputs: ["conclusion", "confidence"],
                },
            ],
        };
    }

    private createAnalyticalFramework(_context: ExecutionContext): ReasoningFramework {
        return {
            type: "analytical",
            steps: [
                {
                    id: "data_gathering",
                    type: "gather_data",
                    description: "Collect relevant data points",
                    inputs: ["context", "sources"],
                    outputs: ["data_points"],
                },
                {
                    id: "pattern_analysis",
                    type: "analyze_patterns",
                    description: "Identify patterns and trends",
                    inputs: ["data_points"],
                    outputs: ["patterns", "anomalies"],
                },
                {
                    id: "synthesis",
                    type: "synthesize_findings",
                    description: "Synthesize findings into insights",
                    inputs: ["patterns", "anomalies"],
                    outputs: ["insights", "recommendations"],
                },
            ],
        };
    }

    private createDecisionTreeFramework(context: ExecutionContext): ReasoningFramework {
        return {
            type: "decision_tree",
            steps: [
                {
                    id: "options",
                    type: "identify_options",
                    description: "Identify available options",
                    inputs: ["context", "constraints"],
                    outputs: ["options"],
                },
                {
                    id: "criteria",
                    type: "establish_criteria",
                    description: "Establish decision criteria",
                    inputs: ["goals", "constraints"],
                    outputs: ["criteria", "weights"],
                },
                {
                    id: "evaluation",
                    type: "evaluate_options",
                    description: "Evaluate options against criteria",
                    inputs: ["options", "criteria"],
                    outputs: ["scores", "rankings"],
                },
                {
                    id: "selection",
                    type: "select_option",
                    description: "Select optimal option",
                    inputs: ["rankings", "constraints"],
                    outputs: ["selection", "justification"],
                },
            ],
        };
    }

    private createEvidenceBasedFramework(context: ExecutionContext): ReasoningFramework {
        return {
            type: "evidence_based",
            steps: [
                {
                    id: "hypothesis",
                    type: "form_hypothesis",
                    description: "Form initial hypothesis",
                    inputs: ["context", "claim"],
                    outputs: ["hypothesis"],
                },
                {
                    id: "evidence_gathering",
                    type: "gather_evidence",
                    description: "Gather supporting and contradicting evidence",
                    inputs: ["hypothesis", "sources"],
                    outputs: ["evidence_for", "evidence_against"],
                },
                {
                    id: "evaluation",
                    type: "evaluate_evidence",
                    description: "Evaluate evidence quality and relevance",
                    inputs: ["evidence_for", "evidence_against"],
                    outputs: ["evidence_strength", "gaps"],
                },
                {
                    id: "verdict",
                    type: "reach_verdict",
                    description: "Reach evidence-based conclusion",
                    inputs: ["evidence_strength", "hypothesis"],
                    outputs: ["verdict", "confidence", "caveats"],
                },
            ],
        };
    }

    private async buildKnowledgeBase(
        context: ExecutionContext,
    ): Promise<Map<string, unknown>> {
        const knowledgeBase = new Map<string, unknown>();

        // Add context inputs
        knowledgeBase.set("inputs", context.inputs);
        knowledgeBase.set("config", context.config);
        knowledgeBase.set("constraints", context.constraints);

        // Add historical context
        if (context.history.recentSteps.length > 0) {
            knowledgeBase.set("history", context.history);
        }

        // TODO: Add fact retrieval from knowledge stores

        return knowledgeBase;
    }

    private async executeReasoningFramework(
        framework: ReasoningFramework,
        context: ExecutionContext,
        knowledgeBase: Map<string, unknown>,
    ): Promise<ReasoningResult> {
        const reasoning: string[] = [];
        const evidence: Evidence[] = [];
        const assumptions: string[] = [];
        const workingMemory = new Map(knowledgeBase);

        // Execute each step in the framework
        for (const step of framework.steps) {
            this.logger.debug(`[ReasoningStrategy] Executing step: ${step.id}`, {
                type: step.type,
                inputs: step.inputs,
            });

            // Gather inputs for this step
            const stepInputs = this.gatherStepInputs(step, workingMemory);

            // Execute step logic
            const stepResults = await this.executeReasoningStep(
                step,
                stepInputs,
                context,
            );

            // Store outputs in working memory
            for (const output of step.outputs) {
                workingMemory.set(output, stepResults[output]);
            }

            // Record reasoning trace
            reasoning.push(`${step.description}: ${JSON.stringify(stepResults)}`);

            // Extract evidence and assumptions
            if (stepResults.evidence) {
                evidence.push(...(stepResults.evidence as Evidence[]));
            }
            if (stepResults.assumptions) {
                assumptions.push(...(stepResults.assumptions as string[]));
            }
        }

        // Extract final conclusion
        const conclusion = workingMemory.get("conclusion") ||
                         workingMemory.get("selection") ||
                         workingMemory.get("verdict") ||
                         workingMemory.get("insights");

        const confidence = (workingMemory.get("confidence") as number) || this.CONFIDENCE_THRESHOLD;

        return {
            conclusion,
            reasoning,
            evidence,
            confidence,
            assumptions,
        };
    }

    private gatherStepInputs(
        step: ReasoningStep,
        workingMemory: Map<string, unknown>,
    ): Record<string, unknown> {
        const inputs: Record<string, unknown> = {};

        for (const inputKey of step.inputs) {
            inputs[inputKey] = workingMemory.get(inputKey);
        }

        return inputs;
    }

    private async executeReasoningStep(
        step: ReasoningStep,
        inputs: Record<string, unknown>,
        context: ExecutionContext,
    ): Promise<Record<string, unknown>> {
        this.logger.debug("[ReasoningStrategy] Executing reasoning step", {
            stepType: step.type,
            description: step.description,
        });

        try {
            // Build a reasoning prompt for the LLM
            const prompt = this.buildReasoningPrompt(step, inputs, context);

            // Prepare LLM request
            const llmRequest = {
                model: context.config.model as string || "gpt-4o-mini",
                messages: [
                    {
                        role: "user" as const,
                        content: prompt,
                    },
                ],
                systemMessage: "You are an analytical reasoning assistant. Provide structured, logical analysis with clear justification for your conclusions.",
                maxTokens: 1000,
                temperature: 0.3, // Low temperature for consistent reasoning
            };

            // Get available resources
            const availableResources = {
                maxCredits: context.resources?.credits || 5000,
                maxTokens: 1000,
                maxTime: 15000,
                tools: context.resources?.tools || [],
            };

            // Get user data
            const userData = context.userId ? {
                id: context.userId,
                name: context.config.userName as string,
            } : undefined;

            // Execute LLM reasoning
            const response = await this.llmService.executeRequest(
                llmRequest,
                availableResources,
                userData,
            );

            // Parse the response into structured outputs
            return this.parseReasoningResponse(response.content, step);

        } catch (error) {
            this.logger.warn("[ReasoningStrategy] LLM reasoning failed, using fallback", {
                stepType: step.type,
                error: error instanceof Error ? error.message : String(error),
            });

            // Fallback to simulated reasoning
            return this.getSimulatedReasoningResult(step, inputs);
        }
    }

    private buildReasoningPrompt(
        step: ReasoningStep,
        inputs: Record<string, unknown>,
        context: ExecutionContext,
    ): string {
        const parts: string[] = [];

        // Add step description and goal
        parts.push(`Task: ${step.description}`);
        parts.push(`Goal: ${step.type.replace(/_/g, " ")}`);

        // Add available inputs
        parts.push("Available Information:");
        for (const [key, value] of Object.entries(inputs)) {
            if (value !== undefined && value !== null) {
                parts.push(`- ${key}: ${JSON.stringify(value)}`);
            }
        }

        // Add context if relevant
        if (context.inputs && Object.keys(context.inputs).length > 0) {
            parts.push("Original Context:");
            for (const [key, value] of Object.entries(context.inputs)) {
                parts.push(`- ${key}: ${JSON.stringify(value)}`);
            }
        }

        // Add specific instructions based on step type
        parts.push(this.getStepSpecificInstructions(step));

        // Request structured output
        parts.push("Provide your analysis in a structured format with clear reasoning.");

        return parts.join("\n\n");
    }

    private getStepSpecificInstructions(step: ReasoningStep): string {
        switch (step.type) {
            case "identify_premises":
                return "Instructions:\n1. Identify the key premises from the available information\n2. List any assumptions being made\n3. Note any missing information that might be needed";

            case "apply_rules":
                return "Instructions:\n1. Apply logical rules to the premises\n2. Make valid inferences\n3. Avoid logical fallacies";

            case "draw_conclusion":
                return "Instructions:\n1. Synthesize the inferences into a clear conclusion\n2. Assess your confidence level (0.0-1.0)\n3. Justify your reasoning";

            case "gather_data":
                return "Instructions:\n1. Extract relevant data points\n2. Assess the relevance of each data point\n3. Identify any data quality issues";

            case "analyze_patterns":
                return "Instructions:\n1. Look for patterns in the data\n2. Identify any anomalies or outliers\n3. Assess the significance of patterns found";

            default:
                return "Instructions:\nAnalyze the information systematically and provide clear, logical reasoning.";
        }
    }

    private parseReasoningResponse(content: string, step: ReasoningStep): Record<string, unknown> {
        try {
            // Try to extract structured information from the response
            const result: Record<string, unknown> = {};

            // Basic parsing for different step types
            switch (step.type) {
                case "identify_premises": {
                    const premises = this.extractListFromText(content, ["premise", "given", "fact"]);
                    const assumptions = this.extractListFromText(content, ["assumption", "assume", "presuming"]);
                    result.premises = premises;
                    result.assumptions = assumptions;
                    break;
                }

                case "apply_rules": {
                    const inferences = this.extractListFromText(content, ["inference", "therefore", "conclude", "implies"]);
                    result.inferences = inferences;
                    break;
                }

                case "draw_conclusion": {
                    result.conclusion = this.extractConclusion(content);
                    result.confidence = this.extractConfidence(content);
                    break;
                }

                case "gather_data": {
                    result.data_points = this.extractDataPoints(content);
                    break;
                }

                case "analyze_patterns": {
                    const patterns = this.extractListFromText(content, ["pattern", "trend", "consistent"]);
                    const anomalies = this.extractListFromText(content, ["anomaly", "outlier", "unusual", "exception"]);
                    result.patterns = patterns;
                    result.anomalies = anomalies;
                    break;
                }

                default:
                    result.result = content;
                    result.reasoning = content;
            }

            return result;

        } catch (error) {
            this.logger.warn("[ReasoningStrategy] Failed to parse reasoning response", {
                stepType: step.type,
                error: error instanceof Error ? error.message : String(error),
            });

            // Return the raw content if parsing fails
            return { result: content, reasoning: content };
        }
    }

    private extractListFromText(text: string, keywords: string[]): string[] {
        const items: string[] = [];
        const lines = text.split("\n");

        for (const line of lines) {
            const trimmed = line.trim();
            
            // Look for bullet points or numbered lists
            if (trimmed.match(/^[-*•]\s+/) || trimmed.match(/^\d+\.\s+/)) {
                const content = trimmed.replace(/^[-*•]\s+/, "").replace(/^\d+\.\s+/, "");
                
                // Check if this item relates to our keywords
                const lowerContent = content.toLowerCase();
                if (keywords.some(keyword => lowerContent.includes(keyword)) || items.length === 0) {
                    items.push(content);
                }
            }
        }

        return items.length > 0 ? items : [text.trim()];
    }

    private extractConclusion(text: string): string {
        // Look for conclusion indicators
        const conclusionPatterns = [
            /conclusion:?\s*(.+)/i,
            /therefore,?\s*(.+)/i,
            /in conclusion,?\s*(.+)/i,
            /the result is:?\s*(.+)/i,
        ];

        for (const pattern of conclusionPatterns) {
            const match = text.match(pattern);
            if (match) {
                return match[1].trim();
            }
        }

        // Fallback: use the last sentence or paragraph
        const sentences = text.split(/[.!?]+/).filter(s => s.trim().length > 0);
        return sentences[sentences.length - 1]?.trim() || text.trim();
    }

    private extractConfidence(text: string): number {
        // Look for confidence indicators
        const confidencePatterns = [
            /confidence:?\s*([0-9.]+)/i,
            /([0-9.]+)\s*confidence/i,
            /([0-9]{1,3})%/,
        ];

        for (const pattern of confidencePatterns) {
            const match = text.match(pattern);
            if (match) {
                const value = parseFloat(match[1]);
                return value > 1 ? value / 100 : value; // Convert percentage if needed
            }
        }

        // Heuristic based on language
        const lowerText = text.toLowerCase();
        if (lowerText.includes("certain") || lowerText.includes("definite")) {
            return 0.9;
        } else if (lowerText.includes("likely") || lowerText.includes("probable")) {
            return 0.7;
        } else if (lowerText.includes("possible") || lowerText.includes("might")) {
            return 0.5;
        } else if (lowerText.includes("uncertain") || lowerText.includes("unclear")) {
            return 0.3;
        }

        return 0.7; // Default confidence
    }

    private extractDataPoints(text: string): Array<{ key: string; value: unknown; relevance: number }> {
        const dataPoints: Array<{ key: string; value: unknown; relevance: number }> = [];
        const lines = text.split("\n");

        for (const line of lines) {
            const trimmed = line.trim();
            
            // Look for key-value patterns
            const kvMatch = trimmed.match(/(.+?):\s*(.+)/);
            if (kvMatch) {
                dataPoints.push({
                    key: kvMatch[1].trim(),
                    value: kvMatch[2].trim(),
                    relevance: 0.8, // Default relevance
                });
            }
        }

        return dataPoints.length > 0 ? dataPoints : [{
            key: "analysis",
            value: text.trim(),
            relevance: 0.7,
        }];
    }

    private getSimulatedReasoningResult(step: ReasoningStep, inputs: Record<string, unknown>): Record<string, unknown> {
        // Fallback to original simulation logic
        switch (step.type) {
            case "identify_premises":
                return {
                    premises: ["Input data is valid", "Context is complete"],
                    assumptions: ["No hidden constraints"],
                };

            case "apply_rules":
                return {
                    inferences: ["Data suggests positive outcome", "No contradictions found"],
                };

            case "draw_conclusion":
                return {
                    conclusion: "Task can be completed successfully",
                    confidence: 0.85,
                };

            case "gather_data":
                return {
                    data_points: Object.entries(inputs).map(([k, v]) => ({
                        key: k,
                        value: v,
                        relevance: 0.8,
                    })),
                };

            case "analyze_patterns":
                return {
                    patterns: ["Consistent input structure", "Normal parameter ranges"],
                    anomalies: [],
                };

            default:
                return {
                    result: `Executed ${step.type}`,
                    ...inputs,
                };
        }
    }

    private async validateReasoning(
        result: ReasoningResult,
        framework: ReasoningFramework,
    ): Promise<ValidationResult[]> {
        const validations: ValidationResult[] = [];

        // Logic validation
        validations.push({
            type: "logic",
            passed: this.validateLogicalConsistency(result),
            message: "Logical consistency check",
        });

        // Completeness validation
        validations.push({
            type: "completeness",
            passed: this.validateCompleteness(result, framework),
            message: "Reasoning completeness check",
        });

        // Bias detection
        validations.push({
            type: "bias",
            passed: this.detectBias(result) === null,
            message: "Bias detection check",
        });

        return validations;
    }

    private validateLogicalConsistency(result: ReasoningResult): boolean {
        // Check for contradictions in reasoning
        const reasoningText = result.reasoning.join(" ");
        
        // Simple contradiction patterns
        const contradictionPatterns = [
            /both true and false/i,
            /contradicts earlier/i,
            /inconsistent with/i,
        ];

        return !contradictionPatterns.some(pattern => pattern.test(reasoningText));
    }

    private validateCompleteness(
        result: ReasoningResult,
        framework: ReasoningFramework,
    ): boolean {
        // Check if all framework steps produced outputs
        const expectedOutputs = framework.steps.flatMap(s => s.outputs);
        const hasConclusion = result.conclusion !== undefined;
        const hasReasoning = result.reasoning.length >= framework.steps.length;

        return hasConclusion && hasReasoning;
    }

    private detectBias(result: ReasoningResult): string | null {
        // Simple bias detection
        const biasIndicators = [
            { pattern: /always|never|all|none/i, type: "absolute thinking" },
            { pattern: /obviously|clearly|definitely/i, type: "overconfidence" },
            { pattern: /probably not|unlikely|doubt/i, type: "negative bias" },
        ];

        const reasoningText = result.reasoning.join(" ");
        
        for (const indicator of biasIndicators) {
            if (indicator.pattern.test(reasoningText)) {
                return indicator.type;
            }
        }

        return null;
    }

    private formatReasoningOutputs(
        result: ReasoningResult,
        validations: ValidationResult[],
        context: ExecutionContext,
    ): Record<string, unknown> {
        const outputs: Record<string, unknown> = {
            conclusion: result.conclusion,
            confidence: result.confidence,
            reasoning_steps: result.reasoning,
        };

        // Add evidence if requested
        if (context.config.includeEvidence) {
            outputs.evidence = result.evidence;
        }

        // Add assumptions if any
        if (result.assumptions.length > 0) {
            outputs.assumptions = result.assumptions;
        }

        // Add validation summary
        outputs.validation = {
            passed: validations.every(v => v.passed),
            checks: validations.map(v => ({
                type: v.type,
                passed: v.passed,
                message: v.message,
            })),
        };

        return outputs;
    }

    private calculateResourceUsage(
        framework: ReasoningFramework,
        result: ReasoningResult,
        executionTime: number,
    ): ResourceUsage {
        const baseTokens = 500;
        const tokensPerStep = 300;
        const tokensPerEvidence = 50;

        return {
            tokens: baseTokens + 
                   (framework.steps.length * tokensPerStep) +
                   (result.evidence.length * tokensPerEvidence),
            apiCalls: 1, // Typically one call for structured reasoning
            computeTime: executionTime,
            cost: 0.03, // Moderate cost for reasoning
        };
    }

    private calculatePerformanceScore(
        result: ReasoningResult,
        validations: ValidationResult[],
    ): number {
        let score = this.BASE_SCORE; // Base score

        // Confidence contribution
        score += result.confidence * this.SCORING_FACTOR;

        // Validation contribution
        const validationScore = validations.filter(v => v.passed).length / validations.length;
        score += validationScore * this.PENALTY_FACTOR;

        // Evidence quality contribution
        if (result.evidence.length > 0) {
            const avgEvidenceConfidence = result.evidence.reduce(
                (sum, e) => sum + e.confidence, 0,
            ) / result.evidence.length;
            score += avgEvidenceConfidence * this.PENALTY_FACTOR;
        }

        return Math.min(1, score);
    }

    private suggestImprovements(
        result: ReasoningResult,
        validations: ValidationResult[],
    ): string[] {
        const improvements: string[] = [];

        // Check validation failures
        const failedValidations = validations.filter(v => !v.passed);
        if (failedValidations.length > 0) {
            improvements.push(`Address validation issues: ${failedValidations.map(v => v.type).join(", ")}`);
        }

        // Check confidence
        if (result.confidence < this.CONFIDENCE_THRESHOLD) {
            improvements.push("Gather more evidence to increase confidence");
        }

        // Check assumptions
        if (result.assumptions.length > 3) {
            improvements.push("Reduce number of assumptions by gathering more facts");
        }

        return improvements;
    }

    private estimateComplexity(context: ExecutionContext): number {
        let complexity = 1;

        // More inputs = more complex
        complexity += Object.keys(context.inputs).length * this.PENALTY_FACTOR;

        // More constraints = more complex
        if (context.constraints.requiredConfidence) {
            complexity += this.CONSTRAINT_COMPLEXITY;
        }

        // Historical context adds complexity
        const HISTORY_COMPLEXITY_WEIGHT = 0.1;
        const MAX_HISTORY_COMPLEXITY = 0.5;
        complexity += Math.min(context.history.recentSteps.length * HISTORY_COMPLEXITY_WEIGHT, MAX_HISTORY_COMPLEXITY);

        return Math.min(complexity, 3); // Cap at 3x
    }
}

/**
 * Internal types for reasoning strategy
 */
interface ValidationResult {
    type: string;
    passed: boolean;
    message: string;
}
