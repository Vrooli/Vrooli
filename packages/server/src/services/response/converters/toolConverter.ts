import type { Tool } from "@vrooli/shared";
import type { ChatCompletionTool } from "openai/resources/chat/completions.js";
import { logger } from "../../../events/logger.js";

/**
 * ToolConverter - Handles conversion between internal and provider-specific tool formats
 * 
 * This module provides type-safe conversions between our internal Tool representation
 * and various LLM provider formats. It maintains separation between our provider-agnostic
 * internal format and provider-specific implementations.
 * 
 * Design Principles:
 * - Explicit, type-safe conversions
 * - Validation of required fields
 * - Clear error handling
 * - Extensible for new providers
 */
export class ToolConverter {
    /**
     * Convert internal Tool format to OpenAI's ChatCompletionTool format
     * 
     * Maps:
     * - name → function.name
     * - description → function.description
     * - inputSchema → function.parameters
     * - Adds required type: 'function'
     * - Sets strict: true for better validation
     * 
     * @param tools Array of internal Tool objects
     * @returns Array of OpenAI ChatCompletionTool objects
     * @throws Error if required fields are missing or invalid
     */
    static toOpenAIFormat(tools: Tool[]): ChatCompletionTool[] {
        return tools.map((tool, index) => {
            // Validate required fields
            if (!tool.name) {
                throw new Error(`Tool at index ${index} is missing required field: name`);
            }
            
            if (!tool.inputSchema || typeof tool.inputSchema !== "object") {
                logger.warn(`Tool "${tool.name}" has invalid or missing inputSchema, using empty object`);
            }

            // Convert to OpenAI ChatCompletionTool format
            const openAITool: ChatCompletionTool = {
                type: "function" as const,
                function: {
                    name: tool.name,
                    description: tool.description || undefined,
                    parameters: tool.inputSchema || {},
                    strict: true, // Enable strict parameter validation
                },
            };

            return openAITool;
        });
    }

    /**
     * Convert internal Tool format to Anthropic's tool format
     * 
     * NOTE: This is a placeholder for future implementation.
     * Anthropic's actual tool format would need to be determined
     * from their API documentation.
     * 
     * @param tools Array of internal Tool objects
     * @returns Array of Anthropic tool objects
     */
    static toAnthropicFormat(tools: Tool[]): unknown[] {
        // TODO: Implement when adding Anthropic support
        logger.warn("Anthropic tool conversion not yet implemented");
        return tools.map(tool => ({
            name: tool.name,
            description: tool.description,
            input_schema: tool.inputSchema,
            // Add Anthropic-specific fields here
        }));
    }

    /**
     * Convert internal Tool format to Mistral's tool format
     * 
     * NOTE: This is a placeholder for future implementation.
     * Mistral's actual tool format would need to be determined
     * from their API documentation.
     * 
     * @param tools Array of internal Tool objects
     * @returns Array of Mistral tool objects
     */
    static toMistralFormat(tools: Tool[]): unknown[] {
        // TODO: Implement when adding Mistral support
        logger.warn("Mistral tool conversion not yet implemented");
        return tools.map(tool => ({
            name: tool.name,
            description: tool.description,
            parameters: tool.inputSchema,
            // Add Mistral-specific fields here
        }));
    }

    /**
     * Validate that a tool has all required fields for conversion
     * 
     * @param tool Tool to validate
     * @returns true if valid, false otherwise
     */
    static isValidTool(tool: Tool): boolean {
        return !!(
            tool.name && 
            typeof tool.name === "string" &&
            tool.inputSchema && 
            typeof tool.inputSchema === "object"
        );
    }

    /**
     * Filter and convert only valid tools, logging warnings for invalid ones
     * 
     * @param tools Array of internal Tool objects
     * @returns Array of OpenAI ChatCompletionTool objects (only valid tools)
     */
    static toOpenAIFormatSafe(tools: Tool[]): ChatCompletionTool[] {
        const validTools: Tool[] = [];
        const invalidTools: string[] = [];

        tools.forEach(tool => {
            if (this.isValidTool(tool)) {
                validTools.push(tool);
            } else {
                invalidTools.push(tool.name || "unnamed");
            }
        });

        if (invalidTools.length > 0) {
            logger.warn(`Skipping invalid tools: ${invalidTools.join(", ")}`);
        }

        return this.toOpenAIFormat(validTools);
    }
}
