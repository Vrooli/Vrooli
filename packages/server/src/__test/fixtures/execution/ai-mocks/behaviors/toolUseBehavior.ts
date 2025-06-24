/**
 * Tool Use Behavior
 * 
 * Implements realistic tool calling behaviors for AI mocks.
 */

import type { LLMRequest, LLMTool } from "@vrooli/shared";
import type { AIMockConfig, AIMockToolCall, StatefulMockConfig, DynamicMockConfig } from "../types.js";
import { aiToolCallFixtures } from "../fixtures/toolCallResponses.js";

/**
 * Create a mock that intelligently selects tools
 */
export function createIntelligentToolSelector(): DynamicMockConfig {
    return {
        matcher: (request: LLMRequest): AIMockConfig | null => {
            if (!request.tools || request.tools.length === 0) {
                return null;
            }
            
            const userQuery = extractUserQuery(request);
            const relevantTools = selectRelevantTools(userQuery, request.tools);
            
            if (relevantTools.length === 0) {
                return aiToolCallFixtures.noToolsNeeded();
            }
            
            const toolCalls = planToolExecution(userQuery, relevantTools);
            
            return {
                content: generateToolUseExplanation(toolCalls),
                toolCalls,
                confidence: calculateToolSelectionConfidence(toolCalls, relevantTools),
                metadata: {
                    availableTools: request.tools.length,
                    selectedTools: toolCalls.length,
                    selectionStrategy: "relevance-based"
                }
            };
        }
    };
}

/**
 * Create a mock with tool execution optimization
 */
export function createOptimizedToolExecutor(): StatefulMockConfig<{
    toolPerformance: Map<string, { avgTime: number; successRate: number; calls: number }>;
    parallelizationEnabled: boolean;
}> {
    return {
        initialState: {
            toolPerformance: new Map(),
            parallelizationEnabled: true
        },
        
        behavior: (request, state) => {
            if (!request.tools || request.tools.length === 0) {
                return aiToolCallFixtures.noToolsNeeded();
            }
            
            const query = extractUserQuery(request);
            const executionPlan = createOptimalExecutionPlan(
                query,
                request.tools,
                state.toolPerformance,
                state.parallelizationEnabled
            );
            
            // Update performance metrics
            executionPlan.forEach(step => {
                const perf = state.toolPerformance.get(step.tool) || {
                    avgTime: 100,
                    successRate: 1,
                    calls: 0
                };
                perf.calls++;
                state.toolPerformance.set(step.tool, perf);
            });
            
            const toolCalls = executionPlan.map(step => ({
                name: step.tool,
                arguments: step.arguments,
                result: step.expectedResult
            }));
            
            return {
                content: `Executing optimized tool plan with ${executionPlan.filter(s => s.parallel).length} parallel operations.`,
                toolCalls,
                metadata: {
                    executionStrategy: "optimized",
                    parallelSteps: executionPlan.filter(s => s.parallel).length,
                    sequentialSteps: executionPlan.filter(s => !s.parallel).length,
                    estimatedTime: calculateEstimatedTime(executionPlan, state.toolPerformance)
                }
            };
        }
    };
}

/**
 * Create a mock with adaptive tool retry logic
 */
export function createAdaptiveToolRetry(): StatefulMockConfig<{
    toolFailures: Map<string, { count: number; lastError: string }>;
    retryStrategies: Map<string, string>;
}> {
    return {
        initialState: {
            toolFailures: new Map(),
            retryStrategies: new Map()
        },
        
        behavior: (request, state) => {
            const query = extractUserQuery(request);
            const requiredTool = identifyRequiredTool(query, request.tools || []);
            
            if (!requiredTool) {
                return aiToolCallFixtures.noToolsNeeded();
            }
            
            const failures = state.toolFailures.get(requiredTool.name);
            
            if (failures && failures.count > 0) {
                // Apply retry strategy
                const strategy = selectRetryStrategy(requiredTool.name, failures);
                state.retryStrategies.set(requiredTool.name, strategy);
                
                const adaptedCall = applyRetryStrategy(requiredTool, strategy, failures);
                
                return {
                    content: `Retrying ${requiredTool.name} with ${strategy} strategy after ${failures.count} failures.`,
                    toolCalls: [adaptedCall],
                    reasoning: `Previous error: ${failures.lastError}. Applying ${strategy} to overcome the issue.`,
                    confidence: Math.max(0.5, 0.9 - (failures.count * 0.1)),
                    metadata: {
                        retryAttempt: failures.count + 1,
                        strategy,
                        previousError: failures.lastError
                    }
                };
            }
            
            // First attempt
            return {
                content: `Using ${requiredTool.name} to ${query}`,
                toolCalls: [{
                    name: requiredTool.name,
                    arguments: generateDefaultArguments(requiredTool)
                }],
                confidence: 0.9
            };
        }
    };
}

/**
 * Create a mock with tool composition capabilities
 */
export function createToolComposer(): DynamicMockConfig {
    return {
        matcher: (request) => {
            if (!request.tools || request.tools.length < 2) {
                return null;
            }
            
            const query = extractUserQuery(request);
            const compositionPlan = planToolComposition(query, request.tools);
            
            if (!compositionPlan) {
                return null;
            }
            
            return {
                content: `I'll accomplish this by combining multiple tools: ${compositionPlan.description}`,
                toolCalls: compositionPlan.steps.map(step => ({
                    name: step.tool,
                    arguments: step.arguments,
                    result: step.transforms ? step.transforms(step.previousResult) : undefined
                })),
                reasoning: compositionPlan.reasoning,
                confidence: compositionPlan.confidence,
                metadata: {
                    compositionType: compositionPlan.type,
                    toolCount: compositionPlan.steps.length
                }
            };
        }
    };
}

/**
 * Create a mock with context-aware tool usage
 */
export function createContextAwareToolUser(): StatefulMockConfig<{
    conversationContext: {
        topics: Set<string>;
        entities: Map<string, any>;
        preferences: Map<string, string>;
    };
}> {
    return {
        initialState: {
            conversationContext: {
                topics: new Set(),
                entities: new Map(),
                preferences: new Map()
            }
        },
        
        behavior: (request, state) => {
            // Update context from conversation
            updateConversationContext(request, state.conversationContext);
            
            if (!request.tools || request.tools.length === 0) {
                return aiToolCallFixtures.noToolsNeeded();
            }
            
            const query = extractUserQuery(request);
            const contextualizedTools = selectToolsWithContext(
                query,
                request.tools,
                state.conversationContext
            );
            
            return {
                content: generateContextAwareResponse(contextualizedTools, state.conversationContext),
                toolCalls: contextualizedTools.map(ct => ({
                    name: ct.tool.name,
                    arguments: enrichArgumentsWithContext(
                        ct.baseArguments,
                        state.conversationContext,
                        ct.tool
                    )
                })),
                confidence: 0.88,
                metadata: {
                    contextFactors: {
                        topics: Array.from(state.conversationContext.topics),
                        entityCount: state.conversationContext.entities.size,
                        preferencesApplied: contextualizedTools.filter(ct => ct.preferenceApplied).length
                    }
                }
            };
        }
    };
}

/**
 * Create a mock with tool discovery capabilities
 */
export function createToolDiscoveryMock(): DynamicMockConfig {
    return {
        matcher: (request) => {
            const query = extractUserQuery(request);
            
            // Check if user is asking about available tools
            if (query.match(/what tools|available tools|can you|capabilities/i)) {
                const tools = request.tools || [];
                
                return {
                    content: formatToolDiscoveryResponse(tools),
                    reasoning: "User is inquiring about available capabilities",
                    confidence: 0.95,
                    metadata: {
                        discoveryType: "explicit-request",
                        toolCount: tools.length
                    }
                };
            }
            
            // Implicit discovery - suggest tools for the task
            if (request.tools && request.tools.length > 0) {
                const suggestions = suggestToolsForTask(query, request.tools);
                
                if (suggestions.length > 0) {
                    return {
                        content: `For your task, I can use: ${suggestions.map(s => s.tool).join(", ")}. ${suggestions[0].reason}`,
                        toolCalls: suggestions.slice(0, 1).map(s => ({
                            name: s.tool,
                            arguments: s.suggestedArgs
                        })),
                        confidence: 0.85,
                        metadata: {
                            discoveryType: "task-based-suggestion",
                            suggestions: suggestions.map(s => ({ tool: s.tool, reason: s.reason }))
                        }
                    };
                }
            }
            
            return null;
        }
    };
}

/**
 * Helper functions
 */
function extractUserQuery(request: LLMRequest): string {
    const userMessages = request.messages.filter(m => m.role === "user");
    return userMessages[userMessages.length - 1]?.content || "";
}

function selectRelevantTools(query: string, tools: LLMTool[]): LLMTool[] {
    const queryLower = query.toLowerCase();
    
    return tools.filter(tool => {
        const nameRelevance = tool.function.name.toLowerCase().split("_")
            .some(part => queryLower.includes(part));
        const descRelevance = tool.function.description.toLowerCase().split(" ")
            .some(word => queryLower.includes(word));
        
        return nameRelevance || descRelevance;
    });
}

function planToolExecution(query: string, tools: LLMTool[]): AIMockToolCall[] {
    return tools.map(tool => ({
        name: tool.function.name,
        arguments: generateDefaultArguments(tool)
    }));
}

function generateToolUseExplanation(toolCalls: AIMockToolCall[]): string {
    if (toolCalls.length === 0) {
        return "I can answer this directly without using any tools.";
    } else if (toolCalls.length === 1) {
        return `I'll use the ${toolCalls[0].name} tool to help with this.`;
    } else {
        return `I'll use ${toolCalls.length} tools to comprehensively address your request.`;
    }
}

function calculateToolSelectionConfidence(selected: AIMockToolCall[], available: LLMTool[]): number {
    const selectionRatio = selected.length / available.length;
    if (selectionRatio > 0.8) return 0.7; // Too many tools selected
    if (selectionRatio < 0.2) return 0.95; // Very selective
    return 0.85;
}

function createOptimalExecutionPlan(
    query: string,
    tools: LLMTool[],
    performance: Map<string, any>,
    allowParallel: boolean
): Array<{
    tool: string;
    arguments: any;
    parallel: boolean;
    expectedResult: any;
}> {
    // Simplified execution planning
    const plan = tools.slice(0, 3).map((tool, index) => ({
        tool: tool.function.name,
        arguments: generateDefaultArguments(tool),
        parallel: allowParallel && index > 0,
        expectedResult: { success: true }
    }));
    
    return plan;
}

function calculateEstimatedTime(plan: any[], performance: Map<string, any>): number {
    let totalTime = 0;
    let parallelTime = 0;
    
    plan.forEach(step => {
        const perf = performance.get(step.tool);
        const stepTime = perf?.avgTime || 100;
        
        if (step.parallel) {
            parallelTime = Math.max(parallelTime, stepTime);
        } else {
            totalTime += stepTime;
        }
    });
    
    return totalTime + parallelTime;
}

function identifyRequiredTool(query: string, tools: LLMTool[]): LLMTool | null {
    const relevantTools = selectRelevantTools(query, tools);
    return relevantTools[0] || null;
}

function selectRetryStrategy(toolName: string, failures: { count: number; lastError: string }): string {
    if (failures.lastError.includes("timeout")) return "increase-timeout";
    if (failures.lastError.includes("invalid")) return "validate-inputs";
    if (failures.lastError.includes("rate")) return "exponential-backoff";
    if (failures.count > 2) return "fallback-method";
    return "simple-retry";
}

function applyRetryStrategy(
    tool: LLMTool,
    strategy: string,
    failures: any
): AIMockToolCall {
    const baseArgs = generateDefaultArguments(tool);
    
    switch (strategy) {
        case "increase-timeout":
            return {
                name: tool.function.name,
                arguments: { ...baseArgs, timeout: 30000 }
            };
        case "validate-inputs":
            return {
                name: tool.function.name,
                arguments: sanitizeArguments(baseArgs)
            };
        case "exponential-backoff":
            return {
                name: tool.function.name,
                arguments: baseArgs,
                delay: Math.pow(2, failures.count) * 1000
            };
        default:
            return {
                name: tool.function.name,
                arguments: baseArgs
            };
    }
}

function generateDefaultArguments(tool: LLMTool): any {
    const params = tool.function.parameters as any;
    const args: any = {};
    
    if (params.properties) {
        Object.entries(params.properties).forEach(([key, schema]: [string, any]) => {
            if (params.required?.includes(key)) {
                args[key] = generateValueForSchema(schema);
            }
        });
    }
    
    return args;
}

function generateValueForSchema(schema: any): any {
    switch (schema.type) {
        case "string":
            return schema.default || "example";
        case "number":
            return schema.default || 0;
        case "boolean":
            return schema.default || false;
        case "array":
            return [];
        case "object":
            return {};
        default:
            return null;
    }
}

function sanitizeArguments(args: any): any {
    const sanitized: any = {};
    
    Object.entries(args).forEach(([key, value]) => {
        if (value !== null && value !== undefined) {
            sanitized[key] = typeof value === "string" ? value.trim() : value;
        }
    });
    
    return sanitized;
}

function planToolComposition(
    query: string,
    tools: LLMTool[]
): {
    type: string;
    description: string;
    reasoning: string;
    confidence: number;
    steps: Array<{
        tool: string;
        arguments: any;
        previousResult?: any;
        transforms?: (input: any) => any;
    }>;
} | null {
    // Look for composition patterns
    if (query.includes("and then") || query.includes("followed by")) {
        return {
            type: "sequential",
            description: "Sequential tool composition",
            reasoning: "The task requires multiple steps in sequence",
            confidence: 0.85,
            steps: tools.slice(0, 2).map((tool, i) => ({
                tool: tool.function.name,
                arguments: generateDefaultArguments(tool),
                previousResult: i > 0 ? {} : undefined
            }))
        };
    }
    
    if (query.includes("combine") || query.includes("together")) {
        return {
            type: "parallel-merge",
            description: "Parallel execution with merged results",
            reasoning: "Multiple data sources need to be combined",
            confidence: 0.82,
            steps: tools.slice(0, 3).map(tool => ({
                tool: tool.function.name,
                arguments: generateDefaultArguments(tool)
            }))
        };
    }
    
    return null;
}

function updateConversationContext(request: LLMRequest, context: any): void {
    request.messages.forEach(msg => {
        // Extract topics
        const topics = msg.content.match(/\b(api|database|user|system|performance)\b/gi);
        if (topics) {
            topics.forEach(topic => context.topics.add(topic.toLowerCase()));
        }
        
        // Extract entities (simplified)
        const entities = msg.content.match(/\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\b/g);
        if (entities) {
            entities.forEach(entity => {
                context.entities.set(entity, { type: "named_entity", count: 1 });
            });
        }
    });
}

function selectToolsWithContext(
    query: string,
    tools: LLMTool[],
    context: any
): Array<{
    tool: LLMTool;
    baseArguments: any;
    preferenceApplied: boolean;
}> {
    return tools
        .filter(tool => {
            // Filter based on context topics
            const toolTopics = tool.function.name.toLowerCase().split("_");
            return toolTopics.some(topic => context.topics.has(topic));
        })
        .map(tool => ({
            tool,
            baseArguments: generateDefaultArguments(tool),
            preferenceApplied: false
        }));
}

function enrichArgumentsWithContext(args: any, context: any, tool: LLMTool): any {
    const enriched = { ...args };
    
    // Add context-specific parameters
    if (context.entities.size > 0 && enriched.filter === undefined) {
        enriched.filter = Array.from(context.entities.keys());
    }
    
    return enriched;
}

function generateContextAwareResponse(tools: any[], context: any): string {
    const topics = Array.from(context.topics);
    return `Based on our discussion about ${topics.join(", ")}, I'll use ${tools.length} relevant tools.`;
}

function formatToolDiscoveryResponse(tools: LLMTool[]): string {
    if (tools.length === 0) {
        return "I don't have any tools available for this conversation.";
    }
    
    const toolList = tools.map(t => 
        `- ${t.function.name}: ${t.function.description}`
    ).join("\n");
    
    return `I have access to ${tools.length} tools:\n${toolList}`;
}

function suggestToolsForTask(
    query: string,
    tools: LLMTool[]
): Array<{
    tool: string;
    reason: string;
    suggestedArgs: any;
}> {
    const suggestions: Array<{
        tool: string;
        reason: string;
        suggestedArgs: any;
    }> = [];
    
    tools.forEach(tool => {
        const relevance = calculateToolRelevance(query, tool);
        if (relevance > 0.5) {
            suggestions.push({
                tool: tool.function.name,
                reason: `This tool can ${tool.function.description.toLowerCase()}`,
                suggestedArgs: generateDefaultArguments(tool)
            });
        }
    });
    
    return suggestions.sort((a, b) => b.tool.length - a.tool.length);
}

function calculateToolRelevance(query: string, tool: LLMTool): number {
    const queryWords = query.toLowerCase().split(/\s+/);
    const toolWords = [
        ...tool.function.name.toLowerCase().split("_"),
        ...tool.function.description.toLowerCase().split(/\s+/)
    ];
    
    const commonWords = queryWords.filter(word => 
        toolWords.some(toolWord => toolWord.includes(word) || word.includes(toolWord))
    );
    
    return commonWords.length / queryWords.length;
}