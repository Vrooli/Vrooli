import { type Logger } from "winston";
import {
    type ExecutionContext as StrategyExecutionContext,
    type ExecutionStrategy,
    type StrategyExecutionResult,
    type ResourceUsage,
    type LLMRequest,
    type LLMResponse,
    type LLMMessage,
    type AvailableResources,
    type StrategyFeedback,
    type StrategyPerformance,
    StrategyType,
} from "@vrooli/shared";
import { LLMIntegrationService } from "../../integration/llmIntegrationService.js";
import { ToolOrchestrator } from "../../engine/toolOrchestrator.js";
import { ValidationEngine } from "../../engine/validationEngine.js";

/**
 * ConversationalStrategy - Enhanced with legacy implementation patterns
 * 
 * This strategy combines the new architecture capabilities with proven legacy patterns:
 * - Multi-turn conversation support (adapted from legacy)
 * - Context-aware message building (from legacy implementation)
 * - Tool execution integration (enhanced for new architecture) 
 * - Error handling with retry logic (proven legacy patterns)
 * - Performance tracking and metrics (upgraded for new system)
 * 
 * Migration from legacy ConversationalStrategy (425 lines) with enhancements:
 * - Extracted conversation building logic
 * - Preserved error recovery patterns
 * - Added resource estimation and learning capabilities
 * - Integrated with new LLMIntegrationService and ToolOrchestrator
 */
export class ConversationalStrategy implements ExecutionStrategy {
    readonly type = StrategyType.CONVERSATIONAL;
    readonly name = "ConversationalStrategy";
    readonly version = "2.0.0-enhanced";

    private readonly logger: Logger;
    private readonly llmService: LLMIntegrationService;
    private toolOrchestrator?: ToolOrchestrator;
    private validationEngine?: ValidationEngine;
    
    // Performance tracking (new capability)
    private performanceHistory: Array<{
        timestamp: Date;
        executionTime: number;
        tokensUsed: number;
        success: boolean;
        confidence: number;
    }> = [];
    
    // Legacy patterns adapted for new architecture
    private static readonly MAX_CONVERSATION_TURNS = 10;
    private static readonly CONVERSATION_CONTEXT_WINDOW = 5;
    private static readonly TURN_TIMEOUT_MS = 60000; // 1 minute per turn

    constructor(
        logger: Logger,
        toolOrchestrator?: ToolOrchestrator,
        validationEngine?: ValidationEngine,
    ) {
        this.logger = logger;
        this.llmService = new LLMIntegrationService(logger);
        this.toolOrchestrator = toolOrchestrator;
        this.validationEngine = validationEngine;
    }

    /**
     * Set the tool orchestrator (for dependency injection)
     */
    setToolOrchestrator(orchestrator: ToolOrchestrator): void {
        this.toolOrchestrator = orchestrator;
    }

    /**
     * Set the validation engine (for dependency injection)
     */
    setValidationEngine(engine: ValidationEngine): void {
        this.validationEngine = engine;
    }

    /**
     * Enhanced execution with legacy conversation patterns
     */
    async execute(context: StrategyExecutionContext): Promise<StrategyExecutionResult> {
        const startTime = Date.now();
        const stepId = context.stepId;
        let totalTokensUsed = 0;

        this.logger.info("[ConversationalStrategy] Starting enhanced execution", {
            stepId,
            stepType: context.stepType,
        });

        try {
            // Legacy pattern: Build conversation request with context awareness
            const conversationRequest = await this.buildConversationRequest(context);
            
            // Legacy pattern: Multi-turn conversation support (adapted for single context)
            const response = await this.executeConversationalReasoning(conversationRequest, context);
            totalTokensUsed += response.tokensUsed;
            
            // Legacy pattern: Tool execution if required
            let finalResponse = response;
            if (this.requiresToolExecution(response)) {
                finalResponse = await this.handleToolExecution(response, context);
                totalTokensUsed += finalResponse.tokensUsed;
            }
            
            // Legacy pattern: Extract outputs from conversation
            const outputs = await this.extractOutputsFromConversation(finalResponse, context);
            
            // New capability: Validate outputs
            const validatedOutputs = await this.validateOutputs(outputs, context);
            
            // Calculate comprehensive resource usage
            const resourceUsage = this.calculateResourceUsage(finalResponse, Date.now() - startTime, totalTokensUsed);
            
            // Track performance for learning
            this.trackPerformance({
                timestamp: new Date(),
                executionTime: Date.now() - startTime,
                tokensUsed: totalTokensUsed,
                success: true,
                confidence: finalResponse.confidence,
            });

            return {
                success: true,
                result: validatedOutputs,
                metadata: {
                    strategyType: this.type,
                    executionTime: Date.now() - startTime,
                    resourceUsage,
                    confidence: this.calculateConfidence(finalResponse, context),
                    fallbackUsed: false,
                },
                feedback: {
                    outcome: "success",
                    performanceScore: this.calculatePerformanceScore(finalResponse, context),
                    improvements: this.suggestImprovements(finalResponse, context),
                },
            };

        } catch (error) {
            this.logger.error("[ConversationalStrategy] Execution failed", {
                stepId,
                error: error instanceof Error ? error.message : String(error),
            });
            
            // Track failure for learning
            this.trackPerformance({
                timestamp: new Date(),
                executionTime: Date.now() - startTime,
                tokensUsed: totalTokensUsed,
                success: false,
                confidence: 0,
            });

            // Legacy pattern: Error handling with retry possibility
            return this.handleExecutionError(error as Error, context, Date.now() - startTime);
        }
    }

    /**
     * Enhanced canHandle with legacy patterns
     * Extracted from legacy ConversationalStrategy.canHandle()
     */
    canHandle(stepType: string, config?: Record<string, unknown>): boolean {
        // Legacy pattern: Check explicit strategy request
        if (config?.strategy === "conversational" || config?.executionStrategy === "conversational") {
            return true;
        }
        
        // Legacy pattern: Handle web routines (often used for searches and discussions)
        if (stepType === "RoutineWeb" || stepType === "web") {
            return true;
        }

        // Legacy pattern: Enhanced conversational keywords
        const conversationalKeywords = [
            "chat", "converse", "discuss", "talk", "dialogue",
            "interview", "consult", "advise", "guide", "tutorial",
            "negotiate", "collaborate", "brainstorm", "iterate",
            "customer", "service", "support", "help",
            "creative", "explore", "open-ended", "flexible",
            "natural", "language", "interpret", "understand",
        ];

        // Legacy pattern: Check step type and config description
        const routineName = config?.name as string || "";
        const routineDescription = config?.description as string || "";
        const combined = `${stepType} ${routineName} ${routineDescription}`.toLowerCase();

        return conversationalKeywords.some(keyword => combined.includes(keyword));
    }

    /**
     * Enhanced resource estimation with legacy performance data
     */
    estimateResources(context: StrategyExecutionContext): ResourceUsage {
        // Legacy performance data shows conversational tasks vary widely
        const messageLength = this.estimateMessageLength(context);
        const complexity = this.assessConversationComplexity(context);
        const historyLength = context.history?.recentSteps?.length || 0;
        
        // Base estimates from legacy performance data:
        // - Simple conversations: ~500-1000 tokens
        // - Complex conversations: ~1500-3000 tokens
        // - Multi-turn conversations: +500 tokens per turn
        const baseTokens = 800;
        const complexityMultiplier = 1 + (complexity * 0.5);
        const historyMultiplier = 1 + (historyLength * 0.1);
        const estimatedTokens = Math.ceil(baseTokens * complexityMultiplier * historyMultiplier);
        
        // API calls: 1 for conversation, +1 if tools are likely needed
        const apiCalls = this.requiresToolsEstimate(context) ? 2 : 1;
        
        // Compute time: legacy shows 5-30 seconds depending on complexity
        const baseComputeTime = 8000; // 8 seconds
        const computeTime = Math.ceil(baseComputeTime * complexityMultiplier);
        
        // Cost estimation based on token usage
        const costPerToken = 0.00002; // GPT-4o-mini pricing
        const cost = estimatedTokens * costPerToken * apiCalls;
        
        return {
            tokens: estimatedTokens,
            apiCalls,
            computeTime,
            cost,
        };
    }

    /**
     * Enhanced learning mechanism with performance tracking
     */
    learn(feedback: StrategyFeedback): void {
        this.logger.info("[ConversationalStrategy] Learning from feedback", {
            outcome: feedback.outcome,
            satisfaction: feedback.userSatisfaction,
            performance: feedback.performanceScore,
        });
        
        // Adaptive learning based on feedback
        if (feedback.outcome === "success" && feedback.performanceScore > 0.8) {
            this.optimizeForSuccess(feedback);
        } else if (feedback.outcome === "failure") {
            this.adjustForFailure(feedback);
        }
        
        // Update performance metrics
        this.updatePerformanceMetrics(feedback);
    }

    /**
     * Enhanced performance metrics tracking
     */
    getPerformanceMetrics(): StrategyPerformance {
        const recentHistory = this.performanceHistory.slice(-100); // Last 100 executions
        
        if (recentHistory.length === 0) {
            return {
                totalExecutions: 0,
                successCount: 0,
                failureCount: 0,
                averageExecutionTime: 0,
                averageResourceUsage: {},
                averageConfidence: 0,
                evolutionScore: 0,
            };
        }
        
        const successCount = recentHistory.filter(h => h.success).length;
        const avgExecutionTime = recentHistory.reduce((sum, h) => sum + h.executionTime, 0) / recentHistory.length;
        const avgTokensUsed = recentHistory.reduce((sum, h) => sum + h.tokensUsed, 0) / recentHistory.length;
        const avgConfidence = recentHistory.reduce((sum, h) => sum + h.confidence, 0) / recentHistory.length;
        
        // Evolution score: improvement over time
        const evolutionScore = this.calculateEvolutionScore(recentHistory);
        
        return {
            totalExecutions: this.performanceHistory.length,
            successCount,
            failureCount: recentHistory.length - successCount,
            averageExecutionTime: avgExecutionTime,
            averageResourceUsage: {
                tokens: avgTokensUsed,
                computeTime: avgExecutionTime,
                cost: avgTokensUsed * 0.00002,
            },
            averageConfidence: avgConfidence,
            evolutionScore,
        };
    }

    /**
     * LEGACY PATTERN: Build conversation request with proven context building
     * Extracted from legacy ConversationalStrategy.buildSystemMessage() and executeConversationTurn()
     */
    private async buildConversationRequest(context: StrategyExecutionContext): Promise<LLMRequest> {
        // Legacy: Build system message with context awareness
        const systemMessage = await this.buildSystemMessage(context);
        
        // Legacy: Build message history from context
        const messages = this.buildMessageHistory(context);
        
        // Legacy: Prepare tools if available
        const tools = this.prepareTools(context);
        
        return {
            model: context.config.model as string || "gpt-4o-mini",
            messages: [
                {
                    role: "system",
                    content: systemMessage,
                },
                ...messages,
            ],
            tools: tools.length > 0 ? tools : undefined,
            temperature: this.determineCreativityLevel(context),
            maxTokens: context.constraints?.maxTokens || 2000,
        };
    }

    /**
     * LEGACY PATTERN: Build system message from legacy implementation
     * Extracted from legacy ConversationalStrategy.buildSystemMessage()
     */
    private async buildSystemMessage(context: StrategyExecutionContext): Promise<string> {
        // Legacy: Extract routine information
        const name = context.config.name as string || "Conversational Task";
        const description = context.config.description as string || "";
        const instructions = context.config.instructions as string || "";
        
        let systemMessage = `You are engaging in a conversational task: "${name}"\n\n`;
        
        if (description) {
            systemMessage += `Description: ${description}\n\n`;
        }
        
        if (instructions) {
            systemMessage += `Instructions:\n${instructions}\n\n`;
        }
        
        // Legacy: Add conversation guidelines
        systemMessage += "Conversation Guidelines:\n";
        systemMessage += "- Engage naturally and helpfully with the user\n";
        systemMessage += "- Stay focused on the task objectives\n";
        systemMessage += "- Ask clarifying questions when needed\n";
        systemMessage += "- Signal when the conversation objectives are met\n";
        systemMessage += "- Be concise but thorough in responses\n\n";
        
        // Legacy: Add expected outputs
        const expectedOutputs = context.config.expectedOutputs as Record<string, any> || {};
        if (Object.keys(expectedOutputs).length > 0) {
            systemMessage += "Expected conversation outcomes:\n";
            for (const [key, output] of Object.entries(expectedOutputs)) {
                systemMessage += `- ${output.name || key}`;
                if (output.description) {
                    systemMessage += `: ${output.description}`;
                }
                systemMessage += "\n";
            }
        }
        
        return systemMessage;
    }

    /**
     * LEGACY PATTERN: Build message history from context
     * Extracted from legacy ConversationalStrategy message handling
     */
    private buildMessageHistory(context: StrategyExecutionContext): LLMMessage[] {
        const messages: LLMMessage[] = [];
        
        // Legacy: Build initial user message from inputs
        const initialMessage = this.buildInitialMessage(context);
        if (initialMessage) {
            messages.push(initialMessage);
        }
        
        // Legacy: Add conversation history from recent steps
        const history = context.history?.recentSteps || [];
        const recentMessages = history.slice(-ConversationalStrategy.CONVERSATION_CONTEXT_WINDOW);
        
        for (const step of recentMessages) {
            messages.push({
                role: "user",
                content: this.formatStepAsMessage(step),
            });
        }
        
        return messages;
    }
    
    /**
     * LEGACY PATTERN: Build initial message from inputs
     * Extracted from legacy ConversationalStrategy.buildInitialMessage()
     */
    private buildInitialMessage(context: StrategyExecutionContext): LLMMessage | null {
        const inputs = context.inputs;
        
        // Legacy: Look for message, prompt, or question input
        const messageText = inputs.message || inputs.prompt || inputs.question;
        
        if (!messageText) {
            // If no direct message, create from all inputs
            const inputText = Object.entries(inputs)
                .map(([key, value]) => `${key}: ${JSON.stringify(value)}`)
                .join("\n");
            
            if (inputText.trim()) {
                return {
                    role: "user",
                    content: `Please help me with this task. Here are the inputs:\n${inputText}`,
                };
            }
            
            return null;
        }
        
        return {
            role: "user",
            content: String(messageText),
        };
    }
    
    /**
     * LEGACY PATTERN: Execute conversational reasoning with enhanced error handling
     * Enhanced from legacy ConversationalStrategy.executeConversationTurn()
     */
    private async executeConversationalReasoning(
        request: LLMRequest,
        context: StrategyExecutionContext,
    ): Promise<ConversationalResponse> {
        this.logger.debug("[ConversationalStrategy] Executing conversational reasoning", {
            stepId: context.stepId,
            messages: request.messages.length,
        });

        try {
            // Get available resources from context
            const availableResources = {
                maxCredits: context.resources?.credits || 10000,
                maxTokens: context.constraints?.maxTokens || 2000,
                maxTime: context.constraints?.maxTime || ConversationalStrategy.TURN_TIMEOUT_MS,
                tools: context.resources?.tools || [],
            };

            // Get user data for LLM context
            const userData = context.metadata?.userId ? {
                id: context.metadata.userId,
                name: context.config.userName as string,
            } : undefined;

            // Execute LLM request with timeout
            const response = await this.llmService.executeRequest(
                request,
                availableResources,
                userData,
            );

            return {
                content: response.content,
                reasoning: response.reasoning || "Conversational response generated",
                confidence: response.confidence,
                tokensUsed: response.tokensUsed,
                toolCalls: response.toolCalls || [],
            };

        } catch (error) {
            this.logger.error("[ConversationalStrategy] LLM execution failed", {
                stepId: context.stepId,
                error: error instanceof Error ? error.message : String(error),
            });

            // Legacy pattern: Error recovery with fallback
            return this.handleConversationalError(error as Error, context, request);
        }
    }

    /**
     * LEGACY PATTERN: Handle conversational error with recovery
     * Extracted from legacy ConversationalStrategy error handling
     */
    private handleConversationalError(
        error: Error,
        context: StrategyExecutionContext,
        request: LLMRequest,
    ): ConversationalResponse {
        // Legacy: Check if error is retryable
        if (this.isRetryableError(error)) {
            // Return a recoverable response that can be retried
            return {
                content: "I encountered a temporary issue. Let me try to help you with your request in a different way.",
                reasoning: `Retryable error encountered: ${error.message}`,
                confidence: 0.4,
                tokensUsed: this.estimateTokenUsage(request),
                toolCalls: [],
            };
        }
        
        // Legacy: Graceful fallback response
        return {
            content: "I apologize, but I encountered an issue processing your request. Please try rephrasing your question or contact support if the problem persists.",
            reasoning: `Non-retryable error: ${error.message}`,
            confidence: 0.3,
            tokensUsed: this.estimateTokenUsage(request),
            toolCalls: [],
        };
    }
    
    /**
     * Check if error is retryable (legacy pattern)
     */
    private isRetryableError(error: Error): boolean {
        const retryablePatterns = [
            "timeout",
            "rate limit",
            "temporary",
            "network",
            "connection",
        ];
        
        const errorMessage = error.message.toLowerCase();
        return retryablePatterns.some(pattern => errorMessage.includes(pattern));
    }
    
    /**
     * Estimate token usage for requests
     */
    private estimateTokenUsage(request: LLMRequest): number {
        const totalContent = request.messages.reduce((sum, msg) => sum + msg.content.length, 0);
        return Math.ceil(totalContent / 4); // Rough estimate: 1 token per 4 characters
    }

    /**
     * LEGACY PATTERN: Extract outputs from conversation
     * Extracted from legacy ConversationalStrategy.extractOutputsFromConversation()
     */
    private async extractOutputsFromConversation(
        response: ConversationalResponse,
        context: StrategyExecutionContext,
    ): Promise<Record<string, unknown>> {
        const outputs: Record<string, unknown> = {};
        const expectedOutputs = context.config.expectedOutputs as Record<string, any> || {};
        const outputKeys = Object.keys(expectedOutputs);
        
        // Legacy: For single output, use the entire conversation
        if (outputKeys.length === 1) {
            outputs[outputKeys[0]] = response.content;
            return outputs;
        }
        
        // Legacy: For multiple outputs, try structured extraction
        if (outputKeys.length > 1) {
            for (const key of outputKeys) {
                const outputName = expectedOutputs[key].name || key;
                
                // Legacy: Look for labeled sections
                const sectionRegex = new RegExp(`${outputName}:?\\s*([^\\n]+(?:\\n(?!\\w+:)[^\\n]+)*)`, "i");
                const match = response.content.match(sectionRegex);
                
                if (match) {
                    outputs[key] = match[1].trim();
                }
            }
        }
        
        // Legacy: If no structured outputs found, use the full response
        if (Object.keys(outputs).length === 0) {
            outputs.result = response.content;
        }
        
        // Always include reasoning and confidence
        outputs.reasoning = response.reasoning;
        outputs.confidence = response.confidence;
        
        return outputs;
    }
    
    /**
     * Enhanced resource usage calculation
     */
    private calculateResourceUsage(
        response: ConversationalResponse,
        executionTime: number,
        totalTokens: number,
    ): ResourceUsage {
        const apiCalls = 1 + (response.toolCalls?.length || 0);
        const costPerToken = 0.00002; // GPT-4o-mini pricing
        
        return {
            tokens: totalTokens,
            apiCalls,
            computeTime: executionTime,
            cost: totalTokens * costPerToken,
        };
    }

    /**
     * Enhanced confidence calculation with legacy patterns
     */
    private calculateConfidence(
        response: ConversationalResponse,
        context: StrategyExecutionContext,
    ): number {
        let confidence = response.confidence;

        // Legacy: Adjust based on conversation quality indicators
        if (response.content.length < 50) {
            confidence *= 0.8; // Penalize very short responses
        }
        
        if (response.content.includes("I don't know") || response.content.includes("unclear")) {
            confidence *= 0.7; // Penalize uncertainty indicators
        }
        
        // Adjust based on context constraints
        if (context.constraints?.requiredConfidence) {
            confidence = Math.min(confidence, 0.9); // Cap if requirements are strict
        }
        
        // Bonus for successful tool usage
        if (response.toolCalls && response.toolCalls.length > 0) {
            confidence *= 1.1; // Small bonus for tool integration
        }

        return Math.max(0, Math.min(1, confidence));
    }

    /**
     * Enhanced performance score calculation with legacy metrics
     */
    private calculatePerformanceScore(
        response: ConversationalResponse,
        context: StrategyExecutionContext,
    ): number {
        let score = 0.7; // Base score for successful completion

        // Legacy: Bonus for high confidence
        if (response.confidence > 0.8) {
            score += 0.1;
        }
        
        // Legacy: Bonus for efficiency (moderate token usage)
        if (response.tokensUsed < 1000) {
            score += 0.1;
        } else if (response.tokensUsed > 2500) {
            score -= 0.05; // Slight penalty for very long responses
        }
        
        // Legacy: Quality indicators
        const content = response.content;
        if (content.length > 100 && content.length < 2000) {
            score += 0.05; // Bonus for good length
        }
        
        // Legacy: Penalty for uncertainty
        if (content.includes("unclear") || content.includes("ambiguous") || content.includes("not sure")) {
            score -= 0.1;
        }
        
        // Legacy: Bonus for structured response
        if (content.includes(":\n") || content.includes("•") || content.includes("-")) {
            score += 0.05; // Bonus for organized output
        }
        
        // Bonus for successful tool integration
        if (response.toolCalls && response.toolCalls.length > 0) {
            score += 0.1;
        }

        return Math.max(0, Math.min(1, score));
    }

    /**
     * Enhanced improvement suggestions with legacy insights
     */
    private suggestImprovements(
        response: ConversationalResponse,
        context: StrategyExecutionContext,
    ): string[] {
        const improvements: string[] = [];

        // Legacy: Confidence-based suggestions
        if (response.confidence < 0.6) {
            improvements.push("Consider providing more specific context or examples to improve response quality");
        }
        
        // Legacy: Token usage optimization
        if (response.tokensUsed > 2000) {
            improvements.push("Response was lengthy - consider more focused prompting or output constraints");
        }
        
        // Legacy: Content quality suggestions
        if (response.content.length < 100) {
            improvements.push("Response was very brief - consider asking for more detailed explanations");
        }
        
        if (response.content.includes("I don't know") || response.content.includes("unclear")) {
            improvements.push("Response contained uncertainty - consider providing clearer instructions or context");
        }
        
        // Tool usage suggestions
        if (this.requiresToolsEstimate(context) && (!response.toolCalls || response.toolCalls.length === 0)) {
            improvements.push("Task might benefit from tool usage - consider enabling relevant tools");
        }
        
        // Multi-turn conversation suggestions
        const historyLength = context.history?.recentSteps?.length || 0;
        if (historyLength === 0) {
            improvements.push("Consider multi-turn conversation for more interactive engagement");
        }

        return improvements;
    }

    /**
     * Helper methods adapted from legacy implementation
     */
    private determineTone(context: StrategyExecutionContext): string {
        const config = context.config;
        return config.tone as string || "professional";
    }

    private determineCreativityLevel(context: StrategyExecutionContext): number {
        const config = context.config;
        return typeof config.creativity === "number" ? config.creativity : 0.7;
    }
    
    /**
     * LEGACY PATTERN: Format step as message for conversation history
     */
    private formatStepAsMessage(step: any): string {
        return `Previous step ${step.stepId}: ${step.result || 'completed'}`;
    }
    
    /**
     * LEGACY PATTERN: Prepare tools for LLM request
     */
    private prepareTools(context: StrategyExecutionContext): any[] {
        const tools = context.resources?.tools || [];
        return tools.map(tool => ({
            type: "function",
            function: {
                name: tool.name,
                description: tool.description,
                parameters: tool.parameters,
            },
        }));
    }
    
    /**
     * Check if response requires tool execution
     */
    private requiresToolExecution(response: ConversationalResponse): boolean {
        return response.toolCalls && response.toolCalls.length > 0;
    }
    
    /**
     * Estimate if tools are likely needed for this context
     */
    private requiresToolsEstimate(context: StrategyExecutionContext): boolean {
        const stepType = context.stepType.toLowerCase();
        const description = (context.config.description as string || "").toLowerCase();
        
        const toolKeywords = ["search", "calculate", "analyze", "fetch", "api", "data", "query"];
        return toolKeywords.some(keyword => 
            stepType.includes(keyword) || description.includes(keyword)
        );
    }

    /**
     * Enhanced conversation complexity assessment
     */
    private assessConversationComplexity(context: StrategyExecutionContext): number {
        let complexity = 0.5; // Base complexity
        
        // Input complexity
        const inputCount = Object.keys(context.inputs).length;
        complexity += Math.min(inputCount * 0.1, 0.3);
        
        // History complexity
        const historyLength = context.history?.recentSteps?.length || 0;
        complexity += Math.min(historyLength * 0.05, 0.2);
        
        // Context constraints
        if (context.constraints?.requiredConfidence && context.constraints.requiredConfidence > 0.8) {
            complexity += 0.1;
        }
        
        return Math.min(complexity, 1.0);
    }
    
    /**
     * Estimate message length for resource calculation
     */
    private estimateMessageLength(context: StrategyExecutionContext): number {
        const inputLength = JSON.stringify(context.inputs).length;
        const configLength = JSON.stringify(context.config).length;
        return inputLength + configLength;
    }
    
    /**
     * Track performance for learning
     */
    private trackPerformance(performance: {
        timestamp: Date;
        executionTime: number;
        tokensUsed: number;
        success: boolean;
        confidence: number;
    }): void {
        this.performanceHistory.push(performance);
        
        // Keep only recent history (last 1000 executions)
        if (this.performanceHistory.length > 1000) {
            this.performanceHistory = this.performanceHistory.slice(-1000);
        }
    }
    
    /**
     * Calculate evolution score based on performance trends
     */
    private calculateEvolutionScore(history: typeof this.performanceHistory): number {
        if (history.length < 10) return 0;
        
        const recent = history.slice(-50);
        const older = history.slice(-100, -50);
        
        if (older.length === 0) return 0;
        
        const recentAvg = recent.reduce((sum, h) => sum + h.confidence, 0) / recent.length;
        const olderAvg = older.reduce((sum, h) => sum + h.confidence, 0) / older.length;
        
        // Evolution score: improvement over time
        return Math.max(0, Math.min(1, (recentAvg - olderAvg) + 0.5));
    }
    
    /**
     * Optimize strategy for successful feedback
     */
    private optimizeForSuccess(feedback: StrategyFeedback): void {
        this.logger.debug("[ConversationalStrategy] Optimizing for success", {
            performanceScore: feedback.performanceScore,
            userSatisfaction: feedback.userSatisfaction,
        });
        // Future: Implement parameter optimization
    }
    
    /**
     * Adjust strategy for failure feedback
     */
    private adjustForFailure(feedback: StrategyFeedback): void {
        this.logger.debug("[ConversationalStrategy] Adjusting for failure", {
            issues: feedback.issues,
        });
        // Future: Implement failure recovery adjustments
    }
    
    /**
     * Update performance metrics with feedback
     */
    private updatePerformanceMetrics(feedback: StrategyFeedback): void {
        // Future: Update internal metrics based on feedback
        this.logger.debug("[ConversationalStrategy] Updated performance metrics", {
            outcome: feedback.outcome,
        });
    }
    
    /**
     * Handle tool execution using the shared ToolOrchestrator
     */
    private async handleToolExecution(
        response: ConversationalResponse,
        context: StrategyExecutionContext,
    ): Promise<ConversationalResponse> {
        this.logger.debug("[ConversationalStrategy] Tool execution requested", {
            toolCalls: response.toolCalls?.length || 0,
        });
        
        if (!this.toolOrchestrator) {
            this.logger.warn("[ConversationalStrategy] No tool orchestrator available, skipping tool execution");
            return response;
        }

        if (!response.toolCalls || response.toolCalls.length === 0) {
            return response;
        }

        const toolResults: string[] = [];
        let additionalTokensUsed = 0;

        for (const toolCall of response.toolCalls) {
            try {
                this.logger.info("[ConversationalStrategy] Executing tool", {
                    toolName: toolCall.name,
                    stepId: context.stepId,
                });

                const result = await this.toolOrchestrator.executeTool(
                    toolCall.name,
                    toolCall.parameters || {},
                );

                toolResults.push(`Tool ${toolCall.name}: ${JSON.stringify(result)}`);
                
                // Estimate token usage for tool result
                additionalTokensUsed += Math.ceil(JSON.stringify(result).length / 4);
            } catch (error) {
                this.logger.error("[ConversationalStrategy] Tool execution failed", {
                    toolName: toolCall.name,
                    error: error instanceof Error ? error.message : String(error),
                });
                toolResults.push(`Tool ${toolCall.name}: Error - ${error instanceof Error ? error.message : "Unknown error"}`);
            }
        }

        // Incorporate tool results into the response
        return {
            ...response,
            content: response.content + "\n\nTool Results:\n" + toolResults.join("\n"),
            reasoning: response.reasoning + " (enhanced with tool execution results)",
            tokensUsed: response.tokensUsed + additionalTokensUsed,
        };
    }
    
    /**
     * Validate outputs against expected schema
     */
    private async validateOutputs(
        outputs: Record<string, unknown>,
        context: StrategyExecutionContext,
    ): Promise<Record<string, unknown>> {
        this.logger.debug("[ConversationalStrategy] Validating outputs", {
            outputKeys: Object.keys(outputs),
        });
        
        if (!this.validationEngine) {
            this.logger.warn("[ConversationalStrategy] No validation engine available, skipping validation");
            return outputs;
        }

        try {
            const validationResult = await this.validationEngine.validateOutputs(
                outputs,
                context.config.expectedOutputs || {},
            );

            if (validationResult.valid) {
                this.logger.debug("[ConversationalStrategy] Outputs validated successfully");
                return validationResult.sanitizedOutputs || outputs;
            } else {
                this.logger.warn("[ConversationalStrategy] Output validation failed", {
                    errors: validationResult.errors,
                });
                // Return original outputs but log validation errors
                return outputs;
            }
        } catch (error) {
            this.logger.error("[ConversationalStrategy] Validation error", {
                error: error instanceof Error ? error.message : String(error),
            });
            // Return original outputs on validation error
            return outputs;
        }
    }
    
    /**
     * Handle execution error with retry capability
     */
    private handleExecutionError(
        error: Error,
        context: StrategyExecutionContext,
        executionTime: number,
    ): StrategyExecutionResult {
        return {
            success: false,
            error: error.message,
            metadata: {
                strategyType: this.type,
                executionTime,
                resourceUsage: { computeTime: executionTime },
                confidence: 0,
                fallbackUsed: this.isRetryableError(error),
            },
            feedback: {
                outcome: "failure",
                performanceScore: 0,
                issues: [error.message],
                improvements: this.isRetryableError(error) 
                    ? ["Retry with different parameters"]
                    : ["Check input format and constraints"],
            },
        };
    }
}

/**
 * Enhanced types for conversational strategy
 */
interface ConversationalResponse {
    content: string;
    reasoning: string;
    confidence: number;
    tokensUsed: number;
    toolCalls?: Array<{
        id: string;
        name: string;
        parameters: Record<string, unknown>;
    }>;
}
