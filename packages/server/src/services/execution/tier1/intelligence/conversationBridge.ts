import { type Logger } from "winston";
import { generatePK, type ChatMessage, type BotParticipant, type ChatConfigObject, type BotConfigObject, type SessionUser, ChatConfig, DEFAULT_LANGUAGE } from "@vrooli/shared";
import { completionService } from "../../../conversation/responseEngine.js";
import { PromptTemplateService } from "./promptService.js";

/**
 * Conversation Bridge
 * 
 * Bridges the execution architecture with the existing conversation/responseEngine
 * to enable AI reasoning in swarms. This is a minimal integration that reuses
 * the existing infrastructure instead of reimplementing it.
 */
export class ConversationBridge {
    private readonly logger: Logger;
    private readonly promptService: PromptTemplateService;

    constructor(logger: Logger) {
        this.logger = logger;
        this.promptService = new PromptTemplateService(logger);
    }

    /**
     * Generate AI response for a swarm agent using direct ReasoningEngine integration
     */
    async generateAgentResponse(
        agent: BotParticipant,
        swarmState: any,
        swarmConfig: any,
        userPrompt: string,
        chatId?: string,
    ): Promise<string> {
        try {
            // Build agent context using prompt template
            const systemPrompt = await this.promptService.buildAgentContext(
                agent,
                swarmState,
                swarmConfig,
            );

            // Get conversation state and user context
            const conversationState = chatId ? 
                await completionService.getConversationState(chatId) : null;
            
            // Create proper SessionUser for billing and permissions
            const sessionUser: SessionUser = {
                id: "swarm-system",
                creditAccountId: "system-account", // TODO: Use proper account from swarm context
                hasPremium: false,
                languages: [DEFAULT_LANGUAGE],
            };

            // Get available tools from conversation state or use defaults
            const toolRegistry = completionService.getToolRegistry();
            const availableToolNames = conversationState?.availableTools || [
                "update_swarm_shared_state", 
                "resource_manage", 
                "run_routine", 
                "spawn_swarm"
            ];
            
            // Resolve tool names to full Tool objects
            const availableTools = availableToolNames
                .map(toolName => {
                    const toolDef = toolRegistry.getToolDefinition(typeof toolName === "string" ? toolName : toolName.name);
                    if (!toolDef) {
                        this.logger.warn(`Tool definition not found: ${toolName}`);
                        return null;
                    }
                    return toolDef;
                })
                .filter(tool => tool !== null);

            // Get default conversation config if none exists
            const config = conversationState?.config || {
                goal: swarmConfig?.goal || "Process current task",
                limits: ChatConfig.defaultLimits(),
                stats: ChatConfig.defaultStats(),
                __version: "1.0"
            } as ChatConfigObject;

            // Use ReasoningEngine.runLoop() directly - the native single-agent reasoning loop
            const reasoningEngine = completionService.getReasoningEngine();
            const response = await reasoningEngine.runLoop(
                { text: userPrompt },     // Standalone prompt
                systemPrompt,             // Agent-specific system message
                availableTools,           // Full tool definitions  
                agent,                    // Bot participant
                sessionUser.creditAccountId,
                config,
                10,                       // Tool call allocation
                BigInt(5000),            // Credit allocation (5000 credits)
                sessionUser,
                chatId,
                agent.config?.model,     // Use agent's preferred model
            );

            return response.finalMessage.text;
        } catch (error) {
            this.logger.error("[ConversationBridge] Failed to generate agent response", {
                agentId: agent.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Simple reasoning request without full conversation context
     */
    async reason(
        prompt: string,
        context?: Record<string, unknown>,
    ): Promise<string> {
        const agent: BotParticipant = {
            id: generatePK(),
            name: "Reasoning Agent",
            config: { __version: "1.0", model: "gpt-4" } as BotConfigObject,
            meta: { role: "analyzer", temporary: true },
        };

        const swarmState = {
            state: "reasoning",
            ...context,
        };

        const swarmConfig = {
            goal: "Analyze and provide reasoning",
        };

        return this.generateAgentResponse(
            agent,
            swarmState,
            swarmConfig,
            prompt,
        );
    }

    /**
     * Gets conversation state from the completion service
     */
    async getConversationState(conversationId: string) {
        return completionService.getConversationState(conversationId);
    }

    /**
     * Updates conversation configuration
     */
    updateConversationConfig(conversationId: string, config: ChatConfigObject): void {
        // Delegate to completion service, which manages the conversation store
        completionService.updateConversationConfig(conversationId, config);
    }
}