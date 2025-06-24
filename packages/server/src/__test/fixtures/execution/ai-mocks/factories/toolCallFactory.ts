/**
 * Tool Call Factory
 * 
 * Factory for creating AI responses that include tool calls.
 */

import type { LLMToolCall } from "@vrooli/shared";
import type { ToolCallResult } from "../../../../../services/execution/integration/llmIntegrationService.js";
import type { AIMockConfig, AIMockToolCall, MockFactoryResult } from "../types.js";
import { createAIMockResponse } from "./responseFactory.js";

/**
 * Create a response with tool calls
 */
export function createToolCallResponse(config: {
    content?: string;
    toolCalls: AIMockToolCall[];
    model?: string;
    includeResults?: boolean;
}): MockFactoryResult {
    const mockConfig: AIMockConfig = {
        content: config.content || "I'll help you with that using the following tools:",
        model: config.model,
        toolCalls: config.toolCalls
    };
    
    const result = createAIMockResponse(mockConfig);
    
    // Add tool results if requested
    if (config.includeResults) {
        const toolResults = config.toolCalls.map(tc => 
            createToolResult(tc)
        );
        
        result.response.metadata = {
            ...result.response.metadata,
            toolResults
        };
    }
    
    return result;
}

/**
 * Create a multi-step tool calling scenario
 */
export function createMultiStepToolResponse(steps: Array<{
    explanation: string;
    tools: AIMockToolCall[];
}>): MockFactoryResult[] {
    return steps.map((step, index) => {
        const isLast = index === steps.length - 1;
        const mockConfig: AIMockConfig = {
            content: step.explanation,
            toolCalls: step.tools,
            metadata: {
                step: index + 1,
                totalSteps: steps.length,
                isComplete: isLast
            }
        };
        
        return createAIMockResponse(mockConfig);
    });
}

/**
 * Create a tool discovery response
 */
export function createToolDiscoveryResponse(
    availableTools: string[],
    query: string
): MockFactoryResult {
    const selectedTools = selectRelevantTools(availableTools, query);
    const content = `Based on your query, I'll use the following tools: ${selectedTools.join(", ")}`;
    
    return createAIMockResponse({
        content,
        metadata: {
            availableTools,
            selectedTools,
            query
        }
    });
}

/**
 * Create a tool validation response
 */
export function createToolValidationResponse(
    toolName: string,
    args: Record<string, unknown>,
    validationResult: { valid: boolean; errors?: string[] }
): MockFactoryResult {
    if (validationResult.valid) {
        return createAIMockResponse({
            content: `The ${toolName} tool call is valid and ready to execute.`,
            toolCalls: [{
                name: toolName,
                arguments: args
            }],
            confidence: 0.95
        });
    } else {
        return createAIMockResponse({
            content: `The ${toolName} tool call has validation errors: ${validationResult.errors?.join(", ")}`,
            confidence: 0.7,
            metadata: {
                validationErrors: validationResult.errors
            }
        });
    }
}

/**
 * Create parallel tool calls
 */
export function createParallelToolCalls(
    tools: Array<{ name: string; arguments: any; description?: string }>
): MockFactoryResult {
    const descriptions = tools
        .filter(t => t.description)
        .map(t => t.description)
        .join(" ");
    
    const content = descriptions || 
        `Executing ${tools.length} tools in parallel to gather information.`;
    
    return createAIMockResponse({
        content,
        toolCalls: tools.map(t => ({
            name: t.name,
            arguments: t.arguments
        })),
        metadata: {
            executionMode: "parallel",
            toolCount: tools.length
        }
    });
}

/**
 * Create sequential tool calls with dependencies
 */
export function createSequentialToolCalls(
    sequence: Array<{
        tool: AIMockToolCall;
        dependsOn?: string;
        processResult?: (previousResults: any[]) => any;
    }>
): MockFactoryResult[] {
    const results: MockFactoryResult[] = [];
    const accumulatedResults: any[] = [];
    
    for (let i = 0; i < sequence.length; i++) {
        const step = sequence[i];
        const processedArgs = step.processResult 
            ? step.processResult(accumulatedResults)
            : step.tool.arguments;
        
        const content = step.dependsOn
            ? `Using results from ${step.dependsOn} to call ${step.tool.name}`
            : `Calling ${step.tool.name}`;
        
        const mockResult = createAIMockResponse({
            content,
            toolCalls: [{
                ...step.tool,
                arguments: processedArgs
            }],
            metadata: {
                sequenceStep: i + 1,
                totalSteps: sequence.length,
                dependsOn: step.dependsOn
            }
        });
        
        results.push(mockResult);
        accumulatedResults.push(step.tool.result || {});
    }
    
    return results;
}

/**
 * Create a tool retry scenario
 */
export function createToolRetryResponse(
    toolName: string,
    originalArgs: any,
    error: string,
    correctedArgs: any
): MockFactoryResult[] {
    // First attempt - fails
    const firstAttempt = createAIMockResponse({
        content: `Attempting to call ${toolName}...`,
        toolCalls: [{
            name: toolName,
            arguments: originalArgs,
            error
        }],
        metadata: {
            attempt: 1,
            success: false
        }
    });
    
    // Retry with correction
    const retry = createAIMockResponse({
        content: `The previous call failed with: ${error}. Retrying with corrected parameters...`,
        toolCalls: [{
            name: toolName,
            arguments: correctedArgs
        }],
        metadata: {
            attempt: 2,
            success: true,
            correction: "Applied parameter validation and type conversion"
        }
    });
    
    return [firstAttempt, retry];
}

/**
 * Helper functions
 */
function createToolResult(toolCall: AIMockToolCall): ToolCallResult {
    return {
        toolName: toolCall.name,
        input: toolCall.arguments,
        output: toolCall.result || {},
        success: !toolCall.error,
        error: toolCall.error
    };
}

function selectRelevantTools(availableTools: string[], query: string): string[] {
    // Simple relevance matching for demo purposes
    const queryLower = query.toLowerCase();
    const relevanceMap: Record<string, string[]> = {
        search: ["search", "find", "look", "query"],
        calculate: ["calculate", "compute", "math", "sum", "average"],
        create: ["create", "make", "generate", "build"],
        update: ["update", "modify", "change", "edit"],
        delete: ["delete", "remove", "clear", "erase"]
    };
    
    const selected: string[] = [];
    
    for (const tool of availableTools) {
        const keywords = relevanceMap[tool.toLowerCase()] || [tool.toLowerCase()];
        if (keywords.some(keyword => queryLower.includes(keyword))) {
            selected.push(tool);
        }
    }
    
    // If no tools match, select first available
    if (selected.length === 0 && availableTools.length > 0) {
        selected.push(availableTools[0]);
    }
    
    return selected;
}

/**
 * Create a complex tool orchestration scenario
 */
export function createToolOrchestrationScenario(config: {
    goal: string;
    availableTools: string[];
    constraints?: string[];
}): {
    planning: MockFactoryResult;
    execution: MockFactoryResult[];
    summary: MockFactoryResult;
} {
    // Planning phase
    const planning = createAIMockResponse({
        content: `To achieve "${config.goal}", I'll need to coordinate multiple tools.`,
        reasoning: `Analyzing available tools and constraints to create an execution plan...`,
        metadata: {
            phase: "planning",
            availableTools: config.availableTools,
            constraints: config.constraints
        }
    });
    
    // Execution phase (simplified for demo)
    const execution = [
        createToolCallResponse({
            content: "Gathering initial data...",
            toolCalls: [{
                name: config.availableTools[0],
                arguments: { query: config.goal }
            }]
        }),
        createToolCallResponse({
            content: "Processing results...",
            toolCalls: [{
                name: config.availableTools[1] || "process",
                arguments: { data: "mock-data" }
            }]
        })
    ];
    
    // Summary phase
    const summary = createAIMockResponse({
        content: `Successfully completed the task. Used ${execution.length} tools to achieve the goal.`,
        metadata: {
            phase: "summary",
            toolsUsed: execution.length,
            success: true
        }
    });
    
    return { planning, execution, summary };
}