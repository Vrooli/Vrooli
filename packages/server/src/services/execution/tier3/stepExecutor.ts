/**
 * StepExecutor - Simplified Tier 3 Implementation
 * 
 * A pure executor that:
 * - Executes individual routine steps
 * - Selects appropriate execution strategy
 * - Calls tools via MCP protocol
 * - Returns results to Tier 2
 * 
 * NO state management, NO resource tracking, NO approval flows.
 * All orchestration happens in Tier 2 and state/resources are managed by SwarmContextManager.
 */

import {
    type ChatMessage,
    type CodeLanguage,
    type ExecutionResult,
    type StepExecutionInput,
    type TierCommunicationInterface,
    type TierExecutionRequest,
    type ConfigCallDataApi,
    type RoutineVersionConfigObject,
    type SubroutineIOMapping,
    CallDataApiConfig,
    ApiVersionConfig,
    nanoid,
} from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { runUserCode } from "../../../tasks/sandbox/process.js";
import { HTTPClient, type APICallOptions } from "../../http/httpClient.js";
import { APIKeyService, type UserContext } from "../../http/apiKeyService.js";
import { MCPClient } from "./tools/mcpClient.js";

// Default configuration constants
const MOCK_TOKEN_COUNT = 100;
const MOCK_EXECUTION_TIME = 500;

//TODO: Remove this once we have a proper step definition
export interface StepDefinition {
    id: string;
    type: "llm_call" | "tool_call" | "code_execution" | "api_call";
    strategy?: "conversational" | "reasoning" | "deterministic";
    inputs: Record<string, any>;
    config?: {
        model?: string;
        temperature?: number;
        tools?: string[];
        maxTokens?: number;
        timeout?: number;
        retries?: number;
    };
    // NEW: Enhanced config-driven execution support
    routineConfig?: RoutineVersionConfigObject;
    ioMapping?: SubroutineIOMapping;
    userLanguages?: string[];
}

export interface StepResult {
    success: boolean;
    outputs?: Record<string, any>;
    error?: string;
    metadata?: {
        tokensUsed?: number;
        executionTime?: number;
        model?: string;
    };
}

/**
 * Simple Step Executor - The ENTIRE Tier 3 implementation
 * 
 * This replaces 5,000+ lines of complex code with ~200 lines of focused execution logic.
 */
export class StepExecutor implements TierCommunicationInterface {
    private readonly mcpClient: MCPClient;
    private readonly httpClient: HTTPClient;
    private readonly apiKeyService: APIKeyService;

    constructor() {
        this.mcpClient = new MCPClient();
        this.httpClient = new HTTPClient();
        this.apiKeyService = new APIKeyService();
    }

    /**
     * Execute a tier request - adapts TierExecutionRequest to our simple StepDefinition
     */
    async execute<TInput extends StepExecutionInput, TOutput>(
        request: TierExecutionRequest<TInput>,
    ): Promise<ExecutionResult<TOutput>> {
        const startTime = Date.now();
        
        try {
            // Convert TierExecutionRequest to simple StepDefinition
            const step: StepDefinition = {
                id: request.input.stepId || `step-${Date.now()}`,
                type: this.inferStepType(request.input),
                strategy: request.input.strategy,
                inputs: request.input.inputs || {},
                config: request.input.config,
            };

            // Execute the step
            const result = await this.executeStep(step);

            // Convert StepResult to ExecutionResult
            return {
                success: result.success,
                outputs: result.outputs as TOutput,
                error: result.error ? {
                    code: "EXECUTION_ERROR",
                    message: result.error,
                    tier: "tier3",
                    type: "StepExecutionError",
                } : undefined,
                metadata: {
                    executionTime: Date.now() - startTime,
                    ...result.metadata,
                },
            };
        } catch (error) {
            logger.error("[StepExecutor] Execution failed", {
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                success: false,
                error: {
                    code: "EXECUTION_ERROR",
                    message: error instanceof Error ? error.message : String(error),
                    tier: "tier3",
                    type: "StepExecutionError",
                },
                metadata: {
                    executionTime: Date.now() - startTime,
                },
            } as ExecutionResult<TOutput>;
        }
    }

    /**
     * Execute a single step - the core functionality
     */
    private async executeStep(step: StepDefinition): Promise<StepResult> {
        const startTime = Date.now();

        logger.info("[StepExecutor] Executing step", {
            id: step.id,
            type: step.type,
            strategy: step.strategy,
        });

        try {
            switch (step.type) {
                case "llm_call":
                    return await this.executeLLMCall(step);
                
                case "tool_call":
                    return await this.executeToolCall(step);
                
                case "code_execution":
                    return await this.executeCode(step);
                
                case "api_call":
                    return await this.executeAPICall(step);
                
                default:
                    throw new Error(`Unknown step type: ${step.type}`);
            }
        } catch (error) {
            logger.error("[StepExecutor] Step execution failed", {
                stepId: step.id,
                type: step.type,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                success: false,
                error: error instanceof Error ? error.message : String(error),
                metadata: {
                    executionTime: Date.now() - startTime,
                },
            };
        }
    }

    /**
     * Execute LLM call - routes to appropriate service based on strategy
     */
    private async executeLLMCall(step: StepDefinition): Promise<StepResult> {
        try {
            // Build conversation history from inputs
            const messages = this.buildConversationHistory(step);
            
            // Log the LLM call details
            logger.info("[StepExecutor] Executing LLM call", {
                stepId: step.id,
                strategy: step.strategy || "reasoning",
                model: step.config?.model || "gpt-4",
                messageCount: messages.length,
            });

            // For now, return a mock response that follows the expected format
            // TODO: Integrate with ResponseService when dependencies are properly resolved
            // Different mock responses based on strategy
            let mockResponse: string;
            const strategy = step.strategy || "reasoning";
            
            switch (strategy) {
                case "conversational":
                    mockResponse = `Conversational response for step ${step.id}. I understand your request and would engage in natural dialogue.`;
                    break;
                case "reasoning":
                    mockResponse = `Reasoning response for step ${step.id}. Let me analyze this step by step: 1) Understanding the input, 2) Processing the logic, 3) Providing a structured response.`;
                    break;
                case "deterministic":
                    mockResponse = `Deterministic response for step ${step.id}. Processing input directly without additional reasoning.`;
                    break;
                default:
                    mockResponse = `Default response for step ${step.id} using ${strategy} strategy.`;
            }

            return {
                success: true,
                outputs: {
                    message: mockResponse,
                    finishReason: "stop",
                    strategy,
                    model: step.config?.model || "gpt-4",
                },
                metadata: {
                    tokensUsed: MOCK_TOKEN_COUNT,
                    executionTime: MOCK_EXECUTION_TIME,
                    model: step.config?.model || "gpt-4",
                },
            };
        } catch (error) {
            logger.error("[StepExecutor] LLM call failed", {
                stepId: step.id,
                strategy: step.strategy,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                success: false,
                error: error instanceof Error ? error.message : "Unknown LLM execution error",
                metadata: {
                    executionTime: 0,
                    model: step.config?.model || "gpt-4",
                },
            };
        }
    }

    /**
     * Execute tool call via MCP
     */
    private async executeToolCall(step: StepDefinition): Promise<StepResult> {
        const toolName = step.inputs.tool || step.inputs.name;
        const toolArgs = step.inputs.arguments || step.inputs.args || {};

        const result = await this.mcpClient.callTool(toolName, toolArgs);
        
        return {
            success: result.success,
            outputs: result.result,
            error: result.error,
            metadata: {
                executionTime: 0, // Will be set by caller
            },
        };
    }

    /**
     * Execute sandboxed code
     */
    private async executeCode(step: StepDefinition): Promise<StepResult> {
        try {
            const code = step.inputs.code as string;
            const codeLanguage = (step.inputs.codeLanguage || step.inputs.language || "Javascript") as CodeLanguage;
            const input = step.inputs.input || step.inputs.args || {};

            if (!code) {
                return {
                    success: false,
                    error: "No code provided for execution",
                    metadata: {
                        executionTime: 0,
                    },
                };
            }

            const result = await runUserCode({
                code,
                codeLanguage,
                input,
            });

            if (result.__type === "error") {
                return {
                    success: false,
                    error: result.error || "Code execution failed",
                    metadata: {
                        executionTime: 0,
                    },
                };
            }

            return {
                success: true,
                outputs: {
                    result: result.output,
                    type: result.__type,
                },
                metadata: {
                    executionTime: 0, // Will be set by caller
                },
            };
        } catch (error) {
            logger.error("[StepExecutor] Code execution failed", {
                stepId: step.id,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                success: false,
                error: error instanceof Error ? error.message : "Unknown code execution error",
                metadata: {
                    executionTime: 0,
                },
            };
        }
    }

    /**
     * Execute API call to external services using config-driven approach
     */
    private async executeAPICall(step: StepDefinition): Promise<StepResult> {
        try {
            // Require config-driven approach
            if (!step.routineConfig?.callDataApi || !step.ioMapping) {
                throw new Error("API calls require routineConfig.callDataApi and ioMapping");
            }

            logger.info("[StepExecutor] Starting config-driven API call", {
                stepId: step.id,
                endpoint: step.routineConfig.callDataApi.schema?.endpoint,
                method: step.routineConfig.callDataApi.schema?.method,
            });

            // Build API call from config
            const apiCallOptions = await this.buildConfigDrivenAPICall(step);
            
            // Get user context and apply authentication if configured
            const userContext = this.getUserContext(step);
            const credentials = await this.getAPICredentials(apiCallOptions.url, userContext, step);
            if (credentials) {
                apiCallOptions.auth = credentials;
            }

            logger.info("[StepExecutor] Executing API call", {
                stepId: step.id,
                url: apiCallOptions.url,
                method: apiCallOptions.method,
                hasAuth: !!apiCallOptions.auth,
                authType: apiCallOptions.auth?.type,
            });

            // Make the HTTP request
            const response = await this.httpClient.makeRequest(apiCallOptions);

            // Transform response with output mapping
            return this.transformConfigDrivenResponse(response, step);

        } catch (error) {
            logger.error("[StepExecutor] API call failed", {
                stepId: step.id,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                success: false,
                error: error instanceof Error ? error.message : "Unknown API call error",
                metadata: {
                    executionTime: 0,
                },
            };
        }
    }

    /**
     * Build config-driven API call using CallDataApiConfig and template processing
     */
    private async buildConfigDrivenAPICall(step: StepDefinition): Promise<APICallOptions> {
        if (!step.routineConfig?.callDataApi || !step.ioMapping) {
            throw new Error("Config-driven API call requires routineConfig.callDataApi and ioMapping");
        }

        // Create CallDataApiConfig instance
        const apiConfig = new CallDataApiConfig(step.routineConfig.callDataApi);
        const userLanguages = step.userLanguages || ["en"];

        logger.debug("[StepExecutor] Processing config-driven API template", {
            stepId: step.id,
            endpoint: apiConfig.schema.endpoint,
            method: apiConfig.schema.method,
            userLanguages,
        });

        // Build the API call using template processing
        const processedCall = this.processAPITemplate(apiConfig.schema, step.ioMapping, userLanguages);

        // Get timeout and retry settings from API config or routine config
        const timeout = processedCall.timeoutMs || 
                       step.routineConfig.timeout?.request || 
                       step.config?.timeout;
                       
        const retries = step.routineConfig.retry?.maxAttempts || 
                       step.config?.retries;

        return {
            url: processedCall.endpoint,
            method: processedCall.method,
            headers: processedCall.headers || {},
            body: processedCall.body,
            timeout,
            retries,
        };
    }

    /**
     * Process API templates with dynamic values
     */
    private processAPITemplate(
        apiSchema: ConfigCallDataApi, 
        ioMapping: SubroutineIOMapping, 
        userLanguages: string[],
    ): ConfigCallDataApi {
        const templateConfig = {
            inputs: ioMapping.inputs,
            userLanguages,
            seededIds: {} as Record<string, string>,
        };

        return {
            endpoint: this.processTemplate(apiSchema.endpoint, templateConfig),
            method: apiSchema.method,
            headers: apiSchema.headers ? this.processTemplateObject(apiSchema.headers, templateConfig) : undefined,
            body: apiSchema.body ? this.processTemplateValue(apiSchema.body, templateConfig) : undefined,
            timeoutMs: apiSchema.timeoutMs,
            meta: apiSchema.meta,
        };
    }


    /**
     * Get user context from step or create default
     */
    private getUserContext(step: StepDefinition): UserContext {
        // In production, this would come from TierExecutionRequest.context
        return {
            userId: "step-executor-user", // Would come from actual context
            teamId: undefined, // Would come from actual context
        };
    }


    /**
     * Get API credentials for the request based on configuration
     */
    private async getAPICredentials(url: string, userContext: UserContext, step: StepDefinition): Promise<any> {
        try {
            // Check if authentication is configured in the API config
            const authConfig = step.routineConfig?.callDataApi?.schema?.authentication;
            if (!authConfig || authConfig.type === "none") {
                return undefined;
            }

            // Service must be explicitly configured
            if (!authConfig.settings?.service) {
                logger.warn("[StepExecutor] Authentication configured but no service specified", {
                    url,
                    authType: authConfig.type,
                });
                return undefined;
            }

            // Look up API key for the configured service
            const apiKey = await this.apiKeyService.getApiKey(authConfig.settings.service, userContext);
            if (apiKey) {
                // Override auth type and header from configuration
                apiKey.authType = authConfig.type || apiKey.authType;
                apiKey.headerName = authConfig.parameterName;
                
                logger.debug("[StepExecutor] Retrieved API credentials", {
                    service: apiKey.service,
                    authType: apiKey.authType,
                    headerName: apiKey.headerName,
                    isTeamKey: apiKey.isTeamKey,
                });
                
                return this.apiKeyService.createAuthConfig(apiKey);
            }

            logger.debug("[StepExecutor] No API credentials found for service", { 
                service: authConfig.settings.service, 
            });
            return undefined;

        } catch (error) {
            logger.warn("[StepExecutor] Failed to retrieve API credentials", {
                url,
                error: error instanceof Error ? error.message : String(error),
            });
            return undefined;
        }
    }

    /**
     * Template processing methods for config-driven API calls
     */
    private processTemplate(template: string, config: any): string {
        const placeholderRegex = /\{\{([^}]+)\}\}/g;
        return template.replace(placeholderRegex, (_, placeholder) => {
            const value = this.getPlaceholderValue(placeholder.trim(), config);
            return typeof value === "string" ? value : JSON.stringify(value);
        });
    }

    private processTemplateObject(template: Record<string, string>, config: any): Record<string, string> {
        const result: Record<string, string> = {};
        for (const [key, value] of Object.entries(template)) {
            result[key] = this.processTemplate(value, config);
        }
        return result;
    }

    private processTemplateValue(template: any, config: any): any {
        if (typeof template === "string") {
            return this.processTemplate(template, config);
        } else if (Array.isArray(template)) {
            return template.map(item => this.processTemplateValue(item, config));
        } else if (typeof template === "object" && template !== null) {
            const result: Record<string, any> = {};
            for (const [key, value] of Object.entries(template)) {
                result[key] = this.processTemplateValue(value, config);
            }
            return result;
        }
        return template;
    }

    private getPlaceholderValue(placeholder: string, config: any): unknown {
        // Handle routine inputs: input.inputName
        if (placeholder.startsWith("input.")) {
            const inputName = placeholder.slice("input.".length);
            const inputValue = config.inputs[inputName]?.value;
            if (inputValue === undefined) {
                throw new Error(`Input "${inputName}" not found in ioMapping`);
            }
            return inputValue;
        }

        // Handle special functions
        switch (placeholder) {
            case "userLanguage":
                return config.userLanguages[0] || "en";
            case "userLanguages":
                return config.userLanguages;
            case "now()":
                return new Date().toISOString();
            case "random()":
                return Math.random();
        }

        // Handle nanoid with optional seed
        if (placeholder.startsWith("nanoid")) {
            const args = placeholder.slice("nanoid(".length, -")".length).trim();
            if (args) {
                const seededId = config.seededIds[args];
                if (seededId) return seededId;
                const id = nanoid();
                config.seededIds[args] = id;
                return id;
            }
            return nanoid();
        }

        throw new Error(`Unknown placeholder: ${placeholder}`);
    }


    /**
     * Transform response using config-driven output mapping
     */
    private transformConfigDrivenResponse(response: any, step: StepDefinition): StepResult {
        const result: StepResult = {
            success: response.success,
            outputs: {},
            error: response.error,
            metadata: {
                executionTime: response.metadata.executionTime,
                url: response.metadata.url,
                method: response.metadata.method,
                retries: response.metadata.retries,
            },
        };

        // Apply output mapping if configured
        if (step.routineConfig?.callDataApi?.schema?.outputMapping && step.ioMapping) {
            const outputMapping = step.routineConfig.callDataApi.schema.outputMapping;
            
            // Process each output mapping
            for (const [outputName, sourcePath] of Object.entries(outputMapping)) {
                // Get value from response using dot notation
                const value = this.getValueFromPath(response, sourcePath);
                
                // Update ioMapping outputs
                if (step.ioMapping.outputs && step.ioMapping.outputs[outputName]) {
                    step.ioMapping.outputs[outputName].value = value;
                }
                
                // Also include in result outputs
                result.outputs![outputName] = value;
            }
        } else {
            // No mapping configured - return raw response data
            result.outputs = {
                status: response.status,
                statusText: response.statusText,
                headers: response.headers,
                data: response.data,
                url: response.metadata.finalUrl || response.metadata.url,
            };
        }

        return result;
    }

    /**
     * Get value from object using dot notation path
     */
    private getValueFromPath(obj: any, path: string): any {
        const parts = path.split(".");
        let current = obj;
        
        for (const part of parts) {
            if (current === null || current === undefined) {
                return undefined;
            }
            
            // Handle array notation like "data.items[0]"
            const arrayMatch = part.match(/^(.+)\[(\d+)\]$/);
            if (arrayMatch) {
                const [, key, index] = arrayMatch;
                current = current[key]?.[parseInt(index, 10)];
            } else {
                current = current[part];
            }
        }
        
        return current;
    }

    /**
     * Infer step type from input structure
     */
    private inferStepType(input: StepExecutionInput): StepDefinition["type"] {
        // Check for explicit type in input
        if (input.type) return input.type as StepDefinition["type"];
        
        // Check based on presence of specific inputs
        if (input.messages || input.prompt) return "llm_call";
        if (input.tool || input.toolName) return "tool_call";
        if (input.code || input.script) return "code_execution";
        
        // For API calls, require explicit type or config
        if (input.routineConfig?.callDataApi) return "api_call";
        
        // Default to LLM call if unclear
        return "llm_call";
    }

    /**
     * Get current status
     */
    async getStatus() {
        return {
            healthy: true,
            tier: "tier3" as const,
            activeExecutions: 0, // Always 0 - we're stateless!
        };
    }

    /**
     * Helper method to build conversation history from step inputs
     */
    private buildConversationHistory(step: StepDefinition): ChatMessage[] {
        const messages: ChatMessage[] = [];

        // Handle various input formats
        if (step.inputs.messages && Array.isArray(step.inputs.messages)) {
            // Direct messages array
            return step.inputs.messages.map((msg: any) => ({
                id: `${step.id}-${Date.now()}`,
                config: {
                    text: msg.content || msg.text || String(msg),
                    language: "en",
                },
                role: msg.role || "user",
                createdAt: new Date(),
                updatedAt: new Date(),
                userId: msg.userId || "step-executor-user",
            }));
        } 

        if (step.inputs.prompt || step.inputs.message) {
            // Single prompt/message
            const text = step.inputs.prompt || step.inputs.message;
            messages.push({
                id: `${step.id}-prompt`,
                config: {
                    text: String(text),
                    language: "en",
                },
                role: "user",
                createdAt: new Date(),
                updatedAt: new Date(),
                userId: "step-executor-user",
            });
        }

        // If no specific message format, create a generic request
        if (messages.length === 0) {
            const inputs = Object.entries(step.inputs)
                .filter(([key]) => !["systemMessage", "instructions"].includes(key))
                .map(([key, value]) => `${key}: ${JSON.stringify(value)}`)
                .join("\n");

            messages.push({
                id: `${step.id}-generic`,
                config: {
                    text: inputs || "Please process this step.",
                    language: "en",
                },
                role: "user",
                createdAt: new Date(),
                updatedAt: new Date(),
                userId: "step-executor-user",
            });
        }

        return messages;
    }
}

